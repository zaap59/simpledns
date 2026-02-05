package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Session represents an authenticated user session
type Session struct {
	Username  string
	ExpiresAt time.Time
}

var (
	sessions   = make(map[string]Session)
	sessionsMu sync.RWMutex
)

const (
	sessionCookieName = "simpledns_session"
	sessionDuration   = 24 * time.Hour
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a password with a hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateSessionToken creates a cryptographically secure session token
func GenerateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateSession creates a new session for a user
func CreateSession(username string) (string, error) {
	token, err := GenerateSessionToken()
	if err != nil {
		return "", err
	}

	sessionsMu.Lock()
	sessions[token] = Session{
		Username:  username,
		ExpiresAt: time.Now().Add(sessionDuration),
	}
	sessionsMu.Unlock()

	return token, nil
}

// GetSession retrieves a session by token
func GetSession(token string) (Session, bool) {
	sessionsMu.RLock()
	session, exists := sessions[token]
	sessionsMu.RUnlock()

	if !exists {
		return Session{}, false
	}

	if time.Now().After(session.ExpiresAt) {
		DeleteSession(token)
		return Session{}, false
	}

	return session, true
}

// DeleteSession removes a session
func DeleteSession(token string) {
	sessionsMu.Lock()
	delete(sessions, token)
	sessionsMu.Unlock()
}

// AdminExists checks if an admin user has been created
func AdminExists() bool {
	if database == nil || database.db == nil {
		return false
	}

	var count int
	err := database.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// CreateAdmin creates the admin user with the given password
func CreateAdmin(password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	_, err = database.db.Exec(`
		INSERT INTO users (username, password_hash) VALUES ('admin', ?)
	`, hash)
	return err
}

// ValidateLogin checks if the username and password are valid
func ValidateLogin(username, password string) bool {
	if database == nil || database.db == nil {
		return false
	}

	var hash string
	err := database.db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&hash)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		return false
	}

	return CheckPasswordHash(password, hash)
}

// AuthMiddleware checks if the user is authenticated
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for login and setup pages
		path := c.Request.URL.Path
		if path == "/login" || path == "/setup" {
			c.Next()
			return
		}

		// Check if admin exists - if not, redirect to setup
		if !AdminExists() {
			c.Redirect(http.StatusFound, "/setup")
			c.Abort()
			return
		}

		// Check for session cookie
		token, err := c.Cookie(sessionCookieName)
		if err != nil || token == "" {
			c.Redirect(http.StatusFound, "/login?redirect="+c.Request.URL.Path)
			c.Abort()
			return
		}

		// Validate session
		session, valid := GetSession(token)
		if !valid {
			c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login?redirect="+c.Request.URL.Path)
			c.Abort()
			return
		}

		// Store username in context
		c.Set("username", session.Username)
		c.Next()
	}
}

// handleLogin handles the login page and form submission
func handleLogin(c *gin.Context) {
	// If admin doesn't exist, redirect to setup
	if !AdminExists() {
		c.Redirect(http.StatusFound, "/setup")
		return
	}

	if c.Request.Method == "GET" {
		redirect := c.Query("redirect")
		if redirect == "" {
			redirect = "/"
		}

		tmpl := template.Must(template.New("login").Parse(loginHTML))
		c.Header("Content-Type", "text/html")
		if err := tmpl.Execute(c.Writer, gin.H{
			"Redirect": redirect,
			"Error":    "",
		}); err != nil {
			slog.Error("failed to render login template", "error", err)
		}
		return
	}

	// POST - handle login
	username := c.PostForm("username")
	password := c.PostForm("password")
	redirect := c.PostForm("redirect")
	if redirect == "" {
		redirect = "/"
	}

	if !ValidateLogin(username, password) {
		tmpl := template.Must(template.New("login").Parse(loginHTML))
		c.Header("Content-Type", "text/html")
		if err := tmpl.Execute(c.Writer, gin.H{
			"Redirect": redirect,
			"Error":    "Invalid username or password",
		}); err != nil {
			slog.Error("failed to render login template", "error", err)
		}
		return
	}

	// Create session
	token, err := CreateSession(username)
	if err != nil {
		tmpl := template.Must(template.New("login").Parse(loginHTML))
		c.Header("Content-Type", "text/html")
		if err := tmpl.Execute(c.Writer, gin.H{
			"Redirect": redirect,
			"Error":    "Failed to create session",
		}); err != nil {
			slog.Error("failed to render login template", "error", err)
		}
		return
	}

	// Set session cookie
	c.SetCookie(sessionCookieName, token, int(sessionDuration.Seconds()), "/", "", false, true)
	c.Redirect(http.StatusFound, redirect)
}

// handleSetup handles the initial admin setup page
func handleSetup(c *gin.Context) {
	// If admin already exists, redirect to login
	if AdminExists() {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if c.Request.Method == "GET" {
		tmpl := template.Must(template.New("setup").Parse(setupHTML))
		c.Header("Content-Type", "text/html")
		if err := tmpl.Execute(c.Writer, gin.H{
			"Error": "",
		}); err != nil {
			slog.Error("failed to render setup template", "error", err)
		}
		return
	}

	// POST - handle setup
	password := c.PostForm("password")
	confirmPassword := c.PostForm("confirm_password")

	if password == "" {
		tmpl := template.Must(template.New("setup").Parse(setupHTML))
		c.Header("Content-Type", "text/html")
		if err := tmpl.Execute(c.Writer, gin.H{
			"Error": "Password is required",
		}); err != nil {
			slog.Error("failed to render setup template", "error", err)
		}
		return
	}

	if len(password) < 8 {
		tmpl := template.Must(template.New("setup").Parse(setupHTML))
		c.Header("Content-Type", "text/html")
		if err := tmpl.Execute(c.Writer, gin.H{
			"Error": "Password must be at least 8 characters",
		}); err != nil {
			slog.Error("failed to render setup template", "error", err)
		}
		return
	}

	if password != confirmPassword {
		tmpl := template.Must(template.New("setup").Parse(setupHTML))
		c.Header("Content-Type", "text/html")
		if err := tmpl.Execute(c.Writer, gin.H{
			"Error": "Passwords do not match",
		}); err != nil {
			slog.Error("failed to render setup template", "error", err)
		}
		return
	}

	if err := CreateAdmin(password); err != nil {
		tmpl := template.Must(template.New("setup").Parse(setupHTML))
		c.Header("Content-Type", "text/html")
		if err := tmpl.Execute(c.Writer, gin.H{
			"Error": "Failed to create admin user: " + err.Error(),
		}); err != nil {
			slog.Error("failed to render setup template", "error", err)
		}
		return
	}

	// Create session and redirect to dashboard
	token, _ := CreateSession("admin")
	c.SetCookie(sessionCookieName, token, int(sessionDuration.Seconds()), "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

// handleLogout handles user logout
func handleLogout(c *gin.Context) {
	token, err := c.Cookie(sessionCookieName)
	if err == nil && token != "" {
		DeleteSession(token)
	}
	c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}

// UpdatePassword updates the password for a user
func UpdatePassword(username, newPassword string) error {
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	_, err = database.db.Exec(`
		UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE username = ?
	`, hash, username)
	return err
}

// APIToken represents an API token
type APIToken struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Token      string `json:"token,omitempty"` // Only set when creating
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at,omitempty"`
}

// GenerateAPIToken creates a new API token
func GenerateAPIToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "sdns_" + hex.EncodeToString(bytes), nil
}

// HashAPIToken hashes an API token for storage
func HashAPIToken(token string) string {
	// Use SHA256 for API tokens (faster than bcrypt, still secure for tokens)
	bytes := make([]byte, 32)
	copy(bytes, []byte(token))
	return hex.EncodeToString(bytes)
}

// CreateAPIToken creates a new API token for a user
func CreateAPIToken(username, name string) (*APIToken, error) {
	if database == nil || database.db == nil {
		return nil, sql.ErrConnDone
	}

	// Get user ID
	var userID int64
	err := database.db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
	if err != nil {
		return nil, err
	}

	// Generate token
	token, err := GenerateAPIToken()
	if err != nil {
		return nil, err
	}

	// Hash token for storage
	tokenHash := HashAPIToken(token)

	// Insert token
	result, err := database.db.Exec(`
		INSERT INTO api_tokens (user_id, name, token_hash) VALUES (?, ?, ?)
	`, userID, name, tokenHash)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()

	return &APIToken{
		ID:        id,
		Name:      name,
		Token:     token, // Return the raw token only on creation
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// ListAPITokens returns all API tokens for a user (without the actual token)
func ListAPITokens(username string) ([]APIToken, error) {
	if database == nil || database.db == nil {
		return nil, sql.ErrConnDone
	}

	rows, err := database.db.Query(`
		SELECT t.id, t.name, t.created_at, t.last_used_at
		FROM api_tokens t
		JOIN users u ON t.user_id = u.id
		WHERE u.username = ?
		ORDER BY t.created_at DESC
	`, username)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tokens []APIToken
	for rows.Next() {
		var t APIToken
		var lastUsed sql.NullString
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt, &lastUsed); err != nil {
			return nil, err
		}
		if lastUsed.Valid {
			t.LastUsedAt = lastUsed.String
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}

// DeleteAPIToken deletes an API token
func DeleteAPIToken(username string, tokenID int64) error {
	if database == nil || database.db == nil {
		return sql.ErrConnDone
	}

	_, err := database.db.Exec(`
		DELETE FROM api_tokens 
		WHERE id = ? AND user_id = (SELECT id FROM users WHERE username = ?)
	`, tokenID, username)
	return err
}

// ValidateAPIToken checks if an API token is valid and returns the username
func ValidateAPIToken(token string) (string, bool) {
	if database == nil || database.db == nil {
		return "", false
	}

	tokenHash := HashAPIToken(token)

	var username string
	var tokenID int64
	err := database.db.QueryRow(`
		SELECT u.username, t.id
		FROM api_tokens t
		JOIN users u ON t.user_id = u.id
		WHERE t.token_hash = ?
	`, tokenHash).Scan(&username, &tokenID)
	if err != nil {
		return "", false
	}

	// Update last used timestamp
	go func() {
		_, _ = database.db.Exec("UPDATE api_tokens SET last_used_at = CURRENT_TIMESTAMP WHERE id = ?", tokenID)
	}()

	return username, true
}

// APIAuthMiddleware checks for API token or session authentication
func APIAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for Bearer token in Authorization header
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token := authHeader[7:]
			if username, valid := ValidateAPIToken(token); valid {
				c.Set("username", username)
				c.Set("auth_type", "api_token")
				c.Next()
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API token"})
			c.Abort()
			return
		}

		// Check for X-API-Key header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			if username, valid := ValidateAPIToken(apiKey); valid {
				c.Set("username", username)
				c.Set("auth_type", "api_token")
				c.Next()
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}

		// Fall back to session authentication
		token, err := c.Cookie(sessionCookieName)
		if err == nil && token != "" {
			if session, valid := GetSession(token); valid {
				c.Set("username", session.Username)
				c.Set("auth_type", "session")
				c.Next()
				return
			}
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		c.Abort()
	}
}

// handleAccount handles the account/password management page
func handleAccount(c *gin.Context) {
	username, _ := c.Get("username")
	usernameStr := username.(string)

	// Get API tokens for display
	tokens, _ := ListAPITokens(usernameStr)

	if c.Request.Method == "GET" {
		tmpl := template.Must(template.New("account").Parse(headerHTML + sidebarHTML + accountHTML))
		c.Header("Content-Type", "text/html")
		if err := tmpl.Execute(c.Writer, gin.H{
			"Username":        usernameStr,
			"Mode":            dbMode,
			"CurrentPath":     "/account",
			"Error":           "",
			"Success":         "",
			"APITokens":       tokens,
			"PageTitle":       "Account",
			"ShowSetupButton": true,
		}); err != nil {
			slog.Error("failed to render account template", "error", err)
		}
		return
	}

	// POST - handle password change
	currentPassword := c.PostForm("current_password")
	newPassword := c.PostForm("new_password")
	confirmPassword := c.PostForm("confirm_password")

	renderError := func(errMsg string) {
		tmpl := template.Must(template.New("account").Parse(headerHTML + sidebarHTML + accountHTML))
		c.Header("Content-Type", "text/html")
		if err := tmpl.Execute(c.Writer, gin.H{
			"Username":        usernameStr,
			"Mode":            dbMode,
			"CurrentPath":     "/account",
			"Error":           errMsg,
			"Success":         "",
			"APITokens":       tokens,
			"PageTitle":       "Account",
			"ShowSetupButton": true,
		}); err != nil {
			slog.Error("failed to render account template", "error", err)
		}
	}

	// Validate current password
	if !ValidateLogin(usernameStr, currentPassword) {
		renderError("Current password is incorrect")
		return
	}

	// Validate new password
	if len(newPassword) < 8 {
		renderError("New password must be at least 8 characters")
		return
	}

	if newPassword != confirmPassword {
		renderError("New passwords do not match")
		return
	}

	// Update password
	if err := UpdatePassword(usernameStr, newPassword); err != nil {
		renderError("Failed to update password: " + err.Error())
		return
	}

	// Refresh tokens list
	tokens, _ = ListAPITokens(usernameStr)

	// Success
	tmpl := template.Must(template.New("account").Parse(headerHTML + sidebarHTML + accountHTML))
	c.Header("Content-Type", "text/html")
	if err := tmpl.Execute(c.Writer, gin.H{
		"Username":        usernameStr,
		"Mode":            dbMode,
		"CurrentPath":     "/account",
		"Error":           "",
		"Success":         "Password updated successfully",
		"APITokens":       tokens,
		"PageTitle":       "Account",
		"ShowSetupButton": true,
	}); err != nil {
		slog.Error("failed to render account template", "error", err)
	}
}

// handleCreateAPIToken handles API token creation
func handleCreateAPIToken(c *gin.Context) {
	username, _ := c.Get("username")
	usernameStr := username.(string)

	name := c.PostForm("token_name")
	if name == "" {
		name = "API Token"
	}

	token, err := CreateAPIToken(usernameStr, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, token)
}

// handleDeleteAPIToken handles API token deletion
func handleDeleteAPIToken(c *gin.Context) {
	username, _ := c.Get("username")
	usernameStr := username.(string)

	var tokenID int64
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &tokenID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
		return
	}

	if err := DeleteAPIToken(usernameStr, tokenID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// handleListAPITokens returns the list of API tokens or renders the tokens page
func handleListAPITokens(c *gin.Context) {
	username, _ := c.Get("username")
	usernameStr := username.(string)

	tokens, err := ListAPITokens(usernameStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tokens"})
		return
	}

	// Check Accept header - if JSON requested, return JSON
	accept := c.GetHeader("Accept")
	if accept == "application/json" {
		c.JSON(http.StatusOK, tokens)
		return
	}

	// Otherwise render the tokens page
	tmpl := template.Must(template.New("tokens").Parse(headerHTML + sidebarHTML + apiTokensHTML))
	c.Header("Content-Type", "text/html")
	if err := tmpl.Execute(c.Writer, gin.H{
		"Username":        usernameStr,
		"Mode":            dbMode,
		"CurrentPath":     "/account/tokens",
		"APITokens":       tokens,
		"PageTitle":       "API Tokens",
		"ShowSetupButton": true,
	}); err != nil {
		slog.Error("failed to render tokens template", "error", err)
	}
}

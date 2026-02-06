package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SyncToken is stored in the config table
const syncTokenConfigKey = "sync_token"

// SyncZoneData represents zone data for synchronization
type SyncZoneData struct {
	Zone    DBZone     `json:"zone"`
	Records []DBRecord `json:"records"`
}

// SyncResponse is the response from the master server
type SyncResponse struct {
	Zones     []SyncZoneData `json:"zones"`
	Timestamp string         `json:"timestamp"`
	ZoneNames []string       `json:"zone_names"`
}

// SlaveInfo is sent by slave when registering
type SlaveInfo struct {
	Name      string `json:"name"`
	IPAddress string `json:"ip_address"`
	Port      int    `json:"port"`
}

// === Sync Token Management ===

// GenerateSyncToken creates a new sync token for master
func GenerateSyncToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetSyncToken retrieves or generates the sync token
func GetSyncToken() (string, error) {
	if database == nil {
		return "", fmt.Errorf("database not initialized")
	}

	// Try to get existing token
	token, err := database.GetConfig(syncTokenConfigKey)
	if err == nil && token != "" {
		return token, nil
	}

	// Generate new token
	newToken, err := GenerateSyncToken()
	if err != nil {
		return "", err
	}

	// Store it
	if err := database.SetConfig(syncTokenConfigKey, newToken); err != nil {
		return "", err
	}

	return newToken, nil
}

// RegenerateSyncToken creates a new sync token (invalidates old one)
func RegenerateSyncToken() (string, error) {
	newToken, err := GenerateSyncToken()
	if err != nil {
		return "", err
	}

	if err := database.SetConfig(syncTokenConfigKey, newToken); err != nil {
		return "", err
	}

	return newToken, nil
}

// HashToken creates a SHA256 hash of the token for comparison
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// === Sync API Handlers (Master side) ===

// ValidateSyncToken middleware validates the sync token
func ValidateSyncToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix if present
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Get stored sync token
		storedToken, err := GetSyncToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get sync token"})
			c.Abort()
			return
		}

		// Compare tokens
		if token != storedToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid sync token"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// HandleSyncRegister handles slave registration
func HandleSyncRegister(c *gin.Context) {
	if serverRole != "master" {
		c.JSON(http.StatusForbidden, gin.H{"error": "this server is not a master"})
		return
	}

	var info SlaveInfo
	if err := c.ShouldBindJSON(&info); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Use client IP if not provided
	if info.IPAddress == "" {
		info.IPAddress = c.ClientIP()
	}
	if info.Name == "" {
		info.Name = "slave-" + info.IPAddress
	}

	// Port information may be 0 if unknown
	slave, err := database.RegisterSlave(info.Name, info.IPAddress, info.Port)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.Info("Slave registered", "name", slave.Name, "ip", slave.IPAddress, "id", slave.ID)
	c.JSON(http.StatusOK, gin.H{
		"message":  "registered",
		"slave_id": slave.ID,
	})
}

// HandleSyncHeartbeat handles slave heartbeat
func HandleSyncHeartbeat(c *gin.Context) {
	if serverRole != "master" {
		c.JSON(http.StatusForbidden, gin.H{"error": "this server is not a master"})
		return
	}

	slaveIDStr := c.Query("slave_id")
	if slaveIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slave_id required"})
		return
	}

	slaveID, err := strconv.ParseInt(slaveIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid slave_id"})
		return
	}

	if err := database.UpdateSlaveHeartbeat(slaveID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// HandleSyncZones returns all zones and records for sync
func HandleSyncZones(c *gin.Context) {
	if serverRole != "master" {
		c.JSON(http.StatusForbidden, gin.H{"error": "this server is not a master"})
		return
	}

	// Optional: filter by version
	sinceVersionStr := c.Query("since_version")
	var sinceVersion int
	if sinceVersionStr != "" {
		var err error
		sinceVersion, err = strconv.Atoi(sinceVersionStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid since_version"})
			return
		}
	}

	zones, err := database.ListZones()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var syncData []SyncZoneData
	for _, zone := range zones {
		// Filter by version if requested
		if sinceVersion > 0 && zone.Version <= sinceVersion {
			continue
		}

		records, err := database.ListRecordsByZone(zone.ID)
		if err != nil {
			slog.Warn("Failed to list records for zone", "zone", zone.Name, "error", err)
			continue
		}

		syncData = append(syncData, SyncZoneData{
			Zone:    zone,
			Records: records,
		})
	}

	// Update slave sync status if slave_id provided
	// Use total number of zones on the master, not only the number returned for this incremental sync,
	// otherwise a sync with no changes would set the display to 0.
	slaveIDStr := c.Query("slave_id")
	if slaveIDStr != "" {
		if slaveID, err := strconv.ParseInt(slaveIDStr, 10, 64); err == nil {
			totalZones := len(zones)
			_ = database.UpdateSlaveSyncStatus(slaveID, totalZones)
		}
	}

	// Include a full list of zone names so slaves can reconcile deletions
	zoneNames := make([]string, 0, len(zones))
	for _, z := range zones {
		zoneNames = append(zoneNames, z.Name)
	}

	c.JSON(http.StatusOK, SyncResponse{
		Zones:     syncData,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		ZoneNames: zoneNames,
	})
}

// HandleGetSlaves returns list of registered slaves (for UI)
func HandleGetSlaves(c *gin.Context) {
	if serverRole != "master" {
		c.JSON(http.StatusOK, gin.H{"slaves": []DBSlave{}, "message": "not a master server"})
		return
	}

	// Mark stale slaves (no heartbeat for 2 minutes)
	_ = database.MarkStaleSlaves(120)

	slaves, err := database.ListSlaves()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if slaves == nil {
		slaves = []DBSlave{}
	}

	c.JSON(http.StatusOK, gin.H{"slaves": slaves})
}

// HandleDeleteSlave removes a slave
func HandleDeleteSlave(c *gin.Context) {
	if serverRole != "master" {
		c.JSON(http.StatusForbidden, gin.H{"error": "this server is not a master"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid slave id"})
		return
	}

	if err := database.DeleteSlave(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.Info("Slave deleted", "id", id)
	c.JSON(http.StatusOK, gin.H{"message": "slave deleted"})
}

// HandleGetSyncToken returns the sync token (for UI display)
func HandleGetSyncToken(c *gin.Context) {
	if serverRole != "master" {
		c.JSON(http.StatusOK, gin.H{"token": "", "message": "not a master server"})
		return
	}

	token, err := GetSyncToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// HandleRegenerateSyncToken creates a new sync token
func HandleRegenerateSyncToken(c *gin.Context) {
	if serverRole != "master" {
		c.JSON(http.StatusForbidden, gin.H{"error": "this server is not a master"})
		return
	}

	token, err := RegenerateSyncToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	slog.Info("Sync token regenerated")
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// === Slave Sync Logic ===

var slaveID int64 = 0
var lastSyncVersion int = 0

// master connection state (for slave mode UI)
var masterConnected bool = false
var masterLastContact time.Time

func setMasterConnected(v bool) {
	masterConnected = v
	if v {
		masterLastContact = time.Now().UTC()
	}
}

// getMasterURL constructs the master API URL from host and port
func getMasterURL() string {
	return fmt.Sprintf("http://%s:%d", masterAPIHost, masterAPIPort)
}

// StartSlaveSync starts the sync goroutine for slave mode
func StartSlaveSync() {
	if serverRole != "slave" || masterAPIHost == "" || masterToken == "" {
		slog.Info("Slave sync not started", "role", serverRole, "master_host", masterAPIHost)
		return
	}

	slog.Info("Starting slave sync", "master", getMasterURL(), "interval", syncInterval)

	// Ensure our local sync token matches the configured master token so master can authenticate when pushing
	if masterToken != "" && database != nil {
		_ = database.SetConfig(syncTokenConfigKey, masterToken)
	}

	// Register with master
	if err := registerWithMaster(); err != nil {
		slog.Error("Failed to register with master", "error", err)
	} else {
		// After successful registration, reset lastSyncVersion to 0 to request a full sync
		// This handles cases where local zone versions are out-of-sync/higher than the master
		lastSyncVersion = 0
	}

	// Start sync loop
	go func() {
		ticker := time.NewTicker(syncInterval)
		defer ticker.Stop()

		// Initial sync
		if err := syncFromMaster(); err != nil {
			slog.Error("Initial sync failed", "error", err)
		}

		for range ticker.C {
			// Send heartbeat
			if err := sendHeartbeat(); err != nil {
				slog.Warn("Heartbeat failed", "error", err)
			}

			// Sync zones
			if err := syncFromMaster(); err != nil {
				slog.Error("Sync failed", "error", err)
			}
		}
	}()
}

func registerWithMaster() error {
	url := getMasterURL() + "/api/sync/register"

	// Get our hostname
	hostname := "slave"

	// Determine our IP and port to advertise to master
	ip := getOutboundIP()
	port := 0
	if webServerPort > 0 {
		port = webServerPort
	}

	body, _ := json.Marshal(SlaveInfo{
		Name:      hostname,
		IPAddress: ip,
		Port:      port,
	})

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+masterToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		setMasterConnected(false)
		return fmt.Errorf("failed to connect to master: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		setMasterConnected(false)
		return fmt.Errorf("registration failed: %s - %s", resp.Status, string(bodyBytes))
	}

	var result struct {
		SlaveID int64 `json:"slave_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	slaveID = result.SlaveID
	setMasterConnected(true)
	slog.Info("Registered with master", "slave_id", slaveID)
	return nil
}

func sendHeartbeat() error {
	if slaveID == 0 {
		return registerWithMaster()
	}

	url := fmt.Sprintf("%s/api/sync/heartbeat?slave_id=%d", getMasterURL(), slaveID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+masterToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		setMasterConnected(false)
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		setMasterConnected(false)
		return fmt.Errorf("heartbeat failed: %s", resp.Status)
	}

	setMasterConnected(true)
	return nil
}

func syncFromMaster() error {
	url := fmt.Sprintf("%s/api/sync/zones?slave_id=%d&since_version=%d",
		getMasterURL(), slaveID, lastSyncVersion)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+masterToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		setMasterConnected(false)
		return fmt.Errorf("failed to connect to master: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		setMasterConnected(false)
		return fmt.Errorf("sync failed: %s - %s", resp.Status, string(bodyBytes))
	}

	var syncResp SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		setMasterConnected(false)
		return err
	}

	// We successfully contacted the master
	setMasterConnected(true)

	// If the master provided a full list of zone names, check if the master
	// has zones that are missing locally. If so, force a full sync so the
	// slave pulls newly-created or re-created zones even when versions don't
	// indicate a change.
	if len(syncResp.ZoneNames) > 0 {
		masterSet := make(map[string]struct{}, len(syncResp.ZoneNames))
		for _, n := range syncResp.ZoneNames {
			masterSet[n] = struct{}{}
		}

		localZones, err := database.ListZones()
		if err == nil {
			localSet := make(map[string]struct{}, len(localZones))
			for _, lz := range localZones {
				localSet[lz.Name] = struct{}{}
			}

			missingAny := false
			for name := range masterSet {
				if _, ok := localSet[name]; !ok {
					missingAny = true
					slog.Info("Master has zone missing locally; scheduling full sync", "zone", name)
					break
				}
			}

			if missingAny && lastSyncVersion != 0 {
				lastSyncVersion = 0
				// Trigger an immediate follow-up full sync asynchronously
				go func() {
					time.Sleep(200 * time.Millisecond)
					if err := syncFromMaster(); err != nil {
						slog.Warn("Follow-up full sync (missing zones) failed", "error", err)
					}
				}()
				// Return now; the follow-up full sync will perform the full sync work
				return nil
			}
		}
	}

	if len(syncResp.Zones) == 0 {
		slog.Debug("No zones to sync")
	} else {
		slog.Info("Syncing zones from master", "count", len(syncResp.Zones))

		// Apply zones and track highest version synced
		for _, syncData := range syncResp.Zones {
			if err := applyZoneSync(syncData); err != nil {
				slog.Error("Failed to sync zone", "zone", syncData.Zone.Name, "error", err)
				continue
			}
			if syncData.Zone.Version > lastSyncVersion {
				lastSyncVersion = syncData.Zone.Version
			}
		}

		// Reload DNS zones after applying changes
		if err := ReloadFromDB(); err != nil {
			slog.Warn("Failed to reload zones after sync", "error", err)
		}
	}

	// Reconcile deletions: if master provided the full list of zone names, remove local zones not present on master
	if len(syncResp.ZoneNames) > 0 {
		masterSet := make(map[string]struct{}, len(syncResp.ZoneNames))
		for _, n := range syncResp.ZoneNames {
			masterSet[n] = struct{}{}
		}

		localZones, err := database.ListZones()
		if err == nil {
			deletedAny := false
			for _, lz := range localZones {
				if _, ok := masterSet[lz.Name]; !ok {
					if derr := database.DeleteZone(lz.ID); derr == nil {
						deletedAny = true
						slog.Info("Deleted local zone not present on master", "zone", lz.Name)
					} else {
						slog.Warn("Failed to delete local zone during reconciliation", "zone", lz.Name, "error", derr)
					}
				}
			}
			// If we deleted zones, reset lastSyncVersion so we request a full sync (in case new zones have lower versions)
			if deletedAny {
				lastSyncVersion = 0
				slog.Info("Reconciliation removed local zones; reset lastSyncVersion to force full sync")
				// Trigger an immediate follow-up full sync asynchronously
				go func() {
					time.Sleep(500 * time.Millisecond)
					if err := syncFromMaster(); err != nil {
						slog.Warn("Follow-up full sync failed", "error", err)
					}
				}()
			}

			// Reload after possible deletions
			if err := ReloadFromDB(); err != nil {
				slog.Warn("Failed to reload zones after reconciliation", "error", err)
			}
		}
	}

	slog.Info("Sync completed", "zones_returned", len(syncResp.Zones), "last_version", lastSyncVersion)
	return nil
}

func applyZoneSync(syncData SyncZoneData) error {
	// Check if zone exists
	existingZone, _ := database.GetZoneByName(syncData.Zone.Name)

	if existingZone == nil {
		// Create new zone
		zone := &DBZone{
			Name:    syncData.Zone.Name,
			Enabled: syncData.Zone.Enabled,
			TTL:     syncData.Zone.TTL,
			NS:      syncData.Zone.NS,
			Admin:   syncData.Zone.Admin,
			Serial:  syncData.Zone.Serial,
			Refresh: syncData.Zone.Refresh,
			Retry:   syncData.Zone.Retry,
			Expire:  syncData.Zone.Expire,
			Version: syncData.Zone.Version,
		}
		if err := database.CreateZone(zone); err != nil {
			return fmt.Errorf("failed to create zone: %w", err)
		}

		// Get the created zone to get its ID
		createdZone, err := database.GetZoneByName(zone.Name)
		if err != nil {
			return fmt.Errorf("failed to get created zone: %w", err)
		}

		// Add records
		for _, record := range syncData.Records {
			record.ZoneID = createdZone.ID
			if err := database.CreateRecord(&record); err != nil {
				slog.Warn("Failed to create record", "zone", zone.Name, "record", record.Name, "error", err)
			}
		}

		slog.Info("Zone created from sync", "zone", zone.Name, "records", len(syncData.Records))
	} else {
		// Update existing zone
		existingZone.Enabled = syncData.Zone.Enabled
		existingZone.TTL = syncData.Zone.TTL
		existingZone.NS = syncData.Zone.NS
		existingZone.Admin = syncData.Zone.Admin
		existingZone.Serial = syncData.Zone.Serial
		existingZone.Refresh = syncData.Zone.Refresh
		existingZone.Retry = syncData.Zone.Retry
		existingZone.Expire = syncData.Zone.Expire
		existingZone.Version = syncData.Zone.Version

		if err := database.UpdateZone(existingZone); err != nil {
			return fmt.Errorf("failed to update zone: %w", err)
		}

		// Delete existing records and re-create
		existingRecords, _ := database.ListRecordsByZone(existingZone.ID)
		for _, r := range existingRecords {
			_ = database.DeleteRecord(r.ID)
		}

		// Add new records
		for _, record := range syncData.Records {
			record.ZoneID = existingZone.ID
			if err := database.CreateRecord(&record); err != nil {
				slog.Warn("Failed to create record", "zone", existingZone.Name, "record", record.Name, "error", err)
			}
		}

		slog.Info("Zone updated from sync", "zone", existingZone.Name, "records", len(syncData.Records))
	}

	return nil
}

// Master-side: push a zone payload to a slave
func pushZoneToSlave(slave DBSlave, zoneID int64) error {
	port := slave.Port
	if port == 0 {
		port = 8080
	}

	// Build payload for the specific zone
	z, err := database.GetZone(zoneID)
	if err != nil {
		return err
	}
	records, _ := database.ListRecordsByZone(z.ID)

	syncResp := SyncResponse{
		Zones: []SyncZoneData{{
			Zone:    *z,
			Records: records,
		}},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Include full list of zone names to help slaves reconcile deletions
	allZones, _ := database.ListZones()
	zoneNames := make([]string, 0, len(allZones))
	for _, az := range allZones {
		zoneNames = append(zoneNames, az.Name)
	}
	syncResp.ZoneNames = zoneNames

	data, _ := json.Marshal(syncResp)

	url := fmt.Sprintf("http://%s:%d/api/replication/push", slave.IPAddress, port)
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Use master's sync token to authenticate
	token, err := GetSyncToken()
	if err == nil && token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push failed: %s - %s", resp.Status, string(bodyBytes))
	}

	return nil
}

// Push zone to all registered slaves (async)
func pushZoneToAllSlaves(zoneID int64) {
	slaves, err := database.ListSlaves()
	if err != nil {
		slog.Warn("failed to list slaves for push", "error", err)
		return
	}
	for _, s := range slaves {
		go func(sl DBSlave) {
			if err := pushZoneToSlave(sl, zoneID); err != nil {
				slog.Warn("failed to push zone to slave", "slave", sl.IPAddress, "error", err)
			} else {
				slog.Info("Pushed zone to slave", "slave", sl.IPAddress, "zone_id", zoneID)
			}
		}(s)
	}
}

// Push full zone name list to all slaves (used after deletions)
func pushZoneListToAllSlaves() {
	slaves, err := database.ListSlaves()
	if err != nil {
		slog.Warn("failed to list slaves for push zone list", "error", err)
		return
	}

	allZones, _ := database.ListZones()
	zoneNames := make([]string, 0, len(allZones))
	for _, az := range allZones {
		zoneNames = append(zoneNames, az.Name)
	}

	syncResp := SyncResponse{
		Zones:     []SyncZoneData{},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		ZoneNames: zoneNames,
	}

	data, _ := json.Marshal(syncResp)

	for _, s := range slaves {
		go func(sl DBSlave) {
			port := sl.Port
			if port == 0 {
				port = 8080
			}
			url := fmt.Sprintf("http://%s:%d/api/replication/push", sl.IPAddress, port)
			req, err := http.NewRequest("POST", url, bytes.NewReader(data))
			if err != nil {
				slog.Warn("failed to build push request", "slave", sl.IPAddress, "error", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			token, err := GetSyncToken()
			if err == nil && token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				slog.Warn("failed to push zone list to slave", "slave", sl.IPAddress, "error", err)
				return
			}
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				slog.Warn("push zone list returned non-OK", "slave", sl.IPAddress, "status", resp.Status)
			}
		}(s)
	}
}

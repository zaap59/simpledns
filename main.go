package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
	"gopkg.in/yaml.v3"
)

var zones map[string][]dns.RR
var forwarders []string
var forwardTimeout time.Duration = 2 * time.Second
var loadedZoneNames []string
var dbMode string = "files" // "files" or "sqlite"

// Server role configuration
var serverRole string = "master" // "master" or "slave"
var dnsPort int = 53             // DNS server port

// Web server port (set from config in main)
var webServerPort int = 0

// Slave sync configuration
var masterAPIHost string = "" // Master server IP (e.g., 192.168.1.1)
var masterAPIPort int = 8080  // Master API port (default 8080)
var masterToken string = ""   // Sync token for authentication with master
var syncInterval time.Duration = 30 * time.Second

// flag types that track whether they were set on the command line
type stringFlag struct {
	value string
	set   bool
}

func (s *stringFlag) Set(v string) error { s.value = v; s.set = true; return nil }
func (s *stringFlag) String() string     { return s.value }

// YAML Zone structures
type YAMLZoneConfig struct {
	ZoneConfig struct {
		Name   string `yaml:"name"`
		Origin string `yaml:"origin"`
		TTL    int    `yaml:"ttl"`
	} `yaml:"zone_config"`
	SOA struct {
		NS      string `yaml:"ns"`
		Admin   string `yaml:"admin"`
		Serial  int    `yaml:"serial"`
		Refresh int    `yaml:"refresh"`
		Retry   int    `yaml:"retry"`
		Expire  int    `yaml:"expire"`
	} `yaml:"soa"`
	DNSRecords []struct {
		Name  string `yaml:"name"`
		Type  string `yaml:"type"`
		Value string `yaml:"value"`
		TTL   int    `yaml:"ttl"`
	} `yaml:"dns_records"`
}

// debug can be enabled via the CLI flag `-debug`

type AppConfig struct {
	DBType            string   `yaml:"db_type" json:"db_type,omitempty"`
	DBPath            string   `yaml:"db_path" json:"db_path,omitempty"`
	ZonesDir          string   `yaml:"zones_dir" json:"zones_dir,omitempty"`
	Forwarders        []string `yaml:"forwarders" json:"forwarders,omitempty"`
	ForwardTimeoutSec int      `yaml:"forward_timeout_seconds" json:"forward_timeout_seconds,omitempty"`
	Addr              string   `yaml:"addr" json:"addr,omitempty"`
	WebEnabled        bool     `yaml:"web_enabled" json:"web_enabled,omitempty"`
	WebPort           int      `yaml:"web_port" json:"web_port,omitempty"`
	ServerRole        string   `yaml:"server_role" json:"server_role,omitempty"`
	DNSPort           int      `yaml:"dns_port" json:"dns_port,omitempty"`
	MasterAPIHost     string   `yaml:"master_api_host" json:"master_api_host,omitempty"`
	MasterAPIPort     int      `yaml:"master_api_port" json:"master_api_port,omitempty"`
	MasterToken       string   `yaml:"master_token" json:"master_token,omitempty"`
	SyncIntervalSec   int      `yaml:"sync_interval_seconds" json:"sync_interval_seconds,omitempty"`
}

func loadAppConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func parseForwarders(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// add default port 53 if missing
		if !strings.Contains(p, ":") {
			p = p + ":53"
		}
		out = append(out, p)
	}
	return out
}

func forwardQuery(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	c := &dns.Client{Timeout: forwardTimeout}
	for _, srv := range forwarders {
		resp, _, err := c.ExchangeContext(ctx, msg, srv)
		if err != nil {
			slog.Debug("forward to %s failed", "server", srv, "error", err)
			continue
		}
		if resp == nil {
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("no upstream answered")
}

func mustNewRR(s string) dns.RR {
	rr, err := dns.NewRR(s)
	if err != nil {
		log.Fatalf("invalid RR %q: %v", s, err)
	}
	return rr
}

// loadZonesFromYAMLFile loads a single YAML zone file
func loadZonesFromYAMLFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var zoneConfig YAMLZoneConfig
	if err := yaml.Unmarshal(data, &zoneConfig); err != nil {
		return fmt.Errorf("invalid YAML zone file %s: %w", path, err)
	}

	if zones == nil {
		zones = make(map[string][]dns.RR)
	}

	zoneName := dns.Fqdn(zoneConfig.ZoneConfig.Name)
	loadedZoneNames = append(loadedZoneNames, zoneName)

	// Convert SOA record
	soaStr := fmt.Sprintf("%s 3600 IN SOA %s %s %d %d %d %d 3600",
		zoneName,
		zoneConfig.SOA.NS,
		strings.Replace(zoneConfig.SOA.Admin, "@", ".", 1),
		zoneConfig.SOA.Serial,
		zoneConfig.SOA.Refresh,
		zoneConfig.SOA.Retry,
		zoneConfig.SOA.Expire,
	)
	soaRR := mustNewRR(soaStr)
	zones[zoneName] = append(zones[zoneName], soaRR)

	// Convert NS record
	nsStr := fmt.Sprintf("%s 3600 IN NS %s", zoneName, zoneConfig.SOA.NS)
	nsRR := mustNewRR(nsStr)
	zones[zoneName] = append(zones[zoneName], nsRR)

	// Convert DNS records
	for _, record := range zoneConfig.DNSRecords {
		ttl := record.TTL
		if ttl == 0 {
			ttl = zoneConfig.ZoneConfig.TTL
		}

		// Build record name (relative to zone origin)
		recordName := record.Name
		if recordName == "@" {
			recordName = zoneName
		} else if !strings.HasSuffix(recordName, ".") {
			recordName = recordName + "." + zoneName
		}

		rrStr := fmt.Sprintf("%s %d IN %s %s", recordName, ttl, record.Type, record.Value)
		rr, err := dns.NewRR(rrStr)
		if err != nil {
			return fmt.Errorf("invalid RR in %s: %q: %w", path, rrStr, err)
		}
		name := dns.Fqdn(rr.Header().Name)
		zones[name] = append(zones[name], rr)
	}

	return nil
}

func loadZonesFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if zones == nil {
		zones = make(map[string][]dns.RR)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		base := e.Name()

		// Only load YAML files (.yaml or .yml)
		if strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".yml") {
			if err := loadZonesFromYAMLFile(path); err != nil {
				return fmt.Errorf("parse YAML %s: %w", path, err)
			}
		}
		// Ignore other file types
	}
	return nil
}

func initZones(confDir string) {
	// Load zones from conf directory
	if confDir != "" {
		if info, err := os.Stat(confDir); err == nil && info.IsDir() {
			if err := loadZonesFromDir(confDir); err == nil {
				slog.Info("Loaded zones from directory", "path", confDir)
				return
			} else {
				slog.Warn("Failed to load zones from directory", "path", confDir, "error", err)
			}
		}
	}

	// Fallback defaults
	zones = map[string][]dns.RR{
		"example.local.": {
			mustNewRR("example.local. 3600 IN A 127.0.0.1"),
		},
		"www.example.local.": {
			mustNewRR("www.example.local. 3600 IN CNAME example.local."),
		},
	}
}

// ZoneInfo represents zone information for the web interface
type ZoneInfo struct {
	ID      int64        `json:"id"`
	Name    string       `json:"name"`
	Enabled bool         `json:"enabled"`
	Serial  int          `json:"serial"`
	Version int          `json:"version"`
	TTL     int          `json:"ttl"`
	NS      string       `json:"ns"`
	Admin   string       `json:"admin"`
	Refresh int          `json:"refresh"`
	Retry   int          `json:"retry"`
	Expire  int          `json:"expire"`
	Records []RecordInfo `json:"records"`
}

// RecordInfo represents a DNS record for the web interface
type RecordInfo struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	TTL      uint32 `json:"ttl"`
	Priority int    `json:"priority"`
}

// getZonesInfo returns structured information about loaded zones
func getZonesInfo() []ZoneInfo {
	// In SQLite mode, get zones with their IDs from database
	if dbMode == "sqlite" && database != nil {
		return getZonesInfoFromDB()
	}

	// In files mode, build from in-memory zones
	zoneMap := make(map[string]*ZoneInfo)

	for name, rrList := range zones {
		for _, rr := range rrList {
			zoneName := findZoneForRecord(name)
			if zoneName == "" {
				zoneName = name
			}

			if _, exists := zoneMap[zoneName]; !exists {
				zoneMap[zoneName] = &ZoneInfo{Name: strings.TrimSuffix(zoneName, "."), Enabled: true, Records: []RecordInfo{}}
			}

			record := RecordInfo{
				Name:  rr.Header().Name,
				Type:  dns.TypeToString[rr.Header().Rrtype],
				TTL:   rr.Header().Ttl,
				Value: strings.TrimPrefix(rr.String(), rr.Header().String()),
			}
			zoneMap[zoneName].Records = append(zoneMap[zoneName].Records, record)
		}
	}

	result := make([]ZoneInfo, 0, len(zoneMap))
	for _, zi := range zoneMap {
		result = append(result, *zi)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// getZonesInfoFromDB returns zone info from SQLite database with IDs
func getZonesInfoFromDB() []ZoneInfo {
	dbZones, err := database.ListZones()
	if err != nil {
		return nil
	}

	result := make([]ZoneInfo, 0, len(dbZones))
	for _, dbZone := range dbZones {
		zi := ZoneInfo{
			ID:      dbZone.ID,
			Name:    strings.TrimSuffix(dbZone.Name, "."),
			Enabled: dbZone.Enabled,
			Serial:  dbZone.Serial,
			Version: dbZone.Version,
			TTL:     dbZone.TTL,
			NS:      dbZone.NS,
			Admin:   dbZone.Admin,
			Refresh: dbZone.Refresh,
			Retry:   dbZone.Retry,
			Expire:  dbZone.Expire,
		}

		records, _ := database.ListRecordsByZone(dbZone.ID)
		for _, r := range records {
			zi.Records = append(zi.Records, RecordInfo{
				ID:       r.ID,
				Name:     r.Name,
				Type:     r.Type,
				Value:    r.Value,
				TTL:      uint32(r.TTL),
				Priority: r.Priority,
			})
		}

		result = append(result, zi)
	}

	return result
}

// findZoneForRecord finds the zone name for a given record
func findZoneForRecord(recordName string) string {
	for _, zoneName := range loadedZoneNames {
		if strings.HasSuffix(recordName, zoneName) || recordName == zoneName {
			return zoneName
		}
	}
	return ""
}

// Web handlers
func handleWebIndex(c *gin.Context) {
	tmpl := template.Must(template.New("index").Parse(headerHTML + sidebarHTML + indexHTML))
	zones := getZonesInfo()
	totalRecords := 0
	for _, z := range zones {
		totalRecords += len(z.Records)
	}
	canEdit := dbMode == "sqlite" && serverRole != "slave"
	data := struct {
		Zones           []ZoneInfo
		ZoneCount       int
		RecordCount     int
		Mode            string
		EditMode        bool
		CanEdit         bool
		Forwarders      []string
		CurrentPath     string
		PageTitle       string
		ShowSetupButton bool
	}{
		Zones:           zones,
		ZoneCount:       len(zones),
		RecordCount:     totalRecords,
		Mode:            dbMode,
		EditMode:        dbMode == "sqlite",
		CanEdit:         canEdit,
		Forwarders:      forwarders,
		CurrentPath:     "/",
		PageTitle:       "Dashboard",
		ShowSetupButton: true,
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		slog.Error("failed to render template", "error", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
	}
}

func handleWebZoneRecords(c *gin.Context) {
	zoneName := c.Param("zone")

	// Find the zone
	zones := getZonesInfo()
	var zone *ZoneInfo
	for i := range zones {
		if zones[i].Name == zoneName {
			zone = &zones[i]
			break
		}
	}

	if zone == nil {
		c.String(http.StatusNotFound, "Zone not found")
		return
	}

	tmpl := template.Must(template.New("zone_records").Parse(sidebarHTML + zoneRecordsHTML))
	canEdit := dbMode == "sqlite" && serverRole != "slave"
	data := struct {
		Zone        *ZoneInfo
		AllZones    []ZoneInfo
		Mode        string
		EditMode    bool
		CanEdit     bool
		CurrentPath string
	}{
		Zone:        zone,
		AllZones:    zones,
		Mode:        dbMode,
		EditMode:    dbMode == "sqlite",
		CanEdit:     canEdit,
		CurrentPath: "/zones",
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		slog.Error("failed to render template", "error", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
	}
}

func handleWebZoneSettings(c *gin.Context) {
	zoneName := c.Param("zone")

	// Find the zone
	zones := getZonesInfo()
	var zone *ZoneInfo
	for i := range zones {
		if zones[i].Name == zoneName {
			zone = &zones[i]
			break
		}
	}

	if zone == nil {
		c.String(http.StatusNotFound, "Zone not found")
		return
	}

	funcMap := template.FuncMap{
		"divideBy": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
	}
	tmpl := template.Must(template.New("zone_settings").Funcs(funcMap).Parse(sidebarHTML + zoneSettingsHTML))
	canEdit := dbMode == "sqlite" && serverRole != "slave"
	data := struct {
		Zone        *ZoneInfo
		AllZones    []ZoneInfo
		Mode        string
		EditMode    bool
		CanEdit     bool
		CurrentPath string
	}{
		Zone:        zone,
		AllZones:    zones,
		Mode:        dbMode,
		EditMode:    dbMode == "sqlite",
		CanEdit:     canEdit,
		CurrentPath: "/zones",
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		slog.Error("failed to render template", "error", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
	}
}

func handleWebSettings(c *gin.Context) {
	tmpl := template.Must(template.New("settings").Parse(headerHTML + sidebarHTML + globalSettingsHTML))
	data := struct {
		Mode            string
		EditMode        bool
		CanEdit         bool
		Forwarders      []string
		CurrentPath     string
		PageTitle       string
		ShowSetupButton bool
	}{
		Mode:            dbMode,
		EditMode:        dbMode == "sqlite",
		CanEdit:         dbMode == "sqlite" && serverRole != "slave",
		Forwarders:      forwarders,
		CurrentPath:     "/settings",
		PageTitle:       "Settings",
		ShowSetupButton: true,
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		slog.Error("failed to render template", "error", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
	}
}

func handleWebReplication(c *gin.Context) {
	tmpl := template.Must(template.New("replication").Parse(headerHTML + sidebarHTML + replicationHTML))
	data := struct {
		Mode            string
		EditMode        bool
		CanEdit         bool
		CurrentPath     string
		PageTitle       string
		ShowSetupButton bool
		MasterHost      string
		MasterPort      int
		SyncInterval    int
	}{
		Mode:            dbMode,
		EditMode:        dbMode == "sqlite",
		CanEdit:         dbMode == "sqlite" && serverRole != "slave",
		CurrentPath:     "/replication",
		PageTitle:       "Replication",
		ShowSetupButton: true,
		MasterHost:      masterAPIHost,
		MasterPort:      masterAPIPort,
		SyncInterval:    int(syncInterval.Seconds()),
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		slog.Error("failed to render template", "error", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
	}
}

func handleAPIZones(c *gin.Context) {
	c.JSON(http.StatusOK, getZonesInfo())
}

func handleAPIHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"mode":       dbMode,
		"zones":      len(loadedZoneNames),
		"forwarders": len(forwarders),
	})
}

// handleConfigModalJS serves the config modal JavaScript
func handleConfigModalJS(c *gin.Context) {
	c.Header("Content-Type", "application/javascript")
	c.String(http.StatusOK, configModalJS)
}

// handleAPIServerInfo returns server information including IP address, role, and ports
func handleAPIServerInfo(c *gin.Context) {
	// Try to get the server's IP from the request
	serverIP := c.Request.Host
	// Remove port if present
	if idx := strings.LastIndex(serverIP, ":"); idx != -1 {
		serverIP = serverIP[:idx]
	}
	// If it's localhost, try to get a better IP
	if serverIP == "localhost" || serverIP == "127.0.0.1" {
		serverIP = getOutboundIP()
	}
	lastContact := ""
	if !masterLastContact.IsZero() {
		lastContact = masterLastContact.Format(time.RFC3339)
	}

	c.JSON(http.StatusOK, gin.H{
		"ip":                  serverIP,
		"role":                serverRole,
		"dns_port":            dnsPort,
		"mode":                dbMode,
		"zones_count":         len(loadedZoneNames),
		"forwarders":          len(forwarders),
		"master_connected":    masterConnected,
		"master_last_contact": lastContact,
	})
}

// getOutboundIP gets the preferred outbound IP of this machine
func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer func() { _ = conn.Close() }()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// startWebServer starts the web interface server using Gin
func startWebServer(port int) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Static files (no auth required)
	router.GET("/static/config-modal.js", handleConfigModalJS)

	// Public routes (no auth required)
	router.GET("/login", handleLogin)
	router.POST("/login", handleLogin)
	router.GET("/setup", handleSetup)
	router.POST("/setup", handleSetup)
	router.GET("/logout", handleLogout)
	router.GET("/api/health", handleAPIHealth)

	// Protected routes (auth required)
	protected := router.Group("/")
	protected.Use(AuthMiddleware())
	{
		protected.GET("/", handleWebIndex)
		protected.GET("/settings", handleWebSettings)
		protected.GET("/replication", handleWebReplication)
		protected.GET("/account", handleAccount)
		protected.POST("/account", handleAccount)
		protected.POST("/account/tokens", handleCreateAPIToken)
		protected.DELETE("/account/tokens/:id", handleDeleteAPIToken)
		protected.GET("/account/tokens", handleListAPITokens)
		protected.GET("/zones/:zone/records", handleWebZoneRecords)
		protected.GET("/zones/:zone/settings", handleWebZoneSettings)
		protected.GET("/api/server-info", handleAPIServerInfo)
	}

	// Register CRUD routes only in sqlite mode, otherwise just read-only zones
	if dbMode == "sqlite" {
		registerAPIRoutes(router)
	} else {
		router.GET("/api/zones", handleAPIZones)
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	go func() {
		slog.Info("Starting web server", "addr", server.Addr, "mode", dbMode)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to start web server", "error", err)
		}
	}()

	return server
}

func handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	// Indicate recursion is available if we have forwarders configured
	if len(forwarders) > 0 {
		m.RecursionAvailable = true
	}

	if len(r.Question) == 0 {
		slog.Debug("Received empty query", "client", w.RemoteAddr())
		if err := w.WriteMsg(m); err != nil {
			slog.Debug("WriteMsg error on empty query", "client", w.RemoteAddr(), "error", err)
		}
		return
	}

	q := r.Question[0]
	name := q.Name
	qtype := q.Qtype
	t := dns.TypeToString[qtype]

	// Check if this query matches a loaded zone (log INFO for local, DEBUG for forwarded)
	isLocalZone := false
	for _, zoneName := range loadedZoneNames {
		if strings.HasSuffix(name, zoneName) || name == zoneName {
			isLocalZone = true
			break
		}
	}

	if isLocalZone {
		slog.Info("Received query", "client", w.RemoteAddr(), "name", name, "type", t)
	} else {
		slog.Debug("Received query", "client", w.RemoteAddr(), "name", name, "type", t)
	}

	answers := []dns.RR{}
	if rrlist, ok := zones[name]; ok {
		for _, rr := range rrlist {
			if qtype == dns.TypeANY || rr.Header().Rrtype == qtype {
				answers = append(answers, rr)
			}
			// If asked for A but we have a CNAME, include the CNAME
			if qtype == dns.TypeA && rr.Header().Rrtype == dns.TypeCNAME {
				answers = append(answers, rr)
			}
		}
	}

	if len(answers) == 0 {
		// Try forwarding if configured
		if len(forwarders) > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), forwardTimeout)
			defer cancel()
			if resp, err := forwardQuery(ctx, r); err == nil && resp != nil {
				slog.Debug("Forwarded query", "name", name, "client", w.RemoteAddr())
				// preserve original ID
				resp.Id = r.Id
				if err := w.WriteMsg(resp); err != nil {
					slog.Debug("failed to write forwarded response", "client", w.RemoteAddr(), "error", err)
				}
				return
			} else {
				slog.Debug("forwarding failed", "name", name, "error", err)
			}
		}

		m.Rcode = dns.RcodeNameError // NXDOMAIN
		if err := w.WriteMsg(m); err != nil {
			slog.Warn("Failed to send NXDOMAIN", "name", name, "client", w.RemoteAddr(), "error", err)
		} else {
			slog.Info("Sent NXDOMAIN", "name", name, "client", w.RemoteAddr())
		}
		return
	}

	m.Answer = append(m.Answer, answers...)
	if err := w.WriteMsg(m); err != nil {
		slog.Warn("Failed to send reply", "name", name, "client", w.RemoteAddr(), "error", err)
	} else {
		slog.Info("Replied", "name", name, "client", w.RemoteAddr(), "answers", len(m.Answer))
	}
}

func main() {
	// Use flag types that record whether they were set so flags can override config file
	var zonesDirFlag stringFlag
	var forwardersFlag stringFlag
	var configFileFlag stringFlag
	var masterHostFlag stringFlag
	var masterPortFlag int
	var masterTokenFlag stringFlag
	var logLevelFlag string

	// register flags with defaults
	configFileFlag.value = "config.yaml"
	zonesDirFlag.value = "zones"
	flag.Var(&configFileFlag, "config-file", "path to the configuration file (YAML format)")
	flag.Var(&zonesDirFlag, "zones-dir", "directory containing zone files (YAML format)")
	flag.Var(&forwardersFlag, "forwarders", "comma-separated upstream DNS servers (host[:port], default port 53)")
	flag.Var(&masterHostFlag, "master-api-host", "master server IP address for slave mode (e.g., 192.168.1.1)")
	flag.IntVar(&masterPortFlag, "master-api-port", 0, "master server API port for slave mode (default 8080)")
	flag.Var(&masterTokenFlag, "master-token", "sync token for authentication with master server")
	flag.StringVar(&logLevelFlag, "log-level", "info", "log level (debug, info, warn, error)")
	flag.Parse()

	// Configure slog based on log level
	var logLevel slog.Level
	switch strings.ToLower(logLevelFlag) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	// Create handler with the configured level
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	slog.SetDefault(slog.New(handler))

	slog.Info("Starting simple DNS server")

	// Web server config (defaults)
	webEnabled := false
	webPort := 8080
	dbPath := "simpledns.db"

	// Load optional app config file if present
	if cfgApp, err := loadAppConfig(configFileFlag.value); err == nil {
		// Set db_type mode (files or sqlite)
		if cfgApp.DBType != "" {
			dbMode = cfgApp.DBType
		}
		if cfgApp.DBPath != "" {
			dbPath = cfgApp.DBPath
		}

		if !zonesDirFlag.set && cfgApp.ZonesDir != "" {
			zonesDirFlag.value = cfgApp.ZonesDir
		}
		if !forwardersFlag.set && cfgApp.Forwarders != nil {
			parsed := make([]string, 0, len(cfgApp.Forwarders))
			for _, p := range cfgApp.Forwarders {
				if p == "" {
					continue
				}
				if !strings.Contains(p, ":") {
					p = p + ":53"
				}
				parsed = append(parsed, p)
			}
			forwarders = parsed
		}
		if cfgApp.ForwardTimeoutSec > 0 {
			forwardTimeout = time.Duration(cfgApp.ForwardTimeoutSec) * time.Second
		}
		// Web server config
		webEnabled = cfgApp.WebEnabled
		if cfgApp.WebPort > 0 {
			webPort = cfgApp.WebPort
		}
		// Server role and DNS port config
		if cfgApp.ServerRole != "" {
			serverRole = cfgApp.ServerRole
		}
		if cfgApp.DNSPort > 0 {
			dnsPort = cfgApp.DNSPort
		}
		// Slave sync config
		if cfgApp.MasterAPIHost != "" {
			masterAPIHost = cfgApp.MasterAPIHost
		}
		if cfgApp.MasterAPIPort > 0 {
			masterAPIPort = cfgApp.MasterAPIPort
		}
		if cfgApp.MasterToken != "" {
			masterToken = cfgApp.MasterToken
		}
		if cfgApp.SyncIntervalSec > 0 {
			syncInterval = time.Duration(cfgApp.SyncIntervalSec) * time.Second
		}
	}

	// CLI flags override config
	if forwardersFlag.set {
		forwarders = parseForwarders(forwardersFlag.value)
	}
	if masterHostFlag.set {
		masterAPIHost = masterHostFlag.value
	}
	if masterPortFlag > 0 {
		masterAPIPort = masterPortFlag
	}
	if masterTokenFlag.set {
		masterToken = masterTokenFlag.value
	}

	if forwarders == nil {
		forwarders = []string{}
	}

	// Initialize based on db_type mode
	if dbMode == "sqlite" {
		slog.Info("Running in SQLite mode", "db_path", dbPath)
		if err := InitDatabase(dbPath); err != nil {
			slog.Error("failed to initialize database", "error", err)
			os.Exit(1)
		}
		// Load zones and forwarders from database
		if err := ReloadFromDB(); err != nil {
			slog.Warn("failed to load from database", "error", err)
		}
	} else {
		slog.Info("Running in files mode", "zones_dir", zonesDirFlag.value)
		initZones(zonesDirFlag.value)
	}

	// Always log the effective configuration and loaded zone names at startup
	uniq := make(map[string]struct{}, len(loadedZoneNames))
	for _, z := range loadedZoneNames {
		if z == "" {
			continue
		}
		uniq[z] = struct{}{}
	}
	zoneNames := make([]string, 0, len(uniq))
	for z := range uniq {
		zoneNames = append(zoneNames, z)
	}
	sort.Strings(zoneNames)
	slog.Info("Config initialized", "role", serverRole, "mode", dbMode, "dns_port", dnsPort, "forwarders", len(forwarders), "forward_timeout", forwardTimeout, "loaded_zones", len(zoneNames))
	if len(zoneNames) > 0 {
		slog.Info("Loaded zones", "zones", zoneNames)
	} else {
		slog.Info("No zones loaded - use API to add zones")
	}

	dns.HandleFunc(".", handleDNS)

	dnsAddr := fmt.Sprintf(":%d", dnsPort)
	udpServer := &dns.Server{Addr: dnsAddr, Net: "udp"}
	tcpServer := &dns.Server{Addr: dnsAddr, Net: "tcp"}

	// Start web server if enabled
	var webServer *http.Server
	if webEnabled {
		// set global for replication registration
		webServerPort = webPort
		webServer = startWebServer(webPort)
	}

	// Start slave sync if in slave mode
	if serverRole == "slave" && masterAPIHost != "" && masterToken != "" {
		StartSlaveSync()
	}

	// Run servers in goroutines
	go func() {
		slog.Info("Starting UDP server", "addr", udpServer.Addr)
		if err := udpServer.ListenAndServe(); err != nil {
			slog.Error("failed to start UDP server", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		slog.Info("Starting TCP server", "addr", tcpServer.Addr)
		if err := tcpServer.ListenAndServe(); err != nil {
			slog.Error("failed to start TCP server", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for signal to shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("Shutting down servers...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = udpServer.ShutdownContext(ctx)
	_ = tcpServer.ShutdownContext(ctx)
	if webServer != nil {
		_ = webServer.Shutdown(ctx)
	}
	if database != nil {
		_ = database.Close()
	}
	slog.Info("Servers stopped")
}

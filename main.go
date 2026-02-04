package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"log/slog"
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
	Records []RecordInfo `json:"records"`
}

// RecordInfo represents a DNS record for the web interface
type RecordInfo struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   uint32 `json:"ttl"`
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
				zoneMap[zoneName] = &ZoneInfo{Name: zoneName, Records: []RecordInfo{}}
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
			ID:   dbZone.ID,
			Name: dbZone.Name,
		}

		records, _ := database.ListRecordsByZone(dbZone.ID)
		for _, r := range records {
			zi.Records = append(zi.Records, RecordInfo{
				ID:    r.ID,
				Name:  r.Name,
				Type:  r.Type,
				Value: r.Value,
				TTL:   uint32(r.TTL),
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
	tmpl := template.Must(template.New("index").Parse(indexHTML))
	data := struct {
		Zones       []ZoneInfo
		ZoneCount   int
		RecordCount int
		Mode        string
		EditMode    bool
		Forwarders  []string
	}{
		Zones:      getZonesInfo(),
		ZoneCount:  len(loadedZoneNames),
		Mode:       dbMode,
		EditMode:   dbMode == "sqlite",
		Forwarders: forwarders,
	}
	for _, z := range data.Zones {
		data.RecordCount += len(z.Records)
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

// startWebServer starts the web interface server using Gin
func startWebServer(port int) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Routes
	router.GET("/", handleWebIndex)
	router.GET("/api/health", handleAPIHealth)

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

// HTML template for web interface
const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SimpleDNS - Zone Viewer</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        primary: '#3498db',
                        'primary-dark': '#2980b9',
                    }
                }
            }
        }
    </script>
    <style type="text/tailwindcss">
        @layer utilities {
            .toggle { transition: transform 0.3s; }
            .toggle.open { transform: rotate(180deg); }
        }
    </style>
</head>
<body class="bg-gray-100 text-gray-800 font-sans">
    <div class="max-w-6xl mx-auto p-5">
        <!-- Header -->
        <header class="bg-slate-800 text-white p-5 mb-5 rounded-lg">
            <div class="flex justify-between items-start">
                <div>
                    <h1 class="text-2xl font-bold">🌐 SimpleDNS Zone Viewer</h1>
                    <div class="flex gap-4 mt-3">
                        <span class="bg-white/10 px-4 py-2 rounded">📁 Zones: {{.ZoneCount}}</span>
                        <span class="bg-white/10 px-4 py-2 rounded">📝 Records: {{.RecordCount}}</span>
                        <span class="bg-white/10 px-4 py-2 rounded">⚙️ Mode: {{.Mode}}</span>
                    </div>
                </div>
                {{if .EditMode}}
                <button onclick="showAddZoneModal()" class="bg-green-500 hover:bg-green-600 px-4 py-2 rounded font-medium transition-colors">+ Add Zone</button>
                {{end}}
            </div>
        </header>
        
        {{if .EditMode}}
        <!-- Add Zone Modal -->
        <div id="addZoneModal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
            <div class="bg-white rounded-lg p-6 w-full max-w-md mx-4">
                <h2 class="text-xl font-bold mb-4">Add New Zone</h2>
                <form id="addZoneForm" onsubmit="submitZone(event)">
                    <div class="mb-4">
                        <label class="block text-sm font-medium mb-1">Zone Name *</label>
                        <input type="text" name="name" required placeholder="example.com" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                    </div>
                    <div class="grid grid-cols-2 gap-4 mb-4">
                        <div>
                            <label class="block text-sm font-medium mb-1">TTL</label>
                            <input type="number" name="ttl" value="3600" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                        </div>
                        <div>
                            <label class="block text-sm font-medium mb-1">NS Server</label>
                            <input type="text" name="ns" placeholder="ns1.example.com" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                        </div>
                    </div>
                    <div class="flex gap-2 justify-end">
                        <button type="button" onclick="hideAddZoneModal()" class="px-4 py-2 border rounded hover:bg-gray-100">Cancel</button>
                        <button type="submit" class="px-4 py-2 bg-primary text-white rounded hover:bg-primary-dark">Create Zone</button>
                    </div>
                </form>
            </div>
        </div>
        
        <!-- Add Record Modal -->
        <div id="addRecordModal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
            <div class="bg-white rounded-lg p-6 w-full max-w-md mx-4">
                <h2 class="text-xl font-bold mb-4">Add Record to <span id="recordZoneName"></span></h2>
                <form id="addRecordForm" onsubmit="submitRecord(event)">
                    <input type="hidden" name="zone_id" id="recordZoneId">
                    <input type="hidden" name="zone_index" id="recordZoneIndex">
                    <div class="mb-4">
                        <label class="block text-sm font-medium mb-1">Name *</label>
                        <input type="text" name="name" required placeholder="www or @" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                    </div>
                    <div class="grid grid-cols-2 gap-4 mb-4">
                        <div>
                            <label class="block text-sm font-medium mb-1">Type *</label>
                            <select name="type" required class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                                <option value="A">A</option>
                                <option value="AAAA">AAAA</option>
                                <option value="CNAME">CNAME</option>
                                <option value="MX">MX</option>
                                <option value="TXT">TXT</option>
                                <option value="NS">NS</option>
                            </select>
                        </div>
                        <div>
                            <label class="block text-sm font-medium mb-1">TTL</label>
                            <input type="number" name="ttl" value="3600" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                        </div>
                    </div>
                    <div class="mb-4">
                        <label class="block text-sm font-medium mb-1">Value *</label>
                        <input type="text" name="value" required placeholder="192.168.1.1" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                    </div>
                    <div class="flex gap-2 justify-end">
                        <button type="button" onclick="hideAddRecordModal()" class="px-4 py-2 border rounded hover:bg-gray-100">Cancel</button>
                        <button type="submit" class="px-4 py-2 bg-primary text-white rounded hover:bg-primary-dark">Add Record</button>
                    </div>
                </form>
            </div>
        </div>
        
        <!-- Edit Record Modal -->
        <div id="editRecordModal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
            <div class="bg-white rounded-lg p-6 w-full max-w-md mx-4">
                <h2 class="text-xl font-bold mb-4">Edit Record</h2>
                <form id="editRecordForm" onsubmit="submitEditRecord(event)">
                    <input type="hidden" name="record_id" id="editRecordId">
                    <input type="hidden" name="row_element" id="editRecordRow">
                    <div class="mb-4">
                        <label class="block text-sm font-medium mb-1">Name *</label>
                        <input type="text" name="name" id="editRecordName" required placeholder="www or @" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                    </div>
                    <div class="grid grid-cols-2 gap-4 mb-4">
                        <div>
                            <label class="block text-sm font-medium mb-1">Type *</label>
                            <select name="type" id="editRecordType" required class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                                <option value="A">A</option>
                                <option value="AAAA">AAAA</option>
                                <option value="CNAME">CNAME</option>
                                <option value="MX">MX</option>
                                <option value="TXT">TXT</option>
                                <option value="NS">NS</option>
                            </select>
                        </div>
                        <div>
                            <label class="block text-sm font-medium mb-1">TTL</label>
                            <input type="number" name="ttl" id="editRecordTTL" value="3600" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                        </div>
                    </div>
                    <div class="mb-4">
                        <label class="block text-sm font-medium mb-1">Value *</label>
                        <input type="text" name="value" id="editRecordValue" required placeholder="192.168.1.1" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                    </div>
                    <div class="flex gap-2 justify-end">
                        <button type="button" onclick="hideEditRecordModal()" class="px-4 py-2 border rounded hover:bg-gray-100">Cancel</button>
                        <button type="submit" class="px-4 py-2 bg-primary text-white rounded hover:bg-primary-dark">Save Changes</button>
                    </div>
                </form>
            </div>
        </div>
        {{end}}
        
        <!-- Filters -->
        <div class="bg-white rounded-lg p-4 mb-5 shadow flex gap-2 items-center flex-wrap">
            <label class="font-medium mr-2">Filter by type:</label>
            <button class="filter-btn px-3 py-1.5 border-2 border-primary bg-primary text-white rounded cursor-pointer text-sm transition-all" data-filter="all">All</button>
            <button class="filter-btn px-3 py-1.5 border-2 border-gray-300 bg-white rounded cursor-pointer text-sm transition-all hover:border-primary" data-filter="A">A</button>
            <button class="filter-btn px-3 py-1.5 border-2 border-gray-300 bg-white rounded cursor-pointer text-sm transition-all hover:border-primary" data-filter="AAAA">AAAA</button>
            <button class="filter-btn px-3 py-1.5 border-2 border-gray-300 bg-white rounded cursor-pointer text-sm transition-all hover:border-primary" data-filter="CNAME">CNAME</button>
            <button class="filter-btn px-3 py-1.5 border-2 border-gray-300 bg-white rounded cursor-pointer text-sm transition-all hover:border-primary" data-filter="MX">MX</button>
            <button class="filter-btn px-3 py-1.5 border-2 border-gray-300 bg-white rounded cursor-pointer text-sm transition-all hover:border-primary" data-filter="TXT">TXT</button>
            <button class="filter-btn px-3 py-1.5 border-2 border-gray-300 bg-white rounded cursor-pointer text-sm transition-all hover:border-primary" data-filter="NS">NS</button>
            <button class="filter-btn px-3 py-1.5 border-2 border-gray-300 bg-white rounded cursor-pointer text-sm transition-all hover:border-primary" data-filter="SOA">SOA</button>
        </div>
        
        <!-- Zones -->
        {{if .Zones}}
            {{range $index, $zone := .Zones}}
            <div class="bg-white rounded-lg mb-3 shadow overflow-hidden" data-zone-id="{{$zone.ID}}">
                <div class="bg-primary hover:bg-primary-dark text-white p-4 font-bold text-lg cursor-pointer flex justify-between items-center transition-colors">
                    <span onclick="toggleZone({{$index}})" class="flex-1">{{$zone.Name}} <span class="text-sm opacity-90 font-normal" id="count-{{$index}}">({{len $zone.Records}} records)</span></span>
                    <div class="flex items-center gap-2">
                        {{if $.EditMode}}
                        <button onclick="event.stopPropagation(); showAddRecordModal({{$zone.ID}}, '{{$zone.Name}}', {{$index}})" class="text-sm bg-white/20 hover:bg-white/30 px-2 py-1 rounded">+ Record</button>
                        <button onclick="event.stopPropagation(); deleteZone({{$zone.ID}}, '{{$zone.Name}}')" class="text-sm bg-red-500/80 hover:bg-red-600 px-2 py-1 rounded">🗑</button>
                        {{end}}
                        <span class="toggle text-xl" id="toggle-{{$index}}" onclick="toggleZone({{$index}})">▼</span>
                    </div>
                </div>
                <div class="hidden" id="zone-{{$index}}">
                    <table class="w-full">
                        <thead>
                            <tr class="bg-gray-50">
                                <th class="px-5 py-3 text-left font-semibold text-gray-600 border-b">Name</th>
                                <th class="px-5 py-3 text-left font-semibold text-gray-600 border-b">Type</th>
                                <th class="px-5 py-3 text-left font-semibold text-gray-600 border-b">Value</th>
                                <th class="px-5 py-3 text-left font-semibold text-gray-600 border-b">TTL</th>
                                {{if $.EditMode}}<th class="px-5 py-3 text-left font-semibold text-gray-600 border-b w-16">Actions</th>{{end}}
                            </tr>
                        </thead>
                        <tbody id="tbody-{{$index}}">
                            {{range $zone.Records}}
                            <tr data-type="{{.Type}}" data-record-id="{{.ID}}" class="hover:bg-gray-50 border-b border-gray-100">
                                <td class="px-5 py-3 font-mono text-sm" data-field="name">{{.Name}}</td>
                                <td class="px-5 py-3" data-field="type">
                                    {{if eq .Type "A"}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-blue-100 text-blue-700">{{.Type}}</span>
                                    {{else if eq .Type "AAAA"}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-green-100 text-green-700">{{.Type}}</span>
                                    {{else if eq .Type "CNAME"}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-orange-100 text-orange-700">{{.Type}}</span>
                                    {{else if eq .Type "MX"}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-pink-100 text-pink-700">{{.Type}}</span>
                                    {{else if eq .Type "TXT"}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-purple-100 text-purple-700">{{.Type}}</span>
                                    {{else if eq .Type "NS"}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-teal-100 text-teal-700">{{.Type}}</span>
                                    {{else if eq .Type "SOA"}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-gray-200 text-gray-700">{{.Type}}</span>
                                    {{else}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-gray-100 text-gray-600">{{.Type}}</span>
                                    {{end}}
                                </td>
                                <td class="px-5 py-3 font-mono text-sm text-gray-600" data-field="value">{{.Value}}</td>
                                <td class="px-5 py-3 text-gray-500" data-field="ttl">{{.TTL}}</td>
                                {{if $.EditMode}}<td class="px-5 py-3 flex gap-1">
                                    <button onclick="showEditRecordModal({{.ID}}, this)" class="text-blue-500 hover:text-blue-700" title="Edit">✏️</button>
                                    <button onclick="deleteRecord({{.ID}}, this)" class="text-red-500 hover:text-red-700" title="Delete">🗑</button>
                                </td>{{end}}
                            </tr>
                            {{end}}
                        </tbody>
                    </table>
                </div>
            </div>
            {{end}}
        {{else}}
            <div class="bg-white rounded-lg shadow">
                <div class="text-center py-10 text-gray-400">No zones loaded</div>
            </div>
        {{end}}
        
        <!-- Forwarders Section -->
        <div class="bg-white rounded-lg shadow mb-5 mt-5">
            <div class="flex justify-between items-center px-5 py-4 border-b">
                <h2 class="text-lg font-bold text-gray-700">🔀 Forwarders ({{len .Forwarders}})</h2>
                {{if .EditMode}}
                <button onclick="showAddForwarderModal()" class="bg-green-500 hover:bg-green-600 text-white px-3 py-1 rounded text-sm">+ Add</button>
                {{end}}
            </div>
            <div class="p-5">
                {{if .Forwarders}}
                <div class="flex flex-wrap gap-2" id="forwarders-list">
                    {{range .Forwarders}}
                    <div class="flex items-center gap-2 bg-gray-100 px-3 py-2 rounded" data-forwarder="{{.}}">
                        <span class="font-mono text-sm">{{.}}</span>
                        {{if $.EditMode}}
                        <button onclick="deleteForwarder('{{.}}', this)" class="text-red-500 hover:text-red-700 text-sm">✕</button>
                        {{end}}
                    </div>
                    {{end}}
                </div>
                {{else}}
                <div class="text-center text-gray-400" id="no-forwarders">No forwarders configured</div>
                {{end}}
            </div>
        </div>
        
        {{if .EditMode}}
        <!-- Add Forwarder Modal -->
        <div id="addForwarderModal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
            <div class="bg-white rounded-lg p-6 w-full max-w-md mx-4">
                <h2 class="text-xl font-bold mb-4">Add Forwarder</h2>
                <form id="addForwarderForm" onsubmit="submitForwarder(event)">
                    <div class="mb-4">
                        <label class="block text-sm font-medium mb-1">DNS Server Address *</label>
                        <input type="text" name="address" required placeholder="8.8.8.8 or 8.8.8.8:53" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                        <p class="text-xs text-gray-500 mt-1">IP address or hostname, optionally with port (default: 53)</p>
                    </div>
                    <div class="flex gap-2 justify-end">
                        <button type="button" onclick="hideAddForwarderModal()" class="px-4 py-2 border rounded hover:bg-gray-100">Cancel</button>
                        <button type="submit" class="px-4 py-2 bg-primary text-white rounded hover:bg-primary-dark">Add Forwarder</button>
                    </div>
                </form>
            </div>
        </div>
        {{end}}
        
        <!-- Footer -->
        <footer class="text-center py-5 text-gray-400 text-sm">
            SimpleDNS &bull; <a href="/api/zones" class="text-primary hover:underline">API</a> &bull; <a href="/api/health" class="text-primary hover:underline">Health</a>
        </footer>
    </div>
    
    <script>
        // Toggle zone visibility using Tailwind's hidden class
        function toggleZone(index) {
            const records = document.getElementById('zone-' + index);
            const toggle = document.getElementById('toggle-' + index);
            records.classList.toggle('hidden');
            toggle.classList.toggle('open');
        }
        
        // Filter functionality
        let activeFilter = 'all';
        document.querySelectorAll('.filter-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.filter-btn').forEach(b => {
                    b.classList.remove('bg-primary', 'text-white', 'border-primary');
                    b.classList.add('bg-white', 'border-gray-300');
                });
                btn.classList.remove('bg-white', 'border-gray-300');
                btn.classList.add('bg-primary', 'text-white', 'border-primary');
                activeFilter = btn.dataset.filter;
                applyFilter();
            });
        });
        
        function applyFilter() {
            document.querySelectorAll('tr[data-type]').forEach(row => {
                if (activeFilter === 'all' || row.dataset.type === activeFilter) {
                    row.classList.remove('hidden');
                } else {
                    row.classList.add('hidden');
                }
            });
        }
        
        // Modal functions (only used in sqlite mode)
        function showAddZoneModal() {
            document.getElementById('addZoneModal').classList.remove('hidden');
            document.getElementById('addZoneModal').classList.add('flex');
        }
        function hideAddZoneModal() {
            document.getElementById('addZoneModal').classList.add('hidden');
            document.getElementById('addZoneModal').classList.remove('flex');
            document.getElementById('addZoneForm').reset();
        }
        function showAddRecordModal(zoneId, zoneName, zoneIndex) {
            document.getElementById('recordZoneId').value = zoneId;
            document.getElementById('recordZoneIndex').value = zoneIndex;
            document.getElementById('recordZoneName').textContent = zoneName;
            document.getElementById('addRecordModal').classList.remove('hidden');
            document.getElementById('addRecordModal').classList.add('flex');
        }
        function hideAddRecordModal() {
            document.getElementById('addRecordModal').classList.add('hidden');
            document.getElementById('addRecordModal').classList.remove('flex');
            document.getElementById('addRecordForm').reset();
        }
        
        async function submitZone(e) {
            e.preventDefault();
            const form = e.target;
            const data = {
                name: form.name.value,
                ttl: parseInt(form.ttl.value) || 3600,
                ns: form.ns.value || ''
            };
            try {
                const resp = await fetch('/api/zones', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(data)
                });
                if (resp.ok) {
                    window.location.reload();
                } else {
                    const err = await resp.json();
                    alert('Error: ' + (err.error || 'Failed to create zone'));
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
        
        async function submitRecord(e) {
            e.preventDefault();
            const form = e.target;
            const zoneId = form.zone_id.value;
            const zoneIndex = form.zone_index.value;
            const recordName = form.name.value;
            const recordType = form.type.value;
            const recordValue = form.value.value;
            const recordTTL = parseInt(form.ttl.value) || 3600;
            
            const data = {
                name: recordName,
                type: recordType,
                value: recordValue,
                ttl: recordTTL
            };
            
            try {
                const resp = await fetch('/api/zones/' + zoneId + '/records', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(data)
                });
                if (resp.ok) {
                    const record = await resp.json();
                    // Add row to table dynamically
                    addRecordRow(zoneIndex, record.id, recordName, recordType, recordValue, recordTTL);
                    // Update record count
                    updateRecordCount(zoneIndex, 1);
                    // Ensure zone is expanded
                    const zoneRecords = document.getElementById('zone-' + zoneIndex);
                    zoneRecords.classList.remove('hidden');
                    document.getElementById('toggle-' + zoneIndex).classList.add('open');
                    // Close modal and reset form
                    hideAddRecordModal();
                } else {
                    const err = await resp.json();
                    alert('Error: ' + (err.error || 'Failed to create record'));
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
        
        function addRecordRow(zoneIndex, recordId, name, type, value, ttl) {
            const tbody = document.getElementById('tbody-' + zoneIndex);
            const row = document.createElement('tr');
            row.setAttribute('data-type', type);
            row.setAttribute('data-record-id', recordId);
            row.className = 'hover:bg-gray-50 border-b border-gray-100 bg-green-50';
            row.innerHTML = ` + "`" + `
                <td class="px-5 py-3 font-mono text-sm" data-field="name">${name}</td>
                <td class="px-5 py-3" data-field="type">${getTypeBadge(type)}</td>
                <td class="px-5 py-3 font-mono text-sm text-gray-600" data-field="value">${value}</td>
                <td class="px-5 py-3 text-gray-500" data-field="ttl">${ttl}</td>
                <td class="px-5 py-3 flex gap-1">
                    <button onclick="showEditRecordModal(${recordId}, this)" class="text-blue-500 hover:text-blue-700" title="Edit">✏️</button>
                    <button onclick="deleteRecord(${recordId}, this)" class="text-red-500 hover:text-red-700" title="Delete">🗑</button>
                </td>
            ` + "`" + `;
            tbody.appendChild(row);
            // Remove highlight after animation
            setTimeout(() => row.classList.remove('bg-green-50'), 2000);
        }
        
        function updateRecordCount(zoneIndex, delta) {
            const countEl = document.getElementById('count-' + zoneIndex);
            const match = countEl.textContent.match(/\((\d+)/);
            if (match) {
                const newCount = parseInt(match[1]) + delta;
                countEl.textContent = '(' + newCount + ' records)';
            }
        }
        
        async function deleteZone(id, name) {
            if (!confirm('Delete zone ' + name + ' and all its records?')) return;
            try {
                const resp = await fetch('/api/zones/' + id, { method: 'DELETE' });
                if (resp.ok) {
                    window.location.reload();
                } else {
                    alert('Failed to delete zone');
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
        
        function showEditRecordModal(id, btn) {
            const row = btn.closest('tr');
            document.getElementById('edit_record_id').value = id;
            document.getElementById('edit_record_name').value = row.querySelector('[data-field="name"]').textContent;
            document.getElementById('edit_record_type').value = row.querySelector('[data-field="type"]').textContent.trim();
            document.getElementById('edit_record_value').value = row.querySelector('[data-field="value"]').textContent;
            document.getElementById('edit_record_ttl').value = row.querySelector('[data-field="ttl"]').textContent;
            document.getElementById('editRecordModal').classList.remove('hidden');
        }
        
        function hideEditRecordModal() {
            document.getElementById('editRecordModal').classList.add('hidden');
        }
        
        async function submitEditRecord(event) {
            event.preventDefault();
            const id = document.getElementById('edit_record_id').value;
            const data = {
                name: document.getElementById('edit_record_name').value,
                type: document.getElementById('edit_record_type').value,
                value: document.getElementById('edit_record_value').value,
                ttl: parseInt(document.getElementById('edit_record_ttl').value) || 3600
            };
            try {
                const resp = await fetch('/api/records/' + id, {
                    method: 'PUT',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(data)
                });
                if (resp.ok) {
                    // Update row in DOM
                    const row = document.querySelector('tr[data-record-id="' + id + '"]');
                    if (row) {
                        row.querySelector('[data-field="name"]').textContent = data.name;
                        row.querySelector('[data-field="type"]').innerHTML = getTypeBadge(data.type);
                        row.querySelector('[data-field="value"]').textContent = data.value;
                        row.querySelector('[data-field="ttl"]').textContent = data.ttl;
                    }
                    hideEditRecordModal();
                } else {
                    const err = await resp.json();
                    alert('Failed to update record: ' + (err.error || 'Unknown error'));
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
        
        function getTypeBadge(type) {
            const colors = {
                'A': 'bg-blue-100 text-blue-700',
                'AAAA': 'bg-green-100 text-green-700',
                'CNAME': 'bg-orange-100 text-orange-700',
                'MX': 'bg-pink-100 text-pink-700',
                'TXT': 'bg-purple-100 text-purple-700',
                'NS': 'bg-teal-100 text-teal-700',
                'SOA': 'bg-gray-200 text-gray-700'
            };
            const color = colors[type] || 'bg-gray-100 text-gray-600';
            return '<span class="px-2 py-0.5 rounded text-sm font-medium ' + color + '">' + type + '</span>';
        }
        
        async function deleteRecord(id, btn) {
            if (!confirm('Delete this record?')) return;
            try {
                const resp = await fetch('/api/records/' + id, { method: 'DELETE' });
                if (resp.ok) {
                    // Remove row from DOM
                    const row = btn.closest('tr');
                    const tbody = row.parentElement;
                    const zoneIndex = tbody.id.replace('tbody-', '');
                    row.remove();
                    // Update count
                    updateRecordCount(zoneIndex, -1);
                } else {
                    alert('Failed to delete record');
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
        
        // Forwarder functions
        function showAddForwarderModal() {
            document.getElementById('addForwarderModal').classList.remove('hidden');
            document.getElementById('addForwarderModal').classList.add('flex');
        }
        
        function hideAddForwarderModal() {
            document.getElementById('addForwarderModal').classList.add('hidden');
            document.getElementById('addForwarderModal').classList.remove('flex');
            document.getElementById('addForwarderForm').reset();
        }
        
        async function submitForwarder(event) {
            event.preventDefault();
            const form = event.target;
            let address = form.address.value.trim();
            // Add default port if not specified
            if (!address.includes(':')) {
                address = address + ':53';
            }
            try {
                const resp = await fetch('/api/forwarders', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({ address: address })
                });
                if (resp.ok) {
                    // Add to DOM
                    addForwarderToList(address);
                    hideAddForwarderModal();
                } else {
                    const err = await resp.json();
                    alert('Failed to add forwarder: ' + (err.error || 'Unknown error'));
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
        
        function addForwarderToList(address) {
            const list = document.getElementById('forwarders-list');
            const noForwarders = document.getElementById('no-forwarders');
            if (noForwarders) noForwarders.remove();
            
            if (!list) {
                // Create the list container if it doesn't exist
                const container = document.querySelector('.bg-white.rounded-lg.shadow.mb-5.mt-5 .p-5');
                const newList = document.createElement('div');
                newList.className = 'flex flex-wrap gap-2';
                newList.id = 'forwarders-list';
                container.appendChild(newList);
            }
            
            const div = document.createElement('div');
            div.className = 'flex items-center gap-2 bg-gray-100 px-3 py-2 rounded bg-green-100';
            div.setAttribute('data-forwarder', address);
            div.innerHTML = '<span class="font-mono text-sm">' + address + '</span>' +
                '<button onclick="deleteForwarder(\'' + address + '\', this)" class="text-red-500 hover:text-red-700 text-sm">✕</button>';
            document.getElementById('forwarders-list').appendChild(div);
            setTimeout(() => div.classList.remove('bg-green-100'), 2000);
        }
        
        async function deleteForwarder(address, btn) {
            if (!confirm('Remove forwarder ' + address + '?')) return;
            try {
                const resp = await fetch('/api/forwarders/' + encodeURIComponent(address), { method: 'DELETE' });
                if (resp.ok) {
                    btn.closest('[data-forwarder]').remove();
                } else {
                    alert('Failed to remove forwarder');
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
    </script>
</body>
</html>
`

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
	var logLevelFlag string

	// register flags with defaults
	configFileFlag.value = "config.yaml"
	zonesDirFlag.value = "zones"
	flag.Var(&configFileFlag, "config-file", "path to the configuration file (YAML format)")
	flag.Var(&zonesDirFlag, "zones-dir", "directory containing zone files (YAML format)")
	flag.Var(&forwardersFlag, "forwarders", "comma-separated upstream DNS servers (host[:port], default port 53)")
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
	}

	// CLI flags override config
	if forwardersFlag.set {
		forwarders = parseForwarders(forwardersFlag.value)
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
	slog.Info("Config initialized", "mode", dbMode, "forwarders", len(forwarders), "forward_timeout", forwardTimeout, "loaded_zones", len(zoneNames))
	if len(zoneNames) > 0 {
		slog.Info("Loaded zones", "zones", zoneNames)
	} else {
		slog.Info("No zones loaded - use API to add zones")
	}

	dns.HandleFunc(".", handleDNS)

	udpServer := &dns.Server{Addr: ":53", Net: "udp"}
	tcpServer := &dns.Server{Addr: ":53", Net: "tcp"}

	// Start web server if enabled
	var webServer *http.Server
	if webEnabled {
		webServer = startWebServer(webPort)
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

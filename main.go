package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/miekg/dns"
	"gopkg.in/yaml.v3"
)

var zones map[string][]dns.RR
var debug bool
var forwarders []string
var forwardTimeout time.Duration = 2 * time.Second
var loadedZoneNames []string

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
	ZonesDir          string   `yaml:"zones_dir" json:"zones_dir,omitempty"`
	Forwarders        []string `json:"forwarders,omitempty"`
	ForwardTimeoutSec int      `json:"forward_timeout_seconds,omitempty"`
	Addr              string   `json:"addr,omitempty"`
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
			if debug {
				log.Printf("forward to %s failed: %v", srv, err)
			}
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
				log.Printf("Loaded zones from directory %s", confDir)
				return
			} else {
				log.Printf("Failed to load zones from dir %s: %v", confDir, err)
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

func handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	// Indicate recursion is available if we have forwarders configured
	if len(forwarders) > 0 {
		m.RecursionAvailable = true
	}

	if len(r.Question) == 0 {
		if debug {
			log.Printf("Received empty query from %s", w.RemoteAddr())
		}
		if err := w.WriteMsg(m); err != nil && debug {
			log.Printf("WriteMsg error (empty query) to %s: %v", w.RemoteAddr(), err)
		}
		return
	}

	q := r.Question[0]
	name := q.Name
	qtype := q.Qtype

	if debug {
		t := dns.TypeToString[qtype]
		log.Printf("Received query from %s: %s %s", w.RemoteAddr(), name, t)
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
				if debug {
					log.Printf("Forwarded %s to upstream, replying to %s", name, w.RemoteAddr())
				}
				// preserve original ID
				resp.Id = r.Id
				if err := w.WriteMsg(resp); err != nil && debug {
					log.Printf("failed to write forwarded response to %s: %v", w.RemoteAddr(), err)
				}
				return
			} else if debug {
				log.Printf("forwarding failed for %s: %v", name, err)
			}
		}

		m.Rcode = dns.RcodeNameError // NXDOMAIN
		if err := w.WriteMsg(m); err != nil {
			if debug {
				log.Printf("Failed to send NXDOMAIN for %s to %s: %v", name, w.RemoteAddr(), err)
			}
		} else if debug {
			log.Printf("Sent NXDOMAIN for %s to %s", name, w.RemoteAddr())
		}
		return
	}

	m.Answer = append(m.Answer, answers...)
	if err := w.WriteMsg(m); err != nil {
		if debug {
			log.Printf("Failed to send reply for %s to %s: %v", name, w.RemoteAddr(), err)
		}
	} else if debug {
		log.Printf("Replied to %s: %d answer(s) for %s", w.RemoteAddr(), len(m.Answer), name)
	}
}

func main() {
	log.Println("Starting simple DNS server...")
	// Use flag types that record whether they were set so flags can override config file
	var zonesDirFlag stringFlag
	var forwardersFlag stringFlag
	var configFileFlag stringFlag

	// register flags with defaults
	configFileFlag.value = "config.yaml"
	zonesDirFlag.value = "conf"
	flag.Var(&configFileFlag, "config-file", "path to the configuration file (YAML format)")
	flag.Var(&zonesDirFlag, "zones-dir", "directory containing zone files (YAML format)")
	flag.Var(&forwardersFlag, "forwarders", "comma-separated upstream DNS servers (host[:port], default port 53)")
	flag.BoolVar(&debug, "debug", false, "enable debug logs (show received queries)")
	flag.Parse()

	// Load optional app config file if present
	if cfgApp, err := loadAppConfig(configFileFlag.value); err == nil {
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
	}

	// CLI flags override config
	if forwardersFlag.set {
		forwarders = parseForwarders(forwardersFlag.value)
	}
	// debug can be enabled with the -debug flag

	if forwarders == nil {
		forwarders = []string{}
	}

	initZones(zonesDirFlag.value)
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
	log.Printf("Config initialized: zones_dir=%s forwarders=%v forward_timeout=%s loaded_zones=%d",
		zonesDirFlag.value, forwarders, forwardTimeout, len(zoneNames))
	if len(zoneNames) > 0 {
		log.Printf("Loaded zones: %v", zoneNames)
	}

	dns.HandleFunc(".", handleDNS)

	udpServer := &dns.Server{Addr: ":53", Net: "udp"}
	tcpServer := &dns.Server{Addr: ":53", Net: "tcp"}

	// Run servers in goroutines
	go func() {
		log.Printf("Starting UDP server on %s", udpServer.Addr)
		if err := udpServer.ListenAndServe(); err != nil {
			log.Fatalf("failed to start UDP server: %v", err)
		}
	}()

	go func() {
		log.Printf("Starting TCP server on %s", tcpServer.Addr)
		if err := tcpServer.ListenAndServe(); err != nil {
			log.Fatalf("failed to start TCP server: %v", err)
		}
	}()

	// Wait for signal to shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down servers...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = udpServer.ShutdownContext(ctx)
	_ = tcpServer.ShutdownContext(ctx)
	fmt.Println("Servers stopped")
}

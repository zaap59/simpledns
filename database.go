package main

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/miekg/dns"
)

// Database holds the SQLite connection
type Database struct {
	db *sql.DB
	mu sync.RWMutex
}

// DBZone represents a zone in the database
type DBZone struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	TTL     int    `json:"ttl"`
	NS      string `json:"ns"`
	Admin   string `json:"admin"`
	Serial  int    `json:"serial"`
	Refresh int    `json:"refresh"`
	Retry   int    `json:"retry"`
	Expire  int    `json:"expire"`
}

// DBRecord represents a DNS record in the database
type DBRecord struct {
	ID       int64  `json:"id"`
	ZoneID   int64  `json:"zone_id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
}

// DBForwarder represents a forwarder in the database
type DBForwarder struct {
	ID       int64  `json:"id"`
	Address  string `json:"address"`
	Priority int    `json:"priority"`
}

// DBConfig represents a config entry in the database
type DBConfig struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var database *Database

// InitDatabase initializes the SQLite database
func InitDatabase(dbPath string) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	database = &Database{db: db}

	// Create tables
	if err := database.createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Run migrations for existing databases
	if err := database.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// runMigrations applies database migrations for schema changes
func (d *Database) runMigrations() error {
	// Add priority column to records table if it doesn't exist
	_, err := d.db.Exec(`ALTER TABLE records ADD COLUMN priority INTEGER DEFAULT 0`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		// Ignore "duplicate column name" error as it means the column already exists
		return nil
	}
	return nil
}

// createTables creates the database schema
func (d *Database) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS zones (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		enabled INTEGER DEFAULT 1,
		ttl INTEGER DEFAULT 3600,
		ns TEXT DEFAULT 'ns1.local.',
		admin TEXT DEFAULT 'admin.local.',
		serial INTEGER DEFAULT 1,
		refresh INTEGER DEFAULT 3600,
		retry INTEGER DEFAULT 600,
		expire INTEGER DEFAULT 86400,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		zone_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		value TEXT NOT NULL,
		ttl INTEGER DEFAULT 3600,
		priority INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (zone_id) REFERENCES zones(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS forwarders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		address TEXT UNIQUE NOT NULL,
		priority INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS api_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		token_hash TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_records_zone_id ON records(zone_id);
	CREATE INDEX IF NOT EXISTS idx_records_name ON records(name);
	CREATE INDEX IF NOT EXISTS idx_api_tokens_hash ON api_tokens(token_hash);
	`

	_, err := d.db.Exec(schema)
	return err
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// Zone CRUD operations

// CreateZone creates a new zone
func (d *Database) CreateZone(zone *DBZone) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Ensure zone name does not have trailing dot
	zone.Name = strings.TrimSuffix(zone.Name, ".")

	result, err := d.db.Exec(`
		INSERT INTO zones (name, enabled, ttl, ns, admin, serial, refresh, retry, expire)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, zone.Name, zone.Enabled, zone.TTL, zone.NS, zone.Admin, zone.Serial, zone.Refresh, zone.Retry, zone.Expire)
	if err != nil {
		return err
	}

	zone.ID, _ = result.LastInsertId()
	return nil
}

// GetZone retrieves a zone by ID
func (d *Database) GetZone(id int64) (*DBZone, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	zone := &DBZone{}
	err := d.db.QueryRow(`
		SELECT id, name, enabled, ttl, ns, admin, serial, refresh, retry, expire
		FROM zones WHERE id = ?
	`, id).Scan(&zone.ID, &zone.Name, &zone.Enabled, &zone.TTL, &zone.NS, &zone.Admin,
		&zone.Serial, &zone.Refresh, &zone.Retry, &zone.Expire)
	if err != nil {
		return nil, err
	}
	return zone, nil
}

// GetZoneByName retrieves a zone by name
func (d *Database) GetZoneByName(name string) (*DBZone, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	name = strings.TrimSuffix(name, ".")
	zone := &DBZone{}
	err := d.db.QueryRow(`
		SELECT id, name, enabled, ttl, ns, admin, serial, refresh, retry, expire
		FROM zones WHERE name = ?
	`, name).Scan(&zone.ID, &zone.Name, &zone.Enabled, &zone.TTL, &zone.NS, &zone.Admin,
		&zone.Serial, &zone.Refresh, &zone.Retry, &zone.Expire)
	if err != nil {
		return nil, err
	}
	return zone, nil
}

// ListZones returns all zones
func (d *Database) ListZones() ([]DBZone, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, name, enabled, ttl, ns, admin, serial, refresh, retry, expire
		FROM zones ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var zones []DBZone
	for rows.Next() {
		var z DBZone
		if err := rows.Scan(&z.ID, &z.Name, &z.Enabled, &z.TTL, &z.NS, &z.Admin,
			&z.Serial, &z.Refresh, &z.Retry, &z.Expire); err != nil {
			return nil, err
		}
		zones = append(zones, z)
	}
	return zones, nil
}

// UpdateZone updates a zone
func (d *Database) UpdateZone(zone *DBZone) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	zone.Name = strings.TrimSuffix(zone.Name, ".")
	_, err := d.db.Exec(`
		UPDATE zones SET name = ?, enabled = ?, ttl = ?, ns = ?, admin = ?, 
		serial = serial + 1, refresh = ?, retry = ?, expire = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, zone.Name, zone.Enabled, zone.TTL, zone.NS, zone.Admin, zone.Refresh, zone.Retry, zone.Expire, zone.ID)
	return err
}

// DeleteZone deletes a zone and its records
func (d *Database) DeleteZone(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`DELETE FROM zones WHERE id = ?`, id)
	return err
}

// Record CRUD operations

// CreateRecord creates a new record
func (d *Database) CreateRecord(record *DBRecord) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	result, err := d.db.Exec(`
		INSERT INTO records (zone_id, name, type, value, ttl, priority)
		VALUES (?, ?, ?, ?, ?, ?)
	`, record.ZoneID, record.Name, strings.ToUpper(record.Type), record.Value, record.TTL, record.Priority)
	if err != nil {
		return err
	}

	record.ID, _ = result.LastInsertId()

	// Update zone serial
	_, _ = d.db.Exec(`UPDATE zones SET serial = serial + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, record.ZoneID)

	return nil
}

// GetRecord retrieves a record by ID
func (d *Database) GetRecord(id int64) (*DBRecord, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	record := &DBRecord{}
	err := d.db.QueryRow(`
		SELECT id, zone_id, name, type, value, ttl, priority
		FROM records WHERE id = ?
	`, id).Scan(&record.ID, &record.ZoneID, &record.Name, &record.Type, &record.Value, &record.TTL, &record.Priority)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// ListRecordsByZone returns all records for a zone
func (d *Database) ListRecordsByZone(zoneID int64) ([]DBRecord, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, zone_id, name, type, value, ttl, priority
		FROM records WHERE zone_id = ? ORDER BY type, name
	`, zoneID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var records []DBRecord
	for rows.Next() {
		var r DBRecord
		if err := rows.Scan(&r.ID, &r.ZoneID, &r.Name, &r.Type, &r.Value, &r.TTL, &r.Priority); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

// UpdateRecord updates a record
func (d *Database) UpdateRecord(record *DBRecord) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`
		UPDATE records SET name = ?, type = ?, value = ?, ttl = ?, priority = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, record.Name, strings.ToUpper(record.Type), record.Value, record.TTL, record.Priority, record.ID)
	if err != nil {
		return err
	}

	// Update zone serial
	_, _ = d.db.Exec(`UPDATE zones SET serial = serial + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, record.ZoneID)

	return err
}

// DeleteRecord deletes a record
func (d *Database) DeleteRecord(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Get zone_id first for serial update
	var zoneID int64
	_ = d.db.QueryRow(`SELECT zone_id FROM records WHERE id = ?`, id).Scan(&zoneID)

	_, err := d.db.Exec(`DELETE FROM records WHERE id = ?`, id)
	if err != nil {
		return err
	}

	// Update zone serial
	if zoneID > 0 {
		_, _ = d.db.Exec(`UPDATE zones SET serial = serial + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, zoneID)
	}

	return nil
}

// Forwarder CRUD operations

// CreateForwarder creates a new forwarder
func (d *Database) CreateForwarder(forwarder *DBForwarder) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Add default port if missing
	addr := forwarder.Address
	if !strings.Contains(addr, ":") {
		addr = addr + ":53"
	}

	result, err := d.db.Exec(`
		INSERT INTO forwarders (address, priority)
		VALUES (?, ?)
	`, addr, forwarder.Priority)
	if err != nil {
		return err
	}

	forwarder.ID, _ = result.LastInsertId()
	forwarder.Address = addr
	return nil
}

// ListForwarders returns all forwarders
func (d *Database) ListForwarders() ([]DBForwarder, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, address, priority
		FROM forwarders ORDER BY priority, id
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var forwarders []DBForwarder
	for rows.Next() {
		var f DBForwarder
		if err := rows.Scan(&f.ID, &f.Address, &f.Priority); err != nil {
			return nil, err
		}
		forwarders = append(forwarders, f)
	}
	return forwarders, nil
}

// DeleteForwarder deletes a forwarder by ID
func (d *Database) DeleteForwarder(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`DELETE FROM forwarders WHERE id = ?`, id)
	return err
}

// DeleteForwarderByAddress deletes a forwarder by address
func (d *Database) DeleteForwarderByAddress(address string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	result, err := d.db.Exec(`DELETE FROM forwarders WHERE address = ?`, address)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("forwarder not found")
	}
	return nil
}

// Config operations

// SetConfig sets a config value
func (d *Database) SetConfig(key, value string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`
		INSERT INTO config (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = CURRENT_TIMESTAMP
	`, key, value, value)
	return err
}

// GetConfig gets a config value
func (d *Database) GetConfig(key string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var value string
	err := d.db.QueryRow(`SELECT value FROM config WHERE key = ?`, key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

// LoadZonesFromDB loads zones from SQLite into memory for DNS resolution
func LoadZonesFromDB() error {
	if database == nil {
		return fmt.Errorf("database not initialized")
	}

	dbZones, err := database.ListZones()
	if err != nil {
		return err
	}

	// Reset zones
	zones = make(map[string][]dns.RR)
	loadedZoneNames = nil

	for _, dbZone := range dbZones {
		// Skip disabled zones
		if !dbZone.Enabled {
			continue
		}

		zoneName := dns.Fqdn(dbZone.Name)
		loadedZoneNames = append(loadedZoneNames, zoneName)

		// Create SOA record
		soaStr := fmt.Sprintf("%s %d IN SOA %s %s %d %d %d %d 3600",
			zoneName, dbZone.TTL,
			dns.Fqdn(dbZone.NS),
			strings.Replace(dbZone.Admin, "@", ".", 1),
			dbZone.Serial, dbZone.Refresh, dbZone.Retry, dbZone.Expire,
		)
		if soaRR, err := dns.NewRR(soaStr); err == nil {
			zones[zoneName] = append(zones[zoneName], soaRR)
		}

		// Create NS record
		nsStr := fmt.Sprintf("%s %d IN NS %s", zoneName, dbZone.TTL, dns.Fqdn(dbZone.NS))
		if nsRR, err := dns.NewRR(nsStr); err == nil {
			zones[zoneName] = append(zones[zoneName], nsRR)
		}

		// Load records for this zone
		records, err := database.ListRecordsByZone(dbZone.ID)
		if err != nil {
			continue
		}

		for _, record := range records {
			// Build record name
			recordName := record.Name
			if recordName == "@" {
				recordName = zoneName
			} else if !strings.HasSuffix(recordName, ".") {
				recordName = recordName + "." + zoneName
			}

			rrStr := fmt.Sprintf("%s %d IN %s %s", recordName, record.TTL, record.Type, record.Value)
			if rr, err := dns.NewRR(rrStr); err == nil {
				name := dns.Fqdn(rr.Header().Name)
				zones[name] = append(zones[name], rr)
			}
		}
	}

	return nil
}

// LoadForwardersFromDB loads forwarders from SQLite into memory
// If no forwarders are in the database, keeps existing forwarders (from config file)
func LoadForwardersFromDB() error {
	if database == nil {
		return fmt.Errorf("database not initialized")
	}

	dbForwarders, err := database.ListForwarders()
	if err != nil {
		return err
	}

	// Only override if there are forwarders in the database
	// Otherwise keep the ones from config file
	if len(dbForwarders) > 0 {
		forwarders = make([]string, 0, len(dbForwarders))
		for _, f := range dbForwarders {
			forwarders = append(forwarders, f.Address)
		}
	}

	return nil
}

// ReloadFromDB reloads zones and forwarders from database
func ReloadFromDB() error {
	if err := LoadZonesFromDB(); err != nil {
		return err
	}
	if err := LoadForwardersFromDB(); err != nil {
		return err
	}
	return nil
}

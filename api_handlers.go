package main

import (
	"net/http"
	"strconv"

	"log/slog"

	"github.com/gin-gonic/gin"
)

// API request/response types

type CreateZoneRequest struct {
	Name    string `json:"name" binding:"required"`
	Enabled *bool  `json:"enabled"`
	TTL     int    `json:"ttl"`
	NS      string `json:"ns"`
	Admin   string `json:"admin"`
	Refresh int    `json:"refresh"`
	Retry   int    `json:"retry"`
	Expire  int    `json:"expire"`
}

type CreateRecordRequest struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required"`
	Value    string `json:"value" binding:"required"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
}

type CreateForwarderRequest struct {
	Address  string `json:"address" binding:"required"`
	Priority int    `json:"priority"`
}

// Zone handlers

func handleAPICreateZone(c *gin.Context) {
	var req CreateZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	zone := &DBZone{
		Name:    req.Name,
		Enabled: true,
		TTL:     req.TTL,
		NS:      req.NS,
		Admin:   req.Admin,
		Serial:  1,
		Refresh: req.Refresh,
		Retry:   req.Retry,
		Expire:  req.Expire,
	}

	// Set defaults
	if req.Enabled != nil {
		zone.Enabled = *req.Enabled
	}
	if zone.TTL == 0 {
		zone.TTL = 3600
	}
	if zone.NS == "" {
		zone.NS = "ns1." + req.Name
	}
	if zone.Admin == "" {
		zone.Admin = "admin." + req.Name
	}
	if zone.Refresh == 0 {
		zone.Refresh = 3600
	}
	if zone.Retry == 0 {
		zone.Retry = 600
	}
	if zone.Expire == 0 {
		zone.Expire = 86400
	}

	if err := database.CreateZone(zone); err != nil {
		slog.Error("failed to create zone", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create zone"})
		return
	}

	// Reload zones into memory
	if err := LoadZonesFromDB(); err != nil {
		slog.Error("failed to reload zones", "error", err)
	}

	slog.Info("Zone created", "name", zone.Name, "id", zone.ID)
	c.JSON(http.StatusCreated, zone)
}

func handleAPIGetZone(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid zone id"})
		return
	}

	zone, err := database.GetZone(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "zone not found"})
		return
	}

	// Get records for this zone
	records, _ := database.ListRecordsByZone(id)

	c.JSON(http.StatusOK, gin.H{
		"zone":    zone,
		"records": records,
	})
}

func handleAPIListZones(c *gin.Context) {
	zones, err := database.ListZones()
	if err != nil {
		slog.Error("failed to list zones", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list zones"})
		return
	}

	// Include record count for each zone
	type ZoneWithCount struct {
		DBZone
		RecordCount int `json:"record_count"`
	}

	result := make([]ZoneWithCount, 0, len(zones))
	for _, z := range zones {
		records, _ := database.ListRecordsByZone(z.ID)
		result = append(result, ZoneWithCount{
			DBZone:      z,
			RecordCount: len(records),
		})
	}

	c.JSON(http.StatusOK, result)
}

func handleAPIUpdateZone(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid zone id"})
		return
	}

	var req CreateZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	zone := &DBZone{
		ID:      id,
		Name:    req.Name,
		Enabled: true,
		TTL:     req.TTL,
		NS:      req.NS,
		Admin:   req.Admin,
		Refresh: req.Refresh,
		Retry:   req.Retry,
		Expire:  req.Expire,
	}

	if req.Enabled != nil {
		zone.Enabled = *req.Enabled
	}

	if err := database.UpdateZone(zone); err != nil {
		slog.Error("failed to update zone", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update zone"})
		return
	}

	// Reload zones into memory
	if err := LoadZonesFromDB(); err != nil {
		slog.Error("failed to reload zones", "error", err)
	}

	slog.Info("Zone updated", "name", zone.Name, "id", zone.ID)
	c.JSON(http.StatusOK, zone)
}

func handleAPIToggleZone(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid zone id"})
		return
	}

	zone, err := database.GetZone(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "zone not found"})
		return
	}

	// Toggle the enabled status
	zone.Enabled = !zone.Enabled

	if err := database.UpdateZone(zone); err != nil {
		slog.Error("failed to toggle zone", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to toggle zone"})
		return
	}

	// Reload zones into memory
	if err := LoadZonesFromDB(); err != nil {
		slog.Error("failed to reload zones", "error", err)
	}

	slog.Info("Zone toggled", "name", zone.Name, "enabled", zone.Enabled)
	c.JSON(http.StatusOK, gin.H{"enabled": zone.Enabled})
}

func handleAPIDeleteZone(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid zone id"})
		return
	}

	zone, err := database.GetZone(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "zone not found"})
		return
	}

	if err := database.DeleteZone(id); err != nil {
		slog.Error("failed to delete zone", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete zone"})
		return
	}

	// Reload zones into memory
	if err := LoadZonesFromDB(); err != nil {
		slog.Error("failed to reload zones", "error", err)
	}

	slog.Info("Zone deleted", "name", zone.Name, "id", id)
	c.JSON(http.StatusOK, gin.H{"message": "zone deleted"})
}

// Record handlers

func handleAPICreateRecord(c *gin.Context) {
	zoneIDStr := c.Param("id")
	zoneID, err := strconv.ParseInt(zoneIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid zone id"})
		return
	}

	// Verify zone exists
	if _, err := database.GetZone(zoneID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "zone not found"})
		return
	}

	var req CreateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	record := &DBRecord{
		ZoneID:   zoneID,
		Name:     req.Name,
		Type:     req.Type,
		Value:    req.Value,
		TTL:      req.TTL,
		Priority: req.Priority,
	}

	if record.TTL == 0 {
		record.TTL = 3600
	}

	if err := database.CreateRecord(record); err != nil {
		slog.Error("failed to create record", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create record"})
		return
	}

	// Reload zones into memory
	if err := LoadZonesFromDB(); err != nil {
		slog.Error("failed to reload zones", "error", err)
	}

	slog.Info("Record created", "name", record.Name, "type", record.Type, "id", record.ID)
	c.JSON(http.StatusCreated, record)
}

func handleAPIListRecords(c *gin.Context) {
	zoneIDStr := c.Param("id")
	zoneID, err := strconv.ParseInt(zoneIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid zone id"})
		return
	}

	records, err := database.ListRecordsByZone(zoneID)
	if err != nil {
		slog.Error("failed to list records", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list records"})
		return
	}

	c.JSON(http.StatusOK, records)
}

func handleAPIUpdateRecord(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid record id"})
		return
	}

	existing, err := database.GetRecord(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	var req CreateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	record := &DBRecord{
		ID:       id,
		ZoneID:   existing.ZoneID,
		Name:     req.Name,
		Type:     req.Type,
		Value:    req.Value,
		TTL:      req.TTL,
		Priority: req.Priority,
	}

	if record.TTL == 0 {
		record.TTL = 3600
	}

	if err := database.UpdateRecord(record); err != nil {
		slog.Error("failed to update record", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update record"})
		return
	}

	// Reload zones into memory
	if err := LoadZonesFromDB(); err != nil {
		slog.Error("failed to reload zones", "error", err)
	}

	slog.Info("Record updated", "name", record.Name, "type", record.Type, "id", record.ID)
	c.JSON(http.StatusOK, record)
}

func handleAPIDeleteRecord(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid record id"})
		return
	}

	record, err := database.GetRecord(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	if err := database.DeleteRecord(id); err != nil {
		slog.Error("failed to delete record", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete record"})
		return
	}

	// Reload zones into memory
	if err := LoadZonesFromDB(); err != nil {
		slog.Error("failed to reload zones", "error", err)
	}

	slog.Info("Record deleted", "name", record.Name, "id", id)
	c.JSON(http.StatusOK, gin.H{"message": "record deleted"})
}

// handleAPIDeleteRecordInZone handles DELETE /api/zones/:id/records/:record_id
func handleAPIDeleteRecordInZone(c *gin.Context) {
	zoneIDStr := c.Param("id")
	zoneID, err := strconv.ParseInt(zoneIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid zone id"})
		return
	}

	recordIDStr := c.Param("record_id")
	recordID, err := strconv.ParseInt(recordIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid record id"})
		return
	}

	// Verify zone exists
	if _, err := database.GetZone(zoneID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "zone not found"})
		return
	}

	record, err := database.GetRecord(recordID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	// Verify record belongs to the zone
	if record.ZoneID != zoneID {
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found in this zone"})
		return
	}

	if err := database.DeleteRecord(recordID); err != nil {
		slog.Error("failed to delete record", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete record"})
		return
	}

	// Reload zones into memory
	if err := LoadZonesFromDB(); err != nil {
		slog.Error("failed to reload zones", "error", err)
	}

	slog.Info("Record deleted", "name", record.Name, "zone_id", zoneID, "record_id", recordID)
	c.JSON(http.StatusOK, gin.H{"message": "record deleted"})
}

// handleAPIUpdateRecordInZone handles PUT /api/zones/:id/records/:record_id
func handleAPIUpdateRecordInZone(c *gin.Context) {
	zoneIDStr := c.Param("id")
	zoneID, err := strconv.ParseInt(zoneIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid zone id"})
		return
	}

	recordIDStr := c.Param("record_id")
	recordID, err := strconv.ParseInt(recordIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid record id"})
		return
	}

	// Verify zone exists
	if _, err := database.GetZone(zoneID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "zone not found"})
		return
	}

	existing, err := database.GetRecord(recordID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	// Verify record belongs to the zone
	if existing.ZoneID != zoneID {
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found in this zone"})
		return
	}

	var req CreateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	record := &DBRecord{
		ID:       recordID,
		ZoneID:   zoneID,
		Name:     req.Name,
		Type:     req.Type,
		Value:    req.Value,
		TTL:      req.TTL,
		Priority: req.Priority,
	}

	if record.TTL == 0 {
		record.TTL = 3600
	}

	if err := database.UpdateRecord(record); err != nil {
		slog.Error("failed to update record", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update record"})
		return
	}

	// Reload zones into memory
	if err := LoadZonesFromDB(); err != nil {
		slog.Error("failed to reload zones", "error", err)
	}

	slog.Info("Record updated", "name", record.Name, "type", record.Type, "zone_id", zoneID, "record_id", recordID)
	c.JSON(http.StatusOK, record)
}

// handleAPIGetRecordInZone handles GET /api/zones/:id/records/:record_id
func handleAPIGetRecordInZone(c *gin.Context) {
	zoneIDStr := c.Param("id")
	zoneID, err := strconv.ParseInt(zoneIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid zone id"})
		return
	}

	recordIDStr := c.Param("record_id")
	recordID, err := strconv.ParseInt(recordIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid record id"})
		return
	}

	// Verify zone exists
	if _, err := database.GetZone(zoneID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "zone not found"})
		return
	}

	record, err := database.GetRecord(recordID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	// Verify record belongs to the zone
	if record.ZoneID != zoneID {
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found in this zone"})
		return
	}

	c.JSON(http.StatusOK, record)
}

// Forwarder handlers

func handleAPICreateForwarder(c *gin.Context) {
	var req CreateForwarderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	forwarder := &DBForwarder{
		Address:  req.Address,
		Priority: req.Priority,
	}

	if err := database.CreateForwarder(forwarder); err != nil {
		slog.Error("failed to create forwarder", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create forwarder"})
		return
	}

	// Reload forwarders into memory
	if err := LoadForwardersFromDB(); err != nil {
		slog.Error("failed to reload forwarders", "error", err)
	}

	slog.Info("Forwarder created", "address", forwarder.Address, "id", forwarder.ID)
	c.JSON(http.StatusCreated, forwarder)
}

func handleAPIListForwarders(c *gin.Context) {
	forwarders, err := database.ListForwarders()
	if err != nil {
		slog.Error("failed to list forwarders", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list forwarders"})
		return
	}

	c.JSON(http.StatusOK, forwarders)
}

func handleAPIDeleteForwarder(c *gin.Context) {
	// The parameter can be an ID or an address
	param := c.Param("id")

	// Try to parse as ID first
	if id, err := strconv.ParseInt(param, 10, 64); err == nil {
		if err := database.DeleteForwarder(id); err != nil {
			slog.Error("failed to delete forwarder", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete forwarder"})
			return
		}
		slog.Info("Forwarder deleted", "id", id)
	} else {
		// Treat as address
		if err := database.DeleteForwarderByAddress(param); err != nil {
			slog.Error("failed to delete forwarder", "error", err, "address", param)
			c.JSON(http.StatusNotFound, gin.H{"error": "forwarder not found"})
			return
		}
		slog.Info("Forwarder deleted", "address", param)
	}

	// Reload forwarders into memory
	if err := LoadForwardersFromDB(); err != nil {
		slog.Error("failed to reload forwarders", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "forwarder deleted"})
}

// registerAPIRoutes registers all CRUD API routes (only in sqlite mode)
func registerAPIRoutes(router *gin.Engine) {
	api := router.Group("/api")
	api.Use(APIAuthMiddleware())
	{
		// Zones CRUD
		api.POST("/zones", handleAPICreateZone)
		api.GET("/zones", handleAPIListZones)
		api.GET("/zones/:id", handleAPIGetZone)
		api.PUT("/zones/:id", handleAPIUpdateZone)
		api.PATCH("/zones/:id/toggle", handleAPIToggleZone)
		api.DELETE("/zones/:id", handleAPIDeleteZone)

		// Records CRUD (use :id consistently)
		api.POST("/zones/:id/records", handleAPICreateRecord)
		api.GET("/zones/:id/records", handleAPIListRecords)
		api.GET("/zones/:id/records/:record_id", handleAPIGetRecordInZone)
		api.PUT("/zones/:id/records/:record_id", handleAPIUpdateRecordInZone)
		api.DELETE("/zones/:id/records/:record_id", handleAPIDeleteRecordInZone)

		// Legacy record routes (for backward compatibility)
		api.PUT("/records/:id", handleAPIUpdateRecord)
		api.DELETE("/records/:id", handleAPIDeleteRecord)

		// Forwarders CRUD
		api.POST("/forwarders", handleAPICreateForwarder)
		api.GET("/forwarders", handleAPIListForwarders)
		api.DELETE("/forwarders/:id", handleAPIDeleteForwarder)
	}
}

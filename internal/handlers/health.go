package handlers

import (
	"context"
	"net/http"
	"time"
)

type HealthResponse struct {
	Status   string         `json:"status"`
	Database DatabaseHealth `json:"database"`
	Service  string         `json:"service"`
	Version  string         `json:"version"`
	Uptime   string         `json:"uptime"`
}

type DatabaseHealth struct {
	Status       string `json:"status"`
	ResponseTime string `json:"response_time"`
}

var startTime = time.Now()

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Check database connection
	start := time.Now()
	dbStatus := "healthy"

	if err := h.service.HealthCheck(ctx); err != nil {
		h.logger.Error("Database health check failed: %v", err)
		dbStatus = "unhealthy"
		respondError(w, http.StatusServiceUnavailable, "UNHEALTHY", "Database connection failed")
		return
	}

	responseTime := time.Since(start)

	uptime := time.Since(startTime)

	response := HealthResponse{
		Status: "healthy",
		Database: DatabaseHealth{
			Status:       dbStatus,
			ResponseTime: responseTime.String(),
		},
		Service: "pr-reviewer-service",
		Version: "1.0.0",
		Uptime:  uptime.String(),
	}

	respondJSON(w, http.StatusOK, response)
}

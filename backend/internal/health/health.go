package health

import (
	"net/http"
	"runtime"
	"time"

	"backend/config"
	"github.com/gin-gonic/gin"
)

var startTime time.Time

func init() {
	startTime = time.Now()
}

type MemoryStats struct {
	AllocMB      uint64 `json:"alloc_mb"`
	TotalAllocMB uint64 `json:"total_alloc_mb"`
	SysMB        uint64 `json:"sys_mb"`
	NumGC        uint32 `json:"num_gc"`
}

type DependencyCheck struct {
	Status  string `json:"status"`
	Details string `json:"details"`
}

type HealthResponse struct {
	Status        string                     `json:"status"`
	Environment   string                     `json:"environment"`
	UptimeSeconds float64                    `json:"uptime_seconds"`
	Timestamp     string                     `json:"timestamp"`
	Memory        MemoryStats                `json:"memory"`
	Dependencies  map[string]DependencyCheck `json:"dependencies"`
}

// HealthCheckHandler godoc
// @Summary      Get health status
// @Description  Returns the health status, uptime, system metrics and dependency checks of the API
// @Tags         Health
// @Accept       json
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Router       /health [get]
func HealthCheckHandler(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memStats := MemoryStats{
		AllocMB:      m.Alloc / 1024 / 1024,
		TotalAllocMB: m.TotalAlloc / 1024 / 1024,
		SysMB:        m.Sys / 1024 / 1024,
		NumGC:        m.NumGC,
	}

	oauthStatus := "UP"
	oauthDetails := "OAuth mode: " + config.ActiveConfig.OAuthMode
	if config.ActiveConfig.OAuthMode == "real" && config.ActiveConfig.GoogleClientID == "DEV_CLIENT_ID.apps.googleusercontent.com" {
		oauthStatus = "DEGRADED"
		oauthDetails = "Real OAuth configured with placeholder credentials"
	}

	response := HealthResponse{
		Status:        "UP",
		Environment:   config.ActiveConfig.EnvName,
		UptimeSeconds: time.Since(startTime).Seconds(),
		Timestamp:     time.Now().Format(time.RFC3339),
		Memory:        memStats,
		Dependencies: map[string]DependencyCheck{
			"google_oauth": {
				Status:  oauthStatus,
				Details: oauthDetails,
			},
		},
	}

	c.JSON(http.StatusOK, response)
}

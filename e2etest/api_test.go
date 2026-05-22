package e2etest

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

func getBackendURL() string {
	url := os.Getenv("BACKEND_URL")
	if url == "" {
		url = "http://localhost:8080"
	}
	return url
}

func TestAPIHealthEndpoint(t *testing.T) {
	backendURL := getBackendURL()
	
	// Create client with timeout
	client := &http.Client{Timeout: 5 * time.Second}
	
	// Query /health endpoint
	resp, err := client.Get(backendURL + "/health")
	if err != nil {
		t.Fatalf("Failed to call health check: %v. Is backend running?", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	// Parse JSON
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode health check response: %v", err)
	}

	// Assert structures
	status, ok := body["status"].(string)
	if !ok || status != "UP" {
		t.Errorf("Expected status 'UP', got %v", body["status"])
	}

	env, ok := body["environment"].(string)
	if !ok || env == "" {
		t.Errorf("Expected non-empty environment string, got %v", body["environment"])
	}

	memory, ok := body["memory"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected memory stats block, got none")
	}

	if _, ok := memory["alloc_mb"].(float64); !ok {
		t.Errorf("Expected alloc_mb metric, got %v", memory["alloc_mb"])
	}

	deps, ok := body["dependencies"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected dependencies block, got none")
	}

	oauth, ok := deps["google_oauth"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected google_oauth dependency block, got none")
	}

	if _, ok := oauth["status"].(string); !ok {
		t.Errorf("Expected oauth status check, got %v", oauth["status"])
	}
}

func TestAPISwaggerEndpoint(t *testing.T) {
	backendURL := getBackendURL()
	client := &http.Client{Timeout: 5 * time.Second}

	// Query Swagger HTML
	resp, err := client.Get(backendURL + "/swagger/index.html")
	if err != nil {
		t.Fatalf("Failed to access swagger HTML: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected swagger HTML response status 200 OK, got %d", resp.StatusCode)
	}

	// Query Swagger JSON structure
	respJSON, err := client.Get(backendURL + "/swagger/doc.json")
	if err != nil {
		t.Fatalf("Failed to access swagger JSON docs: %v", err)
	}
	defer respJSON.Body.Close()

	if respJSON.StatusCode != http.StatusOK {
		t.Errorf("Expected swagger doc.json response status 200 OK, got %d", respJSON.StatusCode)
	}

	var swagDoc map[string]interface{}
	if err := json.NewDecoder(respJSON.Body).Decode(&swagDoc); err != nil {
		t.Fatalf("Failed to decode swagger doc.json: %v", err)
	}

	// Check fields
	if version, ok := swagDoc["swagger"].(string); !ok || version != "2.0" {
		t.Errorf("Expected swagger version '2.0', got %v", swagDoc["swagger"])
	}

	info, ok := swagDoc["info"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected swagger info block, got none")
	}

	if _, ok := info["title"].(string); !ok {
		t.Errorf("Expected dynamic swagger title string in info, got %v", info["title"])
	}
}

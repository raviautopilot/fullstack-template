package e2etest

import (
	"encoding/json"
	"os"
)

type Config struct {
	FrontendURL      string `json:"frontend_url"`
	BackendURL       string `json:"backend_url"`
	ChromeDriverPath string `json:"chromedriver_path"`
	ChromiumPath     string `json:"chromium_path"`
	ChromeDriverPort int    `json:"chromedriver_port"`
	Headless         bool   `json:"headless"`
	EnableEvidence   bool   `json:"enable_evidence"`
	EvidenceDir      string `json:"evidence_dir"`
	ReportPath       string `json:"report_path"`
}

var DefaultConfig = Config{
	FrontendURL:      "http://localhost:5173",
	BackendURL:       "http://localhost:8080",
	ChromeDriverPath: "/usr/bin/chromedriver",
	ChromiumPath:     "/usr/bin/chromium",
	ChromeDriverPort: 8082,
	Headless:         true,
	EnableEvidence:   true,
	EvidenceDir:      "report_evidences",
	ReportPath:       "report.html",
}

// LoadConfig parses a config JSON file, generating a default sample if it is missing
func LoadConfig(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Generate default sample config
		data, err := json.MarshalIndent(DefaultConfig, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Environment variable overrides (highest precedence for CI/CD flexibility)
	if envFrontend := os.Getenv("FRONTEND_URL"); envFrontend != "" {
		config.FrontendURL = envFrontend
	}
	if envBackend := os.Getenv("BACKEND_URL"); envBackend != "" {
		config.BackendURL = envBackend
	}
	if envDriver := os.Getenv("CHROMEDRIVER_PATH"); envDriver != "" {
		config.ChromeDriverPath = envDriver
	}
	if envChromium := os.Getenv("CHROMIUM_PATH"); envChromium != "" {
		config.ChromiumPath = envChromium
	}

	return &config, nil
}

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	EnvName            string `json:"env_name"`
	Port               string `json:"port"`
	OAuthMode          string `json:"oauth_mode"` // "mock" or "real"
	GoogleClientID     string `json:"google_client_id"`
	GoogleClientSecret string `json:"google_client_secret"`
	OAuthRedirectURL   string `json:"oauth_redirect_url"`
	JWTSecret          string `json:"jwt_secret"`
	FrontendURL        string `json:"frontend_url"`
}

var ActiveConfig *Config

func Load() (*Config, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}

	configFilename := fmt.Sprintf("config.%s.json", env)
	
	// We'll search in a few common paths to ensure ease of execution
	searchPaths := []string{
		filepath.Join(".", "config", configFilename),
		filepath.Join(".", configFilename),
		filepath.Join("backend", "config", configFilename),
		filepath.Join("..", "config", configFilename),
	}

	var data []byte
	var err error
	var foundPath string

	for _, path := range searchPaths {
		data, err = os.ReadFile(path)
		if err == nil {
			foundPath = path
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("could not find configuration file config.%s.json in searched paths: %v", env, searchPaths)
	}

	fmt.Printf("Loading config '%s' from path: %s\n", env, foundPath)

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing configuration file: %w", err)
	}

	// Environment variable overrides (useful for Docker/CI/CD overrides)
	if portOverride := os.Getenv("PORT"); portOverride != "" {
		cfg.Port = portOverride
	}
	if oauthModeOverride := os.Getenv("OAUTH_MODE"); oauthModeOverride != "" {
		cfg.OAuthMode = oauthModeOverride
	}
	if clientIDOverride := os.Getenv("GOOGLE_CLIENT_ID"); clientIDOverride != "" {
		cfg.GoogleClientID = clientIDOverride
	}
	if clientSecretOverride := os.Getenv("GOOGLE_CLIENT_SECRET"); clientSecretOverride != "" {
		cfg.GoogleClientSecret = clientSecretOverride
	}
	if redirectURLOverride := os.Getenv("OAUTH_REDIRECT_URL"); redirectURLOverride != "" {
		cfg.OAuthRedirectURL = redirectURLOverride
	}

	ActiveConfig = &cfg
	return &cfg, nil
}

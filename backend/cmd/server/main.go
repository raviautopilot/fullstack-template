package main

import (
	"fmt"
	"log"

	"backend/config"
	"backend/internal/auth"
	"backend/internal/router"
)

// @title           Antigravity Monorepo API
// @version         1.0.0
// @description     A beautiful modern Web API backend built with Gin, Swagger, and Google OAuth.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /
func main() {
	fmt.Println("==================================================")
	fmt.Println("      Starting Antigravity Go Gin Backend         ")
	fmt.Println("==================================================")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to initialize configuration: %v", err)
	}

	// Initialize OAuth
	auth.InitOauth()

	// Setup Router
	r := router.SetupRouter()

	address := fmt.Sprintf(":%s", cfg.Port)
	fmt.Printf("Backend starting in environment '%s'\n", cfg.EnvName)
	fmt.Printf("Dynamic Swagger docs available at http://localhost%s/swagger/index.html\n", address)
	fmt.Printf("Listening and serving HTTP on %s\n", address)
	fmt.Println("==================================================")

	if err := r.Run(address); err != nil {
		log.Fatalf("Failed to run HTTP server: %v", err)
	}
}

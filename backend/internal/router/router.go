package router

import (
	"fmt"
	"net/http"

	"backend/config"
	"backend/docs" // Will be populated by swag init or placeholder
	"backend/internal/auth"
	"backend/internal/health"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func CORSMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		frontendURL := "http://localhost:5173"
		if config.ActiveConfig != nil && config.ActiveConfig.FrontendURL != "" {
			frontendURL = config.ActiveConfig.FrontendURL
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", frontendURL)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})
}

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(CORSMiddleware())

	// Configure Swagger Info dynamically based on active environment
	if config.ActiveConfig != nil {
		docs.SwaggerInfo.Title = fmt.Sprintf("Antigravity Monorepo API - [%s]", config.ActiveConfig.EnvName)
		docs.SwaggerInfo.Description = fmt.Sprintf("Antigravity Gin Web API running in %s mode.", config.ActiveConfig.EnvName)
		docs.SwaggerInfo.Version = "1.0.0"
		docs.SwaggerInfo.BasePath = "/"
		
		// Configure dynamic Host and Schemes per environment
		switch config.ActiveConfig.EnvName {
		case "local":
			docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%s", config.ActiveConfig.Port)
			docs.SwaggerInfo.Schemes = []string{"http"}
		case "tst":
			docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%s", config.ActiveConfig.Port)
			docs.SwaggerInfo.Schemes = []string{"http"}
		case "dev":
			docs.SwaggerInfo.Host = "api.dev.myapp.com"
			docs.SwaggerInfo.Schemes = []string{"https"}
		case "prd":
			docs.SwaggerInfo.Host = "api.myapp.com"
			docs.SwaggerInfo.Schemes = []string{"https"}
		default:
			docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%s", config.ActiveConfig.Port)
			docs.SwaggerInfo.Schemes = []string{"http"}
		}
	}

	// Health Check
	r.GET("/health", health.HealthCheckHandler)

	// Auth Endpoints
	r.GET("/auth/login", auth.LoginHandler)
	r.GET("/auth/mock-consent", auth.MockConsentHandler)
	r.GET("/auth/callback", auth.CallbackHandler)
	r.GET("/auth/logout", auth.LogoutHandler)

	// Protected User API
	r.GET("/api/user", auth.UserHandler)

	// Swagger Endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}

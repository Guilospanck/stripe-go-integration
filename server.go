package main

import (
	"fmt"

	"github.com/Guilospanck/stripe-go-integration/handlers"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Could not load environment variables")
		return
	}

	// Echo instance
	e := echo.New()

	// Gets the IP from caller when not using proxy
	e.IPExtractor = echo.ExtractIPDirect()
	// Gets the IP from caller when using X-Forwarded-For in the proxy (nginx, for example)
	// e.IPExtractor = echo.ExtractIPFromXFFHeader()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/ping", handlers.PingHandler)
	e.POST("/webhook", handlers.WebhookHandler)

	// Start server
	e.Logger.Fatal(e.Start(":4444"))
}

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stripe/stripe-go"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Could not load environment variables")
		return
	}

	// get stripe key
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", hello)
	e.POST("/webhook", webhook)

	// Start server
	e.Logger.Fatal(e.Start(":4444"))
}

// Handlers
func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func webhook(c echo.Context) error {
	req := c.Request()
	res := c.Response()

	const MaxBodyBytes = int64(65536)
	req.Body = http.MaxBytesReader(res.Writer, req.Body, MaxBodyBytes)

	body, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
		res.WriteHeader(http.StatusServiceUnavailable)
		return err
	}

	fmt.Fprintf(os.Stdout, "Got body: %s\n", body)

	res.WriteHeader(http.StatusOK)
	return c.String(http.StatusOK, string(body[:]))
}

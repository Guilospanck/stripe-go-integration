package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
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
	e.GET("/ping", ping)
	e.POST("/webhook", webhookHandler)

	// Start server
	e.Logger.Fatal(e.Start(":4444"))
}

// Handlers
func ping(c echo.Context) error {
	return c.String(http.StatusOK, "Pong!")
}

func webhookHandler(c echo.Context) error {
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

	// Checks webhook signature.
	// This makes sure that the POST request is actually coming from Stripe.
	stripeWebhookSecret := os.Getenv("STRIPE_WEBHOOK_KEY")
	signatureHeader := req.Header.Get("Stripe-Signature")

	event, err := webhook.ConstructEvent(body, signatureHeader, stripeWebhookSecret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error verifying webhook signature: %v\n", err)
		res.WriteHeader(http.StatusBadRequest)
		return err
	}

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			res.WriteHeader(http.StatusBadRequest)
			return err
		}

		orderPaid := session.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid
		if orderPaid {
			// create user account
			createCustomerAccount(session.CustomerDetails.Name, session.CustomerDetails.Email)

			// send email
			sendCustomerEmail(session.CustomerDetails.Email)
		}
	}

	res.WriteHeader(http.StatusOK)
	return nil
}

func createCustomerAccount(name, email string) {
	// save customer data to database
	fmt.Println("Customer account created!")
}

func sendCustomerEmail(email string) {
	// send email to customer with his temporary credentials
	fmt.Println("Email sent to customer!")
}

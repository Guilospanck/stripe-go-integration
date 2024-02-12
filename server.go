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
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/webhook"
)

type User struct {
	Name                string `json:"name"`
	Email               string `json:"email"`
	Password            string `json:"password"`
	SubscriptionStatus  string `json:"subscriptionStatus"`
	ExpireDateTimestamp int64  `json:"expireDateTimestamp"`
}

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
	case "invoice.paid":
		var invoice stripe.Invoice
		err := json.Unmarshal(event.Data.Raw, &invoice)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			res.WriteHeader(http.StatusBadRequest)
			return err
		}

		customerEmail := invoice.CustomerEmail
		customerName := invoice.CustomerName

		subs, err := subscription.Get(invoice.Subscription.ID, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting subscription from subscription ID: %v\n", err)
			res.WriteHeader(http.StatusBadRequest)
			return err
		}

		subscriptionStatus := subs.Status // Possible values are `incomplete`, `incomplete_expired`, `trialing`, `active`, `past_due`, `canceled`, or `unpaid`.
		expireDateTimestamp := subs.CurrentPeriodEnd * 1000

		// checks if customer exists
		user, userExists := customerAlreadyInTheDB(customerEmail)
		if !userExists {
			// create user account
			user := createUserAccount(customerName, customerEmail, subscriptionStatus, expireDateTimestamp)
			// send email with his credentials
			sendUserEmail(user.Email, user.Password)
		} else {
			// update user subscription status and expire date
			updateUserAccount(*user, subscriptionStatus, expireDateTimestamp)
		}

	case "customer.subscription.deleted":
		var customerSubscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &customerSubscription)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			res.WriteHeader(http.StatusBadRequest)
			return err
		}

		// status := customerSubscription.Status
		// userEmail := customerSubscription.Customer.Email

		// TODO: update status of user with email `userEmail` to `status`
	}

	res.WriteHeader(http.StatusOK)
	return nil

}

func updateUserAccount(user User, subscriptionStatus stripe.SubscriptionStatus, expireDateTimestamp int64) {
	user.SubscriptionStatus = string(subscriptionStatus)
	user.ExpireDateTimestamp = expireDateTimestamp

	fmt.Println()
	fmt.Printf("Updated user %+v account!", user)
	fmt.Println()
}

func customerAlreadyInTheDB(customerEmail string) (*User, bool) {
	// checks if the customer is already an user
	return nil, false
}

func createUserAccount(name string, email string, subscriptionStatus stripe.SubscriptionStatus, expireDateTimestamp int64) User {
	password := _generateUserTemporaryPassword()

	user := User{Email: email, Name: name, Password: password, SubscriptionStatus: string(subscriptionStatus), ExpireDateTimestamp: expireDateTimestamp}

	// save customer data to database
	fmt.Println()
	fmt.Printf("User %+v account created!", user)
	fmt.Println()

	return user
}

func sendUserEmail(email, password string) {
	// send email to user with his temporary credentials
	fmt.Println()
	fmt.Printf("Email sent to %s with his new credentials: %s!", email, password)
	fmt.Println()
}

func _generateUserTemporaryPassword() string {
	// generates temporary password
	password := "apple-potato-mirror"

	fmt.Println()
	fmt.Printf("Generated %s\n", password)
	fmt.Println()

	return password
}

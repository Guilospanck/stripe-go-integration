package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"

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

// From https://docs.stripe.com/ips#webhook-notifications
var allowedStripeIPs = [...]string{
	"3.18.12.63",
	"3.130.192.231",
	"13.235.14.237",
	"13.235.122.149",
	"18.211.135.69",
	"35.154.171.200",
	"52.15.183.38",
	"54.88.130.119",
	"54.88.130.237",
	"54.187.174.169",
	"54.187.205.235",
	"54.187.216.72",
	"::1",
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

	// Gets the IP from caller when not using proxy
	e.IPExtractor = echo.ExtractIPDirect()
	// Gets the IP from caller when using X-Forwarded-For in the proxy (nginx, for example)
	// e.IPExtractor = echo.ExtractIPFromXFFHeader()

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
	ipFromStripeWebhook := c.RealIP()

	// Checks webhook coming from allowed IP
	if !slices.Contains[[]string](allowedStripeIPs[:], ipFromStripeWebhook) {
		fmt.Fprintln(os.Stderr, "You shall not pass")
		err := echo.ErrBadRequest
		err.Message = "IP not allowed"
		return err
	}

	// Checks webhook signature.
	// This makes sure that the POST request is actually coming from Stripe.
	stripeWebhookSecret := os.Getenv("STRIPE_WEBHOOK_KEY")
	signatureHeader := req.Header.Get("Stripe-Signature")

	const MaxBodyBytes = int64(65536)
	req.Body = http.MaxBytesReader(res.Writer, req.Body, MaxBodyBytes)

	body, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
		res.WriteHeader(http.StatusServiceUnavailable)
		return err
	}

	event, err := webhook.ConstructEvent(body, signatureHeader, stripeWebhookSecret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error verifying webhook signature: %v\n", err)
		res.WriteHeader(http.StatusBadRequest)
		return err
	}

	// Use a goroutine so we can acknowledge events immediately
	// See: https://docs.stripe.com/webhooks#acknowledge-events-immediately
	go checkEventTypes(res, event)

	res.WriteHeader(http.StatusOK)
	return nil

}

func checkEventTypes(res *echo.Response, event stripe.Event) error {
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
			user.SubscriptionStatus = string(subscriptionStatus)
			user.ExpireDateTimestamp = expireDateTimestamp
			updateUserAccount(*user)
		}

	case "customer.subscription.deleted":
		/*
			From https://stripe.com/docs/billing/subscriptions/cancel#events
				Stripe sends a customer.subscription.deleted event when a customer’s subscription is canceled immediately.
				If the customer.subscription.deleted event’s request property isn’t null, that indicates the cancellation
				was made by your request (as opposed to automatically based upon your subscription settings).

				If you instead cancel a subscription at the end of the billing period (that is, by setting cancel_at_period_end to true),
				a customer.subscription.updated event is immediately triggered.
				That event reflects the change in the subscription’s cancel_at_period_end value.
				When the subscription is actually canceled at the end of the period, a customer.subscription.deleted event then occurs.
		*/
		var customerSubscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &customerSubscription)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			res.WriteHeader(http.StatusBadRequest)
			return err
		}

		userEmail := customerSubscription.Customer.Email
		status := customerSubscription.Status

		// cancellationFeedback := customerSubscription.CancellationDetails.Feedback // possible values: "customer_service",	"low_quality",	"missing_features",	"other",	"switched_service",	"too_complex",	"too_expensive",	"unused"
		// TODO: do something with the cancellationFeedback

		user, err := getUserFromDB(userEmail)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			res.WriteHeader(http.StatusBadRequest)
			return err
		}

		user.SubscriptionStatus = string(status)

		updateUserAccount(user)

	default:
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
	}

	return nil
}

func updateUserAccount(user User) {
	fmt.Println()
	fmt.Printf("Updated user %+v account!", user)
	fmt.Println()
}

func getUserFromDB(email string) (User, error) {
	user := User{
		Email:               email,
		Name:                "User from DB",
		ExpireDateTimestamp: 1707820250079,
		Password:            "",
		SubscriptionStatus:  "Active",
	}

	return user, nil
}

func customerAlreadyInTheDB(customerEmail string) (*User, bool) {
	// getUserFromDB(customerEmail)

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

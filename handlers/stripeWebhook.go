package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"

	"github.com/Guilospanck/stripe-go-integration/application"
	"github.com/Guilospanck/stripe-go-integration/utils"
	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v76/webhook"
)

func WebhookHandler(c echo.Context) error {
	req := c.Request()
	res := c.Response()
	ipFromStripeWebhook := c.RealIP()

	// Checks webhook coming from allowed IP
	if !slices.Contains[[]string](utils.AllowedStripeIPs[:], ipFromStripeWebhook) {
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
	go application.CheckEventTypes(res, event)

	res.WriteHeader(http.StatusOK)
	return nil
}

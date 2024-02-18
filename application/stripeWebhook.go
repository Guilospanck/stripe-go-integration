package application

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/Guilospanck/stripe-go-integration/repository"
	"github.com/Guilospanck/stripe-go-integration/utils"
	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/subscription"
)

func CheckEventTypes(res *echo.Response, event stripe.Event) error {
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
			fmt.Fprintf(os.Stderr, "Error getting subscription from subscription ID [%s]: %v\n", invoice.Subscription.ID, err)
			res.WriteHeader(http.StatusBadRequest)
			return err
		}

		subscriptionStatus := subs.Status // Possible values are `incomplete`, `incomplete_expired`, `trialing`, `active`, `past_due`, `canceled`, or `unpaid`.
		expireDateTimestamp := subs.CurrentPeriodEnd * 1000
		// Add 12h as a security. Sometimes the invoice takes some time to be processed even when there's nothing wrong with the payment methods.
		expireDateTimestamp += utils.TwelveHoursInMilliseconds

		// checks if customer exists
		user, userExists := repository.CustomerAlreadyInTheDB(customerEmail)
		if !userExists {
			// create user account
			user := repository.CreateUserAccount(customerName, customerEmail, subscriptionStatus, expireDateTimestamp)
			// send email with his credentials
			repository.SendUserEmail(user.Email, user.Password)
		} else {
			// update user subscription status and expire date
			user.SubscriptionStatus = string(subscriptionStatus)
			user.ExpireDateTimestamp = expireDateTimestamp
			repository.UpdateUserAccount(*user)
		}

	case "customer.subscription.updated":
		var customerSubscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &customerSubscription)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			res.WriteHeader(http.StatusBadRequest)
			return err
		}

		userEmail := customerSubscription.Customer.Email
		status := customerSubscription.Status

		user, err := repository.GetUserFromDB(userEmail)
		if err != nil {
			fmt.Fprintf(os.Stderr, "User [%s] does not exist or err: %v\n", userEmail, err)
			res.WriteHeader(http.StatusBadRequest)
			return err
		}

		user.SubscriptionStatus = string(status)

		repository.UpdateUserAccount(user)

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

		user, err := repository.GetUserFromDB(userEmail)
		if err != nil {
			fmt.Fprintf(os.Stderr, "User [%s] does not exist or err: %v\n", userEmail, err)
			res.WriteHeader(http.StatusBadRequest)
			return err
		}

		user.SubscriptionStatus = string(status)

		repository.UpdateUserAccount(user)

	default:
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
	}

	return nil
}

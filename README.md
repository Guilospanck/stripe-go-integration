# stripe-go-integration

## Stripe CLI

[Install Stripe CLI](https://stripe.com/docs/stripe-cli#install):

```sh
brew install stripe/stripe-cli/stripe
```

Then, run:

```sh
stripe login
```

This will redirect you to the Stripe dashboard in order to allow the access of the CLI to your application.

## How to run (dev/test mode)

Follow these points:

1) On a terminal tab, run the server:

```sh
go run .
```

> Server will be running on port :4444

2) On another tab, let Stripe CLI listen to events and do the redirection to your webhook endpoint:

```sh
stripe listen --forward-to localhost:4444/webhook
```

3) Finally, in another one, trigger an event also using Stripe CLI:

```sh
stripe trigger [event_type]
# > stripe trigger customer.created
```

> [Here](https://stripe.com/docs/api/events/types) you can find a list of all Stripe event types.

3.1) You can also trigger an event by going to your checkout in a human way, i.e., clicking on the checkout link of your checkout page and adding some test data. Testing data example:

```md
Fill out your payment form with test data
- Enter `4242 4242 4242 4242` as the card number
- Enter any future date for card expiry
- Enter any 3-digit number for CVV
- Enter any billing postal code (`90210`)
```

More testing data can be found [here](https://stripe.com/docs/testing).

## Misc

### API Keys

The Stripe API keys (public and private) can be found in [here](https://dashboard.stripe.com/test/apikeys), for the test mode, or in [here](https://dashboard.stripe.com/apikeys) for the production mode.

### Webhook key

The webhook key can be found, in test mode, when you run the `stripe listen` command.
For the production mode, you will need to get it at [Developers - Webhook](https://dashboard.stripe.com/test/webhooks).

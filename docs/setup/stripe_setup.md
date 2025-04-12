# Stripe Setup

Stripe is used for payment processing in Vrooli applications. This guide will help you set up Stripe integration for both development and production environments.

## Prerequisites

Before you begin, you'll need:
- A Stripe account (sign up at [stripe.com](https://stripe.com))
- Your application running locally (see [Repo Setup](repo_setup.md))

## Development Setup

### 1. Create a Stripe Account

1. Go to [stripe.com](https://stripe.com) and sign up for an account
2. Make sure you're in "Test Mode" (toggle in the dashboard) while developing

### 2. Get Your API Keys

1. In the Stripe Dashboard, go to "Developers" → "API keys"
2. Note down your "Publishable key" and "Secret key"

### 3. Configure Environment Variables

Add the following to your `.env` file:

```
# Stripe (Test Mode)
STRIPE_SECRET_KEY="sk_test_..."
STRIPE_PUBLISHABLE_KEY="pk_test_..."
STRIPE_WEBHOOK_SECRET=""  # We'll set this up later
```

### 4. Set Up Webhooks for Local Development

To receive Stripe events locally:

1. Install the Stripe CLI from [stripe.com/docs/stripe-cli](https://stripe.com/docs/stripe-cli)
2. Log in to your Stripe account via the CLI:
   ```bash
   stripe login
   ```
3. Start the webhook forwarding:
   ```bash
   stripe listen --forward-to http://localhost:5329/api/payments/webhook
   ```
4. Note the webhook signing secret displayed in your terminal
5. Update your `.env` file with the webhook secret:
   ```
   STRIPE_WEBHOOK_SECRET="whsec_..."
   ```

## Production Setup

### 1. Switch to Live Mode

1. Complete your Stripe account verification
2. Switch to "Live Mode" in the Stripe Dashboard
3. Get your live API keys from "Developers" → "API keys"

### 2. Configure Production Environment Variables

Update your production environment (`.env-prod` or equivalent) with:

```
# Stripe (Live Mode)
STRIPE_SECRET_KEY="sk_live_..."
STRIPE_PUBLISHABLE_KEY="pk_live_..."
STRIPE_WEBHOOK_SECRET="whsec_..."  # We'll set this up next
```

### 3. Set Up Production Webhooks

1. In the Stripe Dashboard, go to "Developers" → "Webhooks"
2. Click "Add Endpoint"
3. Set the endpoint URL to your production API endpoint:
   ```
   https://yourdomain.com/api/payments/webhook
   ```
4. Select events to send (typically: `payment_intent.succeeded`, `payment_intent.payment_failed`, `checkout.session.completed`)
5. After creating the webhook, reveal and copy the signing secret
6. Update your production environment with the webhook secret

## Testing Your Integration

### Test Payments in Development

Use Stripe's test cards to verify your integration:

- For successful payments: `4242 4242 4242 4242`
- For failed payments: `4000 0000 0000 0002`
- Use any future expiration date, any 3-digit CVC, and any 5-digit ZIP code

### Verify Webhook Reception

After setting up webhooks:

1. Trigger a test event from the Stripe Dashboard or by making a test payment
2. Check your application logs to verify the webhook was received and processed
3. Verify the appropriate actions were taken in your application (e.g., subscription activated)

## Additional Configuration

### Setting Up Products and Prices

For subscription-based services:

1. Go to "Products" in the Stripe Dashboard
2. Create products and prices matching your application's offerings
3. Note down the price IDs for use in your application

### Customizing the Checkout Experience

For a branded checkout experience:

1. Go to "Settings" → "Branding" in the Stripe Dashboard
2. Customize colors, logo, and icon
3. Set your business information and customer support details

For more information, refer to the [official Stripe documentation](https://stripe.com/docs).
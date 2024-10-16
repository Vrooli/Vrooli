# Payments With Stripe
We use Stripe to process payments. This includes:  
- Subscriptions
- Donations
- API credits

In this guide, we'll walk through how to set up Stripe for use in this application.

**NOTE:** Stripe has a [test environment and a production environment](https://stripe.com/docs/keys#test-live-modes). You'll need to set up both environments if you want to test and deploy the application. You can switch between these in the Stripe dashboard by toggling the *Test mode* switch in the top right corner.

## Create a Stripe Account
1. Go to [Stripe's website](https://stripe.com/) and click "Start Now" to create an account.
2. Enter your email, full name, password, and click "Create Account."
3. Complete the onboarding process by providing the required business information.

## Update Branding
Stripe provides its own pages for checking out, managing payments, and more. These pages should be branded with the company's logo and colors. To do this:
1. Open [Stripe's branding page](https://dashboard.stripe.com/settings/branding).
2. Set the icon, logo, colors, etc. 
3. Look at all of the preview variations to make sure everything looks good.

## Create Products
In both your test and production environment, you need to create Products and Prices. Currently, we offer one subscription, which is a product with multiple prices (monthly and yearly). We also offer API credits, which is a product with a single price, and donations, which is also product with a single price.

Here is how to create a product:  
1. In the Stripe dashboard, navigate to "Products" in the sidebar.
2. Click "Add product," fill out the details for your product (including prices), and save.
3. After creating the product, you can add new prices by clicking the product, clicking "Add another price," entering the desired price, and saving. 
4. Repeat these steps for all necessary products and prices.

**NOTE:** You cannot change the price of a Price object after it has been created. If you need to change the price, you'll need to create a new Price object, migrate any subscriptions to the new price, and archive the old one.

## Enable Customer Portal
The Stripe Customer Portal allows customers to manage their own subscriptions and payment methods. To access the portal, we must send the customer a link to the portal through their email. This link is generated by Stripe and is unique to each customer.

To enable the Customer Portal (do this for both test and production environments):
1. In the Stripe dashboard, navigate to "Settings."
2. Scroll down and click on "Customer portal."
3. Make sure that the following options are enabled: 
    - Invoice history
    - Customer information, minus shipping address and tax ID
    - Payment methods
    - Cancellations, with "end of billing period" mode and all cancellation reasons enabled
    - Subscriptions, with the product you created for the selected plan
4. Copy the portal link and click "Save."
5. Save the portal link in TODO

## Subscribe to Webhooks
Stripe uses webhooks to notify the server of events, such as a successful payment or a subscription cancellation. We need to subscribe to these webhooks so that we can handle them in the server.

To subscribe to webhooks for production:
1. In the Stripe dashboard, navigate to "Developers" in the navbar.
2. Click on the "Webhooks" tab.
3. Click "Add endpoint."
4. Enter the URL for our server's Stripe webhook handler (e.g. `https://vrooli.com/webhooks/stripe`).
5. Select the following events:
    - `checkout.session.completed`
    - `customer.session.expired`
    - `customer.deleted`
    - `customer.source.expiring`
    - `customer.subscription.deleted`
    - `customer.subscription.trial_will_end`
    - `customer.subscription.updated`
    - `invoice.created`
    - `invoice.payment_failed`
    - `invoice.payment_succeeded`
    - `price.updated`
Each event corresponds to a function in the server. If you don't select one of these events, there may be problems. If you select too many, you will get non-critical error logs in the server.

## Set Up the Server
To find the Stripe code in the server, search for `getPriceIds`. This is a function that stores all of the test and production price IDs. Replace these with the price IDs you created earlier. Other than that, the server should be ready to go!

**Note:** Refer to the [Stripe API documentation](https://stripe.com/docs/api) and [webhook documentation](https://stripe.com/docs/webhooks) when updating the server. ChatGPT also knows a lot about Stripe.

## Test the Server
To simulate events for local testing, use the Stripe CLI. This forwards events from your Stripe account to a local webhook running on your machine. It also allows you to trigger events manually. Here's how to set it up:

1. **Install and Login to the Stripe CLI:** You can either run `scripts/stripeSetup.sh`, or follow the instructions [here](https://stripe.com/docs/stripe-cli#install).
2. **Start listening to events:** Enter `stripe listen --forward-to localhost:5329/webhooks/stripe` in the terminal. If needed, update the port to match the `PORT_SERVER` environment variable. This will start listening for events and forward them to the server. 
3. **Update .env:** Update `STRIPE_WEBHOOK_SECRET` in the `.env` file, then run `./scripts/setSecrets -e development`. **NOTE:** Comment out the existing key instead of deleting it, so you can switch back to it later. You probably won't be able to find that key again.
4. **Fully restart the server:** In another shell, run `docker-compose down && docker-compose up -d` to fully restart the server. This is necessary to update the environment variables. Make sure that the Stripe CLI is still running in the other shell.
3. **Trigger events:** You should test the following events using our app's UI:
    - Buying every subscription plan option, donating, and buying API credits
    - Switching plans (e.g. monthly -> yearly)
    - Cancelling your plan

Refer to the [Stripe API documentation](https://stripe.com/docs/api) and [webhook documentation](https://stripe.com/docs/webhooks) when updating the server.
"""

## Fixing errors
Here are some errors I've run into before while testing the server locally, with their solutions.

### "You must provide at least one recurring price in `subscription` mode when using prices"
This error occurs when you try to create a subscription with a one-time price. To fix this, make sure you are using the correct price ID for the subscription. You can check this by going to the Stripe dashboard, clicking on the product, and clicking on the price. The price ID is in the URL.

If the price ID matches the intended price, then the price is probably not set up correctly. Look at the price's *Interval* field. If this says "One-Time" and you are setting up a subscription (or vice-versa), then you need to create a new price with the correct interval.
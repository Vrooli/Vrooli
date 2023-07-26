import { PaymentType } from "@local/shared";
import express, { Express, Request, Response } from "express";
import Stripe from "stripe";
import { logger } from "../events";
import { withRedis } from "../redisConn";
import { withPrisma } from "../utils/withPrisma";

// Disable Stripe if any of the required environment variables are missing 
// (Publishable key only required for client-side Stripe)
// NOTE: Make sure to update these are production keys before deploying
const canUseStripe = () => Boolean(process.env.STRIPE_SECRET_KEY) && Boolean(process.env.STRIPE_WEBHOOK_SECRET);

// Declare price IDs
// NOTE 1: Make sure to update these are production keys before deploying
// NOTE 2: Make sure these are PRICE IDs, not PRODUCT IDs
const STRIPE_PRICE_IDS = {
    donation: "price_1NG6LnJq1sLW02CV1bhcCYZG",
    premium: {
        monthly: "price_1MrTMEJq1sLW02CVHdm1U247",
        yearly: "price_1MrUzeJq1sLW02CVEFdKKQNu",
    },
};

/**
 * Returns the customer ID from the Stripe API response.
 * @param {string | Stripe.Customer | Stripe.DeletedCustomer | null} customer The customer field in a Stripe event.
 * This could be a string (representing the customer ID), a Stripe.Customer object, a Stripe.DeletedCustomer object, or null.
 * @returns {string} The customer ID as a string if it exists
 * @throws {Error} Throws an error if the customer ID doesn't exist or couldn't be found.
 */
const getCustomerId = (customer: string | Stripe.Customer | Stripe.DeletedCustomer | null | undefined): string => {
    if (typeof customer === "string") {
        return customer;
    } else if (customer && typeof customer === "object") {
        return customer.id; // Access 'id' field if 'customer' is an object
    } else {
        throw new Error("Customer ID not found");
    }
};

/**
 * Returns the subscription ID from the Stripe API response.
 * @param {string | Stripe.Subscription | null} subscription The subscription field in a Stripe event.
 * This could be a string (representing the subscription ID), a Stripe.Subscription object, or null.
 * @returns {string} The subscription ID as a string if it exists.
 * @throws {Error} Throws an error if the subscription ID doesn't exist or couldn't be found.
 */
function getSubscriptionId(subscription: string | Stripe.Subscription | null): string {
    if (typeof subscription === "string") {
        return subscription;
    } else if (subscription && typeof subscription === "object") {
        return subscription.id; // Access 'id' field if 'subscription' is an object
    } else {
        throw new Error("Subscription ID not found");
    }
}

/**
 * Sets up Stripe-related routes on the provided Express application instance.
 *
 * @param app - The Express application instance to attach routes to.
 */
export const setupStripe = (app: Express): void => {
    if (!canUseStripe()) {
        logger.error("Missing one or more Stripe secret keys", { trace: "0489" });
        return;
    }
    // Set up Stripe
    const stripe = new Stripe(process.env.STRIPE_SECRET_KEY!, {
        apiVersion: "2022-11-15",
        typescript: true,
        appInfo: {
            name: "Vrooli",
            url: "https://vrooli.com",
            version: "2.0.0",
        },
    });
    // Create endpoint for checking prices for premium subscriptions
    app.get("/api/premium-prices", async (req: Request, res: Response) => {
        // Initialize result
        const data: { monthly: number, yearly: number } = {
            monthly: NaN,
            yearly: NaN,
        };
        let wasMonthlyCached = false;
        let wasYearlyCached = false;
        // Try to find cached price
        await withRedis({
            process: async (redisClient) => {
                const cachedMonthlyPrice = await redisClient.get("premium-price-monthly");
                if (cachedMonthlyPrice) {
                    data.monthly = Number(cachedMonthlyPrice);
                    wasMonthlyCached = true;
                }
                const cachedYearlyPrice = await redisClient.get("premium-price-yearly");
                if (cachedYearlyPrice) {
                    data.yearly = Number(cachedYearlyPrice);
                    wasYearlyCached = true;
                }
            },
            trace: "0516",
        });
        // Fetch prices from Stripe
        if (!data.monthly) {
            const monthlyPrice = await stripe.prices.retrieve(STRIPE_PRICE_IDS.premium.monthly);
            data.monthly = monthlyPrice.unit_amount ?? 0;
        }
        if (!data.yearly) {
            const yearlyPrice = await stripe.prices.retrieve(STRIPE_PRICE_IDS.premium.yearly);
            data.yearly = yearlyPrice.unit_amount ?? 0;
        }
        // Cache prices
        await withRedis({
            process: async (redisClient) => {
                if (!wasMonthlyCached) {
                    await redisClient.set("premium-price-monthly", data.monthly);
                    await redisClient.expire("premium-price-monthly", 60 * 60 * 24);
                }
                if (!wasYearlyCached) {
                    await redisClient.set("premium-price-yearly", data.yearly);
                    await redisClient.expire("premium-price-yearly", 60 * 60 * 24);
                }
            },
            trace: "0517",
        });
        // Send result
        res.json({ data });
    });
    // Create endpoint for buying a premium subscription or donating
    app.post("/api/create-checkout-session", async (req: Request, res: Response) => {
        // Get userId and variant from request body
        const userId: string = req.body.userId;
        const variant: "yearly" | "monthly" | "donation" = req.body.variant;
        // Determine price API ID based on variant. Select a product in the Stripe dashboard 
        // to find this information
        let priceId: string;
        let paymentType: PaymentType;
        if (variant === "yearly") {
            priceId = STRIPE_PRICE_IDS.premium.yearly;
            paymentType = PaymentType.PremiumYearly;
        } else if (variant === "monthly") {
            priceId = STRIPE_PRICE_IDS.premium.monthly;
            paymentType = PaymentType.PremiumMonthly;
        } else if (variant === "donation") {
            priceId = STRIPE_PRICE_IDS.donation;
            paymentType = PaymentType.Donation;
        } else {
            logger.error("Invalid variant", { trace: "0436", userId, variant });
            res.status(400).json({ error: "Invalid variant" });
            return;
        }
        await withPrisma({
            process: async (prisma) => {
                // Get user from database
                const user = await prisma.user.findUnique({
                    where: { id: userId },
                    select: {
                        emails: { select: { emailAddress: true } },
                        stripeCustomerId: true,
                    },
                });
                if (!user) {
                    logger.error("User not found.", { trace: "0519", userId });
                    res.status(400).json({ error: "User not found." });
                    return;
                }
                // Create or retrieve a Stripe customer
                let stripeCustomerId = user.stripeCustomerId;
                // If customer exists, make sure it's valid
                if (stripeCustomerId) {
                    try {
                        // Check if the customer exists in Stripe
                        await stripe.customers.retrieve(stripeCustomerId);
                    } catch (error) {
                        // The customer does not exist in Stripe. Set stripeCustomerId to null so we can create a new customer
                        stripeCustomerId = null;
                        // Also log an error because this shouldn't happen
                        logger.error("Invalid Stripe customer ID", { trace: "0521", userId, stripeCustomerId, error });
                    }
                }
                // If customer doesn't exist (or the existing ID was invalid), create a new customer
                if (!stripeCustomerId) {
                    const stripeCustomer = await stripe.customers.create({
                        email: user?.emails?.length ? user.emails[0].emailAddress : undefined,
                    });
                    stripeCustomerId = stripeCustomer.id;
                    // Store Stripe customer ID in your database
                    await prisma.user.update({
                        where: { id: userId },
                        data: {
                            stripeCustomerId,
                        },
                    });
                }
                // Create checkout session
                try {
                    const session = await stripe.checkout.sessions.create({
                        success_url: "https://vrooli.com/premium?status=success",
                        cancel_url: "https://vrooli.com/premium?status=canceled",
                        payment_method_types: ["card"],
                        line_items: [
                            {
                                price: priceId,
                                quantity: 1,
                            },
                        ],
                        mode: variant === "donation" ? "payment" : "subscription",
                        customer: stripeCustomerId,
                        customer_email: user?.emails?.length ? user.emails[0].emailAddress : undefined,
                        metadata: { userId },
                    });
                    // Create open payment in database, so we can track it
                    // TODO create cron job to clean up old payments that never completed, and to remove premium status from users when expired
                    await prisma.payment.create({
                        data: {
                            amount: (variant === "donation" ? session.amount_total : session.amount_subtotal) ?? 0,
                            checkoutId: session.id,
                            currency: session.currency ?? "usd",
                            description: variant === "donation" ? "Donation" : "Premium subscription - " + variant,
                            paymentMethod: "Stripe",
                            paymentType,
                            status: "Pending",
                        },
                    });
                    // Send session ID as response
                    res.json(session);
                    return;
                } catch (error) {
                    logger.error("Error creating checkout session", { trace: "0437", userId, variant, error });
                    res.status(500).json({ error });
                    return;
                }
            },
            trace: "0437",
            traceObject: { userId, variant },
        });
    });
    // Create endpoint for updating payment method and switching/canceling subscription
    app.post("/api/create-portal-session", async (req: Request, res: Response) => {
        // Get userId from request body
        const userId: string = req.body.userId;

        let stripeCustomerId: string | undefined;
        await withPrisma({
            process: async (prisma) => {
                // Get user from database
                const user = await prisma.user.findUnique({
                    where: { id: userId },
                    select: {
                        stripeCustomerId: true,
                    },
                });
                if (!user) {
                    logger.error("User not found.", { trace: "0520", userId });
                    res.status(400).json({ error: "User not found." });
                    return;
                }
                stripeCustomerId = user.stripeCustomerId ?? undefined;
            },
            trace: "0520",
            traceObject: { userId },
        });

        // Check if the stripeCustomerId is valid
        if (!stripeCustomerId) {
            logger.error("Invalid user", { trace: "0515", userId });
            res.status(400).json({ error: "Invalid user" });
            return;
        }

        // Create a portal session
        try {
            const session = await stripe.billingPortal.sessions.create({
                customer: stripeCustomerId,
                return_url: "https://vrooli.com/",
            });
            // Send session URL as response
            res.json(session);
            return;
        } catch (error) {
            logger.error("Error creating portal session", { trace: "0516", userId, error });
            res.status(500).json({ error });
            return;
        }
    });
    // Create webhook endpoint for messages from Stripe (e.g. payment completed, subscription renewed/canceled)
    app.post("/webhook/stripe", express.raw({ type: "application/json" }), async (req: Request, res: Response) => {
        const sig: string | string[] = req.headers["stripe-signature"] || "";
        try {
            // Parse event
            const event = stripe.webhooks.constructEvent(req.body, sig, process.env.STRIPE_WEBHOOK_SECRET!);
            // Handle the event
            switch (event.type) {
                // Checkout completed for donation or subscription
                case "checkout.session.completed": {
                    const session = event.data.object as Stripe.Checkout.Session;
                    const checkoutId = session.id;
                    const customerId = getCustomerId(session.customer);
                    await withPrisma({
                        process: async (prisma) => {
                            // Find payment in database
                            const payments = await prisma.payment.findMany({
                                where: {
                                    checkoutId,
                                    paymentMethod: "Stripe",
                                    user: { stripeCustomerId: customerId },
                                },
                            });
                            // If not found, log error
                            if (payments.length === 0) {
                                logger.error("Payment not found. Someone is gonna be mad", { trace: "0439", checkoutId, userId });
                                return;
                            }
                            // Update payment in database to indicate it was paid
                            const payment = await prisma.payment.update({
                                where: { id: payments[0].id },
                                data: {
                                    status: "Paid",
                                },
                                select: {
                                    user: { select: { id: true } },
                                },
                            });
                            // If subscription, upsert premium status
                            if (payments[0].paymentType === PaymentType.PremiumMonthly || payments[0].paymentType === PaymentType.PremiumYearly) {
                                const enabledAt = new Date().toISOString();
                                const expiresAt = new Date(Date.now() + (payments[0].paymentType === PaymentType.PremiumMonthly ? 30 : 365) * 24 * 60 * 60 * 1000).toISOString();
                                const isActive = true;
                                const premiums = await prisma.premium.findMany({
                                    where: { user: { id: payment.user!.id } }, // User should exist based on findMany query above
                                });
                                if (premiums.length === 0) {
                                    await prisma.premium.create({
                                        data: {
                                            enabledAt,
                                            expiresAt,
                                            isActive,
                                            user: { connect: { id: payment.user!.id } }, // User should exist based on findMany query above
                                        },
                                    });
                                } else {
                                    await prisma.premium.update({
                                        where: { id: premiums[0].id },
                                        data: {
                                            enabledAt: premiums[0].enabledAt ?? enabledAt,
                                            expiresAt,
                                            isActive,
                                        },
                                    });
                                }
                            }
                        },
                        trace: "0454",
                        traceObject: { customerId, session, checkoutId },
                    });
                    break;
                }
                // Checkout expired before payment
                case "checkout.session.expired": {
                    const session = event.data.object as Stripe.Checkout.Session;
                    const checkoutId = session.id;
                    const customerId = getCustomerId(session.customer);
                    await withPrisma({
                        process: async (prisma) => {
                            // Find payment in database
                            const payments = await prisma.payment.findMany({
                                where: {
                                    checkoutId,
                                    paymentMethod: "Stripe",
                                    status: "Pending",
                                    user: { stripeCustomerId: customerId },
                                },
                            });
                            if (payments.length > 0) {
                                // Delete payment in database
                                await prisma.payment.delete({
                                    where: { id: payments[0].id },
                                });
                            }
                        },
                        trace: "0455",
                        traceObject: { customerId, session, checkoutId },
                    });
                    break;
                }
                // Customer was deleted
                case "customer.deleted": {
                    const customer = event.data.object as Stripe.Customer;
                    const customerId = getCustomerId(customer);
                    await withPrisma({
                        process: async (prisma) => {
                            // Remove customer ID from user in database
                            await prisma.user.update({
                                where: { stripeCustomerId: customerId },
                                data: { stripeCustomerId: null },
                            });
                        },
                        trace: "0522",
                        traceObject: { customerId },
                    });
                    break;
                }
                // Customer updated their payment method
                case "customer.source.updated": {
                    //TODO for morning: complete updating all of these webhook cases. Also figure out which ones should notify user (e.g. payment failed), and add notification logic
                    const source = event.data.object as Stripe.Source;
                    const customerId = getCustomerId(source.customer);
                    await withPrisma({
                        process: async (prisma) => {
                            //TODO
                        },
                        trace: "0523",
                        traceObject: { customerId },
                    });
                    break;
                }
                // User added a subscription outside of a checkout session (e.g. we did it in the Stripe dashboard)
                case "customer.subscription.updated": {
                    const subscription = event.data.object as Stripe.Subscription;
                    const customerId = getCustomerId(subscription.customer);
                    //TODO
                    break;
                }
                // User canceled subscription
                case "customer.subscription.deleted": {
                    const subscription = event.data.object as Stripe.Subscription;
                    const customerId = getCustomerId(subscription.customer);
                    await withPrisma({
                        process: async (prisma) => {
                            // Find premium record in database
                            const canceledPremiumRecord = await prisma.premium.findFirst({
                                where: { user: { id: userId } },
                            });
                            // If found, set as inactive
                            if (canceledPremiumRecord) {
                                await prisma.premium.update({
                                    where: { id: canceledPremiumRecord.id },
                                    data: {
                                        isActive: false,
                                    },
                                });
                            }
                            // Otherwise, log an error and return
                            else {
                                logger.error("Premium record not found on customer.subscription.deleted", { trace: "0459", userId });
                                res.status(400).send("Premium record not found");
                                return;
                            }
                        },
                        trace: "0460",
                        traceObject: { userId },
                    });
                    break;
                }
                // User changed subscription (monthly -> yearly, yearly -> monthly)
                case "customer.subscription.updated": {
                    const subscription = event.data.object as Stripe.Subscription;
                    const customerId = getCustomerId(subscription.customer);
                    const newPaymentType = subscription.plan.interval === "month" ? PaymentType.PremiumMonthly : PaymentType.PremiumYearly;
                    await withPrisma({
                        process: async (prisma) => {
                            // Find premium record in database
                            const premiumRecord = await prisma.premium.findFirst({
                                where: { user: { id: userId } },
                            });
                            // If found, update the expiration date
                            if (premiumRecord) {
                                await prisma.premium.update({
                                    where: { id: premiumRecord.id },
                                    data: {
                                        expiresAt: new Date(Date.now() + (newPaymentType === PaymentType.PremiumMonthly ? 30 : 365) * 24 * 60 * 60 * 1000).toISOString(),
                                    },
                                });
                            }
                            // Otherwise, log an error and return
                            else {
                                logger.error("Premium record not found on customer.subscription.updated", { trace: "0457", userId });
                                res.status(400).send("Premium record not found");
                                return;
                            }
                        },
                        trace: "0458",
                        traceObject: { userId, subscription, newPaymentType },
                    });
                    break;
                }
                // Payment failed
                case "invoice.payment_failed": {
                    const invoice = event.data.object as Stripe.Invoice;
                    const subscriptionId = getSubscriptionId(invoice.subscription);
                    const customerId = getCustomerId(invoice.customer);
                    await withPrisma({
                        process: async (prisma) => {
                            // Find payment in database
                            const payments = await prisma.payment.findMany({
                                where: {
                                    user: { stripeCustomerId: customerId },
                                    paymentMethod: "Stripe",
                                },
                            });
                            // If not found, log an error and return
                            if (payments.length === 0) {
                                logger.error("Payment not found on invoice.payment_failed", { trace: "0443", customerId });
                                res.status(400).send("Payment not found");
                                return;
                            }
                            // Update the payment status to "Failed" in the database
                            await prisma.payment.update({
                                where: { id: payments[0].id },
                                data: {
                                    status: "Failed",
                                },
                            });
                        },
                        trace: "0456",
                        traceObject: { customerId, invoice, subscriptionId },
                    });
                    // Log the error
                    logger.error("Payment failed", { trace: "0444", customerId, subscriptionId });
                    break;
                }
                // Subscription automatically renewed
                case "invoice.payment_succeeded": {
                    const invoice = event.data.object as Stripe.Invoice;
                    const subscriptionId = getSubscriptionId(invoice.subscription);
                    const customerId = getCustomerId(invoice.customer);
                    await withPrisma({
                        process: async (prisma) => {
                            // Find premium record in database
                            const renewedPremiumRecord = await prisma.premium.findFirst({
                                where: { user: { id: userId } },
                            });
                            // If found, set as active and update the expiration date
                            if (renewedPremiumRecord) {
                                const renewedPaymentType = renewedPremiumRecord.expiresAt && new Date(renewedPremiumRecord.expiresAt) < new Date() ? PaymentType.PremiumMonthly : PaymentType.PremiumYearly;
                                await prisma.premium.update({
                                    where: { id: renewedPremiumRecord.id },
                                    data: {
                                        expiresAt: new Date(Date.now() + (renewedPaymentType === PaymentType.PremiumMonthly ? 30 : 365) * 24 * 60 * 60 * 1000).toISOString(),
                                        isActive: true,
                                    },
                                });
                            }
                            // Otherwise, log an error and return
                            else {
                                logger.error("Premium record not found on invoice.payment_succeeded", { trace: "0461", userId });
                                res.status(400).send("Premium record not found");
                                return;
                            }
                        },
                        trace: "0462",
                        traceObject: { userId, renewedInvoice, renewedSubscriptionId },
                    });
                    break;
                }
                // Details of a price have been updated
                // NOTE: You can't actually update a price's price; you have to create a new price. 
                // This means this condition isn't needed unless Stripe changes how prices work. 
                // But I already wrote it, so I'm leaving it here.
                case "price.updated": {
                    const price = event.data.object as Stripe.Price;
                    // Find the price ID and updated price
                    const updatedPriceId = price.id;
                    const updatedAmount = price.unit_amount;
                    // Update cached price
                    await withRedis({
                        process: async (redisClient) => {
                            let key: string | null = null;
                            // Check if the updated price ID matches any of our IDs
                            if (updatedPriceId === STRIPE_PRICE_IDS.premium.monthly) {
                                key = "premium-price-monthly";
                            } else if (updatedPriceId === STRIPE_PRICE_IDS.premium.yearly) {
                                key = "premium-price-yearly";
                            }
                            // If a matching key was found, update the value in Redis
                            if (key) {
                                if (updatedAmount === null) {
                                    await redisClient.del(key);
                                } else {
                                    await redisClient.set(key, updatedAmount);
                                    await redisClient.expire(key, 60 * 60 * 24); // expire after 24 hours
                                }
                            }
                        },
                        trace: "0518",
                    });
                    break;
                }
                // If this default is reached, either the webhook specified in Stripe is too broad, 
                // or we're missing an important event
                default:
                    logger.warning("Unhandled Stripe event", { trace: "0438", event: event.type });
                    break;
            }
        } catch (error: any) {
            res.status(400).send(`Webhook Error: ${error.message}`);
            return;
        }
        // Return a 200 response to acknowledge receipt of the event
        res.send();
    });
};

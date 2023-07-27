import { HttpStatus, PaymentType } from "@local/shared";
import { PaymentStatus } from "@prisma/client";
import express, { Express, Request, Response } from "express";
import Stripe from "stripe";
import { logger } from "../events";
import { withRedis } from "../redisConn";
import { sendCreditCardExpiringSoon, sendPaymentFailed, sendPaymentThankYou, sendSubscriptionCanceled } from "../tasks";
import { PrismaType } from "../types";
import { withPrisma } from "../utils/withPrisma";

type EventHandlerArgs = {
    event: Stripe.Event,
    prisma: PrismaType,
    stripe: Stripe,
    res: Response,
}

type HandlerResult = {
    status: HttpStatus;
    message?: string;
}

type Handler = (args: EventHandlerArgs) => Promise<HandlerResult>;

/** 
 * Finds all available price IDs for the current environment (development or production)
 * NOTE: Make sure these are PRICE IDs, not PRODUCT IDs
 **/
const getPriceIds = () => {
    if (process.env.NODE_ENV === "development") {
        return {
            donation: "price_1NG6LnJq1sLW02CV1bhcCYZG",
            premium: {
                monthly: "price_1NYaGQJq1sLW02CVFAO6bPu4",
                yearly: "price_1MrUzeJq1sLW02CVEFdKKQNu",
            },
        };
    }
    return {
        donation: "TODO",
        premium: {
            monthly: "TODO",
            yearly: "TODO",
        },
    };
};

/** Finds the payment type of a Stripe product*/
const getPaymentType = (product: string | Stripe.Product | Stripe.DeletedProduct): PaymentType => {
    const productId = typeof product === "string" ? product : product.id;
    if (productId === getPriceIds().donation) {
        return PaymentType.Donation;
    } else if (productId === getPriceIds().premium.monthly) {
        return PaymentType.PremiumMonthly;
    } else if (productId === getPriceIds().premium.yearly) {
        return PaymentType.PremiumYearly;
    } else {
        throw new Error("Invalid product ID");
    }
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
 * Simplifies the return logic for Stripe event handlers, by handling 
 * logging, setting res.status, and returning a message.
 * @param status HTTP status code
 * @param res Express response object
 * @param message Message to return
 * @param trace Trace to log
 * @param ...args Additional arguments to log
 * @returns Object with status and message
 */
const handlerResult = (
    status: HttpStatus,
    res: Response,
    message?: string,
    trace?: string,
    ...args: unknown[]
): HandlerResult => {
    if (status !== HttpStatus.Ok) {
        logger.error(message ?? "Stripe handler error", { trace: trace ?? "0523", ...args });
    }
    res.status(status).send(message);
    return { status, message };
};

/** Checkout completed for donation or subscription */
const handleCheckoutSessionCompleted = async ({ event, prisma, stripe, res }: EventHandlerArgs): Promise<HandlerResult> => {
    const session = event.data.object as Stripe.Checkout.Session;
    const checkoutId = session.id;
    const customerId = getCustomerId(session.customer);
    // Find payment in database
    const payments = await prisma.payment.findMany({
        where: {
            checkoutId,
            paymentMethod: "Stripe",
            user: { stripeCustomerId: customerId },
        },
    });
    // If not found, log error. This is BAD
    if (payments.length === 0) {
        return handlerResult(HttpStatus.InternalServerError, res, "Payment not found.", "0439", { checkoutId, customerId });
    }
    // Update payment in database to indicate it was paid
    const payment = await prisma.payment.update({
        where: { id: payments[0].id },
        data: {
            status: "Paid",
        },
        select: {
            paymentType: true,
            user: {
                select: {
                    id: true,
                    emails: { select: { emailAddress: true } },
                },
            },
        },
    });
    // If there is not a user associated with this payment, log an error and return
    if (!payment.user) {
        return handlerResult(HttpStatus.InternalServerError, res, "User not found.", "0440", { checkoutId, customerId });
    }
    // If subscription, upsert premium status
    if (payments[0].paymentType === PaymentType.PremiumMonthly || payments[0].paymentType === PaymentType.PremiumYearly) {
        // Get subscription
        if (!session.subscription) {
            return handlerResult(HttpStatus.InternalServerError, res, "Subscription not found.", "0224", { checkoutId, customerId });
        }
        // If subscription is a string (i.e. the ID), fetch the full subscription object from Stripe
        const subscription = typeof session.subscription === "string" ?
            await stripe.subscriptions.retrieve(session.subscription) :
            session.subscription;
        // Find enabledAt and expiresAt
        const enabledAt = new Date().toISOString();
        const expiresAt = new Date(subscription.current_period_end * 1000).toISOString();
        // Upsert premium status
        const premiums = await prisma.premium.findMany({
            where: { user: { id: payment.user.id } }, // User should exist based on findMany query above
        });
        if (premiums.length === 0) {
            await prisma.premium.create({
                data: {
                    enabledAt,
                    expiresAt,
                    isActive: true,
                    user: { connect: { id: payment.user.id } }, // User should exist based on findMany query above
                },
            });
        } else {
            await prisma.premium.update({
                where: { id: premiums[0].id },
                data: {
                    enabledAt: premiums[0].enabledAt ?? enabledAt,
                    expiresAt,
                    isActive: true,
                },
            });
        }
    }
    // Send thank you notification
    for (const email of payment.user.emails) {
        sendPaymentThankYou(email.emailAddress, payment.paymentType);
    }
    return handlerResult(HttpStatus.Ok, res);
};

/** Checkout expired before payment */
const handleCheckoutSessionExpired = async ({ event, prisma, res }: EventHandlerArgs): Promise<HandlerResult> => {
    const session = event.data.object as Stripe.Checkout.Session;
    const checkoutId = session.id;
    const customerId = getCustomerId(session.customer);
    // Find payment in database
    const payments = await prisma.payment.findMany({
        where: {
            checkoutId,
            paymentMethod: "Stripe",
            status: "Pending",
            user: { stripeCustomerId: customerId },
        },
    });
    if (payments.length === 0) {
        return handlerResult(HttpStatus.InternalServerError, res, "Payment not found.", "0425", { checkoutId, customerId });
    }
    // Delete payment in database
    await prisma.payment.delete({
        where: { id: payments[0].id },
    });
    return handlerResult(HttpStatus.Ok, res);
};

/** Customer was deleted */
const handleCustomerDeleted = async ({ event, prisma, res }: EventHandlerArgs): Promise<HandlerResult> => {
    const customer = event.data.object as Stripe.Customer;
    const customerId = getCustomerId(customer);
    // Remove customer ID from user in database
    await prisma.user.update({
        where: { stripeCustomerId: customerId },
        data: { stripeCustomerId: null },
    });
    return handlerResult(HttpStatus.Ok, res);
};

/** Customer's credit card is about to expire */
const handleCustomerSourceExpiring = async ({ event, prisma, res }: EventHandlerArgs): Promise<HandlerResult> => {
    const customer = event.data.object as Stripe.Customer;
    const customerId = getCustomerId(customer);
    // Find user with the given Stripe customer ID
    const user = await prisma.user.findFirst({
        where: { stripeCustomerId: customerId },
        select: {
            emails: { select: { emailAddress: true } },
        },
    });
    if (!user) {
        return handlerResult(HttpStatus.InternalServerError, res, "User not found", "0468", { customerId });
    }
    // Send notification
    for (const email of user.emails) {
        sendCreditCardExpiringSoon(email.emailAddress);
    }
    return handlerResult(HttpStatus.Ok, res);
};

/**
 * User canceled subscription. They still have paid for their current 
 * billing period, so don't set as inactive
 */
const handleCustomerSubscriptionDeleted = async ({ event, prisma, res }: EventHandlerArgs): Promise<HandlerResult> => {
    const subscription = event.data.object as Stripe.Subscription;
    const customerId = getCustomerId(subscription.customer);
    // Find user 
    const user = await prisma.user.findFirst({
        where: { stripeCustomerId: customerId },
        select: {
            emails: { select: { emailAddress: true } },
        },
    });
    if (!user) {
        return handlerResult(HttpStatus.InternalServerError, res, "User not found.", "0459", { customerId });
    }
    // If found, send notification
    for (const email of user.emails) {
        sendSubscriptionCanceled(email.emailAddress);
    }
    return handlerResult(HttpStatus.Ok, res);
};

/** Trial ending in a few days */
const handleCustomerSubscriptionTrialWillEnd = async ({ res }: EventHandlerArgs): Promise<HandlerResult> => {
    return handlerResult(HttpStatus.NotImplemented, res, "customer.subscription.trial_will_end not implemented", "0460");
};

/** User updated subscription (e.g. switched from monthly to yearly) */
const handleCustomerSubscriptionUpdated = async ({ event, prisma, res }: EventHandlerArgs): Promise<HandlerResult> => {
    const subscription = event.data.object as Stripe.Subscription;
    const customerId = getCustomerId(subscription.customer);
    // Check that subscription contains exactly one item
    if (subscription.items.data.length !== 1) {
        return handlerResult(HttpStatus.InternalServerError, res, `Subscription contains ${subscription.items.data.length} items`, "0471", { customerId });
    }
    // Find premium record in database
    const premiumRecord = await prisma.premium.findFirst({
        where: { user: { stripeCustomerId: customerId } },
    });
    if (!premiumRecord) {
        return handlerResult(HttpStatus.InternalServerError, res, "Premium record not found on customer.subscription.updated", "0457", { customerId });
    }
    // Update the expiration date
    if (premiumRecord) {
        // Update expiration date based on current period end of subscription
        const nextBillingCycleTimestamp = subscription.current_period_end;
        const newExpiresAt = new Date(nextBillingCycleTimestamp * 1000);

        await prisma.premium.update({
            where: { id: premiumRecord.id },
            data: { expiresAt: newExpiresAt.toISOString() },
        });
    }
    return handlerResult(HttpStatus.Ok, res);
};

/** Payment created, but not finalized */
const handleInvoicePaymentCreated = async ({ event, prisma, res }: EventHandlerArgs): Promise<HandlerResult> => {
    const invoice = event.data.object as Stripe.Invoice;
    const customerId = getCustomerId(invoice.customer);
    // Find user with the given Stripe customer ID
    const user = await prisma.user.findFirst({
        where: { stripeCustomerId: customerId },
    });
    // Check if user is found
    if (!user) {
        return handlerResult(HttpStatus.InternalServerError, res, "User not found", "0456", { customerId, invoice: invoice.id });
    }
    // Check if invoice.lines, invoice.lines.data[0], invoice.lines.data[0].price, and invoice.lines.data[0].price.product are not null
    const product = invoice.lines && invoice.lines.data[0] && invoice.lines.data[0].price && invoice.lines.data[0].price.product;
    if (!product) {
        return handlerResult(HttpStatus.InternalServerError, res, "Product not found", "0225", { customerId, invoice: invoice.id });
    }
    // Create new payment in the database
    const newPayment = await prisma.payment.create({
        data: {
            checkoutId: invoice.id,
            amount: invoice.amount_due,
            currency: invoice.currency,
            paymentMethod: "Stripe",
            paymentType: getPaymentType(product),
            status: PaymentStatus.Pending,
            userId: user.id,
            description: "Pending payment for Invoice ID: " + invoice.id,
        },
    });
    // Check if payment creation was successful
    if (!newPayment) {
        return handlerResult(HttpStatus.InternalServerError, res, "Payment creation failed", "0458", { customerId, invoice: invoice.id });
    }
    return handlerResult(HttpStatus.Ok, res);
};

/** Payment failed */
const handleInvoicePaymentFailed = async ({ event, prisma, res }: EventHandlerArgs): Promise<HandlerResult> => {
    const invoice = event.data.object as Stripe.Invoice;
    const customerId = getCustomerId(invoice.customer);
    // Find corresponding payment in the database
    const payment = await prisma.payment.findFirst({
        where: {
            checkoutId: invoice.id,
            user: { stripeCustomerId: customerId },
        },
        select: {
            id: true,
            paymentType: true,
            user: {
                select: {
                    emails: { select: { emailAddress: true } },
                },
            },
        },
    });
    if (!payment) {
        return handlerResult(HttpStatus.InternalServerError, res, "Invoice payment not found.", "0444", { customerId, invoice: invoice.id });
    }
    // Update the payment status to Failed
    await prisma.payment.update({
        where: { id: payment.id },
        data: { status: PaymentStatus.Failed },
    });
    // Notify user
    for (const email of payment.user?.emails ?? []) {
        sendPaymentFailed(email.emailAddress, payment.paymentType);
    }
    return handlerResult(HttpStatus.Ok, res);
};


/** Payment succeeded (e.g. subscription renewed) */
const handleInvoicePaymentSucceeded = async ({ event, prisma, stripe, res }: EventHandlerArgs): Promise<HandlerResult> => {
    const invoice = event.data.object as Stripe.Invoice;
    const customerId = getCustomerId(invoice.customer);
    // Find the user associated with the invoice
    const user = await prisma.user.findFirst({
        where: { stripeCustomerId: customerId },
    });
    if (!user) {
        return handlerResult(HttpStatus.InternalServerError, res, "User not found.", "0461", { customerId, invoice: invoice.id });
    }
    // Find the corresponding payment in the database
    const payment = await prisma.payment.findFirst({
        where: {
            checkoutId: invoice.id,
            user: { stripeCustomerId: customerId },
        },
        select: {
            id: true,
            paymentType: true,
            user: {
                select: {
                    id: true,
                },
            },
        },
    });
    if (!payment) {
        return handlerResult(HttpStatus.InternalServerError, res, "Invoice payment not found", "0462", { customerId, invoice: invoice.id });
    }
    // Update the payment status to indicate it was successfully paid
    await prisma.payment.update({
        where: { id: payment.id },
        data: { status: "Paid" },
    });
    if (!payment.user) {
        return handlerResult(HttpStatus.InternalServerError, res, "User not found.", "0223", { customerId, invoice: invoice.id });
    }
    // If subscription, update or create premium status
    if (payment.paymentType === PaymentType.PremiumMonthly || payment.paymentType === PaymentType.PremiumYearly) {
        // Fetch subscription from Stripe
        const stripeSubscription = await stripe.subscriptions.retrieve(invoice.subscription as string);
        const expiresAt = new Date(stripeSubscription.current_period_end * 1000).toISOString();
        const premiums = await prisma.premium.findMany({
            where: { user: { id: payment.user.id } },
        });
        if (premiums.length === 0) {
            await prisma.premium.create({
                data: {
                    enabledAt: new Date().toISOString(),
                    expiresAt,
                    isActive: true,
                    user: { connect: { id: payment.user.id } },
                },
            });
        } else {
            await prisma.premium.update({
                where: { id: premiums[0].id },
                data: {
                    expiresAt,
                    isActive: true,
                },
            });
        }
    }
    // Probably? don't need to send an email for this one
    return handlerResult(HttpStatus.Ok, res);
};

/**
 * Details of a price have been updated
 * NOTE: You can't actually update a price's price; you have to create a new price. 
 * This means this condition isn't needed unless Stripe changes how prices work. 
 * But I already wrote it, so I'm leaving it here.
 */
const handlePriceUpdated = async ({ event, res }: EventHandlerArgs): Promise<HandlerResult> => {
    const price = event.data.object as Stripe.Price;
    // Find the price ID and updated price
    const updatedPriceId = price.id;
    const updatedAmount = price.unit_amount;
    // Update cached price
    await withRedis({
        process: async (redisClient) => {
            let key: string | null = null;
            // Check if the updated price ID matches any of our IDs
            if (updatedPriceId === getPriceIds().premium.monthly) {
                key = "premium-price-monthly";
            } else if (updatedPriceId === getPriceIds().premium.yearly) {
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
    return handlerResult(HttpStatus.Ok, res);
};

/**
 * Sets up Stripe-related routes on the provided Express application instance.
 *
 * @param app - The Express application instance to attach routes to.
 */
export const setupStripe = (app: Express): void => {
    if (process.env.STRIPE_SECRET_KEY === undefined || process.env.STRIPE_WEBHOOK_SECRET === undefined) {
        logger.error("Missing one or more Stripe secret keys", { trace: "0489" });
        return;
    }
    // Set up Stripe
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
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
            const monthlyPrice = await stripe.prices.retrieve(getPriceIds().premium.monthly);
            data.monthly = monthlyPrice.unit_amount ?? 0;
        }
        if (!data.yearly) {
            const yearlyPrice = await stripe.prices.retrieve(getPriceIds().premium.yearly);
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
            priceId = getPriceIds().premium.yearly;
            paymentType = PaymentType.PremiumYearly;
        } else if (variant === "monthly") {
            priceId = getPriceIds().premium.monthly;
            paymentType = PaymentType.PremiumMonthly;
        } else if (variant === "donation") {
            priceId = getPriceIds().donation;
            paymentType = PaymentType.Donation;
        } else {
            logger.error("Invalid variant", { trace: "0436", userId, variant });
            res.status(HttpStatus.BadRequest).json({ error: "Invalid variant" });
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
                    res.status(HttpStatus.InternalServerError).json({ error: "User not found." });
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
                try {
                    const urlBase = process.env.VITE_SERVER_LOCATION === "local" ? "http://localhost:3000" : "https://vrooli.com";
                    // Create checkout session 
                    // NOTE: I previously tried also passing in the customer email 
                    // to save the customer some typing, but Stripe doesn't allow 
                    // this to be sent along with the customer ID.
                    const session = await stripe.checkout.sessions.create({
                        success_url: `${urlBase}/premium?status=success`,
                        cancel_url: `${urlBase}/premium?status=canceled`,
                        payment_method_types: ["card"],
                        line_items: [
                            {
                                price: priceId,
                                quantity: 1,
                            },
                        ],
                        mode: variant === "donation" ? "payment" : "subscription",
                        customer: stripeCustomerId,
                        metadata: { userId },
                    });
                    // Create open payment in database, so we can track it
                    await prisma.payment.create({
                        data: {
                            amount: (variant === "donation" ? session.amount_total : session.amount_subtotal) ?? 0,
                            checkoutId: session.id,
                            currency: session.currency ?? "usd",
                            description: variant === "donation" ? "Donation" : "Premium subscription - " + variant,
                            paymentMethod: "Stripe",
                            paymentType,
                            status: "Pending",
                            user: { connect: { id: userId } },
                        },
                    });
                    // Send session ID as response
                    res.json(session);
                    return;
                } catch (error) {
                    logger.error("Error creating checkout session", { trace: "0437", userId, variant, error });
                    res.status(HttpStatus.InternalServerError).json({ error });
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
                    res.status(HttpStatus.InternalServerError).json({ error: "User not found." });
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
            res.status(HttpStatus.InternalServerError).json({ error: "Invalid user" });
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
            res.status(HttpStatus.InternalServerError).json({ error });
            return;
        }
    });
    // Create webhook endpoint for messages from Stripe (e.g. payment completed, subscription renewed/canceled)
    app.post("/webhook/stripe", express.raw({ type: "application/json" }), async (req: Request, res: Response) => {
        const sig: string | string[] = req.headers["stripe-signature"] || "";
        let result: HandlerResult = { status: HttpStatus.InternalServerError, message: "Webhook encountered an error." };
        await withPrisma({
            process: async (prisma) => {
                // Parse event
                // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
                const event = stripe.webhooks.constructEvent(req.body, sig, process.env.STRIPE_WEBHOOK_SECRET!);
                // Call appropriate handler
                const eventHandlers: { [key: string]: Handler } = {
                    "checkout.session.completed": handleCheckoutSessionCompleted,
                    "checkout.session.expired": handleCheckoutSessionExpired,
                    "customer.deleted": handleCustomerDeleted,
                    "customer.source.expiring": handleCustomerSourceExpiring,
                    "customer.subscription.deleted": handleCustomerSubscriptionDeleted,
                    "customer.subscription.trial_will_end": handleCustomerSubscriptionTrialWillEnd,
                    "customer.subscription.updated": handleCustomerSubscriptionUpdated,
                    "invoice.created": handleInvoicePaymentCreated,
                    "invoice.payment_failed": handleInvoicePaymentFailed,
                    "invoice.payment_succeeded": handleInvoicePaymentSucceeded,
                    "price.updated": handlePriceUpdated,
                };
                if (event.type in eventHandlers) {
                    result = await eventHandlers[event.type]({ event, prisma, stripe, res });
                } else {
                    logger.warning("Unhandled Stripe event", { trace: "0438", event: event.type });
                }
            },
            trace: "0454",
        });
        res.status(result.status);
        if (result.message) {
            res.send(result.message);
        } else {
            res.send();
        }
    });
};

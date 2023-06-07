import { PaymentType } from "@local/shared";
import express, { Express, Request, Response } from "express";
import Stripe from "stripe";
import { logger } from "../events";
import { withPrisma } from "../utils/withPrisma";

// Disable Stripe if any of the required environment variables are missing 
// (Publishable key only required for client-side Stripe)
// NOTE: Make sure to update these are production keys before deploying
const canUseStripe = Boolean(process.env.STRIPE_SECRET_KEY) && Boolean(process.env.STRIPE_WEBHOOK_SECRET);

// Declare price IDs
// NOTE 1: Make sure to update these are production keys before deploying
// NOTE 2: Make sure these are PRICE IDs, not PRODUCT IDs
const prices = {
    donation: "price_1NG6LnJq1sLW02CV1bhcCYZG",
    premium: {
        monthly: "price_1MrTMEJq1sLW02CVHdm1U247",
        yearly: "price_1MrUzeJq1sLW02CVEFdKKQNu",
    },
};

/**
 * Sets up Stripe-related routes on the provided Express application instance.
 *
 * @param app - The Express application instance to attach routes to.
 */
export const setupStripe = (app: Express): void => {
    if (!canUseStripe) {
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
            priceId = prices.premium.yearly;
            paymentType = PaymentType.PremiumYearly;
        } else if (variant === "monthly") {
            priceId = prices.premium.monthly;
            paymentType = PaymentType.PremiumMonthly;
        } else if (variant === "donation") {
            priceId = prices.donation;
            paymentType = PaymentType.Donation;
        } else {
            logger.error("Invalid variant", { trace: "0436", userId, variant });
            res.status(400).json({ error: "Invalid variant" });
            return;
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
                metadata: { userId },
            });
            // Create open payment in database, so we can track it
            // TODO create cron job to clean up old payments that never completed, and to remove premium status from users when expired
            await withPrisma({
                process: async (prisma) => {
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
                },
                trace: "0437",
                traceObject: { userId, variant },
            });
            // Send session ID as response
            res.json(session);
            return;
        } catch (error) {
            logger.error("Error creating checkout session", { trace: "0437", userId, variant, error });
            res.status(500).json({ error });
            return;
        }
    });
    // Create webhook endpoint for messages from Stripe (e.g. payment completed, subscription renewed/canceled)
    app.post("/webhook/stripe", express.raw({ type: "application/json" }), async (req: Request, res: Response) => {
        const sig: string | string[] = req.headers["stripe-signature"] || "";
        let event;
        try {
            // Parse event
            event = stripe.webhooks.constructEvent(req.body, sig, process.env.STRIPE_WEBHOOK_SECRET!);
            let userId;
            let prisma;
            let payments;
            let session;
            let checkoutId;
            // Handle the event
            switch (event.type) {
                // Donation completed or subscription created
                case "checkout.session.completed":
                    // Find payment in database
                    session = event.data.object;
                    checkoutId = session.id;
                    userId = session.metadata.userId;
                    await withPrisma({
                        process: async (prisma) => {
                            // Find payment in database
                            payments = await prisma.payment.findMany({
                                where: {
                                    checkoutId,
                                    paymentMethod: "Stripe",
                                    user: { id: userId },
                                },
                            });
                            // If not found, log error
                            if (payments.length === 0) {
                                logger.error("Payment not found. Someone is gonna be mad", { trace: "0439", checkoutId, userId });
                                return;
                            }
                            // Update payment in database to indicate it was paid
                            await prisma.payment.update({
                                where: { id: payments[0].id },
                                data: {
                                    status: "Paid",
                                },
                            });
                            // If subscription, upsert premium status
                            if (payments[0].paymentType === PaymentType.PremiumMonthly || payments[0].paymentType === PaymentType.PremiumYearly) {
                                const enabledAt = new Date().toISOString();
                                const expiresAt = new Date(Date.now() + (payments[0].paymentType === PaymentType.PremiumMonthly ? 30 : 365) * 24 * 60 * 60 * 1000).toISOString();
                                const isActive = true;
                                const premiums = await prisma.premium.findMany({
                                    where: { user: { id: userId } },
                                });
                                if (premiums.length === 0) {
                                    await prisma.premium.create({
                                        data: {
                                            enabledAt,
                                            expiresAt,
                                            isActive,
                                            user: { connect: { id: userId } },
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
                        traceObject: { userId, session, checkoutId },
                    });
                    break;
                // Session expired before payment
                case "checkout.session.expired":
                    // Find payment in database
                    session = event.data.object;
                    checkoutId = session.id;
                    userId = session.metadata.userId;
                    await withPrisma({
                        process: async (prisma) => {
                            // Find payment in database
                            payments = await prisma.payment.findMany({
                                where: {
                                    checkoutId,
                                    paymentMethod: "Stripe",
                                    status: "Pending",
                                    user: { id: userId },
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
                        traceObject: { userId, session, checkoutId },
                    });
                    break;
                // Payment failed
                case "invoice.payment_failed": {
                    const invoice = event.data.object;
                    const subscriptionId = invoice.subscription;
                    userId = invoice.metadata.userId;
                    await withPrisma({
                        process: async (prisma) => {
                            // Find payment in database
                            payments = await prisma.payment.findMany({
                                where: {
                                    user: { id: userId },
                                    paymentMethod: "Stripe",
                                },
                            });
                            // If not found, log an error and return
                            if (payments.length === 0) {
                                logger.error("Payment not found on invoice.payment_failed", { trace: "0443", userId });
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
                        traceObject: { userId, invoice, subscriptionId },
                    });
                    // Log the error
                    logger.error("Payment failed", { trace: "0444", userId, subscriptionId });
                    break;
                }
                // Subscription changed (monthly -> yearly, yearly -> monthly)
                case "customer.subscription.updated": {
                    const subscription = event.data.object;
                    userId = subscription.metadata.userId;
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
                // Subscription canceled
                case "customer.subscription.deleted":
                    userId = event.data.object.metadata.userId;
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
                // Subscription renewed
                case "invoice.payment_succeeded": {
                    const renewedInvoice = event.data.object;
                    const renewedSubscriptionId = renewedInvoice.subscription;
                    userId = renewedInvoice.metadata.userId;
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
                // If this default is reached, the webhook specified in Stripe is too broad
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

import { i18nConfig, PaymentType } from "@local/shared";
import { ApolloServer } from "apollo-server-express";
import cookieParser from "cookie-parser";
import cors from "cors";
import express from "express";
import fs from "fs";
import { graphqlUploadExpress } from "graphql-upload";
import i18next from "i18next";
import Stripe from "stripe";
import * as auth from "./auth/request";
import { schema } from "./endpoints";
import apiRest from "./endpoints/rest/api";
import { logger } from "./events/logger";
import { context, depthLimit } from "./middleware";
import { initializeRedis } from "./redisConn";
import { initCountsCronJobs, initEventsCronJobs, initExpirePremiumCronJob, initGenerateEmbeddingsCronJob, initModerationCronJobs, initSitemapCronJob, initStatsCronJobs } from "./schedules";
import { setupDatabase } from "./utils/setupDatabase";
import { withPrisma } from "./utils/withPrisma";

const debug = process.env.NODE_ENV === "development";

const SERVER_URL = process.env.VITE_SERVER_LOCATION === "local" ?
    "http://localhost:5329/api" :
    process.env.SERVER_URL ?
        process.env.SERVER_URL :
        `http://${process.env.SITE_IP}:5329/api`;

const main = async () => {
    logger.info("Starting server...");

    // Check for required .env variables
    const requiredEnvs = ["PROJECT_DIR", "VITE_SERVER_LOCATION", "LETSENCRYPT_EMAIL", "VAPID_PUBLIC_KEY", "VAPID_PRIVATE_KEY"];
    for (const env of requiredEnvs) {
        if (!process.env[env]) {
            logger.error(`🚨 ${env} not in environment variables. Stopping server`, { trace: "0007" });
            process.exit(1);
        }
    }

    // Check for JWT public/private key files
    const requiredKeyFiles = ["jwt_priv.pem", "jwt_pub.pem"];
    for (const keyFile of requiredKeyFiles) {
        try {
            const key = fs.readFileSync(`${process.env.PROJECT_DIR}/${keyFile}`);
            if (!key) {
                logger.error(`🚨 ${keyFile} not found. Stopping server`, { trace: "0448" });
                process.exit(1);
            }
        } catch (error) {
            logger.error(`🚨 ${keyFile} not found. Stopping server`, { trace: "0449", error });
            process.exit(1);
        }
    }

    // Initialize translations
    await i18next.init(i18nConfig(debug));

    // Setup databases
    // Prisma
    await setupDatabase();
    // Redis 
    try {
        await initializeRedis();
        logger.info("✅ Connected to Redis");
    } catch (error) {
        logger.error("🚨 Failed to connect to Redis", { trace: "0207", error });
    }

    const app = express();

    // // For parsing application/xwww-
    // app.use(express.urlencoded({ extended: false }));
    // For parsing cookies
    app.use(cookieParser());

    // For authentication
    app.use(auth.authenticate);

    // Cross-Origin access. Accepts requests from localhost and dns
    // Disable if API is open to the public
    // API is open, so allow all origins
    app.use(cors({
        credentials: true,
        origin: true, //safeOrigins(),
    }));

    // For parsing application/json. 
    app.use((req, res, next) => {
        // Exclude on stripe webhook endpoint
        if (req.originalUrl === "/webhook/stripe") {
            next();
        } else {
            express.json({ limit: "20mb" })(req, res, next);
        }
    });

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
    app.post("/api/create-checkout-session", async (req, res) => {
        // Get userId and variant from request body
        const userId: string = req.body.userId;
        const variant: "yearly" | "monthly" | "donation" = req.body.variant;
        // Determine price API ID based on variant. Select a product in the Stripe dashboard 
        // to find this information
        let price: string;
        let paymentType: PaymentType;
        if (variant === "yearly") {
            price = "price_1MrUzeJq1sLW02CVEFdKKQNu";
            paymentType = PaymentType.PremiumYearly;
        } else if (variant === "monthly") {
            price = "price_1MrTMEJq1sLW02CVHdm1U247";
            paymentType = PaymentType.PremiumMonthly;
        } else if (variant === "donation") {
            price = "price_1MrTMlJq1sLW02CVK3ILOa6w";
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
                        price,
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
    // Create webhook endpoint for Stripe
    const endpointSecret = "whsec_590a2c3d0442e852131c914226f2642b24724674ffbccc518ca9607e865da233";
    app.post("/webhook/stripe", express.raw({ type: "application/json" }), async (request, response) => {
        const sig: string | string[] = request.headers["stripe-signature"] || "";
        let event;
        try {
            // Parse event
            event = stripe.webhooks.constructEvent(request.body, sig, endpointSecret);
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
                                response.status(400).send("Payment not found");
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
                                response.status(400).send("Premium record not found");
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
                                response.status(400).send("Premium record not found");
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
                                response.status(400).send("Premium record not found");
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
            response.status(400).send(`Webhook Error: ${error.message}`);
            return;
        }
        // Return a 200 response to acknowledge receipt of the event
        response.send();
    });

    // Set static folders
    // app.use(`/api/images`, express.static(`${process.env.PROJECT_DIR}/data/images`));

    // Set up image uploading
    app.use("/api/v2", graphqlUploadExpress({ maxFileSize: 10000000, maxFiles: 100 }));

    // Set up REST API
    app.use("/api/v2", apiRest);
    // app.use("/api/v2/apis", (req, res) => {
    //     console.log("Test endpoint hit");
    //     res.status(200).send("Test endpoint");
    // });

    // Apollo server for latest API version
    const apollo_options_latest = new ApolloServer({
        cache: "bounded" as any,
        introspection: debug,
        schema,
        context: (c) => context(c), // Allows request and response to be included in the context
        validationRules: [depthLimit(13)], // Prevents DoS attack from arbitrarily-nested query
    });
    // Start server
    await apollo_options_latest.start();
    // Configure server with ExpressJS settings and path
    apollo_options_latest.applyMiddleware({
        app,
        path: "/api/v2/graphql",
        cors: false,
    });
    // Additional Apollo server that wraps the latest version. Uses previous endpoint, and transforms the schema
    // to be compatible with the latest version.
    // const apollo_options_previous = new ApolloServer({
    //     cache: 'bounded' as any,
    //     introspection: debug,
    //     schema: schema,
    //     context: (c) => context(c), // Allows request and response to be included in the context
    //     validationRules: [depthLimit(10)] // Prevents DoS attack from arbitrarily-nested query
    // });
    // // Start server
    // await apollo_options_previous.start();
    // // Configure server with ExpressJS settings and path
    // apollo_options_previous.applyMiddleware({
    //     app,
    //     path: `/api/v2`,
    //     cors: false
    // });

    // Start Express server
    app.listen(5329);

    // Start cron jobs
    initStatsCronJobs();
    initEventsCronJobs();
    initCountsCronJobs();
    initSitemapCronJob();
    initModerationCronJobs();
    initExpirePremiumCronJob();
    initGenerateEmbeddingsCronJob();

    logger.info(`🚀 Server running at ${SERVER_URL}`);
};

main();

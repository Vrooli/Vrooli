import { PaymentType } from "@local/consts";
import { i18nConfig } from "@local/translations";
import { PrismaClient } from "@prisma/client";
import { ApolloServer } from "apollo-server-express";
import cookieParser from "cookie-parser";
import cors from "cors";
import express from "express";
import fs from "fs";
import { graphqlUploadExpress } from "graphql-upload";
import i18next from "i18next";
import path from "path";
import Stripe from "stripe";
import * as auth from "./auth/request";
import { schema } from "./endpoints";
import { logger } from "./events/logger";
import { context, depthLimit } from "./middleware";
import { initializeRedis } from "./redisConn";
import render from "./renderer";
import { initSitemapCronJob } from "./schedules";
import { initCountsCronJobs } from "./schedules/counts";
import { initEventsCronJobs } from "./schedules/events";
import { initModerationCronJobs } from "./schedules/moderate";
import { initExpirePremiumCronJob } from "./schedules/premium/base";
import { initStatsCronJobs } from "./schedules/stats";
import { isBot } from "./utils";
import { setupDatabase } from "./utils/setupDatabase";
const debug = process.env.NODE_ENV === "development";
const SERVER_URL = process.env.VITE_SERVER_LOCATION === "local" ?
    "http://localhost:5329/api" :
    process.env.VITE_SERVER_URL ?
        process.env.VITE_SERVER_URL :
        `http://${process.env.VITE_SITE_IP}:5329/api`;
const main = async () => {
    logger.info("Starting server...");
    const requiredEnvs = ["PROJECT_DIR", "VITE_SERVER_LOCATION", "LETSENCRYPT_EMAIL", "VITE_VAPID_PUBLIC_KEY", "VAPID_PRIVATE_KEY"];
    for (const env of requiredEnvs) {
        console.log("checking env", env, process.env[env]);
        if (!process.env[env]) {
            console.log("oh nooooo", env, process.env);
            logger.error(`🚨 ${env} not in environment variables. Stopping server`, { trace: "0007" });
            process.exit(1);
        }
    }
    const requiredKeyFiles = ["jwt_priv.pem", "jwt_pub.pem"];
    for (const keyFile of requiredKeyFiles) {
        try {
            const key = fs.readFileSync(`${process.env.PROJECT_DIR}/${keyFile}`);
            if (!key) {
                logger.error(`🚨 ${keyFile} not found. Stopping server`, { trace: "0448" });
                process.exit(1);
            }
        }
        catch (error) {
            logger.error(`🚨 ${keyFile} not found. Stopping server`, { trace: "0449", error });
            process.exit(1);
        }
    }
    await i18next.init(i18nConfig(debug));
    await setupDatabase();
    try {
        await initializeRedis();
        logger.info("✅ Connected to Redis");
    }
    catch (error) {
        logger.error("🚨 Failed to connect to Redis", { trace: "0207", error });
    }
    const app = express();
    app.use(cookieParser());
    app.use(auth.authenticate);
    app.use(cors({
        credentials: true,
        origin: true,
    }));
    app.use((req, res, next) => {
        if (req.originalUrl === "/webhook/stripe") {
            next();
        }
        else {
            express.json({ limit: "20mb" })(req, res, next);
        }
    });
    const stripe = new Stripe(process.env.STRIPE_SECRET_KEY, {
        apiVersion: "2022-11-15",
        typescript: true,
        appInfo: {
            name: "Vrooli",
            url: "https://vrooli.com",
            version: "2.0.0",
        },
    });
    app.post("/api/create-checkout-session", async (req, res) => {
        const userId = req.body.userId;
        const variant = req.body.variant;
        let price;
        let paymentType;
        if (variant === "yearly") {
            price = "price_1MrUzeJq1sLW02CVEFdKKQNu";
            paymentType = PaymentType.PremiumYearly;
        }
        else if (variant === "monthly") {
            price = "price_1MrTMEJq1sLW02CVHdm1U247";
            paymentType = PaymentType.PremiumMonthly;
        }
        else if (variant === "donation") {
            price = "price_1MrTMlJq1sLW02CVK3ILOa6w";
            paymentType = PaymentType.Donation;
        }
        else {
            logger.error("Invalid variant", { trace: "0436", userId, variant });
            res.status(400).json({ error: "Invalid variant" });
            return;
        }
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
            const prisma = new PrismaClient();
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
            await prisma.$disconnect();
            res.json(session);
            return;
        }
        catch (error) {
            logger.error("Error creating checkout session", { trace: "0437", userId, variant, error });
            res.status(500).json({ error });
            return;
        }
    });
    const endpointSecret = "whsec_590a2c3d0442e852131c914226f2642b24724674ffbccc518ca9607e865da233";
    app.post("/webhook/stripe", express.raw({ type: "application/json" }), async (request, response) => {
        const sig = request.headers["stripe-signature"] || "";
        let event;
        try {
            event = stripe.webhooks.constructEvent(request.body, sig, endpointSecret);
            let userId;
            let prisma;
            let payments;
            let session;
            let checkoutId;
            switch (event.type) {
                case "checkout.session.completed":
                    session = event.data.object;
                    checkoutId = session.id;
                    userId = session.metadata.userId;
                    prisma = new PrismaClient();
                    payments = await prisma.payment.findMany({
                        where: {
                            checkoutId,
                            paymentMethod: "Stripe",
                            user: { id: userId },
                        },
                    });
                    if (payments.length === 0) {
                        logger.error("Payment not found. Someone is gonna be mad", { trace: "0439", checkoutId, userId });
                        break;
                    }
                    await prisma.payment.update({
                        where: { id: payments[0].id },
                        data: {
                            status: "Paid",
                        },
                    });
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
                        }
                        else {
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
                    break;
                case "checkout.session.expired":
                    session = event.data.object;
                    checkoutId = session.id;
                    userId = session.metadata.userId;
                    prisma = new PrismaClient();
                    payments = await prisma.payment.findMany({
                        where: {
                            checkoutId,
                            paymentMethod: "Stripe",
                            status: "Pending",
                            user: { id: userId },
                        },
                    });
                    if (payments.length === 0) {
                        break;
                    }
                    await prisma.payment.delete({
                        where: { id: payments[0].id },
                    });
                    break;
                case "invoice.payment_failed":
                    const invoice = event.data.object;
                    const subscriptionId = invoice.subscription;
                    userId = invoice.metadata.userId;
                    prisma = new PrismaClient();
                    payments = await prisma.payment.findMany({
                        where: {
                            user: { id: userId },
                            paymentMethod: "Stripe",
                        },
                    });
                    if (payments.length === 0) {
                        logger.error("Payment not found on invoice.payment_failed", { trace: "0443", userId });
                        response.status(400).send("Payment not found");
                        return;
                    }
                    await prisma.payment.update({
                        where: { id: payments[0].id },
                        data: {
                            status: "Failed",
                        },
                    });
                    await prisma.$disconnect();
                    logger.error("Payment failed", { trace: "0444", userId, subscriptionId });
                    response.status(400).send("Payment failed");
                    break;
                case "customer.subscription.updated":
                    const subscription = event.data.object;
                    userId = subscription.metadata.userId;
                    const newPaymentType = subscription.plan.interval === "month" ? PaymentType.PremiumMonthly : PaymentType.PremiumYearly;
                    prisma = new PrismaClient();
                    const premiumRecord = await prisma.premium.findFirst({
                        where: { user: { id: userId } },
                    });
                    if (premiumRecord) {
                        await prisma.premium.update({
                            where: { id: premiumRecord.id },
                            data: {
                                expiresAt: new Date(Date.now() + (newPaymentType === PaymentType.PremiumMonthly ? 30 : 365) * 24 * 60 * 60 * 1000).toISOString(),
                            },
                        });
                    }
                    await prisma.$disconnect();
                    break;
                case "customer.subscription.deleted":
                    userId = event.data.object.metadata.userId;
                    prisma = new PrismaClient();
                    const canceledPremiumRecord = await prisma.premium.findFirst({
                        where: { user: { id: userId } },
                    });
                    if (canceledPremiumRecord) {
                        await prisma.premium.update({
                            where: { id: canceledPremiumRecord.id },
                            data: {
                                isActive: false,
                            },
                        });
                    }
                    await prisma.$disconnect();
                    break;
                case "invoice.payment_succeeded":
                    const renewedInvoice = event.data.object;
                    const renewedSubscriptionId = renewedInvoice.subscription;
                    userId = renewedInvoice.metadata.userId;
                    prisma = new PrismaClient();
                    const renewedPremiumRecord = await prisma.premium.findFirst({
                        where: { user: { id: userId } },
                    });
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
                    await prisma.$disconnect();
                    break;
                default:
                    logger.warning("Unhandled Stripe event", { trace: "0438", event: event.type });
                    break;
            }
        }
        catch (error) {
            response.status(400).send(`Webhook Error: ${error.message}`);
            return;
        }
        response.send();
    });
    app.use("/assets", express.static(path.join(__dirname, "../../ui/src/assets")));
    app.use("/api/v2", graphqlUploadExpress({ maxFileSize: 10000000, maxFiles: 100 }));
    const apollo_options_latest = new ApolloServer({
        cache: "bounded",
        introspection: debug,
        schema,
        context: (c) => context(c),
        validationRules: [depthLimit(13)],
    });
    await apollo_options_latest.start();
    apollo_options_latest.applyMiddleware({
        app,
        path: "/api/v2",
        cors: false,
    });
    app.get("*", (req, res) => {
        const html = render(req);
        const isRequestFromBot = isBot(req.headers["user-agent"] || "");
        let openGraphTags = "";
        if (isRequestFromBot) {
            openGraphTags = `
            <meta property="og:title" content="Your Page Title" />
            <meta property="og:description" content="Your Page Description" />
            <meta property="og:image" content="Your Page Image URL" />
            <meta property="og:url" content="Your Page URL" />
          `;
        }
        res.send(`
          <!DOCTYPE html>
          <html lang="en">
            <head>
              <meta charset="UTF-8" />
              <meta name="viewport" content="width=device-width, initial-scale=1.0" />
              <title>Your App Title</title>
              ${openGraphTags}
            </head>
            <body>
              <div id="root">${html}</div>
              <script src="/@vite/client"></script>
              <script src="/src/main.tsx"></script>
            </body>
          </html>
        `);
    });
    app.listen(5329);
    initStatsCronJobs();
    initEventsCronJobs();
    initCountsCronJobs();
    initSitemapCronJob();
    initModerationCronJobs();
    initExpirePremiumCronJob();
    logger.info(`🚀 Server running at ${SERVER_URL}`);
};
main();
//# sourceMappingURL=index.js.map
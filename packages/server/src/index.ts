import { PrismaClient } from '@prisma/client';
import { PaymentType } from '@shared/consts';
import { i18nConfig } from '@shared/translations';
import { ApolloServer } from 'apollo-server-express';
import cookieParser from 'cookie-parser';
import cors from "cors";
import express from 'express';
import { graphqlUploadExpress } from 'graphql-upload';
import i18next from 'i18next';
import Stripe from 'stripe';
import * as auth from './auth/request';
import { schema } from './endpoints';
import { logger } from './events/logger';
import { context, depthLimit } from './middleware';
import { initializeRedis } from './redisConn';
import { initSitemapCronJob } from './schedules';
import { initCountsCronJobs } from './schedules/counts';
import { initEventsCronJobs } from './schedules/events';
import { initModerationCronJobs } from './schedules/moderate';
import { initExpirePremiumCronJob } from './schedules/premium/base';
import { initStatsCronJobs } from './schedules/stats';
import { setupDatabase } from './utils/setupDatabase';

const debug = process.env.NODE_ENV === 'development';

const SERVER_URL = process.env.VITE_SERVER_LOCATION === 'local' ?
    `http://localhost:5329/api` :
    Boolean(process.env.SERVER_URL) ?
        process.env.SERVER_URL :
        `http://${process.env.SITE_IP}:5329/api`;

const main = async () => {
    logger.info('Starting server...');

    // Check for required .env variables
    const requiredEnvs = ['VITE_SERVER_LOCATION', 'JWT_SECRET', 'LETSENCRYPT_EMAIL', 'VAPID_PUBLIC_KEY', 'VAPID_PRIVATE_KEY'];
    for (const env of requiredEnvs) {
        if (!process.env[env]) {
            console.error('uh oh', process.env[env]);
            logger.error(`🚨 ${env} not in environment variables. Stopping server`, { trace: '0007' });
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
        logger.info('✅ Connected to Redis');
    } catch (error) {
        logger.error('🚨 Failed to connect to Redis', { trace: '0207', error });
    }

    const app = express();

    // // For parsing application/xwww-
    // app.use(express.urlencoded({ extended: false }));
    // For parsing cookies
    app.use(cookieParser(process.env.JWT_SECRET));

    // For authentication
    app.use(auth.authenticate);

    // Cross-Origin access. Accepts requests from localhost and dns
    // Disable if API is open to the public
    // API is open, so allow all origins
    app.use(cors({
        credentials: true,
        origin: true, //safeOrigins(),
    }))

    // For parsing application/json. 
    app.use((req, res, next) => {
        console.log('in app.use bodyparser', req.originalUrl);
        // Exclude on stripe webhook endpoint
        if (req.originalUrl === '/webhook/stripe') {
            next();
        } else {
            express.json({ limit: '20mb' })(req, res, next);
        }
    });

    // Set up Stripe
    const stripe = new Stripe(process.env.STRIPE_SECRET_KEY!, {
        apiVersion: '2022-11-15',
        typescript: true,
        appInfo: {
            name: 'Vrooli',
            url: 'https://vrooli.com',
            version: '2.0.0',
        },
    });
    // Create endpoint for buying a premium subscription or donating
    app.post('/api/create-checkout-session', async (req, res) => {
        console.log('create-checkout-session 1')
        // Get userId and variant from request body
        const userId: string = req.body.userId;
        const variant: 'yearly' | 'monthly' | 'donation' = req.body.variant;
        console.log('create-checkout-session 2', userId, variant)
        // Determine price API ID based on variant. Select a product in the Stripe dashboard 
        // to find this information
        let price: string;
        let paymentType: PaymentType;
        if (variant === 'yearly') {
            price = 'price_1MrUzeJq1sLW02CVEFdKKQNu';
            paymentType = PaymentType.PremiumYearly;
        } else if (variant === 'monthly') {
            price = 'price_1MrTMEJq1sLW02CVHdm1U247';
            paymentType = PaymentType.PremiumMonthly;
        } else if (variant === 'donation') {
            price = 'price_1MrTMlJq1sLW02CVK3ILOa6w';
            paymentType = PaymentType.Donation;
        } else {
            logger.error('Invalid variant', { trace: '0436', userId, variant })
            res.status(400).json({ error: 'Invalid variant' });
            return;
        }
        // Create checkout session
        try {
            const session = await stripe.checkout.sessions.create({
                success_url: 'https://vrooli.com/premium?status=success',
                cancel_url: 'https://vrooli.com/premium?status=canceled',
                payment_method_types: ['card'],
                line_items: [
                    {
                        price,
                        quantity: 1,
                    },
                ],
                mode: variant === 'donation' ? 'payment' : 'subscription',
                metadata: { userId },
            });
            console.log('create-checkout-session 3', JSON.stringify(session), '\n\n');
            // Create open payment in database, so we can track it
            // TODO create cron job to clean up old payments that never completed, and to remove premium status from users when expired
            const prisma = new PrismaClient();
            await prisma.payment.create({
                data: {
                    amount: (variant === 'donation' ? session.amount_total : session.amount_subtotal) ?? 0,
                    checkoutId: session.id,
                    currency: session.currency ?? 'usd',
                    description: variant === 'donation' ? 'Donation' : 'Premium subscription - ' + variant,
                    paymentMethod: 'Stripe',
                    paymentType,
                    status: 'Pending',
                    user: { connect: { id: userId } },
                }
            });
            console.log('create-checkout-session 4', session.id)
            await prisma.$disconnect();
            // Send session ID as response
            res.json(session);
            return;
        } catch (error) {
            logger.error('Error creating checkout session', { trace: '0437', userId, variant, error })
            res.status(500).json({ error });
            return;
        }
    });
    // Create webhook endpoint for Stripe
    const endpointSecret = "whsec_590a2c3d0442e852131c914226f2642b24724674ffbccc518ca9607e865da233";
    app.post('/webhook/stripe', express.raw({ type: 'application/json' }), async (request, response) => {
        const sig: string | string[] = request.headers['stripe-signature'] || '';
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
                case 'checkout.session.completed':
                    // Find payment in database
                    session = event.data.object;
                    checkoutId = session.id;
                    userId = session.metadata.userId;
                    prisma = new PrismaClient();
                    payments = await prisma.payment.findMany({
                        where: {
                            checkoutId,
                            paymentMethod: 'Stripe',
                            user: { id: userId },
                        }
                    })
                    // If not found, log error
                    if (payments.length === 0) {
                        logger.error('Payment not found. Someone is gonna be mad', { trace: '0439', checkoutId, userId });
                        break;
                    }
                    // Update payment in database to indicate it was paid
                    await prisma.payment.update({
                        where: { id: payments[0].id },
                        data: {
                            status: 'Paid',
                        }
                    })
                    // If subscription, upsert premium status
                    if (payments[0].paymentType === PaymentType.PremiumMonthly || payments[0].paymentType === PaymentType.PremiumYearly) {
                        const enabledAt = new Date().toISOString();
                        const expiresAt = new Date(Date.now() + (payments[0].paymentType === PaymentType.PremiumMonthly ? 30 : 365) * 24 * 60 * 60 * 1000).toISOString();
                        const isActive = true;
                        const premiums = await prisma.premium.findMany({
                            where: { user: { id: userId } }
                        })
                        if (premiums.length === 0) {
                            await prisma.premium.create({
                                data: {
                                    enabledAt,
                                    expiresAt,
                                    isActive,
                                    user: { connect: { id: userId } },
                                }
                            })
                        } else {
                            await prisma.premium.update({
                                where: { id: premiums[0].id },
                                data: {
                                    enabledAt: premiums[0].enabledAt ?? enabledAt,
                                    expiresAt,
                                    isActive,
                                }
                            })
                        }
                    }
                    break;
                // Session expired before payment
                case 'checkout.session.expired':
                    // Find payment in database
                    session = event.data.object;
                    checkoutId = session.id;
                    userId = session.metadata.userId;
                    prisma = new PrismaClient();
                    payments = await prisma.payment.findMany({
                        where: {
                            checkoutId,
                            paymentMethod: 'Stripe',
                            status: 'Pending',
                            user: { id: userId },
                        }
                    })
                    // If not found, ignore
                    if (payments.length === 0) {
                        break;
                    }
                    // Delete payment in database
                    await prisma.payment.delete({
                        where: { id: payments[0].id }
                    })
                    break;
                // Payment failed
                case 'invoice.payment_failed':
                    const invoice = event.data.object;
                    const subscriptionId = invoice.subscription;
                    userId = invoice.metadata.userId;
                    // Find the payment in the database
                    prisma = new PrismaClient();
                    payments = await prisma.payment.findMany({
                        where: {
                            user: { id: userId },
                            paymentMethod: 'Stripe',
                        },
                    });
                    // If not found, log an error and return
                    if (payments.length === 0) {
                        logger.error('Payment not found on invoice.payment_failed', { trace: '0443', userId });
                        response.status(400).send('Payment not found');
                        return;
                    }
                    // Update the payment status to "Failed" in the database
                    await prisma.payment.update({
                        where: { id: payments[0].id },
                        data: {
                            status: 'Failed',
                        },
                    });
                    // Disconnect the Prisma client
                    await prisma.$disconnect();
                    // Log the error and return a response
                    logger.error('Payment failed', { trace: '0444', userId, subscriptionId });
                    response.status(400).send('Payment failed');
                    break;
                // Subscription changed (monthly -> yearly, yearly -> monthly)
                case 'customer.subscription.updated':
                    const subscription = event.data.object;
                    userId = subscription.metadata.userId;
                    const newPaymentType = subscription.plan.interval === 'month' ? PaymentType.PremiumMonthly : PaymentType.PremiumYearly;

                    prisma = new PrismaClient();
                    const premiumRecord = await prisma.premium.findFirst({
                        where: { user: { id: userId } }
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

                // Subscription canceled
                case 'customer.subscription.deleted':
                    userId = event.data.object.metadata.userId;

                    prisma = new PrismaClient();
                    const canceledPremiumRecord = await prisma.premium.findFirst({
                        where: { user: { id: userId } }
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

                // Subscription renewed
                case 'invoice.payment_succeeded':
                    const renewedInvoice = event.data.object;
                    const renewedSubscriptionId = renewedInvoice.subscription;
                    userId = renewedInvoice.metadata.userId;

                    prisma = new PrismaClient();
                    const renewedPremiumRecord = await prisma.premium.findFirst({
                        where: { user: { id: userId } }
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
                // If this default is reached, the webhook specified in Stripe is too broad
                default:
                    logger.warning('Unhandled Stripe event', { trace: '0438', event: event.type });
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
    app.use(`/api/v2`, graphqlUploadExpress({ maxFileSize: 10000000, maxFiles: 100 }))

    // Apollo server for latest API version
    const apollo_options_latest = new ApolloServer({
        cache: 'bounded' as any,
        introspection: debug,
        schema: schema,
        context: (c) => context(c), // Allows request and response to be included in the context
        validationRules: [depthLimit(13)] // Prevents DoS attack from arbitrarily-nested query
    });
    // Start server
    await apollo_options_latest.start();
    // Configure server with ExpressJS settings and path
    apollo_options_latest.applyMiddleware({
        app,
        path: `/api/v2`,
        cors: false
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

    logger.info(`🚀 Server running at ${SERVER_URL}`);
}

main();
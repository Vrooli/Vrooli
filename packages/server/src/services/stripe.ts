import { API_CREDITS_MULTIPLIER, API_CREDITS_PREMIUM, CheckCreditsPaymentParams, CheckCreditsPaymentResponse, CheckSubscriptionParams, CheckSubscriptionResponse, CheckoutSessionMetadata, CreateCheckoutSessionParams, CreateCheckoutSessionResponse, CreatePortalSessionParams, DAYS_180_MS, HttpStatus, LINKS, PaymentStatus, PaymentType, StripeEndpoint, SubscriptionPricesResponse } from "@local/shared";
import { PrismaPromise } from "@prisma/client";
import express, { Express, Request, Response } from "express";
import Stripe from "stripe";
import { prismaInstance } from "../db/instance";
import { logger } from "../events/logger";
import { withRedis } from "../redisConn";
import { UI_URL } from "../server";
import { emitSocketEvent } from "../sockets/events";
import { sendCreditCardExpiringSoon, sendPaymentFailed, sendPaymentThankYou, sendSubscriptionCanceled, sendTrialEndingSoon } from "../tasks/email/queue";

type EventHandlerArgs = {
    event: Stripe.Event,
    stripe: Stripe,
    res: Response,
}

type HandlerResult = {
    status: HttpStatus;
    message?: string;
}

type Handler = (args: EventHandlerArgs) => Promise<HandlerResult>;

// Make sure these are the price IDs when you view the Stripe dashboard in test mode
export const PRICE_IDS_DEV: Record<PaymentType, string> = {
    Credits: "price_1OyietJq1sLW02CVCUlV78AN",
    Donation: "price_1NG6LnJq1sLW02CV1bhcCYZG",
    PremiumMonthly: "price_1NYaGQJq1sLW02CVFAO6bPu4",
    PremiumYearly: "price_1MrUzeJq1sLW02CVEFdKKQNu",
};

// Make sure these are the price IDs when you view the Stripe dashboard in live mode
export const PRICE_IDS_PROD: Record<PaymentType, string> = {
    Credits: "price_1OygWtJq1sLW02CVK43shmp2",
    Donation: "price_1Oyg5rJq1sLW02CVtmlSyjtX",
    PremiumMonthly: "price_1OygGCJq1sLW02CVyRVlw7xg",
    PremiumYearly: "price_1OygGCJq1sLW02CVDVmTfWDA",
};

export enum PaymentMethod {
    Stripe = "Stripe",
}

/** 
 * Finds all available price IDs for the current environment (development or production)
 * NOTE: Make sure these are PRICE IDs, not PRODUCT IDs
 **/
export const getPriceIds = (): Record<PaymentType, string> => {
    return process.env.NODE_ENV === "production" ? PRICE_IDS_PROD : PRICE_IDS_DEV;
};

/** 
 * Finds the payment type of a Stripe price 
 * 
 * @throws Error if the price ID is invalid
 */
export const getPaymentType = (price: string | Stripe.Price | Stripe.DeletedPrice): PaymentType => {
    const priceId = typeof price === "string" ? price : price.id;
    const prices = getPriceIds();
    const matchingPrice = Object.entries(prices).find(([_key, value]) => value === priceId);
    if (!matchingPrice) {
        throw new Error("Invalid price ID");
    }
    return matchingPrice[0] as PaymentType;
};

/**
 * Returns the customer ID from the Stripe API response.
 * @param customer The customer field in a Stripe event.
 * This could be a string (representing the customer ID), a Stripe.Customer object, a Stripe.DeletedCustomer object, or null.
 * @returns The customer ID as a string if it exists
 * @throws hrows an error if the customer ID doesn't exist or couldn't be found.
 */
export const getCustomerId = (customer: string | Stripe.Customer | Stripe.DeletedCustomer | null | undefined): string => {
    if (typeof customer === "string") {
        return customer;
    } else if (customer && typeof customer === "object" && Object.prototype.hasOwnProperty.call(customer, "id") && typeof customer.id === "string") {
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
export const handlerResult = (
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

/**
 * Stores a price in redis for a given payment type
 */
export const storePrice = async (paymentType: PaymentType, price: number): Promise<void> => {
    await withRedis({
        process: async (redisClient) => {
            const key = `stripe-payment-${process.env.NODE_ENV}-${paymentType}`;
            await redisClient.set(key, price);
            await redisClient.expire(key, 60 * 60 * 24); // expire after 24 hours
        },
        trace: "0517",
    });
};

/**
 * Fetches a price from redis for a given payment type, 
 * or null if the price is not found
 */
export const fetchPriceFromRedis = async (paymentType: PaymentType): Promise<number | null> => {
    let result: number | null = null;
    await withRedis({
        process: async (redisClient) => {
            const key = `stripe-payment-${process.env.NODE_ENV}-${paymentType}`;
            const price = await redisClient.get(key);
            if (Number.isInteger(price) && Number(price) >= 0) {
                result = Number(price);
            }
        },
        trace: "0185",
    });
    return result;
};

/**
 * @returns True if the Stripe object was created in the environment matching the current environment
 */
export function isInCorrectEnvironment(object: { livemode: boolean }): boolean {
    if (process.env.NODE_ENV === "production") {
        return object.livemode;
    }
    return !object.livemode;
}

/** @returns True if the Stripe session should be counted as rewarding a subscription */
export const isValidSubscriptionSession = (session: Stripe.Checkout.Session, userId: string): boolean => {
    const { paymentType, userId: sessionUserId } = session.metadata as CheckoutSessionMetadata;
    return [PaymentType.PremiumMonthly, PaymentType.PremiumYearly].includes(paymentType) // Is a payment type we recognize
        && sessionUserId === userId // Was initiated by this user
        && session.status === "complete" // Was completed
        && isInCorrectEnvironment(session);
};

/** @returns True if the Stripe session should be counted as rewarding credits */
export const isValidCreditsPurchaseSession = (session: Stripe.Checkout.Session, userId: string): boolean => {
    const { paymentType, userId: sessionUserId } = session.metadata as CheckoutSessionMetadata;
    return paymentType === PaymentType.Credits // Is a credit purchase
        && sessionUserId === userId // Was initiated by this user
        && session.status === "complete" // Was completed
        && isInCorrectEnvironment(session); // Is in the correct environment (optional, depending on your setup)
};

type GetVerifiedSubscriptionInfoResult = {
    session: Stripe.Checkout.Session,
    subscription: Stripe.Subscription,
    paymentType: PaymentType.PremiumMonthly | PaymentType.PremiumYearly,
};

/**
 * @param stripe The Stripe object for interacting with the Stripe API
 * @param customerId The Stripe customer ID
 * @param userId The user ID linked to the customer ID
 * @returns Verified subscription information, including the session and payment type, 
 * or null if the subscription is not found or is invalid
 */
export const getVerifiedSubscriptionInfo = async (
    stripe: Stripe,
    customerId: string,
    userId: string,
): Promise<GetVerifiedSubscriptionInfoResult | null> => {
    // Retrieve subscriptions for the customer, and filter out inactive subscriptions
    const subscriptions = (await stripe.subscriptions.list({
        customer: customerId,
        status: "all",
    })).data.filter(sub => ["active", "trialing"].includes(sub.status) && isInCorrectEnvironment(sub));

    // We could stop here, but we want to make sure that the subscription was initiated 
    // by this user (and they didn't switch emails to duplicate their subscription).
    // To do this, we'll check the session metadata associated with the subscription
    for (const subscription of subscriptions) {
        // Find sessions associated with the subscription. 
        // There should only be 1, but you never know
        const sessions = await stripe.checkout.sessions.list({
            subscription: subscription.id,
            limit: 5,
        });

        // If one of the sessions has metadata that matches the user ID and a 
        // subscription tier we recognize, we can consider this the verified subscription
        const verifiedSession = sessions.data.find(session => isValidSubscriptionSession(session, userId));
        if (verifiedSession) {
            // As a last check, we'll make sure that the payment type is one we recognize
            const { paymentType } = verifiedSession.metadata as CheckoutSessionMetadata;
            const validPaymentTypes = [PaymentType.PremiumMonthly, PaymentType.PremiumYearly];
            if (validPaymentTypes.includes(paymentType)) {
                return {
                    session: verifiedSession,
                    subscription,
                    paymentType: paymentType as PaymentType.PremiumMonthly | PaymentType.PremiumYearly,
                };
            }
        }
    }

    // If we didn't find a verified subscription, return null
    return null;
};

type GetStripeCustomerIdResult = {
    emails: { emailAddress: string }[],
    hasPremium: boolean,
    /** Only returned if `validateSubscription` is true and we find a subscription */
    subscriptionInfo: GetVerifiedSubscriptionInfoResult | null,
    stripeCustomerId: string | null,
    userId: string | null,
};

/** 
 * @param stripe The Stripe object for interacting with the Stripe API
 * @param userId The user ID to find the Stripe customer ID for
 * @param validateSubscription If true, will skip customer IDs that are not associated with a valid subscription. 
 * This is useful if someone used stripe before with one email, then subscribed with another.
 * @returns Verified customer information, including the stripeCustomerId, emails, and userId
 */
export async function getVerifiedCustomerInfo({
    userId,
    stripe,
    validateSubscription,
}: {
    userId: string | undefined,
    stripe: Stripe,
    validateSubscription: boolean,
}): Promise<GetStripeCustomerIdResult> {
    const result: GetStripeCustomerIdResult = {
        emails: [],
        hasPremium: false,
        subscriptionInfo: null,
        stripeCustomerId: null,
        userId: null,
    };
    if (!userId) {
        return result;
    }
    // Find the user in the database
    const user = await prismaInstance.user.findUnique({
        where: { id: userId },
        select: {
            id: true,
            emails: { select: { emailAddress: true } },
            premium: { select: { isActive: true } },
            stripeCustomerId: true,
        },
    });
    if (!user) {
        logger.error("User not found.", { trace: "0519", userId });
        return result;
    }
    result.emails = user.emails || [];
    result.hasPremium = user.premium?.isActive ?? false;
    result.stripeCustomerId = user.stripeCustomerId || null;
    result.userId = user.id;
    // Validate the stripeCustomerId by checking if it exists in Stripe
    if (result.stripeCustomerId) {
        try {
            const customer = await stripe.customers.retrieve(result.stripeCustomerId);
            // If the customer was deleted, set stripeCustomerId to null
            if (customer.deleted) {
                result.stripeCustomerId = null;
                logger.error("Stripe customer was deleted", { trace: "0522", result });
            }
            // Validate subscription if requested
            if (validateSubscription) {
                const subscriptionInfo = await getVerifiedSubscriptionInfo(stripe, result.stripeCustomerId as string, userId);
                result.subscriptionInfo = subscriptionInfo;
                if (!subscriptionInfo) {
                    result.stripeCustomerId = null;
                    logger.info("Stripe customer ID is not associated with an active subscription", { trace: "0518", result });
                }
            }
        } catch (error) {
            // If the customer doesn't exist, set stripeCustomerId to null
            result.stripeCustomerId = null;
            logger.error("Invalid Stripe customer ID", { trace: "0521", result, error });
        }
    }
    // If the user does not have a Stripe customer ID,
    // we can check if any of their emails are associated with a Stripe customer ID
    if (!result.stripeCustomerId) {
        for (const email of result.emails) {
            // Find active stripe customers associated with the email
            const customers = (await stripe.customers.list({
                // NOTE: This is case-sensitive. If the user's Stripe email is not what 
                // we have in the database exactly, this will not work. But there's 
                // not much we can do about that.
                email: email.emailAddress,
                limit: 1,
            })).data.filter(customer => !customer.deleted);
            if (!customers.length) continue;
            result.stripeCustomerId = customers[0].id || null;
            if (!result.stripeCustomerId) continue;
            // Validate subscription if requested
            if (validateSubscription) {
                const subscriptionInfo = await getVerifiedSubscriptionInfo(stripe, result.stripeCustomerId, userId);
                result.subscriptionInfo = subscriptionInfo;
                if (!subscriptionInfo) {
                    result.stripeCustomerId = null;
                    logger.info("Stripe customer ID is not associated with an active subscription", { trace: "0516", result });
                    continue;
                }
            }
            // Update the user's stripeCustomerId
            await prismaInstance.user.update({
                where: { id: userId },
                data: { stripeCustomerId: result.stripeCustomerId },
            });
            break;
        }
    }
    return result;
}

/**
 * Creates and sets a Stripe customer ID for a user
 * @returns The newly-created Stripe customer ID
 */
export async function createStripeCustomerId({
    customerInfo,
    requireUserToExist,
    stripe,
}: {
    customerInfo: GetStripeCustomerIdResult,
    requireUserToExist: boolean,
    stripe: Stripe,
}): Promise<string> {
    // Throw error if there should be a user but there isn't
    if (requireUserToExist && !customerInfo.userId) {
        throw new Error("User not found.");
    }
    // Create a new Stripe customer
    const stripeCustomer = await stripe.customers.create({
        email: customerInfo.emails.length ? customerInfo.emails[0].emailAddress : undefined,
    });
    // Store Stripe customer ID in your database
    if (customerInfo.userId) {
        await prismaInstance.user.update({
            where: { id: customerInfo.userId },
            data: { stripeCustomerId: stripeCustomer.id },
        });
    }
    return stripeCustomer.id;
}

/** 
 * Processes a completed payment.
 * @param stripe The Stripe object for interacting with the Stripe API
 * @param session The Stripe checkout session object
 * @param subscription The Stripe subscription object linked to the session, if known
 * @throws Error if something goes wrong, such as if the user is not found
 */
export const processPayment = async (
    stripe: Stripe,
    session: Stripe.Checkout.Session,
    subscription?: Stripe.Subscription,
): Promise<void> => {
    const checkoutId = session.id;
    const customerId = getCustomerId(session.customer);
    const { paymentType, userId } = session.metadata as CheckoutSessionMetadata;
    // Skip if payment already processed
    const payments = await prismaInstance.payment.findMany({
        where: {
            checkoutId,
            paymentMethod: PaymentMethod.Stripe,
            user: { stripeCustomerId: customerId },
        },
    });
    if (payments.length > 0 && !payments.some(payment => payment.status === "Pending")) {
        return;
    }
    // Upsert paid payment
    const data = {
        amount: session.amount_subtotal ?? 0, // Use subtotal because total includes tax, discounts, etc.
        checkoutId,
        currency: session.currency ?? "usd",
        description: "Checkout: " + paymentType,
        paymentMethod: PaymentMethod.Stripe,
        paymentType,
        status: PaymentStatus.Paid,
    } as const;
    const paymentId = payments.length > 0 ? payments[0].id : undefined;
    const paymentSelect = {
        amount: true,
        paymentType: true,
        user: {
            select: {
                id: true,
                emails: { select: { emailAddress: true } },
            },
        },
    } as const;
    const payment = paymentId ? await prismaInstance.payment.update({
        where: { id: paymentId },
        data,
        select: paymentSelect,
    }) : await prismaInstance.payment.create({
        data: {
            ...data,
            user: userId ? { connect: { id: userId } } : undefined,
        },
        select: paymentSelect,
    });
    // If there is not a user associated with this payment, throw an error
    if (!payment.user) {
        throw new Error("User not found.");
    }
    // If credits, award user the relevant number of credits
    if (payment.paymentType === PaymentType.Credits) {
        const creditsToAward = (BigInt(payment.amount) * API_CREDITS_MULTIPLIER);
        await prismaInstance.user.update({
            where: { id: payment.user.id },
            data: {
                premium: {
                    upsert: {
                        create: {
                            credits: creditsToAward,
                        },
                        update: {
                            credits: { increment: creditsToAward },
                        },
                    },
                },
            },
        });
    }
    // If subscription, upsert premium status
    else if (payment.paymentType === PaymentType.PremiumMonthly || payment.paymentType === PaymentType.PremiumYearly) {
        // Get subscription
        if (!isValidSubscriptionSession(session, payment.user.id)) {
            throw new Error("Invalid subscription session");
        }
        const knownSubscription = subscription
            ? subscription
            : typeof session.subscription === "string"
                ? await stripe.subscriptions.retrieve(session.subscription as string)
                : session.subscription;
        if (!knownSubscription) {
            throw new Error("Subscription not found.");
        }
        // Find enabledAt and expiresAt
        const enabledAt = new Date().toISOString();
        const expiresAt = new Date(knownSubscription.current_period_end * 1000).toISOString();
        // Upsert premium status
        const premiums = await prismaInstance.premium.findMany({
            where: { user: { id: payment.user.id } }, // User should exist based on findMany query above
        });
        if (premiums.length === 0) {
            await prismaInstance.premium.create({
                data: {
                    enabledAt,
                    expiresAt,
                    isActive: true,
                    credits: API_CREDITS_PREMIUM,
                    user: { connect: { id: payment.user.id } }, // User should exist based on findMany query above
                },
            });
        } else {
            await prismaInstance.premium.update({
                where: { id: premiums[0].id },
                data: {
                    enabledAt: premiums[0].enabledAt ?? enabledAt,
                    expiresAt,
                    isActive: true,
                    credits: API_CREDITS_PREMIUM,
                },
            });
        }
        emitSocketEvent("apiCredits", payment.user.id, { credits: API_CREDITS_PREMIUM + "" });
    }
    // Send thank you notification
    for (const email of payment.user.emails) {
        sendPaymentThankYou(email.emailAddress, [PaymentType.PremiumMonthly, PaymentType.PremiumYearly].includes(payment.paymentType as PaymentType));
    }
};

/** Checkout completed for donation or subscription */
const handleCheckoutSessionCompleted = async ({ event, stripe, res }: EventHandlerArgs): Promise<HandlerResult> => {
    try {
        await processPayment(stripe, event.data.object as Stripe.Checkout.Session);
        return handlerResult(HttpStatus.Ok, res);
    } catch (error) {
        return handlerResult(HttpStatus.InternalServerError, res, "Caught error in handleCheckoutSessionCompleted", "0440", error);
    }
};

/** Checkout expired before payment */
export const handleCheckoutSessionExpired = async ({ event, res }: EventHandlerArgs): Promise<HandlerResult> => {
    const session = event.data.object as Stripe.Checkout.Session;
    const checkoutId = session.id;
    const customerId = getCustomerId(session.customer);
    // Find payment in database
    const payments = await prismaInstance.payment.findMany({
        where: {
            checkoutId,
            paymentMethod: PaymentMethod.Stripe,
            status: PaymentStatus.Pending,
            user: { stripeCustomerId: customerId },
        },
    });
    if (payments.length === 0) {
        // If we never created a payment for this, do nothing
        return handlerResult(HttpStatus.Ok, res);
    }
    // Mark payments as failed
    for (const payment of payments) {
        await prismaInstance.payment.update({
            where: { id: payment.id },
            data: { status: PaymentStatus.Failed },
        });
    }
    return handlerResult(HttpStatus.Ok, res);
};

/** Customer was deleted */
export const handleCustomerDeleted = async ({ event, res }: EventHandlerArgs): Promise<HandlerResult> => {
    const customer = event.data.object as Stripe.Customer;
    const customerId = getCustomerId(customer);
    // Attempt to find the user associated with the given customerId
    const user = await prismaInstance.user.findUnique({
        where: { stripeCustomerId: customerId },
    });
    // If the user exists, remove the customerId
    if (user) {
        await prismaInstance.user.update({
            where: { stripeCustomerId: customerId },
            data: { stripeCustomerId: null },
        });
    }
    // Return an OK status regardless of whether the user was found or not
    return handlerResult(HttpStatus.Ok, res);
};

/** Customer's credit card is about to expire */
export const handleCustomerSourceExpiring = async ({ event, res }: Omit<EventHandlerArgs, "stripe">): Promise<HandlerResult> => {
    const customer = event.data.object as Stripe.Customer;
    const customerId = getCustomerId(customer);
    // Find user with the given Stripe customer ID
    const user = await prismaInstance.user.findFirst({
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
export const handleCustomerSubscriptionDeleted = async ({ event, res }: Omit<EventHandlerArgs, "stripe">): Promise<HandlerResult> => {
    const subscription = event.data.object as Stripe.Subscription;
    const customerId = getCustomerId(subscription.customer);
    // Find user 
    const user = await prismaInstance.user.findFirst({
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
export const handleCustomerSubscriptionTrialWillEnd = async ({ event, res }: Omit<EventHandlerArgs, "stripe">): Promise<HandlerResult> => {
    const subscription = event.data.object as Stripe.Subscription;
    const customerId = getCustomerId(subscription.customer);
    // Find user with the given Stripe customer ID
    const user = await prismaInstance.user.findFirst({
        where: { stripeCustomerId: customerId },
        select: {
            emails: { select: { emailAddress: true } },
        },
    });
    if (!user) {
        return handlerResult(HttpStatus.InternalServerError, res, "User not found.", "0192", { customerId });
    }
    // If found, send notification
    for (const email of user.emails) {
        sendTrialEndingSoon(email.emailAddress);
    }
    return handlerResult(HttpStatus.Ok, res);
};

/**
 * Calculates the expiration date and active status of a subscription.
 * 
 * @param currentPeriodEnd - The end of the current billing cycle as a Unix timestamp in seconds.
 * @returns The new expiration date and active status.
 */
export const calculateExpiryAndStatus = (currentPeriodEnd: number) => {
    const newExpiresAt = new Date(currentPeriodEnd * 1000); // Convert seconds to milliseconds
    const now = new Date();
    const isActive = newExpiresAt > now;
    return { newExpiresAt, isActive };
};

/** 
 * Occurs whenever a subscription changes (e.g., switching from monthly 
 * to yearly, or changing the status from trial to active). 
 */
export const handleCustomerSubscriptionUpdated = async ({ event, res }: Omit<EventHandlerArgs, "stripe">): Promise<HandlerResult> => {
    const subscription = event.data.object as Stripe.Subscription;
    const customerId = getCustomerId(subscription.customer);

    // Skip processing if the subscription is not marked as active. 
    // Canceled subscriptions are handled by the customer.subscription.deleted event.
    if (subscription.status !== "active") {
        // This isn't a problem with the request itself, so we'll return a 200 status code
        return handlerResult(HttpStatus.Ok, res, "Subscription is not active.");
    }

    const user = await prismaInstance.user.findFirst({
        where: { stripeCustomerId: customerId },
        select: {
            id: true,
            emails: {
                select: {
                    emailAddress: true,
                },
            },
        },
    });
    if (!user) {
        return handlerResult(HttpStatus.InternalServerError, res, "User not found on customer.subscription.updated", "0457", { customerId });
    }

    // Calculate new expiration date and active status. 
    // Should always be active, unless the new expiry date is somehow in the past
    const { newExpiresAt, isActive } = calculateExpiryAndStatus(subscription.current_period_end);
    // Note that we're not updating the user's credits here, as that would only happen when 
    // activating a subscription. That's handled by the checkout.session.completed event.
    await prismaInstance.user.update({
        where: { id: user.id },
        data: {
            premium: {
                upsert: {
                    create: {
                        enabledAt: isActive ? new Date() : undefined,
                        expiresAt: newExpiresAt,
                        isActive,
                    },
                    update: {
                        expiresAt: newExpiresAt,
                        isActive,
                    },
                },
            },
        },
    });

    if (isActive) {
        for (const email of user.emails) {
            sendPaymentThankYou(email.emailAddress, false);
        }
    } else {
        for (const email of user.emails) {
            sendSubscriptionCanceled(email.emailAddress);
        }
    }

    return handlerResult(HttpStatus.Ok, res);
};

export const parseInvoiceData = (invoice: Stripe.Invoice) => {
    // For our purposes, we can assume that there is only one line in the invoice.
    const line = invoice.lines.data.length > 0 ? invoice.lines.data[0] : null;
    if (!line) {
        return { error: "No lines found in invoice" };
    }
    const { amount, price } = line;
    const paymentType = price ? getPaymentType(price) : null;
    // If this is a payment type we don't understand, return error
    if (!paymentType) {
        return { error: "Unknown payment type" };
    }
    // Return data used to upsert payments from invoices
    return {
        data: {
            checkoutId: invoice.id, // Typically we use a checkout ID here. But for invoices we'll use the invoice ID
            amount,
            currency: invoice.currency,
            paymentMethod: PaymentMethod.Stripe,
            paymentType,
            description: "Payment for Invoice ID: " + invoice.id,
        },
    } as const;
};

/** Payment created, but not finalized */
export const handleInvoicePaymentCreated = async ({ event, res }: Omit<EventHandlerArgs, "stripe">): Promise<HandlerResult> => {
    const invoice = event.data.object as Stripe.Invoice;
    const customerId = getCustomerId(invoice.customer);
    // Find user with the given Stripe customer ID
    const user = await prismaInstance.user.findFirst({
        where: { stripeCustomerId: customerId },
    });
    // Check if user is found
    if (!user) {
        return handlerResult(HttpStatus.InternalServerError, res, "User not found", "0456", { customerId, invoice });
    }
    const { data: paymentData, error: paymentError } = parseInvoiceData(invoice);
    if (paymentError || !paymentData) {
        return handlerResult(HttpStatus.InternalServerError, res, paymentError ?? "Unknown error occurred", "0225", { customerId, invoice });
    }
    // Upsert payment in the database
    const existingPayment = await prismaInstance.payment.findFirst({
        where: {
            checkoutId: invoice.id,
            user: { stripeCustomerId: customerId },
        },
        select: { id: true },
    });
    if (existingPayment) {
        await prismaInstance.payment.update({
            where: { id: existingPayment.id },
            data: {
                ...paymentData,
                status: PaymentStatus.Pending,
            },
        });
    } else {
        await prismaInstance.payment.create({
            data: {
                ...paymentData,
                status: PaymentStatus.Pending,
                user: { connect: { id: user.id } },
            },
        });
    }
    return handlerResult(HttpStatus.Ok, res);
};

/** Payment failed */
export const handleInvoicePaymentFailed = async ({ event, res }: Omit<EventHandlerArgs, "stripe">): Promise<HandlerResult> => {
    const invoice = event.data.object as Stripe.Invoice;
    const customerId = getCustomerId(invoice.customer);
    // Find corresponding payment in the database
    const paymentSelect = {
        id: true,
        paymentType: true,
        user: {
            select: {
                emails: { select: { emailAddress: true } },
            },
        },
    } as const;
    let payment = await prismaInstance.payment.findFirst({
        where: {
            checkoutId: invoice.id,
            user: { stripeCustomerId: customerId },
        },
        select: paymentSelect,
    });
    if (!payment) {
        logger.warning("Did not find existing invoice. This may indicate that handleInvoicePaymentCreated is not working propertly", { trace: "0172", invoice });
        // Create a new payment
        const { data: paymentData, error: paymentError } = parseInvoiceData(invoice);
        if (paymentError || !paymentData) {
            return handlerResult(HttpStatus.InternalServerError, res, paymentError ?? "Unknown error occurred", "0229", { customerId, invoice });
        }
        payment = await prismaInstance.payment.create({
            data: {
                ...paymentData,
                status: PaymentStatus.Failed,
                user: {
                    connect: { stripeCustomerId: customerId },
                },
            },
            select: paymentSelect,
        });
    } else {
        // Update the payment status to Failed
        await prismaInstance.payment.update({
            where: { id: payment.id },
            data: { status: PaymentStatus.Failed },
        });
    }
    // Notify user
    for (const email of payment.user?.emails ?? []) {
        sendPaymentFailed(email.emailAddress, payment.paymentType as PaymentType);
    }
    return handlerResult(HttpStatus.Ok, res);
};


/** Invoice payment succeeded (e.g. subscription renewed) */
export const handleInvoicePaymentSucceeded = async ({ event, stripe, res }: EventHandlerArgs): Promise<HandlerResult> => {
    const invoice = event.data.object as Stripe.Invoice;
    const customerId = getCustomerId(invoice.customer);
    // Find corresponding payment in the database
    const paymentSelect = {
        id: true,
        paymentType: true,
        user: {
            select: {
                id: true,
                emails: { select: { emailAddress: true } },
            },
        },
    } as const;
    let payment = await prismaInstance.payment.findFirst({
        where: {
            checkoutId: invoice.id,
            user: { stripeCustomerId: customerId },
        },
        select: paymentSelect,
    });
    if (!payment) {
        logger.warning("Did not find existing invoice. This may indicate that handleInvoicePaymentCreated is not working propertly", { trace: "0458", invoice });
        // Create a new payment
        const { data: paymentData, error: paymentError } = parseInvoiceData(invoice);
        if (paymentError || !paymentData) {
            return handlerResult(HttpStatus.InternalServerError, res, paymentError ?? "Unknown error occurred", "0231", { customerId, invoice });
        }
        payment = await prismaInstance.payment.create({
            data: {
                ...paymentData,
                status: PaymentStatus.Paid,
                user: {
                    connect: { stripeCustomerId: customerId },
                },
            },
            select: paymentSelect,
        });
    } else {
        // Update the payment status to Paid
        await prismaInstance.payment.update({
            where: { id: payment.id },
            data: { status: PaymentStatus.Paid },
        });
    }
    // If subscription, update or create premium status
    const isSubscription = [PaymentType.PremiumMonthly, PaymentType.PremiumYearly].includes(payment.paymentType as PaymentType);
    if (isSubscription && payment.user) {
        // Fetch subscription from Stripe
        const subscriptionId = typeof invoice.subscription === "string" ? invoice.subscription : invoice.subscription?.id;
        let subscription: Stripe.Response<Stripe.Subscription> | null = null;
        try {
            if (subscriptionId) {
                subscription = await stripe.subscriptions.retrieve(subscriptionId);
            }
        } catch (error) {
            logger.error("Caught error while fetching subscription", { trace: "0540", error });
        }
        if (!subscription) {
            return handlerResult(HttpStatus.InternalServerError, res, "Subscription not found", "0234", { customerId, invoice });
        }
        const { newExpiresAt, isActive } = calculateExpiryAndStatus(subscription.current_period_end);
        // Note that we're not updating the user's credits here, as that would only happen when 
        // activating a subscription. That's handled by the checkout.session.completed event.
        await prismaInstance.user.update({
            where: { id: payment.user.id },
            data: {
                premium: {
                    upsert: {
                        create: {
                            enabledAt: isActive ? new Date() : undefined,
                            expiresAt: newExpiresAt,
                            isActive,
                        },
                        update: {
                            expiresAt: newExpiresAt,
                            isActive,
                        },
                    },
                },
            },
        });
    }
    return handlerResult(HttpStatus.Ok, res);
};

/**
 * Details of a price have been updated
 * NOTE: You can't actually update a price's price; you have to create a new price. 
 * This means this condition isn't needed unless Stripe changes how prices work. 
 * But I already wrote it, so I'm leaving it here.
 */
export const handlePriceUpdated = async ({ event, res }: Omit<EventHandlerArgs, "stripe">): Promise<HandlerResult> => {
    const price = event.data.object as Stripe.Price;
    try {
        const paymentType = getPaymentType(price);
        const updatedAmount = price.unit_amount;
        if (updatedAmount) {
            await storePrice(paymentType, updatedAmount ?? 0);
            return handlerResult(HttpStatus.Ok, res);
        }
    } catch (error) {
        logger.error("Caught error in handlePriceUpdated", { trace: "0172", error });
    }
    return handlerResult(HttpStatus.InternalServerError, res, "Price amount not found", "0170", { price });
};

/**
 * Endpoint for checking subscription prices in the current environment.
 */
export const checkSubscriptionPrices = async (stripe: Stripe, res: Response): Promise<void> => {
    try {
        // Initialize result
        const data: SubscriptionPricesResponse = {
            monthly: await fetchPriceFromRedis(PaymentType.PremiumMonthly) ?? NaN,
            yearly: await fetchPriceFromRedis(PaymentType.PremiumYearly) ?? NaN,
        };
        // Fetch prices from Stripe
        if (!Number.isInteger(data.monthly)) {
            const monthlyPrice = await stripe.prices.retrieve(getPriceIds().PremiumMonthly);
            data.monthly = monthlyPrice?.unit_amount ?? NaN;
            storePrice(PaymentType.PremiumMonthly, data.monthly);
        }
        if (!Number.isInteger(data.yearly)) {
            const yearlyPrice = await stripe.prices.retrieve(getPriceIds().PremiumYearly);
            data.yearly = yearlyPrice?.unit_amount ?? NaN;
            storePrice(PaymentType.PremiumYearly, data.yearly);
        }
        // Send result
        res.status(HttpStatus.Ok).json({ data });
    } catch (error) {
        logger.error("Caught error while checking subscription prices", { trace: "0392", error });
        res.status(HttpStatus.InternalServerError).json({ error });
    }
};

/**
 * Creates a checkout session for buying a subscription, donation, credits, or other payment.
 */
async function createCheckoutSession(stripe: Stripe, req: Request, res: Response): Promise<void> {
    const { amount, userId, variant } = req.body as CreateCheckoutSessionParams;
    const priceId = getPriceIds()[variant];
    const paymentType = variant as PaymentType;
    if (!priceId) {
        logger.error("Invalid variant", { trace: "0436", userId, variant });
        res.status(HttpStatus.BadRequest).json({ error: "Invalid variant" });
        return;
    }
    try {
        const customerInfo = await getVerifiedCustomerInfo({ userId, stripe, validateSubscription: false });
        // We typically need the user to exist, but not always
        const paymentsThatDontNeedUser = [PaymentType.Donation];
        if (!customerInfo.userId && !paymentsThatDontNeedUser.includes(paymentType)) {
            res.status(HttpStatus.InternalServerError).json({ error: "User not found." });
            return;
        }
        // Create customer ID if it doesn't exist
        if (!customerInfo.stripeCustomerId) {
            customerInfo.stripeCustomerId = await createStripeCustomerId({
                customerInfo,
                requireUserToExist: !paymentsThatDontNeedUser.includes(paymentType),
                stripe,
            });
        }
        let line_items: Stripe.Checkout.SessionCreateParams.LineItem[] = [];
        if ([PaymentType.Credits, PaymentType.Donation].includes(paymentType) && amount) {
            const integerAmount = Math.max(100, Math.round(amount)); // Should be at minimum 1 USD
            // For credits with a custom amount, create a dynamic price object
            line_items = [{
                price_data: {
                    currency: "usd",
                    product_data: {
                        name: paymentType === PaymentType.Credits
                            ? "Credits Purchase"
                            : "Donation",
                        description: paymentType === PaymentType.Credits
                            ? "Use credits to chat with bots, run automated routines, autofill forms, and perform any other AI-driven tasks. Purchase credits today and transform how you work, create, and collaborate with Vrooli💙"
                            : "Support our mission to create an automated and open-source economy with a one-time donation. Thank you for believing in us and our vision for a smarter, more connected world💙",
                    },
                    unit_amount: integerAmount,
                },
                quantity: 1,
            }];
        } else {
            const priceId = getPriceIds()[variant];
            if (!priceId) {
                logger.error("Invalid variant", { trace: "0436", userId, variant });
                res.status(HttpStatus.BadRequest).json({ error: "Invalid variant" });
                return;
            }
            // For non-credit variants or credits without a specified amount, use predefined priceId
            line_items = [{
                price: priceId,
                quantity: 1,
            }];
        }
        // Create checkout session 
        const session = await stripe.checkout.sessions.create({
            success_url: `${UI_URL}${LINKS.Pro}?status="success"&paymentType="${paymentType}"`,
            cancel_url: `${UI_URL}${LINKS.Pro}?status="canceled"&paymentType="${paymentType}"`,
            payment_method_types: ["card"],
            line_items,
            mode: [PaymentType.Credits, PaymentType.Donation].includes(variant) ? "payment" : "subscription",
            customer: customerInfo.stripeCustomerId,
            metadata: {
                paymentType,
                userId: userId ?? null,
            } as CheckoutSessionMetadata,
        });
        // Redirect to checkout page
        if (session.url) {
            const data: CreateCheckoutSessionResponse = { url: session.url };
            res.status(HttpStatus.Ok).json({ data });
        } else {
            throw new Error("No session URL returned from Stripe");
        }
    } catch (error) {
        logger.error("Caught error in create-checkout-session", { trace: "0437", error, userId, variant });
        res.status(HttpStatus.InternalServerError).json({ error });
    }
}

/**
 * Creates a portal session for updating payment method or switching/canceling subscription.
 */
async function createPortalSession(stripe: Stripe, req: Request, res: Response): Promise<void> {
    const { userId, returnUrl } = req.body as CreatePortalSessionParams;
    try {
        const customerInfo = await getVerifiedCustomerInfo({ userId, stripe, validateSubscription: false });
        if (!customerInfo.userId) {
            res.status(HttpStatus.InternalServerError).json({ error: "User not found." });
            return;
        }
        // Create customer ID if it doesn't exist
        if (!customerInfo.stripeCustomerId) {
            customerInfo.stripeCustomerId = await createStripeCustomerId({
                customerInfo,
                requireUserToExist: true,
                stripe,
            });
        }
        // Create portal session
        const session = await stripe.billingPortal.sessions.create({
            customer: customerInfo.stripeCustomerId as string,
            return_url: returnUrl ?? UI_URL,
        });
        // Redirect to portal page
        const data: CreateCheckoutSessionResponse = { url: session.url };
        res.status(HttpStatus.Ok).json({ data });
    } catch (error) {
        logger.error("Caught error in create-portal-session", { trace: "0520", error, userId, returnUrl });
        res.status(HttpStatus.InternalServerError).json({ error });
    }
}

/**
 * Checks your subscription status, and fixes your subscription status if a payment
 * was not processed correctly.
 */
const checkSubscription = async (stripe: Stripe, req: Request, res: Response): Promise<void> => {
    let data: CheckSubscriptionResponse = { status: "not_subscribed" };
    const { userId } = req.body as CheckSubscriptionParams;
    try {
        const customerInfo = await getVerifiedCustomerInfo({ userId, stripe, validateSubscription: true });
        if (!customerInfo.userId) {
            throw new Error("User not found");
        }
        // If already active, return
        if (customerInfo.hasPremium) {
            data.status = "already_subscribed";
            res.status(HttpStatus.Ok).json({ data });
            return;
        }
        // Return found subscription info
        if (customerInfo.subscriptionInfo) {
            data = {
                status: "now_subscribed",
                paymentType: customerInfo.subscriptionInfo.paymentType,
            };
            // Process payment so the user gets their subscription
            await processPayment(stripe, customerInfo.subscriptionInfo.session, customerInfo.subscriptionInfo.subscription);
            res.status(HttpStatus.Ok).json({ data });
        }
    } catch (error) {
        logger.error("Caught error checking subscription status", { trace: "0430", error, userId });
        res.status(HttpStatus.InternalServerError).json({ error });
    }
};

/**
 * Checks if a Stripe object is older than a specified age difference. 
 * Useful for expiration purposes.
 */
export const isStripeObjectOlderThan = (
    stripeObject: { created: number }, // "created" is in seconds, according to Stripe API
    ageDifferenceMs: number,
) => {
    const now = Date.now();
    return (stripeObject.created * 1000) + ageDifferenceMs < now;
};

/**
 * Fixes any credits that were paid for but not awarded.
 */
const checkCreditsPayment = async (stripe: Stripe, req: Request, res: Response): Promise<void> => {
    const data: CheckCreditsPaymentResponse = { status: "already_received_all_credits" };
    const { userId } = req.body as CheckCreditsPaymentParams;
    try {
        const customerInfo = await getVerifiedCustomerInfo({ userId, stripe, validateSubscription: false });
        if (!customerInfo.userId) {
            throw new Error("User not found");
        }
        if (!customerInfo.stripeCustomerId) {
            data.status = "already_received_all_credits";
            res.status(HttpStatus.Ok).json({ data });
            return;
        }
        // Find checkout sessions associated with the user
        let sessionsChecked = 0;
        let creditsToAward = BigInt(0); // Credits to award in cents * API_CREDITS_MULTIPLIER
        const upsertOperations: Promise<object>[] = []; // Holds upsert operations for payments
        let done = false;
        // Continually fetch sessions until we reach a maximum of 250, 
        // or until we reach sessions older than 180 days
        while (!done && sessionsChecked < 250) {
            // Fetch sessions
            const sessionsList = await stripe.checkout.sessions.list({
                customer: customerInfo.stripeCustomerId,
                limit: 25,
            });
            sessionsChecked += sessionsList.data.length;
            // Collect valid and complete API credit sessions
            let validCreditSessions = sessionsList.data.filter(session => isValidCreditsPurchaseSession(session, userId));
            // Fetch payments in database. missing ones should be counted, as well as ones marked as pending
            const existingPayments = await prismaInstance.payment.findMany({
                where: {
                    checkoutId: { in: validCreditSessions.map(session => session.id) },
                    paymentMethod: PaymentMethod.Stripe,
                    user: { stripeCustomerId: customerInfo.stripeCustomerId },
                },
                select: {
                    id: true,
                    checkoutId: true,
                    status: true,
                },
            });
            // Filter out sessions which we've already processed and marked as paid
            validCreditSessions = validCreditSessions.filter(session => {
                const payment = existingPayments.find(payment => payment.checkoutId === session.id);
                return !payment || payment.status !== PaymentStatus.Paid;
            });
            // Accumulate credis to award and payments that need to be upserted
            for (const session of validCreditSessions) {
                creditsToAward += (BigInt(session.amount_subtotal ?? 0) * API_CREDITS_MULTIPLIER); // Use subtotal because total includes tax, discounts, etc.
                const payment = existingPayments.find(payment => payment.checkoutId === session.id);
                const paymentData = {
                    amount: session.amount_subtotal ?? 0, // Use subtotal because total includes tax, discounts, etc.
                    checkoutId: session.id,
                    currency: session.currency ?? "usd", // TODO Hopefully always usd, or else our credits logic is flawed (both here and in processPayment). We'll have to see...
                    status: PaymentStatus.Paid,
                    paymentMethod: PaymentMethod.Stripe,
                };
                if (payment) {
                    upsertOperations.push(prismaInstance.payment.update({
                        where: { id: payment.id },
                        data: paymentData,
                    }));
                } else {
                    upsertOperations.push(prismaInstance.payment.create({
                        data: {
                            ...paymentData,
                            description: "Credits Purchase",
                            user: { connect: { id: customerInfo.userId } },
                        },
                    }));
                }
            }
            // If no more sessions to fetch or the last fetched session was over 180 days ago, break
            if (!sessionsList.has_more || isStripeObjectOlderThan(sessionsList.data[sessionsList.data.length - 1], DAYS_180_MS)) {
                done = true;
            }
        }
        // If there are credits to award, award them and upsert payments in the database
        if (creditsToAward > 0) {
            const updateUserCredits = prismaInstance.user.update({
                where: { id: userId },
                data: {
                    premium: {
                        upsert: {
                            create: {
                                credits: creditsToAward,
                            },
                            update: {
                                credits: { increment: creditsToAward },
                            },
                        },
                    },
                },
            });
            // Execute all operations in one transaction
            await prismaInstance.$transaction([updateUserCredits, ...upsertOperations] as PrismaPromise<object>[]);
            data.status = "new_credits_received";
            res.status(HttpStatus.Ok).json({ data });
        } else {
            data.status = "already_received_all_credits";
            res.status(HttpStatus.Ok).json({ data });
        }
    } catch (error) {
        logger.error("Caught error checking credits payment status", { trace: "0439", error, userId });
        res.status(HttpStatus.InternalServerError).json({ error });
    }
};

/**
 * Webhook handler for Stripe events.
 */
const handleStripeWebhook = async (stripe: Stripe, req: Request, res: Response): Promise<void> => {
    const sig: string | string[] = req.headers["stripe-signature"] || "";
    let result: HandlerResult = { status: HttpStatus.InternalServerError, message: "Webhook encountered an error." };
    try {
        // Parse event and verify that it comes from Stripe
        const event = stripe.webhooks.constructEvent(req.body, sig, process.env.STRIPE_WEBHOOK_SECRET || "");
        // Call appropriate handler
        const eventHandlers: { [key in Stripe.WebhookEndpointUpdateParams.EnabledEvent]?: Handler } = {
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
            result = await eventHandlers[event.type]({ event, stripe, res });
        } else {
            // We should only be subscribing to events we want to handle. Meaning if 
            // we reach here, we should go to the Stripe dashboard and remove the unhandled event
            // from the webhook configuration
            logger.warning("Unhandled Stripe event", { trace: "0438", event: event.type });
        }
    } catch (error) {
        logger.error("Caught error in /webhooks/stripe", { trace: "0454", error });
    }
    res.status(result.status);
    if (result.message) {
        res.send(result.message);
    } else {
        res.send();
    }
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
    const stripe = new Stripe(process.env.STRIPE_SECRET_KEY || "", {
        apiVersion: "2022-11-15",
        typescript: true,
        appInfo: {
            name: "Vrooli",
            url: UI_URL,
            version: "2.0.0",
        },
    });
    app.get("/api" + StripeEndpoint.SubscriptionPrices, async (_req: Request, res: Response) => {
        await checkSubscriptionPrices(stripe, res);
    });
    app.post("/api" + StripeEndpoint.CreateCheckoutSession, async (req: Request, res: Response) => {
        await createCheckoutSession(stripe, req, res);
    });
    app.post("/api" + StripeEndpoint.CreatePortalSession, async (req: Request, res: Response) => {
        await createPortalSession(stripe, req, res);
    });
    app.post("/api" + StripeEndpoint.CheckSubscription, async (req: Request, res: Response) => {
        await checkSubscription(stripe, req, res);
    });
    app.post("/api" + StripeEndpoint.CheckCreditsPayment, async (req: Request, res: Response) => {
        await checkCreditsPayment(stripe, req, res);
    });
    app.post("/webhooks/stripe", express.raw({ type: "application/json" }), async (req: Request, res: Response) => {
        await handleStripeWebhook(stripe, req, res);
    });
};

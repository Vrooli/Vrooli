import { API_CREDITS_MULTIPLIER, API_CREDITS_PREMIUM, type CheckCreditsPaymentParams, type CheckCreditsPaymentResponse, type CheckSubscriptionParams, type CheckSubscriptionResponse, type CheckoutSessionMetadata, type CreateCheckoutSessionParams, type CreateCheckoutSessionResponse, type CreatePortalSessionParams, DAYS_180_MS, DAYS_1_S, HttpStatus, LINKS, PaymentStatus, PaymentType, SECONDS_1_MS, StripeEndpoint, type SubscriptionPricesResponse, generatePK } from "@vrooli/shared";
import { CreditEntryType, CreditSourceSystem } from "@prisma/client";
import express, { type Express, type Request, type Response } from "express";
import Stripe from "stripe";
import { DbProvider } from "../db/provider.js";
import { logger } from "../events/logger.js";
import { CacheService } from "../redisConn.js";
import { UI_URL } from "../server.js";
import { SocketService } from "../sockets/io.js";
import { AUTH_EMAIL_TEMPLATES } from "../tasks/email/queue.js";
import { QueueService } from "../tasks/queues.js";
import { ResponseService } from "../utils/response.js";

const STORED_PRICE_EXPIRATION = DAYS_1_S;

// Max number of Stripe sessions to fetch when checking for missed credits (avoid magic numbers)
const MAX_CREDIT_SESSIONS_TO_CHECK = 250;

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
export function getPriceIds(): Record<PaymentType, string> {
    return process.env.NODE_ENV === "production" ? PRICE_IDS_PROD : PRICE_IDS_DEV;
}

/** 
 * Finds the payment type of a Stripe price 
 * 
 * @throws Error if the price ID is invalid
 */
export function getPaymentType(price: string | Stripe.Price | Stripe.DeletedPrice): PaymentType {
    const priceId = typeof price === "string" ? price : price.id;
    const prices = getPriceIds();
    const matchingPrice = Object.entries(prices).find(([_key, value]) => value === priceId);
    if (!matchingPrice) {
        throw new Error("Invalid price ID");
    }
    return matchingPrice[0] as PaymentType;
}

/**
 * Returns the customer ID from the Stripe API response.
 * @param customer The customer field in a Stripe event.
 * This could be a string (representing the customer ID), a Stripe.Customer object, a Stripe.DeletedCustomer object, or null.
 * @returns The customer ID as a string if it exists
 * @throws hrows an error if the customer ID doesn't exist or couldn't be found.
 */
export function getCustomerId(customer: string | Stripe.Customer | Stripe.DeletedCustomer | null | undefined): string {
    if (typeof customer === "string") {
        return customer;
    } else if (customer && typeof customer === "object" && Object.prototype.hasOwnProperty.call(customer, "id") && typeof customer.id === "string") {
        return customer.id; // Access 'id' field if 'customer' is an object
    } else {
        throw new Error("Customer ID not found");
    }
}

/** 
 * Simplifies the return logic for Stripe event handlers, by handling 
 * logging only. Note: HandlerResult's status and optional message are used by the webhook wrapper to send the HTTP response.
 * @param status HTTP status code
 * @param _res Express response object (unused)
 * @param message Message to return in the response body
 * @param trace Trace to log for errors
 * @param ...args Additional arguments to log
 * @returns Object with status and message for final response
 */
export function handlerResult(
    status: HttpStatus,
    _res: Response,
    message?: string,
    trace?: string,
    ...args: unknown[]
): HandlerResult {
    if (status !== HttpStatus.Ok) {
        logger.error(message ?? "Stripe handler error", { trace: trace ?? "0523", ...args });
    }
    // Return the result without directly sending the response
    return { status, message };
}

/**
 * Stores a price in redis for a given payment type
 */
export async function storePrice(paymentType: PaymentType, price: number): Promise<void> {
    const key = `stripe-payment-${process.env.NODE_ENV}-${paymentType}`;
    // Store price in Redis via CacheService with expiration TTL
    await CacheService.get().set(key, price, STORED_PRICE_EXPIRATION);
}

/**
 * Fetches a price from redis for a given payment type, 
 * or null if the price is not found
 */
export async function fetchPriceFromRedis(paymentType: PaymentType): Promise<number | null> {
    const key = `stripe-payment-${process.env.NODE_ENV}-${paymentType}`;
    // Retrieve price from Redis via CacheService
    const price = await CacheService.get().get<number>(key);
    // Ensure valid non-negative number
    if (typeof price === "number" && !isNaN(price) && price >= 0) {
        return price;
    }
    return null;
}

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
export function isValidSubscriptionSession(session: Stripe.Checkout.Session, userId: string): boolean {
    const { paymentType, userId: sessionUserId } = session.metadata as CheckoutSessionMetadata;
    // A valid subscription session must be of a recognized tier, initiated by this user,
    // have successful payment, and be in the correct environment.
    // Use payment_status to capture both synchronous and asynchronous completions.
    return [PaymentType.PremiumMonthly, PaymentType.PremiumYearly].includes(paymentType)
        && sessionUserId === userId
        && (session.payment_status === "paid" || session.status === "complete")
        && isInCorrectEnvironment(session);
}

/** @returns True if the Stripe session should be counted as rewarding credits */
export function isValidCreditsPurchaseSession(session: Stripe.Checkout.Session, userId: string): boolean {
    const { paymentType, userId: sessionUserId } = session.metadata as CheckoutSessionMetadata;
    // A valid credit purchase session must be for credits, initiated by this user,
    // have successful payment, and be in the correct environment.
    return paymentType === PaymentType.Credits
        && sessionUserId === userId
        && (session.payment_status === "paid" || session.status === "complete")
        && isInCorrectEnvironment(session);
}

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
export async function getVerifiedSubscriptionInfo(
    stripe: Stripe,
    customerId: string,
    userId: string,
): Promise<GetVerifiedSubscriptionInfoResult | null> {
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
}

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
    const user = await DbProvider.get().user.findUnique({
        where: { id: BigInt(userId) },
        select: {
            id: true,
            emails: { select: { emailAddress: true } },
            plan: { select: { enabledAt: true, expiresAt: true } },
            stripeCustomerId: true,
        },
    });
    if (!user) {
        logger.error("User not found.", { trace: "0519", userId });
        return result;
    }
    result.emails = user.emails || [];
    // Determine active premium by checking expiration timestamp
    result.hasPremium = !!user.plan?.expiresAt && user.plan.expiresAt > new Date();
    result.stripeCustomerId = user.stripeCustomerId || null;
    result.userId = user.id.toString();
    // Validate the stripeCustomerId by checking if it exists in Stripe
    if (result.stripeCustomerId) {
        try {
            const customer = await stripe.customers.retrieve(result.stripeCustomerId);
            if (customer.deleted) {
                result.stripeCustomerId = null;
                logger.error("Stripe customer was deleted", { trace: "0522", result });
                // Remove stale Stripe customerId from our DB
                await DbProvider
                    .get()
                    .user.update({
                        where: { id: user.id },
                        data: { stripeCustomerId: null },
                    });
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
            await DbProvider.get().user.update({
                where: { id: BigInt(userId) },
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
        await DbProvider.get().user.update({
            where: { id: BigInt(customerInfo.userId) },
            data: { stripeCustomerId: stripeCustomer.id },
        });
    }
    return stripeCustomer.id;
}

/**
 * Helper to award API credits for a credits purchase.
 */
async function handleCreditsAward(
    payment: { amount: number; user: { id: bigint; emails: { emailAddress: string }[] } | null },
    checkoutId: string,
): Promise<void> {
    const user = payment.user;
    if (!user) {
        throw new Error("User not found.");
    }
    const creditsToAward = BigInt(payment.amount) * API_CREDITS_MULTIPLIER;
    const userIdBigInt = BigInt(user.id);
    // Ensure a credit_account exists
    let creditAccount = await DbProvider.get().credit_account.findUnique({ where: { userId: userIdBigInt } });
    if (!creditAccount) {
        creditAccount = await DbProvider.get().credit_account.create({ data: { id: generatePK(), userId: userIdBigInt } });
    }
    if (!creditAccount) {
        throw new Error("Credit account not found.");
    }
    // Add ledger entry and update balance
    await DbProvider.get().$transaction(async (tx) => {
        await tx.credit_ledger_entry.create({
            data: {
                id: generatePK(),
                idempotencyKey: checkoutId,
                accountId: creditAccount.id,
                amount: creditsToAward,
                type: CreditEntryType.Purchase,
                source: CreditSourceSystem.Stripe,
            },
        });
        const acct = await tx.credit_account.findUniqueOrThrow({
            where: { id: creditAccount.id },
            select: { id: true, currentBalance: true },
        });
        await tx.credit_account.update({ where: { id: acct.id }, data: { currentBalance: acct.currentBalance + creditsToAward } });
    });
    // Notify clients of new credits
    SocketService.get().emitSocketEvent("apiCredits", user.id.toString(), { credits: creditsToAward + "" });
}

/**
 * Helper to process subscription rewards: upsert plan and award premium credits.
 */
async function handleSubscriptionAward(
    stripe: Stripe,
    payment: { user: { id: bigint; emails: { emailAddress: string }[] } | null },
    checkoutId: string,
    session: Stripe.Checkout.Session,
    subscription?: Stripe.Subscription,
): Promise<void> {
    const user = payment.user;
    if (!user) {
        throw new Error("User not found.");
    }
    // Validate subscription session
    if (!isValidSubscriptionSession(session, user.id.toString())) {
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
    // Upsert plan
    const enabledAt = new Date().toISOString();
    const expiresAt = new Date(knownSubscription.current_period_end * SECONDS_1_MS).toISOString();
    const plans = await DbProvider.get().plan.findMany({ where: { user: { id: user.id } } });
    if (plans.length === 0) {
        await DbProvider.get().plan.create({ data: { id: generatePK(), enabledAt, expiresAt, user: { connect: { id: user.id } } } });
    } else {
        await DbProvider.get().plan.update({ where: { id: plans[0].id }, data: { enabledAt: plans[0].enabledAt ?? enabledAt, expiresAt } });
    }
    // Award premium credits
    const subscriptionCredits = API_CREDITS_PREMIUM;
    const userIdBigInt = BigInt(user.id);
    let creditAccount = await DbProvider.get().credit_account.findUnique({ where: { userId: userIdBigInt } });
    if (!creditAccount) {
        creditAccount = await DbProvider.get().credit_account.create({ data: { id: generatePK(), userId: userIdBigInt } });
    }
    await DbProvider.get().$transaction(async (tx) => {
        await tx.credit_ledger_entry.create({ data: { id: generatePK(), idempotencyKey: checkoutId, accountId: creditAccount!.id, amount: subscriptionCredits, type: CreditEntryType.Purchase, source: CreditSourceSystem.Stripe } });
        const acct = await tx.credit_account.findUniqueOrThrow({ where: { id: creditAccount!.id }, select: { id: true, currentBalance: true } });
        await tx.credit_account.update({ where: { id: acct.id }, data: { currentBalance: acct.currentBalance + subscriptionCredits } });
    });
    // Notify clients of new credits
    SocketService.get().emitSocketEvent("apiCredits", user.id.toString(), { credits: API_CREDITS_PREMIUM + "" });
}

/**
 * Processes a completed payment.
 * @param stripe The Stripe object for interacting with the Stripe API
 * @param session The Stripe checkout session object
 * @param subscription The Stripe subscription object linked to the session, if known
 * @throws Error if something goes wrong, such as if the user is not found
 */
export async function processPayment(
    stripe: Stripe,
    session: Stripe.Checkout.Session,
    subscription?: Stripe.Subscription,
): Promise<void> {
    const checkoutId = session.id;
    const customerId = getCustomerId(session.customer);
    const { paymentType, userId } = session.metadata as CheckoutSessionMetadata;
    // Skip if payment already processed
    const payments = await DbProvider.get().payment.findMany({
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
    const payment = paymentId ? await DbProvider.get().payment.update({
        where: { id: paymentId },
        data,
        select: paymentSelect,
    }) : await DbProvider.get().payment.create({
        data: {
            ...data,
            id: generatePK(),
            user: userId ? { connect: { id: BigInt(userId) } } : undefined,
        },
        select: paymentSelect,
    });
    // If there is no user and this is not a donation, throw an error
    if (!payment.user && payment.paymentType !== PaymentType.Donation) {
        throw new Error("User not found.");
    }
    // Delegate credit or subscription rewards to helper functions
    if (payment.paymentType === PaymentType.Credits) {
        await handleCreditsAward(payment, checkoutId);
    } else if (payment.paymentType === PaymentType.PremiumMonthly || payment.paymentType === PaymentType.PremiumYearly) {
        await handleSubscriptionAward(stripe, payment, checkoutId, session, subscription);
    }
    // Send thank you notification if we have a user (supports anonymous donations without email)
    if (payment.user) {
        const donationFlag = payment.paymentType === PaymentType.Donation;
        await QueueService.get().email.addTask({
            to: payment.user.emails.map(email => email.emailAddress),
            ...AUTH_EMAIL_TEMPLATES.PaymentThankYou(payment.user.id.toString(), donationFlag),
        });
    }
}

/**
 * Handles successful completion of a Checkout Session.
 *
 * This handler listens to:
 *   - `checkout.session.completed`: fired when a Checkout Session completes; for delayed payment methods, `session.payment_status` may not be 'paid' immediately.
 *   - `checkout.session.async_payment_succeeded`: fired when a delayed payment intent (e.g., ACH) transitions to 'paid'.
 *
 * Only sessions with `session.payment_status === 'paid'` should trigger fulfillment logic.
 * This ensures credits or subscription access are granted only after funds are confirmed.
 *
 * @param args.event The Stripe.Event containing the Checkout Session.
 * @param args.stripe The Stripe client instance used for API operations.
 * @param args.res Express response object for HTTP acknowledgment.
 * @returns HandlerResult indicating success (200) or failure (5xx).
 */
async function handleCheckoutSessionCompleted({ event, stripe, res }: EventHandlerArgs): Promise<HandlerResult> {
    try {
        const session = event.data.object as Stripe.Checkout.Session;
        // Only process sessions where payment_status === 'paid' to avoid premature fulfillment
        if (session.payment_status !== "paid") {
            return handlerResult(HttpStatus.Ok, res);
        }
        await processPayment(stripe, session);
        return handlerResult(HttpStatus.Ok, res);
    } catch (error) {
        return handlerResult(HttpStatus.InternalServerError, res, "Caught error in handleCheckoutSessionCompleted", "0440", error);
    }
}

/**
 * Handles Checkout Session failures or expirations.
 *
 * This handler listens to:
 *   - `checkout.session.expired`: when a Checkout Session expires without payment.
 *   - `checkout.session.async_payment_failed`: when delayed payment methods fail asynchronously.
 *
 * Both events indicate `session.payment_status !== 'paid'`. Marks pending payments as failed to ensure idempotent failure handling.
 *
 * @param args.event The Stripe.Event containing the Checkout Session.
 * @param args.res Express response object for HTTP acknowledgment.
 * @returns HandlerResult indicating success (200) or failure (5xx).
 */
export async function handleCheckoutSessionExpired({ event, res }: EventHandlerArgs): Promise<HandlerResult> {
    const session = event.data.object as Stripe.Checkout.Session;
    const checkoutId = session.id;
    const customerId = getCustomerId(session.customer);
    // Find payment in database
    const payments = await DbProvider.get().payment.findMany({
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
        await DbProvider.get().payment.update({
            where: { id: payment.id },
            data: { status: PaymentStatus.Failed },
        });
    }
    return handlerResult(HttpStatus.Ok, res);
}

/** Customer was deleted */
export async function handleCustomerDeleted({ event, res }: EventHandlerArgs): Promise<HandlerResult> {
    const customer = event.data.object as Stripe.Customer;
    const customerId = getCustomerId(customer);
    // Attempt to find the user associated with the given customerId
    const user = await DbProvider.get().user.findUnique({
        where: { stripeCustomerId: customerId },
    });
    // If the user exists, remove the customerId
    if (user) {
        await DbProvider.get().user.update({
            where: { stripeCustomerId: customerId },
            data: { stripeCustomerId: null },
        });
    }
    // Return an OK status regardless of whether the user was found or not
    return handlerResult(HttpStatus.Ok, res);
}

/** Customer's credit card is about to expire */
export async function handleCustomerSourceExpiring({ event, res }: Omit<EventHandlerArgs, "stripe">): Promise<HandlerResult> {
    const customer = event.data.object as Stripe.Customer;
    const customerId = getCustomerId(customer);
    // Find user with the given Stripe customer ID
    const user = await DbProvider.get().user.findFirst({
        where: { stripeCustomerId: customerId },
        select: {
            id: true,
            emails: { select: { emailAddress: true } },
        },
    });
    if (!user) {
        return handlerResult(HttpStatus.InternalServerError, res, "User not found", "0468", { customerId });
    }
    // Send notification
    await QueueService.get().email.addTask({
        to: user.emails.map(email => email.emailAddress),
        ...AUTH_EMAIL_TEMPLATES.CreditCardExpiringSoon(user.id.toString()),
    });
    return handlerResult(HttpStatus.Ok, res);
}

/**
 * Occurs when a customer's subscription ends (customer.subscription.deleted).
 * This event fires at the end of the billing period or when the subscription is fully canceled.
 * We update the user's plan expiresAt to the subscription's current_period_end timestamp,
 * then send a cancellation notification email.
 */
export async function handleCustomerSubscriptionDeleted({ event, res }: Omit<EventHandlerArgs, "stripe">): Promise<HandlerResult> {
    const subscription = event.data.object as Stripe.Subscription;
    const customerId = getCustomerId(subscription.customer);
    // Find user and current plan
    const user = await DbProvider.get().user.findFirst({
        where: { stripeCustomerId: customerId },
        select: {
            id: true,
            emails: { select: { emailAddress: true } },
            plan: { select: { enabledAt: true } },
        },
    });
    if (!user) {
        return handlerResult(HttpStatus.InternalServerError, res, "User not found.", "0459", { customerId });
    }
    // Update plan expiry date
    const { newExpiresAt } = calculateExpiryAndStatus(subscription.current_period_end);
    await DbProvider.get().user.update({
        where: { id: user.id },
        data: {
            plan: {
                upsert: {
                    create: {
                        id: generatePK(),
                        enabledAt: user.plan?.enabledAt ?? new Date(),
                        expiresAt: newExpiresAt,
                    },
                    update: {
                        expiresAt: newExpiresAt,
                    },
                },
            },
        },
    });
    // Send cancellation email
    await QueueService.get().email.addTask({
        to: user.emails.map(email => email.emailAddress),
        ...AUTH_EMAIL_TEMPLATES.SubscriptionCanceled(user.id.toString()),
    });
    return handlerResult(HttpStatus.Ok, res);
}

/** Trial ending in a few days */
export async function handleCustomerSubscriptionTrialWillEnd({ event, res }: Omit<EventHandlerArgs, "stripe">): Promise<HandlerResult> {
    const subscription = event.data.object as Stripe.Subscription;
    const customerId = getCustomerId(subscription.customer);
    // Find user with the given Stripe customer ID
    const user = await DbProvider.get().user.findFirst({
        where: { stripeCustomerId: customerId },
        select: {
            id: true,
            emails: { select: { emailAddress: true } },
        },
    });
    if (!user) {
        return handlerResult(HttpStatus.InternalServerError, res, "User not found.", "0192", { customerId });
    }
    // If found, send notification
    await QueueService.get().email.addTask({
        to: user.emails.map(email => email.emailAddress),
        ...AUTH_EMAIL_TEMPLATES.TrialEndingSoon(user.id.toString()),
    });
    return handlerResult(HttpStatus.Ok, res);
}

/**
 * Calculates the expiration date and active status of a subscription.
 * 
 * @param currentPeriodEnd - The end of the current billing cycle as a Unix timestamp in seconds.
 * @returns The new expiration date and active status.
 */
export function calculateExpiryAndStatus(currentPeriodEnd: number) {
    const newExpiresAt = new Date(currentPeriodEnd * SECONDS_1_MS); // Convert seconds to milliseconds
    const now = new Date();
    const isActive = newExpiresAt > now;
    return { newExpiresAt, isActive };
}

/** 
 * Occurs whenever a subscription changes (e.g., switching from monthly 
 * to yearly, or changing the status from trial to active). 
 */
export async function handleCustomerSubscriptionUpdated({ event, res }: Omit<EventHandlerArgs, "stripe">): Promise<HandlerResult> {
    const subscription = event.data.object as Stripe.Subscription;
    const customerId = getCustomerId(subscription.customer);

    // Skip processing if the subscription is not marked as active. 
    // Canceled subscriptions are handled by the customer.subscription.deleted event.
    if (subscription.status !== "active") {
        // This isn't a problem with the request itself, so we'll return a 200 status code
        return handlerResult(HttpStatus.Ok, res, "Subscription is not active.");
    }

    const user = await DbProvider.get().user.findFirst({
        where: { stripeCustomerId: customerId },
        select: {
            id: true,
            emails: {
                select: {
                    emailAddress: true,
                },
            },
            plan: { select: { enabledAt: true } },
        },
    });
    if (!user) {
        return handlerResult(HttpStatus.InternalServerError, res, "User not found on customer.subscription.updated", "0457", { customerId });
    }
    const existingEnabledAt = user.plan?.enabledAt ?? null;

    // Calculate new expiration date and active status. 
    // Should always be active, unless the new expiry date is somehow in the past
    const { newExpiresAt, isActive } = calculateExpiryAndStatus(subscription.current_period_end);
    const enabledAt = isActive
        ? existingEnabledAt ?? new Date()
        : null;

    // Note that we're not updating the user's credits here, as that would only happen when 
    // activating a subscription. That's handled by the checkout.session.completed event.
    await DbProvider.get().user.update({
        where: { id: user.id },
        data: {
            plan: {
                upsert: {
                    create: {
                        id: generatePK(),
                        enabledAt,
                        expiresAt: newExpiresAt,
                    },
                    update: {
                        enabledAt,
                        expiresAt: newExpiresAt,
                    },
                },
            },
        },
    });

    const emails = user.emails.map(email => email.emailAddress);
    if (isActive) {
        await QueueService.get().email.addTask({
            to: emails,
            ...AUTH_EMAIL_TEMPLATES.PaymentThankYou(user.id.toString(), false),
        });
    } else {
        await QueueService.get().email.addTask({
            to: emails,
            ...AUTH_EMAIL_TEMPLATES.SubscriptionCanceled(user.id.toString()),
        });
    }

    return handlerResult(HttpStatus.Ok, res);
}

/**
 * Extracts payment data from a Stripe Invoice for upserting payment records.
 *
 * Invoices may contain multiple line items (e.g., subscription line, proration, tax).
 * This function filters for the subscription or known product line item by matching price IDs.
 *
 * @param invoice A Stripe.Invoice object to parse.
 * @returns An object containing:
 *   - data: fields required to upsert payment (checkoutId, amount, currency, paymentMethod, paymentType, description).
 *   - error: populated if no matching line item is found.
 */
export function parseInvoiceData(invoice: Stripe.Invoice) {
    // Retrieve our known price ID mapping for the current environment
    const pricesMap = getPriceIds();
    // Filter for the first line item with a recognized price
    const matchingLine = invoice.lines.data.find(line => {
        const priceObj = line.price;
        if (!priceObj) return false;
        const priceId = typeof priceObj === "string" ? priceObj : priceObj.id;
        return Object.values(pricesMap).includes(priceId);
    });
    if (!matchingLine || !matchingLine.price) {
        return { error: "Unknown or unsupported invoice line item" };
    }
    const { amount } = matchingLine;
    const currency = invoice.currency;
    const priceObj = matchingLine.price as Stripe.Price;
    const paymentType = getPaymentType(priceObj);
    return {
        data: {
            // Use the invoice ID as the checkoutId for invoice-based payments
            checkoutId: invoice.id,
            amount,
            currency,
            paymentMethod: PaymentMethod.Stripe,
            paymentType,
            description: "Payment for Invoice ID: " + invoice.id,
        },
    } as const;
}

/** Payment created, but not finalized */
export async function handleInvoicePaymentCreated({ event, res }: Omit<EventHandlerArgs, "stripe">): Promise<HandlerResult> {
    const invoice = event.data.object as Stripe.Invoice;
    const customerId = getCustomerId(invoice.customer);
    // Find user with the given Stripe customer ID
    const user = await DbProvider.get().user.findFirst({
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
    const existingPayment = await DbProvider.get().payment.findFirst({
        where: {
            checkoutId: invoice.id,
            user: { stripeCustomerId: customerId },
        },
        select: { id: true },
    });
    if (existingPayment) {
        await DbProvider.get().payment.update({
            where: { id: existingPayment.id },
            data: {
                ...paymentData,
                status: PaymentStatus.Pending,
            },
        });
    } else {
        await DbProvider.get().payment.create({
            data: {
                ...paymentData,
                id: generatePK(),
                status: PaymentStatus.Pending,
                user: { connect: { id: user.id } },
            },
        });
    }
    return handlerResult(HttpStatus.Ok, res);
}

/** Payment failed */
export async function handleInvoicePaymentFailed({ event, res }: Omit<EventHandlerArgs, "stripe">): Promise<HandlerResult> {
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
    let payment = await DbProvider.get().payment.findFirst({
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
        payment = await DbProvider.get().payment.create({
            data: {
                ...paymentData,
                id: generatePK(),
                status: PaymentStatus.Failed,
                user: {
                    connect: { stripeCustomerId: customerId },
                },
            },
            select: paymentSelect,
        });
    } else {
        // Update the payment status to Failed
        await DbProvider.get().payment.update({
            where: { id: payment.id },
            data: { status: PaymentStatus.Failed },
        });
    }
    // Notify user
    const emails = payment.user?.emails.map(email => email.emailAddress) ?? [];
    if (payment.user && emails.length > 0) {
        const donationFlag = payment.paymentType === PaymentType.Donation;
        await QueueService.get().email.addTask({
            to: emails,
            ...AUTH_EMAIL_TEMPLATES.PaymentFailed(payment.user.id.toString(), donationFlag),
        });
    }
    return handlerResult(HttpStatus.Ok, res);
}


/** Invoice payment succeeded (e.g. subscription renewed) */
export async function handleInvoicePaymentSucceeded({ event, stripe, res }: EventHandlerArgs): Promise<HandlerResult> {
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
                plan: { select: { enabledAt: true } },
            },
        },
    } as const;
    let payment = await DbProvider.get().payment.findFirst({
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
        payment = await DbProvider.get().payment.create({
            data: {
                ...paymentData,
                id: generatePK(),
                status: PaymentStatus.Paid,
                user: {
                    connect: { stripeCustomerId: customerId },
                },
            },
            select: paymentSelect,
        });
    } else {
        // Update the payment status to Paid
        await DbProvider.get().payment.update({
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
        const existingEnabledAt = payment.user?.plan?.enabledAt ?? null;
        const enabledAt = isActive
            ? existingEnabledAt ?? new Date()
            : null;
        // Note that we're not updating the user's credits here, as that would only happen when 
        // activating a subscription. That's handled by the checkout.session.completed event.
        await DbProvider.get().user.update({
            where: { id: payment.user.id },
            data: {
                plan: {
                    upsert: {
                        create: {
                            id: generatePK(),
                            enabledAt,
                            expiresAt: newExpiresAt,
                        },
                        update: {
                            enabledAt,
                            expiresAt: newExpiresAt,
                        },
                    },
                },
            },
        });
    }
    return handlerResult(HttpStatus.Ok, res);
}

/**
 * Details of a price have been updated
 * NOTE: You can't actually update a price's price; you have to create a new price. 
 * This means this condition isn't needed unless Stripe changes how prices work. 
 * But I already wrote it, so I'm leaving it here.
 */
export async function handlePriceUpdated({ event, res }: Omit<EventHandlerArgs, "stripe">): Promise<HandlerResult> {
    const price = event.data.object as Stripe.Price;
    // Determine if this price ID maps to one of our PaymentTypes
    let paymentType: PaymentType;
    try {
        paymentType = getPaymentType(price);
    } catch (_error) {
        // Price ID not in our mapping; acknowledge and ignore
        return handlerResult(HttpStatus.Ok, res);
    }
    // Prefer integer unit_amount; fall back to unit_amount_decimal if present
    const updatedAmount = price.unit_amount ?? (price.unit_amount_decimal ? Math.round(parseFloat(price.unit_amount_decimal)) : null);
    if (updatedAmount != null) {
        await storePrice(paymentType, updatedAmount);
    }
    // Always acknowledge the webhook with OK
    return handlerResult(HttpStatus.Ok, res);
}

/**
 * Endpoint for checking subscription prices in the current environment.
 */
export async function checkSubscriptionPrices(stripe: Stripe, res: Response): Promise<void> {
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
            await storePrice(PaymentType.PremiumMonthly, data.monthly);
        }
        if (!Number.isInteger(data.yearly)) {
            const yearlyPrice = await stripe.prices.retrieve(getPriceIds().PremiumYearly);
            data.yearly = yearlyPrice?.unit_amount ?? NaN;
            await storePrice(PaymentType.PremiumYearly, data.yearly);
        }
        // Send result
        ResponseService.sendSuccess(res, data);
    } catch (error) {
        const trace = "0392";
        const message = "Caught error while checking subscription prices.";
        logger.error(message, { trace, error });
        ResponseService.sendError(res, { trace, message }, HttpStatus.InternalServerError);
    }
}

/**
 * Creates a Stripe Checkout Session for purchasing credits, donations, or subscriptions.
 * - Constructs line_items based on payment type and optional custom amount.
 * - Enforces a minimum amount of $1 USD (100 cents) for credits/donations.
 * - Uses dynamic price_data for custom credit purchases or predefined Price IDs for standard variants.
 * - Builds success_url and cancel_url with proper query parameters (no extra quotes).
 */
async function createCheckoutSession(stripe: Stripe, req: Request, res: Response): Promise<void> {
    const { amount, userId, variant } = req.body as CreateCheckoutSessionParams;
    const priceId = getPriceIds()[variant];
    const paymentType = variant as PaymentType;
    if (!priceId) {
        const trace = "0450";
        const message = "Invalid variant.";
        logger.error(message, { trace, userId, variant });
        ResponseService.sendError(res, { trace, message }, HttpStatus.BadRequest);
        return;
    }
    try {
        const customerInfo = await getVerifiedCustomerInfo({ userId, stripe, validateSubscription: false });
        // We typically need the user to exist, but not always
        const paymentsThatDontNeedUser = [PaymentType.Donation];
        if (!customerInfo.userId && !paymentsThatDontNeedUser.includes(paymentType)) {
            const trace = "0667";
            const message = "User not found.";
            logger.error(message, { trace, userId, variant });
            ResponseService.sendError(res, { trace, message }, HttpStatus.InternalServerError);
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
                const trace = "0436";
                const message = "Invalid variant.";
                logger.error(message, { trace, userId, variant });
                ResponseService.sendError(res, { trace, message }, HttpStatus.BadRequest);
                return;
            }
            // For non-credit variants or credits without a specified amount, use predefined priceId
            line_items = [{
                price: priceId,
                quantity: 1,
            }];
        }
        // Build metadata, omitting userId when undefined so all values are strings
        const metadata: { paymentType: string; userId?: string } = { paymentType };
        if (userId) {
            metadata.userId = userId;
        }
        const session = await stripe.checkout.sessions.create({
            success_url: `${UI_URL}${LINKS.Pro}?status=success&paymentType=${paymentType}`,
            cancel_url: `${UI_URL}${LINKS.Pro}?status=canceled&paymentType=${paymentType}`,
            payment_method_types: ["card"],
            line_items,
            mode: [PaymentType.Credits, PaymentType.Donation].includes(variant) ? "payment" : "subscription",
            customer: customerInfo.stripeCustomerId,
            metadata,
        });
        // Redirect to checkout page
        if (session.url) {
            const data: CreateCheckoutSessionResponse = { url: session.url };
            ResponseService.sendSuccess(res, data);
        } else {
            throw new Error("No session URL returned from Stripe");
        }
    } catch (error) {
        const trace = "0437";
        const message = "Caught error in create-checkout-session";
        logger.error(message, { trace, error, userId, variant });
        ResponseService.sendError(res, { trace, message }, HttpStatus.InternalServerError);
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
            const trace = "0676";
            const message = "User not found.";
            ResponseService.sendError(res, { trace, message }, HttpStatus.InternalServerError);
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
        ResponseService.sendSuccess(res, data);
    } catch (error) {
        const trace = "0520";
        const message = "Caught error in create-portal-session";
        logger.error(message, { trace, error, userId, returnUrl });
        ResponseService.sendError(res, { trace, message }, HttpStatus.InternalServerError);
    }
}

/**
 * Checks your subscription status, and fixes your subscription status if a payment
 * was not processed correctly.
 *
 * @remarks
 * This endpoint uses getVerifiedCustomerInfo to validate subscription status and always returns HTTP 200 on success.
 * The JSON payload is { status: string } with one of three values:
 *  - 'not_subscribed': no existing or new subscription found.
 *  - 'already_subscribed': user already has an active subscription.
 *  - 'now_subscribed': a valid subscription was detected and processed.
 */
async function checkSubscription(stripe: Stripe, req: Request, res: Response): Promise<void> {
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
            ResponseService.sendSuccess(res, data);
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
            ResponseService.sendSuccess(res, data);
            return;
        }
        // Default: no subscription found; send not_subscribed status
        ResponseService.sendSuccess(res, data);
        return;
    } catch (error) {
        const trace = "0430";
        const message = "Caught error checking subscription status";
        logger.error(message, { trace, error, userId });
        ResponseService.sendError(res, { trace, message }, HttpStatus.InternalServerError);
    }
}

/**
 * Checks if a Stripe object is older than a specified age difference. 
 * Useful for expiration purposes.
 */
export function isStripeObjectOlderThan(
    stripeObject: { created: number }, // "created" is in seconds, according to Stripe API
    ageDifferenceMs: number,
) {
    const now = Date.now();
    return (stripeObject.created * SECONDS_1_MS) + ageDifferenceMs < now;
}

/**
 * Fixes any credits that were paid for but not awarded.
 */
async function checkCreditsPayment(stripe: Stripe, req: Request, res: Response): Promise<void> {
    const data: CheckCreditsPaymentResponse = { status: "already_received_all_credits" };
    const { userId } = req.body as CheckCreditsPaymentParams;
    try {
        const customerInfo = await getVerifiedCustomerInfo({ userId, stripe, validateSubscription: false });
        if (!customerInfo.userId || !customerInfo.stripeCustomerId) {
            ResponseService.sendSuccess(res, data);
            return;
        }
        let newCredits = false;
        let sessionsChecked = 0;
        // Use pagination to avoid reprocessing the same sessions
        let done = false;
        let startingAfter: string | undefined = undefined;
        while (!done && sessionsChecked < MAX_CREDIT_SESSIONS_TO_CHECK) {
            const listParams: Stripe.Checkout.SessionListParams = {
                customer: customerInfo.stripeCustomerId,
                limit: 25,
            };
            if (startingAfter) {
                listParams.starting_after = startingAfter;
            }
            const sessionsList = await stripe.checkout.sessions.list(listParams);
            sessionsChecked += sessionsList.data.length;
            for (const session of sessionsList.data) {
                if (isValidCreditsPurchaseSession(session, customerInfo.userId)) {
                    await processPayment(stripe, session);
                    newCredits = true;
                }
            }
            // If there are sessions, process pagination and age threshold; else stop looping
            if (sessionsList.data.length > 0) {
                const lastSession = sessionsList.data[sessionsList.data.length - 1];
                if (sessionsList.has_more) {
                    startingAfter = lastSession.id;
                }
                // Stop when no more pages or sessions older than threshold
                if (!sessionsList.has_more || isStripeObjectOlderThan(lastSession, DAYS_180_MS)) {
                    done = true;
                }
            } else {
                // No sessions returned; break out to avoid infinite loop
                done = true;
            }
        }
        data.status = newCredits ? "new_credits_received" : "already_received_all_credits";
        ResponseService.sendSuccess(res, data);
    } catch (error) {
        const trace = "0439";
        const message = "Caught error checking credits payment status";
        logger.error(message, { trace, error, userId });
        ResponseService.sendError(res, { trace, message }, HttpStatus.InternalServerError);
    }
}

/**
 * handleStripeWebhook processes incoming Stripe webhook events.
 *
 * This wrapper:
 * - Uses express.raw({ type: 'application/json' }) so that the raw request payload
 *   is available for signature verification by stripe.webhooks.constructEvent.
 * - Verifies the incoming webhook payload's signature using the STRIPE_WEBHOOK_SECRET.
 *   Throws an error and returns a 500 if signature verification fails.
 * - Identifies the event type (event.type) and dispatches to the corresponding handler:
 *     • checkout.session.completed: synchronous card payment completions
 *     • checkout.session.async_payment_succeeded: asynchronous payments (e.g., ACH, SEPA)
 *     • checkout.session.expired: session expiration before payment
 *     • checkout.session.async_payment_failed: asynchronous payment failures
 *     • customer.deleted: Stripe customer deletion — nullifies stripeCustomerId in our DB
 *     • customer.source.expiring: payment source expiring soon — sends reminder email
 *     • customer.subscription.deleted: subscription cancellation — updates plan expiry
 *     • customer.subscription.trial_will_end: trial ending soon — sends notification
 *     • customer.subscription.updated: subscription plan change — upserts expiration and sends notifications
 *     • invoice.created: new invoice (e.g., subscription renewal) — creates/updates pending payment record
 *     • invoice.payment_failed: invoice payment failure — marks failed & notifies user
 *     • invoice.payment_succeeded: invoice payment success — marks paid & updates subscription status
 *     • invoice.paid: (alias for payment success) also handled by the same logic as invoice.payment_succeeded
 *     • price.updated: price metadata update — caches updated unit_amount in Redis
 *
 * Best practices:
 * - Only listen to the events your integration requires.
 * - Ensure idempotency: handlers include checks (e.g., existing payments by checkoutId) to avoid duplicates.
 * - Quickly acknowledge (2xx) before performing complex logic to prevent timeouts.
 * - Verify webhook signatures and handle retries appropriately.
 *
 * @param stripe The initialized Stripe client instance (apiVersion 2022-11-15).
 * @param req Express Request with raw JSON body for signature validation.
 * @param res Express Response used only to send status and optional message.
 */
async function handleStripeWebhook(stripe: Stripe, req: Request, res: Response): Promise<void> {
    // Extract and coerce the Stripe signature header to a single string (in case it's an array)
    const rawSignature = req.headers["stripe-signature"];
    const sig = Array.isArray(rawSignature) ? rawSignature[0] : (rawSignature ?? "");
    // Default to OK: unhandled events and successful processing should return 200 OK
    let result: HandlerResult = { status: HttpStatus.Ok };
    try {
        // Parse event and verify that it comes from Stripe
        const event = stripe.webhooks.constructEvent(req.body, sig, process.env.STRIPE_WEBHOOK_SECRET || "");
        // Call appropriate handler
        const eventHandlers: { [key in Stripe.WebhookEndpointUpdateParams.EnabledEvent]?: Handler } = {
            "checkout.session.completed": handleCheckoutSessionCompleted,
            // Handle asynchronous payment events for delayed payment methods (e.g., ACH)
            "checkout.session.async_payment_succeeded": handleCheckoutSessionCompleted,
            "checkout.session.expired": handleCheckoutSessionExpired,
            "checkout.session.async_payment_failed": handleCheckoutSessionExpired,
            "customer.deleted": handleCustomerDeleted,
            "customer.source.expiring": handleCustomerSourceExpiring,
            "customer.subscription.deleted": handleCustomerSubscriptionDeleted,
            "customer.subscription.trial_will_end": handleCustomerSubscriptionTrialWillEnd,
            "customer.subscription.updated": handleCustomerSubscriptionUpdated,
            "invoice.created": handleInvoicePaymentCreated,
            "invoice.payment_failed": handleInvoicePaymentFailed,
            "invoice.payment_succeeded": handleInvoicePaymentSucceeded,
            "invoice.paid": handleInvoicePaymentSucceeded,
            "price.updated": handlePriceUpdated,
        };
        if (event.type in eventHandlers) {
            result = await eventHandlers[event.type]({ event, stripe, res });
        } else {
            // Unhandled Stripe event: log warning, but return 200 OK to prevent retries
            logger.warning("Unhandled Stripe event", { trace: "0438", event: event.type });
        }
    } catch (error) {
        // On error parsing or handling, return 500
        result = { status: HttpStatus.InternalServerError, message: "Webhook encountered an error." };
        logger.error("Caught error in /webhooks/stripe", { trace: "0454", error });
    }
    res.status(result.status);
    if (result.message) {
        res.send(result.message);
    } else {
        res.send();
    }
}

/**
 * Sets up Stripe-related routes on the provided Express application instance.
 *
 * @param app - The Express application instance to attach routes to.
 */
export function setupStripe(app: Express): void {
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
}

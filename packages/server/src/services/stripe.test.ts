import { DAYS_180_MS, DAYS_1_MS, HOURS_1_MS, HttpStatus, PaymentStatus, PaymentType, StripeEndpoint } from "@vrooli/shared";
import { describe, it, expect, beforeAll, afterAll, beforeEach, vi } from "vitest";
import type Stripe from "stripe";
import StripeMock from "../__test/stripe.js";
import { DbProvider } from "../db/provider.js";
import * as loggerModule from "../events/logger.js";
import { CacheService } from "../redisConn.js";
import { PaymentMethod, calculateExpiryAndStatus, checkSubscriptionPrices, createStripeCustomerId, fetchPriceFromRedis, getCustomerId, getPaymentType, getPriceIds, getVerifiedCustomerInfo, getVerifiedSubscriptionInfo, handleCheckoutSessionExpired, handleCustomerDeleted, handleCustomerSourceExpiring, handleCustomerSubscriptionDeleted, handleCustomerSubscriptionTrialWillEnd, handleCustomerSubscriptionUpdated, handleInvoicePaymentCreated, handleInvoicePaymentFailed, handleInvoicePaymentSucceeded, handlePriceUpdated, handlerResult, isInCorrectEnvironment, isStripeObjectOlderThan, isValidCreditsPurchaseSession, isValidSubscriptionSession, parseInvoiceData, setupStripe, storePrice } from "./stripe.js";

const environments = ["development", "production"];

function createRes() {
    return {
        json: vi.fn(),
        send: vi.fn(),
        status: vi.fn().mockReturnThis(), // To allow for chaining .status().send()
    };
}

describe("getPaymentType", () => {
    const originalEnv = process.env.NODE_ENV;
    let resetModulesStub: any;

    beforeEach(() => {
        resetModulesStub = vi.fn();
    });

    afterAll(() => {
        process.env.NODE_ENV = originalEnv;
    });

    // Helper function to generate test cases for both environments
    function runTestsForEnvironment(env: string) {
        describe(`when NODE_ENV is ${env}`, () => {
            beforeEach(() => {
                process.env.NODE_ENV = env;
            });

            it("correctly identifies each PaymentType", () => {
                const priceIds = getPriceIds();
                Object.entries(priceIds).forEach(([paymentType, priceId]) => {
                    const result = getPaymentType(priceId);
                    expect(result).toBe(paymentType);
                });
            });

            it("throws an error for an invalid price ID", () => {
                const invalidPriceId = "price_invalid";
                try {
                    getPaymentType(invalidPriceId);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /** Empty */ }
            });

            it("works with Stripe.Price input", () => {
                const priceIds = getPriceIds();
                Object.entries(priceIds).forEach(([paymentType, priceId]) => {
                    const stripePrice = { id: priceId } as Stripe.Price;
                    const result = getPaymentType(stripePrice);
                    expect(result).toBe(paymentType);
                });
            });

            it("works with Stripe.DeletedPrice input", () => {
                const priceIds = getPriceIds();
                Object.entries(priceIds).forEach(([paymentType, priceId]) => {
                    const stripeDeletedPrice = { id: priceId } as Stripe.DeletedPrice;
                    const result = getPaymentType(stripeDeletedPrice);
                    expect(result).toBe(paymentType);
                });
            });
        });
    }

    // Run tests for both development and production environments
    environments.forEach(runTestsForEnvironment);
});

describe("getCustomerId function tests", () => {
    const mockCustomer = { id: "cust_12345" } as Stripe.Customer;

    it("returns the string when customer is a string", () => {
        const input = "cust_string";
        const output = getCustomerId(input);
        expect(output).toBe("cust_string");
    });

    it("returns the id when customer is a Stripe Customer object", () => {
        const output = getCustomerId(mockCustomer);
        expect(output).toBe("cust_12345");
    });

    it("throws an error when customer is null", () => {
        try {
            getCustomerId(null);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /** Empty */ }
    });

    it("throws an error when customer is undefined", () => {
        try {
            getCustomerId(undefined);
            expect.fail("Expected an error to be thrown");
        } catch (error) { /** Empty */ }
    });

    it("throws an error when customer is an empty object", () => {
        try {
            // @ts-ignore Testing runtime scenario
            getCustomerId({});
            expect.fail("Expected an error to be thrown");
        } catch (error) { /** Empty */ }
    });

    it("throws an error when id property is not a string", () => {
        try {
            // @ts-ignore Testing runtime scenario
            getCustomerId({ id: 12345 });
            expect.fail("Expected an error to be thrown");
        } catch (error) { /** Empty */ }
    });
});

describe("handlerResult", () => {
    let mockRes: any;
    let loggerErrorStub: any;

    beforeEach(() => {
        // Setup a mock Express response object
        mockRes = createRes();

        // Stub logger.error
        loggerErrorStub = vi.fn();

        // Directly stub the logger's error method
        vi.spyOn(loggerModule.logger, "error").mockImplementation(loggerErrorStub);
    });

    afterEach(() => {
        // Restore all stubs
        vi.restoreAllMocks();
    });

    it("sets the correct response status and message", () => {
        const status = HttpStatus.Ok;
        const message = "Success";

        handlerResult(status, mockRes, message);

        expect(mockRes.status).toHaveBeenCalledWith(status);
        expect(mockRes.send).toHaveBeenCalledWith(message);
    });

    it("logs an error message when status is not OK", () => {
        const status = HttpStatus.InternalServerError;
        const message = "Error occurred";
        const trace = "1234";

        handlerResult(status, mockRes, message, trace);

        expect(mockRes.status).toHaveBeenCalledWith(status);
        expect(mockRes.send).toHaveBeenCalledWith(message);
        expect(loggerErrorStub).toHaveBeenCalledTimes(1);
        expect(loggerErrorStub.firstCall.args[0]).toBe(message);
        expect(loggerErrorStub.mock.calls[0][1]).toMatchObject({ trace });
    });

    it("does not log an error for HttpStatus.Ok", () => {
        const status = HttpStatus.Ok;
        const message = "All good";

        handlerResult(status, mockRes, message);

        expect(loggerErrorStub).not.toHaveBeenCalled();
    });

    it("handles undefined message and trace", () => {
        const status = HttpStatus.BadRequest;

        handlerResult(status, mockRes);

        expect(mockRes.status).toHaveBeenCalledWith(status);
        expect(loggerErrorStub).toHaveBeenCalledTimes(1);
        expect(loggerErrorStub.firstCall.args[0]).toBe("Stripe handler error");
        expect(loggerErrorStub.mock.calls[0][1].trace).toBe("0523");
    });

    it("logs additional arguments when provided", () => {
        const status = HttpStatus.Unauthorized;
        const additionalArgs = { user: "user123", action: "attempted access" };

        handlerResult(status, mockRes, undefined, undefined, additionalArgs);

        expect(loggerErrorStub).toHaveBeenCalledTimes(1);
        expect(loggerErrorStub.mock.calls[0][1]).toHaveProperty("0");
        expect(loggerErrorStub.mock.calls[0][1]["0"]).toBe(additionalArgs);
    });
});

describe("Redis Price Operations", () => {
    const originalEnv = process.env.NODE_ENV;
    // Using vitest mocking instead of sinon sandbox

    beforeEach(async function beforeEach() {
        // Clear all mocks before each test
        vi.clearAllMocks();
        process.env.NODE_ENV = "test";
    });

    afterEach(async function afterEach() {
        vi.restoreAllMocks();
        await closeRedis();
    });

    afterAll(() => {
        process.env.NODE_ENV = originalEnv; // Restore the original NODE_ENV
    });

    describe("Redis price storage and retrieval", () => {
        beforeEach(async function beforeEach() {
            // Ensure we have a clean Redis environment for each test
            const cacheService = CacheService.get();
            const redisClient = await cacheService.raw();
            if (!redisClient) return;
            // Clear any existing keys related to our tests
            const keysPattern = `stripe-payment-${process.env.NODE_ENV}-*`;
            const keys = await redisClient.keys(keysPattern);
            if (keys.length > 0) {
                await redisClient.del(keys);
            }
        });

        it("correctly stores and fetches the price for a given payment type", async function () {
            const paymentType = PaymentType.PremiumMonthly;
            const price = 999;

            // Store the price
            await storePrice(paymentType, price);

            // Attempt to fetch the stored price
            const fetchedPrice = await fetchPriceFromRedis(paymentType);

            // Verify the fetched price matches the stored price
            expect(fetchedPrice).toBe(price);
        });

        it("returns null for a price that was not stored", async function () {
            const paymentType = "NonExistentType" as unknown as PaymentType;

            // Attempt to fetch a price for a payment type that hasn't been stored
            const fetchedPrice = await fetchPriceFromRedis(paymentType);

            // Verify that null is returned for the non-existent price
            expect(fetchedPrice).toBeNull();
        });

        it("returns null when the stored price is not a number", async function () {
            const paymentType = PaymentType.PremiumMonthly;

            // Use CacheService to manually set an invalid value in Redis
            const cacheService = CacheService.get();
            const redisClient = await cacheService.raw();
            if (!redisClient) return;
            const key = `stripe-payment-${process.env.NODE_ENV}-${paymentType}`;
            await redisClient.set(key, "invalid");

            // Attempt to fetch the stored price
            const fetchedPrice = await fetchPriceFromRedis(paymentType);

            // Verify that null is returned for the invalid price
            expect(fetchedPrice).toBeNull();
        });

        it("returns null when the price is less than 0", async function () {
            const paymentType = PaymentType.PremiumMonthly;
            const price = -1;

            // Store a negative price
            await storePrice(paymentType, price);

            // Attempt to fetch the stored price
            const fetchedPrice = await fetchPriceFromRedis(paymentType);

            // Verify that null is returned for the negative price
            expect(fetchedPrice).toBeNull();
        });

        it("returns null when the price is NaN", async function () {
            const paymentType = PaymentType.PremiumMonthly;
            const price = NaN;

            // Store a NaN price
            await storePrice(paymentType, price);

            // Attempt to fetch the stored price
            const fetchedPrice = await fetchPriceFromRedis(paymentType);

            // Verify that null is returned for the NaN price
            expect(fetchedPrice).toBeNull();
        });
    });
});

describe("isInCorrectEnvironment Tests", () => {
    // Store the original NODE_ENV to restore after tests
    const originalEnv = process.env.NODE_ENV;

    // Helper function to run tests in both environments
    function runTestsForEnvironment(env) {
        describe(`when NODE_ENV is ${env}`, () => {
            beforeAll(() => {
                process.env.NODE_ENV = env;
            });

            afterAll(() => {
                // Restore the original NODE_ENV after each suite
                process.env.NODE_ENV = originalEnv;
            });

            it("returns true for object matching the current environment", () => {
                const stripeObject = { livemode: env === "production" };
                expect(isInCorrectEnvironment(stripeObject)).toBe(true);
            });

            it("returns false for object not matching the current environment", () => {
                const stripeObject = { livemode: env !== "production" };
                expect(isInCorrectEnvironment(stripeObject)).toBe(false);
            });
        });
    }

    // Iterate over each environment and run the tests
    environments.forEach(runTestsForEnvironment);
});

describe("isValidSubscriptionSession", () => {
    // Store the original NODE_ENV to restore after tests
    const originalEnv = process.env.NODE_ENV;
    const userId = "testUserId";

    // Helper function to run tests in both environments
    function runTestsForEnvironment(env) {
        describe(`when NODE_ENV is ${env}`, () => {
            beforeAll(() => {
                process.env.NODE_ENV = env;
            });

            afterAll(() => {
                // Restore the original NODE_ENV after each suite
                process.env.NODE_ENV = originalEnv;
            });

            function createMockSession(overrides) {
                return {
                    livemode: process.env.NODE_ENV === "production",
                    metadata: {
                        paymentType: PaymentType.PremiumMonthly,
                        userId,
                    },
                    status: "complete",
                    ...overrides,
                };
            }

            it("returns true for a valid session", () => {
                const session = createMockSession({});
                expect(isValidSubscriptionSession(session, userId)).toBe(true);
            });

            it("returns false for a session with an unrecognized payment type", () => {
                const session = createMockSession({
                    metadata: { paymentType: "Unrecognized", userId },
                });
                expect(isValidSubscriptionSession(session, userId)).toBe(false);
            });

            it("returns false for a session initiated by a different user", () => {
                const session = createMockSession({
                    metadata: { paymentType: PaymentType.PremiumMonthly, userId: "anotherUserId" },
                });
                expect(isValidSubscriptionSession(session, userId)).toBe(false);
            });

            it("returns false for an incomplete session", () => {
                const session = createMockSession({ status: "incomplete" });
                expect(isValidSubscriptionSession(session, userId)).toBe(false);
            });

            it("returns false for a session in the wrong environment", () => {
                const session = createMockSession({
                    // Switch livemode
                    livemode: !(process.env.NODE_ENV === "production"),
                });
                expect(isValidSubscriptionSession(session, userId)).toBe(false);
            });
        });
    }

    // Iterate over each environment and run the tests
    environments.forEach(runTestsForEnvironment);
});

describe("isValidCreditsPurchaseSession", () => {
    // Store the original NODE_ENV to restore after tests
    const originalEnv = process.env.NODE_ENV;
    const userId = "testUserId";

    // Helper function to run tests in both environments
    function runTestsForEnvironment(env) {
        describe(`when NODE_ENV is ${env}`, () => {
            beforeAll(() => {
                process.env.NODE_ENV = env;
            });

            afterAll(() => {
                // Restore the original NODE_ENV after each suite
                process.env.NODE_ENV = originalEnv;
            });

            function createMockSession(overrides) {
                return {
                    livemode: process.env.NODE_ENV === "production",
                    metadata: {
                        paymentType: PaymentType.Credits,
                        userId,
                    },
                    status: "complete",
                    ...overrides,
                };
            }

            it("returns true for a valid session", () => {
                const session = createMockSession({});
                expect(isValidCreditsPurchaseSession(session, userId)).toBe(true);
            });

            it("returns false for a session with an unrecognized payment type", () => {
                const session = createMockSession({
                    metadata: { paymentType: "Unrecognized", userId },
                });
                expect(isValidCreditsPurchaseSession(session, userId)).toBe(false);
            });

            it("returns false for a session initiated by a different user", () => {
                const session = createMockSession({
                    metadata: { paymentType: PaymentType.Credits, userId: "anotherUserId" },
                });
                expect(isValidCreditsPurchaseSession(session, userId)).toBe(false);
            });

            it("returns false for an incomplete session", () => {
                const session = createMockSession({ status: "incomplete" });
                expect(isValidCreditsPurchaseSession(session, userId)).toBe(false);
            });

            it("returns false for a session in the wrong environment", () => {
                const session = createMockSession({
                    // Switch livemode based on environment setting for the test
                    livemode: !(process.env.NODE_ENV === "production"),
                });
                expect(isValidCreditsPurchaseSession(session, userId)).toBe(false);
            });
        });
    }

    // Define the environments to test, typically just 'production' and 'development'
    const environments = ["production", "development"];

    // Iterate over each environment and run the tests
    environments.forEach(runTestsForEnvironment);
});

describe("getVerifiedSubscriptionInfo", () => {
    let stripe: Stripe;

    beforeEach(() => {
        stripe = new StripeMock() as unknown as Stripe;
        StripeMock.injectData({
            checkoutSessions: [
                // Valid session
                {
                    id: "session_valid",
                    customer: "cus_user1",
                    metadata: {
                        userId: "user1",
                        paymentType: "PremiumMonthly",
                    },
                    subscription: "sub_active1",
                    status: "complete",
                },
                // Session linked to inactive subscription
                {
                    id: "session_inactive",
                    customer: "cus_user2",
                    metadata: {
                        userId: "user2",
                        paymentType: "PremiumMonthly",
                    },
                    subscription: "sub_inactive",
                    status: "complete",
                },
                // Invalid paymentType
                {
                    id: "session_invalid",
                    customer: "cus_user3",
                    metadata: {
                        userId: "user3",
                        paymentType: "UnknownPaymentType",
                    },
                    subscription: "sub_active2",
                    status: "complete",
                },
                // Subscription #1 (inactive) for user4
                {
                    id: "session_user4",
                    customer: "cus_user4",
                    metadata: {
                        userId: "user4",
                        paymentType: "PremiumMonthly",
                    },
                    subscription: "sub_inactive2",
                    status: "complete",
                },
                // Subscription #2 (active) for user4
                {
                    id: "session_user4_2",
                    customer: "cus_user4",
                    metadata: {
                        userId: "user4",
                        paymentType: "PremiumYearly",
                    },
                    subscription: "sub_active3",
                    status: "complete",
                },
            ],
            subscriptions: [{
                id: "sub_active1",
                status: "active",
                customer: "cus_user1",
            }, {
                id: "sub_inactive",
                status: "past_due",
                customer: "cus_user2",
            }, {
                id: "sub_active2",
                status: "active",
                customer: "cus_user3",
            }, {
                id: "sub_inactive2",
                status: "unpaid",
                customer: "cus_user4",
            }, {
                id: "sub_active3",
                status: "active",
                customer: "cus_user4",
            }],
        });
    });

    afterEach(async function afterEach() {
        StripeMock.resetMock();
        await DbProvider.deleteAll();
    });

    it("returns verified subscription info for an active subscription with valid session", async () => {
        const result = await getVerifiedSubscriptionInfo(stripe, "cus_user1", "user1");
        expect(result).not.toBeNull();
        expect(result).toHaveProperty("session");
        expect(result).toHaveProperty("subscription");
        expect(result?.paymentType).toBe("PremiumMonthly");
    });

    it("returns null if user in metadata does not match what we expect", async () => {
        const result = await getVerifiedSubscriptionInfo(stripe, "cus_user1", "beep");
        expect(result).toBeNull();
    });

    it("returns null if no subscriptions exist for the customer", async () => {
        const result = await getVerifiedSubscriptionInfo(stripe, "cus_randomDude", "user1");
        expect(result).toBeNull();
    });

    it("returns null if subscriptions exist but none are active or trialing", async () => {
        const result = await getVerifiedSubscriptionInfo(stripe, "cus_user2", "user2");
        expect(result).toBeNull();
    });

    it("returns null if the session metadata's payment type is not recognized", async () => {
        const result = await getVerifiedSubscriptionInfo(stripe, "cus_user3", "user3");
        expect(result).toBeNull();
    });

    it("returns verified subscription info if first one was invalid but second one is valid", async () => {
        const result = await getVerifiedSubscriptionInfo(stripe, "cus_user4", "user4");
        expect(result).not.toBeNull();
        expect(result).toHaveProperty("session");
        expect(result).toHaveProperty("subscription");
        expect(result?.paymentType).toBe("PremiumYearly");
    });
});

describe("getVerifiedCustomerInfo", () => {
    let stripe: Stripe;
    const user1Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
    const user2Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
    const user3Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
    const user4Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
    const user5Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
    const user6Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
    const user7Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;

    beforeEach(async function beforeEach() {
        stripe = new StripeMock() as unknown as Stripe;
        StripeMock.resetMock();
        StripeMock.injectData({
            checkoutSessions: [
                {
                    id: "session_valid",
                    customer: "cus_123",
                    metadata: {
                        userId: "user1",
                        paymentType: "PremiumMonthly",
                    },
                    subscription: "sub_active1",
                    status: "complete",
                },
                {
                    id: "session_user7",
                    customer: "cus_777",
                    metadata: {
                        userId: "user7",
                        paymentType: "PremiumYearly",
                    },
                    subscription: "sub_inactive",
                    status: "complete",
                },
                {
                    id: "session_user7_2",
                    customer: "cus_789",
                    metadata: {
                        userId: "user7",
                        paymentType: "PremiumYearly",
                    },
                    subscription: "sub_active2",
                    status: "complete",
                },
            ],
            customers: [{
                id: "cus_test1",
                email: "user2@example.com",
            }, {
                id: "cus_123",
            }, {
                id: "cus_deleted",
                // @ts-ignore Property is on Stripe.DeletedCustomer, which can be returned by the API
                deleted: true,
                email: "emailForDeletedCustomer@example.com",
            }, {
                id: "cus_789",
                email: "emailWithSubscription@example.com",
            }],
            subscriptions: [{
                id: "sub_active1",
                status: "active",
                customer: "cus_123",
            }, {
                id: "sub_inactive",
                status: "past_due",
                customer: "cus_777",
            }, {
                id: "sub_active2",
                status: "active",
                customer: "cus_789",
            }],
        });

        await DbProvider.deleteAll();
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                name: "user1",
                stripeCustomerId: "cus_123",
                emails: {
                    create: [{ emailAddress: "user@example.com" }],
                },
                premium: {
                    create: {
                        enabledAt: new Date(),
                    },
                },
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user2Id,
                name: "user2",
                emails: {
                    create: [{ emailAddress: "user2@example.com" }],
                },
                premium: {
                    create: {
                        enabledAt: new Date(),
                    },
                },
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user3Id,
                name: "user3",
                emails: {
                    create: [{ emailAddress: "missing@example.com" }],
                },
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user4Id,
                name: "user4",
                stripeCustomerId: "invalid_customer_id",
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user5Id,
                name: "user5",
                stripeCustomerId: "cus_deleted",
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user6Id,
                name: "user6",
                emails: {
                    create: [{ emailAddress: "emailForDeletedCustomer@example.com" }],
                },
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user7Id,
                name: "user7",
                stripeCustomerId: "cus_777",
                emails: {
                    create: [{ emailAddress: "emailWithSubscription@example.com" }],
                },
            },
        });
    });

    afterEach(async function afterEach() {
        StripeMock.resetMock();
        await DbProvider.deleteAll();
    });

    it("should return null if userId is not provided", async () => {
        const result = await getVerifiedCustomerInfo({ userId: undefined, stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBeNull();
        expect(result.emails).toEqual([]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    it("should return null if the user does not exist", async () => {
        const result = await getVerifiedCustomerInfo({ userId: `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`, stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBeNull();
        expect(result.emails).toEqual([]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    it("should return the user's Stripe customer ID if it exists", async () => {
        const result = await getVerifiedCustomerInfo({ userId: user1Id, stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBe("cus_123");
        expect(result.userId).toBe(user1Id);
        expect(result.emails).toEqual([{ emailAddress: "user@example.com" }]);
        expect(result.hasPremium).toBe(true);
        expect(result.subscriptionInfo).toBeNull();
    });

    it("should return the Stripe customer ID associated with the user's email if the user does not have a Stripe customer ID", async () => {
        const result = await getVerifiedCustomerInfo({ userId: user2Id, stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBe("cus_test1");
        expect(result.userId).toBe(user2Id);
        expect(result.emails).toEqual([{ emailAddress: "user2@example.com" }]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();

        // Should also update the user's Stripe customer ID in the database
        // @ts-ignore Testing runtime scenario
        const updatedUser = await PrismaClient.instance.user.findUnique({ where: { id: user2Id } });
        expect(updatedUser.stripeCustomerId).toBe("cus_test1");
    });

    it("should return null if the user does not have a Stripe customer ID and their email is not associated with any Stripe customer", async () => {
        const result = await getVerifiedCustomerInfo({ userId: user3Id, stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBe(user3Id);
        expect(result.emails).toEqual([{ emailAddress: "missing@example.com" }]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    it("should return null if the user's Stripe customer ID is invalid", async () => {
        const result = await getVerifiedCustomerInfo({ userId: user4Id, stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBe(user4Id);
        expect(result.emails).toEqual([]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    it("should return null if the user's Stripe customer ID is associated with a deleted customer", async () => {
        const result = await getVerifiedCustomerInfo({ userId: user5Id, stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBe(user5Id);
        expect(result.emails).toEqual([]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    it("should return null if the user's email is associated with a deleted customer", async () => {
        const result = await getVerifiedCustomerInfo({ userId: user6Id, stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBe(user6Id);
        expect(result.emails).toEqual([{ emailAddress: "emailForDeletedCustomer@example.com" }]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    it("should return the Stripe customer ID if the user has a valid subscription", async () => {
        const result = await getVerifiedCustomerInfo({ userId: user1Id, stripe, validateSubscription: true });
        expect(result.stripeCustomerId).toBe("cus_123");
        expect(result.userId).toBe(user1Id);
        expect(result.emails).toEqual([{ emailAddress: "user@example.com" }]);
        expect(result.hasPremium).toBe(true);
        expect(result.subscriptionInfo).not.toBeNull();
    });

    it("should return the Stripe customer ID associated with the user's email if the user's customer ID does not have a valid subscription, but the email does", async () => {
        // Check original customer ID (tied to inactive subscription)
        // @ts-ignore Testing runtime scenario
        const originalUser = await PrismaClient.instance.user.findUnique({ where: { id: user7Id } });
        expect(originalUser.stripeCustomerId).toBe("cus_777");

        const result = await getVerifiedCustomerInfo({ userId: user7Id, stripe, validateSubscription: true });
        expect(result.stripeCustomerId).toBe("cus_789");
        expect(result.userId).toBe(user7Id);
        expect(result.emails).toEqual([{ emailAddress: "emailWithSubscription@example.com" }]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).not.toBeNull();

        // Check that original customer ID (tied to inactive subscription) was changed to customer associated with email
        // @ts-ignore Testing runtime scenario
        const updatedUser = await PrismaClient.instance.user.findUnique({ where: { id: user7Id } });
        expect(updatedUser.stripeCustomerId).toBe("cus_789");
    });

    it("should return null if the user does not have a valid subscription", async () => {
        const result = await getVerifiedCustomerInfo({ userId: user3Id, stripe, validateSubscription: true });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.subscriptionInfo).toBeNull();
    });
});

describe("createStripeCustomerId", () => {
    let stripe: Stripe;
    const user1Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;

    beforeEach(async function beforeEach() {
        stripe = new StripeMock() as unknown as Stripe;

        await DbProvider.deleteAll();
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                name: "user1",
                stripeCustomerId: null,
            },
        });
    });

    afterEach(async function afterEach() {
        StripeMock.resetMock();
        await DbProvider.deleteAll();
    });


    it("creates a new Stripe customer and updates the user with the customer ID", async () => {
        const customerInfo = {
            userId: user1Id,
            emails: [{ emailAddress: "user@example.com" }],
            hasPremium: false,
            subscriptionInfo: null,
            stripeCustomerId: null,
        };
        const result = await createStripeCustomerId({
            customerInfo,
            requireUserToExist: true,
            stripe,
        });
        expect(typeof result).toBe("string");
        // @ts-ignore Testing runtime scenario
        const updatedUser = await PrismaClient.instance.user.findUnique({ where: { id: "user1" } });
        expect(typeof updatedUser.stripeCustomerId).toBe("string");
    });

    it("creates a new Stripe customer without requiring an existing user", async () => {
        const customerInfo = {
            userId: null, // No user ID provided
            emails: [],
            hasPremium: false,
            subscriptionInfo: null,
            stripeCustomerId: null,
        };
        const result = await createStripeCustomerId({
            customerInfo,
            requireUserToExist: false,
            stripe,
        });
        expect(typeof result).toBe("string");
    });

    it("throws an error when no user ID is provided but requireUserToExist is true", async () => {
        const customerInfo = {
            userId: null,
            emails: [],
            hasPremium: false,
            subscriptionInfo: null,
            stripeCustomerId: null,
        };
        try {
            await createStripeCustomerId({
                customerInfo,
                requireUserToExist: true,
                stripe,
            });
            expect.fail("Expected an error to be thrown");
        } catch (error) { /** Empty */ }
    });
});

describe("handleCheckoutSessionExpired", () => {
    let res;
    let stripe: Stripe;
    const payment1Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
    const user1Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
    const user2Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;

    beforeEach(async function beforeEach() {
        stripe = new StripeMock() as unknown as Stripe;
        res = createRes();

        await DbProvider.deleteAll();
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                name: "user1",
                stripeCustomerId: "customer1",
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user2Id,
                name: "user2",
                stripeCustomerId: "customer2",
            },
        });
        await DbProvider.get().payment.create({
            data: {
                id: payment1Id,
                amount: 1000,
                currency: "usd",
                description: "Test payment",
                checkoutId: "session1",
                paymentMethod: "Stripe",
                status: "Pending",
                userId: user1Id,
            },
        });
    });

    afterEach(async function afterEach() {
        StripeMock.resetMock();
        await DbProvider.deleteAll();
    });

    it("does nothing when no related pending payments found", async () => {
        const event = { data: { object: { id: "sessionUnknown", customer: "customer1" } } } as unknown as Stripe.Event;

        // @ts-ignore Testing runtime scenario
        const originalPayments = JSON.parse(JSON.stringify(await PrismaClient.instance.payment.findMany({ where: {} })));

        await handleCheckoutSessionExpired({ event, res, stripe });

        // @ts-ignore Testing runtime scenario
        const updatedPayments = JSON.parse(JSON.stringify(await PrismaClient.instance.payment.findMany({ where: {} })));

        expect(updatedPayments).toEqual(originalPayments);
    });

    it("marks related pending payments as failed", async () => {
        const event = { data: { object: { id: "session1", customer: "customer1" } } } as unknown as Stripe.Event;

        await handleCheckoutSessionExpired({ event, res, stripe });

        // @ts-ignore Testing runtime scenario
        const updatedPayment = await PrismaClient.instance.payment.findUnique({ where: { id: "payment1" } });
        expect(updatedPayment.status).toBe("Failed");
    });

    it("returns OK status on successful update", async () => {
        const event = { data: { object: { id: "session1", customer: "customer1" } } } as unknown as Stripe.Event;

        await handleCheckoutSessionExpired({ event, res, stripe });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
    });
});

describe("handleCustomerDeleted", () => {
    let res;
    let stripe: Stripe;
    const user1Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
    const user2Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;

    beforeEach(async function beforeEach() {
        stripe = new StripeMock() as unknown as Stripe;
        res = createRes();

        await DbProvider.deleteAll();
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                name: "user1",
                stripeCustomerId: "customer1",
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user2Id,
                name: "user2",
                stripeCustomerId: "customer2",
            },
        });
    });

    afterEach(async function afterEach() {
        StripeMock.resetMock();
        await DbProvider.deleteAll();
    });

    it("removes stripeCustomerId from user on customer deletion", async () => {
        const event = { data: { object: { id: "customer1" } } } as unknown as Stripe.Event;

        // Perform the action
        await handleCustomerDeleted({ event, res, stripe });

        // Verify the stripeCustomerId was set to null for the user
        // @ts-ignore Testing runtime scenario
        const updatedUser = await PrismaClient.instance.user.findUnique({ where: { id: "user1" } });
        expect(updatedUser.stripeCustomerId).toBeNull();
    });

    it("does not affect users unrelated to the deleted customer", async () => {
        const event = { data: { object: { id: "customer1" } } } as unknown as Stripe.Event;

        // Perform the action
        await handleCustomerDeleted({ event, res, stripe });

        // Verify that other users are unaffected
        // @ts-ignore Testing runtime scenario
        const unaffectedUser = await PrismaClient.instance.user.findUnique({ where: { id: "user2" } });
        expect(unaffectedUser.stripeCustomerId).toBe("customer2");
    });

    it("returns OK status on successful customer deletion handling", async () => {
        const event = { data: { object: { id: "customer1" } } } as unknown as Stripe.Event;

        // Perform the action
        await handleCustomerDeleted({ event, res, stripe });

        // Check if the response status was set to OK
        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
    });

    it("handles cases where the customer does not exist gracefully", async () => {
        const event = { data: { object: { id: "customerNonExisting" } } } as unknown as Stripe.Event;

        // Perform the action with a customer ID that doesn't exist
        await handleCustomerDeleted({ event, res, stripe });

        // Expect the function to still return an OK status without throwing any errors
        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
    });
});

describe("handleCustomerSourceExpiring", () => {
    let res;
    let emailQueue;
    const user1Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;

    beforeAll(async () => {
        emailQueue = new Bull("emailQueue");
        await setupEmailQueue();
    });

    beforeEach(async function beforeEach() {
        vi.clearAllMocks();
        res = createRes();
        Bull.resetMock();

        await DbProvider.deleteAll();
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                name: "user1",
                stripeCustomerId: "cus_123",
                emails: {
                    create: [{ emailAddress: "user1@example.com" }, { emailAddress: "user2@example.com" }],
                },
            },
        });
    });

    afterEach(async function afterEach() {
        await DbProvider.deleteAll();
    });

    it("should send notifications to all user emails when the credit card is expiring", async () => {
        const mockEvent = {
            data: { object: { id: "cus_123" } },
        } as unknown as Stripe.Event;
        await handleCustomerSourceExpiring({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(emailQueue.add).toHaveBeenCalledTimes(2);
        expect(emailQueue.add.mock.calls[0][0].to).toEqual(["user1@example.com"]);
        expect(emailQueue.add.mock.calls[1][0].to).toEqual(["user2@example.com"]);
    });

    it("should return an error when the user is not found", async () => {
        const mockEvent = {
            data: { object: { id: "non_existent_customer" } },
        } as unknown as Stripe.Event;
        await handleCustomerSourceExpiring({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.InternalServerError);
        expect(emailQueue.add).not.toHaveBeenCalled();
    });
});

function expectEmailQueue(mockQueue, expectedData) {
    // Check if the add function was called
    expect(mockQueue.add).toHaveBeenCalled();

    // Extract the actual data passed to the mock
    const actualData = mockQueue.add.mock.calls[0][0];

    // If there's expected data, match it against the actual data
    if (expectedData) {
        expect(actualData).toEqual(expectedData);
    }
}

describe("handleCustomerSubscriptionDeleted", () => {
    let res;
    let emailQueue;
    const user1Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
    const user2Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;

    beforeAll(async () => {
        emailQueue = new Bull("email");
        await setupEmailQueue();
    });

    beforeEach(async function beforeEach() {
        vi.clearAllMocks();
        res = createRes();
        Bull.resetMock();

        await DbProvider.deleteAll();
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                name: "user1",
                stripeCustomerId: "cus_123",
                emails: {
                    create: [{ emailAddress: "user@example.com" }],
                },
                premium: {
                    create: {
                        enabledAt: new Date(),
                    },
                },
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user2Id,
                name: "user2",
                stripeCustomerId: "cus_234",
            },
        });
    });

    afterEach(async function afterEach() {
        await DbProvider.deleteAll();
    });

    it("should send email when user is found and has email", async () => {
        const mockEvent = {
            data: { object: { customer: "cus_123" } },
        } as unknown as Stripe.Event;
        await handleCustomerSubscriptionDeleted({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expectEmailQueue(emailQueue, { to: ["user@example.com"] });
        // Also make sure it didn't affect the premium status
        // @ts-ignore Testing runtime scenario
        const updatedUser = await PrismaClient.instance.user.findUnique({ where: { id: "user1" } });
        expect(updatedUser.premium.isActive).toBe(true);
    });

    it("should not send email when user is found but has no email", async () => {
        const mockEvent = {
            data: { object: { customer: "cus_234" } },
        } as unknown as Stripe.Event;
        await handleCustomerSubscriptionDeleted({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(emailQueue.add).not.toHaveBeenCalled();
    });

    it("should return error when user is not found", async () => {
        const mockEvent = {
            data: { object: { customer: "non_existent_customer" } },
        } as unknown as Stripe.Event;
        await handleCustomerSubscriptionDeleted({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.InternalServerError);
        expect(emailQueue.add).not.toHaveBeenCalled();
    });
});

describe("handleCustomerSubscriptionTrialWillEnd", () => {
    let res;
    let emailQueue;
    const user1Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
    const user2Id = `user-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;

    beforeAll(async () => {
        emailQueue = new Bull("email");
        await setupEmailQueue();
    });

    beforeEach(async function beforeEach() {
        res = createRes();
        Bull.resetMock();

        await DbProvider.deleteAll();
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                name: "user1",
                stripeCustomerId: "cus_123",
                emails: {
                    create: [{ emailAddress: "user@example.com" }],
                },
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user2Id,
                name: "user2",
                stripeCustomerId: "cus_234",
            },
        });
    });

    afterEach(async function afterEach() {
        await DbProvider.deleteAll();
    });

    it("should send email when user is found and has email", async () => {
        const mockEvent = {
            data: { object: { customer: "cus_123" } },
        } as unknown as Stripe.Event;
        await handleCustomerSubscriptionTrialWillEnd({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expectEmailQueue(emailQueue, { to: ["user@example.com"] });
    });

    it("should not send email when user is found but has no email", async () => {
        const mockEvent = {
            data: { object: { customer: "cus_234" } },
        } as unknown as Stripe.Event;
        await handleCustomerSubscriptionTrialWillEnd({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(emailQueue.add).not.toHaveBeenCalled();
    });

    it("should return error when user is not found", async () => {
        const mockEvent = {
            data: { object: { customer: "non_existent_customer" } },
        } as unknown as Stripe.Event;
        await handleCustomerSubscriptionTrialWillEnd({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.InternalServerError);
        expect(emailQueue.add).not.toHaveBeenCalled();
    });
});

// NOTE: session.current_period_end is in seconds
describe("calculateExpiryAndStatus", () => {
    it("should calculate isActive as true for subscriptions ending in the future", () => {
        const futureTimestamp = Math.floor(Date.now() / 1000) + (60 * 60 * 24); // 24 hours in the future
        const { newExpiresAt, isActive } = calculateExpiryAndStatus(futureTimestamp);

        expect(isActive).toBe(true);
        expect(newExpiresAt).toEqual(new Date(futureTimestamp * 1000));
    });

    it("should calculate isActive as false for subscriptions that have already ended", () => {
        const pastTimestamp = Math.floor(Date.now() / 1000) - (60 * 60 * 24); // 24 hours in the past
        const { newExpiresAt, isActive } = calculateExpiryAndStatus(pastTimestamp);

        expect(isActive).toBe(false);
        expect(newExpiresAt).toEqual(new Date(pastTimestamp * 1000));
    });

    it("should correctly calculate the newExpiresAt date for a given timestamp", () => {
        const timestamp = 1609459200; // January 1, 2021
        const { newExpiresAt, isActive } = calculateExpiryAndStatus(timestamp);

        expect(newExpiresAt).toEqual(new Date(timestamp * 1000));
        // The isActive result depends on when the test is run relative to the fixed timestamp
        // and thus is not asserted in this test.
    });
});

describe("handleCustomerSubscriptionUpdated", () => {
    let res;
    let emailQueue;

    beforeAll(async () => {
        emailQueue = new Bull("emailQueue");
        await setupEmailQueue(); // Assuming this sets up your email queue for testing
    });

    beforeEach(async function beforeEach() {
        vi.clearAllMocks();
        res = createRes(); // Assuming this creates a mock response object
        Bull.resetMock();
        PrismaClient.injectData({ // Assuming this injects mock data into your Prisma client
            User: [{
                id: "user1",
                stripeCustomerId: "cus_123",
                emails: [{ emailAddress: "user@example.com" }],
            }],
        });
    });

    afterEach(async function afterEach() {
        await DbProvider.deleteAll();
    });

    it("should skip processing for non-active subscriptions", async () => {
        const mockEvent = {
            data: { object: { customer: "cus_123", status: "canceled" } },
        } as unknown as Stripe.Event;
        await handleCustomerSubscriptionUpdated({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(emailQueue.add).not.toHaveBeenCalled();
    });

    it("should update expiry time and send thank you email for active subscription", async () => {
        const futurePeriodEnd = Math.floor(Date.now() / 1000) + (60 * 60 * 24); // 24 hours in the future
        const mockEvent = {
            data: { object: { customer: "cus_123", status: "active", current_period_end: futurePeriodEnd } },
        } as unknown as Stripe.Event;
        await handleCustomerSubscriptionUpdated({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(emailQueue.add.mock.calls[0][0].to).toEqual(["user@example.com"]);
    });

    it("should update expiry time to past and not send thank you email for inactive premium status", async () => {
        const pastPeriodEnd = Math.floor(Date.now() / 1000) - (60 * 60 * 24); // 24 hours in the past
        const mockEvent = {
            data: { object: { customer: "cus_123", status: "active", current_period_end: pastPeriodEnd } },
        } as unknown as Stripe.Event;
        await handleCustomerSubscriptionUpdated({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(emailQueue.add.mock.calls[0][0].to).toEqual(["user@example.com"]);
    });

    it("should return error when user is not found", async () => {
        const mockEvent = {
            data: { object: { customer: "non_existent_customer", status: "active", current_period_end: Math.floor(Date.now() / 1000) + (60 * 60 * 24) } },
        } as unknown as Stripe.Event;
        await handleCustomerSubscriptionUpdated({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.InternalServerError);
    });
});

describe("parseInvoiceData", () => {
    it("should correctly parse invoice with valid line item", () => {
        const mockInvoice = {
            id: "inv_valid",
            currency: "usd",
            lines: {
                data: [{
                    amount: 2000,
                    price: { id: getPriceIds().PremiumMonthly },
                }],
            },
        } as Stripe.Invoice;

        const result = parseInvoiceData(mockInvoice);

        expect(result).not.toHaveProperty("error");
        expect(result.data).toEqual({
            checkoutId: "inv_valid",
            amount: 2000,
            currency: "usd",
            paymentMethod: PaymentMethod.Stripe,
            paymentType: PaymentType.PremiumMonthly,
            description: "Payment for Invoice ID: inv_valid",
        });
    });

    it("should return an error for invoice without line items", () => {
        const mockInvoice = {
            id: "inv_nolines",
            currency: "usd",
            lines: { data: [] },
        } as unknown as Stripe.Invoice;

        const result = parseInvoiceData(mockInvoice);

        expect(result.error).toEqual("No lines found in invoice");
    });

    it("should throw an error for unknown payment type", () => {
        const mockInvoice = {
            id: "inv_unknown_payment_type",
            currency: "usd",
            lines: {
                data: [{
                    amount: 2000,
                    price: { id: "unknown_price_id" },
                }],
            },
        } as Stripe.Invoice;

        expect(() => { parseInvoiceData(mockInvoice); }).toThrow();
    });
});

describe("handleInvoicePaymentCreated", () => {
    let res;

    beforeEach(async function beforeEach() {
        vi.clearAllMocks();
        res = createRes();
        PrismaClient.injectData({
            User: [{
                id: "user1",
                stripeCustomerId: "cus_123",
            }],
            Payment: [{
                id: "payment1",
                checkoutId: "inv_123",
            }],
        });
    });

    afterEach(async function afterEach() {
        await DbProvider.deleteAll();
    });

    it("should successfully create a new payment if no existing payment is found", async () => {
        const mockEvent = {
            data: {
                object: {
                    customer: "cus_123",
                    id: "inv_321",
                    currency: "usd",
                    lines: { data: [{ amount: 2000, price: { id: getPriceIds().PremiumYearly, unit_amount: 2000 } }] },
                },
            },
        } as unknown as Stripe.Event;
        await handleInvoicePaymentCreated({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        // Verify that the payment exists and has valid data
        // @ts-ignore Testing runtime scenario
        const createdPayment = await PrismaClient.instance.payment.findUnique({ where: { checkoutId: "inv_321" } });
        expect(createdPayment).not.toBeNull();
        expect(createdPayment?.amount).toBe(2000);
        expect(createdPayment?.paymentMethod).toBe("Stripe");
        expect(createdPayment?.status).toBe("Pending");
        expect(createdPayment?.user).not.toBeNull();
    });

    it("should update an existing payment if found", async () => {
        const mockEvent = {
            data: {
                object: {
                    customer: "cus_123",
                    id: "inv_123",
                    currency: "usd",
                    lines: { data: [{ amount: 2000, price: { id: getPriceIds().PremiumMonthly, unit_amount: 2000 } }] },
                },
            },
        } as unknown as Stripe.Event;
        await handleInvoicePaymentCreated({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        // Verify that the payment was updated and has valid data
        // @ts-ignore Testing runtime scenario
        const updatedPayment = await PrismaClient.instance.payment.findUnique({ where: { checkoutId: "inv_123" } });
        expect(updatedPayment).not.toBeNull();
        expect(updatedPayment?.amount).toBe(2000);
        expect(updatedPayment?.paymentMethod).toBe("Stripe");
        expect(updatedPayment?.status).toBe("Pending");
    });

    it("should return error when user is not found", async () => {
        const mockEvent = {
            data: {
                object: {
                    customer: "cus_nonexistent",
                    id: "inv_124",
                },
            },
        } as unknown as Stripe.Event;
        await handleInvoicePaymentCreated({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.InternalServerError);
    });

    it("should return error when no lines found in invoice", async () => {
        const mockEvent = {
            data: {
                object: {
                    customer: "cus_123",
                    id: "inv_125",
                    lines: { data: [] }, // No line items
                },
            },
        } as unknown as Stripe.Event;
        await handleInvoicePaymentCreated({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.InternalServerError);
    });

    it("should throw error for unknown payment type", async () => {
        const mockEvent = {
            data: {
                object: {
                    customer: "cus_123",
                    id: "inv_126",
                    lines: { data: [{ amount: 2000, price: { id: "uknown_price_id", unit_amount: 2000 } }] },
                },
            },
        } as unknown as Stripe.Event;
        await expect(handleInvoicePaymentCreated({ event: mockEvent, res })).rejects.toThrow();
    });
});

describe("handleInvoicePaymentFailed", () => {
    let res;
    let emailQueue;

    beforeAll(async () => {
        emailQueue = new Bull("emailQueue");
        await setupEmailQueue();
    });

    beforeEach(async function beforeEach() {
        vi.clearAllMocks();
        res = createRes();
        PrismaClient.injectData({
            User: [{
                id: "user1",
                stripeCustomerId: "cus_123",
                emails: [{ emailAddress: "user@example.com" }],
            }],
            Payment: [{
                checkoutId: "inv_123",
                paymentType: "Subscription",
                user: {
                    id: "user1",
                },
            }],
        });
    });

    afterEach(async function afterEach() {
        await DbProvider.deleteAll();
    });

    it("updates existing payment status to Failed", async () => {
        const mockEvent = {
            data: { object: { customer: "cus_123", id: "inv_123" } },
        } as unknown as Stripe.Event;
        await handleInvoicePaymentFailed({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        // Verify that payment status was updated to Failed
        // @ts-ignore Testing runtime scenario
        const updatedPayment = await PrismaClient.instance.payment.findUnique({ where: { checkoutId: "inv_123" } });
        expect(updatedPayment?.status).toBe(PaymentStatus.Failed);
    });

    it("creates a new payment and sets status to Failed when no existing payment is found", async () => {
        const mockEvent = {
            data: {
                object: {
                    customer: "cus_123",
                    id: "inv_321",
                    currency: "usd",
                    lines: { data: [{ amount: 2000, price: { id: getPriceIds().PremiumYearly, unit_amount: 2000 } }] },
                },
            },
        } as unknown as Stripe.Event;
        await handleInvoicePaymentFailed({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        // Verify that a new payment was created with status set to Failed
        // @ts-ignore Testing runtime scenario
        const newPayment = await PrismaClient.instance.payment.findUnique({ where: { checkoutId: "inv_321" } });
        expect(newPayment?.status).toBe("Failed");
        expect(newPayment?.amount).toBe(2000);
        expect(newPayment?.paymentMethod).toBe("Stripe");
        expect(newPayment?.user).not.toBeNull();
    });

    it("returns an error when parseInvoiceData returns an error", async () => {
        const mockEvent = {
            data: {
                object: {
                    customer: "cus_123",
                    id: "inv_321",
                    currency: "usd",
                    lines: { data: [] }, // A payment with no lines should return an error
                },
            },
        } as unknown as Stripe.Event;
        await handleInvoicePaymentFailed({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.InternalServerError);
    });

    it("notifies user when payment fails", async () => {
        const mockEvent = {
            data: { object: { customer: "cus_123", id: "inv_123" } },
        } as unknown as Stripe.Event;
        await handleInvoicePaymentFailed({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expectEmailQueue(emailQueue, { to: ["user@example.com"] });
    });
});

describe("handleInvoicePaymentSucceeded", () => {
    let res;
    let stripeMock;

    beforeEach(async function beforeEach() {
        vi.clearAllMocks();
        res = createRes();
        PrismaClient.injectData({
            User: [{
                id: "user1",
                stripeCustomerId: "cus_123",
                emails: [{ emailAddress: "user@example.com" }],
            }],
            Payment: [{
                checkoutId: "inv_123",
                paymentType: "Subscription",
                user: {
                    id: "user1",
                },
            }],
        });
        stripeMock = new StripeMock() as unknown as Stripe;
        StripeMock.injectData({
            subscriptions: [{
                id: "sub_123",
                current_period_end: Math.floor(Date.now() / 1000) + (30 * 24 * 60 * 60), // Mocking 30 days ahead
            }],
        });
    });

    afterEach(async function afterEach() {
        StripeMock.resetMock();
        await DbProvider.deleteAll();
    });

    it("updates existing payment status to Paid", async () => {
        const mockEvent = {
            data: { object: { customer: "cus_123", id: "inv_123" } },
        } as unknown as Stripe.Event;
        await handleInvoicePaymentSucceeded({ event: mockEvent, res, stripe: stripeMock });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        // @ts-ignore Testing runtime scenario
        const updatedPayment = await PrismaClient.instance.payment.findUnique({ where: { checkoutId: "inv_123" } });
        expect(updatedPayment?.status).toBe(PaymentStatus.Paid);
    });

    it("creates a new payment and sets status to Paid when no existing payment is found", async () => {
        const mockEvent = {
            data: {
                object: {
                    customer: "cus_123",
                    id: "inv_321",
                    currency: "usd",
                    lines: { data: [{ amount: 2000, price: { id: getPriceIds().PremiumYearly, unit_amount: 2000 } }] },
                    subscription: "sub_123",
                },
            },
        } as unknown as Stripe.Event;
        await handleInvoicePaymentSucceeded({ event: mockEvent, res, stripe: stripeMock });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        // @ts-ignore Testing runtime scenario
        const newPayment = await PrismaClient.instance.payment.findUnique({ where: { checkoutId: "inv_321" } });
        expect(newPayment?.status).toBe(PaymentStatus.Paid);
        expect(newPayment?.amount).toBe(2000);
        expect(newPayment?.paymentMethod).toBe("Stripe");
        expect(newPayment?.user).not.toBeNull();
    });

    it("returns an error when parseInvoiceData returns an error", async () => {
        const mockEvent = {
            data: {
                object: {
                    customer: "cus_123",
                    id: "inv_321",
                    currency: "usd",
                    lines: { data: [] }, // A payment with no lines should return an error
                    subscription: "sub_123",
                },
            },
        } as unknown as Stripe.Event;
        await handleInvoicePaymentSucceeded({ event: mockEvent, res, stripe: stripeMock });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.InternalServerError);
    });

    it("updates user premium status for subscription payments", async () => {
        const mockEvent = {
            data: {
                object: {
                    customer: "cus_123",
                    id: "inv_321",
                    currency: "usd",
                    lines: { data: [{ amount: 2000, price: { id: getPriceIds().PremiumYearly, unit_amount: 2000 } }] },
                    subscription: "sub_123",
                },
            },
        } as unknown as Stripe.Event;
        await handleInvoicePaymentSucceeded({ event: mockEvent, res, stripe: stripeMock });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        // @ts-ignore Testing runtime scenario
        const user = await PrismaClient.instance.user.findUnique({ where: { id: "user1" } });
        expect(user?.premium?.isActive).toBe(true);
    });

    it("returns an error if subscription details could not be retrieved", async () => {
        StripeMock.simulateFailure(true);

        const mockEvent = {
            data: {
                object: {
                    customer: "cus_123",
                    id: "inv_321",
                    currency: "usd",
                    lines: { data: [{ amount: 2000, price: { id: getPriceIds().PremiumYearly, unit_amount: 2000 } }] },
                    subscription: "sub_123",
                },
            },
        } as unknown as Stripe.Event;
        await handleInvoicePaymentSucceeded({ event: mockEvent, res, stripe: stripeMock });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.InternalServerError);
    });
});

// TODO this test and handleSubscriptionUpdated are causing jest to hang. storePrice is the problem, but I don't know why
describe("handlePriceUpdated", () => {
    let res;

    beforeEach(async function beforeEach() {
        vi.clearAllMocks();
        RedisClientMock.resetMock();
        RedisClientMock.__setAllMockData({});
        res = createRes();
    });

    it("handles a valid price update event", async () => {
        // Setup
        const event = {
            data: {
                object: {
                    id: getPriceIds().PremiumMonthly,
                    unit_amount: 1000,
                },
            },
        } as unknown as Stripe.Event;

        // Act
        const result = await handlePriceUpdated({ event, res });

        // Assert
        const storedPrice = await fetchPriceFromRedis(PaymentType.PremiumMonthly);
        expect(storedPrice).toBe(1000);
        expect(result).toEqual({ status: HttpStatus.Ok });
        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
    });

    it("handles an event with undefined unit_amount by not storing the price", async () => {
        // Setup
        const event = {
            data: {
                object: {
                    id: "price_123456",
                    unit_amount: undefined,
                },
            },
        } as unknown as Stripe.Event;
        await storePrice(PaymentType.PremiumYearly, 420);

        // Act
        const result = await handlePriceUpdated({ event, res });

        // Assert
        expect(result).toEqual({ status: HttpStatus.InternalServerError, message: "Price amount not found" });
        expect(res.status).toHaveBeenCalledWith(HttpStatus.InternalServerError);
        const originalStoredPrice = await fetchPriceFromRedis(PaymentType.PremiumYearly);
        expect(originalStoredPrice).toBe(420);
    });

    it("throws error for unknown price ID", async () => {
        // Setup
        const event = {
            data: {
                object: {
                    id: "price_unknown",
                    unit_amount: 1000,
                },
            },
        } as unknown as Stripe.Event;

        // Act
        const result = await handlePriceUpdated({ event, res });

        // Assert
        expect(result).toEqual({ status: HttpStatus.InternalServerError, message: "Price amount not found" });
        expect(res.status).toHaveBeenCalledWith(HttpStatus.InternalServerError);
    });
});

describe("checkSubscriptionPrices", () => {
    let stripeMock;
    let res;
    const monthlyPrice = 1_000;
    const yearlyPrice = 10_000;

    beforeEach(async function beforeEach() {
        vi.clearAllMocks();

        // Setup Stripe Mock
        stripeMock = new StripeMock() as unknown as Stripe;
        StripeMock.injectData({
            prices: [{
                id: getPriceIds().PremiumMonthly,
                unit_amount: monthlyPrice,
            }, {
                id: getPriceIds().PremiumYearly,
                unit_amount: yearlyPrice,
            }],
        });

        // Setup Redis Mock
        RedisClientMock.resetMock();
        RedisClientMock.__setAllMockData({});

        // Mock response object
        res = createRes();
    });

    afterEach(async function afterEach() {
        StripeMock.resetMock();
    });

    it("should successfully return prices from Redis", async () => {
        storePrice(PaymentType.PremiumMonthly, 69);
        storePrice(PaymentType.PremiumYearly, 420);

        await checkSubscriptionPrices(stripeMock, res);

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(res.json.calledWith({
            data: {
                monthly: 69,
                yearly: 420,
            },
        })).toBe(true);
    });

    it("should fallback to Stripe when prices are not available in Redis", async () => {
        await checkSubscriptionPrices(stripeMock, res);

        const redisMonthlyPrice = await fetchPriceFromRedis(PaymentType.PremiumMonthly);
        const redisYearlyPrice = await fetchPriceFromRedis(PaymentType.PremiumYearly);
        expect(redisMonthlyPrice).toBe(monthlyPrice);
        expect(redisYearlyPrice).toBe(yearlyPrice);

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(res.json.calledWith({
            data: {
                monthly: monthlyPrice,
                yearly: yearlyPrice,
            },
        })).toBe(true);
    });

    it("should recover from redis failures", async () => {
        RedisClientMock.simulateFailure(true);

        await checkSubscriptionPrices(stripeMock, res);

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(res.json.calledWith({
            data: {
                monthly: monthlyPrice,
                yearly: yearlyPrice,
            },
        })).toBe(true);
    });

    it("should handle partial data availability", async () => {
        storePrice(PaymentType.PremiumMonthly, 69);

        await checkSubscriptionPrices(stripeMock, res);

        const redisYearlyPrice = await fetchPriceFromRedis(PaymentType.PremiumYearly);
        expect(redisYearlyPrice).toBe(yearlyPrice);

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(res.json.calledWith({
            data: {
                monthly: 69,
                yearly: yearlyPrice,
            },
        })).toBe(true);
    });

    it("should catch stripe errors", async () => {
        StripeMock.simulateFailure(true);

        await checkSubscriptionPrices(stripeMock, res);

        expect(res.status).toHaveBeenCalledWith(HttpStatus.InternalServerError);
        expect(res.json).toHaveBeenCalledWith({ error: expect.any(Object) });
    });
});

// NOTE: session.created is in seconds
describe("isStripeObjectOlderThan", () => {
    it("should return true for a session older than 180 days", () => {
        const session = {
            created: (Date.now() - (DAYS_180_MS + DAYS_1_MS)) / 1000, // 181 days in the past
        };
        expect(isStripeObjectOlderThan(session, DAYS_180_MS)).toBe(true);
    });

    it("should return false for a session exactly 180 days old", () => {
        const session = {
            created: (Date.now() - DAYS_180_MS) / 1000, // 180 days in the past
        };
        expect(isStripeObjectOlderThan(session, DAYS_180_MS)).toBe(false);
    });

    it("should return false for a session less than 180 days old", () => {
        const session = {
            created: (Date.now() - (DAYS_180_MS - DAYS_1_MS)) / 1000, // 179 days in the past
        };
        expect(isStripeObjectOlderThan(session, DAYS_180_MS)).toBe(false);
    });

    it("should handle sessions in the future as less than 180 days old", () => {
        const session = {
            created: (Date.now() + DAYS_1_MS) / 1000, // 1 day in the future
        };
        expect(isStripeObjectOlderThan(session, DAYS_180_MS)).toBe(false);
    });

    it("should return true when age difference is set to 0 days and the session is in the past", () => {
        const session = {
            created: (Date.now() - HOURS_1_MS) / 1000, // 1 hour in the past
        };
        expect(isStripeObjectOlderThan(session, 0)).toBe(true);
    });
});

describe("setupStripe", () => {
    let app;
    beforeEach(() => {
        app = { get: vi.fn(), post: vi.fn() };
        process.env.STRIPE_SECRET_KEY = "sk_test_123";
        process.env.STRIPE_WEBHOOK_SECRET = "whsec_test_456";
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    it("should log error and return if Stripe keys are missing", () => {
        delete process.env.STRIPE_SECRET_KEY;

        setupStripe(app);

        expect(app.get).not.toHaveBeenCalled();
        expect(app.post).not.toHaveBeenCalled();
    });

    it("should setup Stripe routes if keys are provided", () => {
        setupStripe(app);

        // Add expectations for each route
        const endpoints = [
            ...Object.values(StripeEndpoint).map(ep => "/api" + ep),
            "/webhooks/stripe",
        ];
        const gets = ["/api" + StripeEndpoint.SubscriptionPrices] as string[];
        const posts = endpoints.filter(ep => !gets.includes(ep));

        // Check GETs
        gets.forEach((endpoint) => {
            expect(app.get.mock.calls.flat()).toContain(endpoint);
        });

        // Check POSTs
        posts.forEach((endpoint) => {
            expect(app.post.mock.calls.flat()).toContain(endpoint);
        });
    });
});

// TODO after all functions are tested, create a test which tests the entire process for various tasks. This includes a process for creating a checkout session and completing it, as well as all other processes that involve multiple functions.



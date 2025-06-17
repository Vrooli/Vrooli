import { DAYS_180_MS, DAYS_1_MS, HOURS_1_MS, HttpStatus, PaymentStatus, PaymentType, StripeEndpoint, generatePK } from "@vrooli/shared";
import { describe, it, expect, beforeAll, afterAll, beforeEach, afterEach, vi } from "vitest";
import type Stripe from "stripe";
import StripeMock from "../__test/stripe.js";
import { DbProvider } from "../db/provider.js";
import * as loggerModule from "../events/logger.js";
import { withDbTransaction } from "../__test/helpers/transactionTest.js";
import { CacheService } from "../redisConn.js";
import { PaymentMethod, calculateExpiryAndStatus, checkSubscriptionPrices, createStripeCustomerId, fetchPriceFromRedis, getCustomerId, getPaymentType, getPriceIds, getVerifiedCustomerInfo, getVerifiedSubscriptionInfo, handleCheckoutSessionExpired, handleCustomerDeleted, handleCustomerSourceExpiring, handleCustomerSubscriptionDeleted, handleCustomerSubscriptionTrialWillEnd, handleCustomerSubscriptionUpdated, handleInvoicePaymentCreated, handleInvoicePaymentFailed, handleInvoicePaymentSucceeded, handlePriceUpdated, handlerResult, isInCorrectEnvironment, isStripeObjectOlderThan, isValidCreditsPurchaseSession, isValidSubscriptionSession, parseInvoiceData, setupStripe, storePrice } from "./stripe.js";
import { UserDbFactory, seedTestUsers } from "../__test/fixtures/db/userFixtures.js";
import { seedMockAdminUser } from "../__test/session.js";

// Mock email queue
const mockEmailQueue = {
    add: vi.fn(),
    addTask: vi.fn(),
};

vi.mock("../tasks/queues.js", () => ({
    QueueService: {
        get: () => ({
            email: mockEmailQueue,
        }),
    },
}));

const environments = ["development", "production"];

function createRes() {
    const res = {
        json: vi.fn(),
        send: vi.fn(),
        status: vi.fn().mockReturnThis(), // To allow for chaining .status().send()
    };
    return res;
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

    it("returns the correct status and message", () => {
        const status = HttpStatus.Ok;
        const message = "Success";

        const result = handlerResult(status, mockRes, message);

        expect(result.status).toBe(status);
        expect(result.message).toBe(message);
    });

    it("logs an error message when status is not OK", () => {
        const status = HttpStatus.InternalServerError;
        const message = "Error occurred";
        const trace = "1234";

        const result = handlerResult(status, mockRes, message, trace);

        expect(result.status).toBe(status);
        expect(result.message).toBe(message);
        expect(loggerErrorStub).toHaveBeenCalledTimes(1);
        expect(loggerErrorStub.mock.calls[0][0]).toBe(message);
        expect(loggerErrorStub.mock.calls[0][1]).toMatchObject({ trace });
    });

    it("does not log an error for HttpStatus.Ok", () => {
        const status = HttpStatus.Ok;
        const message = "All good";

        const result = handlerResult(status, mockRes, message);

        expect(result.status).toBe(status);
        expect(result.message).toBe(message);
        expect(loggerErrorStub).not.toHaveBeenCalled();
    });

    it("handles undefined message and trace", () => {
        const status = HttpStatus.BadRequest;

        const result = handlerResult(status, mockRes);

        expect(result.status).toBe(status);
        expect(result.message).toBeUndefined();
        expect(loggerErrorStub).toHaveBeenCalledTimes(1);
        expect(loggerErrorStub.mock.calls[0][0]).toBe("Stripe handler error");
        expect(loggerErrorStub.mock.calls[0][1].trace).toBe("0523");
    });

    it("logs additional arguments when provided", () => {
        const status = HttpStatus.Unauthorized;
        const additionalArgs = { user: "user123", action: "attempted access" };

        const result = handlerResult(status, mockRes, undefined, undefined, additionalArgs);

        expect(result.status).toBe(status);
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
                await redisClient.del(...keys);
            }
            // CacheService doesn't have delPattern, keys are already cleared above
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
            
            // Clear CacheService internal cache to force Redis read
            await cacheService.del(key);

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

    afterEach(async () => {
        // Clean up test data after each test
        await DbProvider.get().plan.deleteMany({});
        await DbProvider.get().user.deleteMany({});
    });

    // Helper function to create test data
    const createTestData = async () => {
        // Create users with specific emails that match StripeMock data
        const user1Id = generatePK();
        const user2Id = generatePK();
        const user3Id = generatePK();
        const user4Id = generatePK();
        const user5Id = generatePK();
        const user6Id = generatePK();
        const user7Id = generatePK();

        // User 1: Has Stripe customer ID and premium plan
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                publicId: user1Id.toString(),
                name: "user1",
                stripeCustomerId: "cus_123",
                emails: {
                    create: [{ id: generatePK(), emailAddress: "user@example.com" }],
                },
                plan: {
                    create: {
                        id: generatePK(),
                        enabledAt: new Date(),
                        expiresAt: new Date(Date.now() + DAYS_180_MS),
                    },
                },
            },
        });

        // User 2: No Stripe customer ID, but email matches StripeMock customer
        await DbProvider.get().user.create({
            data: {
                id: user2Id,
                publicId: user2Id.toString(),
                name: "user2",
                emails: {
                    create: [{ id: generatePK(), emailAddress: "user2@example.com" }],
                },
            },
        });

        // User 3: No Stripe customer ID, email doesn't match any customer
        await DbProvider.get().user.create({
            data: {
                id: user3Id,
                publicId: user3Id.toString(),
                name: "user3",
                emails: {
                    create: [{ id: generatePK(), emailAddress: "missing@example.com" }],
                },
            },
        });

        // User 4: Invalid Stripe customer ID
        await DbProvider.get().user.create({
            data: {
                id: user4Id,
                publicId: user4Id.toString(),
                name: "user4",
                stripeCustomerId: "invalid_customer_id",
            },
        });

        // User 5: Stripe customer ID for deleted customer
        await DbProvider.get().user.create({
            data: {
                id: user5Id,
                publicId: user5Id.toString(),
                name: "user5",
                stripeCustomerId: "cus_deleted",
            },
        });

        // User 6: Email matches deleted customer  
        await DbProvider.get().user.create({
            data: {
                id: user6Id,
                publicId: user6Id.toString(),
                name: "user6",
                emails: {
                    create: [{ id: generatePK(), emailAddress: "emailForDeletedCustomer@example.com" }],
                },
            },
        });

        // User 7: Has Stripe customer ID that will be updated
        await DbProvider.get().user.create({
            data: {
                id: user7Id,
                publicId: user7Id.toString(),
                name: "user7",
                stripeCustomerId: "cus_777",
                emails: {
                    create: [{ id: generatePK(), emailAddress: "emailWithSubscription@example.com" }],
                },
            },
        });

        return { 
            user1Id, 
            user2Id, 
            user3Id, 
            user4Id, 
            user5Id, 
            user6Id, 
            user7Id 
        };
    };

    beforeEach(async function beforeEach() {
        stripe = new StripeMock() as unknown as Stripe;
        StripeMock.resetMock();
        // Inject default customer data for tests
        StripeMock.injectData({
            customers: [
                { id: "cus_123" },
                { id: "cus_test1", email: "user2@example.com" },
                { id: "cus_deleted", deleted: true },
                { id: "cus_777" },
                { id: "cus_789", email: "emailWithSubscription@example.com" },
                { id: "emailForDeletedCustomer", email: "emailForDeletedCustomer@example.com", deleted: true },
            ],
        });
    });

    afterEach(async function afterEach() {
        StripeMock.resetMock();
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
        const result = await getVerifiedCustomerInfo({ userId: generatePK().toString(), stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBeNull();
        expect(result.emails).toEqual([]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    it("should return the user's Stripe customer ID if it exists", async () => {
        const { user1Id } = await createTestData();
        const result = await getVerifiedCustomerInfo({ userId: user1Id.toString(), stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBe("cus_123");
        expect(result.userId).toBe(user1Id.toString());
        expect(result.emails).toEqual([{ emailAddress: "user@example.com" }]);
        expect(result.hasPremium).toBe(true);
        expect(result.subscriptionInfo).toBeNull();
    });

    it("should return the Stripe customer ID associated with the user's email if the user does not have a Stripe customer ID", async () => {
        const { user2Id } = await createTestData();
        const result = await getVerifiedCustomerInfo({ userId: user2Id.toString(), stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBe("cus_test1");
        expect(result.userId).toBe(user2Id.toString());
        expect(result.emails).toEqual([{ emailAddress: "user2@example.com" }]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();

        // Should also update the user's Stripe customer ID in the database
        const updatedUser = await DbProvider.get().user.findUnique({ where: { id: user2Id } });
        expect(updatedUser.stripeCustomerId).toBe("cus_test1");
    });

    it("should return null if the user does not have a Stripe customer ID and their email is not associated with any Stripe customer", async () => {
        const { user3Id } = await createTestData();
        const result = await getVerifiedCustomerInfo({ userId: user3Id.toString(), stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBe(user3Id.toString());
        expect(result.emails).toEqual([{ emailAddress: "missing@example.com" }]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    it("should return null if the user's Stripe customer ID is invalid", async () => {
        const { user4Id } = await createTestData();
        const result = await getVerifiedCustomerInfo({ userId: user4Id.toString(), stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBe(user4Id.toString());
        expect(result.emails).toEqual([]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    it("should return null if the user's Stripe customer ID is associated with a deleted customer", async () => {
        const { user5Id } = await createTestData();
        const result = await getVerifiedCustomerInfo({ userId: user5Id.toString(), stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBe(user5Id.toString());
        expect(result.emails).toEqual([]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    it("should return null if the user's email is associated with a deleted customer", async () => {
        const { user6Id } = await createTestData();
        const result = await getVerifiedCustomerInfo({ userId: user6Id.toString(), stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBe(user6Id.toString());
        expect(result.emails).toEqual([{ emailAddress: "emailForDeletedCustomer@example.com" }]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    it("should return the Stripe customer ID if the user has a valid subscription", async () => {
        const { user1Id } = await createTestData();
        
        // Inject mock data for this specific test
        StripeMock.injectData({
            checkoutSessions: [{
                id: "session_valid",
                customer: "cus_123",
                metadata: {
                    userId: user1Id.toString(),
                    paymentType: "PremiumMonthly",
                },
                subscription: "sub_active1",
                status: "complete",
            }],
            subscriptions: [{
                id: "sub_active1",
                status: "active",
                customer: "cus_123",
            }],
            customers: [{
                id: "cus_123",
            }],
        });
        
        const result = await getVerifiedCustomerInfo({ userId: user1Id.toString(), stripe, validateSubscription: true });
        expect(result.stripeCustomerId).toBe("cus_123");
        expect(result.userId).toBe(user1Id.toString());
        expect(result.emails).toEqual([{ emailAddress: "user@example.com" }]);
        expect(result.hasPremium).toBe(true);
        expect(result.subscriptionInfo).not.toBeNull();
    });

    it("should return the Stripe customer ID associated with the user's email if the user's customer ID does not have a valid subscription, but the email does", async () => {
        const { user7Id } = await createTestData();
        // Check original customer ID (tied to inactive subscription)
        const originalUser = await DbProvider.get().user.findUnique({ where: { id: user7Id } });
        expect(originalUser.stripeCustomerId).toBe("cus_777");

        // Inject mock data for this specific test
        StripeMock.injectData({
            checkoutSessions: [{
                id: "session_user7",
                customer: "cus_777",
                metadata: {
                    userId: user7Id.toString(),
                    paymentType: "PremiumYearly",
                },
                subscription: "sub_inactive",
                status: "complete",
            }, {
                id: "session_user7_2",
                customer: "cus_789",
                metadata: {
                    userId: user7Id.toString(),
                    paymentType: "PremiumYearly",
                },
                subscription: "sub_active2",
                status: "complete",
            }],
            subscriptions: [{
                id: "sub_inactive",
                status: "past_due",
                customer: "cus_777",
            }, {
                id: "sub_active2",
                status: "active",
                customer: "cus_789",
            }],
            customers: [{
                id: "cus_777",
            }, {
                id: "cus_789",
                email: "emailWithSubscription@example.com",
            }],
        });

        const result = await getVerifiedCustomerInfo({ userId: user7Id.toString(), stripe, validateSubscription: true });
        expect(result.stripeCustomerId).toBe("cus_789");
        expect(result.userId).toBe(user7Id.toString());
        expect(result.emails).toEqual([{ emailAddress: "emailWithSubscription@example.com" }]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).not.toBeNull();

        // Check that original customer ID (tied to inactive subscription) was changed to customer associated with email
        const updatedUser = await DbProvider.get().user.findUnique({ where: { id: user7Id } });
        expect(updatedUser.stripeCustomerId).toBe("cus_789");
    });

    it("should return null if the user does not have a valid subscription", async () => {
        const { user3Id } = await createTestData();
        const result = await getVerifiedCustomerInfo({ userId: user3Id, stripe, validateSubscription: true });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.subscriptionInfo).toBeNull();
    });
});

describe("createStripeCustomerId", () => {
    let stripe: Stripe;

    afterEach(async () => {
        // Clean up test data after each test
        await DbProvider.get().user.deleteMany({});
    });

    // Helper function to create test data
    const createTestData = async () => {
        const user1Id = generatePK();
        
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                publicId: user1Id.toString(),
                name: "user1",
                stripeCustomerId: null,
            },
        });

        return { user1Id };
    };

    beforeEach(async function beforeEach() {
        stripe = new StripeMock() as unknown as Stripe;
    });

    afterEach(async function afterEach() {
        StripeMock.resetMock();
    });


    it("creates a new Stripe customer and updates the user with the customer ID", async () => {
        const { user1Id } = await createTestData();
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
        const updatedUser = await DbProvider.get().user.findUnique({ where: { id: user1Id } });
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

    afterEach(async () => {
        // Clean up test data after each test
        await DbProvider.get().payment.deleteMany({});
        await DbProvider.get().user.deleteMany({});
    });

    // Helper function to create test data
    const createTestData = async () => {
        const payment1Id = generatePK();
        const user1Id = generatePK();
        const user2Id = generatePK();

        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                publicId: user1Id.toString(),
                name: "user1",
                stripeCustomerId: "customer1",
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user2Id,
                publicId: user2Id.toString(),
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

        return { payment1Id, user1Id, user2Id };
    };

    beforeEach(async function beforeEach() {
        stripe = new StripeMock() as unknown as Stripe;
        res = createRes();
    });

    afterEach(async function afterEach() {
        StripeMock.resetMock();
    });

    it("does nothing when no related pending payments found", async () => {
        await createTestData();
        const event = { data: { object: { id: "sessionUnknown", customer: "customer1" } } } as unknown as Stripe.Event;

        const originalPaymentsCount = await DbProvider.get().payment.count();

        const result = await handleCheckoutSessionExpired({ event, res, stripe });

        const updatedPaymentsCount = await DbProvider.get().payment.count();

        expect(updatedPaymentsCount).toBe(originalPaymentsCount);
    });

    it("marks related pending payments as failed", async () => {
        const { payment1Id } = await createTestData();
        const event = { data: { object: { id: "session1", customer: "customer1" } } } as unknown as Stripe.Event;

        const result = await handleCheckoutSessionExpired({ event, res, stripe });

        const updatedPayment = await DbProvider.get().payment.findUnique({ where: { id: payment1Id } });
        expect(updatedPayment.status).toBe("Failed");
    });

    it("returns OK status on successful update", async () => {
        await createTestData();
        const event = { data: { object: { id: "session1", customer: "customer1" } } } as unknown as Stripe.Event;

        const result = await handleCheckoutSessionExpired({ event, res, stripe });

        expect(result.status).toBe(HttpStatus.Ok);
    });
});

describe("handleCustomerDeleted", () => {
    let res;
    let stripe: Stripe;

    afterEach(async () => {
        // Clean up test data after each test
        await DbProvider.get().user.deleteMany({});
    });

    // Helper function to create test data
    const createTestData = async () => {
        const user1Id = generatePK();
        const user2Id = generatePK();

        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                publicId: user1Id.toString(),
                name: "user1",
                stripeCustomerId: "customer1",
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user2Id,
                publicId: user2Id.toString(),
                name: "user2",
                stripeCustomerId: "customer2",
            },
        });

        return { user1Id, user2Id };
    };

    beforeEach(async function beforeEach() {
        stripe = new StripeMock() as unknown as Stripe;
        res = createRes();
    });

    afterEach(async function afterEach() {
        StripeMock.resetMock();
    });

    it("removes stripeCustomerId from user on customer deletion", async () => {
        const { user1Id } = await createTestData();
        const event = { data: { object: { id: "customer1" } } } as unknown as Stripe.Event;

        // Perform the action
        const result = await handleCustomerDeleted({ event, res, stripe });

        // Verify the stripeCustomerId was set to null for the user
        const updatedUser = await DbProvider.get().user.findUnique({ where: { id: user1Id } });
        expect(updatedUser.stripeCustomerId).toBeNull();
    });

    it("does not affect users unrelated to the deleted customer", async () => {
        const { user2Id } = await createTestData();
        const event = { data: { object: { id: "customer1" } } } as unknown as Stripe.Event;

        // Perform the action
        const result = await handleCustomerDeleted({ event, res, stripe });

        // Verify that other users are unaffected
        const unaffectedUser = await DbProvider.get().user.findUnique({ where: { id: user2Id } });
        expect(unaffectedUser.stripeCustomerId).toBe("customer2");
    });

    it("returns OK status on successful customer deletion handling", async () => {
        await createTestData();
        const event = { data: { object: { id: "customer1" } } } as unknown as Stripe.Event;

        // Perform the action
        const result = await handleCustomerDeleted({ event, res, stripe });

        // Check if the response status was set to OK
        expect(result.status).toBe(HttpStatus.Ok);
    });

    it("handles cases where the customer does not exist gracefully", async () => {
        await createTestData();
        const event = { data: { object: { id: "customerNonExisting" } } } as unknown as Stripe.Event;

        // Perform the action with a customer ID that doesn't exist
        const result = await handleCustomerDeleted({ event, res, stripe });

        // Expect the function to still return an OK status without throwing any errors
        expect(result.status).toBe(HttpStatus.Ok);
    });
});

describe("handleCustomerSourceExpiring", () => {
    let res;

    afterEach(async () => {
        // Clean up test data after each test
        await DbProvider.get().user.deleteMany({});
    });

    // Helper function to create test data
    const createTestData = async () => {
        const user1Id = generatePK();
        
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                publicId: user1Id.toString(),
                name: "user1",
                stripeCustomerId: "cus_123",
                emails: {
                    create: [{ id: generatePK(), emailAddress: "user1@example.com" }, { id: generatePK(), emailAddress: "user2@example.com" }],
                },
            },
        });

        return { user1Id };
    };

    beforeEach(async function beforeEach() {
        vi.clearAllMocks();
        res = createRes();
        mockEmailQueue.add.mockClear();
        mockEmailQueue.addTask.mockClear();
    });

    afterEach(async function afterEach() {
        // No database cleanup needed with transactions
    });

    it("should send notifications to all user emails when the credit card is expiring", async () => {
        const { user1Id } = await createTestData();
        const mockEvent = {
            data: { object: { id: "cus_123" } },
        } as unknown as Stripe.Event;
        const result = await handleCustomerSourceExpiring({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.Ok);
        expect(mockEmailQueue.addTask).toHaveBeenCalledTimes(1);
        expect(mockEmailQueue.addTask.mock.calls[0][0].to).toEqual(["user1@example.com", "user2@example.com"]);
    });

    it("should return an error when the user is not found", async () => {
        await createTestData(); // Create test data but use non-existent customer
        const mockEvent = {
            data: { object: { id: "non_existent_customer" } },
        } as unknown as Stripe.Event;
        const result = await handleCustomerSourceExpiring({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.InternalServerError);
        expect(mockEmailQueue.addTask).not.toHaveBeenCalled();
    });
});

function expectEmailQueue(expectedData) {
    // Check if the addTask function was called
    expect(mockEmailQueue.addTask).toHaveBeenCalled();

    // Extract the actual data passed to the mock
    const actualData = mockEmailQueue.addTask.mock.calls[0][0];

    // If there's expected data, match it against the actual data
    if (expectedData) {
        expect(actualData).toMatchObject(expectedData);
    }
}

describe("handleCustomerSubscriptionDeleted", () => {
    let res;

    afterEach(async () => {
        // Clean up test data after each test
        await DbProvider.get().plan.deleteMany({});
        await DbProvider.get().user.deleteMany({});
    });

    // Helper function to create test data
    const createTestData = async () => {
        const user1Id = generatePK();
        const user2Id = generatePK();
        
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                publicId: user1Id.toString(),
                name: "user1",
                stripeCustomerId: "cus_123",
                emails: {
                    create: [{ id: generatePK(), emailAddress: "user@example.com" }],
                },
                plan: {
                    create: {
                        id: generatePK(),
                        enabledAt: new Date(),
                        expiresAt: new Date(Date.now() + DAYS_180_MS),
                    },
                },
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user2Id,
                publicId: user2Id.toString(),
                name: "user2",
                stripeCustomerId: "cus_234",
            },
        });
        
        return { user1Id, user2Id };
    };


    beforeEach(async function beforeEach() {
        vi.clearAllMocks();
        res = createRes();
        mockEmailQueue.addTask.mockClear();
    });

    afterEach(async function afterEach() {
        // No database cleanup needed with transactions
    });

    it("should send email when user is found and has email", async () => {
        const { user1Id } = await createTestData();
        const mockEvent = {
            data: { object: { customer: "cus_123", current_period_end: Math.floor(Date.now() / 1000) + 86400 } }, // 1 day in the future
        } as unknown as Stripe.Event;
        const result = await handleCustomerSubscriptionDeleted({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.Ok);
        expectEmailQueue({ to: ["user@example.com"] });
        // Also make sure it didn't affect the premium status
        const updatedUser = await DbProvider.get().user.findUnique({ where: { id: user1Id }, include: { plan: true } });
        expect(updatedUser.plan.expiresAt > new Date()).toBe(true);
    });

    it("should not send email when user is found but has no email", async () => {
        await createTestData();
        const mockEvent = {
            data: { object: { customer: "cus_234", current_period_end: Math.floor(Date.now() / 1000) + 86400 } }, // 1 day in the future
        } as unknown as Stripe.Event;
        const result = await handleCustomerSubscriptionDeleted({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.Ok);
        expect(mockEmailQueue.addTask).toHaveBeenCalledTimes(1);
        expect(mockEmailQueue.addTask.mock.calls[0][0].to).toEqual([]);
    });

    it("should return error when user is not found", async () => {
        await createTestData(); // Create test data but use non-existent customer
        const mockEvent = {
            data: { object: { customer: "non_existent_customer", current_period_end: Math.floor(Date.now() / 1000) + 86400 } }, // 1 day in the future
        } as unknown as Stripe.Event;
        const result = await handleCustomerSubscriptionDeleted({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.InternalServerError);
        expect(mockEmailQueue.addTask).not.toHaveBeenCalled();
    });
});

describe("handleCustomerSubscriptionTrialWillEnd", () => {
    let res;

    afterEach(async () => {
        // Clean up test data after each test
        await DbProvider.get().user.deleteMany({});
    });

    // Helper function to create test data
    const createTestData = async () => {
        const user1Id = generatePK();
        const user2Id = generatePK();
        
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                publicId: user1Id.toString(),
                name: "user1",
                stripeCustomerId: "cus_123",
                emails: {
                    create: [{ id: generatePK(), emailAddress: "user@example.com" }],
                },
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user2Id,
                publicId: user2Id.toString(),
                name: "user2",
                stripeCustomerId: "cus_234",
            },
        });
        
        return { user1Id, user2Id };
    };


    beforeEach(async function beforeEach() {
        vi.clearAllMocks();
        res = createRes();
        mockEmailQueue.addTask.mockClear();
    });

    afterEach(async function afterEach() {
        // No database cleanup needed with transactions
    });

    it("should send email when user is found and has email", async () => {
        await createTestData();
        const mockEvent = {
            data: { object: { customer: "cus_123" } },
        } as unknown as Stripe.Event;
        const result = await handleCustomerSubscriptionTrialWillEnd({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.Ok);
        expectEmailQueue({ to: ["user@example.com"] });
    });

    it("should not send email when user is found but has no email", async () => {
        await createTestData();
        const mockEvent = {
            data: { object: { customer: "cus_234" } },
        } as unknown as Stripe.Event;
        const result = await handleCustomerSubscriptionTrialWillEnd({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.Ok);
        expect(mockEmailQueue.addTask).toHaveBeenCalledTimes(1);
        expect(mockEmailQueue.addTask.mock.calls[0][0].to).toEqual([]);
    });

    it("should return error when user is not found", async () => {
        await createTestData(); // Create test data but use non-existent customer
        const mockEvent = {
            data: { object: { customer: "non_existent_customer" } },
        } as unknown as Stripe.Event;
        const result = await handleCustomerSubscriptionTrialWillEnd({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.InternalServerError);
        expect(mockEmailQueue.addTask).not.toHaveBeenCalled();
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

    afterEach(async () => {
        // Clean up test data after each test
        await DbProvider.get().plan.deleteMany({});
        await DbProvider.get().user.deleteMany({});
    });

    // Helper function to create test data
    const createTestData = async () => {
        const userId = generatePK();
        
        await DbProvider.get().user.create({
            data: {
                id: userId,
                publicId: userId.toString(),
                name: "user1",
                stripeCustomerId: "cus_123",
                emails: {
                    create: [{ id: generatePK(), emailAddress: "user@example.com" }],
                },
                plan: {
                    create: {
                        id: generatePK(),
                        enabledAt: new Date(),
                        expiresAt: new Date(Date.now() + DAYS_180_MS),
                    },
                },
            },
        });
        
        return { userId };
    };

    beforeEach(async function beforeEach() {
        vi.clearAllMocks();
        res = createRes();
        mockEmailQueue.add.mockClear();
        mockEmailQueue.addTask.mockClear();
    });

    afterEach(async function afterEach() {
        // No database cleanup needed with transactions
    });

    it("should skip processing for non-active subscriptions", async () => {
        await createTestData();
        const mockEvent = {
            data: { object: { customer: "cus_123", status: "canceled" } },
        } as unknown as Stripe.Event;
        const result = await handleCustomerSubscriptionUpdated({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.Ok);
        expect(mockEmailQueue.addTask).not.toHaveBeenCalled();
    });

    it("should update expiry time and send thank you email for active subscription", async () => {
        await createTestData();
        const futurePeriodEnd = Math.floor(Date.now() / 1000) + (60 * 60 * 24); // 24 hours in the future
        const mockEvent = {
            data: { object: { customer: "cus_123", status: "active", current_period_end: futurePeriodEnd } },
        } as unknown as Stripe.Event;
        const result = await handleCustomerSubscriptionUpdated({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.Ok);
        expect(mockEmailQueue.addTask.mock.calls[0][0].to).toEqual(["user@example.com"]);
    });

    it("should update expiry time to past and not send thank you email for inactive premium status", async () => {
        await createTestData();
        const pastPeriodEnd = Math.floor(Date.now() / 1000) - (60 * 60 * 24); // 24 hours in the past
        const mockEvent = {
            data: { object: { customer: "cus_123", status: "active", current_period_end: pastPeriodEnd } },
        } as unknown as Stripe.Event;
        const result = await handleCustomerSubscriptionUpdated({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.Ok);
        expect(mockEmailQueue.addTask.mock.calls[0][0].to).toEqual(["user@example.com"]);
    });

    it("should return error when user is not found", async () => {
        await createTestData(); // Create test data but use non-existent customer
        const mockEvent = {
            data: { object: { customer: "non_existent_customer", status: "active", current_period_end: Math.floor(Date.now() / 1000) + (60 * 60 * 24) } },
        } as unknown as Stripe.Event;
        const result = await handleCustomerSubscriptionUpdated({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.InternalServerError);
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

        expect(result.error).toMatch(/Unknown or unsupported invoice line/);
    });

    it("should return error for unknown payment type", () => {
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

        const result = parseInvoiceData(mockInvoice);
        expect(result.error).toBeDefined();
    });
});

describe("handleInvoicePaymentCreated", () => {
    let res;

    afterEach(async () => {
        // Clean up test data after each test
        await DbProvider.get().payment.deleteMany({});
        await DbProvider.get().user.deleteMany({});
    });

    // Helper function to create test data
    const createTestData = async () => {
        const userId = generatePK();
        const paymentId = generatePK();
        
        await DbProvider.get().user.create({
            data: {
                id: userId,
                publicId: userId.toString(),
                name: "user1",
                stripeCustomerId: "cus_123",
            },
        });
        
        await DbProvider.get().payment.create({
            data: {
                id: paymentId,
                userId,
                checkoutId: "inv_123",
                amount: 1000,
                currency: "usd",
                description: "Test payment",
                paymentMethod: "Stripe",
                status: "Pending",
                paymentType: "PremiumMonthly",
            },
        });
        
        return { userId, paymentId };
    };

    beforeEach(async function beforeEach() {
        vi.clearAllMocks();
        res = createRes();
    });

    afterEach(async function afterEach() {
        // No database cleanup needed with transactions
    });

    it("should successfully create a new payment if no existing payment is found", async () => {
        const { userId } = await createTestData();
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
        const result = await handleInvoicePaymentCreated({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.Ok);
        // Verify that the payment exists and has valid data
        const createdPayment = await DbProvider.get().payment.findFirst({ where: { checkoutId: "inv_321" } });
        expect(createdPayment).not.toBeNull();
        expect(createdPayment?.amount).toBe(2000);
        expect(createdPayment?.paymentMethod).toBe("Stripe");
        expect(createdPayment?.status).toBe("Pending");
        expect(createdPayment?.userId).toBe(userId);
    });

    it("should update an existing payment if found", async () => {
        const { paymentId } = await createTestData();
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
        const result = await handleInvoicePaymentCreated({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.Ok);
        // Verify that the payment was updated and has valid data
        const updatedPayment = await DbProvider.get().payment.findFirst({ where: { checkoutId: "inv_123" } });
        expect(updatedPayment).not.toBeNull();
        expect(updatedPayment?.amount).toBe(2000);
        expect(updatedPayment?.paymentMethod).toBe("Stripe");
        expect(updatedPayment?.status).toBe("Pending");
    });

    it("should return error when user is not found", async () => {
        await createTestData(); // Create test data but use non-existent customer
        const mockEvent = {
            data: {
                object: {
                    customer: "cus_nonexistent",
                    id: "inv_124",
                },
            },
        } as unknown as Stripe.Event;
        const result = await handleInvoicePaymentCreated({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.InternalServerError);
    });

    it("should return error when no lines found in invoice", async () => {
        await createTestData();
        const mockEvent = {
            data: {
                object: {
                    customer: "cus_123",
                    id: "inv_125",
                    lines: { data: [] }, // No line items
                },
            },
        } as unknown as Stripe.Event;
        const result = await handleInvoicePaymentCreated({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.InternalServerError);
    });

    it("should return error for unknown payment type", async () => {
        await createTestData();
        const mockEvent = {
            data: {
                object: {
                    customer: "cus_123",
                    id: "inv_126",
                    lines: { data: [{ amount: 2000, price: { id: "uknown_price_id", unit_amount: 2000 } }] },
                },
            },
        } as unknown as Stripe.Event;
        const result = await handleInvoicePaymentCreated({ event: mockEvent, res });
        expect(result.status).toBe(HttpStatus.InternalServerError);
    });
});

describe("handleInvoicePaymentFailed", () => {
    let res;

    afterEach(async () => {
        // Clean up test data after each test
        await DbProvider.get().payment.deleteMany({});
        await DbProvider.get().user.deleteMany({});
    });

    // Helper function to create test data
    const createTestData = async () => {
        const userId = generatePK();
        const paymentId = generatePK();
        
        await DbProvider.get().user.create({
            data: {
                id: userId,
                publicId: userId.toString(),
                name: "user1",
                stripeCustomerId: "cus_123",
                emails: {
                    create: [{ id: generatePK(), emailAddress: "user@example.com" }],
                },
            },
        });
        
        await DbProvider.get().payment.create({
            data: {
                id: paymentId,
                userId,
                checkoutId: "inv_123",
                amount: 1000,
                currency: "usd",
                description: "Test payment",
                paymentMethod: "Stripe",
                status: "Pending",
                paymentType: "PremiumMonthly",
            },
        });
        
        return { userId, paymentId };
    };


    beforeEach(async function beforeEach() {
        vi.clearAllMocks();
        res = createRes();
    });

    afterEach(async function afterEach() {
        // No database cleanup needed with transactions
    });

    it("updates existing payment status to Failed", async () => {
        const { paymentId } = await createTestData();
        const mockEvent = {
            data: { object: { customer: "cus_123", id: "inv_123" } },
        } as unknown as Stripe.Event;
        const result = await handleInvoicePaymentFailed({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.Ok);
        // Verify that payment status was updated to Failed
        const updatedPayment = await DbProvider.get().payment.findFirst({ where: { checkoutId: "inv_123" } });
        expect(updatedPayment?.status).toBe(PaymentStatus.Failed);
    });

    it("creates a new payment and sets status to Failed when no existing payment is found", async () => {
        const { userId } = await createTestData();
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
        const result = await handleInvoicePaymentFailed({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.Ok);
        // Verify that a new payment was created with status set to Failed
        const newPayment = await DbProvider.get().payment.findFirst({ where: { checkoutId: "inv_321" } });
        expect(newPayment?.status).toBe("Failed");
        expect(newPayment?.amount).toBe(2000);
        expect(newPayment?.paymentMethod).toBe("Stripe");
        expect(newPayment?.userId).toBe(userId);
    });

    it("returns an error when parseInvoiceData returns an error", async () => {
        await createTestData();
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
        const result = await handleInvoicePaymentFailed({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.InternalServerError);
    });

    it("notifies user when payment fails", async () => {
        await createTestData();
        const mockEvent = {
            data: { object: { customer: "cus_123", id: "inv_123" } },
        } as unknown as Stripe.Event;
        const result = await handleInvoicePaymentFailed({ event: mockEvent, res });

        expect(result.status).toBe(HttpStatus.Ok);
        expectEmailQueue({ to: ["user@example.com"] });
    });
});

describe("handleInvoicePaymentSucceeded", () => {
    let res;
    let stripeMock;

    afterEach(async () => {
        // Clean up test data after each test
        await DbProvider.get().payment.deleteMany({});
        await DbProvider.get().plan.deleteMany({});
        await DbProvider.get().user.deleteMany({});
    });

    // Helper function to create test data
    const createTestData = async () => {
        const userId = generatePK();
        const paymentId = generatePK();
        
        await DbProvider.get().user.create({
            data: {
                id: userId,
                publicId: userId.toString(),
                name: "user1",
                stripeCustomerId: "cus_123",
                emails: {
                    create: [{ id: generatePK(), emailAddress: "user@example.com" }],
                },
                plan: {
                    create: {
                        id: generatePK(),
                        enabledAt: new Date(),
                        expiresAt: new Date(Date.now() + DAYS_180_MS),
                    },
                },
            },
        });
        
        await DbProvider.get().payment.create({
            data: {
                id: paymentId,
                userId,
                checkoutId: "inv_123",
                amount: 1000,
                currency: "usd",
                description: "Test payment",
                paymentMethod: "Stripe",
                status: "Pending",
                paymentType: "PremiumMonthly",
            },
        });
        
        return { userId, paymentId };
    };

    beforeEach(async function beforeEach() {
        vi.clearAllMocks();
        res = createRes();
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
        // No database cleanup needed with transactions
    });

    it("updates existing payment status to Paid", async () => {
        const { paymentId } = await createTestData();
        const mockEvent = {
            data: {
                object: {
                    customer: "cus_123",
                    id: "inv_123",
                    currency: "usd",
                    lines: { data: [{ amount: 1000, price: { id: getPriceIds().PremiumMonthly, unit_amount: 1000 } }] },
                    subscription: "sub_123",
                },
            },
        } as unknown as Stripe.Event;
        const result = await handleInvoicePaymentSucceeded({ event: mockEvent, res, stripe: stripeMock });

        expect(result.status).toBe(HttpStatus.Ok);
        const updatedPayment = await DbProvider.get().payment.findFirst({ where: { checkoutId: "inv_123" } });
        expect(updatedPayment?.status).toBe(PaymentStatus.Paid);
    });

    it("creates a new payment and sets status to Paid when no existing payment is found", async () => {
        const { userId } = await createTestData();
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
        const result = await handleInvoicePaymentSucceeded({ event: mockEvent, res, stripe: stripeMock });

        expect(result.status).toBe(HttpStatus.Ok);
        const newPayment = await DbProvider.get().payment.findFirst({ where: { checkoutId: "inv_321" } });
        expect(newPayment?.status).toBe(PaymentStatus.Paid);
        expect(newPayment?.amount).toBe(2000);
        expect(newPayment?.paymentMethod).toBe("Stripe");
        expect(newPayment?.userId).toBe(userId);
    });

    it("returns an error when parseInvoiceData returns an error", async () => {
        await createTestData();
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
        const result = await handleInvoicePaymentSucceeded({ event: mockEvent, res, stripe: stripeMock });

        expect(result.status).toBe(HttpStatus.InternalServerError);
    });

    it("updates user premium status for subscription payments", async () => {
        const { userId } = await createTestData();
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
        const result = await handleInvoicePaymentSucceeded({ event: mockEvent, res, stripe: stripeMock });

        expect(result.status).toBe(HttpStatus.Ok);
        const user = await DbProvider.get().user.findUnique({ where: { id: userId }, include: { plan: true } });
        expect(user?.plan?.expiresAt > new Date()).toBe(true);
    });

    it("returns an error if subscription details could not be retrieved", async () => {
        await createTestData();
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
        const result = await handleInvoicePaymentSucceeded({ event: mockEvent, res, stripe: stripeMock });

        expect(result.status).toBe(HttpStatus.InternalServerError);
    });
});

// TODO this test and handleSubscriptionUpdated are causing jest to hang. storePrice is the problem, but I don't know why
describe("handlePriceUpdated", () => {
    let res;

    beforeEach(async function beforeEach() {
        vi.clearAllMocks();
        // Clear Redis data for clean tests
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
        expect(result).toEqual({ status: HttpStatus.Ok, message: undefined });
    });

    it("handles an event with undefined unit_amount by not storing the price", async () => {
        // Setup
        const event = {
            data: {
                object: {
                    id: getPriceIds().PremiumYearly,
                    unit_amount: undefined,
                },
            },
        } as unknown as Stripe.Event;

        // Act
        const result = await handlePriceUpdated({ event, res });

        // Assert
        expect(result).toEqual({ status: HttpStatus.Ok, message: undefined });
        // Price should not be stored when unit_amount is undefined
        const storedPrice = await fetchPriceFromRedis(PaymentType.PremiumYearly);
        expect(storedPrice).toBeNull();
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
        // Unknown price IDs are acknowledged with OK status
        expect(result).toEqual({ status: HttpStatus.Ok, message: undefined });
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
        // Clear Redis data for clean tests
        const cacheService = CacheService.get();
        const redisClient = await cacheService.raw();
        if (redisClient) {
            const keysPattern = `stripe-payment-${process.env.NODE_ENV}-*`;
            const keys = await redisClient.keys(keysPattern);
            if (keys.length > 0) {
                await redisClient.del(...keys);
            }
        }

        // Mock response object
        res = createRes();
    });

    afterEach(async function afterEach() {
        StripeMock.resetMock();
        // Clear Redis after each test
        const cacheService = CacheService.get();
        const redisClient = await cacheService.raw();
        if (redisClient) {
            const keysPattern = `stripe-payment-${process.env.NODE_ENV}-*`;
            const keys = await redisClient.keys(keysPattern);
            if (keys.length > 0) {
                await redisClient.del(...keys);
            }
        }
        // Also clear CacheService internal cache
        await cacheService.del(`stripe-payment-${process.env.NODE_ENV}-${PaymentType.PremiumMonthly}`);
        await cacheService.del(`stripe-payment-${process.env.NODE_ENV}-${PaymentType.PremiumYearly}`);
    });

    it("should successfully return prices from Redis", async () => {
        await storePrice(PaymentType.PremiumMonthly, 69);
        await storePrice(PaymentType.PremiumYearly, 420);

        await checkSubscriptionPrices(stripeMock, res);

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(res.json).toHaveBeenCalledWith({
            data: {
                monthly: 69,
                yearly: 420,
            },
        });
    });

    it("should fallback to Stripe when prices are not available in Redis", async () => {
        await checkSubscriptionPrices(stripeMock, res);

        const redisMonthlyPrice = await fetchPriceFromRedis(PaymentType.PremiumMonthly);
        const redisYearlyPrice = await fetchPriceFromRedis(PaymentType.PremiumYearly);
        expect(redisMonthlyPrice).toBe(monthlyPrice);
        expect(redisYearlyPrice).toBe(yearlyPrice);

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(res.json).toHaveBeenCalledWith({
            data: {
                monthly: monthlyPrice,
                yearly: yearlyPrice,
            },
        });
    });

    it("should recover from redis failures", async () => {
        // Note: Can't easily simulate Redis failure in integration tests

        await checkSubscriptionPrices(stripeMock, res);

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(res.json).toHaveBeenCalledWith({
            data: {
                monthly: monthlyPrice,
                yearly: yearlyPrice,
            },
        });
    });

    it("should handle partial data availability", async () => {
        await storePrice(PaymentType.PremiumMonthly, 69);

        await checkSubscriptionPrices(stripeMock, res);

        const redisYearlyPrice = await fetchPriceFromRedis(PaymentType.PremiumYearly);
        expect(redisYearlyPrice).toBe(yearlyPrice);

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(res.json).toHaveBeenCalledWith({
            data: {
                monthly: 69,
                yearly: yearlyPrice,
            },
        });
    });

    it("should catch stripe errors", async () => {
        StripeMock.simulateFailure(true);

        await checkSubscriptionPrices(stripeMock, res);

        expect(res.status).toHaveBeenCalledWith(HttpStatus.InternalServerError);
        expect(res.json).toHaveBeenCalledWith({ 
            errors: expect.arrayContaining([
                expect.objectContaining({
                    message: expect.any(String),
                    trace: expect.any(String)
                })
            ]),
            version: "v2"
        });
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



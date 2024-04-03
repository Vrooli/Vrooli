/* eslint-disable @typescript-eslint/ban-ts-comment */
import { HttpStatus, PaymentType, StripeEndpoint } from "@local/shared";
import Stripe from "stripe";
import pkg from "../__mocks__/@prisma/client";
import Bull from "../__mocks__/bull";
import { RedisClientMock } from "../__mocks__/redis";
import StripeMock from "../__mocks__/stripe";
import { setupEmailQueue } from "../tasks/email/queue";
import { checkSubscriptionPrices, createStripeCustomerId, fetchPriceFromRedis, getCustomerId, getPaymentType, getPriceIds, getVerifiedCustomerInfo, getVerifiedSubscriptionInfo, handleCheckoutSessionExpired, handleCustomerDeleted, handleCustomerSubscriptionDeleted, handleCustomerSubscriptionTrialWillEnd, handlePriceUpdated, handlerResult, isInCorrectEnvironment, isStripeObjectOlderThan, isValidCreditsPurchaseSession, isValidSubscriptionSession, setupStripe, storePrice } from "./stripe";

const { PrismaClient } = pkg;

const environments = ["development", "production"];

const createRes = () => ({
    json: jest.fn(),
    send: jest.fn(),
    status: jest.fn().mockReturnThis(), // To allow for chaining .status().send()
});

describe("getPaymentType", () => {
    const originalEnv = process.env.NODE_ENV;

    beforeEach(() => {
        jest.resetModules();
    });

    afterAll(() => {
        process.env.NODE_ENV = originalEnv;
    });

    // Helper function to generate test cases for both environments
    const runTestsForEnvironment = (env: string) => {
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
                expect(() => getPaymentType(invalidPriceId)).toThrow("Invalid price ID");
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
    };

    // Run tests for both development and production environments
    environments.forEach(runTestsForEnvironment);
});

describe("getCustomerId function tests", () => {
    const mockCustomer = { id: "cust_12345" } as Stripe.Customer;

    test("returns the string when customer is a string", () => {
        const input = "cust_string";
        const output = getCustomerId(input);
        expect(output).toBe("cust_string");
    });

    test("returns the id when customer is a Stripe Customer object", () => {
        const output = getCustomerId(mockCustomer);
        expect(output).toBe("cust_12345");
    });

    test("throws an error when customer is null", () => {
        expect(() => {
            getCustomerId(null);
        }).toThrow("Customer ID not found");
    });

    test("throws an error when customer is undefined", () => {
        expect(() => {
            getCustomerId(undefined);
        }).toThrow("Customer ID not found");
    });

    test("throws an error when customer is an empty object", () => {
        expect(() => {
            // @ts-ignore Testing runtime scenario
            getCustomerId({});
        }).toThrow("Customer ID not found");
    });

    test("throws an error when id property is not a string", () => {
        expect(() => {
            // @ts-ignore Testing runtime scenario
            getCustomerId({ id: 12345 });
        }).toThrow("Customer ID not found");
    });
});

describe("handlerResult", () => {
    let mockRes: any;

    beforeEach(() => {
        // Reset mocks
        jest.resetAllMocks();

        // Setup a mock Express response object
        mockRes = {
            status: jest.fn().mockReturnThis(),
            send: jest.fn().mockReturnThis(),
        };
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
    });

    it("does not log an error for HttpStatus.OK", () => {
        const status = HttpStatus.Ok;
        const message = "All good";

        handlerResult(status, mockRes, message);
    });

    it("handles undefined message and trace", () => {
        const status = HttpStatus.BadRequest;

        handlerResult(status, mockRes);

        expect(mockRes.status).toHaveBeenCalledWith(status);
    });

    it("logs additional arguments when provided", () => {
        const status = HttpStatus.Unauthorized;
        const additionalArgs = { user: "user123", action: "attempted access" };

        handlerResult(status, mockRes, undefined, undefined, additionalArgs);
    });
});

describe("Redis Price Operations", () => {
    const originalEnv = process.env.NODE_ENV;

    beforeEach(() => {
        jest.clearAllMocks();
        RedisClientMock.resetMock();
        RedisClientMock.__setAllMockData({});
    });

    afterAll(() => {
        process.env.NODE_ENV = originalEnv; // Restore the original NODE_ENV
    });

    const runTestsForEnvironment = (env) => {
        describe(`when NODE_ENV is ${env}`, () => {
            beforeEach(() => {
                process.env.NODE_ENV = env;
            });

            it("correctly stores and fetches the price for a given payment type", async () => {
                const paymentType = PaymentType.PremiumMonthly;
                const price = 999;

                // Store the price
                await storePrice(paymentType, price);

                // Attempt to fetch the stored price
                const fetchedPrice = await fetchPriceFromRedis(paymentType);

                // Verify the fetched price matches the stored price
                expect(fetchedPrice).toBe(price);
            });

            it("returns null for a price that was not stored", async () => {
                const paymentType = "NonExistentType" as unknown as PaymentType;

                // Attempt to fetch a price for a payment type that hasn't been stored
                const fetchedPrice = await fetchPriceFromRedis(paymentType);

                // Verify that null is returned for the non-existent price
                expect(fetchedPrice).toBeNull();
            });

            it("returns null when the stored price is not a number", async () => {
                const paymentType = PaymentType.PremiumMonthly;
                const price = "invalid" as unknown as number;

                // Store an invalid price
                await storePrice(paymentType, price);

                // Attempt to fetch the stored price
                const fetchedPrice = await fetchPriceFromRedis(paymentType);

                // Verify that null is returned for the invalid price
                expect(fetchedPrice).toBeNull();
            });

            it("returns null when the price is less than 0", async () => {
                const paymentType = PaymentType.PremiumMonthly;
                const price = -1;

                // Store a negative price
                await storePrice(paymentType, price);

                // Attempt to fetch the stored price
                const fetchedPrice = await fetchPriceFromRedis(paymentType);

                // Verify that null is returned for the negative price
                expect(fetchedPrice).toBeNull();
            });

            it("returns null when the price is NaN", async () => {
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
    };

    // Run tests for both development and production environments
    environments.forEach(runTestsForEnvironment);
});

describe("isInCorrectEnvironment Tests", () => {
    // Store the original NODE_ENV to restore after tests
    const originalEnv = process.env.NODE_ENV;

    // Helper function to run tests in both environments
    const runTestsForEnvironment = (env) => {
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
    };

    // Iterate over each environment and run the tests
    environments.forEach(runTestsForEnvironment);
});

describe("isValidSubscriptionSession", () => {
    // Store the original NODE_ENV to restore after tests
    const originalEnv = process.env.NODE_ENV;
    const userId = "testUserId";

    // Helper function to run tests in both environments
    const runTestsForEnvironment = (env) => {
        describe(`when NODE_ENV is ${env}`, () => {
            beforeAll(() => {
                process.env.NODE_ENV = env;
            });

            afterAll(() => {
                // Restore the original NODE_ENV after each suite
                process.env.NODE_ENV = originalEnv;
            });

            const createMockSession = (overrides) => ({
                livemode: process.env.NODE_ENV === "production",
                metadata: {
                    paymentType: PaymentType.PremiumMonthly,
                    userId,
                },
                status: "complete",
                ...overrides,
            });

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
    };

    // Iterate over each environment and run the tests
    environments.forEach(runTestsForEnvironment);
});

describe("isValidCreditsPurchaseSession", () => {
    // Store the original NODE_ENV to restore after tests
    const originalEnv = process.env.NODE_ENV;
    const userId = "testUserId";

    // Helper function to run tests in both environments
    const runTestsForEnvironment = (env) => {
        describe(`when NODE_ENV is ${env}`, () => {
            beforeAll(() => {
                process.env.NODE_ENV = env;
            });

            afterAll(() => {
                // Restore the original NODE_ENV after each suite
                process.env.NODE_ENV = originalEnv;
            });

            const createMockSession = (overrides) => ({
                livemode: process.env.NODE_ENV === "production",
                metadata: {
                    paymentType: PaymentType.Credits,
                    userId,
                },
                status: "complete",
                ...overrides,
            });

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
    };

    // Define the environments to test, typically just 'production' and 'development'
    const environments = ["production", "development"];

    // Iterate over each environment and run the tests
    environments.forEach(runTestsForEnvironment);
});

describe("getVerifiedSubscriptionInfo", () => {
    let stripe: Stripe;

    beforeEach(() => {
        jest.clearAllMocks();
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

    afterEach(() => {
        StripeMock.resetMock();
        PrismaClient.clearData();
    });

    test("returns verified subscription info for an active subscription with valid session", async () => {
        const result = await getVerifiedSubscriptionInfo(stripe, "cus_user1", "user1");
        expect(result).not.toBeNull();
        expect(result).toHaveProperty("session");
        expect(result).toHaveProperty("subscription");
        expect(result?.paymentType).toBe("PremiumMonthly");
    });

    test("returns null if user in metadata does not match what we expect", async () => {
        const result = await getVerifiedSubscriptionInfo(stripe, "cus_user1", "beep");
        expect(result).toBeNull();
    });

    test("returns null if no subscriptions exist for the customer", async () => {
        const result = await getVerifiedSubscriptionInfo(stripe, "cus_randomDude", "user1");
        expect(result).toBeNull();
    });

    test("returns null if subscriptions exist but none are active or trialing", async () => {
        const result = await getVerifiedSubscriptionInfo(stripe, "cus_user2", "user2");
        expect(result).toBeNull();
    });

    test("returns null if the session metadata's payment type is not recognized", async () => {
        const result = await getVerifiedSubscriptionInfo(stripe, "cus_user3", "user3");
        expect(result).toBeNull();
    });

    test("returns verified subscription info if first one was invalid but second one is valid", async () => {
        const result = await getVerifiedSubscriptionInfo(stripe, "cus_user4", "user4");
        expect(result).not.toBeNull();
        expect(result).toHaveProperty("session");
        expect(result).toHaveProperty("subscription");
        expect(result?.paymentType).toBe("PremiumYearly");
    });
});

describe("getVerifiedCustomerInfo", () => {
    let stripe: Stripe;

    beforeEach(async () => {
        jest.clearAllMocks();
        stripe = new StripeMock() as unknown as Stripe;
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
        PrismaClient.injectData({
            User: [{
                id: "user1",
                stripeCustomerId: "cus_123",
                emails: [{ emailAddress: "user@example.com" }],
                premium: { isActive: true },
            }, {
                id: "user2",
                emails: [{ emailAddress: "user2@example.com" }],
                premium: { isActive: false },
            }, {
                id: "user3",
                emails: [{ emailAddress: "missing@example.com" }],
            }, {
                id: "user4",
                stripeCustomerId: "invalid_customer_id",
            }, {
                id: "user5",
                stripeCustomerId: "cus_deleted",
            }, {
                id: "user6",
                stripeCustomerId: null,
                emails: [{ emailAddress: "emailForDeletedCustomer@example.com" }],
            }, {
                id: "user7",
                stripeCustomerId: "cus_777",
                emails: [{ emailAddress: "emailWithSubscription@example.com" }],
            }],
        });
    });

    afterEach(() => {
        StripeMock.resetMock();
        PrismaClient.clearData();
    });

    test("should return null if userId is not provided", async () => {
        const result = await getVerifiedCustomerInfo({ userId: undefined, stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBeNull();
        expect(result.emails).toEqual([]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    test("should return null if the user does not exist", async () => {
        const result = await getVerifiedCustomerInfo({ userId: "nonexistent_user", stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBeNull();
        expect(result.emails).toEqual([]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    test("should return the user's Stripe customer ID if it exists", async () => {
        const result = await getVerifiedCustomerInfo({ userId: "user1", stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBe("cus_123");
        expect(result.userId).toBe("user1");
        expect(result.emails).toEqual([{ emailAddress: "user@example.com" }]);
        expect(result.hasPremium).toBe(true);
        expect(result.subscriptionInfo).toBeNull();
    });

    test("should return the Stripe customer ID associated with the user's email if the user does not have a Stripe customer ID", async () => {
        const result = await getVerifiedCustomerInfo({ userId: "user2", stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBe("cus_test1");
        expect(result.userId).toBe("user2");
        expect(result.emails).toEqual([{ emailAddress: "user2@example.com" }]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();

        // Should also update the user's Stripe customer ID in the database
        // @ts-ignore Testing runtime scenario
        const updatedUser = await PrismaClient.instance.user.findUnique({ where: { id: "user2" } });
        expect(updatedUser.stripeCustomerId).toBe("cus_test1");
    });

    test("should return null if the user does not have a Stripe customer ID and their email is not associated with any Stripe customer", async () => {
        const result = await getVerifiedCustomerInfo({ userId: "user3", stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBe("user3");
        expect(result.emails).toEqual([{ emailAddress: "missing@example.com" }]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    test("should return null if the user's Stripe customer ID is invalid", async () => {
        const result = await getVerifiedCustomerInfo({ userId: "user4", stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBe("user4");
        expect(result.emails).toEqual([]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    test("should return null if the user's Stripe customer ID is associated with a deleted customer", async () => {
        const result = await getVerifiedCustomerInfo({ userId: "user5", stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBe("user5");
        expect(result.emails).toEqual([]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    test("should return null if the user's email is associated with a deleted customer", async () => {
        const result = await getVerifiedCustomerInfo({ userId: "user6", stripe, validateSubscription: false });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.userId).toBe("user6");
        expect(result.emails).toEqual([{ emailAddress: "emailForDeletedCustomer@example.com" }]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).toBeNull();
    });

    test("should return the Stripe customer ID if the user has a valid subscription", async () => {
        const result = await getVerifiedCustomerInfo({ userId: "user1", stripe, validateSubscription: true });
        expect(result.stripeCustomerId).toBe("cus_123");
        expect(result.userId).toBe("user1");
        expect(result.emails).toEqual([{ emailAddress: "user@example.com" }]);
        expect(result.hasPremium).toBe(true);
        expect(result.subscriptionInfo).not.toBeNull();
    });

    test("should return the Stripe customer ID associated with the user's email if the user's customer ID does not have a valid subscription, but the email does", async () => {
        // Check original customer ID (tied to inactive subscription)
        // @ts-ignore Testing runtime scenario
        const originalUser = await PrismaClient.instance.user.findUnique({ where: { id: "user7" } });
        expect(originalUser.stripeCustomerId).toBe("cus_777");

        const result = await getVerifiedCustomerInfo({ userId: "user7", stripe, validateSubscription: true });
        expect(result.stripeCustomerId).toBe("cus_789");
        expect(result.userId).toBe("user7");
        expect(result.emails).toEqual([{ emailAddress: "emailWithSubscription@example.com" }]);
        expect(result.hasPremium).toBe(false);
        expect(result.subscriptionInfo).not.toBeNull();

        // Check that original customer ID (tied to inactive subscription) was changed to customer associated with email
        // @ts-ignore Testing runtime scenario
        const updatedUser = await PrismaClient.instance.user.findUnique({ where: { id: "user7" } });
        expect(updatedUser.stripeCustomerId).toBe("cus_789");
    });

    test("should return null if the user does not have a valid subscription", async () => {
        const result = await getVerifiedCustomerInfo({ userId: "user3", stripe, validateSubscription: true });
        expect(result.stripeCustomerId).toBeNull();
        expect(result.subscriptionInfo).toBeNull();
    });
});

describe("createStripeCustomerId", () => {
    let stripe: Stripe;

    beforeEach(async () => {
        jest.clearAllMocks();
        stripe = new StripeMock() as unknown as Stripe;
        PrismaClient.injectData({
            User: [{
                id: "user1",
                stripeCustomerId: null,
            }],
        });
    });

    afterEach(() => {
        StripeMock.resetMock();
        PrismaClient.clearData();
    });


    test("creates a new Stripe customer and updates the user with the customer ID", async () => {
        const customerInfo = {
            userId: "user1",
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
        expect(result).toEqual(expect.any(String));
        // @ts-ignore Testing runtime scenario
        const updatedUser = await PrismaClient.instance.user.findUnique({ where: { id: "user1" } });
        expect(updatedUser.stripeCustomerId).toEqual(expect.any(String));
    });

    test("creates a new Stripe customer without requiring an existing user", async () => {
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
        expect(result).toEqual(expect.any(String));
    });

    test("throws an error when no user ID is provided but requireUserToExist is true", async () => {
        const customerInfo = {
            userId: null,
            emails: [],
            hasPremium: false,
            subscriptionInfo: null,
            stripeCustomerId: null,
        };
        await expect(createStripeCustomerId({
            customerInfo,
            requireUserToExist: true,
            stripe,
        })).rejects.toThrow("User not found.");
    });
});

describe("handleCheckoutSessionExpired", () => {
    let res;
    let stripe: Stripe;

    beforeEach(async () => {
        jest.clearAllMocks();
        stripe = new StripeMock() as unknown as Stripe;
        PrismaClient.injectData({
            Payment: [{
                id: "payment1",
                checkoutId: "session1",
                paymentMethod: "Stripe",
                status: "Pending",
                user: { stripeCustomerId: "customer1" },
            }],
            User: [{
                id: "user1",
                stripeCustomerId: "customer1",
            }, {
                id: "user2",
                stripeCustomerId: "customer2",
            }],
        });
        res = createRes();
    });

    afterEach(() => {
        StripeMock.resetMock();
        PrismaClient.clearData();
    });

    test("does nothing when no related pending payments found", async () => {
        const event = { data: { object: { id: "sessionUnknown", customer: "customer1" } } } as unknown as Stripe.Event;

        // @ts-ignore Testing runtime scenario
        const originalPayments = JSON.parse(JSON.stringify(await PrismaClient.instance.payment.findMany({ where: {} })));

        await handleCheckoutSessionExpired({ event, res, stripe });

        // @ts-ignore Testing runtime scenario
        const updatedPayments = JSON.parse(JSON.stringify(await PrismaClient.instance.payment.findMany({ where: {} })));

        expect(updatedPayments).toEqual(originalPayments);
    });

    test("marks related pending payments as failed", async () => {
        const event = { data: { object: { id: "session1", customer: "customer1" } } } as unknown as Stripe.Event;

        await handleCheckoutSessionExpired({ event, res, stripe });

        // @ts-ignore Testing runtime scenario
        const updatedPayment = await PrismaClient.instance.payment.findUnique({ where: { id: "payment1" } });
        expect(updatedPayment.status).toBe("Failed");
    });

    test("returns OK status on successful update", async () => {
        const event = { data: { object: { id: "session1", customer: "customer1" } } } as unknown as Stripe.Event;

        await handleCheckoutSessionExpired({ event, res, stripe });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
    });
});

describe("handleCustomerDeleted", () => {
    let res;
    let stripe: Stripe;

    beforeEach(async () => {
        jest.clearAllMocks();
        stripe = new StripeMock() as unknown as Stripe;
        PrismaClient.injectData({
            User: [{
                id: "user1",
                stripeCustomerId: "customer1",
            }, {
                id: "user2",
                stripeCustomerId: "customer2",
            }],
        });
        res = createRes();
    });

    afterEach(() => {
        StripeMock.resetMock();
        PrismaClient.clearData();
    });

    test("removes stripeCustomerId from user on customer deletion", async () => {
        const event = { data: { object: { id: "customer1" } } } as unknown as Stripe.Event;

        // Perform the action
        await handleCustomerDeleted({ event, res, stripe });

        // Verify the stripeCustomerId was set to null for the user
        // @ts-ignore Testing runtime scenario
        const updatedUser = await PrismaClient.instance.user.findUnique({ where: { id: "user1" } });
        expect(updatedUser.stripeCustomerId).toBeNull();
    });

    test("does not affect users unrelated to the deleted customer", async () => {
        const event = { data: { object: { id: "customer1" } } } as unknown as Stripe.Event;

        // Perform the action
        await handleCustomerDeleted({ event, res, stripe });

        // Verify that other users are unaffected
        // @ts-ignore Testing runtime scenario
        const unaffectedUser = await PrismaClient.instance.user.findUnique({ where: { id: "user2" } });
        expect(unaffectedUser.stripeCustomerId).toBe("customer2");
    });

    test("returns OK status on successful customer deletion handling", async () => {
        const event = { data: { object: { id: "customer1" } } } as unknown as Stripe.Event;

        // Perform the action
        await handleCustomerDeleted({ event, res, stripe });

        // Check if the response status was set to OK
        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
    });

    test("handles cases where the customer does not exist gracefully", async () => {
        const event = { data: { object: { id: "customerNonExisting" } } } as unknown as Stripe.Event;

        // Perform the action with a customer ID that doesn't exist
        await handleCustomerDeleted({ event, res, stripe });

        // Expect the function to still return an OK status without throwing any errors
        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
    });
});

const expectEmailQueue = (mockQueue, expectedData) => {
    // Check if the add function was called
    expect(mockQueue.add).toHaveBeenCalled();

    // Extract the actual data passed to the mock
    const actualData = mockQueue.add.mock.calls[0][0];

    // If there's expected data, match it against the actual data
    if (expectedData) {
        expect(actualData).toMatchObject(expectedData);
    }
};

describe("handleCustomerSubscriptionDeleted", () => {
    let res;
    let emailQueue;

    beforeAll(async () => {
        emailQueue = new Bull("email");
        await setupEmailQueue();
    });

    beforeEach(() => {
        jest.clearAllMocks();
        res = createRes();
        Bull.resetMock();
        PrismaClient.injectData({
            User: [{
                id: "user1",
                stripeCustomerId: "cus_123",
                emails: [{ emailAddress: "user@example.com" }],
                premium: { isActive: true },
            }, {
                id: "user2",
                stripeCustomerId: "cus_234",
                emails: [],
            }],
        });
    });

    afterEach(() => {
        PrismaClient.clearData();
    });

    test("should send email when user is found and has email", async () => {
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

    test("should not send email when user is found but has no email", async () => {
        const mockEvent = {
            data: { object: { customer: "cus_234" } },
        } as unknown as Stripe.Event;
        await handleCustomerSubscriptionDeleted({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(emailQueue.add).not.toHaveBeenCalled();
    });

    test("should return error when user is not found", async () => {
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

    beforeAll(async () => {
        emailQueue = new Bull("email");
        await setupEmailQueue();
    });

    beforeEach(() => {
        jest.clearAllMocks();
        res = createRes();
        Bull.resetMock();
        PrismaClient.injectData({
            User: [{
                id: "user1",
                stripeCustomerId: "cus_123",
                emails: [{ emailAddress: "user@example.com" }],
            }, {
                id: "user2",
                stripeCustomerId: "cus_234",
                emails: [],
            }],
        });
    });

    afterEach(() => {
        PrismaClient.clearData();
    });

    test("should send email when user is found and has email", async () => {
        const mockEvent = {
            data: { object: { customer: "cus_123" } },
        } as unknown as Stripe.Event;
        await handleCustomerSubscriptionTrialWillEnd({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expectEmailQueue(emailQueue, { to: ["user@example.com"] });
    });

    test("should not send email when user is found but has no email", async () => {
        const mockEvent = {
            data: { object: { customer: "cus_234" } },
        } as unknown as Stripe.Event;
        await handleCustomerSubscriptionTrialWillEnd({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(emailQueue.add).not.toHaveBeenCalled();
    });

    test("should return error when user is not found", async () => {
        const mockEvent = {
            data: { object: { customer: "non_existent_customer" } },
        } as unknown as Stripe.Event;
        await handleCustomerSubscriptionTrialWillEnd({ event: mockEvent, res });

        expect(res.status).toHaveBeenCalledWith(HttpStatus.InternalServerError);
        expect(emailQueue.add).not.toHaveBeenCalled();
    });
});

describe("handlePriceUpdated", () => {
    let res;

    beforeEach(() => {
        jest.clearAllMocks();
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

    beforeEach(() => {
        jest.clearAllMocks();

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

    afterEach(() => {
        StripeMock.resetMock();
    });

    test("should successfully return prices from Redis", async () => {
        storePrice(PaymentType.PremiumMonthly, 69);
        storePrice(PaymentType.PremiumYearly, 420);

        await checkSubscriptionPrices(stripeMock, res);

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(res.json).toHaveBeenCalledWith({
            data: {
                monthly: 69,
                yearly: 420,
            },
        });
    });

    test("should fallback to Stripe when prices are not available in Redis", async () => {
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

    test("should recover from redis failures", async () => {
        RedisClientMock.simulateFailure(true);

        await checkSubscriptionPrices(stripeMock, res);

        expect(res.status).toHaveBeenCalledWith(HttpStatus.Ok);
        expect(res.json).toHaveBeenCalledWith({
            data: {
                monthly: monthlyPrice,
                yearly: yearlyPrice,
            },
        });
    });

    test("should handle partial data availability", async () => {
        storePrice(PaymentType.PremiumMonthly, 69);

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

    test("should catch stripe errors", async () => {
        StripeMock.simulateFailure(true);

        await checkSubscriptionPrices(stripeMock, res);

        expect(res.status).toHaveBeenCalledWith(HttpStatus.InternalServerError);
        expect(res.json).toHaveBeenCalledWith({ error: expect.any(Object) });
    });
});

// NOTE: session.created is in seconds
describe("isStripeObjectOlderThan", () => {
    // Constant for 180 days in milliseconds
    const DAYS_180_MS = 180 * 24 * 60 * 60 * 1000;
    const DAYS_1_MS = 24 * 60 * 60 * 1000;
    const HOURS_1_MS = 60 * 60 * 1000;

    test("should return true for a session older than 180 days", () => {
        const session = {
            created: (Date.now() - (DAYS_180_MS + DAYS_1_MS)) / 1000, // 181 days in the past
        };
        expect(isStripeObjectOlderThan(session, DAYS_180_MS)).toBe(true);
    });

    test("should return false for a session exactly 180 days old", () => {
        const session = {
            created: (Date.now() - DAYS_180_MS) / 1000, // 180 days in the past
        };
        expect(isStripeObjectOlderThan(session, DAYS_180_MS)).toBe(false);
    });

    test("should return false for a session less than 180 days old", () => {
        const session = {
            created: (Date.now() - (DAYS_180_MS - DAYS_1_MS)) / 1000, // 179 days in the past
        };
        expect(isStripeObjectOlderThan(session, DAYS_180_MS)).toBe(false);
    });

    test("should handle sessions in the future as less than 180 days old", () => {
        const session = {
            created: (Date.now() + DAYS_1_MS) / 1000, // 1 day in the future
        };
        expect(isStripeObjectOlderThan(session, DAYS_180_MS)).toBe(false);
    });

    test("should return true when age difference is set to 0 days and the session is in the past", () => {
        const session = {
            created: (Date.now() - HOURS_1_MS) / 1000, // 1 hour in the past
        };
        expect(isStripeObjectOlderThan(session, 0)).toBe(true);
    });
});

describe("setupStripe", () => {
    let app;
    beforeEach(() => {
        app = { get: jest.fn(), post: jest.fn() };
        process.env.STRIPE_SECRET_KEY = "sk_test_123";
        process.env.STRIPE_WEBHOOK_SECRET = "whsec_test_456";
    });

    afterEach(() => {
        jest.clearAllMocks();
    });

    test("should log error and return if Stripe keys are missing", () => {
        delete process.env.STRIPE_SECRET_KEY;

        setupStripe(app);

        expect(app.get).not.toHaveBeenCalled();
        expect(app.post).not.toHaveBeenCalled();
    });

    test("should setup Stripe routes if keys are provided", () => {
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

import { HttpStatus, PaymentType } from "@local/shared";
import Stripe from "stripe";
import { RedisClientMock } from "../__mocks__/redis";
import { fetchPriceFromRedis, getPaymentType, getPriceIds, handlerResult, isInCorrectEnvironment, isValidSubscriptionSession, storePrice } from "./stripe";

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
    ["development", "production"].forEach(runTestsForEnvironment);
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
    ["development", "production"].forEach(runTestsForEnvironment);
});

describe("isInCorrectEnvironment Tests", () => {
    const environments = ["development", "production"];

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
    const environments = ["development", "production"];

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
                    liveMode: process.env.NODE_ENV === "production" ? false : true,
                });
                expect(isValidSubscriptionSession(session, userId)).toBe(false);
            });
        });
    };
});

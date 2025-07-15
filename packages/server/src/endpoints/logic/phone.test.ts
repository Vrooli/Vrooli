import { type PhoneCreateInput, type SendVerificationTextInput, type ValidateVerificationTextInput, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { mockAuthenticatedSession, mockLoggedOutSession, mockApiSession, mockWriteAuthPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { assertEndpointRequiresAuth } from "../../__test/endpoints.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { phone_createOne } from "../generated/phone_createOne.js";
import { phone_validate } from "../generated/phone_validate.js";
import { phone_verify } from "../generated/phone_verify.js";
import { phone } from "./phone.js";
// Import database fixtures
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { createPhoneDbFactory } from "../../__test/fixtures/db/PhoneDbFactory.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

describe("EndpointsPhone", () => {
    let phoneFactory: ReturnType<typeof createPhoneDbFactory>;

    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);

        // Initialize factory
        phoneFactory = createPhoneDbFactory();
    });

    afterEach(async () => {
        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["user","user_auth","email","session"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn('Test cleanup incomplete:', orphans);
        }
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.minimal(DbProvider.get());
    }););

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("createOne", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertEndpointRequiresAuth(
                    phone.createOne,
                    {
                        id: generatePK(),
                        phoneNumber: "+1234567890",
                    },
                    phone_createOne,
                );
            });

            it("requires auth write permissions", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWritePrivatePermissions(["Phone"]), // Wrong permission type
                });

                const input: PhoneCreateInput = {
                    id: generatePK(),
                    phoneNumber: "+1234567890",
                };

                await expect(phone.createOne({ input }, { req, res }, phone_createOne))
                    .rejects.toThrow(CustomError);
            });
        });

        describe("valid", () => {
            it("creates new phone number for user", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: PhoneCreateInput = {
                    id: generatePK(),
                    phoneNumber: "+1234567890",
                };

                const result = await phone.createOne({ input }, { req, res }, phone_createOne);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("Phone");
                expect(result.id).toBe(input.id);
                expect(result.phoneNumber).toBe("+1234567890");
                expect(result.isVerified).toBe(false);
                expect(result.user?.id).toBe(testUser[0].id.toString());

                // Verify in database
                const createdPhone = await DbProvider.get().phone.findUnique({
                    where: { id: BigInt(input.id) },
                });
                expect(createdPhone).toBeDefined();
                expect(createdPhone?.phoneNumber).toBe("+1234567890");
                expect(createdPhone?.userId).toEqual(testUser[0].id);
            });

            it("creates phone with country code normalization", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: PhoneCreateInput = {
                    id: generatePK(),
                    phoneNumber: "1234567890", // Without country code
                };

                const result = await phone.createOne({ input }, { req, res }, phone_createOne);

                // Should normalize to include +1 country code for US numbers
                expect(result.phoneNumber).toBe("+1234567890");
            });

            it("creates phone with international number", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: PhoneCreateInput = {
                    id: generatePK(),
                    phoneNumber: "+44123456789", // UK number
                };

                const result = await phone.createOne({ input }, { req, res }, phone_createOne);

                expect(result.phoneNumber).toBe("+44123456789");
            });

            it("allows multiple phones per user", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                // Create first phone
                const input1: PhoneCreateInput = {
                    id: generatePK(),
                    phoneNumber: "+1234567890",
                };

                const result1 = await phone.createOne({ input: input1 }, { req, res }, phone_createOne);
                expect(result1.phoneNumber).toBe("+1234567890");

                // Create second phone
                const input2: PhoneCreateInput = {
                    id: generatePK(),
                    phoneNumber: "+1987654321",
                };

                const result2 = await phone.createOne({ input: input2 }, { req, res }, phone_createOne);
                expect(result2.phoneNumber).toBe("+1987654321");

                // Verify both phones exist for the user
                const userPhones = await DbProvider.get().phone.findMany({
                    where: { userId: testUser[0].id },
                });
                expect(userPhones).toHaveLength(2);
                
                const numbers = userPhones.map(p => p.phoneNumber);
                expect(numbers).toContain("+1234567890");
                expect(numbers).toContain("+1987654321");
            });

            it("creates phone with verification code", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: PhoneCreateInput = {
                    id: generatePK(),
                    phoneNumber: "+1234567890",
                    verificationCode: "123456",
                };

                const result = await phone.createOne({ input }, { req, res }, phone_createOne);

                expect(result.phoneNumber).toBe("+1234567890");

                // Verify code is set in database
                const createdPhone = await DbProvider.get().phone.findUnique({
                    where: { id: BigInt(input.id) },
                });
                expect(createdPhone?.verificationCode).toBe("123456");
                expect(createdPhone?.lastVerificationCodeRequestAttempt).toBeDefined();
            });
        });

        describe("invalid", () => {
            it("rejects duplicate phone number", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create existing phone
                await DbProvider.get().phone.create({
                    data: phoneFactory.createMinimal({
                        phoneNumber: "+1234567890",
                        userId: testUser[0].id,
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: PhoneCreateInput = {
                    id: generatePK(),
                    phoneNumber: "+1234567890", // Duplicate
                };

                await expect(phone.createOne({ input }, { req, res }, phone_createOne))
                    .rejects.toThrow();
            });

            it("rejects invalid phone number format", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: PhoneCreateInput = {
                    id: generatePK(),
                    phoneNumber: "invalid-phone", // Invalid format
                };

                await expect(phone.createOne({ input }, { req, res }, phone_createOne))
                    .rejects.toThrow();
            });

            it("rejects phone number too short", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: PhoneCreateInput = {
                    id: generatePK(),
                    phoneNumber: "+123", // Too short
                };

                await expect(phone.createOne({ input }, { req, res }, phone_createOne))
                    .rejects.toThrow();
            });

            it("rejects phone number too long", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: PhoneCreateInput = {
                    id: generatePK(),
                    phoneNumber: "+123456789012345678901234567890", // Too long
                };

                await expect(phone.createOne({ input }, { req, res }, phone_createOne))
                    .rejects.toThrow();
            });

            it("enforces phone limit per user", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create maximum allowed phones (assume limit is 3)
                const phonePromises = [];
                for (let i = 0; i < 3; i++) {
                    phonePromises.push(
                        DbProvider.get().phone.create({
                            data: phoneFactory.createMinimal({
                                phoneNumber: `+123456789${i}`,
                                userId: testUser[0].id,
                            }),
                        }),
                    );
                }
                await Promise.all(phonePromises);

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: PhoneCreateInput = {
                    id: generatePK(),
                    phoneNumber: "+1999999999", // Would exceed limit
                };

                await expect(phone.createOne({ input }, { req, res }, phone_createOne))
                    .rejects.toThrow(CustomError);
            });
        });
    });

    describe("verify", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertEndpointRequiresAuth(
                    phone.verify,
                    { phoneNumber: "+1234567890" },
                    phone_verify,
                );
            });

            it("requires auth write permissions", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWritePrivatePermissions(["Phone"]), // Wrong permission type
                });

                const input: SendVerificationTextInput = {
                    phoneNumber: "+1234567890",
                };

                await expect(phone.verify({ input }, { req, res }, phone_verify))
                    .rejects.toThrow(CustomError);
            });
        });

        describe("valid", () => {
            it("sends verification text for user's phone", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create phone for the user
                const userPhone = await DbProvider.get().phone.create({
                    data: phoneFactory.createMinimal({
                        phoneNumber: "+1234567890",
                        userId: testUser[0].id,
                        isVerified: false,
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: SendVerificationTextInput = {
                    phoneNumber: "+1234567890",
                };

                const result = await phone.verify({ input }, { req, res }, phone_verify);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("Success");
                expect(result.success).toBe(true);

                // Verify that verification code was set
                const updatedPhone = await DbProvider.get().phone.findUnique({
                    where: { id: userPhone.id },
                });
                expect(updatedPhone?.verificationCode).toBeDefined();
                expect(updatedPhone?.verificationCode).not.toBe("");
                expect(updatedPhone?.lastVerificationCodeRequestAttempt).toBeDefined();
            });

            it("updates existing verification code", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create phone with existing verification code
                const userPhone = await DbProvider.get().phone.create({
                    data: phoneFactory.createMinimal({
                        phoneNumber: "+1234567890",
                        userId: testUser[0].id,
                        verificationCode: "oldcode",
                        lastVerificationCodeRequestAttempt: new Date(Date.now() - 60000), // 1 minute ago
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: SendVerificationTextInput = {
                    phoneNumber: "+1234567890",
                };

                const result = await phone.verify({ input }, { req, res }, phone_verify);

                expect(result.success).toBe(true);

                // Verify that verification code was updated
                const updatedPhone = await DbProvider.get().phone.findUnique({
                    where: { id: userPhone.id },
                });
                expect(updatedPhone?.verificationCode).toBeDefined();
                expect(updatedPhone?.verificationCode).not.toBe("oldcode"); // Should be new code
                
                const timeDiff = Date.now() - new Date(updatedPhone!.lastVerificationCodeRequestAttempt!).getTime();
                expect(timeDiff).toBeLessThan(5000); // Should be very recent (within 5 seconds)
            });

            it("respects rate limiting", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                await DbProvider.get().phone.create({
                    data: phoneFactory.createMinimal({
                        phoneNumber: "+1234567890",
                        userId: testUser[0].id,
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: SendVerificationTextInput = {
                    phoneNumber: "+1234567890",
                };

                // Send multiple verification requests (should succeed up to rate limit)
                await phone.verify({ input }, { req, res }, phone_verify);
                await phone.verify({ input }, { req, res }, phone_verify);
                
                // Should succeed up to rate limit (25 per user)
                const result = await phone.verify({ input }, { req, res }, phone_verify);
                expect(result.success).toBe(true);
            });
        });

        describe("invalid", () => {
            it("rejects verification for non-existent phone", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: SendVerificationTextInput = {
                    phoneNumber: "+1999999999",
                };

                await expect(phone.verify({ input }, { req, res }, phone_verify))
                    .rejects.toThrow(CustomError);
            });

            it("rejects verification for another user's phone", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
                
                // Create phone for user 1
                await DbProvider.get().phone.create({
                    data: phoneFactory.createMinimal({
                        phoneNumber: "+1234567890",
                        userId: testUsers[0].id,
                    }),
                });

                // Try to verify as user 2
                const { req, res } = await mockApiSession({
                    userId: testUsers[1].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: SendVerificationTextInput = {
                    phoneNumber: "+1234567890",
                };

                await expect(phone.verify({ input }, { req, res }, phone_verify))
                    .rejects.toThrow(CustomError);
            });
        });
    });

    describe("validate", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertEndpointRequiresAuth(
                    phone.validate,
                    { phoneNumber: "+1234567890", verificationCode: "123456" },
                    phone_validate,
                );
            });

            it("requires auth write permissions", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWritePrivatePermissions(["Phone"]), // Wrong permission type
                });

                const input: ValidateVerificationTextInput = {
                    phoneNumber: "+1234567890",
                    verificationCode: "123456",
                };

                await expect(phone.validate({ input }, { req, res }, phone_validate))
                    .rejects.toThrow(CustomError);
            });
        });

        describe("valid", () => {
            it("validates correct verification code", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create phone with verification code
                await DbProvider.get().phone.create({
                    data: phoneFactory.createMinimal({
                        phoneNumber: "+1234567890",
                        userId: testUser[0].id,
                        verificationCode: "123456",
                        lastVerificationCodeRequestAttempt: new Date(),
                        isVerified: false,
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: ValidateVerificationTextInput = {
                    phoneNumber: "+1234567890",
                    verificationCode: "123456",
                };

                const result = await phone.validate({ input }, { req, res }, phone_validate);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("Success");
                expect(result.success).toBe(true);

                // Verify phone is now marked as verified
                const verifiedPhone = await DbProvider.get().phone.findFirst({
                    where: { 
                        phoneNumber: "+1234567890",
                        userId: testUser[0].id,
                    },
                });
                expect(verifiedPhone?.isVerified).toBe(true);
                expect(verifiedPhone?.verificationCode).toBeNull(); // Code should be cleared
            });

            it("handles case with recent verification code", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create phone with very recent verification code
                await DbProvider.get().phone.create({
                    data: phoneFactory.createMinimal({
                        phoneNumber: "+1234567890",
                        userId: testUser[0].id,
                        verificationCode: "654321",
                        lastVerificationCodeRequestAttempt: new Date(Date.now() - 30000), // 30 seconds ago
                        isVerified: false,
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: ValidateVerificationTextInput = {
                    phoneNumber: "+1234567890",
                    verificationCode: "654321",
                };

                const result = await phone.validate({ input }, { req, res }, phone_validate);

                expect(result.success).toBe(true);
            });
        });

        describe("invalid", () => {
            it("rejects incorrect verification code", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create phone with verification code
                await DbProvider.get().phone.create({
                    data: phoneFactory.createMinimal({
                        phoneNumber: "+1234567890",
                        userId: testUser[0].id,
                        verificationCode: "123456",
                        lastVerificationCodeRequestAttempt: new Date(),
                        isVerified: false,
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: ValidateVerificationTextInput = {
                    phoneNumber: "+1234567890",
                    verificationCode: "wrong123", // Wrong code
                };

                await expect(phone.validate({ input }, { req, res }, phone_validate))
                    .rejects.toThrow(CustomError);
            });

            it("rejects expired verification code", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create phone with expired verification code
                await DbProvider.get().phone.create({
                    data: phoneFactory.createMinimal({
                        phoneNumber: "+1234567890",
                        userId: testUser[0].id,
                        verificationCode: "123456",
                        lastVerificationCodeRequestAttempt: new Date(Date.now() - 24 * 60 * 60 * 1000), // 24 hours ago
                        isVerified: false,
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: ValidateVerificationTextInput = {
                    phoneNumber: "+1234567890",
                    verificationCode: "123456",
                };

                await expect(phone.validate({ input }, { req, res }, phone_validate))
                    .rejects.toThrow(CustomError);
            });

            it("rejects validation for non-existent phone", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: ValidateVerificationTextInput = {
                    phoneNumber: "+1999999999",
                    verificationCode: "123456",
                };

                await expect(phone.validate({ input }, { req, res }, phone_validate))
                    .rejects.toThrow(CustomError);
            });

            it("rejects validation for another user's phone", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
                
                // Create phone for user 1
                await DbProvider.get().phone.create({
                    data: phoneFactory.createMinimal({
                        phoneNumber: "+1234567890",
                        userId: testUsers[0].id,
                        verificationCode: "123456",
                        lastVerificationCodeRequestAttempt: new Date(),
                    }),
                });

                // Try to validate as user 2
                const { req, res } = await mockApiSession({
                    userId: testUsers[1].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: ValidateVerificationTextInput = {
                    phoneNumber: "+1234567890",
                    verificationCode: "123456",
                };

                await expect(phone.validate({ input }, { req, res }, phone_validate))
                    .rejects.toThrow(CustomError);
            });

            it("rejects validation for already verified phone", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create already verified phone
                await DbProvider.get().phone.create({
                    data: phoneFactory.createMinimal({
                        phoneNumber: "+1234567890",
                        userId: testUser[0].id,
                        isVerified: true,
                        verificationCode: null, // Already verified, no code
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: ValidateVerificationTextInput = {
                    phoneNumber: "+1234567890",
                    verificationCode: "123456",
                };

                await expect(phone.validate({ input }, { req, res }, phone_validate))
                    .rejects.toThrow(CustomError);
            });
        });
    });
});

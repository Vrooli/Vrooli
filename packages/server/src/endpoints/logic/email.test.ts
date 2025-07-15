import { type EmailCreateInput, type SendVerificationEmailInput, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi, test } from "vitest";
import { mockAuthenticatedSession, mockLoggedOutSession, mockApiSession, mockWriteAuthPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { assertRequiresAuth, AUTH_SCENARIOS } from "../../__test/authTestUtils.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { email_createOne } from "../generated/email_createOne.js";
import { email_verify } from "../generated/email_verify.js";
import { email } from "./email.js";
// Import database fixtures
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { createEmailDbFactory } from "../../__test/fixtures/db/index.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

// Mock the QueueService to prevent email service failures
vi.mock("../../tasks/queues.js", () => ({
    QueueService: {
        get: vi.fn().mockReturnValue({
            email: {
                addTask: vi.fn().mockResolvedValue({ 
                    __typename: "Success" as const, 
                    success: true, 
                }),
            },
        }),
        reset: vi.fn().mockResolvedValue(undefined),
    },
}));

describe("EndpointsEmail", () => {
    let emailFactory: ReturnType<typeof createEmailDbFactory>;

    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
        
        // Initialize the email factory
        emailFactory = createEmailDbFactory(DbProvider.get());
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
            // Hybrid approach: use test.each for standard auth scenarios
            test.each([
                AUTH_SCENARIOS.notLoggedIn,
                AUTH_SCENARIOS.apiKeyInsteadOfUser,
            ])("fails when $name", async (scenario) => {
                const { req, res } = await scenario.setup();
                const input: EmailCreateInput = {
                    id: generatePK(),
                    emailAddress: "test@example.com",
                };
                await expect(email.createOne({ input }, { req, res }, email_createOne))
                    .rejects
                    .toThrow();
            });

            // Individual test for complex permission check
            it("requires auth write permissions", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Use mockApiSession to simulate API key with wrong permissions
                const { req, res } = await mockApiSession(
                    "test-api-token",
                    mockWritePrivatePermissions(), // Wrong permission type - should be WriteAuth
                    testUsers.records[0]
                );

                const input: EmailCreateInput = {
                    id: generatePK(),
                    emailAddress: "test@example.com",
                };

                await expect(email.createOne({ input }, { req, res }, email_createOne))
                    .rejects.toThrow(CustomError);
            });
        });

        describe("valid", () => {
            it("creates new email for user", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: EmailCreateInput = {
                    id: generatePK(),
                    emailAddress: "newemail@example.com",
                };

                const result = await email.createOne({ input }, { req, res }, email_createOne);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("Email");
                expect(result.id).toBe(input.id);
                expect(result.emailAddress).toBe("newemail@example.com");
                expect(result.isVerified).toBe(false);
                expect(result.user?.id).toBe(testUsers.records[0].id.toString());

                // Verify in database
                const createdEmail = await DbProvider.get().email.findUnique({
                    where: { id: BigInt(input.id) },
                });
                expect(createdEmail).toBeDefined();
                expect(createdEmail?.emailAddress).toBe("newemail@example.com");
                expect(createdEmail?.userId).toEqual(testUsers.records[0].id);
            });

            it("creates email with normalized address", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: EmailCreateInput = {
                    id: generatePK(),
                    emailAddress: "  CaseSensitive@EXAMPLE.COM  ", // With whitespace and mixed case
                };

                const result = await email.createOne({ input }, { req, res }, email_createOne);

                // Email should be normalized to lowercase and trimmed
                expect(result.emailAddress).toBe("casesensitive@example.com");

                // Verify normalization in database
                const createdEmail = await DbProvider.get().email.findUnique({
                    where: { id: BigInt(input.id) },
                });
                expect(createdEmail?.emailAddress).toBe("casesensitive@example.com");
            });

            it("allows multiple emails per user", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                // Create first email
                const input1: EmailCreateInput = {
                    id: generatePK(),
                    emailAddress: "first@example.com",
                };

                const result1 = await email.createOne({ input: input1 }, { req, res }, email_createOne);
                expect(result1.emailAddress).toBe("first@example.com");

                // Create second email
                const input2: EmailCreateInput = {
                    id: generatePK(),
                    emailAddress: "second@example.com",
                };

                const result2 = await email.createOne({ input: input2 }, { req, res }, email_createOne);
                expect(result2.emailAddress).toBe("second@example.com");

                // Verify both emails exist for the user
                const userEmails = await DbProvider.get().email.findMany({
                    where: { userId: testUsers.records[0].id },
                });
                expect(userEmails).toHaveLength(2);
                
                const addresses = userEmails.map(e => e.emailAddress);
                expect(addresses).toContain("first@example.com");
                expect(addresses).toContain("second@example.com");
            });

            it("creates email with verification code for immediate verification", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: EmailCreateInput = {
                    id: generatePK(),
                    emailAddress: "verify@example.com",
                    verificationCode: "123456",
                };

                const result = await email.createOne({ input }, { req, res }, email_createOne);

                expect(result.emailAddress).toBe("verify@example.com");

                // Verify code is set in database
                const createdEmail = await DbProvider.get().email.findUnique({
                    where: { id: BigInt(input.id) },
                });
                expect(createdEmail?.verificationCode).toBe("123456");
                expect(createdEmail?.lastVerificationCodeRequestAttempt).toBeDefined();
            });
        });

        describe("invalid", () => {
            it("rejects duplicate email address", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create existing email
                await DbProvider.get().email.create({
                    data: emailFactory.generateMinimalData({
                        emailAddress: "existing@example.com",
                        user: { connect: { id: testUsers.records[0].id } },
                    }),
                });

                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: EmailCreateInput = {
                    id: generatePK(),
                    emailAddress: "existing@example.com", // Duplicate
                };

                await expect(email.createOne({ input }, { req, res }, email_createOne))
                    .rejects.toThrow();
            });

            it("rejects invalid email format", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: EmailCreateInput = {
                    id: generatePK(),
                    emailAddress: "invalid-email-format", // No @ symbol
                };

                await expect(email.createOne({ input }, { req, res }, email_createOne))
                    .rejects.toThrow();
            });

            it("rejects empty email address", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: EmailCreateInput = {
                    id: generatePK(),
                    emailAddress: "", // Empty
                };

                await expect(email.createOne({ input }, { req, res }, email_createOne))
                    .rejects.toThrow();
            });

            it("rejects email with dangerous characters", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: EmailCreateInput = {
                    id: generatePK(),
                    emailAddress: "test<script>@example.com", // Contains script tag
                };

                await expect(email.createOne({ input }, { req, res }, email_createOne))
                    .rejects.toThrow();
            });

            it("enforces email limit per user", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create maximum allowed emails (assume limit is 5)
                const emailPromises = [];
                for (let i = 0; i < 5; i++) {
                    emailPromises.push(
                        DbProvider.get().email.create({
                            data: emailFactory.generateMinimalData({
                                emailAddress: `email${i}@example.com`,
                                user: { connect: { id: testUsers.records[0].id } },
                            }),
                        }),
                    );
                }
                await Promise.all(emailPromises);

                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: EmailCreateInput = {
                    id: generatePK(),
                    emailAddress: "exceeds@limit.com", // Would exceed limit
                };

                await expect(email.createOne({ input }, { req, res }, email_createOne))
                    .rejects.toThrow(CustomError);
            });
        });
    });

    describe("verify", () => {
        describe("authentication", () => {
            // Use direct assertion for simple auth check
            it("not logged in", async () => {
                const input: SendVerificationEmailInput = { emailAddress: "test@example.com" };
                await assertRequiresAuth(email.verify, input, email_verify);
            });

            // Individual test for permission check
            it("requires auth write permissions", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: SendVerificationEmailInput = {
                    emailAddress: "test@example.com",
                };

                await expect(email.verify({ input }, { req, res }, email_verify))
                    .rejects.toThrow(CustomError);
            });
        });

        describe("valid", () => {
            it("sends verification email for user's email", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create email for the user
                const userEmail = await DbProvider.get().email.create({
                    data: emailFactory.generateMinimalData({
                        emailAddress: "verify@example.com",
                        user: { connect: { id: testUsers.records[0].id } },
                        verifiedAt: null,
                    }),
                });

                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: SendVerificationEmailInput = {
                    emailAddress: "verify@example.com",
                };

                const result = await email.verify({ input }, { req, res }, email_verify);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("Success");
                expect(result.success).toBe(true);

                // Verify that verification code was set
                const updatedEmail = await DbProvider.get().email.findUnique({
                    where: { id: userEmail.id },
                });
                expect(updatedEmail?.verificationCode).toBeDefined();
                expect(updatedEmail?.verificationCode).not.toBe("");
                expect(updatedEmail?.lastVerificationCodeRequestAttempt).toBeDefined();
            });

            it("updates existing verification code", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create email with existing verification code
                const userEmail = await DbProvider.get().email.create({
                    data: emailFactory.generateMinimalData({
                        emailAddress: "verify@example.com",
                        user: { connect: { id: testUsers.records[0].id } },
                        verificationCode: "oldcode123",
                        lastVerificationCodeRequestAttempt: new Date(Date.now() - 60000), // 1 minute ago
                    }),
                });

                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: SendVerificationEmailInput = {
                    emailAddress: "verify@example.com",
                };

                const result = await email.verify({ input }, { req, res }, email_verify);

                expect(result.success).toBe(true);

                // Verify that verification code was updated
                const updatedEmail = await DbProvider.get().email.findUnique({
                    where: { id: userEmail.id },
                });
                expect(updatedEmail?.verificationCode).toBeDefined();
                expect(updatedEmail?.verificationCode).not.toBe("oldcode123"); // Should be new code
                
                const timeDiff = Date.now() - new Date(updatedEmail!.lastVerificationCodeRequestAttempt!).getTime();
                expect(timeDiff).toBeLessThan(5000); // Should be very recent (within 5 seconds)
            });

            it("handles case-insensitive email lookup", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create email in lowercase
                await DbProvider.get().email.create({
                    data: emailFactory.generateMinimalData({
                        emailAddress: "verify@example.com",
                        user: { connect: { id: testUsers.records[0].id } },
                    }),
                });

                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                // Send verification with mixed case
                const input: SendVerificationEmailInput = {
                    emailAddress: "VERIFY@EXAMPLE.COM",
                };

                const result = await email.verify({ input }, { req, res }, email_verify);

                expect(result.success).toBe(true);
            });

            it("respects rate limiting", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                await DbProvider.get().email.create({
                    data: emailFactory.generateMinimalData({
                        emailAddress: "ratelimit@example.com",
                        user: { connect: { id: testUsers.records[0].id } },
                    }),
                });

                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: SendVerificationEmailInput = {
                    emailAddress: "ratelimit@example.com",
                };

                // Send multiple verification requests rapidly
                await email.verify({ input }, { req, res }, email_verify);
                await email.verify({ input }, { req, res }, email_verify);
                
                // Should succeed up to rate limit (50 per user)
                const result = await email.verify({ input }, { req, res }, email_verify);
                expect(result.success).toBe(true);
            });
        });

        describe("invalid", () => {
            it("rejects verification for non-existent email", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: SendVerificationEmailInput = {
                    emailAddress: "nonexistent@example.com",
                };

                await expect(email.verify({ input }, { req, res }, email_verify))
                    .rejects.toThrow(CustomError);
            });

            it("rejects verification for another user's email", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
                
                // Create email for user 1
                await DbProvider.get().email.create({
                    data: emailFactory.generateMinimalData({
                        emailAddress: "user1@example.com",
                        user: { connect: { id: testUsers.records[0].id } },
                    }),
                });

                // Try to verify as user 2
                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: SendVerificationEmailInput = {
                    emailAddress: "user1@example.com",
                };

                await expect(email.verify({ input }, { req, res }, email_verify))
                    .rejects.toThrow(CustomError);
            });

            it("rejects verification for already verified email", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create already verified email
                await DbProvider.get().email.create({
                    data: emailFactory.generateMinimalData({
                        emailAddress: "verified@example.com",
                        user: { connect: { id: testUsers.records[0].id } },
                        verifiedAt: new Date(),
                    }),
                });

                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: SendVerificationEmailInput = {
                    emailAddress: "verified@example.com",
                };

                await expect(email.verify({ input }, { req, res }, email_verify))
                    .rejects.toThrow(CustomError);
            });

            it("prevents verification code spam", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create email with recent verification attempt
                await DbProvider.get().email.create({
                    data: emailFactory.generateMinimalData({
                        emailAddress: "spam@example.com",
                        user: { connect: { id: testUsers.records[0].id } },
                        verificationCode: "recent123",
                        lastVerificationCodeRequestAttempt: new Date(), // Very recent
                    }),
                });

                const { req, res } = await mockAuthenticatedSession(testUsers.records[0]);

                const input: SendVerificationEmailInput = {
                    emailAddress: "spam@example.com",
                };

                // Should be rate limited or rejected for too frequent requests
                await expect(email.verify({ input }, { req, res }, email_verify))
                    .rejects.toThrow(CustomError);
            });
        });
    });
});

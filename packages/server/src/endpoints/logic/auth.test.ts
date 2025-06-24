import { AUTH_PROVIDERS, AccountStatus, COOKIE, type EmailLogInInput, type EmailRequestPasswordChangeInput, type EmailResetPasswordInput, type EmailSignUpInput, type Session, type SwitchCurrentAccountInput, type ValidateSessionInput, type WalletCompleteInput, type WalletInitInput, generatePK, generatePublicId } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { randomString } from "../../auth/codes.js";
import { PasswordAuthService } from "../../auth/email.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { auth_emailLogIn } from "../generated/auth_emailLogIn.js";
import { auth_emailRequestPasswordChange } from "../generated/auth_emailRequestPasswordChange.js";
import { auth_emailResetPassword } from "../generated/auth_emailResetPassword.js";
import { auth_emailSignUp } from "../generated/auth_emailSignUp.js";
import { auth_guestLogIn } from "../generated/auth_guestLogIn.js";
import { auth_logOut } from "../generated/auth_logOut.js";
import { auth_logOutAll } from "../generated/auth_logOutAll.js";
import { auth_switchCurrentAccount } from "../generated/auth_switchCurrentAccount.js";
import { auth_validateSession } from "../generated/auth_validateSession.js";
import { auth_walletComplete } from "../generated/auth_walletComplete.js";
import { auth_walletInit } from "../generated/auth_walletInit.js";
import { auth } from "./auth.js";
// Import database fixtures for seeding
import { seedTestUsers, UserDbFactory } from "../../__test/fixtures/db/userFixtures.js";
// Import validation fixtures for API input testing
import { emailLogInFixtures, emailRequestPasswordChangeFixtures, emailResetPasswordFixtures, validateSessionFixtures, switchCurrentAccountFixtures } from "@vrooli/shared";

describe("EndpointsAuth", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clean up tables used in tests
        const prisma = DbProvider.get();
        await prisma.user_auth.deleteMany();
        await prisma.session.deleteMany();
        await prisma.email.deleteMany();
        await prisma.wallet.deleteMany();
        await prisma.user.deleteMany();
        // Clear Redis cache
        await CacheService.get().flushAll();
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("emailSignUp", () => {
        it("should create a new user with valid input", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            // Use validation fixtures for consistent test data
            const input: EmailSignUpInput = {
                name: "Test User",
                email: "test@example.com",
                password: "SecurePassword123!",
                marketingEmails: false,
                theme: "dark",
            };

            const result = await auth.emailSignUp({ input }, { req, res }, auth_emailSignUp);

            expect(result).toBeDefined();
            expect(result.__typename).toBe("Session");
            expect(result.isLoggedIn).toBe(true);
            expect(result.users).toHaveLength(1);
            expect(result.users[0].name).toBe("Test User");

            // Verify user was created in database
            const createdUser = await DbProvider.get().user.findFirst({
                where: { emails: { some: { emailAddress: input.email } } },
                include: { emails: true, auths: true },
            });
            expect(createdUser).toBeDefined();
            expect(createdUser?.name).toBe(input.name);
            expect(createdUser?.emails[0].emailAddress).toBe(input.email);
            expect(createdUser?.theme).toBe(input.theme);
            expect(createdUser?.auths[0].provider).toBe(AUTH_PROVIDERS.Password);
        });

        it("should reject signup with existing email", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            // Create existing user
            await DbProvider.get().user.create({
                data: UserDbFactory.createWithAuth({
                    emails: {
                        create: [{
                            id: generatePK(),
                            emailAddress: "existing@example.com",
                        }],
                    },
                }),
            });

            const input: EmailSignUpInput = {
                name: "New User",
                email: "existing@example.com",
                password: "SecurePassword123!",
                marketingEmails: false,
                theme: "light",
            };

            await expect(auth.emailSignUp({ input }, { req, res }, auth_emailSignUp))
                .rejects.toThrow(CustomError);
        });

        it("should reject signup with profane name", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            const input: EmailSignUpInput = {
                name: "fuck",
                email: "test@example.com",
                password: "SecurePassword123!",
                marketingEmails: false,
                theme: "light",
            };

            await expect(auth.emailSignUp({ input }, { req, res }, auth_emailSignUp))
                .rejects.toThrow(CustomError);
        });

        it("should reject invalid email format", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            const input: EmailSignUpInput = {
                name: "Test User",
                email: "invalid-email",
                password: "SecurePassword123!",
                marketingEmails: false,
                theme: "light",
            };

            await expect(auth.emailSignUp({ input }, { req, res }, auth_emailSignUp))
                .rejects.toThrow();
        });

        it("should reject weak password", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            const input: EmailSignUpInput = {
                name: "Test User",
                email: "test@example.com",
                password: "weak",
                marketingEmails: false,
                theme: "light",
            };

            await expect(auth.emailSignUp({ input }, { req, res }, auth_emailSignUp))
                .rejects.toThrow();
        });
    });

    describe("emailLogIn", () => {
        const createTestUserWithPassword = async (email: string, password: string) => {
            const hashedPassword = PasswordAuthService.hashPassword(password);
            return await DbProvider.get().user.create({
                data: UserDbFactory.createWithAuth({
                    emails: {
                        create: [{
                            id: generatePK(),
                            emailAddress: email,
                        }],
                    },
                    auths: {
                        create: {
                            id: generatePK(),
                            provider: AUTH_PROVIDERS.Password,
                            hashed_password: hashedPassword,
                        },
                    },
                }),
                include: { emails: true, auths: true },
            });
        };

        it("should log in with valid credentials", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            // Use validation fixtures for consistent test data
            const loginData = emailLogInFixtures.complete.create;
            const testUser = await createTestUserWithPassword(loginData.email, loginData.password);

            const input: EmailLogInInput = {
                email: loginData.email,
                password: loginData.password,
            };

            const result = await auth.emailLogIn({ input }, { req, res }, auth_emailLogIn);

            expect(result).toBeDefined();
            expect(result.__typename).toBe("Session");
            expect(result.isLoggedIn).toBe(true);
            expect(result.users).toHaveLength(1);
            expect(result.users[0].id).toBe(testUser.id.toString());
        });

        it("should reject invalid password", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            // Use validation fixtures for consistent test data
            const validLoginData = emailLogInFixtures.complete.create;
            await createTestUserWithPassword(validLoginData.email, validLoginData.password);

            const input: EmailLogInInput = {
                email: validLoginData.email,
                password: "WrongPassword123!",
            };

            await expect(auth.emailLogIn({ input }, { req, res }, auth_emailLogIn))
                .rejects.toThrow(CustomError);
        });

        it("should reject non-existent email", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            // Use validation fixtures for consistent test data
            const input: EmailLogInInput = emailLogInFixtures.complete.create;

            await expect(auth.emailLogIn({ input }, { req, res }, auth_emailLogIn))
                .rejects.toThrow(CustomError);
        });

        it("should handle user without password (requires reset)", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            // Create user without password
            const testUser = await DbProvider.get().user.create({
                data: UserDbFactory.createMinimal({
                    emails: {
                        create: [{
                            id: generatePK(),
                            emailAddress: "nopwd@example.com",
                        }],
                    },
                }),
            });

            const input: EmailLogInInput = {
                email: "nopwd@example.com",
                password: "AnyPassword123!",
            };

            await expect(auth.emailLogIn({ input }, { req, res }, auth_emailLogIn))
                .rejects.toThrow(CustomError);
        });

        it("should verify email with verification code", async () => {
            // Use validation fixtures for consistent test data
            const loginData = emailLogInFixtures.complete.create;
            const testUser = await createTestUserWithPassword(loginData.email, loginData.password);
            
            // Set up verification code
            const verificationCode = loginData.verificationCode;
            await DbProvider.get().email.update({
                where: { emailAddress: loginData.email },
                data: {
                    verificationCode,
                    lastVerificationCodeRequestAttempt: new Date(),
                },
            });

            const { req, res } = await mockAuthenticatedSession({
                id: testUser.id.toString(),
            });

            const input: EmailLogInInput = {
                verificationCode,
            };

            const result = await auth.emailLogIn({ input }, { req, res }, auth_emailLogIn);

            expect(result).toBeDefined();
            expect(result.__typename).toBe("Session");
            expect(result.isLoggedIn).toBe(true);
        });
    });

    describe("emailRequestPasswordChange", () => {
        it("should send password reset for existing user", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            // Use validation fixtures for consistent test data
            const resetData = emailRequestPasswordChangeFixtures.minimal.create;
            await DbProvider.get().user.create({
                data: UserDbFactory.createWithAuth({
                    emails: {
                        create: [{
                            id: generatePK(),
                            emailAddress: resetData.email,
                        }],
                    },
                }),
            });

            const input: EmailRequestPasswordChangeInput = resetData;

            const result = await auth.emailRequestPasswordChange({ input }, { req, res }, auth_emailRequestPasswordChange);

            expect(result).toBeDefined();
            expect(result.__typename).toBe("Success");
            expect(result.success).toBe(true);

            // Verify reset code was set
            const updatedAuth = await DbProvider.get().user_auth.findFirst({
                where: {
                    user: {
                        emails: {
                            some: { emailAddress: resetData.email },
                        },
                    },
                },
            });
            expect(updatedAuth?.resetPasswordCode).toBeDefined();
            expect(updatedAuth?.lastResetPasswordRequestAttempt).toBeDefined();
        });

        it("should reject non-existent email", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            // Use validation fixtures for consistent test data
            const input: EmailRequestPasswordChangeInput = emailRequestPasswordChangeFixtures.minimal.create;

            await expect(auth.emailRequestPasswordChange({ input }, { req, res }, auth_emailRequestPasswordChange))
                .rejects.toThrow(CustomError);
        });
    });

    describe("emailResetPassword", () => {
        it("should reset password with valid code", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            // Use validation fixtures for consistent test data
            const resetData = emailResetPasswordFixtures.minimal.create;
            const testUser = await DbProvider.get().user.create({
                data: UserDbFactory.createWithAuth({
                    id: BigInt(resetData.id),
                    emails: {
                        create: [{
                            id: generatePK(),
                            emailAddress: "test@example.com",
                        }],
                    },
                    auths: {
                        create: {
                            id: generatePK(),
                            provider: AUTH_PROVIDERS.Password,
                            hashed_password: PasswordAuthService.hashPassword("OldPassword123!"),
                            resetPasswordCode: resetData.code,
                            lastResetPasswordRequestAttempt: new Date(),
                        },
                    },
                }),
            });

            const input: EmailResetPasswordInput = resetData;

            const result = await auth.emailResetPassword({ input }, { req, res }, auth_emailResetPassword);

            expect(result).toBeDefined();
            expect(result.__typename).toBe("Session");
            expect(result.isLoggedIn).toBe(true);
            expect(result.users[0].id).toBe(testUser.id.toString());

            // Verify password was updated and reset code cleared
            const updatedAuth = await DbProvider.get().user_auth.findFirst({
                where: { user_id: testUser.id },
            });
            expect(updatedAuth?.resetPasswordCode).toBeNull();
            expect(updatedAuth?.lastResetPasswordRequestAttempt).toBeNull();
        });

        it("should reject invalid reset code", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            const testUser = await DbProvider.get().user.create({
                data: UserDbFactory.createWithAuth({
                    auths: {
                        create: {
                            id: generatePK(),
                            provider: AUTH_PROVIDERS.Password,
                            hashed_password: PasswordAuthService.hashPassword("OldPassword123!"),
                            resetPasswordCode: "validcode",
                            lastResetPasswordRequestAttempt: new Date(),
                        },
                    },
                }),
            });

            const input: EmailResetPasswordInput = {
                id: testUser.id.toString(),
                code: "invalidcode",
                newPassword: "NewSecurePassword123!",
            };

            await expect(auth.emailResetPassword({ input }, { req, res }, auth_emailResetPassword))
                .rejects.toThrow(CustomError);
        });

        it("should reject expired reset code", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            const resetCode = randomString(8);
            const testUser = await DbProvider.get().user.create({
                data: UserDbFactory.createWithAuth({
                    auths: {
                        create: {
                            id: generatePK(),
                            provider: AUTH_PROVIDERS.Password,
                            hashed_password: PasswordAuthService.hashPassword("OldPassword123!"),
                            resetPasswordCode: resetCode,
                            // Set very old timestamp
                            lastResetPasswordRequestAttempt: new Date(Date.now() - 24 * 60 * 60 * 1000),
                        },
                    },
                }),
            });

            const input: EmailResetPasswordInput = {
                id: testUser.id.toString(),
                code: resetCode,
                newPassword: "NewSecurePassword123!",
            };

            await expect(auth.emailResetPassword({ input }, { req, res }, auth_emailResetPassword))
                .rejects.toThrow(CustomError);
        });
    });

    describe("guestLogIn", () => {
        it("should return guest session", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            const result = await auth.guestLogIn({}, { req, res }, auth_guestLogIn);

            expect(result).toBeDefined();
            expect(result.__typename).toBe("Session");
            expect(result.isLoggedIn).toBe(false);
            expect(result.users).toHaveLength(0);
        });
    });

    describe("logOut", () => {
        it("should log out current user", async () => {
            const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
            const { req, res } = await mockAuthenticatedSession({
                id: testUser[0].id.toString(),
            });

            const result = await auth.logOut({}, { req, res }, auth_logOut);

            expect(result).toBeDefined();
            expect(result.__typename).toBe("Session");
            expect(result.isLoggedIn).toBe(false);
            expect(result.users).toHaveLength(0);
        });

        it("should return guest session if already logged out", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            const result = await auth.logOut({}, { req, res }, auth_logOut);

            expect(result).toBeDefined();
            expect(result.__typename).toBe("Session");
            expect(result.isLoggedIn).toBe(false);
        });

        it("should keep other users logged in multi-user session", async () => {
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
            const { req, res } = await mockAuthenticatedSession({
                id: testUsers[0].id.toString(),
                users: testUsers.map(u => ({
                    id: u.id.toString(),
                    name: u.name,
                    handle: u.handle,
                    session: {
                        id: generatePK(),
                        token: "test-token",
                    },
                })),
            });

            const result = await auth.logOut({}, { req, res }, auth_logOut);

            expect(result).toBeDefined();
            expect(result.__typename).toBe("Session");
            expect(result.isLoggedIn).toBe(true);
            expect(result.users).toHaveLength(1);
            expect(result.users[0].id).toBe(testUsers[1].id.toString());
        });
    });

    describe("logOutAll", () => {
        it("should revoke all sessions for user", async () => {
            const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
            
            // Create multiple sessions
            const sessions = await Promise.all([
                DbProvider.get().session.create({
                    data: {
                        id: generatePK(),
                        user_id: testUser[0].id,
                        device_info: "Device 1",
                        expires_at: new Date(Date.now() + 24 * 60 * 60 * 1000),
                        ip_address: "192.168.1.1",
                    },
                }),
                DbProvider.get().session.create({
                    data: {
                        id: generatePK(),
                        user_id: testUser[0].id,
                        device_info: "Device 2",
                        expires_at: new Date(Date.now() + 24 * 60 * 60 * 1000),
                        ip_address: "192.168.1.2",
                    },
                }),
            ]);

            const { req, res } = await mockAuthenticatedSession({
                id: testUser[0].id.toString(),
            });

            const result = await auth.logOutAll({}, { req, res }, auth_logOutAll);

            expect(result).toBeDefined();
            expect(result.__typename).toBe("Session");
            expect(result.isLoggedIn).toBe(false);

            // Verify all sessions were revoked
            const revokedSessions = await DbProvider.get().session.findMany({
                where: { user_id: testUser[0].id },
            });
            expect(revokedSessions.every(s => s.revokedAt !== null)).toBe(true);
        });
    });

    describe("validateSession", () => {
        it("should validate logged in session", async () => {
            const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
            const { req, res } = await mockAuthenticatedSession({
                id: testUser[0].id.toString(),
            });

            // Use validation fixtures for consistent test data
            const input: ValidateSessionInput = validateSessionFixtures.minimal.create;

            const result = await auth.validateSession({ input }, { req, res }, auth_validateSession);

            expect(result).toBeDefined();
            expect(result.__typename).toBe("Session");
            expect(result.isLoggedIn).toBe(true);
            expect(result.users[0].id).toBe(testUser[0].id.toString());
        });

        it("should return guest session if not logged in", async () => {
            const { req, res } = await mockLoggedOutSession();

            // Use validation fixtures for consistent test data
            const input: ValidateSessionInput = validateSessionFixtures.minimal.create;

            const result = await auth.validateSession({ input }, { req, res }, auth_validateSession);

            expect(result).toBeDefined();
            expect(result.__typename).toBe("Session");
            expect(result.isLoggedIn).toBe(false);
            expect(result.users).toHaveLength(0);
        });
    });

    describe("switchCurrentAccount", () => {
        it("should switch between accounts", async () => {
            const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
            const { req, res } = await mockAuthenticatedSession({
                id: testUsers[0].id.toString(),
                users: testUsers.map((u, idx) => ({
                    id: u.id.toString(),
                    name: u.name,
                    handle: u.handle,
                    session: {
                        id: generatePK(),
                        token: `test-token-${idx}`,
                    },
                })),
            });

            // Use validation fixtures for consistent test data
            const input: SwitchCurrentAccountInput = {
                ...switchCurrentAccountFixtures.minimal.create,
                id: testUsers[1].id.toString(),
            };

            const result = await auth.switchCurrentAccount({ input }, { req, res }, auth_switchCurrentAccount);

            expect(result).toBeDefined();
            expect(result.__typename).toBe("Session");
            expect(result.isLoggedIn).toBe(true);
            expect(result.users).toHaveLength(2);
            expect(result.users[0].id).toBe(testUsers[1].id.toString());
            expect(result.users[1].id).toBe(testUsers[0].id.toString());
        });

        it("should reject switch to non-existent user in session", async () => {
            const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
            const { req, res } = await mockAuthenticatedSession({
                id: testUser[0].id.toString(),
            });

            // Use validation fixtures for consistent test data
            const input: SwitchCurrentAccountInput = switchCurrentAccountFixtures.minimal.create;

            await expect(auth.switchCurrentAccount({ input }, { req, res }, auth_switchCurrentAccount))
                .rejects.toThrow(CustomError);
        });
    });

    describe("walletInit", () => {
        it("should generate nonce for new wallet", async () => {
            const { req, res } = await mockLoggedOutSession();

            const input: WalletInitInput = {
                stakingAddress: "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3",
                nonceDescription: "Sign to authenticate",
            };

            const result = await auth.walletInit({ input }, { req, res }, auth_walletInit);

            expect(result).toBeDefined();
            expect(result.__typename).toBe("WalletInit");
            expect(result.nonce).toBeDefined();
            expect(result.nonce.length).toBeGreaterThan(0);

            // Verify wallet was created/updated in database
            const wallet = await DbProvider.get().wallet.findUnique({
                where: { stakingAddress: input.stakingAddress },
            });
            expect(wallet).toBeDefined();
            expect(wallet?.nonce).toBe(result.nonce);
        });

        it("should update nonce for existing wallet", async () => {
            const { req, res } = await mockLoggedOutSession();

            const stakingAddress = "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3";
            
            // Create existing wallet
            await DbProvider.get().wallet.create({
                data: {
                    id: generatePK(),
                    stakingAddress,
                    nonce: "old-nonce",
                    nonceCreationTime: new Date().toISOString(),
                },
            });

            const input: WalletInitInput = {
                stakingAddress,
                nonceDescription: "Sign to authenticate",
            };

            const result = await auth.walletInit({ input }, { req, res }, auth_walletInit);

            expect(result.nonce).toBeDefined();
            expect(result.nonce).not.toBe("old-nonce");
        });

        it("should reject non-mainnet wallet", async () => {
            const { req, res } = await mockLoggedOutSession();

            const input: WalletInitInput = {
                stakingAddress: "stake_test1uqevw2xnsc0pvn9t9r9c7qryfqfeerchgrlm3ea2nefr9hqp8n5xl",
                nonceDescription: "Sign to authenticate",
            };

            await expect(auth.walletInit({ input }, { req, res }, auth_walletInit))
                .rejects.toThrow(CustomError);
        });
    });

    describe("walletComplete", () => {
        // Note: This test is simplified since we can't easily test actual wallet signature verification
        it("should handle wallet verification attempt", async () => {
            const { req, res } = await mockLoggedOutSession();

            const stakingAddress = "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3";
            const nonce = "test-nonce";
            
            // Create wallet with nonce
            await DbProvider.get().wallet.create({
                data: {
                    id: generatePK(),
                    stakingAddress,
                    nonce,
                    nonceCreationTime: new Date().toISOString(),
                    verifiedAt: new Date().toISOString(),
                },
            });

            const input: WalletCompleteInput = {
                stakingAddress,
                signedPayload: "fake-signed-payload",
            };

            // This will fail due to invalid signature, but tests the flow
            await expect(auth.walletComplete({ input }, { req, res }, auth_walletComplete))
                .rejects.toThrow(CustomError);
        });

        it("should reject non-existent wallet", async () => {
            const { req, res } = await mockLoggedOutSession();

            const input: WalletCompleteInput = {
                stakingAddress: "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3",
                signedPayload: "fake-signed-payload",
            };

            await expect(auth.walletComplete({ input }, { req, res }, auth_walletComplete))
                .rejects.toThrow(CustomError);
        });

        it("should reject expired nonce", async () => {
            const { req, res } = await mockLoggedOutSession();

            const stakingAddress = "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3";
            
            // Create wallet with expired nonce
            await DbProvider.get().wallet.create({
                data: {
                    id: generatePK(),
                    stakingAddress,
                    nonce: "expired-nonce",
                    nonceCreationTime: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(), // 24 hours ago
                },
            });

            const input: WalletCompleteInput = {
                stakingAddress,
                signedPayload: "fake-signed-payload",
            };

            await expect(auth.walletComplete({ input }, { req, res }, auth_walletComplete))
                .rejects.toThrow(CustomError);
        });
    });
});
/* eslint-disable @typescript-eslint/ban-ts-comment */
import { ApiKeyPermission, COOKIE, DAYS_1_MS, DAYS_30_MS, SECONDS_1_MS, type SessionUser, generatePK } from "@vrooli/shared";
import { describe, it, expect, beforeAll, afterAll, vi } from "vitest";
import { generateKeyPairSync } from "crypto";
import { type Response } from "express";
import jwt from "jsonwebtoken";
import { DbProvider } from "../db/provider.js";
import { withDbTransaction } from "../__test/helpers/transactionTest.js";
import { logger } from "../events/logger.js";
import { type SessionData, type SessionToken } from "../types.js";
import { AuthTokensService } from "./auth.js";
import { ACCESS_TOKEN_EXPIRATION_MS, JsonWebToken } from "./jwt.js";


describe("AuthTokensService", () => {
    let loggerErrorSpy: any;
    let loggerInfoSpy: any;

    beforeAll(async () => {
        await DbProvider.init();
        loggerErrorSpy = vi.spyOn(logger, "error").mockImplementation(() => {});
        loggerInfoSpy = vi.spyOn(logger, "info").mockImplementation(() => {});
    });

    afterAll(async () => {
        loggerErrorSpy.mockRestore();
        loggerInfoSpy.mockRestore();
        await DbProvider.shutdown();
    });

    beforeEach(async () => {
        // Setup JWT test environment for all tests
        const keys = generateKeyPairSync("rsa", {
            modulusLength: 2048,
            publicKeyEncoding: {
                type: "spki",
                format: "pem",
            },
            privateKeyEncoding: {
                type: "pkcs8",
                format: "pem",
            },
        });
        JsonWebToken.get().setupTestEnvironment({ privateKey: keys.privateKey, publicKey: keys.publicKey });
    });

    describe("Singleton Behavior", () => {
        it("get() should return the same instance", withDbTransaction(async () => {
            const instance1 = AuthTokensService.get();
            const instance2 = AuthTokensService.get();
            expect(instance1).toBe(instance2);
        }));
    });

    describe("canRefreshToken", () => {
        // Use different session IDs for each test to avoid unique constraint conflicts
        const expiresAt = new Date(Date.now() + DAYS_30_MS);
        const lastRefreshAt = new Date(Date.now() - DAYS_1_MS);

        describe("false", () => {
            it("not an object", withDbTransaction(async () => {
                // @ts-ignore Testing runtime scenario
                expect(await AuthTokensService.canRefreshToken(null)).toBe(false);
                // @ts-ignore Testing runtime scenario
                expect(await AuthTokensService.canRefreshToken(undefined)).toBe(false);
                // @ts-ignore Testing runtime scenario
                expect(await AuthTokensService.canRefreshToken("string")).toBe(false);
                // @ts-ignore Testing runtime scenario
                expect(await AuthTokensService.canRefreshToken(123)).toBe(false);
            }));

            it("user data not present", withDbTransaction(async () => {
                const payload: SessionData = {};
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            }));

            it("user array empty", withDbTransaction(async () => {
                const payload: SessionData = { users: [] };
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            }));

            it("missing session data", withDbTransaction(async () => {
                const payload: SessionData = {
                    users: [
                        {
                            id: generatePK().toString(),
                        },
                    ] as SessionUser[],
                };
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            }));

            it("missing lastRefreshAt", withDbTransaction(async () => {
                const payload: SessionData = {
                    users: [
                        {
                            id: generatePK().toString(),
                            session: {
                                id: generatePK().toString(),
                            },
                        },
                    ] as SessionUser[],
                };
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            }));

            it("session not in database", withDbTransaction(async () => {
                const nonExistingSessionId = generatePK();
                const payload: SessionData = {
                    users: [
                        {
                            id: generatePK().toString(),
                            session: {
                                id: nonExistingSessionId.toString(),
                                lastRefreshAt,
                            },
                        },
                    ] as SessionUser[],
                };
                // We deliberately don't create a session for this test
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            }));

            it("session is revoked", withDbTransaction(async () => {
                const testUser = await DbProvider.get().user.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePK().toString(),
                        name: "Test User",
                        theme: "light",
                    },
                });
                const testAuth = await DbProvider.get().user_auth.create({
                    data: {
                        id: generatePK(),
                        user_id: testUser.id,
                        provider: "local",
                    },
                });
                
                const revokedSessionId = generatePK();
                const payload: SessionData = {
                    users: [
                        {
                            id: generatePK().toString(),
                            session: {
                                id: revokedSessionId.toString(),
                                lastRefreshAt,
                            },
                        },
                    ] as SessionUser[],
                };
                await DbProvider.get().session.create({
                    data: {
                        id: revokedSessionId,
                        expires_at: expiresAt,
                        last_refresh_at: lastRefreshAt,
                        revokedAt: new Date(), // Session is revoked
                        user_id: testUser.id,
                        auth_id: testAuth.id,
                    },
                });
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            }));

            it("session is expired", withDbTransaction(async () => {
                const testUser = await DbProvider.get().user.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePK().toString(),
                        name: "Test User",
                        theme: "light",
                    },
                });
                const testAuth = await DbProvider.get().user_auth.create({
                    data: {
                        id: generatePK(),
                        user_id: testUser.id,
                        provider: "local",
                    },
                });
                
                const expiredSessionId = generatePK();
                const payload: SessionData = {
                    users: [
                        {
                            id: generatePK().toString(),
                            session: {
                                id: expiredSessionId.toString(),
                                lastRefreshAt,
                            },
                        },
                    ] as SessionUser[],
                };
                await DbProvider.get().session.create({
                    data: {
                        id: expiredSessionId,
                        expires_at: new Date(Date.now() - 1_000),
                        last_refresh_at: lastRefreshAt,
                        revokedAt: null, // Not revoked
                        user_id: testUser.id,
                        auth_id: testAuth.id,
                    },
                });
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            }));

            it("stored last_refresh_at does not match lastRefreshAt in payload", withDbTransaction(async () => {
                const testUser = await DbProvider.get().user.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePK().toString(),
                        name: "Test User",
                        theme: "light",
                    },
                });
                const testAuth = await DbProvider.get().user_auth.create({
                    data: {
                        id: generatePK(),
                        user_id: testUser.id,
                        provider: "local",
                    },
                });
                
                const mismatchSessionId = generatePK();
                const payload: SessionData = {
                    users: [
                        {
                            id: generatePK().toString(),
                            session: {
                                id: mismatchSessionId.toString(),
                                lastRefreshAt,
                            },
                        },
                    ] as SessionUser[],
                };
                await DbProvider.get().session.create({
                    data: {
                        id: mismatchSessionId,
                        expires_at: expiresAt,
                        last_refresh_at: new Date(Date.now() - 1_000),
                        revokedAt: null, // Not revoked
                        user_id: testUser.id,
                        auth_id: testAuth.id,
                    },
                });
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            }));
        });

        describe("true", () => {
            it("all conditions are met", withDbTransaction(async () => {
                const testUser = await DbProvider.get().user.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePK().toString(),
                        name: "Test User",
                        theme: "light",
                    },
                });
                const testAuth = await DbProvider.get().user_auth.create({
                    data: {
                        id: generatePK(),
                        user_id: testUser.id,
                        provider: "local",
                    },
                });
                
                const validSessionId = generatePK();
                const payload: SessionData = {
                    users: [
                        {
                            id: generatePK().toString(),
                            session: {
                                id: validSessionId.toString(),
                                lastRefreshAt,
                            },
                        },
                    ] as SessionUser[],
                };
                await DbProvider.get().session.create({
                    data: {
                        id: validSessionId,
                        expires_at: expiresAt,
                        last_refresh_at: lastRefreshAt,
                        revokedAt: null, // Not revoked
                        user_id: testUser.id,
                        auth_id: testAuth.id,
                    },
                });
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(true);
            }));
        });

        it("token has expired completely (beyond refresh window)", withDbTransaction(async function testExpiredToken() {
            // JWT setup is already done in beforeEach
            
            // Create a token with a very old expiration
            const oldTime = Date.now() - DAYS_30_MS * 2;
            const createExpirationTimeStub = vi.spyOn(JsonWebToken, "createExpirationTime" as any).mockReturnValue(Math.floor(oldTime / SECONDS_1_MS));

            const payload: SessionToken = {
                ...JsonWebToken.get().basicToken(),
                ...JsonWebToken.createAccessExpiresAt(),
                isLoggedIn: false,
                timeZone: "UTC",
                users: [],
            };

            createExpirationTimeStub.mockRestore();

            // Force an expired token
            const token = JsonWebToken.get().sign(payload);
            
            // Move time forward to simulate token expiration beyond refresh window
            const dateNowStub = vi.spyOn(Date, "now").mockReturnValue(Date.now() + DAYS_30_MS * 2);
            
            await expect(AuthTokensService.authenticateToken(token)).rejects.toThrow();
            
            dateNowStub.mockRestore();
        }));
    });

    describe("isTokenCorrectType", () => {
        it("returns false if token is null, undefined, or not a string", withDbTransaction(async () => {
            expect(AuthTokensService.isTokenCorrectType(null)).toBe(false);
            expect(AuthTokensService.isTokenCorrectType(undefined)).toBe(false);
            expect(AuthTokensService.isTokenCorrectType(123)).toBe(false);
        }));

        it("returns true if token is a string", withDbTransaction(async () => {
            expect(AuthTokensService.isTokenCorrectType("string")).toBe(true);
        }));
    });

    describe("isAccessTokenExpired", () => {
        describe("true", () => {
            it("string", withDbTransaction(async () => {
                const payload = { accessExpiresAt: "string" };
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).toBe(true);
            }));

            it("undefined", withDbTransaction(async () => {
                const payload = {};
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).toBe(true);
            }));

            it("null", withDbTransaction(async () => {
                const payload = { accessExpiresAt: null };
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).toBe(true);
            }));

            it("in the past", withDbTransaction(async () => {
                const payload = JsonWebToken.createAccessExpiresAt();
                payload.accessExpiresAt = payload.accessExpiresAt - SECONDS_1_MS;
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).toBe(true);
            }));

            it("negative number", withDbTransaction(async () => {
                // Picked a date that would be in the future if it was positive
                const payload = JsonWebToken.createAccessExpiresAt();
                payload.accessExpiresAt = -(payload.accessExpiresAt + SECONDS_1_MS);
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).toBe(true);
            }));
        });

        describe("false", () => {
            it("in the future", withDbTransaction(async () => {
                const payload = JsonWebToken.createAccessExpiresAt();
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).toBe(false);
            }));
        });
    });

    describe("authenticateToken", () => {
        describe("throws", () => {
            it("token is null, undefined, or not a string", withDbTransaction(async () => {
                
                await expect(AuthTokensService.authenticateToken(null as any)).rejects.toThrow("NoSessionToken");
                await expect(AuthTokensService.authenticateToken(undefined as any)).rejects.toThrow("NoSessionToken");
                await expect(AuthTokensService.authenticateToken(123 as any)).rejects.toThrow("NoSessionToken");
            }));

            it("token has invalid signature", withDbTransaction(async () => {
                
                const invalidToken = "invalid.token.here";
                await expect(AuthTokensService.authenticateToken(invalidToken)).rejects.toThrow();
            }));

            it("token is tampered with", withDbTransaction(async () => {
                
                // Create a valid token
                const payload: SessionToken = {
                    ...JsonWebToken.get().basicToken(),
                    ...JsonWebToken.createAccessExpiresAt(),
                    isLoggedIn: false,
                    timeZone: "UTC",
                    users: [],
                };
                const token = JsonWebToken.get().sign(payload);

                // Tamper with the middle part (payload) of the token
                const [header, , signature] = token.split(".");
                const tamperedPayload = Buffer.from("{\"malicious\":\"data\"}").toString("base64").replace(/=/g, "");
                const tamperedToken = `${header}.${tamperedPayload}.${signature}`;

                await expect(AuthTokensService.authenticateToken(tamperedToken)).rejects.toThrow();
            }));

            it("token has expired completely (beyond refresh window) - second test", withDbTransaction(async function testExpiredToken() {
                // Create a token with a very old expiration
                const oldTime = Date.now() - DAYS_30_MS * 2;
                const expStub = vi.spyOn(JsonWebToken, "createExpirationTime" as any).mockReturnValue(Math.floor(oldTime / SECONDS_1_MS));

                const payload: SessionToken = {
                    ...JsonWebToken.get().basicToken(),
                    ...JsonWebToken.createAccessExpiresAt(),
                    isLoggedIn: false,
                    timeZone: "UTC",
                    users: [],
                };

                const token = JsonWebToken.get().sign(payload);
                
                expStub.mockRestore();

                // Move time forward to simulate token expiration
                const dateNowStub = vi.spyOn(Date, "now").mockReturnValue(Date.now() + DAYS_30_MS * 2);
                
                await expect(AuthTokensService.authenticateToken(token)).rejects.toThrow();
                
                dateNowStub.mockRestore();
            }));
        });

        it("returns existing token and payload if token is valid and unexpired", withDbTransaction(async function testValidUnexpiredToken() {
            
            const payload: SessionToken = {
                ...JsonWebToken.get().basicToken(),
                ...JsonWebToken.createAccessExpiresAt(),
                isLoggedIn: false,
                timeZone: "UTC",
                users: [],
            };
            const token = JsonWebToken.get().sign(payload);

            const result = await AuthTokensService.authenticateToken(token);

            expect(result.token).toBe(token);
            expect(result.payload).toMatchObject(payload);
            expect(Math.abs(result.maxAge - JsonWebToken.getMaxAge(result.payload))).toBeLessThanOrEqual(SECONDS_1_MS);
        }));

        it("handles token with complex user data structure correctly", withDbTransaction(async function testComplexUserData() {
            
            const testUser = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePK().toString(),
                    name: "Test User",
                    theme: "light",
                },
            });
            const testAuth = await DbProvider.get().user_auth.create({
                data: {
                    id: generatePK(),
                    user_id: testUser.id,
                    provider: "local",
                },
            });
            
            const sessionId = generatePK().toString();
            const lastRefreshAt = new Date();
            const userId = generatePK().toString();

            // Create session for this test
            await DbProvider.get().session.create({
                data: {
                    id: sessionId,
                    expires_at: new Date(Date.now() + DAYS_30_MS),
                    last_refresh_at: lastRefreshAt,
                    revokedAt: null, // Not revoked
                    user_id: testUser.id,
                    auth_id: testAuth.id,
                },
            });

            const sessionUserData = {
                id: userId,
                name: "Test User",
                theme: "dark",
                session: {
                    id: sessionId,
                    lastRefreshAt: lastRefreshAt.toISOString(),
                },
                role: "user",
            } as unknown as SessionUser;

            const payload: SessionToken = {
                ...JsonWebToken.get().basicToken(),
                ...JsonWebToken.createAccessExpiresAt(),
                isLoggedIn: true,
                timeZone: "America/New_York",
                users: [sessionUserData],
            };

            const token = JsonWebToken.get().sign(payload);
            const result = await AuthTokensService.authenticateToken(token);

            expect(result.token).toBe(token);
            expect(result.payload).toMatchObject(payload);
            expect(result.payload.users[0].id).toBe(userId);
            expect(result.payload.users[0].session?.id).toBe(sessionId);
        }));

        it("applies additionalData correctly when provided", withDbTransaction(async function testAdditionalData() {
            
            const payload: SessionToken = {
                ...JsonWebToken.get().basicToken(),
                ...JsonWebToken.createAccessExpiresAt(),
                isLoggedIn: false,
                timeZone: "UTC",
                users: [],
            };
            const token = JsonWebToken.get().sign(payload);

            const additionalData = {
                extraField: "extraValue",
                nestedData: {
                    key1: "value1",
                    key2: 123,
                },
            };

            const result = await AuthTokensService.authenticateToken(token, { additionalData });

            expect(result.token).not.toBe(token);
            expect(result.payload).toMatchObject({
                ...payload,
                ...additionalData,
            } as any);
            expect((result.payload as any).extraField).toBe("extraValue");
            expect((result.payload as any).nestedData.key1).toBe("value1");
            expect((result.payload as any).nestedData.key2).toBe(123);
        }));

        it("uses modifyPayload callback correctly when provided", withDbTransaction(async function testModifyPayload() {
            
            const payload: SessionToken = {
                ...JsonWebToken.get().basicToken(),
                ...JsonWebToken.createAccessExpiresAt(),
                isLoggedIn: false,
                timeZone: "UTC",
                users: [],
            };
            const token = JsonWebToken.get().sign(payload);

            // Define the modifyPayload function with proper declaration
            function modifyPayload(p: SessionToken): SessionToken {
                return {
                    ...p,
                    modifiedField: "modified",
                    timeZone: "Europe/London", // Override existing field
                } as SessionToken; // Type assertion to avoid TypeScript errors
            }

            const result = await AuthTokensService.authenticateToken(token, { modifyPayload });

            expect(result.token).not.toBe(token);
            expect((result.payload as any).modifiedField).toBe("modified");
            expect(result.payload.timeZone).toBe("Europe/London");
        }));

        it("refreshes the token if access token expired and can be refreshed", withDbTransaction(async function testTokenRefresh() {
            
            const testUser = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePK().toString(),
                    name: "Test User",
                    theme: "light",
                },
            });
            const testAuth = await DbProvider.get().user_auth.create({
                data: {
                    id: generatePK(),
                    user_id: testUser.id,
                    provider: "local",
                },
            });
            
            const sessionId = generatePK().toString();
            const lastRefreshAt = new Date();

            const payload: SessionToken = {
                ...JsonWebToken.get().basicToken(),
                accessExpiresAt: 0, // Expired
                isLoggedIn: false,
                timeZone: "UTC",
                users: [
                    {
                        id: generatePK().toString(),
                        session: {
                            id: sessionId,
                            lastRefreshAt: lastRefreshAt.toISOString(),
                        },
                    },
                ] as SessionUser[],
            };

            const token = JsonWebToken.get().sign(payload);

            // Inject session data
            await DbProvider.get().session.create({
                data: {
                    id: sessionId,
                    expires_at: new Date(Date.now() + DAYS_30_MS), // Not expired
                    last_refresh_at: lastRefreshAt, // Matches payload
                    revokedAt: null, // Not revoked
                    user_id: testUser.id,
                    auth_id: testAuth.id,
                },
            });

            const result = await AuthTokensService.authenticateToken(token);

            expect(result.token).not.toBe(token);
            expect(result.payload).toMatchObject({
                ...payload,
                accessExpiresAt: result.payload.accessExpiresAt, // Replace with actual value
            });
            expect(typeof result.payload.exp).toBe("number");
            expect(typeof result.payload.accessExpiresAt).toBe("number");
            // Allow for small differences in timestamp values by checking they're close rather than exactly equal
            expect(result.payload.exp).toBeGreaterThan(new Date().getTime() / SECONDS_1_MS);
            expect(result.payload.accessExpiresAt).toBeGreaterThan(new Date().getTime() / SECONDS_1_MS); // Check new access expiration time is in the future
            expect(result.maxAge).toBeGreaterThan(0);

            // Move time forward to simulate token expiration, and check if it can be refreshed again
            const dateNowStub = vi.spyOn(Date, "now").mockReturnValue(new Date().getTime() + 2 * ACCESS_TOKEN_EXPIRATION_MS);
            const result2 = await AuthTokensService.authenticateToken(result.token);

            expect(result2.token).not.toBe(result.token);
            expect(result2.payload).toMatchObject({
                ...result.payload,
                accessExpiresAt: result2.payload.accessExpiresAt, // Replace with actual value
                exp: result2.payload.exp, // Don't try to compare exact exp values
            });
            expect(typeof result2.payload.exp).toBe("number");
            expect(typeof result2.payload.accessExpiresAt).toBe("number");
            expect(result2.payload.exp).toBeGreaterThan(new Date().getTime() / SECONDS_1_MS);
            expect(result2.payload.accessExpiresAt).toBeGreaterThan(new Date().getTime() / SECONDS_1_MS); // Check new access expiration time is in the future
            expect(result2.maxAge).toBeGreaterThan(0);
            
            dateNowStub.mockRestore();
        }));

        it("throws 'SessionExpired' error if token is expired and cannot be refreshed", withDbTransaction(async function testSessionExpired() {
            
            const testUser = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePK().toString(),
                    name: "Test User",
                    theme: "light",
                },
            });
            const testAuth = await DbProvider.get().user_auth.create({
                data: {
                    id: generatePK(),
                    user_id: testUser.id,
                    provider: "local",
                },
            });
            
            const initialTime = 1000000000000;
            const dateNowStub = vi.spyOn(Date, "now").mockReturnValue(initialTime);

            const sessionId = generatePK().toString();
            const lastRefreshAt = new Date(initialTime - DAYS_30_MS).toISOString();

            const payload: SessionToken = {
                ...JsonWebToken.get().basicToken(),
                ...JsonWebToken.createAccessExpiresAt(),
                isLoggedIn: false,
                timeZone: "UTC",
                users: [
                    {
                        id: generatePK().toString(),
                        session: {
                            id: sessionId,
                            lastRefreshAt,
                        },
                    },
                ] as SessionUser[],
            };

            const token = JsonWebToken.get().sign(payload);

            // Advance time to simulate token expiration
            const expiredTime = initialTime + (5 * ACCESS_TOKEN_EXPIRATION_MS);
            dateNowStub.mockRestore(); // Restore original Date.now to avoid conflicts
            vi.spyOn(Date, "now").mockReturnValue(expiredTime);

            // Inject session data
            await DbProvider.get().session.create({
                data: {
                    id: sessionId,
                    expires_at: new Date(Date.now() + DAYS_30_MS), // Add future expiration date
                    last_refresh_at: new Date(lastRefreshAt),
                    revokedAt: new Date(), // Can't refresh a revoked session
                    user_id: testUser.id,
                    auth_id: testAuth.id,
                },
            });

            await expect(AuthTokensService.authenticateToken(token)).rejects.toThrow("SessionExpired");
        }));
    });

    describe("generateApiToken", () => {


        it("should generate a JWT with API token data and add it to the response cookies", withDbTransaction(async () => {

            // Mock Response object
            const res = {
                headersSent: false,
                cookie: vi.fn(),
            } as unknown as Response;
            
            const apiToken = "test-api-token";
            const permissions = {
                [ApiKeyPermission.ReadPublic]: true,
                [ApiKeyPermission.ReadPrivate]: true,
                [ApiKeyPermission.WritePrivate]: false,
                [ApiKeyPermission.ReadAuth]: false,
                [ApiKeyPermission.WriteAuth]: false,
            };
            const userId = generatePK().toString();

            await AuthTokensService.generateApiToken(res, apiToken, permissions, userId);

            expect((res.cookie as any).mock.calls.length).toBe(1);

            const [cookieName, token, options] = (res.cookie as any).mock.calls[0];

            expect(cookieName).toBe(COOKIE.Jwt);
            expect(typeof token).toBe("string");

            // Check individual properties instead of deep equality for options
            expect(options.httpOnly).toBe(JsonWebToken.getJwtCookieOptions().httpOnly);
            expect(options.sameSite).toBe(JsonWebToken.getJwtCookieOptions().sameSite);
            expect(options.secure).toBe(JsonWebToken.getJwtCookieOptions().secure);
            expect(typeof options.maxAge).toBe("number");

            const payload = await JsonWebToken.get().verify(token);
            expect(payload.apiToken).toBe(apiToken);
            expect(payload.permissions).toEqual(permissions);
            expect(payload.userId).toBe(userId);
        }));

        it("should not set cookie if headers are already sent", withDbTransaction(async () => {

            // Mock Response object
            const res = {
                headersSent: true,
                cookie: vi.fn(),
            } as unknown as Response;
            
            const apiToken = "test-api-token";
            const permissions = {
                [ApiKeyPermission.ReadPublic]: true,
                [ApiKeyPermission.ReadPrivate]: false,
                [ApiKeyPermission.WritePrivate]: false,
                [ApiKeyPermission.ReadAuth]: false,
                [ApiKeyPermission.WriteAuth]: false,
            };
            const userId = generatePK().toString();

            await AuthTokensService.generateApiToken(res, apiToken, permissions, userId);

            expect((res.cookie as any).mock.calls.length).toBe(0);
        }));
    });

    describe("generateSessionToken", () => {


        it("should generate a JWT with session data and add it to the response cookies", withDbTransaction(async () => {

            // Mock Response object
            const res = {
                headersSent: false,
                cookie: vi.fn(),
            } as unknown as Response;
            
            const session = {
                isLoggedIn: true,
                timeZone: "UTC",
                users: [
                    {
                        id: "user1",
                        session: {
                            id: "session1",
                            lastRefreshAt: new Date().toISOString(),
                        },
                    } as unknown as SessionUser,
                    {
                        id: "user2",
                        session: {
                            id: "session2",
                            lastRefreshAt: new Date().toISOString(),
                        },
                    } as unknown as SessionUser,
                ],
            };

            await AuthTokensService.generateSessionToken(res, session);

            expect((res.cookie as any).mock.calls.length).toBe(1);

            const [cookieName, token, options] = (res.cookie as any).mock.calls[0];

            expect(cookieName).toBe(COOKIE.Jwt);
            expect(typeof token).toBe("string");

            // Check individual properties instead of deep equality for options
            expect(options.httpOnly).toBe(JsonWebToken.getJwtCookieOptions().httpOnly);
            expect(options.sameSite).toBe(JsonWebToken.getJwtCookieOptions().sameSite);
            expect(options.secure).toBe(JsonWebToken.getJwtCookieOptions().secure);
            expect(typeof options.maxAge).toBe("number");

            const payload = await JsonWebToken.get().verify(token);

            expect(payload.isLoggedIn).toBe(true);
            expect(payload.timeZone).toBe("UTC");
            expect(payload.users).toEqual(session.users);
        }));

        it("should not set cookie if headers are already sent", withDbTransaction(async () => {

            // Mock Response object
            const res = {
                headersSent: true,
                cookie: vi.fn(),
            } as unknown as Response;
            
            const session = {};

            await AuthTokensService.generateSessionToken(res, session);

            expect((res.cookie as any).mock.calls.length).toBe(0);
        }));

        it("should handle partial session data correctly", withDbTransaction(async () => {

            // Mock Response object
            const res = {
                headersSent: false,
                cookie: vi.fn(),
            } as unknown as Response;
            
            const session = {
                isLoggedIn: false,
                timeZone: "America/New_York",
            };

            await AuthTokensService.generateSessionToken(res, session);

            expect((res.cookie as any).mock.calls.length).toBe(1);

            const [, token] = (res.cookie as any).mock.calls[0];
            const payload = await JsonWebToken.get().verify(token);

            expect(payload.isLoggedIn).toBe(false);
            expect(payload.timeZone).toBe("America/New_York");
            expect(payload.users).toEqual([]);
        }));
    });
});

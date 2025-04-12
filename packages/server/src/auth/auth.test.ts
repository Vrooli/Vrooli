/* eslint-disable @typescript-eslint/ban-ts-comment */
import { ApiKeyPermission, COOKIE, DAYS_1_MS, DAYS_30_MS, SECONDS_1_MS, SessionUser, uuid } from "@local/shared";
import { expect } from "chai";
import { generateKeyPairSync } from "crypto";
import { Response } from "express";
import jwt from "jsonwebtoken";
import sinon from "sinon";
import { DbProvider } from "../db/provider.js";
import { logger } from "../events/logger.js";
import { type SessionData, type SessionToken } from "../types.js";
import { AuthTokensService } from "./auth.js";
import { ACCESS_TOKEN_EXPIRATION_MS, JsonWebToken } from "./jwt.js";

// Define test user and auth variables for use in tests
let testUser: { id: string };
let testAuth: { id: string; user_id: string };

describe("AuthTokensService", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async function before() {
        this.timeout(10_000);

        await DbProvider.get().user.deleteMany({});
        await DbProvider.get().user_auth.deleteMany({});
        await DbProvider.get().session.deleteMany({});
        // Create test user and auth for all tests
        testUser = await DbProvider.get().user.create({
            data: {
                id: uuid(),
                name: "Test User",
                theme: "light",
            },
        });
        testAuth = await DbProvider.get().user_auth.create({
            data: {
                id: uuid(),
                user_id: testUser.id,
                provider: "local",
            },
        });
    });

    after(async function after() {
        this.timeout(10_000);

        await DbProvider.get().user.deleteMany({});
        await DbProvider.get().user_auth.deleteMany({});
        await DbProvider.get().session.deleteMany({});

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("Singleton Behavior", () => {
        it("get() should return the same instance", () => {
            const instance1 = AuthTokensService.get();
            const instance2 = AuthTokensService.get();
            expect(instance1).to.equal(instance2);
        });
    });

    describe("canRefreshToken", () => {
        // Use different session IDs for each test to avoid unique constraint conflicts
        const expiresAt = new Date(Date.now() + DAYS_30_MS);
        const lastRefreshAt = new Date(Date.now() - DAYS_1_MS);
        let privateKey: string;
        let publicKey: string;

        beforeEach(async function setupAuthTests() {
            this.timeout(10_000);

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
            privateKey = keys.privateKey;
            publicKey = keys.publicKey;
            JsonWebToken.get().setupTestEnvironment({ privateKey, publicKey });
        });

        describe("false", () => {
            it("not an object", async () => {
                // @ts-ignore Testing runtime scenario
                expect(await AuthTokensService.canRefreshToken(null)).to.equal(false);
                // @ts-ignore Testing runtime scenario
                expect(await AuthTokensService.canRefreshToken(undefined)).to.equal(false);
                // @ts-ignore Testing runtime scenario
                expect(await AuthTokensService.canRefreshToken("string")).to.equal(false);
                // @ts-ignore Testing runtime scenario
                expect(await AuthTokensService.canRefreshToken(123)).to.equal(false);
            });

            it("user data not present", async () => {
                const payload: SessionData = {};
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).to.equal(false);
            });

            it("user array empty", async () => {
                const payload: SessionData = { users: [] };
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).to.equal(false);
            });

            it("missing session data", async () => {
                const payload: SessionData = {
                    users: [
                        {
                            id: uuid(),
                        },
                    ] as SessionUser[],
                };
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).to.equal(false);
            });

            it("missing lastRefreshAt", async () => {
                const payload: SessionData = {
                    users: [
                        {
                            id: uuid(),
                            session: {
                                id: uuid(),
                            },
                        },
                    ] as SessionUser[],
                };
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).to.equal(false);
            });

            it("session not in database", async () => {
                const nonExistingSessionId = uuid(); // Generate a different session ID
                const payload: SessionData = {
                    users: [
                        {
                            id: uuid(),
                            session: {
                                id: nonExistingSessionId,
                                lastRefreshAt,
                            },
                        },
                    ] as SessionUser[],
                };
                // We deliberately don't create a session for this test
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).to.equal(false);
            });

            it("session is revoked", async () => {
                const revokedSessionId = uuid(); // Use uuid() instead of baseSessionId + "-revoked"
                const payload: SessionData = {
                    users: [
                        {
                            id: uuid(),
                            session: {
                                id: revokedSessionId,
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
                        revoked: true,
                        user_id: testUser.id,
                        auth_id: testAuth.id,
                    },
                });
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).to.equal(false);
            });

            it("session is expired", async () => {
                const expiredSessionId = uuid(); // Use uuid() instead of baseSessionId + "-expired"
                const payload: SessionData = {
                    users: [
                        {
                            id: uuid(),
                            session: {
                                id: expiredSessionId,
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
                        revoked: false,
                        user_id: testUser.id,
                        auth_id: testAuth.id,
                    },
                });
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).to.equal(false);
            });

            it("stored last_refresh_at does not match lastRefreshAt in payload", async () => {
                const mismatchSessionId = uuid(); // Use uuid() instead of baseSessionId + "-mismatch"
                const payload: SessionData = {
                    users: [
                        {
                            id: uuid(),
                            session: {
                                id: mismatchSessionId,
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
                        revoked: false,
                        user_id: testUser.id,
                        auth_id: testAuth.id,
                    },
                });
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).to.equal(false);
            });
        });

        describe("true", () => {
            it("all conditions are met", async () => {
                const validSessionId = uuid(); // Use uuid() instead of baseSessionId + "-valid"
                const payload: SessionData = {
                    users: [
                        {
                            id: uuid(),
                            session: {
                                id: validSessionId,
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
                        revoked: false,
                        user_id: testUser.id,
                        auth_id: testAuth.id,
                    },
                });
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).to.equal(true);
            });
        });

        it("token has expired completely (beyond refresh window)", async function testExpiredToken() {
            this.timeout(10000);
            // Create a token with a very old expiration
            const oldTime = Date.now() - DAYS_30_MS * 2;
            const expStub = sinon.stub(JsonWebToken, "createExpirationTime").returns(Math.floor(oldTime / SECONDS_1_MS));

            const payload: SessionToken = {
                ...JsonWebToken.get().basicToken(),
                ...JsonWebToken.createAccessExpiresAt(),
                isLoggedIn: false,
                timeZone: "UTC",
                users: [],
            };

            expStub.restore();

            // Force an expired token by manipulating the exp claim
            const token = JsonWebToken.get().sign(payload);
            const decoded = JsonWebToken.get().decode(token);
            if (decoded) {
                decoded.exp = Math.floor((Date.now() - DAYS_30_MS) / SECONDS_1_MS); // Expired 30 days ago

                // Manually create a token with the expired claim
                const tamperedToken = jwt.sign(decoded, privateKey, { algorithm: "RS256" });

                await expect(AuthTokensService.authenticateToken(tamperedToken)).to.be.rejectedWith();
            }
        });
    });

    describe("isTokenCorrectType", () => {
        it("returns false if token is null, undefined, or not a string", () => {
            expect(AuthTokensService.isTokenCorrectType(null)).to.equal(false);
            expect(AuthTokensService.isTokenCorrectType(undefined)).to.equal(false);
            expect(AuthTokensService.isTokenCorrectType(123)).to.equal(false);
        });

        it("returns true if token is a string", () => {
            expect(AuthTokensService.isTokenCorrectType("string")).to.equal(true);
        });
    });

    describe("isAccessTokenExpired", () => {
        describe("true", () => {
            it("string", () => {
                const payload = { accessExpiresAt: "string" };
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).to.equal(true);
            });

            it("undefined", () => {
                const payload = {};
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).to.equal(true);
            });

            it("null", () => {
                const payload = { accessExpiresAt: null };
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).to.equal(true);
            });

            it("in the past", () => {
                const payload = JsonWebToken.createAccessExpiresAt();
                payload.accessExpiresAt = payload.accessExpiresAt - SECONDS_1_MS;
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).to.equal(true);
            });

            it("negative number", () => {
                // Picked a date that would be in the future if it was positive
                const payload = JsonWebToken.createAccessExpiresAt();
                payload.accessExpiresAt = -(payload.accessExpiresAt + SECONDS_1_MS);
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).to.equal(true);
            });
        });

        describe("false", () => {
            it("in the future", () => {
                const payload = JsonWebToken.createAccessExpiresAt();
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).to.equal(false);
            });
        });
    });

    describe("authenticateToken", () => {
        let instance: JsonWebToken;
        let privateKey: string;
        let publicKey: string;

        beforeEach(function setupAuthTests() {
            this.timeout(10000);

            instance = JsonWebToken.get();
            const { privateKey: privKey, publicKey: pubKey } = generateKeyPairSync("rsa", {
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
            privateKey = privKey;
            publicKey = pubKey;
            instance.setupTestEnvironment({ privateKey, publicKey });
        });

        afterEach(async function cleanupAuthTests() {
            sinon.restore();
        });

        describe("throws", () => {
            it("token is null, undefined, or not a string", async () => {
                await expect(AuthTokensService.authenticateToken(null as any)).to.be.rejectedWith("NoSessionToken");
                await expect(AuthTokensService.authenticateToken(undefined as any)).to.be.rejectedWith("NoSessionToken");
                await expect(AuthTokensService.authenticateToken(123 as any)).to.be.rejectedWith("NoSessionToken");
            });

            it("token has invalid signature", async () => {
                const invalidToken = "invalid.token.here";
                await expect(AuthTokensService.authenticateToken(invalidToken)).to.be.rejectedWith();
            });

            it("token is tampered with", async () => {
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

                await expect(AuthTokensService.authenticateToken(tamperedToken)).to.be.rejectedWith();
            });

            it("token has expired completely (beyond refresh window)", async function testExpiredToken() {
                this.timeout(10000);
                // Create a token with a very old expiration
                const oldTime = Date.now() - DAYS_30_MS * 2;
                const expStub = sinon.stub(JsonWebToken, "createExpirationTime").returns(Math.floor(oldTime / SECONDS_1_MS));

                const payload: SessionToken = {
                    ...JsonWebToken.get().basicToken(),
                    ...JsonWebToken.createAccessExpiresAt(),
                    isLoggedIn: false,
                    timeZone: "UTC",
                    users: [],
                };

                expStub.restore();

                // Force an expired token by manipulating the exp claim
                const token = JsonWebToken.get().sign(payload);
                const decoded = JsonWebToken.get().decode(token);
                if (decoded) {
                    decoded.exp = Math.floor((Date.now() - DAYS_30_MS) / SECONDS_1_MS); // Expired 30 days ago

                    // Manually create a token with the expired claim
                    const tamperedToken = jwt.sign(decoded, privateKey, { algorithm: "RS256" });

                    await expect(AuthTokensService.authenticateToken(tamperedToken)).to.be.rejectedWith();
                }
            });
        });

        it("returns existing token and payload if token is valid and unexpired", async function testValidUnexpiredToken() {
            this.timeout(10000);

            const payload: SessionToken = {
                ...JsonWebToken.get().basicToken(),
                ...JsonWebToken.createAccessExpiresAt(),
                isLoggedIn: false,
                timeZone: "UTC",
                users: [],
            };
            const token = JsonWebToken.get().sign(payload);

            const result = await AuthTokensService.authenticateToken(token);

            expect(result.token).to.equal(token);
            expect(result.payload).to.deep.include(payload);
            expect(Math.abs(result.maxAge - JsonWebToken.getMaxAge(result.payload))).to.be.at.most(SECONDS_1_MS);
        });

        it("handles token with complex user data structure correctly", async function testComplexUserData() {
            this.timeout(10000);

            const sessionId = uuid();
            const lastRefreshAt = new Date();
            const userId = uuid();

            // Create session for this test
            await DbProvider.get().session.create({
                data: {
                    id: sessionId,
                    expires_at: new Date(Date.now() + DAYS_30_MS),
                    last_refresh_at: lastRefreshAt,
                    revoked: false,
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

            expect(result.token).to.equal(token);
            expect(result.payload).to.deep.include(payload);
            expect(result.payload.users[0].id).to.equal(userId);
            expect(result.payload.users[0].session?.id).to.equal(sessionId);
        });

        it("applies additionalData correctly when provided", async function testAdditionalData() {
            this.timeout(10000);

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

            expect(result.token).not.to.equal(token);
            expect(result.payload).to.deep.include({
                ...payload,
                ...additionalData,
            } as any);
            expect((result.payload as any).extraField).to.equal("extraValue");
            expect((result.payload as any).nestedData.key1).to.equal("value1");
            expect((result.payload as any).nestedData.key2).to.equal(123);
        });

        it("uses modifyPayload callback correctly when provided", async function testModifyPayload() {
            this.timeout(10000);

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

            expect(result.token).not.to.equal(token);
            expect((result.payload as any).modifiedField).to.equal("modified");
            expect(result.payload.timeZone).to.equal("Europe/London");
        });

        it("refreshes the token if access token expired and can be refreshed", async function testTokenRefresh() {
            this.timeout(10000); // Set higher timeout for this specific test

            const sessionId = uuid();
            const lastRefreshAt = new Date();

            const payload: SessionToken = {
                ...JsonWebToken.get().basicToken(),
                accessExpiresAt: 0, // Expired
                isLoggedIn: false,
                timeZone: "UTC",
                users: [
                    {
                        id: uuid(),
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
                    revoked: false,
                    user_id: testUser.id,
                    auth_id: testAuth.id,
                },
            });

            const result = await AuthTokensService.authenticateToken(token);

            expect(result.token).not.to.equal(token);
            expect(result.payload).to.deep.include({
                ...payload,
                accessExpiresAt: result.payload.accessExpiresAt, // Replace with actual value
            });
            expect(typeof result.payload.exp).to.equal("number");
            expect(typeof result.payload.accessExpiresAt).to.equal("number");
            // Allow for small differences in timestamp values by checking they're close rather than exactly equal
            expect(result.payload.exp).to.be.greaterThan(new Date().getTime() / SECONDS_1_MS);
            expect(result.payload.accessExpiresAt).to.be.greaterThan(new Date().getTime() / SECONDS_1_MS); // Check new access expiration time is in the future
            expect(result.maxAge).to.be.greaterThan(0);

            // Move time forward to simulate token expiration, and check if it can be refreshed again
            sinon.stub(Date, "now").returns(new Date().getTime() + 2 * ACCESS_TOKEN_EXPIRATION_MS);
            const result2 = await AuthTokensService.authenticateToken(result.token);

            expect(result2.token).not.to.equal(result.token);
            expect(result2.payload).to.deep.include({
                ...result.payload,
                accessExpiresAt: result2.payload.accessExpiresAt, // Replace with actual value
                exp: result2.payload.exp, // Don't try to compare exact exp values
            });
            expect(typeof result2.payload.exp).to.equal("number");
            expect(typeof result2.payload.accessExpiresAt).to.equal("number");
            expect(result2.payload.exp).to.be.greaterThan(new Date().getTime() / SECONDS_1_MS);
            expect(result2.payload.accessExpiresAt).to.be.greaterThan(new Date().getTime() / SECONDS_1_MS); // Check new access expiration time is in the future
            expect(result2.maxAge).to.be.greaterThan(0);
        });

        it("throws 'SessionExpired' error if token is expired and cannot be refreshed", async function testSessionExpired() {
            const initialTime = 1000000000000;
            const dateNowStub = sinon.stub(Date, "now").returns(initialTime);

            const sessionId = uuid();
            const lastRefreshAt = new Date(initialTime - DAYS_30_MS).toISOString();

            const payload: SessionToken = {
                ...JsonWebToken.get().basicToken(),
                ...JsonWebToken.createAccessExpiresAt(),
                isLoggedIn: false,
                timeZone: "UTC",
                users: [
                    {
                        id: uuid(),
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
            dateNowStub.restore(); // Restore original Date.now to avoid conflicts
            sinon.stub(Date, "now").returns(expiredTime);

            // Inject session data
            await DbProvider.get().session.create({
                data: {
                    id: sessionId,
                    expires_at: new Date(Date.now() + DAYS_30_MS), // Add future expiration date
                    last_refresh_at: new Date(lastRefreshAt),
                    revoked: true, // Can't refresh a revoked session
                    user_id: testUser.id,
                    auth_id: testAuth.id,
                },
            });

            await expect(AuthTokensService.authenticateToken(token)).to.be.rejectedWith("SessionExpired");
        });
    });

    describe("generateApiToken", () => {
        let instance: JsonWebToken;
        let privateKey: string;
        let publicKey: string;
        let res: Response;

        beforeEach(function setupApiTokenTests() {
            this.timeout(10_000);

            instance = JsonWebToken.get();
            const { privateKey: privKey, publicKey: pubKey } = generateKeyPairSync("rsa", {
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
            privateKey = privKey;
            publicKey = pubKey;
            instance.setupTestEnvironment({ privateKey, publicKey });

            // Mock Response object
            res = {
                headersSent: false,
                cookie: sinon.stub(),
            } as unknown as Response;
        });

        afterEach(() => {
            sinon.restore();
        });

        it("should generate a JWT with API token data and add it to the response cookies", async () => {
            const apiToken = "test-api-token";
            const permissions = {
                [ApiKeyPermission.ReadPublic]: true,
                [ApiKeyPermission.ReadPrivate]: true,
                [ApiKeyPermission.WritePrivate]: false,
                [ApiKeyPermission.ReadAuth]: false,
                [ApiKeyPermission.WriteAuth]: false,
            };
            const userId = uuid();

            await AuthTokensService.generateApiToken(res, apiToken, permissions, userId);

            expect((res.cookie as sinon.SinonStub).called).to.be.true;

            const [cookieName, token, options] = (res.cookie as sinon.SinonStub).args[0];

            expect(cookieName).to.equal(COOKIE.Jwt);
            expect(typeof token).to.equal("string");

            // Check individual properties instead of deep equality for options
            expect(options.httpOnly).to.equal(JsonWebToken.getJwtCookieOptions().httpOnly);
            expect(options.sameSite).to.equal(JsonWebToken.getJwtCookieOptions().sameSite);
            expect(options.secure).to.equal(JsonWebToken.getJwtCookieOptions().secure);
            expect(options.maxAge).to.be.a("number");

            const payload = await JsonWebToken.get().verify(token);
            expect(payload.apiToken).to.equal(apiToken);
            expect(payload.permissions).to.deep.equal(permissions);
            expect(payload.userId).to.equal(userId);
        });

        it("should not set cookie if headers are already sent", async () => {
            res.headersSent = true;
            const apiToken = "test-api-token";
            const permissions = {
                [ApiKeyPermission.ReadPublic]: true,
                [ApiKeyPermission.ReadPrivate]: false,
                [ApiKeyPermission.WritePrivate]: false,
                [ApiKeyPermission.ReadAuth]: false,
                [ApiKeyPermission.WriteAuth]: false,
            };
            const userId = uuid();

            await AuthTokensService.generateApiToken(res, apiToken, permissions, userId);

            expect((res.cookie as sinon.SinonStub).called).to.be.false;
        });
    });

    describe("generateSessionToken", () => {
        let instance: JsonWebToken;
        let privateKey: string;
        let publicKey: string;
        let res: Response;

        beforeEach(function setupSessionTokenTests() {
            this.timeout(10_000);

            instance = JsonWebToken.get();
            const { privateKey: privKey, publicKey: pubKey } = generateKeyPairSync("rsa", {
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
            privateKey = privKey;
            publicKey = pubKey;
            instance.setupTestEnvironment({ privateKey, publicKey });

            // Mock Response object
            res = {
                headersSent: false,
                cookie: sinon.stub(),
            } as unknown as Response;
        });

        afterEach(() => {
            sinon.restore();
        });

        it("should generate a JWT with session data and add it to the response cookies", async () => {
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

            expect((res.cookie as sinon.SinonStub).called).to.be.true;

            const [cookieName, token, options] = (res.cookie as sinon.SinonStub).args[0];

            expect(cookieName).to.equal(COOKIE.Jwt);
            expect(typeof token).to.equal("string");

            // Check individual properties instead of deep equality for options
            expect(options.httpOnly).to.equal(JsonWebToken.getJwtCookieOptions().httpOnly);
            expect(options.sameSite).to.equal(JsonWebToken.getJwtCookieOptions().sameSite);
            expect(options.secure).to.equal(JsonWebToken.getJwtCookieOptions().secure);
            expect(options.maxAge).to.be.a("number");

            const payload = await JsonWebToken.get().verify(token);

            expect(payload.isLoggedIn).to.equal(true);
            expect(payload.timeZone).to.equal("UTC");
            expect(payload.users).to.deep.equal(session.users);
        });

        it("should not set cookie if headers are already sent", async () => {
            res.headersSent = true;
            const session = {};

            await AuthTokensService.generateSessionToken(res, session);

            expect((res.cookie as sinon.SinonStub).called).to.be.false;
        });

        it("should handle partial session data correctly", async () => {
            const session = {
                isLoggedIn: false,
                timeZone: "America/New_York",
            };

            await AuthTokensService.generateSessionToken(res, session);

            expect((res.cookie as sinon.SinonStub).called).to.be.true;

            const [, token] = (res.cookie as sinon.SinonStub).args[0];
            const payload = await JsonWebToken.get().verify(token);

            expect(payload.isLoggedIn).to.equal(false);
            expect(payload.timeZone).to.equal("America/New_York");
            expect(payload.users).to.be.an("array").that.is.empty;
        });
    });
});

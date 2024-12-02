import { DAYS_1_MS, DAYS_30_MS, SECONDS_1_MS, SessionUser, uuid } from "@local/shared";
import { generateKeyPairSync } from "crypto";
import pkg from "../__mocks__/@prisma/client";
import { SessionData, SessionToken } from "../types";
import { AuthTokensService } from "./auth";
import { ACCESS_TOKEN_EXPIRATION_MS, JsonWebToken } from "./jwt";

const { PrismaClient } = pkg;

describe("AuthTokensService", () => {
    beforeEach(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "error").mockImplementation(() => { });
    })

    afterEach(() => {
        PrismaClient.clearData();
        jest.restoreAllMocks();
    });

    describe("Singleton Behavior", () => {
        test("get() should return the same instance", () => {
            const instance1 = AuthTokensService.get();
            const instance2 = AuthTokensService.get();
            expect(instance1).toBe(instance2);
        });
    });

    describe("canRefreshToken", () => {
        const sessionId = uuid();
        const expiresAt = new Date(Date.now() + DAYS_30_MS);
        const lastRefreshAt = new Date(Date.now() - DAYS_1_MS);

        beforeEach(() => {
            jest.spyOn(Date, "now").mockReturnValue(1000000000000); // Mock Date.now()
            PrismaClient.injectData({
                session: [
                    {
                        id: sessionId,
                        expires_at: expiresAt,
                        last_refresh_at: lastRefreshAt,
                        revoked: false,
                    },
                ],
            });
        });

        afterEach(() => {
            jest.restoreAllMocks();
            PrismaClient.clearData();
        });

        describe("false", () => {
            test("not an object", async () => {
                // @ts-ignore Testing runtime scenario
                expect(await AuthTokensService.canRefreshToken(null)).toBe(false);
                // @ts-ignore Testing runtime scenario
                expect(await AuthTokensService.canRefreshToken(undefined)).toBe(false);
                // @ts-ignore Testing runtime scenario
                expect(await AuthTokensService.canRefreshToken("string")).toBe(false);
                // @ts-ignore Testing runtime scenario
                expect(await AuthTokensService.canRefreshToken(123)).toBe(false);
            });

            test("user data not present", async () => {
                const payload: SessionData = {};
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            });

            test("user array empty", async () => {
                const payload: SessionData = { users: [] };
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            });

            test("missing session data", async () => {
                const payload: SessionData = {
                    users: [
                        {
                            id: sessionId,
                        },
                    ] as SessionUser[],
                };
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            });

            test("missing lastRefreshAt", async () => {
                const payload: SessionData = {
                    users: [
                        {
                            id: uuid(),
                            session: {
                                id: sessionId,
                            }
                        },
                    ] as SessionUser[],
                };
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            });

            test("session not in database", async () => {
                const payload: SessionData = {
                    users: [
                        {
                            id: uuid(),
                            session: {
                                id: sessionId,
                                lastRefreshAt,
                            }
                        },
                    ] as SessionUser[],
                };
                PrismaClient.injectData({
                    session: [],
                });
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            });

            test("session is revoked", async () => {
                const payload: SessionData = {
                    users: [
                        {
                            id: uuid(),
                            session: {
                                id: sessionId,
                                lastRefreshAt,
                            }
                        },
                    ] as SessionUser[],
                };
                PrismaClient.injectData({
                    session: [{
                        id: sessionId,
                        expires_at: expiresAt,
                        last_refresh_at: lastRefreshAt,
                        revoked: true,
                    }],
                });
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            });

            test("session is expired", async () => {
                const payload: SessionData = {
                    users: [
                        {
                            id: uuid(),
                            session: {
                                id: sessionId,
                                lastRefreshAt,
                            }
                        },
                    ] as SessionUser[],
                };
                PrismaClient.injectData({
                    session: [{
                        id: sessionId,
                        expires_at: new Date(Date.now() - 1_000),
                        last_refresh_at: lastRefreshAt,
                        revoked: false,
                    }],
                });
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            });

            test("stored last_refresh_at does not match lastRefreshAt in payload", async () => {
                const payload: SessionData = {
                    users: [
                        {
                            id: uuid(),
                            session: {
                                id: sessionId,
                                lastRefreshAt,
                            }
                        },
                    ] as SessionUser[],
                };
                PrismaClient.injectData({
                    session: [
                        {
                            id: sessionId,
                            expires_at: expiresAt,
                            last_refresh_at: new Date(Date.now() - 1_000),
                            revoked: false,
                        },
                    ],
                });
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
            });
        });

        describe("true", () => {
            test("all conditions are met", async () => {
                const payload: SessionData = {
                    users: [
                        {
                            id: uuid(),
                            session: {
                                id: sessionId,
                                lastRefreshAt,
                            }
                        },
                    ] as SessionUser[],
                };
                expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(true);
            });
        });
    });

    describe("isTokenCorrectType", () => {
        test("returns false if token is null, undefined, or not a string", () => {
            expect(AuthTokensService.isTokenCorrectType(null)).toBe(false);
            expect(AuthTokensService.isTokenCorrectType(undefined)).toBe(false);
            expect(AuthTokensService.isTokenCorrectType(123)).toBe(false);
        });

        test("returns true if token is a string", () => {
            expect(AuthTokensService.isTokenCorrectType("string")).toBe(true);
        });
    })

    describe("isAccessTokenExpired", () => {
        describe("true", () => {
            test("string", () => {
                const payload = { accessExpiresAt: "string" };
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).toBe(true);
            });

            test("undefined", () => {
                const payload = {};
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).toBe(true);
            });

            test("null", () => {
                const payload = { accessExpiresAt: null };
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).toBe(true);
            });

            test("in the past", () => {
                const payload = JsonWebToken.createAccessExpiresAt();
                payload.accessExpiresAt = payload.accessExpiresAt - SECONDS_1_MS;
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).toBe(true);
            });

            test("negative number", () => {
                // Picked a date that would be in the future if it was positive
                const payload = JsonWebToken.createAccessExpiresAt();
                payload.accessExpiresAt = -(payload.accessExpiresAt + SECONDS_1_MS);
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).toBe(true);
            });
        });

        describe("false", () => {
            test("in the future", () => {
                const payload = JsonWebToken.createAccessExpiresAt();
                expect(AuthTokensService.isAccessTokenExpired(payload as any)).toBe(false);
            });
        })
    });

    describe("authenticateToken", () => {
        let instance: JsonWebToken;
        let privateKey: string;
        let publicKey: string;

        beforeEach(() => {
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

        afterEach(() => {
            jest.restoreAllMocks();
            PrismaClient.clearData();
        });

        describe("throws", () => {
            test("token is null, undefined, or not a string", async () => {
                await expect(AuthTokensService.authenticateToken(null as any)).rejects.toThrow();
                await expect(AuthTokensService.authenticateToken(undefined as any)).rejects.toThrow();
                await expect(AuthTokensService.authenticateToken(123 as any)).rejects.toThrow();
            });

            test("token has invalid signature", async () => {
                const invalidToken = "invalid.token.here";
                await expect(AuthTokensService.authenticateToken(invalidToken)).rejects.toThrow();
            });
        });

        test("returns existing token and payload if token is valid and unexpired", async () => {
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
        });

        test("returns new token with additional data if token is valid and unexpired, and config has additionalData", async () => {
            const payload: SessionToken = {
                ...JsonWebToken.get().basicToken(),
                ...JsonWebToken.createAccessExpiresAt(),
                isLoggedIn: false,
                timeZone: "UTC",
                users: [],
            };
            const token = JsonWebToken.get().sign(payload);

            const additionalData = { extraField: "extraValue" };
            const result = await AuthTokensService.authenticateToken(token, { additionalData });

            expect(result.token).not.toBe(token);
            expect(result.payload).toMatchObject({ ...payload, ...additionalData });
            expect(Math.abs(result.maxAge - JsonWebToken.getMaxAge(result.payload))).toBeLessThanOrEqual(SECONDS_1_MS);
        });

        test("refreshes the token if access token expired and can be refreshed", async () => {
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
            PrismaClient.injectData({
                session: [
                    {
                        id: sessionId,
                        expires_at: new Date(Date.now() + DAYS_30_MS), // Not expired
                        last_refresh_at: lastRefreshAt, // Matches payload
                        revoked: false,
                    },
                ],
            });

            const result = await AuthTokensService.authenticateToken(token);

            expect(result.token).not.toBe(token);
            expect(result.payload).toEqual({
                ...payload,
                accessExpiresAt: expect.any(Number),
                exp: expect.any(Number),
            });
            expect(result.payload.exp).toBeGreaterThan(new Date().getTime() / SECONDS_1_MS);
            expect(result.payload.accessExpiresAt).toBeGreaterThan(new Date().getTime() / SECONDS_1_MS);
            expect(result.maxAge).toBeGreaterThan(0);

            // Move time forward to simulate token expiration, and check if it can be refreshed again
            jest.spyOn(Date, "now").mockReturnValue(new Date().getTime() + 2 * ACCESS_TOKEN_EXPIRATION_MS);
            const result2 = await AuthTokensService.authenticateToken(result.token);

            expect(result2.token).not.toBe(result.token);
            expect(result2.payload).toEqual({
                ...result.payload,
                accessExpiresAt: expect.any(Number),
                exp: expect.any(Number),
            });
            expect(result2.payload.exp).toBeGreaterThan(new Date().getTime() / SECONDS_1_MS);
            expect(result2.payload.accessExpiresAt).toBeGreaterThan(new Date().getTime() / SECONDS_1_MS);
            expect(result2.maxAge).toBeGreaterThan(0);
        });

        test("throws 'SessionExpired' error if token is expired and cannot be refreshed", async () => {
            const initialTime = 1000000000000;
            jest.spyOn(Date, "now").mockReturnValue(initialTime);

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
                            lastRefreshAt: lastRefreshAt,
                        },
                    },
                ] as SessionUser[],
            };

            const token = JsonWebToken.get().sign(payload);

            // Advance time to simulate token expiration
            const expiredTime = initialTime + (5 * ACCESS_TOKEN_EXPIRATION_MS);
            jest.spyOn(Date, "now").mockReturnValue(expiredTime);

            // Inject session data
            PrismaClient.injectData({
                session: [
                    {
                        id: sessionId,
                        last_refresh_at: new Date(lastRefreshAt),
                        revoked: true, // Can't refresh a revoked session
                    },
                ],
            });

            await expect(AuthTokensService.authenticateToken(token)).rejects.toMatchObject({
                code: "SessionExpired",
            });
        });
    });

    // describe("generateApiToken", () => {
    //     let instance: JsonWebToken;
    //     let privateKey: string;
    //     let publicKey: string;
    //     let res: Response;

    //     beforeEach(() => {
    //         instance = JsonWebToken.get();
    //         const { privateKey: privKey, publicKey: pubKey } = generateKeyPairSync("rsa", {
    //             modulusLength: 2048,
    //             publicKeyEncoding: {
    //                 type: "spki",
    //                 format: "pem",
    //             },
    //             privateKeyEncoding: {
    //                 type: "pkcs8",
    //                 format: "pem",
    //             },
    //         });
    //         privateKey = privKey;
    //         publicKey = pubKey;
    //         instance.setupTestEnvironment({ privateKey, publicKey });

    //         res = {
    //             headersSent: false,
    //             cookie: jest.fn(),
    //         } as unknown as Response;
    //     });

    //     afterEach(() => {
    //         jest.restoreAllMocks();
    //     });

    //     test("should generate a JWT and add it to the response cookies", async () => {
    //         const apiToken = "test-api-token";

    //         await AuthTokensService.generateApiToken(res, apiToken);

    //         expect(res.cookie).toHaveBeenCalled();

    //         const [cookieName, token, options] = (res.cookie as jest.Mock).mock.calls[0];

    //         expect(cookieName).toBe(COOKIE.Jwt);
    //         expect(typeof token).toBe("string");
    //         expect(options).toEqual(JsonWebToken.getJwtCookieOptions());

    //         const payload = await JsonWebToken.get().verify(token);
    //         expect(payload.apiToken).toBe(apiToken);
    //     });

    //     test("should not set cookie if headers are already sent", async () => {
    //         res.headersSent = true;
    //         const apiToken = "test-api-token";

    //         await AuthTokensService.generateApiToken(res, apiToken);

    //         expect(res.cookie).not.toHaveBeenCalled();
    //     });
    // });

    // describe("generateSessionToken", () => {
    //     let instance: JsonWebToken;
    //     let privateKey: string;
    //     let publicKey: string;
    //     let res: Response;

    //     beforeEach(() => {
    //         instance = JsonWebToken.get();
    //         const { privateKey: privKey, publicKey: pubKey } = generateKeyPairSync("rsa", {
    //             modulusLength: 2048,
    //             publicKeyEncoding: {
    //                 type: "spki",
    //                 format: "pem",
    //             },
    //             privateKeyEncoding: {
    //                 type: "pkcs8",
    //                 format: "pem",
    //             },
    //         });
    //         privateKey = privKey;
    //         publicKey = pubKey;
    //         instance.setupTestEnvironment({ privateKey, publicKey });

    //         res = {
    //             headersSent: false,
    //             cookie: jest.fn(),
    //         } as unknown as Response;
    //     });

    //     afterEach(() => {
    //         jest.restoreAllMocks();
    //     });

    //     test("should generate a JWT with session data and add it to the response cookies", async () => {
    //         const session: Partial<SessionData> = {
    //             isLoggedIn: true,
    //             timeZone: "UTC",
    //             users: [
    //                 {
    //                     id: "user1",
    //                     session: {
    //                         id: "session1",
    //                         lastRefreshAt: new Date().toISOString(),
    //                     },
    //                 },
    //                 {
    //                     id: "user2",
    //                     session: {
    //                         id: "session2",
    //                         lastRefreshAt: new Date().toISOString(),
    //                     },
    //                 },
    //             ],
    //         };

    //         // Mock SessionService.getUser to return the current user
    //         jest.spyOn(SessionService, "getUser").mockReturnValue(session.users![0]);

    //         await AuthTokensService.generateSessionToken(res, session);

    //         expect(res.cookie).toHaveBeenCalled();

    //         const [cookieName, token, options] = (res.cookie as jest.Mock).mock.calls[0];

    //         expect(cookieName).toBe(COOKIE.Jwt);
    //         expect(typeof token).toBe("string");
    //         expect(options).toEqual(JsonWebToken.getJwtCookieOptions());

    //         const payload = await JsonWebToken.get().verify(token);

    //         expect(payload.isLoggedIn).toBe(true);
    //         expect(payload.timeZone).toBe("UTC");
    //         expect(payload.users).toEqual(session.users);
    //     });

    //     test("should not set cookie if headers are already sent", async () => {
    //         res.headersSent = true;
    //         const session: Partial<SessionData> = {};

    //         await AuthTokensService.generateSessionToken(res, session);

    //         expect(res.cookie).not.toHaveBeenCalled();
    //     });
    // });
});

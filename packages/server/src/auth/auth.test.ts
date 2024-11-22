import { DAYS_1_MS, DAYS_30_MS, SECONDS_1_MS, SessionUser, uuid } from "@local/shared";
import { generateKeyPairSync } from "crypto";
import pkg from "../__mocks__/@prisma/client";
import { SessionData, SessionToken } from "../types";
import { AuthTokensService } from "./auth";
import { ACCESS_TOKEN_EXPIRATION_MS, JsonWebToken, REFRESH_TOKEN_EXPIRATION_MS } from "./jwt";

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

        beforeEach(() => {
            jest.spyOn(Date, "now").mockReturnValue(1000000000000); // Mock Date.now()
        });

        afterEach(() => {
            jest.restoreAllMocks();
            PrismaClient.clearData();
        });

        test("returns false if payload is not an object", async () => {
            // @ts-ignore Testing runtime scenario
            expect(await AuthTokensService.canRefreshToken(null)).toBe(false);
            // @ts-ignore Testing runtime scenario
            expect(await AuthTokensService.canRefreshToken(undefined)).toBe(false);
            // @ts-ignore Testing runtime scenario
            expect(await AuthTokensService.canRefreshToken("string")).toBe(false);
            // @ts-ignore Testing runtime scenario
            expect(await AuthTokensService.canRefreshToken(123)).toBe(false);
        });

        test("returns false if user is not found", async () => {
            const payload: SessionData = {};
            expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
        });

        test("returns false if sessionId or lastRefreshAt is missing", async () => {
            const payload: SessionData = {
                users: [
                    {
                        id: uuid()
                        // Omit session data
                    },
                ] as SessionUser[],
            };
            const user = { session: {} };
            expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
        });

        test("returns false if session not found in database", async () => {
            const sessionId = uuid();
            const lastRefreshAt = new Date(Date.now() - DAYS_1_MS).toISOString();
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

        test("returns false if session is revoked", async () => {
            const sessionId = uuid();
            const lastRefreshAt = new Date(Date.now() - DAYS_1_MS).toISOString();
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
                        last_refresh_at: new Date(lastRefreshAt),
                        revoked: true,
                    },
                ],
            });
            expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
        });

        test("returns false if stored last_refresh_at does not match lastRefreshAt in payload", async () => {
            const sessionId = uuid();
            const payload: SessionData = {
                users: [
                    {
                        id: uuid(),
                        session: {
                            id: sessionId,
                            lastRefreshAt: new Date(Date.now() - DAYS_1_MS).toISOString(),
                        }
                    },
                ] as SessionUser[],
            };
            PrismaClient.injectData({
                session: [
                    {
                        id: sessionId,
                        last_refresh_at: new Date(Date.now() - DAYS_1_MS + 1_000),
                        revoked: false,
                    },
                ],
            });
            expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
        });

        test("returns false if time since last refresh is greater than REFRESH_TOKEN_EXPIRATION_MS", async () => {
            const sessionId = uuid();
            // Last refreshed longer than REFRESH_TOKEN_EXPIRATION_MS
            const lastRefreshAt = new Date(Date.now() - REFRESH_TOKEN_EXPIRATION_MS - DAYS_1_MS).toISOString();
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
                        last_refresh_at: new Date(lastRefreshAt),
                        revoked: false,
                    },
                ],
            });
            expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(false);
        });

        test("returns true when all conditions are met", async () => {
            const sessionId = uuid();
            const lastRefreshAt = new Date(Date.now() - DAYS_1_MS).toISOString();
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
                        last_refresh_at: new Date(lastRefreshAt),
                        revoked: false,
                    },
                ],
            });
            expect(await AuthTokensService.canRefreshToken(payload as SessionToken)).toBe(true);
        });
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

        test("throws error if token is null, undefined, or not a string", async () => {
            await expect(AuthTokensService.authenticateToken(null as any)).rejects.toThrow();
            await expect(AuthTokensService.authenticateToken(undefined as any)).rejects.toThrow();
            await expect(AuthTokensService.authenticateToken(123 as any)).rejects.toThrow();
        });

        test("throws error if token has invalid signature", async () => {
            const invalidToken = "invalid.token.here";
            await expect(AuthTokensService.authenticateToken(invalidToken)).rejects.toThrow();
        });

        test("returns existing token and payload if token is valid and unexpired", async () => {
            const payload: SessionToken = {
                ...JsonWebToken.get().basicToken(),
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

        test("refreshes the token if expired and can be refreshed", async () => {
            const initialTime = 1000000000000;
            jest.spyOn(Date, "now").mockReturnValue(initialTime);

            const sessionId = uuid();
            const lastRefreshAt = new Date(initialTime - DAYS_30_MS).toISOString();

            const payload: SessionToken = {
                ...JsonWebToken.get().basicToken(),
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
            jest.spyOn(Date, "now").mockReturnValue(expiredTime);

            // Inject session data
            PrismaClient.injectData({
                session: [
                    {
                        id: sessionId,
                        last_refresh_at: new Date(lastRefreshAt),
                        revoked: false,
                    },
                ],
            });

            const result = await AuthTokensService.authenticateToken(token);

            expect(result.token).not.toBe(token);
            expect(result.payload).toEqual({
                ...payload,
                exp: expect.any(Number),
            });
            // "exp" should be around ACCESS_TOKEN_EXPIRATION_MS from now
            expect(Math.abs((result.payload.exp * SECONDS_1_MS) - ACCESS_TOKEN_EXPIRATION_MS - Date.now())).toBeLessThanOrEqual(SECONDS_1_MS);
            expect(Math.abs(result.maxAge - ACCESS_TOKEN_EXPIRATION_MS)).toBeLessThanOrEqual(SECONDS_1_MS);
        });

        test("throws 'SessionExpired' error if token is expired and cannot be refreshed", async () => {
            const initialTime = 1000000000000;
            jest.spyOn(Date, "now").mockReturnValue(initialTime);

            const sessionId = uuid();
            const lastRefreshAt = new Date(initialTime - DAYS_30_MS).toISOString();

            const payload: SessionToken = {
                ...JsonWebToken.get().basicToken(),
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

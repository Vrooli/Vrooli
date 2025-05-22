/* eslint-disable @typescript-eslint/ban-ts-comment */
import { DEFAULT_LANGUAGE, type SessionUser, generatePK } from "@local/shared";
import { expect } from "chai";
import { type Request } from "express";
import type { Cluster, Redis } from "ioredis";
import sinon from "sinon";
import { type Socket } from "socket.io";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { CacheService } from "../redisConn.js";
import { type SessionData } from "../types.js";
import { RequestService } from "./request.js";
import { SessionService } from "./session.js";

describe("RequestService", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;
    let redisClient: Redis | Cluster | null = null;

    before(async function before() {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");

        redisClient = await CacheService.get().raw();
    });

    after(async function after() {
        loggerErrorStub.restore();
        loggerInfoStub.restore();

        if (redisClient) {
            await redisClient.flushall();
        }
    });

    describe("isValidIP", () => {
        it("should validate IPv4 addresses", () => {
            expect(RequestService.isValidIP("192.168.1.1")).to.be.ok;
            expect(RequestService.isValidIP("255.255.255.255")).to.be.ok;
            expect(RequestService.isValidIP("0.0.0.0")).to.be.ok;
            expect(RequestService.isValidIP("999.999.999.999")).to.not.be.ok;
        });

        it("should validate IPv6 addresses", () => {
            expect(RequestService.isValidIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334")).to.be.ok;
            expect(RequestService.isValidIP("::1")).to.be.ok;
            expect(RequestService.isValidIP("::ffff:c0a8:101")).to.be.ok;
            expect(RequestService.isValidIP("gibberish")).to.not.be.ok;
        });
    });

    describe("isValidDomain", () => {
        it("should validate domain names", () => {
            expect(RequestService.isValidDomain("example.com")).to.be.ok;
            expect(RequestService.isValidDomain("subdomain.example.com")).to.be.ok;
            expect(RequestService.isValidDomain("www.example.co.uk")).to.be.ok;
            expect(RequestService.isValidDomain("example")).to.not.be.ok;
            expect(RequestService.isValidDomain("example..com")).to.not.be.ok;
            expect(RequestService.isValidDomain(".com")).to.not.be.ok;
            expect(RequestService.isValidDomain("com.")).to.not.be.ok;
            expect(RequestService.isValidDomain("example.com/")).to.not.be.ok;
        });
    });

    describe("isSafeOrigin", () => {
        beforeEach(async () => {
            RequestService.get().resetCachedOrigins();
        });

        function mockRequest(origin: string | undefined, referer: string | undefined): Request {
            return {
                headers: {
                    origin,
                    referer,
                },
            } as Request;
        }

        it("remote production (actual production, possibly)", () => {
            process.env.NODE_ENV = "production";
            process.env.SITE_IP = "123.69.4.20";
            process.env.VITE_SERVER_LOCATION = "remote";
            process.env.VIRTUAL_HOST = "testsite.com,www.testsite.com,app.testsite.com,www.app.testsite.com";

            const allowedOrigins = [
                "http://123.69.4.20", "http://123.69.4.20:3000",
                "http://testsite.com", "https://testsite.com",
                "http://www.testsite.com", "https://www.testsite.com",
                "http://app.testsite.com", "https://app.testsite.com",
                "http://www.app.testsite.com", "https://www.app.testsite.com",
            ];

            const disallowedOrigins = [
                "http://localhost", "http://localhost:3000",
                "http://192.168.0.1", "http://192.168.0.1:3000",
                "http://142.212.123.075", "http://142.212.123.075:1234",
                "http://www.unsafesite.com", "https://unsafesite.com",
            ];

            // Test allowed origins
            for (const origin of allowedOrigins) {
                expect(RequestService.get().isSafeOrigin(mockRequest(origin, undefined))).to.be.ok;
                expect(RequestService.get().isSafeOrigin(mockRequest(undefined, origin))).to.be.ok;
            }

            // Test disallowed origins
            for (const origin of disallowedOrigins) {
                expect(RequestService.get().isSafeOrigin(mockRequest(origin, undefined))).to.not.be.ok;
                expect(RequestService.get().isSafeOrigin(mockRequest(undefined, origin))).to.not.be.ok;
            }
        });

        it("local production (staging)", () => {
            process.env.NODE_ENV = "production";
            process.env.SITE_IP = "123.69.4.20";
            process.env.VITE_SERVER_LOCATION = "local";
            process.env.VIRTUAL_HOST = "testsite.com,www.testsite.com,app.testsite.com,www.app.testsite.com";

            const allowedOrigins = [
                "http://localhost", "http://localhost:3000",
                "http://123.69.4.20", "http://123.69.4.20:3000",
                "http://192.168.0.1", "http://192.168.0.1:3000",
                "http://testsite.com", "https://testsite.com",
                "http://www.testsite.com", "https://www.testsite.com",
                "http://app.testsite.com", "https://app.testsite.com",
                "http://www.app.testsite.com", "https://www.app.testsite.com",
            ];

            const disallowedOrigins = [
                "http://142.212.123.075", "http://142.212.123.075:1234",
                "http://www.unsafesite.com", "https://unsafesite.com",
            ];

            // Test allowed origins
            for (const origin of allowedOrigins) {
                expect(RequestService.get().isSafeOrigin(mockRequest(origin, undefined))).to.be.ok;
                expect(RequestService.get().isSafeOrigin(mockRequest(undefined, origin))).to.be.ok;
            }

            // Test disallowed origins
            for (const origin of disallowedOrigins) {
                expect(RequestService.get().isSafeOrigin(mockRequest(origin, undefined))).to.not.be.ok;
                expect(RequestService.get().isSafeOrigin(mockRequest(undefined, origin))).to.not.be.ok;
            }
        });

        it("allow all origins in development", () => {
            process.env.NODE_ENV = "development";

            const allowedOrigins = [
                "http://localhost", "http://localhost:3000",
                "http://123.69.4.20", "http://123.69.4.20:3000",
                "http://192.168.0.1", "http://192.168.0.1:3000",
                "http://142.212.123.075", "http://142.212.123.075:1234",
                "http://testsite.com", "https://testsite.com",
                "http://www.testsite.com", "https://www.testsite.com",
                "http://app.testsite.com", "https://app.testsite.com",
                "http://www.app.testsite.com", "https://www.app.testsite.com",
                "http://www.unsafesite.com", "https://unsafesite.com",
            ];

            // Test allowed origins
            for (const origin of allowedOrigins) {
                expect(RequestService.get().isSafeOrigin(mockRequest(origin, undefined))).to.be.ok;
                expect(RequestService.get().isSafeOrigin(mockRequest(undefined, origin))).to.be.ok;
            }
        });
    });

    describe("getDeviceInfo", () => {
        it("should return correct device info when headers are present", () => {
            const req = {
                headers: {
                    "user-agent": "Mozilla/5.0",
                    "accept-language": "en-US,en;q=0.9",
                },
            } as Request;
            const result = RequestService.getDeviceInfo(req);
            expect(result).to.equal("User-Agent: Mozilla/5.0; Accept-Language: en-US,en;q=0.9");
        });

        it("should return 'Unknown' for missing user-agent", () => {
            const req = {
                headers: {
                    "accept-language": "en-US,en;q=0.9",
                },
            } as Request;
            const result = RequestService.getDeviceInfo(req);
            expect(result).to.equal("User-Agent: Unknown; Accept-Language: en-US,en;q=0.9");
        });

        it("should return 'Unknown' for missing accept-language", () => {
            const req = {
                headers: {
                    "user-agent": "Mozilla/5.0",
                },
            } as Request;
            const result = RequestService.getDeviceInfo(req);
            expect(result).to.equal("User-Agent: Mozilla/5.0; Accept-Language: Unknown");
        });

        it("should return 'Unknown' for both missing headers", () => {
            const req = {
                headers: {},
            } as Request;
            const result = RequestService.getDeviceInfo(req);
            expect(result).to.equal("User-Agent: Unknown; Accept-Language: Unknown");
        });
    });

    describe("parseAcceptLanguage", () => {
        it(`should return ["${DEFAULT_LANGUAGE}"] when accept-language header is missing`, () => {
            const req = { headers: {} };
            expect(RequestService.parseAcceptLanguage(req)).to.deep.equal([DEFAULT_LANGUAGE]);
        });

        it(`should return ["${DEFAULT_LANGUAGE}"] when accept-language header is "*" `, () => {
            const req = { headers: { "accept-language": "*" } };
            expect(RequestService.parseAcceptLanguage(req)).to.deep.equal([DEFAULT_LANGUAGE]);
        });

        it("should parse \"en-US,en;q=0.9,fr;q=0.8\" correctly", () => {
            const req = { headers: { "accept-language": "en-US,en;q=0.9,fr;q=0.8" } };
            expect(RequestService.parseAcceptLanguage(req)).to.deep.equal(["en", "en", "fr"]);
        });

        it("should parse \"fr-CA,fr-FR;q=0.8,en-US;q=0.5,en;q=0.3\" correctly", () => {
            const req = { headers: { "accept-language": "fr-CA,fr-FR;q=0.8,en-US;q=0.5,en;q=0.3" } };
            expect(RequestService.parseAcceptLanguage(req)).to.deep.equal(["fr", "fr", "en", "en"]);
        });

        it("should parse \"es\" correctly", () => {
            const req = { headers: { "accept-language": "es" } };
            expect(RequestService.parseAcceptLanguage(req)).to.deep.equal(["es"]);
        });

        it("should parse \"zh-Hant,zh-Hans;q=0.9,ja;q=0.8\" correctly", () => {
            const req = { headers: { "accept-language": "zh-Hant,zh-Hans;q=0.9,ja;q=0.8" } };
            expect(RequestService.parseAcceptLanguage(req)).to.deep.equal(["zh", "zh", "ja"]);
        });

        it("should parse \"en-GB,en;q=0.8\" correctly", () => {
            const req = { headers: { "accept-language": "en-GB,en;q=0.8" } };
            expect(RequestService.parseAcceptLanguage(req)).to.deep.equal(["en", "en"]);
        });

        it("should handle accept-language with wildcard and language codes", () => {
            const req = { headers: { "accept-language": "*,en;q=0.5" } };
            expect(RequestService.parseAcceptLanguage(req)).to.deep.equal(["*", "en"]);
        });

        it("should handle invalid accept-language values", () => {
            const req = { headers: { "accept-language": "invalid" } };
            expect(RequestService.parseAcceptLanguage(req)).to.deep.equal(["invalid"]);
        });

        it(`should return ["${DEFAULT_LANGUAGE}"] when accept-language header is empty string`, () => {
            const req = { headers: { "accept-language": "" } };
            expect(RequestService.parseAcceptLanguage(req)).to.deep.equal([DEFAULT_LANGUAGE]);
        });

        it("should handle accept-language with extra spaces", () => {
            const req = { headers: { "accept-language": " en-US , en ;q=0.9 " } };
            expect(RequestService.parseAcceptLanguage(req)).to.deep.equal([" en", " en "]);
        });

        it(`should return ["${DEFAULT_LANGUAGE}"] when accept-language header is null`, () => {
            const req = { headers: { "accept-language": null } };
            expect(RequestService.parseAcceptLanguage(req)).to.deep.equal([DEFAULT_LANGUAGE]);
        });

        it(`should return ["${DEFAULT_LANGUAGE}"] when accept-language header is undefined`, () => {
            const req = { headers: { "accept-language": undefined } };
            expect(RequestService.parseAcceptLanguage(req)).to.deep.equal([DEFAULT_LANGUAGE]);
        });

        it("should handle accept-language with uppercase letters", () => {
            const req = { headers: { "accept-language": "EN-US,en;q=0.8" } };
            expect(RequestService.parseAcceptLanguage(req)).to.deep.equal(["EN", "en"]);
        });

        it(`should return ["${DEFAULT_LANGUAGE}"]  when accept-language header is not a string`, () => {
            const req = { headers: { "accept-language": 42 } };
            expect(RequestService.parseAcceptLanguage(req)).to.deep.equal([DEFAULT_LANGUAGE]);
        });
    });

    describe("assertRequestFrom", () => {
        let sessionData: SessionData;
        let userData: SessionUser;
        let sandbox;

        beforeEach(() => {
            sandbox = sinon.createSandbox();
            sessionData = {
                isLoggedIn: true,
                apiToken: null,
                fromSafeOrigin: true,
            };
            userData = {
                id: generatePK().toString(),
                username: "testuser",
            } as unknown as SessionUser;
        });

        afterEach(() => {
            sandbox.restore();
        });

        describe("isApiToken conditions", () => {
            it("should pass when isApiToken is true and request is from API token", () => {
                sessionData.isLoggedIn = false;
                sessionData.apiToken = "testApiToken";
                // @ts-ignore Testing runtime scenario
                sandbox.stub(SessionService, "getUser").returns(undefined);

                const req = { session: sessionData };
                const conditions = { isApiToken: true } as const;
                const result = RequestService.assertRequestFrom(req, conditions);
                expect(result).to.deep.equal({});
            });

            it("should throw MustUseApiToken when isApiToken is true but request is not from API token", () => {
                sessionData.isLoggedIn = false;
                sessionData.apiToken = null;
                sandbox.stub(SessionService, "getUser").returns(null);

                const req = { session: sessionData };
                const conditions = { isApiToken: true } as const;
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).to.throw();
            });
        });

        describe("isUser conditions", () => {
            it("should return user data when isUser is true and conditions are met (from safe origin)", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = null;
                sessionData.fromSafeOrigin = true;
                sandbox.stub(SessionService, "getUser").returns(userData);

                const req = { session: sessionData };
                const conditions = { isUser: true } as const;
                const result = RequestService.assertRequestFrom(req, conditions);
                expect(result).to.deep.equal(userData);
            });

            it("should return user data when isUser is true and conditions are met (apiToken)", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = "testApiToken";
                sessionData.fromSafeOrigin = false;
                sandbox.stub(SessionService, "getUser").returns(userData);

                const req = { session: sessionData };
                const conditions = { isUser: true } as const;
                const result = RequestService.assertRequestFrom(req, conditions);
                expect(result).to.deep.equal(userData);
            });

            it("should throw NotLoggedIn when isUser is true and user is not logged in", () => {
                sessionData.isLoggedIn = false;
                sessionData.apiToken = null;
                sessionData.fromSafeOrigin = true;
                sandbox.stub(SessionService, "getUser").returns(null);

                const req = { session: sessionData };
                const conditions = { isUser: true } as const;
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).to.throw();
            });

            it("should throw NotLoggedIn when isUser is true but conditions are not met", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = null;
                sessionData.fromSafeOrigin = false;
                sandbox.stub(SessionService, "getUser").returns(userData);

                const req = { session: sessionData };
                const conditions = { isUser: true } as const;
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).to.throw();
            });
        });

        describe("isOfficialUser conditions", () => {
            it("should return user data when isOfficialUser is true and conditions are met", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = null;
                sessionData.fromSafeOrigin = true;
                sandbox.stub(SessionService, "getUser").returns(userData);

                const req = { session: sessionData };
                const conditions = { isOfficialUser: true } as const;
                const result = RequestService.assertRequestFrom(req, conditions);
                expect(result).to.deep.equal(userData);
            });

            it("should throw NotLoggedInOfficial when isOfficialUser is true but user is not logged in", () => {
                sessionData.isLoggedIn = false;
                sessionData.apiToken = null;
                sessionData.fromSafeOrigin = true;
                sandbox.stub(SessionService, "getUser").returns(null);

                const req = { session: sessionData };
                const conditions = { isOfficialUser: true } as const;
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).to.throw();
            });

            it("should throw NotLoggedInOfficial when isOfficialUser is true but apiToken is true", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = "testApiToken";
                sessionData.fromSafeOrigin = true;
                sandbox.stub(SessionService, "getUser").returns(userData);

                const req = { session: sessionData };
                const conditions = { isOfficialUser: true } as const;
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).to.throw();
            });

            it("should throw NotLoggedInOfficial when isOfficialUser is true but fromSafeOrigin is false", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = null;
                sessionData.fromSafeOrigin = false;
                sandbox.stub(SessionService, "getUser").returns(userData);

                const req = { session: sessionData };
                const conditions = { isOfficialUser: true } as const;
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).to.throw();
            });
        });

        it("should return an empty object when no conditions are specified", () => {
            sandbox.stub(SessionService, "getUser").returns(userData);
            const req = { session: sessionData };
            const conditions = {};
            const result = RequestService.assertRequestFrom(req, conditions);
            expect(result).to.deep.equal({});
        });
    });

    async function setRateLimitTokens(key: string, tokens: number, lastRefill: number) {
        await redisClient?.set(`${key}:tokens`, tokens.toString());
        await redisClient?.set(`${key}:lastRefill`, lastRefill.toString());
    }

    describe("checkRateLimit", () => {
        it("should not throw if all keys are allowed", async () => {
            const keys = ["key1", "key2"];
            const maxTokensList = [100, 100];
            const refillRates = [1, 1]; // 1 token per second
            const now = Date.now();

            // Set tokens >= 1 and recent lastRefill for both keys
            await setRateLimitTokens("key1", 100, now);
            await setRateLimitTokens("key2", 100, now);

            // Should complete without throwing
            await RequestService.get().checkRateLimit(redisClient, keys, maxTokensList, refillRates);
        });

        it("should throw CustomError if any key is not allowed", async () => {
            const keys = ["key1", "key2"];
            const maxTokensList = [100, 100];
            const refillRates = [1, 1]; // 1 token per second
            const now = Date.now();

            // key1 has enough tokens
            await setRateLimitTokens("key1", 100, now);
            // key2 has no tokens and last refill is recent, so no refill occurs
            await setRateLimitTokens("key2", 0, now);

            try {
                await RequestService.get().checkRateLimit(redisClient, keys, maxTokensList, refillRates);
                expect.fail("Expected checkRateLimit to throw");
            } catch (error) {
                expect(error).to.be.instanceOf(CustomError);
                expect(error.code).to.equal("RateLimitExceeded");
            }
        });

        it("should load script and retry on NOSCRIPT error", async () => {
            const keys = ["key1", "key2"];
            const maxTokensList = [100, 100];
            const refillRates = [1, 1];
            const now = Date.now();

            // Set tokens >= 1 for both keys
            await setRateLimitTokens("key1", 100, now);
            await setRateLimitTokens("key2", 100, now);

            // Simulate NOSCRIPT by flushing scripts
            await redisClient?.script("FLUSH");

            // Execute the rate limit check without throwing
            await RequestService.get().checkRateLimit(redisClient, keys, maxTokensList, refillRates);

            // Verify the script was loaded (SHA is set)
            expect(RequestService["tokenBucketScriptSha"]).to.be.a("string");
        });

        it("should not throw if client is null", async () => {
            await RequestService.get().checkRateLimit(null, ["key1"], [100], [1]);
        });
    });

    describe("rateLimit", () => {
        it("should not throw if rate limit is not exceeded for API token (GraphQL)", async () => {
            const operationName = "testOperation1";
            const apiToken = "testApiToken";
            const ip = "192.168.1.1";
            const req = {
                session: {
                    apiToken,
                    fromSafeOrigin: false,
                    isLoggedIn: false,
                },
                body: { operationName },
                ip,
            } as unknown as Request;
            const apiKey = RequestService["buildApiKey"](req);
            const ipKey = RequestService["buildIpKey"](req);

            await setRateLimitTokens(apiKey, 100, Date.now());
            await setRateLimitTokens(ipKey, 100, Date.now());

            // Should complete without throwing
            await RequestService.get().rateLimit({ req, maxApi: 100, maxIp: 100, maxUser: 100, window: 60 });
        });

        it("should throw if API rate limit is exceeded (GraphQL)", async () => {
            const operationName = "testOperation2";
            const apiToken = "testApiToken";
            const ip = "192.168.1.1";
            const req = {
                session: {
                    apiToken,
                    fromSafeOrigin: false,
                    isLoggedIn: false,
                },
                body: { operationName },
                ip,
            } as unknown as Request;
            const apiKey = RequestService["buildApiKey"](req);
            const ipKey = RequestService["buildIpKey"](req);

            await setRateLimitTokens(apiKey, 0, Date.now());
            await setRateLimitTokens(ipKey, 100, Date.now());

            try {
                await RequestService.get().rateLimit({ req, maxApi: 100, maxIp: 100, maxUser: 100, window: 60 });
                expect.fail("Expected rateLimit to throw");
            } catch (error) {
                expect(error).to.be.instanceOf(CustomError);
                expect(error.code).to.equal("RateLimitExceeded");
            }
        });

        it("should not throw if rate limit is not exceeded for IP (no API token, safe origin, REST)", async () => {
            const path = "/api/test";
            const method = "GET";
            const apiToken = null;
            const ip = "192.168.1.1";
            const req = {
                session: {
                    apiToken,
                    fromSafeOrigin: true,
                    isLoggedIn: false,
                },
                route: { path },
                method,
                ip,
            } as unknown as Request;
            const ipKey = RequestService["buildIpKey"](req);
            const now = Date.now();

            await setRateLimitTokens(ipKey, 100, now);

            // Should complete without throwing
            await RequestService.get().rateLimit({ req, maxApi: 100, maxIp: 100, maxUser: 100, window: 60 });
        });

        it("should throw if IP rate limit is exceeded (no API token, safe origin, REST)", async () => {
            const path = "/api/test";
            const method = "GET";
            const apiToken = null;
            const ip = "192.168.1.1";
            const req = {
                session: {
                    apiToken,
                    fromSafeOrigin: true,
                    isLoggedIn: false,
                },
                route: { path },
                method,
                ip,
            } as unknown as Request;
            const ipKey = RequestService["buildIpKey"](req);

            await setRateLimitTokens(ipKey, 0, Date.now());

            try {
                await RequestService.get().rateLimit({ req, maxApi: 100, maxIp: 100, maxUser: 100, window: 60 });
                expect.fail("Expected rateLimit to throw");
            } catch (error) {
                expect(error).to.be.instanceOf(CustomError);
                expect(error.code).to.equal("RateLimitExceeded");
            }
        });

        it("should throw MustUseApiToken if no API token and from unsafe origin", async () => {
            const req = {
                session: {
                    apiToken: null,
                    fromSafeOrigin: false,
                    isLoggedIn: false,
                },
                body: { operationName: "testOperation3" },
                ip: "192.168.1.1",
            } as unknown as Request;

            try {
                await RequestService.get().rateLimit({ req, maxApi: 100, maxIp: 100, maxUser: 100, window: 60 });
                expect.fail("Expected rateLimit to throw");
            } catch (error) {
                expect(error).to.be.instanceOf(CustomError);
                expect(error.code).to.equal("MustUseApiToken");
            }
        });

        it("should not throw if rate limit is not exceeded for logged-in user (other request)", async () => {
            const path = "/other";
            const ip = "192.168.1.1";
            const userData = { id: generatePK().toString(), languages: [] } as unknown as SessionUser;
            const req = {
                session: {
                    apiToken: null,
                    fromSafeOrigin: true,
                    isLoggedIn: true,
                    users: [userData],
                },
                path,
                ip,
            } as unknown as Request;
            const ipKey = RequestService["buildIpKey"](req);

            await setRateLimitTokens(ipKey, 100, Date.now());

            // Should complete without throwing
            await RequestService.get().rateLimit({ req, maxApi: 100, maxIp: 100, maxUser: 100, window: 60 });
        });

        it("should throw if user rate limit is exceeded", async () => {
            const path = "/other";
            const ip = "192.168.1.1";
            const userData = { id: generatePK().toString(), languages: [] } as unknown as SessionUser;
            const req = {
                session: {
                    apiToken: null,
                    fromSafeOrigin: true,
                    isLoggedIn: true,
                    users: [userData],
                },
                path,
                ip,
            } as unknown as Request;
            const ipKey = RequestService["buildIpKey"](req);
            const userKey = RequestService["buildUserKey"](req, userData);

            await setRateLimitTokens(ipKey, 100, Date.now());
            await setRateLimitTokens(userKey, 0, Date.now());

            try {
                await RequestService.get().rateLimit({ req, maxApi: 100, maxIp: 100, maxUser: 100, window: 60 });
                expect.fail("Expected rateLimit to throw");
            } catch (error) {
                expect(error).to.be.instanceOf(CustomError);
                expect(error.code).to.equal("RateLimitExceeded");
            }
        });
    });

    describe("rateLimitSocket", () => {
        it("should not return error if rate limit is not exceeded for guest socket", async () => {
            const socketId = "socket1";
            const ip = "192.168.1.1";
            const socket = {
                session: { isLoggedIn: false, users: [] },
                req: { ip },
                id: socketId,
            } as unknown as Socket;
            const ipKey = RequestService["buildSocketIpKey"](socket);
            const now = Date.now();
            await setRateLimitTokens(ipKey, 100, now);

            const result = await RequestService.get().rateLimitSocket({ socket, maxIp: 100, maxUser: 100, window: 60 });
            expect(result).to.be.undefined;
        });

        it("should return error if IP rate limit is exceeded for guest socket", async () => {
            const socketId = "socket2";
            const ip = "192.168.1.1";
            const socket = {
                session: { isLoggedIn: false, users: [] },
                req: { ip },
                id: socketId,
            } as unknown as Socket;
            const ipKey = RequestService["buildSocketIpKey"](socket);
            await setRateLimitTokens(ipKey, 0, Date.now());

            const error = await RequestService.get().rateLimitSocket({ socket, maxIp: 100, maxUser: 100, window: 60 });
            expect(error).to.be.a("string").and.to.include("RateLimitExceeded");
        });

        it("should not return error if rate limit is not exceeded for logged-in user socket", async () => {
            const socketId = "socket3";
            const ip = "192.168.1.1";
            const userData = { id: generatePK().toString(), languages: [] } as unknown as SessionUser;
            const socket = {
                session: { isLoggedIn: true, users: [userData] },
                req: { ip },
                id: socketId,
            } as unknown as Socket;
            const ipKey = RequestService["buildSocketIpKey"](socket);
            const userKey = RequestService["buildSocketUserKey"](socket, userData);
            const now = Date.now();
            await setRateLimitTokens(ipKey, 100, now);
            await setRateLimitTokens(userKey, 100, now);

            const result = await RequestService.get().rateLimitSocket({ socket, maxIp: 100, maxUser: 100, window: 60 });
            expect(result).to.be.undefined;
        });

        it("should return error if user rate limit is exceeded for logged-in user socket", async () => {
            const socketId = "socket4";
            const ip = "192.168.1.1";
            const userData = { id: generatePK().toString(), languages: [] } as unknown as SessionUser;
            const socket = {
                session: { isLoggedIn: true, users: [userData] },
                req: { ip },
                id: socketId,
            } as unknown as Socket;
            const ipKey = RequestService["buildSocketIpKey"](socket);
            const userKey = RequestService["buildSocketUserKey"](socket, userData);
            await setRateLimitTokens(ipKey, 100, Date.now());
            await setRateLimitTokens(userKey, 0, Date.now());

            const error = await RequestService.get().rateLimitSocket({ socket, maxIp: 100, maxUser: 100, window: 60 });
            expect(error).to.be.a("string").and.to.include("RateLimitExceeded");
        });
    });
});

/* eslint-disable @typescript-eslint/ban-ts-comment */
import { DEFAULT_LANGUAGE, SessionUser } from "@local/shared";
import { Request } from "express";
import { SessionData } from "../types";
import { RequestService } from "./request";
import { SessionService } from "./session";

describe("RequestService", () => {
    describe("isValidIP", () => {
        it("should validate IPv4 addresses", () => {
            expect(RequestService.isValidIP("192.168.1.1")).toBeTruthy();
            expect(RequestService.isValidIP("255.255.255.255")).toBeTruthy();
            expect(RequestService.isValidIP("0.0.0.0")).toBeTruthy();
            expect(RequestService.isValidIP("999.999.999.999")).toBeFalsy();
        });

        it("should validate IPv6 addresses", () => {
            expect(RequestService.isValidIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334")).toBeTruthy();
            expect(RequestService.isValidIP("::1")).toBeTruthy();
            expect(RequestService.isValidIP("::ffff:c0a8:101")).toBeTruthy();
            expect(RequestService.isValidIP("gibberish")).toBeFalsy();
        });
    });

    describe("isValidDomain", () => {
        it("should validate domain names", () => {
            expect(RequestService.isValidDomain("example.com")).toBeTruthy();
            expect(RequestService.isValidDomain("subdomain.example.com")).toBeTruthy();
            expect(RequestService.isValidDomain("www.example.co.uk")).toBeTruthy();
            expect(RequestService.isValidDomain("example")).toBeFalsy();
            expect(RequestService.isValidDomain("example..com")).toBeFalsy();
            expect(RequestService.isValidDomain(".com")).toBeFalsy();
            expect(RequestService.isValidDomain("com.")).toBeFalsy();
            expect(RequestService.isValidDomain("example.com/")).toBeFalsy();
        });
    });

    describe("isSafeOrigin", () => {
        beforeEach(async () => {
            RequestService.get().resetCachedOrigins();
        });

        const mockRequest = (origin: string | undefined, referer: string | undefined): Request => ({
            headers: {
                origin,
                referer,
            },
        } as Request);

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
                expect(RequestService.get().isSafeOrigin(mockRequest(origin, undefined))).toBeTruthy();
                expect(RequestService.get().isSafeOrigin(mockRequest(undefined, origin))).toBeTruthy();
            }

            // Test disallowed origins
            for (const origin of disallowedOrigins) {
                expect(RequestService.get().isSafeOrigin(mockRequest(origin, undefined))).toBeFalsy();
                expect(RequestService.get().isSafeOrigin(mockRequest(undefined, origin))).toBeFalsy();
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
                expect(RequestService.get().isSafeOrigin(mockRequest(origin, undefined))).toBeTruthy();
                expect(RequestService.get().isSafeOrigin(mockRequest(undefined, origin))).toBeTruthy();
            }

            // Test disallowed origins
            for (const origin of disallowedOrigins) {
                expect(RequestService.get().isSafeOrigin(mockRequest(origin, undefined))).toBeFalsy();
                expect(RequestService.get().isSafeOrigin(mockRequest(undefined, origin))).toBeFalsy();
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
                expect(RequestService.get().isSafeOrigin(mockRequest(origin, undefined))).toBeTruthy();
                expect(RequestService.get().isSafeOrigin(mockRequest(undefined, origin))).toBeTruthy();
            }
        });
    });

    describe("getDeviceInfo", () => {
        it("should return correct device info when headers are present", () => {
            const req = {
                headers: {
                    'user-agent': 'Mozilla/5.0',
                    'accept-language': 'en-US,en;q=0.9',
                },
            } as Request;
            const result = RequestService.getDeviceInfo(req);
            expect(result).toBe('User-Agent: Mozilla/5.0; Accept-Language: en-US,en;q=0.9');
        });

        it("should return 'Unknown' for missing user-agent", () => {
            const req = {
                headers: {
                    'accept-language': 'en-US,en;q=0.9',
                },
            } as Request;
            const result = RequestService.getDeviceInfo(req);
            expect(result).toBe('User-Agent: Unknown; Accept-Language: en-US,en;q=0.9');
        });

        it("should return 'Unknown' for missing accept-language", () => {
            const req = {
                headers: {
                    'user-agent': 'Mozilla/5.0',
                },
            } as Request;
            const result = RequestService.getDeviceInfo(req);
            expect(result).toBe('User-Agent: Mozilla/5.0; Accept-Language: Unknown');
        });

        it("should return 'Unknown' for both missing headers", () => {
            const req = {
                headers: {},
            } as Request;
            const result = RequestService.getDeviceInfo(req);
            expect(result).toBe('User-Agent: Unknown; Accept-Language: Unknown');
        });
    });

    describe("parseAcceptLanguage", () => {
        it(`should return ["${DEFAULT_LANGUAGE}"] when accept-language header is missing`, () => {
            const req = { headers: {} };
            expect(RequestService.parseAcceptLanguage(req)).toEqual([DEFAULT_LANGUAGE]);
        });

        it(`should return ["${DEFAULT_LANGUAGE}"] when accept-language header is "*" `, () => {
            const req = { headers: { "accept-language": "*" } };
            expect(RequestService.parseAcceptLanguage(req)).toEqual([DEFAULT_LANGUAGE]);
        });

        it("should parse \"en-US,en;q=0.9,fr;q=0.8\" correctly", () => {
            const req = { headers: { "accept-language": "en-US,en;q=0.9,fr;q=0.8" } };
            expect(RequestService.parseAcceptLanguage(req)).toEqual(["en", "en", "fr"]);
        });

        it("should parse \"fr-CA,fr-FR;q=0.8,en-US;q=0.5,en;q=0.3\" correctly", () => {
            const req = { headers: { "accept-language": "fr-CA,fr-FR;q=0.8,en-US;q=0.5,en;q=0.3" } };
            expect(RequestService.parseAcceptLanguage(req)).toEqual(["fr", "fr", "en", "en"]);
        });

        it("should parse \"es\" correctly", () => {
            const req = { headers: { "accept-language": "es" } };
            expect(RequestService.parseAcceptLanguage(req)).toEqual(["es"]);
        });

        it("should parse \"zh-Hant,zh-Hans;q=0.9,ja;q=0.8\" correctly", () => {
            const req = { headers: { "accept-language": "zh-Hant,zh-Hans;q=0.9,ja;q=0.8" } };
            expect(RequestService.parseAcceptLanguage(req)).toEqual(["zh", "zh", "ja"]);
        });

        it("should parse \"en-GB,en;q=0.8\" correctly", () => {
            const req = { headers: { "accept-language": "en-GB,en;q=0.8" } };
            expect(RequestService.parseAcceptLanguage(req)).toEqual(["en", "en"]);
        });

        it("should handle accept-language with wildcard and language codes", () => {
            const req = { headers: { "accept-language": "*,en;q=0.5" } };
            expect(RequestService.parseAcceptLanguage(req)).toEqual(["*", "en"]);
        });

        it("should handle invalid accept-language values", () => {
            const req = { headers: { "accept-language": "invalid" } };
            expect(RequestService.parseAcceptLanguage(req)).toEqual(["invalid"]);
        });

        it(`should return ["${DEFAULT_LANGUAGE}"] when accept-language header is empty string`, () => {
            const req = { headers: { "accept-language": "" } };
            expect(RequestService.parseAcceptLanguage(req)).toEqual([DEFAULT_LANGUAGE]);
        });

        it("should handle accept-language with extra spaces", () => {
            const req = { headers: { "accept-language": " en-US , en ;q=0.9 " } };
            expect(RequestService.parseAcceptLanguage(req)).toEqual([" en", " en "]);
        });

        it(`should return ["${DEFAULT_LANGUAGE}"] when accept-language header is null`, () => {
            const req = { headers: { "accept-language": null } };
            expect(RequestService.parseAcceptLanguage(req)).toEqual([DEFAULT_LANGUAGE]);
        });

        it(`should return ["${DEFAULT_LANGUAGE}"] when accept-language header is undefined`, () => {
            const req = { headers: { "accept-language": undefined } };
            expect(RequestService.parseAcceptLanguage(req)).toEqual([DEFAULT_LANGUAGE]);
        });

        it("should handle accept-language with uppercase letters", () => {
            const req = { headers: { "accept-language": "EN-US,en;q=0.8" } };
            expect(RequestService.parseAcceptLanguage(req)).toEqual(["EN", "en"]);
        });

        it(`should return ["${DEFAULT_LANGUAGE}"]  when accept-language header is not a string`, () => {
            const req = { headers: { "accept-language": 42 } };
            expect(RequestService.parseAcceptLanguage(req)).toEqual([DEFAULT_LANGUAGE]);
        });
    });

    describe("assertRequestFrom", () => {
        let sessionData: SessionData;
        let userData: SessionUser;

        beforeEach(() => {
            sessionData = {
                isLoggedIn: true,
                apiToken: false,
                fromSafeOrigin: true,
            };
            userData = {
                id: "user123",
                username: "testuser",
            } as unknown as SessionUser;

            jest.spyOn(SessionService, 'getUser').mockReturnValue(userData);
        });

        afterEach(() => {
            jest.restoreAllMocks();
        });

        describe("isApiRoot conditions", () => {
            it("should pass when isApiRoot is true and request is from API root", () => {
                sessionData.isLoggedIn = false;
                sessionData.apiToken = true;
                // @ts-ignore Testing runtime scenario
                jest.spyOn(SessionService, 'getUser').mockReturnValue(undefined);

                const req = { session: sessionData };
                const conditions = { isApiRoot: true };
                const result = RequestService.assertRequestFrom(req, conditions);
                expect(result).toBeUndefined();
            });

            it("should throw MustUseApiToken when isApiRoot is true but request is not from API root", () => {
                sessionData.isLoggedIn = false;
                sessionData.apiToken = false;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(null);

                const req = { session: sessionData };
                const conditions = { isApiRoot: true };
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).toThrow();
            });

            it("should throw MustNotUseApiToken when isApiRoot is false but request is from API root", () => {
                sessionData.isLoggedIn = false;
                sessionData.apiToken = true;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(null);

                const req = { session: sessionData };
                const conditions = { isApiRoot: false };
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).toThrow();
            });

            it("should pass when isApiRoot is false and request is not from API root", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = false;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(userData);

                const req = { session: sessionData };
                const conditions = { isApiRoot: false };
                const result = RequestService.assertRequestFrom(req, conditions);
                expect(result).toBeUndefined();
            });
        });

        describe("isUser conditions", () => {
            it("should return user data when isUser is true and conditions are met (from safe origin)", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = false;
                sessionData.fromSafeOrigin = true;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(userData);

                const req = { session: sessionData };
                const conditions = { isUser: true };
                const result = RequestService.assertRequestFrom(req, conditions);
                expect(result).toEqual(userData);
            });

            it("should return user data when isUser is true and conditions are met (apiToken)", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = true;
                sessionData.fromSafeOrigin = false;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(userData);

                const req = { session: sessionData };
                const conditions = { isUser: true };
                const result = RequestService.assertRequestFrom(req, conditions);
                expect(result).toEqual(userData);
            });

            it("should throw NotLoggedIn when isUser is true and user is not logged in", () => {
                sessionData.isLoggedIn = false;
                sessionData.apiToken = false;
                sessionData.fromSafeOrigin = true;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(null);

                const req = { session: sessionData };
                const conditions = { isUser: true };
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).toThrow();
            });

            it("should throw NotLoggedIn when isUser is true but conditions are not met", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = false;
                sessionData.fromSafeOrigin = false;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(userData);

                const req = { session: sessionData };
                const conditions = { isUser: true };
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).toThrow();
            });

            it("should pass when isUser is false and user is not logged in", () => {
                sessionData.isLoggedIn = false;
                sessionData.apiToken = false;
                sessionData.fromSafeOrigin = false;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(null);

                const req = { session: sessionData };
                const conditions = { isUser: false };
                const result = RequestService.assertRequestFrom(req, conditions);
                expect(result).toBeUndefined();
            });

            it("should throw NotLoggedIn when isUser is false but user is logged in", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = true;
                sessionData.fromSafeOrigin = true;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(userData);

                const req = { session: sessionData };
                const conditions = { isUser: false };
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).toThrow();
            });
        });

        describe("isOfficialUser conditions", () => {
            it("should return user data when isOfficialUser is true and conditions are met", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = false;
                sessionData.fromSafeOrigin = true;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(userData);

                const req = { session: sessionData };
                const conditions = { isOfficialUser: true };
                const result = RequestService.assertRequestFrom(req, conditions);
                expect(result).toEqual(userData);
            });

            it("should throw NotLoggedInOfficial when isOfficialUser is true but user is not logged in", () => {
                sessionData.isLoggedIn = false;
                sessionData.apiToken = false;
                sessionData.fromSafeOrigin = true;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(null);

                const req = { session: sessionData };
                const conditions = { isOfficialUser: true };
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).toThrow();
            });

            it("should throw NotLoggedInOfficial when isOfficialUser is true but apiToken is true", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = true;
                sessionData.fromSafeOrigin = true;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(userData);

                const req = { session: sessionData };
                const conditions = { isOfficialUser: true };
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).toThrow();
            });

            it("should throw NotLoggedInOfficial when isOfficialUser is true but fromSafeOrigin is false", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = false;
                sessionData.fromSafeOrigin = false;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(userData);

                const req = { session: sessionData };
                const conditions = { isOfficialUser: true };
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).toThrow();
            });

            it("should pass when isOfficialUser is false and user is not an official user", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = true;
                sessionData.fromSafeOrigin = false;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(userData);

                const req = { session: sessionData };
                const conditions = { isOfficialUser: false };
                const result = RequestService.assertRequestFrom(req, conditions);
                expect(result).toBeUndefined();
            });

            it("should throw NotLoggedInOfficial when isOfficialUser is false but user is an official user", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = false;
                sessionData.fromSafeOrigin = true;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(userData);

                const req = { session: sessionData };
                const conditions = { isOfficialUser: false };
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).toThrow();
            });
        });

        describe("Combination of conditions", () => {
            it("should pass when all conditions are met", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = false;
                sessionData.fromSafeOrigin = true;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(userData);

                const req = { session: sessionData };
                const conditions = { isUser: true, isOfficialUser: true, isApiRoot: false };
                const result = RequestService.assertRequestFrom(req, conditions);
                expect(result).toEqual(userData);
            });

            it("should throw appropriate error when one of the conditions fails", () => {
                sessionData.isLoggedIn = true;
                sessionData.apiToken = true;
                sessionData.fromSafeOrigin = true;
                jest.spyOn(SessionService, 'getUser').mockReturnValue(userData);

                const req = { session: sessionData };
                const conditions = { isUser: true, isOfficialUser: true, isApiRoot: false };
                expect(() => {
                    RequestService.assertRequestFrom(req, conditions);
                }).toThrow();
            });
        });

        it("should return undefined when no conditions are specified", () => {
            const req = { session: sessionData };
            const conditions = {};
            const result = RequestService.assertRequestFrom(req, conditions);
            expect(result).toBeUndefined();
        });
    });
});
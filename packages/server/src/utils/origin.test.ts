import { Request } from "express";
import { isValidDomain, isValidIP } from "./origin";

describe("Origin Validation Tests", () => {
    describe("isValidIP", () => {
        it("should validate IPv4 addresses", () => {
            expect(isValidIP("192.168.1.1")).toBeTruthy();
            expect(isValidIP("255.255.255.255")).toBeTruthy();
            expect(isValidIP("0.0.0.0")).toBeTruthy();
            expect(isValidIP("999.999.999.999")).toBeFalsy();
        });

        it("should validate IPv6 addresses", () => {
            expect(isValidIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334")).toBeTruthy();
            expect(isValidIP("::1")).toBeTruthy();
            expect(isValidIP("::ffff:c0a8:101")).toBeTruthy();
            expect(isValidIP("gibberish")).toBeFalsy();
        });
    });

    describe("isValidDomain", () => {
        it("should validate domain names", () => {
            expect(isValidDomain("example.com")).toBeTruthy();
            expect(isValidDomain("subdomain.example.com")).toBeTruthy();
            expect(isValidDomain("www.example.co.uk")).toBeTruthy();
            expect(isValidDomain("example")).toBeFalsy();
            expect(isValidDomain("example..com")).toBeFalsy();
            expect(isValidDomain(".com")).toBeFalsy();
            expect(isValidDomain("com.")).toBeFalsy();
            expect(isValidDomain("example.com/")).toBeFalsy();
        });
    });

    describe("isSafeOrigin", () => {
        let isSafeOrigin: (req: Request) => boolean;

        // Dynamically import module
        const importOriginModule = async () => {
            // Reset the module registry to clear the cached state
            jest.resetModules();
            const module = await import("./origin");
            return module.isSafeOrigin;
        };

        beforeEach(async () => {
            // Import the module before each test
            isSafeOrigin = await importOriginModule();
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
                expect(isSafeOrigin(mockRequest(origin, undefined))).toBeTruthy();
                expect(isSafeOrigin(mockRequest(undefined, origin))).toBeTruthy();
            }

            // Test disallowed origins
            for (const origin of disallowedOrigins) {
                expect(isSafeOrigin(mockRequest(origin, undefined))).toBeFalsy();
                expect(isSafeOrigin(mockRequest(undefined, origin))).toBeFalsy();
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
                expect(isSafeOrigin(mockRequest(origin, undefined))).toBeTruthy();
                expect(isSafeOrigin(mockRequest(undefined, origin))).toBeTruthy();
            }

            // Test disallowed origins
            for (const origin of disallowedOrigins) {
                expect(isSafeOrigin(mockRequest(origin, undefined))).toBeFalsy();
                expect(isSafeOrigin(mockRequest(undefined, origin))).toBeFalsy();
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
                expect(isSafeOrigin(mockRequest(origin, undefined))).toBeTruthy();
                expect(isSafeOrigin(mockRequest(undefined, origin))).toBeTruthy();
            }
        });
    });
});

/**
 * Comprehensive test suite for HTTPClient service
 * 
 * Tests security features, authentication methods, SSRF protection,
 * different profiles, and error handling scenarios.
 */

import { describe, test, beforeEach, expect, vi } from "vitest";
import { HTTPClient, type APICallOptions, type AuthConfig } from "../../../services/http/httpClient.js";
import { fetchWrapper } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";

// Mock dependencies
vi.mock("@vrooli/shared", async () => {
    const actual = await vi.importActual("@vrooli/shared");
    return {
        ...actual,
        fetchWrapper: vi.fn(),
    };
});

vi.mock("../../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

const mockFetchWrapper = fetchWrapper as any;
const mockLogger = logger as any;

describe("HTTPClient", () => {
    let httpClient: HTTPClient;

    beforeEach(() => {
        vi.clearAllMocks();
        httpClient = new HTTPClient();
    });

    describe("Constructor and Profile Management", () => {
        test("should use external profile by default", () => {
            const client = new HTTPClient();
            expect(client["profile"].name).toBe("external");
        });

        test("should accept profile parameter", () => {
            const client = new HTTPClient("resource");
            expect(client["profile"].name).toBe("resource");
        });

        test("should fallback to external profile for invalid profile", () => {
            const client = new HTTPClient("invalid" as any);
            expect(client["profile"].name).toBe("external");
        });

        test("should have correct profile configurations", () => {
            const externalProfile = HTTPClient.PROFILES.external;
            expect(externalProfile.allowLocalhost).toBe(false);
            expect(externalProfile.allowPrivateIPs).toBe(false);
            expect(externalProfile.requireSSL).toBe(false);

            const resourceProfile = HTTPClient.PROFILES.resource;
            expect(resourceProfile.allowLocalhost).toBe(true);
            expect(resourceProfile.allowPrivateIPs).toBe(true);
            expect(resourceProfile.blockedDomains).toEqual([]);

            const internalProfile = HTTPClient.PROFILES.internal;
            expect(internalProfile.allowLocalhost).toBe(true);
            expect(internalProfile.allowPrivateIPs).toBe(true);
            expect(internalProfile.blockedDomains).toEqual(["0.0.0.0"]);
        });
    });

    describe("Factory Methods", () => {
        test("forExternalAPIs should create external profile client", () => {
            const client = HTTPClient.forExternalAPIs();
            expect(client["profile"].name).toBe("external");
        });

        test("forResources should create resource profile client", () => {
            const client = HTTPClient.forResources();
            expect(client["profile"].name).toBe("resource");
        });

        test("forInternal should create internal profile client", () => {
            const client = HTTPClient.forInternal();
            expect(client["profile"].name).toBe("internal");
        });
    });

    describe("URL Validation", () => {
        describe("External Profile", () => {
            beforeEach(() => {
                httpClient = new HTTPClient("external");
            });

            test("should allow valid HTTPS URLs", () => {
                expect(() => httpClient["validateUrl"]("https://api.example.com/data")).not.toThrow();
            });

            test("should allow valid HTTP URLs", () => {
                expect(() => httpClient["validateUrl"]("http://api.example.com/data")).not.toThrow();
            });

            test("should reject localhost", () => {
                expect(() => httpClient["validateUrl"]("http://localhost:3000")).toThrow("Localhost access not allowed");
            });

            test("should reject 127.0.0.1", () => {
                expect(() => httpClient["validateUrl"]("http://127.0.0.1:3000")).toThrow("Localhost access not allowed");
            });

            test("should reject private IP ranges", () => {
                expect(() => httpClient["validateUrl"]("http://192.168.1.1")).toThrow("Private IP access not allowed");
                expect(() => httpClient["validateUrl"]("http://10.0.0.1")).toThrow("Private IP access not allowed");
                expect(() => httpClient["validateUrl"]("http://172.16.0.1")).toThrow("Private IP access not allowed");
            });

            test("should reject invalid protocols", () => {
                expect(() => httpClient["validateUrl"]("ftp://example.com")).toThrow("Invalid protocol");
                expect(() => httpClient["validateUrl"]("file:///etc/passwd")).toThrow("Invalid protocol");
            });

            test("should reject malformed URLs", () => {
                expect(() => httpClient["validateUrl"]("not-a-url")).toThrow("Invalid URL");
                expect(() => httpClient["validateUrl"]("")).toThrow("Invalid URL");
            });
        });

        describe("Resource Profile", () => {
            beforeEach(() => {
                httpClient = new HTTPClient("resource");
            });

            test("should allow localhost", () => {
                expect(() => httpClient["validateUrl"]("http://localhost:3000")).not.toThrow();
            });

            test("should allow private IPs", () => {
                expect(() => httpClient["validateUrl"]("http://192.168.1.1")).not.toThrow();
                expect(() => httpClient["validateUrl"]("http://10.0.0.1")).not.toThrow();
            });
        });

        describe("Internal Profile", () => {
            beforeEach(() => {
                httpClient = new HTTPClient("internal");
            });

            test("should allow localhost and private IPs", () => {
                expect(() => httpClient["validateUrl"]("http://localhost:3000")).not.toThrow();
                expect(() => httpClient["validateUrl"]("http://192.168.1.1")).not.toThrow();
            });

            test("should reject 0.0.0.0", () => {
                expect(() => httpClient["validateUrl"]("http://0.0.0.0:3000")).toThrow("Blocked domain/IP");
            });
        });

        test("should warn about unusual ports", () => {
            httpClient["validateUrl"]("http://example.com:9999");
            expect(mockLogger.warn).toHaveBeenCalledWith(
                "[HTTPClient] Unusual port detected",
                expect.objectContaining({
                    port: "9999",
                }),
            );
        });
    });

    describe("Header Sanitization", () => {
        test("should remove forbidden headers based on profile", () => {
            const headers = {
                "Content-Type": "application/json",
                "Host": "example.com",
                "User-Agent": "test-agent",
                "Content-Length": "100",
            };

            const sanitized = httpClient["sanitizeHeaders"](headers);
            expect(sanitized).toEqual({
                "Content-Type": "application/json",
                "User-Agent": "test-agent",
            });
        });

        test("should sanitize header values by removing control characters", () => {
            const headers = {
                "X-Custom": "value\x00with\x1Fcontrol\x7Fchars",
                "X-Normal": "normal value",
            };

            const sanitized = httpClient["sanitizeHeaders"](headers);
            expect(sanitized["X-Custom"]).toBe("valuewithcontrolchars");
            expect(sanitized["X-Normal"]).toBe("normal value");
        });

        test("should remove headers with empty values after sanitization", () => {
            const headers = {
                "X-Empty": "\x00\x1F\x7F",
                "X-Valid": "valid",
            };

            const sanitized = httpClient["sanitizeHeaders"](headers);
            expect(sanitized).toEqual({
                "X-Valid": "valid",
            });
        });

        test("should log warnings for blocked headers", () => {
            const headers = { "Host": "example.com" };
            httpClient["sanitizeHeaders"](headers);

            expect(mockLogger.warn).toHaveBeenCalledWith(
                "[HTTPClient] Blocked forbidden header",
                expect.objectContaining({
                    header: "Host",
                    profile: "external",
                }),
            );
        });
    });

    describe("Authentication", () => {
        let requestInit: RequestInit;

        beforeEach(() => {
            requestInit = {
                method: "GET",
                headers: new Headers(),
            };
        });

        test("should apply Bearer authentication", () => {
            const auth: AuthConfig = {
                type: "bearer",
                token: "test-token",
            };

            httpClient["applyAuthentication"](requestInit, auth);
            const headers = new Headers(requestInit.headers);
            expect(headers.get("Authorization")).toBe("Bearer test-token");
        });

        test("should apply API key authentication with default header", () => {
            const auth: AuthConfig = {
                type: "apikey",
                token: "api-key-value",
            };

            httpClient["applyAuthentication"](requestInit, auth);
            const headers = new Headers(requestInit.headers);
            expect(headers.get("X-API-Key")).toBe("api-key-value");
        });

        test("should apply API key authentication with custom header", () => {
            const auth: AuthConfig = {
                type: "apikey",
                token: "api-key-value",
                headerName: "X-Custom-API-Key",
            };

            httpClient["applyAuthentication"](requestInit, auth);
            const headers = new Headers(requestInit.headers);
            expect(headers.get("X-Custom-API-Key")).toBe("api-key-value");
        });

        test("should apply Basic authentication", () => {
            const auth: AuthConfig = {
                type: "basic",
                username: "user",
                password: "pass",
            };

            httpClient["applyAuthentication"](requestInit, auth);
            const headers = new Headers(requestInit.headers);
            const expectedCredentials = Buffer.from("user:pass").toString("base64");
            expect(headers.get("Authorization")).toBe(`Basic ${expectedCredentials}`);
        });

        test("should apply custom authentication", () => {
            const auth: AuthConfig = {
                type: "custom",
                headerName: "X-Custom-Auth",
                value: "custom-value",
            };

            httpClient["applyAuthentication"](requestInit, auth);
            const headers = new Headers(requestInit.headers);
            expect(headers.get("X-Custom-Auth")).toBe("custom-value");
        });

        test("should warn for unknown auth type", () => {
            const auth = { type: "unknown" } as any;
            httpClient["applyAuthentication"](requestInit, auth);
            expect(mockLogger.warn).toHaveBeenCalledWith(
                "[HTTPClient] Unknown auth type",
                { authType: "unknown" },
            );
        });

        test("should skip auth if required fields are missing", () => {
            const bearerAuth: AuthConfig = { type: "bearer" }; // Missing token
            httpClient["applyAuthentication"](requestInit, bearerAuth);
            const headers = new Headers(requestInit.headers);
            expect(headers.get("Authorization")).toBeNull();

            const basicAuth: AuthConfig = { type: "basic", username: "user" }; // Missing password
            httpClient["applyAuthentication"](requestInit, basicAuth);
            expect(headers.get("Authorization")).toBeNull();
        });
    });

    describe("Body Validation", () => {
        test("should allow null/undefined body", () => {
            expect(() => httpClient["validateBody"](null)).not.toThrow();
            expect(() => httpClient["validateBody"](undefined)).not.toThrow();
        });

        test("should allow valid string body", () => {
            expect(() => httpClient["validateBody"]("test body")).not.toThrow();
        });

        test("should allow valid object body", () => {
            expect(() => httpClient["validateBody"]({ key: "value" })).not.toThrow();
        });

        test("should reject body exceeding size limit", () => {
            const largeBody = "x".repeat(11 * 1024 * 1024); // 11MB
            expect(() => httpClient["validateBody"](largeBody)).toThrow("Request body too large");
        });

        test("should calculate size correctly for object bodies", () => {
            const largeObject = { data: "x".repeat(11 * 1024 * 1024) };
            expect(() => httpClient["validateBody"](largeObject)).toThrow("Request body too large");
        });
    });

    describe("Request Execution", () => {
        const mockResponse = {
            ok: true,
            status: 200,
            statusText: "OK",
            url: "https://api.example.com/data",
            headers: new Map([
                ["content-type", "application/json"],
                ["x-custom", "value"],
            ]),
            json: vi.fn().mockResolvedValue({ data: "response" }),
            text: vi.fn().mockResolvedValue("text response"),
        };

        beforeEach(() => {
            mockFetchWrapper.mockResolvedValue(mockResponse as any);
        });

        test("should make successful GET request", async () => {
            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "GET",
            };

            const result = await httpClient.makeRequest(options);

            expect(result.success).toBe(true);
            expect(result.status).toBe(200);
            expect(result.data).toEqual({ data: "response" });
            expect(result.metadata.url).toBe(options.url);
            expect(result.metadata.method).toBe("GET");
        });

        test("should handle JSON response", async () => {
            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "GET",
            };

            await httpClient.makeRequest(options);
            expect(mockResponse.json).toHaveBeenCalled();
        });

        test("should handle text response when JSON parsing fails", async () => {
            mockResponse.json.mockRejectedValueOnce(new Error("Invalid JSON"));
            
            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "GET",
            };

            const result = await httpClient.makeRequest(options);
            expect(mockResponse.text).toHaveBeenCalled();
            expect(result.data).toBe("text response");
        });

        test("should handle non-JSON response", async () => {
            const textResponse = {
                ...mockResponse,
                headers: new Map([["content-type", "text/plain"]]),
            };
            mockFetchWrapper.mockResolvedValue(textResponse as any);

            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "GET",
            };

            await httpClient.makeRequest(options);
            expect(mockResponse.text).toHaveBeenCalled();
        });

        test("should include request body for non-GET methods", async () => {
            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "POST",
                body: { key: "value" },
            };

            await httpClient.makeRequest(options);

            expect(mockFetchWrapper).toHaveBeenCalledWith(
                options.url,
                expect.objectContaining({
                    method: "POST",
                    body: JSON.stringify(options.body),
                }),
                expect.any(Object),
            );
        });

        test("should not include body for GET requests", async () => {
            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "GET",
                body: { key: "value" },
            };

            await httpClient.makeRequest(options);

            expect(mockFetchWrapper).toHaveBeenCalledWith(
                options.url,
                expect.not.objectContaining({
                    body: expect.anything(),
                }),
                expect.any(Object),
            );
        });

        test("should handle string body", async () => {
            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "POST",
                body: "raw string body",
            };

            await httpClient.makeRequest(options);

            expect(mockFetchWrapper).toHaveBeenCalledWith(
                options.url,
                expect.objectContaining({
                    body: "raw string body",
                }),
                expect.any(Object),
            );
        });

        test("should apply authentication", async () => {
            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "GET",
                auth: {
                    type: "bearer",
                    token: "test-token",
                },
            };

            await httpClient.makeRequest(options);

            expect(mockFetchWrapper).toHaveBeenCalledWith(
                options.url,
                expect.objectContaining({
                    headers: expect.any(Headers),
                }),
                expect.any(Object),
            );
        });

        test("should use profile defaults for timeout and retries", async () => {
            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "GET",
            };

            await httpClient.makeRequest(options);

            expect(mockFetchWrapper).toHaveBeenCalledWith(
                options.url,
                expect.any(Object),
                expect.objectContaining({
                    timeout: 30000, // External profile default
                    maxRetries: 3,
                }),
            );
        });

        test("should override defaults with provided values", async () => {
            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "GET",
                timeout: 5000,
                retries: 1,
            };

            await httpClient.makeRequest(options);

            expect(mockFetchWrapper).toHaveBeenCalledWith(
                options.url,
                expect.any(Object),
                expect.objectContaining({
                    timeout: 5000,
                    maxRetries: 1,
                }),
            );
        });

        test("should extract response headers correctly", async () => {
            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "GET",
            };

            const result = await httpClient.makeRequest(options);

            expect(result.headers).toEqual({
                "content-type": "application/json",
                "x-custom": "value",
            });
        });

        test("should log request and response", async () => {
            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "GET",
                auth: { type: "bearer", token: "test" },
            };

            await httpClient.makeRequest(options);

            expect(mockLogger.info).toHaveBeenCalledWith(
                "[HTTPClient] Making API request",
                expect.objectContaining({
                    url: options.url,
                    method: "GET",
                    profile: "external",
                    hasAuth: true,
                    authType: "bearer",
                }),
            );

            expect(mockLogger.info).toHaveBeenCalledWith(
                "[HTTPClient] API request completed",
                expect.objectContaining({
                    url: options.url,
                    status: 200,
                    success: true,
                }),
            );
        });
    });

    describe("Error Handling", () => {
        test("should handle network errors", async () => {
            mockFetchWrapper.mockRejectedValue(new Error("Network error"));

            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "GET",
            };

            const result = await httpClient.makeRequest(options);

            expect(result.success).toBe(false);
            expect(result.status).toBe(0);
            expect(result.statusText).toBe("Network Error");
            expect(result.error).toBe("Network error");
            expect(result.data).toBeNull();
        });

        test("should handle HTTP error responses", async () => {
            const errorResponse = {
                ok: false,
                status: 404,
                statusText: "Not Found",
                url: "https://api.example.com/data",
                headers: new Map(),
                text: vi.fn().mockResolvedValue("Not found"),
            };
            mockFetchWrapper.mockResolvedValue(errorResponse as any);

            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "GET",
            };

            const result = await httpClient.makeRequest(options);

            expect(result.success).toBe(false);
            expect(result.status).toBe(404);
            expect(result.error).toBe("HTTP 404: Not Found");
        });

        test("should handle URL validation errors", async () => {
            const options: APICallOptions = {
                url: "invalid-url",
                method: "GET",
            };

            await expect(httpClient.makeRequest(options)).rejects.toThrow("Invalid URL");
        });

        test("should handle body validation errors", async () => {
            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "POST",
                body: "x".repeat(11 * 1024 * 1024), // Too large
            };

            await expect(httpClient.makeRequest(options)).rejects.toThrow("Request body too large");
        });

        test("should log errors", async () => {
            mockFetchWrapper.mockRejectedValue(new Error("Test error"));

            const options: APICallOptions = {
                url: "https://api.example.com/data",
                method: "GET",
            };

            await httpClient.makeRequest(options);

            expect(mockLogger.error).toHaveBeenCalledWith(
                "[HTTPClient] API request failed",
                expect.objectContaining({
                    url: options.url,
                    error: "Test error",
                }),
            );
        });
    });

    describe("Profile-Specific Behavior", () => {
        test("external profile should have strict security", () => {
            const client = HTTPClient.forExternalAPIs();
            expect(() => client["validateUrl"]("http://localhost:3000")).toThrow();
            expect(() => client["validateUrl"]("http://192.168.1.1")).toThrow();
        });

        test("resource profile should allow local access", () => {
            const client = HTTPClient.forResources();
            expect(() => client["validateUrl"]("http://localhost:3000")).not.toThrow();
            expect(() => client["validateUrl"]("http://192.168.1.1")).not.toThrow();
        });

        test("internal profile should allow private IPs but block 0.0.0.0", () => {
            const client = HTTPClient.forInternal();
            expect(() => client["validateUrl"]("http://192.168.1.1")).not.toThrow();
            expect(() => client["validateUrl"]("http://0.0.0.0:3000")).toThrow();
        });

        test("profiles should have different timeout defaults", () => {
            expect(HTTPClient.PROFILES.external.defaultTimeout).toBe(30000);
            expect(HTTPClient.PROFILES.resource.defaultTimeout).toBe(5000);
            expect(HTTPClient.PROFILES.internal.defaultTimeout).toBe(10000);
        });

        test("profiles should have different retry defaults", () => {
            expect(HTTPClient.PROFILES.external.defaultRetries).toBe(3);
            expect(HTTPClient.PROFILES.resource.defaultRetries).toBe(5);
            expect(HTTPClient.PROFILES.internal.defaultRetries).toBe(2);
        });
    });

    describe("Private IP Detection", () => {
        test("should correctly identify private IP ranges", () => {
            const client = new HTTPClient("external");
            
            expect(client["isPrivateIP"]("10.0.0.1")).toBe(true);
            expect(client["isPrivateIP"]("172.16.0.1")).toBe(true);
            expect(client["isPrivateIP"]("172.31.255.255")).toBe(true);
            expect(client["isPrivateIP"]("192.168.1.1")).toBe(true);
            expect(client["isPrivateIP"]("169.254.1.1")).toBe(true);
            
            expect(client["isPrivateIP"]("8.8.8.8")).toBe(false);
            expect(client["isPrivateIP"]("172.15.0.1")).toBe(false);
            expect(client["isPrivateIP"]("172.32.0.1")).toBe(false);
            expect(client["isPrivateIP"]("example.com")).toBe(false);
        });
    });
});

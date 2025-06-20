import { describe, expect, it, vi } from "vitest";
import { ApiVersionConfig, type ApiVersionConfigObject } from "./api.js";
import { apiConfigFixtures } from "../../__test/fixtures/config/apiConfigFixtures.js";

describe("ApiVersionConfig", () => {
    const mockLogger = {
        log: vi.fn(),
        info: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    };

    describe("constructor", () => {
        it("should create config with provided values", () => {
            const configObj = apiConfigFixtures.complete;

            const config = new ApiVersionConfig({ config: configObj });

            expect(config.rateLimiting).toEqual(configObj.rateLimiting);
            expect(config.authentication).toEqual(configObj.authentication);
            expect(config.caching).toEqual(configObj.caching);
            expect(config.timeout).toEqual(configObj.timeout);
            expect(config.retry).toEqual(configObj.retry);
            expect(config.documentationLink).toBe(configObj.documentationLink);
            expect(config.schema).toEqual(configObj.schema);
            expect(config.callLink).toBe(configObj.callLink);
        });

        it("should handle minimal config", () => {
            const configObj = apiConfigFixtures.minimal;

            const config = new ApiVersionConfig({ config: configObj });

            expect(config.rateLimiting).toBeUndefined();
            expect(config.authentication).toBeUndefined();
            expect(config.caching).toBeUndefined();
            expect(config.timeout).toBeUndefined();
            expect(config.retry).toBeUndefined();
            expect(config.documentationLink).toBeUndefined();
            expect(config.schema).toBeUndefined();
            expect(config.callLink).toBeUndefined();
        });
    });

    describe("static methods", () => {
        describe("default", () => {
            it("should create config with default values", () => {
                const config = ApiVersionConfig.default();

                expect(config.rateLimiting).toEqual({
                    requestsPerMinute: 1000,
                    burstLimit: 100,
                    useGlobalRateLimit: true,
                });
                expect(config.authentication).toEqual({
                    type: "none",
                });
                expect(config.caching).toEqual({
                    enabled: true,
                    ttl: 3600,
                    invalidation: "ttl",
                });
                expect(config.timeout).toEqual({
                    request: 10000,
                    connection: 10000,
                });
                expect(config.retry).toEqual({
                    maxAttempts: 3,
                    backoffStrategy: "exponential",
                    initialDelay: 1000,
                });
            });
        });

        describe("parse", () => {
            it("should parse valid config with fallbacks enabled", () => {
                const version = {
                    config: apiConfigFixtures.variants.oauth2ProtectedApi,
                };

                const config = ApiVersionConfig.parse(version, mockLogger, { useFallbacks: true });

                expect(config.authentication?.type).toBe("oauth2");
                // Should have default values for other properties
                expect(config.rateLimiting).toEqual(ApiVersionConfig.defaultRateLimiting());
                expect(config.caching).toEqual(ApiVersionConfig.defaultCaching());
                expect(config.timeout).toEqual(ApiVersionConfig.defaultTimeout());
                expect(config.retry).toEqual(ApiVersionConfig.defaultRetry());
            });

            it("should parse config without fallbacks", () => {
                const version = {
                    config: apiConfigFixtures.variants.oauth2ProtectedApi,
                };

                const config = ApiVersionConfig.parse(version, mockLogger, { useFallbacks: false });

                expect(config.authentication?.type).toBe("oauth2");
                // Should not have default values for other properties
                expect(config.rateLimiting).toBeUndefined();
                expect(config.caching).toBeUndefined();
                expect(config.timeout).toBeUndefined();
                expect(config.retry).toBeUndefined();
            });

            it("should parse config with default useFallbacks (true)", () => {
                const version = {
                    config: apiConfigFixtures.minimal,
                };

                const config = ApiVersionConfig.parse(version, mockLogger);

                // Should have default values since useFallbacks defaults to true
                expect(config.rateLimiting).toEqual(ApiVersionConfig.defaultRateLimiting());
                expect(config.authentication).toEqual(ApiVersionConfig.defaultAuthentication());
                expect(config.caching).toEqual(ApiVersionConfig.defaultCaching());
                expect(config.timeout).toEqual(ApiVersionConfig.defaultTimeout());
                expect(config.retry).toEqual(ApiVersionConfig.defaultRetry());
            });
        });

        describe("default value generators", () => {
            it("should provide correct default rate limiting", () => {
                const rateLimiting = ApiVersionConfig.defaultRateLimiting();
                
                expect(rateLimiting).toEqual({
                    requestsPerMinute: 1000,
                    burstLimit: 100,
                    useGlobalRateLimit: true,
                });
            });

            it("should provide correct default authentication", () => {
                const authentication = ApiVersionConfig.defaultAuthentication();
                
                expect(authentication).toEqual({
                    type: "none",
                });
            });

            it("should provide correct default caching", () => {
                const caching = ApiVersionConfig.defaultCaching();
                
                expect(caching).toEqual({
                    enabled: true,
                    ttl: 3600,
                    invalidation: "ttl",
                });
            });

            it("should provide correct default timeout", () => {
                const timeout = ApiVersionConfig.defaultTimeout();
                
                expect(timeout).toEqual({
                    request: 10000,
                    connection: 10000,
                });
            });

            it("should provide correct default retry", () => {
                const retry = ApiVersionConfig.defaultRetry();
                
                expect(retry).toEqual({
                    maxAttempts: 3,
                    backoffStrategy: "exponential",
                    initialDelay: 1000,
                });
            });
        });
    });

    describe("setter methods", () => {
        let config: ApiVersionConfig;

        beforeEach(() => {
            config = ApiVersionConfig.default();
        });

        it("should set rate limiting", () => {
            const newRateLimiting = {
                requestsPerMinute: 500,
                burstLimit: 50,
                useGlobalRateLimit: false,
            };

            config.setRateLimiting(newRateLimiting);
            expect(config.rateLimiting).toEqual(newRateLimiting);
        });

        it("should set authentication", () => {
            const newAuth = {
                type: "bearer",
                location: "header",
                parameterName: "Authorization",
                settings: { tokenPrefix: "Bearer " },
            };

            config.setAuthentication(newAuth);
            expect(config.authentication).toEqual(newAuth);
        });

        it("should set caching", () => {
            const newCaching = {
                enabled: false,
                ttl: 1800,
                invalidation: "manual",
            };

            config.setCaching(newCaching);
            expect(config.caching).toEqual(newCaching);
        });

        it("should set timeout", () => {
            const newTimeout = {
                request: 15000,
                connection: 5000,
            };

            config.setTimeout(newTimeout);
            expect(config.timeout).toEqual(newTimeout);
        });

        it("should set retry", () => {
            const newRetry = {
                maxAttempts: 5,
                backoffStrategy: "linear",
                initialDelay: 2000,
            };

            config.setRetry(newRetry);
            expect(config.retry).toEqual(newRetry);
        });

        it("should allow setting undefined values", () => {
            config.setRateLimiting(undefined);
            config.setAuthentication(undefined);
            config.setCaching(undefined);
            config.setTimeout(undefined);
            config.setRetry(undefined);

            expect(config.rateLimiting).toBeUndefined();
            expect(config.authentication).toBeUndefined();
            expect(config.caching).toBeUndefined();
            expect(config.timeout).toBeUndefined();
            expect(config.retry).toBeUndefined();
        });
    });

    describe("export", () => {
        it("should export all config properties", () => {
            const originalConfig = apiConfigFixtures.complete;

            const config = new ApiVersionConfig({ config: originalConfig });
            const exported = config.export();

            expect(exported.rateLimiting).toEqual(originalConfig.rateLimiting);
            expect(exported.authentication).toEqual(originalConfig.authentication);
            expect(exported.caching).toEqual(originalConfig.caching);
            expect(exported.timeout).toEqual(originalConfig.timeout);
            expect(exported.retry).toEqual(originalConfig.retry);
            // Should also include base properties
            expect(exported.__version).toBe("1.0");
            expect(exported.resources).toEqual(originalConfig.resources);
        });

        it("should export undefined values correctly", () => {
            const minimalConfig = apiConfigFixtures.minimal;

            const config = new ApiVersionConfig({ config: minimalConfig });
            const exported = config.export();

            expect(exported.rateLimiting).toBeUndefined();
            expect(exported.authentication).toBeUndefined();
            expect(exported.caching).toBeUndefined();
            expect(exported.timeout).toBeUndefined();
            expect(exported.retry).toBeUndefined();
        });
    });

    describe("complex scenarios", () => {
        it("should handle complete API configuration", () => {
            const completeConfig = apiConfigFixtures.variants.oauth2ProtectedApi;

            const config = new ApiVersionConfig({ config: completeConfig });

            // Test that all properties are accessible
            expect(config.rateLimiting).toEqual(completeConfig.rateLimiting);
            expect(config.authentication).toEqual(completeConfig.authentication);
            expect(config.caching).toEqual(completeConfig.caching);
            expect(config.timeout).toEqual(completeConfig.timeout);
            expect(config.retry).toEqual(completeConfig.retry);
            expect(config.documentationLink).toBe(completeConfig.documentationLink);
            expect(config.schema).toEqual(completeConfig.schema);
            expect(config.callLink).toBe(completeConfig.callLink);
        });

        it("should maintain consistency through parse and export cycle", () => {
            const originalConfig = apiConfigFixtures.variants.basicAuthApi;

            const version = { config: originalConfig };
            const parsed = ApiVersionConfig.parse(version, mockLogger, { useFallbacks: false });
            const exported = parsed.export();

            expect(exported.authentication).toEqual(originalConfig.authentication);
            expect(exported.rateLimiting).toEqual(originalConfig.rateLimiting);
        });
    });
});
/**
 * Tests for ResourceProvider base class functionality
 */

import { ResourceProvider } from "../ResourceProvider.js";
import { CircuitBreaker } from "../circuitBreaker.js";
import { ResourceCategory, DeploymentType, ResourceHealth, DiscoveryStatus, ResourceEvent } from "../types.js";
import type { HealthCheckResult, ResourceInitOptions } from "../types.js";
import type { AIMetadata } from "../typeRegistry.js";

// Mock HTTPClient
jest.mock("../http/httpClient.js", () => ({
    HTTPClient: {
        forResources: jest.fn(() => ({
            makeRequest: jest.fn(),
        })),
    },
}));

// Mock logger
jest.mock("../../events/logger.js", () => ({
    logger: {
        debug: jest.fn(),
        info: jest.fn(),
        warn: jest.fn(),
        error: jest.fn(),
    },
}));

// Test implementation of ResourceProvider
class TestResourceProvider extends ResourceProvider<"ollama"> {
    readonly id = "ollama";
    readonly category = ResourceCategory.AI;
    readonly displayName = "Test Ollama";
    readonly description = "Test Ollama resource";
    readonly isSupported = true;
    readonly deploymentType = DeploymentType.Local;

    private shouldDiscoverySucceed = true;
    private shouldHealthCheckSucceed = true;
    private discoveryCallCount = 0;
    private healthCheckCallCount = 0;

    // Method to control test behavior
    setDiscoveryResult(succeed: boolean) {
        this.shouldDiscoverySucceed = succeed;
    }

    setHealthCheckResult(succeed: boolean) {
        this.shouldHealthCheckSucceed = succeed;
    }

    getDiscoveryCallCount() {
        return this.discoveryCallCount;
    }

    getHealthCheckCallCount() {
        return this.healthCheckCallCount;
    }

    protected async performDiscovery(): Promise<boolean> {
        this.discoveryCallCount++;
        if (!this.shouldDiscoverySucceed) {
            throw new Error("Discovery failed");
        }
        return this.shouldDiscoverySucceed;
    }

    protected async performHealthCheck(): Promise<HealthCheckResult> {
        this.healthCheckCallCount++;
        if (!this.shouldHealthCheckSucceed) {
            throw new Error("Health check failed");
        }
        return {
            healthy: this.shouldHealthCheckSucceed,
            message: this.shouldHealthCheckSucceed ? "Healthy" : "Unhealthy",
            timestamp: new Date(),
        };
    }

    protected getMetadata(): AIMetadata {
        return {
            version: "1.0.0",
            capabilities: ["test"],
            supportedModels: ["test-model"],
            lastUpdated: new Date(),
        };
    }

    // Expose protected methods for testing
    public getCircuitBreakerStats() {
        return {
            discovery: this.discoveryCircuitBreaker?.getStats(),
            healthCheck: this.healthCheckCircuitBreaker?.getStats(),
        };
    }

    public async restartHealthMonitoringPublic() {
        return this.restartHealthMonitoring();
    }
}

describe("ResourceProvider", () => {
    let provider: TestResourceProvider;
    let mockConfig: any;

    beforeEach(() => {
        provider = new TestResourceProvider();
        mockConfig = {
            enabled: true,
            baseUrl: "http://localhost:11434",
            healthCheck: {
                intervalMs: 30000,
                timeoutMs: 5000,
            },
        };
        jest.clearAllMocks();
    });

    afterEach(async () => {
        await provider.shutdown();
    });

    describe("Initialization", () => {
        it("should initialize successfully with valid config", async () => {
            const options: ResourceInitOptions<"ollama"> = {
                config: mockConfig,
                globalConfig: {} as any,
            };

            await provider.initialize(options);

            expect(provider.isInitialized).toBe(true);
            const info = provider.getPublicInfo();
            expect(info.status).toBe(DiscoveryStatus.Available);
        });

        it("should not initialize when disabled", async () => {
            const disabledConfig = { ...mockConfig, enabled: false };
            const options: ResourceInitOptions<"ollama"> = {
                config: disabledConfig,
                globalConfig: {} as any,
            };

            await provider.initialize(options);

            const info = provider.getPublicInfo();
            expect(info.status).toBe(DiscoveryStatus.NotFound);
        });

        it("should not initialize twice", async () => {
            const options: ResourceInitOptions<"ollama"> = {
                config: mockConfig,
                globalConfig: {} as any,
            };

            await provider.initialize(options);
            await provider.initialize(options); // Second call

            // Should only discover once
            expect(provider.getDiscoveryCallCount()).toBe(1);
        });

        it("should handle initialization failure", async () => {
            provider.setDiscoveryResult(false);
            
            const options: ResourceInitOptions<"ollama"> = {
                config: mockConfig,
                globalConfig: {} as any,
            };

            await provider.initialize(options);

            const info = provider.getPublicInfo();
            expect(info.status).toBe(DiscoveryStatus.NotFound);
        });
    });

    describe("Discovery", () => {
        beforeEach(async () => {
            const options: ResourceInitOptions<"ollama"> = {
                config: mockConfig,
                globalConfig: {} as any,
            };
            await provider.initialize(options);
        });

        it("should discover available resource", async () => {
            provider.setDiscoveryResult(true);
            const result = await provider.discover();

            expect(result).toBe(true);
            expect(provider.getPublicInfo().status).toBe(DiscoveryStatus.Available);
        });

        it("should handle discovery failure", async () => {
            provider.setDiscoveryResult(false);
            const result = await provider.discover();

            expect(result).toBe(false);
            expect(provider.getPublicInfo().status).toBe(DiscoveryStatus.NotFound);
        });

        it("should emit discovery events", async () => {
            const discoveredHandler = jest.fn();
            const lostHandler = jest.fn();
            
            provider.on(ResourceEvent.Discovered, discoveredHandler);
            provider.on(ResourceEvent.Lost, lostHandler);

            // First discovery should emit discovered event
            provider.setDiscoveryResult(true);
            await provider.discover();
            expect(discoveredHandler).toHaveBeenCalledTimes(1);

            // Losing resource should emit lost event
            provider.setDiscoveryResult(false);
            await provider.discover();
            expect(lostHandler).toHaveBeenCalledTimes(1);
        });

        it("should use circuit breaker for discovery", async () => {
            // Force multiple discovery failures to trigger circuit breaker
            provider.setDiscoveryResult(false);
            
            // Trigger enough failures to open circuit
            for (let i = 0; i < 5; i++) {
                await provider.discover();
            }

            const stats = provider.getCircuitBreakerStats();
            expect(stats.discovery?.failureCount).toBeGreaterThan(0);
        });
    });

    describe("Health checking", () => {
        beforeEach(async () => {
            const options: ResourceInitOptions<"ollama"> = {
                config: mockConfig,
                globalConfig: {} as any,
            };
            await provider.initialize(options);
        });

        it("should perform successful health check", async () => {
            provider.setHealthCheckResult(true);
            const result = await provider.healthCheck();

            expect(result.healthy).toBe(true);
            expect(provider.getPublicInfo().health).toBe(ResourceHealth.Healthy);
        });

        it("should handle health check failure", async () => {
            provider.setHealthCheckResult(false);
            const result = await provider.healthCheck();

            expect(result.healthy).toBe(false);
            expect(provider.getPublicInfo().health).toBe(ResourceHealth.Unhealthy);
        });

        it("should emit health change events", async () => {
            const healthChangedHandler = jest.fn();
            provider.on(ResourceEvent.HealthChanged, healthChangedHandler);

            // Change from unknown to healthy
            provider.setHealthCheckResult(true);
            await provider.healthCheck();
            expect(healthChangedHandler).toHaveBeenCalledTimes(1);

            // Change from healthy to unhealthy
            provider.setHealthCheckResult(false);
            await provider.healthCheck();
            expect(healthChangedHandler).toHaveBeenCalledTimes(2);
        });

        it("should not perform health check when not available", async () => {
            // Set resource as not available
            provider.setDiscoveryResult(false);
            await provider.discover();

            const result = await provider.healthCheck();
            expect(result.healthy).toBe(false);
            expect(result.message).toContain("not available");
        });

        it("should track consecutive failures", async () => {
            provider.setHealthCheckResult(false);

            // Multiple health check failures
            for (let i = 0; i < 6; i++) { // More than maxConsecutiveFailures (5)
                await provider.healthCheck();
            }

            // Should have triggered resource loss due to consecutive failures
            const info = provider.getPublicInfo();
            expect(info.status).toBe(DiscoveryStatus.NotFound);
        });
    });

    describe("Resource information", () => {
        beforeEach(async () => {
            const options: ResourceInitOptions<"ollama"> = {
                config: mockConfig,
                globalConfig: {} as any,
            };
            await provider.initialize(options);
        });

        it("should provide public resource info", () => {
            const info = provider.getPublicInfo();

            expect(info.id).toBe("ollama");
            expect(info.category).toBe(ResourceCategory.AI);
            expect(info.displayName).toBe("Test Ollama");
            expect(info.description).toBe("Test Ollama resource");
            expect(info.metadata).toBeDefined();
        });

        it("should provide internal resource info with config", () => {
            const info = provider.getInternalInfo();

            expect(info.id).toBe("ollama");
            expect(info.config).toBeDefined();
            expect(info.config?.enabled).toBe(true);
        });
    });

    describe("Health monitoring", () => {
        it("should start health monitoring when configured", async () => {
            const options: ResourceInitOptions<"ollama"> = {
                config: {
                    ...mockConfig,
                    healthCheck: {
                        intervalMs: 100, // Short interval for testing
                        timeoutMs: 5000,
                    },
                },
                globalConfig: {} as any,
            };

            await provider.initialize(options);

            // Wait for at least one health check interval
            await new Promise(resolve => setTimeout(resolve, 150));

            // Should have performed additional health checks beyond initialization
            expect(provider.getHealthCheckCallCount()).toBeGreaterThan(1);
        });

        it("should restart health monitoring after failures", async () => {
            const options: ResourceInitOptions<"ollama"> = {
                config: mockConfig,
                globalConfig: {} as any,
            };
            await provider.initialize(options);

            // Cause health monitoring to stop due to failures
            provider.setHealthCheckResult(false);
            for (let i = 0; i < 6; i++) {
                await provider.healthCheck();
            }

            // Resource should be marked as lost
            expect(provider.getPublicInfo().status).toBe(DiscoveryStatus.NotFound);

            // Reset resource to available and restart monitoring
            provider.setDiscoveryResult(true);
            provider.setHealthCheckResult(true);
            await provider.discover();
            provider.restartHealthMonitoringPublic();

            // Should be healthy again
            expect(provider.getPublicInfo().status).toBe(DiscoveryStatus.Available);
        });
    });

    describe("Authentication configuration", () => {
        it("should extract API key auth config", async () => {
            const configWithAuth = {
                ...mockConfig,
                apiKey: "test-api-key",
                apiKeyHeader: "X-API-Key",
            };

            const options: ResourceInitOptions<"ollama"> = {
                config: configWithAuth,
                globalConfig: {} as any,
            };
            await provider.initialize(options);

            const authConfig = (provider as any).getAuthConfig();
            expect(authConfig).toBeDefined();
            expect(authConfig?.type).toBe("apikey");
            expect(authConfig?.token).toBe("test-api-key");
            expect(authConfig?.headerName).toBe("X-API-Key");
        });

        it("should extract bearer token auth config", async () => {
            const configWithAuth = {
                ...mockConfig,
                bearerToken: "test-bearer-token",
            };

            const options: ResourceInitOptions<"ollama"> = {
                config: configWithAuth,
                globalConfig: {} as any,
            };
            await provider.initialize(options);

            const authConfig = (provider as any).getAuthConfig();
            expect(authConfig?.type).toBe("bearer");
            expect(authConfig?.token).toBe("test-bearer-token");
        });

        it("should extract basic auth config", async () => {
            const configWithAuth = {
                ...mockConfig,
                username: "testuser",
                password: "testpass",
            };

            const options: ResourceInitOptions<"ollama"> = {
                config: configWithAuth,
                globalConfig: {} as any,
            };
            await provider.initialize(options);

            const authConfig = (provider as any).getAuthConfig();
            expect(authConfig?.type).toBe("basic");
            expect(authConfig?.username).toBe("testuser");
            expect(authConfig?.password).toBe("testpass");
        });
    });

    describe("Shutdown", () => {
        it("should clean up resources on shutdown", async () => {
            const options: ResourceInitOptions<"ollama"> = {
                config: mockConfig,
                globalConfig: {} as any,
            };
            await provider.initialize(options);

            await provider.shutdown();

            const info = provider.getPublicInfo();
            expect(info.status).toBe(DiscoveryStatus.NotFound);
            expect(info.health).toBe(ResourceHealth.Unknown);
            expect(provider.isInitialized).toBe(false);
        });
    });
});
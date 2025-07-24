/**
 * Tests for ResourceRegistry core functionality
 */

import { ResourceRegistry } from "../ResourceRegistry.js";
import { ResourceProvider } from "../ResourceProvider.js";
import { ResourceCategory, DeploymentType, ResourceHealth, DiscoveryStatus } from "../types.js";
import type { HealthCheckResult, ResourceInitOptions } from "../types.js";
import type { AIMetadata } from "../typeRegistry.js";

// Mock dependencies
jest.mock("../../events/logger.js", () => ({
    logger: {
        debug: jest.fn(),
        info: jest.fn(),
        warn: jest.fn(),
        error: jest.fn(),
    },
}));

jest.mock("../resourcesConfig.js", () => ({
    loadResourcesConfig: jest.fn().mockResolvedValue({
        version: "1.0.0",
        enabled: true,
        services: {
            ai: {
                "test-ai": {
                    enabled: true,
                    baseUrl: "http://localhost:8080",
                },
                "test-ai-disabled": {
                    enabled: false,
                    baseUrl: "http://localhost:8081",
                },
            },
            automation: {
                "test-automation": {
                    enabled: true,
                    baseUrl: "http://localhost:8082",
                },
            },
        },
        discovery: {
            enabled: false,
        },
    }),
}));

jest.mock("../http/httpClient.js", () => ({
    HTTPClient: {
        forResources: jest.fn(() => ({
            makeRequest: jest.fn().mockResolvedValue({ success: true }),
        })),
    },
}));

// Test resource implementations
class TestAIResource extends ResourceProvider<"ollama"> {
    readonly id = "test-ai";
    readonly category = ResourceCategory.AI;
    readonly displayName = "Test AI";
    readonly description = "Test AI resource";
    readonly isSupported = true;
    readonly deploymentType = DeploymentType.Local;

    protected async performDiscovery(): Promise<boolean> {
        return true;
    }

    protected async performHealthCheck(): Promise<HealthCheckResult> {
        return {
            healthy: true,
            message: "Test AI is healthy",
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
}

class TestAIResourceDisabled extends ResourceProvider<"ollama"> {
    readonly id = "test-ai-disabled";
    readonly category = ResourceCategory.AI;
    readonly displayName = "Test AI Disabled";
    readonly description = "Test AI resource (disabled)";
    readonly isSupported = true;
    readonly deploymentType = DeploymentType.Local;

    protected async performDiscovery(): Promise<boolean> {
        return true;
    }

    protected async performHealthCheck(): Promise<HealthCheckResult> {
        return {
            healthy: true,
            message: "Test AI disabled is healthy",
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
}

class TestAutomationResource extends ResourceProvider<"n8n"> {
    readonly id = "test-automation";
    readonly category = ResourceCategory.Automation;
    readonly displayName = "Test Automation";
    readonly description = "Test automation resource";
    readonly isSupported = true;
    readonly deploymentType = DeploymentType.Local;

    protected async performDiscovery(): Promise<boolean> {
        return true;
    }

    protected async performHealthCheck(): Promise<HealthCheckResult> {
        return {
            healthy: true,
            message: "Test automation is healthy",
            timestamp: new Date(),
        };
    }

    protected getMetadata() {
        return {
            version: "1.0.0",
            capabilities: ["test"],
            workflowCount: 0,
            supportedTriggers: ["test"],
            supportedActions: ["test"],
            lastUpdated: new Date(),
        };
    }
}

describe("ResourceRegistry", () => {
    let registry: ResourceRegistry;

    beforeEach(() => {
        // Create fresh instance for each test
        (ResourceRegistry as any).instance = undefined;
        registry = ResourceRegistry.getInstance();
    });

    afterEach(async () => {
        await registry.shutdown();
        (ResourceRegistry as any).instance = undefined;
    });

    describe("Singleton pattern", () => {
        it("should return the same instance", () => {
            const registry1 = ResourceRegistry.getInstance();
            const registry2 = ResourceRegistry.getInstance();
            expect(registry1).toBe(registry2);
        });
    });

    describe("Resource registration", () => {
        it("should register resource classes", () => {
            registry.registerResourceClass(TestAIResource);
            
            // Should be registered but not yet instantiated
            const summary = registry.getResourceSummary();
            expect(summary.total).toBe(1);
        });

        it("should handle duplicate registration", () => {
            registry.registerResourceClass(TestAIResource);
            registry.registerResourceClass(TestAIResource); // Duplicate
            
            const summary = registry.getResourceSummary();
            expect(summary.total).toBe(1); // Should not duplicate
        });

        it("should register multiple resource types", () => {
            registry.registerResourceClass(TestAIResource);
            registry.registerResourceClass(TestAutomationResource);
            
            const summary = registry.getResourceSummary();
            expect(summary.total).toBe(2);
        });
    });

    describe("Dependency management", () => {
        beforeEach(() => {
            registry.registerResourceClass(TestAIResource);
            registry.registerResourceClass(TestAutomationResource);
        });

        it("should register resource dependencies", () => {
            registry.registerResourceDependency({
                resourceId: "test-automation",
                dependsOn: ["test-ai"],
                priority: 50,
            });

            const depInfo = registry.getResourceDependencies("test-automation");
            expect(depInfo).toBeDefined();
            expect(depInfo?.dependsOn).toContain("test-ai");
        });

        it("should find dependent resources", () => {
            registry.registerResourceDependency({
                resourceId: "test-automation",
                dependsOn: ["test-ai"],
                priority: 50,
            });

            const dependents = registry.getDependentResources("test-ai");
            expect(dependents).toContain("test-automation");
        });

        it("should check safe shutdown conditions", () => {
            registry.registerResourceDependency({
                resourceId: "test-automation",
                dependsOn: ["test-ai"],
                priority: 50,
            });

            // Before initialization, should be safe to shutdown
            expect(registry.canSafelyShutdownResource("test-ai")).toBe(true);
        });

        it("should register multiple dependencies", () => {
            const dependencies = [
                {
                    resourceId: "test-automation",
                    dependsOn: ["test-ai"],
                    priority: 50,
                },
                {
                    resourceId: "test-ai",
                    dependsOn: [],
                    priority: 100,
                },
            ];

            registry.registerResourceDependencies(dependencies);

            expect(registry.getResourceDependencies("test-automation")).toBeDefined();
            expect(registry.getResourceDependencies("test-ai")).toBeDefined();
        });

        it("should provide dependency manager summary", () => {
            registry.registerResourceDependency({
                resourceId: "test-automation",
                dependsOn: ["test-ai"],
                priority: 50,
            });

            const summary = registry.getDependencyManagerSummary();
            expect(summary.totalResources).toBeGreaterThan(0);
            expect(summary.resourcesWithDependencies).toBeGreaterThan(0);
        });
    });

    describe("Resource initialization", () => {
        beforeEach(() => {
            registry.registerResourceClass(TestAIResource);
            registry.registerResourceClass(TestAIResourceDisabled);
            registry.registerResourceClass(TestAutomationResource);
        });

        it("should initialize with configuration", async () => {
            await registry.initialize();

            const summary = registry.getResourceSummary();
            expect(summary.available).toBeGreaterThan(0); // At least some resources should be available
        });

        it("should only initialize enabled resources", async () => {
            await registry.initialize();

            // test-ai-disabled should not be available because it's disabled in config
            expect(registry.isResourceAvailable("test-ai")).toBe(true);
            expect(registry.isResourceAvailable("test-ai-disabled")).toBe(false);
        });

        it("should handle initialization errors gracefully", async () => {
            // This should not throw even if some resources fail
            await expect(registry.initialize()).resolves.not.toThrow();
        });

        it("should not initialize twice", async () => {
            await registry.initialize();
            await registry.initialize(); // Second call should be ignored
            
            // Should not cause issues
            expect(registry.isResourceAvailable("test-ai")).toBe(true);
        });
    });

    describe("Resource queries", () => {
        beforeEach(async () => {
            registry.registerResourceClass(TestAIResource);
            registry.registerResourceClass(TestAutomationResource);
            await registry.initialize();
        });

        it("should get specific resources", () => {
            const aiResource = registry.getResource("test-ai");
            expect(aiResource).toBeDefined();
            expect(aiResource?.getPublicInfo().category).toBe(ResourceCategory.AI);
        });

        it("should get all resources", () => {
            const allResources = registry.getAllResources();
            expect(allResources.length).toBeGreaterThan(0);
        });

        it("should get resources by category", () => {
            const aiResources = registry.getResourcesByCategory(ResourceCategory.AI);
            const automationResources = registry.getResourcesByCategory(ResourceCategory.Automation);
            
            expect(aiResources.length).toBeGreaterThan(0);
            expect(automationResources.length).toBeGreaterThan(0);
        });

        it("should get resources by status", () => {
            const availableResources = registry.getResourcesByStatus(DiscoveryStatus.Available);
            expect(availableResources.length).toBeGreaterThan(0);
        });

        it("should provide resource summary", () => {
            const summary = registry.getResourceSummary();
            expect(summary.total).toBeGreaterThan(0);
            expect(summary.available).toBeGreaterThan(0);
            expect(summary.byCategory).toBeDefined();
        });

        it("should check resource availability", () => {
            expect(registry.isResourceAvailable("test-ai")).toBe(true);
            expect(registry.isResourceAvailable("non-existent")).toBe(false);
        });
    });

    describe("Health checking", () => {
        beforeEach(async () => {
            registry.registerResourceClass(TestAIResource);
            await registry.initialize();
        });

        it("should provide system health check", () => {
            const healthCheck = registry.getHealthCheck();
            
            expect(healthCheck).toBeDefined();
            expect(healthCheck.status).toBeDefined();
            expect(healthCheck.stats).toBeDefined();
            expect(healthCheck.timestamp).toBeInstanceOf(Date);
        });

        it("should include resource information in health check", () => {
            const healthCheck = registry.getHealthCheck();
            
            expect(healthCheck.resources).toBeDefined();
            expect(healthCheck.resources.length).toBeGreaterThan(0);
        });
    });

    describe("Resource restart functionality", () => {
        beforeEach(async () => {
            registry.registerResourceClass(TestAIResource);
            registry.registerResourceClass(TestAutomationResource);
            
            registry.registerResourceDependency({
                resourceId: "test-automation",
                dependsOn: ["test-ai"],
                priority: 50,
            });
            
            await registry.initialize();
        });

        it("should restart resource with dependents", async () => {
            // This should not throw and should handle the restart gracefully
            await expect(registry.restartResourceWithDependents("test-ai")).resolves.not.toThrow();
            
            // Resources should still be available after restart
            expect(registry.isResourceAvailable("test-ai")).toBe(true);
            expect(registry.isResourceAvailable("test-automation")).toBe(true);
        });
    });

    describe("Shutdown", () => {
        it("should shutdown cleanly", async () => {
            registry.registerResourceClass(TestAIResource);
            await registry.initialize();
            
            await registry.shutdown();
            
            const summary = registry.getResourceSummary();
            expect(summary.available).toBe(0);
        });

        it("should handle shutdown when not initialized", async () => {
            await expect(registry.shutdown()).resolves.not.toThrow();
        });
    });
});
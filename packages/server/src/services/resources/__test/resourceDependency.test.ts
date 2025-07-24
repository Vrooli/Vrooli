/**
 * Tests for Resource Dependency Management
 */

import { ResourceDependencyManager, COMMON_DEPENDENCIES } from "../resourceDependency.js";
import type { ResourceDependency } from "../resourceDependency.js";

describe("ResourceDependencyManager", () => {
    let manager: ResourceDependencyManager;

    beforeEach(() => {
        manager = new ResourceDependencyManager();
    });

    describe("Basic registration", () => {
        it("should register resources without dependencies", () => {
            manager.registerResource("ollama", 100);
            
            const summary = manager.getSummary();
            expect(summary.totalResources).toBe(1);
            expect(summary.resourcesWithDependencies).toBe(0);
        });

        it("should register resource dependencies", () => {
            const dependency: ResourceDependency = {
                resourceId: "automation-service",
                dependsOn: ["ollama"],
                priority: 50,
            };

            manager.registerResourceDependency(dependency);
            
            const info = manager.getDependencyInfo("automation-service");
            expect(info).toBeDefined();
            expect(info?.dependsOn).toEqual(["ollama"]);
            expect(info?.priority).toBe(50);
        });

        it("should prevent self-dependencies", () => {
            const dependency: ResourceDependency = {
                resourceId: "ollama",
                dependsOn: ["ollama"], // Self-dependency
                priority: 100,
            };

            expect(() => manager.registerResourceDependency(dependency))
                .toThrow("Resource ollama cannot depend on itself");
        });
    });

    describe("Initialization planning", () => {
        it("should create single phase for independent resources", () => {
            manager.registerResource("ollama", 100);
            manager.registerResource("localai", 90);
            manager.registerResource("minio", 80);

            const enabledResources = new Set(["ollama", "localai", "minio"]);
            const plan = manager.createInitializationPlan(enabledResources);

            expect(plan.phases).toHaveLength(1);
            expect(plan.phases[0]).toHaveLength(3);
            expect(plan.circularDependencies).toHaveLength(0);
            expect(plan.missingDependencies).toHaveLength(0);

            // Should be ordered by priority (highest first)
            expect(plan.phases[0][0]).toBe("ollama"); // Priority 100
            expect(plan.phases[0][1]).toBe("localai"); // Priority 90
            expect(plan.phases[0][2]).toBe("minio"); // Priority 80
        });

        it("should create multiple phases for dependent resources", () => {
            manager.registerResource("minio", 200); // Storage first
            manager.registerResource("ollama", 100); // AI second
            
            manager.registerResourceDependency({
                resourceId: "automation-service",
                dependsOn: ["ollama", "minio"],
                priority: 50,
            });

            const enabledResources = new Set(["minio", "ollama", "automation-service"]);
            const plan = manager.createInitializationPlan(enabledResources);

            expect(plan.phases).toHaveLength(2);
            
            // Phase 1: Independent resources (sorted by priority)
            expect(plan.phases[0]).toContain("minio");
            expect(plan.phases[0]).toContain("ollama");
            
            // Phase 2: Dependent resource
            expect(plan.phases[1]).toContain("automation-service");
        });

        it("should handle complex dependency chains", () => {
            // Storage layer
            manager.registerResource("minio", 300);
            
            // AI layer depends on storage
            manager.registerResourceDependency({
                resourceId: "ollama",
                dependsOn: ["minio"],
                priority: 200,
            });
            
            // Automation depends on AI
            manager.registerResourceDependency({
                resourceId: "n8n",
                dependsOn: ["ollama"],
                priority: 100,
            });
            
            // Agent depends on automation and AI
            manager.registerResourceDependency({
                resourceId: "puppeteer",
                dependsOn: ["n8n", "ollama"],
                priority: 50,
            });

            const enabledResources = new Set(["minio", "ollama", "n8n", "puppeteer"]);
            const plan = manager.createInitializationPlan(enabledResources);

            expect(plan.phases).toHaveLength(4);
            expect(plan.phases[0]).toEqual(["minio"]);
            expect(plan.phases[1]).toEqual(["ollama"]);
            expect(plan.phases[2]).toEqual(["n8n"]);
            expect(plan.phases[3]).toEqual(["puppeteer"]);
        });

        it("should detect circular dependencies", () => {
            manager.registerResourceDependency({
                resourceId: "serviceA",
                dependsOn: ["serviceB"],
                priority: 100,
            });
            
            manager.registerResourceDependency({
                resourceId: "serviceB",
                dependsOn: ["serviceC"],
                priority: 100,
            });
            
            manager.registerResourceDependency({
                resourceId: "serviceC",
                dependsOn: ["serviceA"], // Creates cycle
                priority: 100,
            });

            const enabledResources = new Set(["serviceA", "serviceB", "serviceC"]);
            const plan = manager.createInitializationPlan(enabledResources);

            expect(plan.circularDependencies.length).toBeGreaterThan(0);
        });

        it("should handle missing dependencies", () => {
            manager.registerResourceDependency({
                resourceId: "automation-service",
                dependsOn: ["ollama", "missing-service"], // missing-service is not registered
                priority: 50,
            });

            const enabledResources = new Set(["automation-service"]);
            const plan = manager.createInitializationPlan(enabledResources);

            expect(plan.missingDependencies).toHaveLength(1);
            expect(plan.missingDependencies[0].resourceId).toBe("automation-service");
            expect(plan.missingDependencies[0].missingDeps).toContain("missing-service");
        });

        it("should handle optional dependencies", () => {
            manager.registerResourceDependency({
                resourceId: "automation-service",
                dependsOn: ["ollama", "optional-service"],
                priority: 50,
                optional: true,
            });

            const enabledResources = new Set(["automation-service"]);
            const plan = manager.createInitializationPlan(enabledResources);

            // Should not report missing dependencies for optional deps
            expect(plan.missingDependencies).toHaveLength(0);
            expect(plan.phases[0]).toContain("automation-service");
        });
    });

    describe("Dependency queries", () => {
        beforeEach(() => {
            manager.registerResource("minio", 200);
            manager.registerResource("ollama", 100);
            
            manager.registerResourceDependency({
                resourceId: "automation-service",
                dependsOn: ["ollama"],
                priority: 50,
            });
            
            manager.registerResourceDependency({
                resourceId: "agent-service",
                dependsOn: ["automation-service", "ollama"],
                priority: 25,
            });
        });

        it("should find dependent resources", () => {
            const dependents = manager.getDependentResources("ollama");
            expect(dependents).toContain("automation-service");
            expect(dependents).toContain("agent-service");
        });

        it("should check safe shutdown conditions", () => {
            const currentlyRunning = new Set(["minio", "ollama", "automation-service", "agent-service"]);
            
            // Can't safely shut down ollama because others depend on it
            expect(manager.canSafelyShutdown("ollama", currentlyRunning)).toBe(false);
            
            // Can safely shut down agent-service (nothing depends on it)
            expect(manager.canSafelyShutdown("agent-service", currentlyRunning)).toBe(true);
            
            // Can safely shut down minio (nothing depends on it)
            expect(manager.canSafelyShutdown("minio", currentlyRunning)).toBe(true);
        });

        it("should allow shutdown when dependents are not running", () => {
            const currentlyRunning = new Set(["minio", "ollama"]);
            // automation-service and agent-service are not running
            
            expect(manager.canSafelyShutdown("ollama", currentlyRunning)).toBe(true);
        });
    });

    describe("Summary and statistics", () => {
        it("should provide accurate summary", () => {
            manager.registerResource("service1", 100); // No dependencies
            manager.registerResource("service2", 90);  // No dependencies
            
            manager.registerResourceDependency({
                resourceId: "service3",
                dependsOn: ["service1"],
                priority: 80,
            });
            
            manager.registerResourceDependency({
                resourceId: "service4",
                dependsOn: ["service1", "service2", "service3"],
                priority: 70,
            });

            const summary = manager.getSummary();
            
            expect(summary.totalResources).toBe(4);
            expect(summary.resourcesWithDependencies).toBe(2); // service3 and service4
            expect(summary.maxDependencies).toBe(3); // service4 has 3 dependencies
            expect(summary.averageDependencies).toBe(1); // (0 + 0 + 1 + 3) / 4 = 1
        });
    });

    describe("Common dependencies configuration", () => {
        it("should provide common dependency templates", () => {
            expect(COMMON_DEPENDENCIES.ai).toBeDefined();
            expect(COMMON_DEPENDENCIES.automation).toBeDefined();
            expect(COMMON_DEPENDENCIES.agents).toBeDefined();
            expect(COMMON_DEPENDENCIES.storage).toBeDefined();
            
            expect(COMMON_DEPENDENCIES.storage.priority).toBeGreaterThan(
                COMMON_DEPENDENCIES.ai.priority ?? 0
            );
        });
    });

    describe("Clear and reset", () => {
        it("should clear all dependencies", () => {
            manager.registerResource("service1", 100);
            manager.registerResourceDependency({
                resourceId: "service2",
                dependsOn: ["service1"],
                priority: 50,
            });

            expect(manager.getSummary().totalResources).toBe(2);

            manager.clear();
            
            expect(manager.getSummary().totalResources).toBe(0);
            expect(manager.getDependencyInfo("service1")).toBeUndefined();
            expect(manager.getDependencyInfo("service2")).toBeUndefined();
        });
    });
});
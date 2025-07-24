/**
 * Tests for ResourceDependencyManager
 */

import { beforeEach, describe, expect, it } from "vitest";
import type { ResourceDependency } from "./resourceDependency.js";
import { COMMON_DEPENDENCIES, ResourceDependencyManager } from "./resourceDependency.js";

describe("ResourceDependencyManager", () => {
    let manager: ResourceDependencyManager;

    beforeEach(() => {
        manager = new ResourceDependencyManager();
    });

    describe("Basic registration", () => {
        it("should register resources without dependencies", () => {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            manager.registerResource("ollama" as any, 100);

            const summary = manager.getSummary();
            expect(summary.totalResources).toBe(1);
            expect(summary.resourcesWithDependencies).toBe(0);
        });

        it("should register resource dependencies", () => {
            const dependency: ResourceDependency = {
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "automation-service" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["ollama" as any],
                priority: 50,
            };

            manager.registerResourceDependency(dependency);

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const info = manager.getDependencyInfo("automation-service" as any);
            expect(info).toBeDefined();
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            expect(info?.dependsOn).toEqual(["ollama" as any]);
            expect(info?.priority).toBe(50);
        });

        it("should prevent self-dependencies", () => {
            const dependency: ResourceDependency = {
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "ollama" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["ollama" as any], // Self-dependency
                priority: 100,
            };

            expect(() => manager.registerResourceDependency(dependency))
                .toThrow("Resource ollama cannot depend on itself");
        });
    });

    describe("Initialization planning", () => {
        it("should create single phase for independent resources", () => {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            manager.registerResource("ollama" as any, 100);
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            manager.registerResource("localai" as any, 90);
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            manager.registerResource("minio" as any, 80);

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const enabledResources = new Set(["ollama", "localai", "minio"]) as any;
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
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            manager.registerResource("minio" as any, 200); // Storage first
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            manager.registerResource("ollama" as any, 100); // AI second

            manager.registerResourceDependency({
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "automation-service" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["ollama" as any, "minio" as any],
                priority: 50,
            });

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const enabledResources = new Set(["minio", "ollama", "automation-service"]) as any;
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
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            manager.registerResource("minio" as any, 300);

            // AI layer depends on storage
            manager.registerResourceDependency({
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "ollama" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["minio" as any],
                priority: 200,
            });

            // Automation depends on AI
            manager.registerResourceDependency({
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "n8n" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["ollama" as any],
                priority: 100,
            });

            // Agent depends on automation and AI
            manager.registerResourceDependency({
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "puppeteer" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["n8n" as any, "ollama" as any],
                priority: 50,
            });

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const enabledResources = new Set(["minio", "ollama", "n8n", "puppeteer"]) as any;
            const plan = manager.createInitializationPlan(enabledResources);

            expect(plan.phases).toHaveLength(4);
            expect(plan.phases[0]).toEqual(["minio"]);
            expect(plan.phases[1]).toEqual(["ollama"]);
            expect(plan.phases[2]).toEqual(["n8n"]);
            expect(plan.phases[3]).toEqual(["puppeteer"]);
        });

        it("should detect circular dependencies", () => {
            manager.registerResourceDependency({
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "serviceA" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["serviceB" as any],
                priority: 100,
            });

            manager.registerResourceDependency({
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "serviceB" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["serviceC" as any],
                priority: 100,
            });

            manager.registerResourceDependency({
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "serviceC" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["serviceA" as any], // Creates cycle
                priority: 100,
            });

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const enabledResources = new Set(["serviceA", "serviceB", "serviceC"]) as any;
            const plan = manager.createInitializationPlan(enabledResources);

            expect(plan.circularDependencies.length).toBeGreaterThan(0);
        });

        it("should handle missing dependencies", () => {
            manager.registerResourceDependency({
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "automation-service" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["ollama" as any, "missing-service" as any], // missing-service is not registered
                priority: 50,
            });

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const enabledResources = new Set(["automation-service"]) as any;
            const plan = manager.createInitializationPlan(enabledResources);

            expect(plan.missingDependencies).toHaveLength(1);
            expect(plan.missingDependencies[0].resourceId).toBe("automation-service");
            expect(plan.missingDependencies[0].missingDeps).toContain("missing-service");
        });

        it("should handle optional dependencies", () => {
            manager.registerResourceDependency({
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "automation-service" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["ollama" as any, "optional-service" as any],
                priority: 50,
                optional: true,
            });

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const enabledResources = new Set(["automation-service"]) as any;
            const plan = manager.createInitializationPlan(enabledResources);

            // Should not report missing dependencies for optional deps
            expect(plan.missingDependencies).toHaveLength(0);
            expect(plan.phases[0]).toContain("automation-service");
        });
    });

    describe("Dependency queries", () => {
        beforeEach(() => {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            manager.registerResource("minio" as any, 200);
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            manager.registerResource("ollama" as any, 100);

            manager.registerResourceDependency({
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "automation-service" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["ollama" as any],
                priority: 50,
            });

            manager.registerResourceDependency({
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "agent-service" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["automation-service" as any, "ollama" as any],
                priority: 25,
            });
        });

        it("should find dependent resources", () => {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const dependents = manager.getDependentResources("ollama" as any);
            expect(dependents).toContain("automation-service");
            expect(dependents).toContain("agent-service");
        });

        it("should check safe shutdown conditions", () => {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const currentlyRunning = new Set(["minio", "ollama", "automation-service", "agent-service"]) as any;

            // Can't safely shut down ollama because others depend on it
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            expect(manager.canSafelyShutdown("ollama" as any, currentlyRunning)).toBe(false);

            // Can safely shut down agent-service (nothing depends on it)
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            expect(manager.canSafelyShutdown("agent-service" as any, currentlyRunning)).toBe(true);

            // Can safely shut down minio (nothing depends on it)
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            expect(manager.canSafelyShutdown("minio" as any, currentlyRunning)).toBe(true);
        });

        it("should allow shutdown when dependents are not running", () => {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const currentlyRunning = new Set(["minio", "ollama"]) as any;
            // automation-service and agent-service are not running

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            expect(manager.canSafelyShutdown("ollama" as any, currentlyRunning)).toBe(true);
        });
    });

    describe("Summary and statistics", () => {
        it("should provide accurate summary", () => {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            manager.registerResource("service1" as any, 100); // No dependencies
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            manager.registerResource("service2" as any, 90);  // No dependencies

            manager.registerResourceDependency({
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "service3" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["service1" as any],
                priority: 80,
            });

            manager.registerResourceDependency({
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "service4" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["service1" as any, "service2" as any, "service3" as any],
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
                COMMON_DEPENDENCIES.ai.priority ?? 0,
            );
        });
    });

    describe("Clear and reset", () => {
        it("should clear all dependencies", () => {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            manager.registerResource("service1" as any, 100);
            manager.registerResourceDependency({
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                resourceId: "service2" as any,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                dependsOn: ["service1" as any],
                priority: 50,
            });

            expect(manager.getSummary().totalResources).toBe(2);

            manager.clear();

            expect(manager.getSummary().totalResources).toBe(0);
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            expect(manager.getDependencyInfo("service1" as any)).toBeUndefined();
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            expect(manager.getDependencyInfo("service2" as any)).toBeUndefined();
        });
    });
});

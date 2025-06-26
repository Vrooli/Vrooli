import { describe, it, expect, beforeEach } from "vitest";
import { createLogger } from "winston";
import { NavigatorRegistry } from "./navigatorRegistry.js";
import { type Navigator } from "@vrooli/shared";

// Mock navigator for testing
class MockNavigator implements Navigator {
    readonly type: string;
    readonly version: string;

    constructor(type: string, version = "1.0.0") {
        this.type = type;
        this.version = version;
    }

    canNavigate(_routine: unknown): boolean {
        return true;
    }

    getStartLocation(_routine: unknown) {
        return {
            id: "start",
            routineId: "test",
            nodeId: "start",
        };
    }

    getNextLocations() {
        return [];
    }

    isEndLocation() {
        return false;
    }

    getStepInfo() {
        return {
            id: "test",
            name: "Test Step",
            type: "action",
        };
    }
}

describe("NavigatorRegistry", () => {
    let registry: NavigatorRegistry;
    let logger: any;

    beforeEach(() => {
        logger = createLogger({ silent: true });
        registry = new NavigatorRegistry(logger);
    });

    describe("Basic Operations", () => {
        it("should register and retrieve navigators", () => {
            const mockNavigator = new MockNavigator("test");
            registry.registerNavigator(mockNavigator);

            const retrieved = registry.getNavigator("test");
            expect(retrieved).toBe(mockNavigator);
        });

        it("should check if navigator exists", () => {
            const mockNavigator = new MockNavigator("test");
            registry.registerNavigator(mockNavigator);

            expect(registry.hasNavigator("test")).toBe(true);
            expect(registry.hasNavigator("nonexistent")).toBe(false);
        });

        it("should list navigator types", () => {
            const nav1 = new MockNavigator("type1");
            const nav2 = new MockNavigator("type2");
            
            registry.registerNavigator(nav1);
            registry.registerNavigator(nav2);

            const types = registry.listNavigatorTypes();
            expect(types).toContain("type1");
            expect(types).toContain("type2");
        });
    });

    describe("Metrics Collection", () => {
        let testNavigator: MockNavigator;

        beforeEach(() => {
            testNavigator = new MockNavigator("test", "2.0.0");
            registry.registerNavigator(testNavigator);
        });

        it("should return initial metrics with zeros", () => {
            const metrics = registry.getNavigatorMetrics();
            
            expect(metrics).toHaveProperty("test");
            expect(metrics.test).toEqual({
                type: "test",
                version: "2.0.0",
                routinesProcessed: 0,
                averageStepTime: 0,
                successRate: 1.0,
            });
        });

        it("should track successful routine executions", () => {
            // Record some successful executions
            registry.recordRoutineSuccess("test", 100);
            registry.recordRoutineSuccess("test", 200);
            registry.recordRoutineSuccess("test", 300);

            const metrics = registry.getNavigatorMetrics();
            
            expect(metrics.test.routinesProcessed).toBe(3);
            expect(metrics.test.averageStepTime).toBe(200); // (100+200+300)/3
            expect(metrics.test.successRate).toBe(1.0);
        });

        it("should track failed routine executions", () => {
            // Record mixed success/failure
            registry.recordRoutineSuccess("test", 100);
            registry.recordRoutineFailure("test");
            registry.recordRoutineSuccess("test", 200);
            registry.recordRoutineFailure("test");

            const metrics = registry.getNavigatorMetrics();
            
            expect(metrics.test.routinesProcessed).toBe(4);
            expect(metrics.test.averageStepTime).toBe(75); // (100+0+200+0)/4
            expect(metrics.test.successRate).toBe(0.5); // 2 success out of 4
        });

        it("should calculate metrics correctly with only failures", () => {
            registry.recordRoutineFailure("test");
            registry.recordRoutineFailure("test");

            const metrics = registry.getNavigatorMetrics();
            
            expect(metrics.test.routinesProcessed).toBe(2);
            expect(metrics.test.averageStepTime).toBe(0);
            expect(metrics.test.successRate).toBe(0.0);
        });

        it("should round success rate to 3 decimal places", () => {
            // Create a case where success rate is not a clean decimal
            registry.recordRoutineSuccess("test", 100);
            registry.recordRoutineSuccess("test", 100);
            registry.recordRoutineFailure("test");

            const metrics = registry.getNavigatorMetrics();
            
            expect(metrics.test.successRate).toBe(0.667); // 2/3 rounded to 3 decimals
        });

        it("should track navigator usage on getNavigator calls", () => {
            // Initially no usage tracked
            let metrics = registry.getNavigatorMetrics();
            expect(metrics.test.routinesProcessed).toBe(0);

            // Getting navigator should not increment processed count but updates usage
            registry.getNavigator("test");
            
            metrics = registry.getNavigatorMetrics();
            expect(metrics.test.routinesProcessed).toBe(0); // Still 0, just usage tracked
        });

        it("should track navigator usage on detectNavigatorType calls", () => {
            const routineDefinition = { type: "test-routine" };
            
            const detectedType = registry.detectNavigatorType(routineDefinition);
            expect(detectedType).toBe("test");
            
            // Should have tracked usage but not execution
            const metrics = registry.getNavigatorMetrics();
            expect(metrics.test.routinesProcessed).toBe(0);
        });
    });

    describe("Multiple Navigators", () => {
        beforeEach(() => {
            registry.registerNavigator(new MockNavigator("nav1", "1.0.0"));
            registry.registerNavigator(new MockNavigator("nav2", "2.0.0"));
        });

        it("should track metrics independently for different navigators", () => {
            registry.recordRoutineSuccess("nav1", 100);
            registry.recordRoutineSuccess("nav1", 200);
            registry.recordRoutineFailure("nav2");
            registry.recordRoutineSuccess("nav2", 300);

            const metrics = registry.getNavigatorMetrics();
            
            expect(metrics.nav1.routinesProcessed).toBe(2);
            expect(metrics.nav1.successRate).toBe(1.0);
            expect(metrics.nav1.averageStepTime).toBe(150);
            
            expect(metrics.nav2.routinesProcessed).toBe(2);
            expect(metrics.nav2.successRate).toBe(0.5);
            expect(metrics.nav2.averageStepTime).toBe(150); // (0+300)/2
        });

        it("should return metrics for all registered navigators", () => {
            const metrics = registry.getNavigatorMetrics();
            
            expect(Object.keys(metrics)).toHaveLength(3); // nav1, nav2, plus native from initializeDefaultNavigators
            expect(metrics).toHaveProperty("nav1");
            expect(metrics).toHaveProperty("nav2");
        });
    });

    describe("Edge Cases", () => {
        it("should handle metrics for navigator without stats gracefully", () => {
            // Register navigator but don't initialize stats manually
            const navigator = new MockNavigator("orphan");
            registry["navigators"].set("orphan", navigator); // Bypass registerNavigator
            
            const metrics = registry.getNavigatorMetrics();
            
            expect(metrics.orphan).toEqual({
                type: "orphan",
                version: "1.0.0",
                routinesProcessed: 0,
                averageStepTime: 0,
                successRate: 1.0,
            });
        });

        it("should handle recording metrics for nonexistent navigator", () => {
            // Should not throw error
            expect(() => {
                registry.recordRoutineSuccess("nonexistent", 100);
                registry.recordRoutineFailure("nonexistent");
            }).not.toThrow();
        });

        it("should overwrite navigator and preserve/reset stats", () => {
            const nav1 = new MockNavigator("test", "1.0.0");
            registry.registerNavigator(nav1);
            
            registry.recordRoutineSuccess("test", 100);
            
            // Overwrite with same type
            const nav2 = new MockNavigator("test", "2.0.0");
            registry.registerNavigator(nav2);
            
            const metrics = registry.getNavigatorMetrics();
            expect(metrics.test.version).toBe("2.0.0");
            expect(metrics.test.routinesProcessed).toBe(1); // Stats preserved
        });
    });
});

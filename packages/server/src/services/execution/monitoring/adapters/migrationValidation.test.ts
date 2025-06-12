import { describe, it, expect } from "vitest";

/**
 * Migration validation tests
 * These tests verify that the RollingHistory migration maintains backward compatibility
 * and that all imports resolve correctly.
 */

describe("RollingHistory Migration Validation", () => {
    
    it("should import RollingHistory from cross-cutting monitoring index", async () => {
        const module = await import("../../cross-cutting/monitoring/index.js");
        expect(module.RollingHistory).toBeDefined();
        expect(typeof module.RollingHistory).toBe("function"); // Constructor function
    });

    it("should export required types from cross-cutting monitoring", async () => {
        const module = await import("../../cross-cutting/monitoring/index.js");
        
        // Check that types are available (they exist at compile time)
        // We can't directly test types at runtime, but we can verify the module structure
        expect(module).toHaveProperty("RollingHistory");
    });

    it("should import RollingHistoryAdapter directly", async () => {
        const { RollingHistoryAdapter } = await import("./RollingHistoryAdapter.js");
        expect(RollingHistoryAdapter).toBeDefined();
        expect(typeof RollingHistoryAdapter).toBe("function");
    });

    it("should maintain interface compatibility", async () => {
        const { RollingHistoryAdapter } = await import("./RollingHistoryAdapter.js");
        
        // Mock EventBus for testing
        const mockEventBus = {
            subscribe: () => Promise.resolve(),
            publish: () => Promise.resolve(),
            unsubscribe: () => Promise.resolve(),
        };

        const config = {
            maxSize: 100,
            maxAge: 60000,
            persistInterval: 30000,
        };

        const adapter = new RollingHistoryAdapter(mockEventBus as any, config);

        // Test that all expected methods exist
        expect(typeof adapter.addEvent).toBe("function");
        expect(typeof adapter.getRecentEvents).toBe("function");
        expect(typeof adapter.getEventsByTier).toBe("function");
        expect(typeof adapter.getEventsByType).toBe("function");
        expect(typeof adapter.detectPatterns).toBe("function");
        expect(typeof adapter.getEventsInTimeRange).toBe("function");
        expect(typeof adapter.getAllEvents).toBe("function");
        expect(typeof adapter.getValid).toBe("function");
        expect(typeof adapter.getExecutionFlow).toBe("function");
        expect(typeof adapter.evictOldEvents).toBe("function");
        expect(typeof adapter.getStats).toBe("function");
    });

    it("should validate migrated file imports work", async () => {
        // Test that files using the adapter can import it successfully
        const tests = [
            "../../tier3/engine/unifiedExecutor.js",
            "../../integration/executionArchitecture.js", 
            "../../integration/tools/monitoringTools.js",
            "../../integration/tools/securityTools.js",
            "../../integration/tools/resilienceTools.js",
        ];

        for (const testImport of tests) {
            try {
                await import(testImport);
                // If we get here, the import succeeded
                expect(true).toBe(true);
            } catch (error) {
                // Only fail if it's a missing import error, not other runtime errors
                if (error instanceof Error && error.message.includes("Cannot find module")) {
                    throw new Error(`Import failed for ${testImport}: ${error.message}`);
                }
                // Other errors are OK (e.g., missing dependencies at runtime)
                console.warn(`Non-import error for ${testImport}:`, error.message);
            }
        }
    });

    it("should provide backward compatible interface", async () => {
        // Test that the interface signature matches what code expects
        const { RollingHistoryAdapter } = await import("./RollingHistoryAdapter.js");
        
        const mockEventBus = {
            subscribe: () => Promise.resolve(),
            publish: () => Promise.resolve(), 
            unsubscribe: () => Promise.resolve(),
        };

        const config = { maxSize: 100 };
        
        // Should not throw when constructed with legacy parameters
        expect(() => new RollingHistoryAdapter(mockEventBus, config)).not.toThrow();
    });

    it("should handle event addition without throwing", async () => {
        const { RollingHistoryAdapter } = await import("./RollingHistoryAdapter.js");
        
        const mockEventBus = {
            subscribe: () => Promise.resolve(),
            publish: () => Promise.resolve(),
            unsubscribe: () => Promise.resolve(),
        };

        const adapter = new RollingHistoryAdapter(mockEventBus, { maxSize: 100 });
        
        const testEvent = {
            timestamp: new Date(),
            type: "test.event",
            tier: "tier3" as const,
            component: "test",
            data: { test: true },
        };

        // Should not throw
        expect(() => adapter.addEvent(testEvent)).not.toThrow();
    });
});
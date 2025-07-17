/**
 * Diagnostic Test for Execution Framework
 * 
 * Minimal test to identify specific issues
 */

import { describe, it, expect } from "vitest";
import "./setup.js"; // Initialize execution test environment

describe("Execution Framework Diagnostics", () => {
    it("should verify environment is set up", async () => {
        // Test basic imports
        const { getEventBus } = await import("../../services/events/eventBus.js");
        expect(getEventBus).toBeDefined();
        expect(typeof getEventBus).toBe("function");
        
        // Test event bus is available
        const eventBus = getEventBus();
        expect(eventBus).toBeDefined();
        
        // Test DbProvider
        const { DbProvider } = await import("../../db/provider.js");
        expect(DbProvider).toBeDefined();
        expect(typeof DbProvider.get).toBe("function");
        
        // This will throw if not initialized - that's expected in test environment
        try {
            const db = DbProvider.get();
            expect(db).toBeDefined();
        } catch (error) {
            console.log("Expected: DbProvider not initialized in test - this is normal");
        }
    });
    
    it("should verify schema registries can be imported", async () => {
        const { RoutineSchemaRegistry } = await import("./schemas/routines/index.js");
        const { AgentSchemaRegistry } = await import("./schemas/agents/index.js");
        const { SwarmSchemaRegistry } = await import("./schemas/swarms/index.js");
        
        expect(RoutineSchemaRegistry).toBeDefined();
        expect(AgentSchemaRegistry).toBeDefined();
        expect(SwarmSchemaRegistry).toBeDefined();
        
        // Test static methods exist
        expect(typeof RoutineSchemaRegistry.list).toBe("function");
        expect(typeof AgentSchemaRegistry.list).toBe("function");
        expect(typeof SwarmSchemaRegistry.list).toBe("function");
    });
    
    it("should verify factories can be imported", async () => {
        const { RoutineFactory } = await import("./factories/routine/RoutineFactory.js");
        const { AgentFactory } = await import("./factories/agent/AgentFactory.js");
        const { SwarmFactory } = await import("./factories/swarm/SwarmFactory.js");
        
        expect(RoutineFactory).toBeDefined();
        expect(AgentFactory).toBeDefined();
        expect(SwarmFactory).toBeDefined();
        
        // Test instantiation
        expect(() => new RoutineFactory()).not.toThrow();
        expect(() => new AgentFactory()).not.toThrow();
        expect(() => new SwarmFactory()).not.toThrow();
    });
    
    it("should verify custom assertions are available", async () => {
        const { initializeExecutionAssertions } = await import("./assertions/index.js");
        expect(initializeExecutionAssertions).toBeDefined();
        
        // Initialize assertions
        initializeExecutionAssertions();
        
        // Test that custom matchers are added
        const testMap = new Map();
        testMap.set("key", "value");
        
        // These will throw if assertions not properly initialized
        expect(testMap).toHaveKey("key");
        expect(testMap).toHaveValue("key", "value");
    });
});

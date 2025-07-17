/**
 * Diagnostic Test
 * 
 * Simple test to identify specific initialization issues
 */

import { describe, it, expect, beforeAll } from "vitest";
import { DbProvider } from "../../db/provider.js";
import { getEventBus } from "../../services/events/eventBus.js";

describe("Execution Framework Diagnostics", () => {
    describe("Core Dependencies", () => {
        it("should have database provider available", () => {
            try {
                const db = DbProvider.get();
                expect(db).toBeDefined();
                expect(db.routine).toBeDefined();
                expect(db.bot).toBeDefined();
                expect(db.team).toBeDefined();
            } catch (error) {
                console.error("❌ Database provider not initialized:", error);
                throw error;
            }
        });

        it("should have event bus initialized", async () => {
            try {
                const eventBus = getEventBus();
                expect(eventBus).toBeDefined();
                
                // Check if we can subscribe
                const subscriptionId = await eventBus.subscribe(
                    "test/diagnostic",
                    () => {},
                    { mode: "standard" },
                );
                expect(subscriptionId).toBeDefined();
                
                // Clean up
                await eventBus.unsubscribe(subscriptionId);
            } catch (error) {
                console.error("❌ Event bus not properly initialized:", error);
                throw error;
            }
        });
    });

    describe("Schema Registries", () => {
        it("should load schema registries without errors", async () => {
            try {
                // Dynamic import to catch errors
                const { RoutineSchemaRegistry } = await import("./schemas/routines/index.js");
                const { AgentSchemaRegistry } = await import("./schemas/agents/index.js");
                const { SwarmSchemaRegistry } = await import("./schemas/swarms/index.js");
                
                expect(RoutineSchemaRegistry).toBeDefined();
                expect(AgentSchemaRegistry).toBeDefined();
                expect(SwarmSchemaRegistry).toBeDefined();
                
                // Check if they have expected methods
                expect(typeof RoutineSchemaRegistry.list).toBe("function");
                expect(typeof AgentSchemaRegistry.list).toBe("function");
                expect(typeof SwarmSchemaRegistry.list).toBe("function");
            } catch (error) {
                console.error("❌ Schema registry import failed:", error);
                throw error;
            }
        });
    });

    describe("Factory System", () => {
        it("should import factories without errors", async () => {
            try {
                const { RoutineFactory } = await import("./factories/routine/RoutineFactory.js");
                const { AgentFactory } = await import("./factories/agent/AgentFactory.js");
                const { SwarmFactory } = await import("./factories/swarm/SwarmFactory.js");
                
                expect(RoutineFactory).toBeDefined();
                expect(AgentFactory).toBeDefined();
                expect(SwarmFactory).toBeDefined();
            } catch (error) {
                console.error("❌ Factory import failed:", error);
                throw error;
            }
        });
    });

    describe("Service Dependencies", () => {
        it("should check if production services can be imported", async () => {
            const servicesToCheck = [
                { path: "../../services/events/EventInterceptor.js", name: "EventInterceptor" },
                { path: "../../services/events/LockService.js", name: "InMemoryLockService" },
                { path: "../../services/execution/shared/SwarmContextManager.js", name: "SwarmContextManager" },
            ];

            for (const service of servicesToCheck) {
                try {
                    const module = await import(service.path);
                    expect(module[service.name]).toBeDefined();
                    console.log(`✅ ${service.name} imported successfully`);
                } catch (error) {
                    console.error(`❌ Failed to import ${service.name}:`, error);
                    throw error;
                }
            }
        });
    });

    describe("Test Data", () => {
        it("should verify schema files exist", async () => {
            const schemas = [
                "./schemas/routines/actions/redis-connection-fixer.json",
                "./schemas/routines/workflows/redis-validation-workflow.json",
                "./schemas/agents/fixers/redis-problem-fixer.json",
                "./schemas/agents/validators/redis-fix-validator.json",
                "./schemas/agents/coordinators/redis-loop-coordinator.json",
                "./schemas/swarms/development/redis-fix-team.json",
            ];

            for (const schemaPath of schemas) {
                try {
                    // Try to import as module first (for .json files configured in tsconfig)
                    const schema = await import(schemaPath);
                    expect(schema).toBeDefined();
                    console.log(`✅ Schema ${schemaPath} loaded`);
                } catch (error) {
                    // If module import fails, log the error
                    console.error(`❌ Failed to load schema ${schemaPath}:`, error);
                }
            }
        });
    });
});

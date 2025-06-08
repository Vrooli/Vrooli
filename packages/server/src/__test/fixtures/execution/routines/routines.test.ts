/**
 * Example test file demonstrating type-safe usage of routine fixtures
 */

import { describe, it, expect } from "vitest";
import { ResourceSubType } from "@vrooli/shared";
import {
    getAllRoutines,
    getRoutineById,
    getRoutinesByStrategy,
    getRoutinesByCategory,
    getRoutinesByResourceSubType,
    getRoutinesForAgent,
    getRoutineStats,
    SEQUENTIAL_ROUTINES,
    BPMN_ROUTINES,
    SECURITY_ROUTINES,
    MEDICAL_ROUTINES,
    AGENT_ROUTINE_MAP,
} from "./index.js";
import type { RoutineFixture, ExecutionStrategy } from "./types.js";

describe("Routine Fixtures Type Safety", () => {
    it("should provide type-safe access to individual routines", () => {
        // Direct access with full type safety
        const hipaaRoutine = SECURITY_ROUTINES.HIPAA_COMPLIANCE_CHECK;
        expect(hipaaRoutine.id).toBe("hipaa_compliance_check");
        expect(hipaaRoutine.resourceSubType).toBe(ResourceSubType.RoutineCode);
        
        // TypeScript knows the exact shape
        const { config, name, description } = hipaaRoutine;
        expect(config.__version).toBe("1.0");
        expect(name).toBeDefined();
        expect(description).toBeDefined();
    });

    it("should return typed arrays from helper functions", () => {
        // getAllRoutines returns RoutineFixture[]
        const allRoutines: RoutineFixture[] = getAllRoutines();
        expect(allRoutines.length).toBeGreaterThan(0);
        
        // Every routine has required properties
        allRoutines.forEach(routine => {
            expect(routine).toHaveProperty("id");
            expect(routine).toHaveProperty("name");
            expect(routine).toHaveProperty("config");
            expect(routine).toHaveProperty("resourceSubType");
        });
    });

    it("should filter routines by strategy with type safety", () => {
        // Type-safe strategy parameter
        const strategies: ExecutionStrategy[] = ["reasoning", "deterministic", "conversational", "auto"];
        
        strategies.forEach(strategy => {
            const routines = getRoutinesByStrategy(strategy);
            routines.forEach(routine => {
                expect(routine.config.executionStrategy).toBe(strategy);
            });
        });
    });

    it("should filter routines by category", () => {
        // Category is strictly typed
        const securityRoutines = getRoutinesByCategory("security");
        expect(securityRoutines.length).toBe(4);
        
        const medicalRoutines = getRoutinesByCategory("medical");
        expect(medicalRoutines.length).toBe(1);
        
        const bpmnRoutines = getRoutinesByCategory("bpmn");
        expect(bpmnRoutines.length).toBe(4);
    });

    it("should find routines by ID with proper undefined handling", () => {
        // getRoutineById returns RoutineFixture | undefined
        const existingRoutine = getRoutineById("hipaa_compliance_check");
        expect(existingRoutine).toBeDefined();
        expect(existingRoutine?.name).toBe("HIPAA Compliance Check");
        
        const nonExistentRoutine = getRoutineById("non_existent_id");
        expect(nonExistentRoutine).toBeUndefined();
    });

    it("should map agents to routines with type safety", () => {
        // Agent IDs are keys in the mapping
        const agentId = "hipaa_compliance_monitor";
        const routineIds = AGENT_ROUTINE_MAP[agentId];
        expect(routineIds).toBeDefined();
        expect(routineIds.length).toBeGreaterThan(0);
        
        // getRoutinesForAgent filters out undefined values
        const routines = getRoutinesForAgent(agentId);
        routines.forEach(routine => {
            expect(routine).toBeDefined();
            expect(routine.id).toBeTruthy();
        });
    });

    it("should provide typed routine statistics", () => {
        const stats = getRoutineStats();
        
        // Stats object has the exact shape defined in types
        expect(stats).toHaveProperty("total");
        expect(stats).toHaveProperty("sequential");
        expect(stats).toHaveProperty("bpmn");
        expect(stats).toHaveProperty("byCategory");
        expect(stats).toHaveProperty("byStrategy");
        expect(stats).toHaveProperty("byResourceType");
        
        // Nested properties are also typed
        expect(stats.byCategory.security).toBe(4);
        expect(stats.byCategory.medical).toBe(1);
        expect(stats.byStrategy.reasoning).toBeGreaterThan(0);
        expect(stats.byStrategy.deterministic).toBeGreaterThan(0);
    });

    it("should filter by resource subtype", () => {
        // ResourceSubType enum provides type safety
        const actionRoutines = getRoutinesByResourceSubType(ResourceSubType.RoutineInternalAction);
        expect(actionRoutines.length).toBeGreaterThan(0);
        
        const generateRoutines = getRoutinesByResourceSubType(ResourceSubType.RoutineGenerate);
        expect(generateRoutines.length).toBeGreaterThan(0);
        
        const multiStepRoutines = getRoutinesByResourceSubType(ResourceSubType.RoutineMultiStep);
        expect(multiStepRoutines.length).toBe(4); // All BPMN workflows
    });

    it("should ensure BPMN workflows have proper configuration", () => {
        const bpmnWorkflows = Object.values(BPMN_ROUTINES);
        
        bpmnWorkflows.forEach(workflow => {
            expect(workflow.resourceSubType).toBe(ResourceSubType.RoutineMultiStep);
            expect(workflow.config.graph).toBeDefined();
            expect(workflow.config.graph?.__type).toBe("BPMN-2.0");
            
            // Check for activity mappings
            const graph = workflow.config.graph;
            if (graph && "schema" in graph) {
                expect(graph.schema.activityMap).toBeDefined();
                expect(graph.schema.rootContext).toBeDefined();
            }
        });
    });
});

// Example of using routine fixtures in actual tests
describe("Example: Testing with Routine Fixtures", () => {
    it("should execute a security routine", async () => {
        const routine = SECURITY_ROUTINES.HIPAA_COMPLIANCE_CHECK;
        
        // Use the routine configuration to set up test
        const { formInput, formOutput, callDataCode } = routine.config;
        
        expect(formInput?.schema.elements).toBeDefined();
        expect(formOutput?.schema.elements).toBeDefined();
        expect(callDataCode?.schema.inputTemplate).toBeDefined();
        
        // The routine is fully typed, so TypeScript helps catch errors
        // routine.config.callDataCode.schema.inputTemplate = null; // ❌ TypeScript error
    });
});
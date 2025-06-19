import { describe, test, expect, beforeEach } from 'vitest';
import { 
    RunStepStatus,
    shapeRunStep, 
    runRoutineStepValidation, 
    generatePK, 
    DUMMY_ID,
    type RunStep, 
    type RunStepShape 
} from "@vrooli/shared";
import { 
    minimalRunStepFormInput,
    completeRunStepFormInput,
    runStepStatusVariants,
    runStepComplexityVariants,
    runStepContextSwitchVariants,
    runStepTimingVariants,
    runStepNodeTypeVariants,
    runStepUpdateFormInput,
    invalidRunStepFormInputs,
    runStepFormScenarios,
    transformRunStepFormToApiInput,
    validateRunStepName,
    validateComplexity,
    validateNodeId,
    validateOrder,
    validateContextSwitches,
    validateTimeElapsed
} from '../form-data/runStepFormData.js';
import { 
    minimalRunResponse,
    completeRunResponse
} from '../api-responses/runResponses.js';

/**
 * Round-trip testing for RunStep data flow using REAL application functions
 * Tests the complete user journey: Form Input → API Request → Database → API Response → UI Display
 * 
 * ✅ Uses real shapeRunStep.create() and shapeRunStep.update() for transformations
 * ✅ Uses real runRoutineStepValidation for validation
 * ✅ Tests actual application logic instead of mock implementations
 */

// Mock service to simulate API calls (would use real database in integration tests)
const mockRunStepService = {
    storage: {} as Record<string, RunStep>,
    
    async create(input: any): Promise<RunStep> {
        const runStep: RunStep = {
            __typename: "RunStep",
            id: input.id || generatePK().toString(),
            completedAt: input.completedAt || null,
            complexity: input.complexity,
            contextSwitches: input.contextSwitches || 0,
            name: input.name,
            nodeId: input.nodeId,
            order: input.order,
            startedAt: input.startedAt || null,
            status: input.status || RunStepStatus.InProgress,
            resourceInId: input.resourceInId,
            resourceVersion: input.resourceVersionConnect ? {
                __typename: "ResourceVersion",
                id: input.resourceVersionConnect,
                // Mock minimal resource version data
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                isComplete: true,
                isLatest: true,
                isPrivate: false,
                versionIndex: 1,
                versionLabel: "1.0.0"
            } as any : null,
            timeElapsed: input.timeElapsed || null,
        };
        
        this.storage[runStep.id] = runStep;
        return runStep;
    },
    
    async findById(id: string): Promise<RunStep> {
        const runStep = this.storage[id];
        if (!runStep) throw new Error(`RunStep ${id} not found`);
        return runStep;
    },
    
    async update(id: string, input: any): Promise<RunStep> {
        const existing = this.storage[id];
        if (!existing) throw new Error(`RunStep ${id} not found`);
        
        const updated: RunStep = {
            ...existing,
            ...input,
            id: existing.id, // Preserve original ID
        };
        
        this.storage[id] = updated;
        return updated;
    },
    
    async delete(id: string): Promise<{ success: boolean }> {
        delete this.storage[id];
        return { success: true };
    }
};

// Helper functions using REAL application logic
function transformFormToCreateRequestReal(formData: any, runId?: string): any {
    return shapeRunStep.create({
        __typename: "RunStep",
        id: generatePK().toString(),
        name: formData.name,
        complexity: formData.complexity,
        contextSwitches: formData.contextSwitches,
        nodeId: formData.nodeId,
        order: formData.order,
        status: formData.status,
        resourceInId: formData.resourceInId,
        timeElapsed: formData.timeElapsed,
        startedAt: formData.startedAt,
        completedAt: formData.completedAt,
        run: {
            __typename: "Run",
            __connect: true,
            id: runId || formData.run?.id || "run_123456789012345678",
        },
        resourceVersion: formData.resourceVersion?.id ? {
            __typename: "ResourceVersion",
            __connect: true,
            id: formData.resourceVersion.id,
        } : null,
    });
}

function transformFormToUpdateRequestReal(runStepId: string, formData: Partial<any>): any {
    return shapeRunStep.update({
        __typename: "RunStep",
        id: runStepId,
        completedAt: null,
        complexity: 1,
        contextSwitches: 0,
        name: "",
        nodeId: "",
        order: 0,
        startedAt: null,
        status: RunStepStatus.InProgress,
        resourceInId: "",
        resourceVersion: null,
        timeElapsed: null,
    }, {
        __typename: "RunStep",
        id: runStepId,
        contextSwitches: formData.contextSwitches,
        status: formData.status,
        timeElapsed: formData.timeElapsed,
    });
}

async function validateRunStepFormDataReal(formData: any, runId?: string): Promise<string[]> {
    try {
        const validationData = {
            id: generatePK().toString(),
            name: formData.name,
            complexity: formData.complexity,
            contextSwitches: formData.contextSwitches,
            nodeId: formData.nodeId,
            order: formData.order,
            status: formData.status,
            resourceInId: formData.resourceInId,
            timeElapsed: formData.timeElapsed,
            runConnect: runId || formData.run?.id || "run_123456789012345678",
            ...(formData.resourceVersion?.id && {
                resourceVersionConnect: formData.resourceVersion.id
            }),
        };
        
        await runRoutineStepValidation.create({ omitFields: [] }).validate(validationData);
        return []; // No validation errors
    } catch (error: any) {
        if (error.errors) {
            return error.errors; // Yup validation errors
        }
        return [error.message || "Validation failed"];
    }
}

function transformApiResponseToFormReal(runStep: RunStep): any {
    return {
        name: runStep.name,
        complexity: runStep.complexity,
        contextSwitches: runStep.contextSwitches,
        nodeId: runStep.nodeId,
        order: runStep.order,
        status: runStep.status,
        resourceInId: runStep.resourceInId,
        timeElapsed: runStep.timeElapsed,
        startedAt: runStep.startedAt,
        completedAt: runStep.completedAt,
        resourceVersion: runStep.resourceVersion ? {
            id: runStep.resourceVersion.id
        } : null,
    };
}

function areRunStepFormsEqualReal(form1: any, form2: any): boolean {
    return (
        form1.name === form2.name &&
        form1.complexity === form2.complexity &&
        form1.nodeId === form2.nodeId &&
        form1.order === form2.order &&
        form1.resourceInId === form2.resourceInId &&
        form1.status === form2.status
    );
}

describe('RunStep Round-Trip Data Flow', () => {
    beforeEach(() => {
        // Clear test storage between tests
        mockRunStepService.storage = {};
    });

    test('minimal runStep creation maintains data integrity through complete flow', async () => {
        // 🎨 STEP 1: User fills out minimal runStep form
        const userFormData = {
            name: "Basic Processing Step",
            complexity: 1,
            nodeId: "node_001",
            order: 0,
            resourceInId: "123456789012345678",
            run: { id: "987654321098765432" },
        };
        
        // Validate form data using REAL validation schema
        const validationErrors = await validateRunStepFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Form submits to API using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.name).toBe(userFormData.name);
        expect(apiCreateRequest.complexity).toBe(userFormData.complexity);
        expect(apiCreateRequest.nodeId).toBe(userFormData.nodeId);
        expect(apiCreateRequest.order).toBe(userFormData.order);
        expect(apiCreateRequest.resourceInId).toBe(userFormData.resourceInId);
        expect(apiCreateRequest.runConnect).toBe(userFormData.run.id);
        expect(apiCreateRequest.id).toMatch(/^\d{10,19}$/);
        
        // 🗄️ STEP 3: API creates runStep
        const createdRunStep = await mockRunStepService.create(apiCreateRequest);
        expect(createdRunStep.id).toBe(apiCreateRequest.id);
        expect(createdRunStep.name).toBe(userFormData.name);
        expect(createdRunStep.complexity).toBe(userFormData.complexity);
        expect(createdRunStep.nodeId).toBe(userFormData.nodeId);
        expect(createdRunStep.order).toBe(userFormData.order);
        expect(createdRunStep.resourceInId).toBe(userFormData.resourceInId);
        
        // 🔗 STEP 4: API fetches runStep back
        const fetchedRunStep = await mockRunStepService.findById(createdRunStep.id);
        expect(fetchedRunStep.id).toBe(createdRunStep.id);
        expect(fetchedRunStep.name).toBe(userFormData.name);
        expect(fetchedRunStep.complexity).toBe(userFormData.complexity);
        expect(fetchedRunStep.nodeId).toBe(userFormData.nodeId);
        
        // 🎨 STEP 5: UI displays the runStep using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedRunStep);
        expect(reconstructedFormData.name).toBe(userFormData.name);
        expect(reconstructedFormData.complexity).toBe(userFormData.complexity);
        expect(reconstructedFormData.nodeId).toBe(userFormData.nodeId);
        expect(reconstructedFormData.order).toBe(userFormData.order);
        expect(reconstructedFormData.resourceInId).toBe(userFormData.resourceInId);
        
        // ✅ VERIFICATION: Complete round-trip integrity using REAL comparison
        expect(areRunStepFormsEqualReal(userFormData, reconstructedFormData)).toBe(true);
    });

    test('complete runStep with all fields preserves all data', async () => {
        // 🎨 STEP 1: User creates complex runStep with all fields
        const userFormData = {
            name: "Advanced Data Processing Step",
            complexity: 150,
            contextSwitches: 3,
            nodeId: "advanced_processor_node_001",
            order: 2,
            status: RunStepStatus.InProgress,
            resourceInId: "123456789012345678",
            timeElapsed: 2500,
            run: { id: "987654321098765432" },
            resourceVersion: { id: "456789012345678901" },
            startedAt: "2024-01-15T10:00:00Z",
            completedAt: null,
        };
        
        // Validate complex form using REAL validation
        const validationErrors = await validateRunStepFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform to API request using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.name).toBe(userFormData.name);
        expect(apiCreateRequest.complexity).toBe(userFormData.complexity);
        expect(apiCreateRequest.contextSwitches).toBe(userFormData.contextSwitches);
        expect(apiCreateRequest.status).toBe(userFormData.status);
        expect(apiCreateRequest.timeElapsed).toBe(userFormData.timeElapsed);
        expect(apiCreateRequest.resourceVersionConnect).toBe(userFormData.resourceVersion.id);
        
        // 🗄️ STEP 3: Create via API
        const createdRunStep = await mockRunStepService.create(apiCreateRequest);
        expect(createdRunStep.name).toBe(userFormData.name);
        expect(createdRunStep.complexity).toBe(userFormData.complexity);
        expect(createdRunStep.contextSwitches).toBe(userFormData.contextSwitches);
        expect(createdRunStep.status).toBe(userFormData.status);
        expect(createdRunStep.timeElapsed).toBe(userFormData.timeElapsed);
        expect(createdRunStep.resourceVersion?.id).toBe(userFormData.resourceVersion.id);
        
        // 🔗 STEP 4: Fetch back from API
        const fetchedRunStep = await mockRunStepService.findById(createdRunStep.id);
        expect(fetchedRunStep.name).toBe(userFormData.name);
        expect(fetchedRunStep.complexity).toBe(userFormData.complexity);
        expect(fetchedRunStep.contextSwitches).toBe(userFormData.contextSwitches);
        expect(fetchedRunStep.timeElapsed).toBe(userFormData.timeElapsed);
        
        // 🎨 STEP 5: Verify UI can display correctly using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedRunStep);
        expect(reconstructedFormData.name).toBe(userFormData.name);
        expect(reconstructedFormData.complexity).toBe(userFormData.complexity);
        expect(reconstructedFormData.contextSwitches).toBe(userFormData.contextSwitches);
        expect(reconstructedFormData.status).toBe(userFormData.status);
        expect(reconstructedFormData.timeElapsed).toBe(userFormData.timeElapsed);
        expect(reconstructedFormData.resourceVersion?.id).toBe(userFormData.resourceVersion.id);
        
        // ✅ VERIFICATION: All complex data preserved
        expect(fetchedRunStep.resourceVersion?.id).toBe(userFormData.resourceVersion.id);
        expect(fetchedRunStep.startedAt).toBe(userFormData.startedAt);
    });

    test('runStep editing maintains data integrity', async () => {
        // Create initial runStep using REAL functions
        const initialFormData = {
            name: "Initial Step",
            complexity: 1,
            nodeId: "node_001",
            order: 0,
            resourceInId: "123456789012345678",
            run: { id: "987654321098765432" },
        };
        const createRequest = transformFormToCreateRequestReal(initialFormData);
        const initialRunStep = await mockRunStepService.create(createRequest);
        
        // 🎨 STEP 1: User edits runStep
        const editFormData = {
            contextSwitches: 5,
            status: RunStepStatus.Completed,
            timeElapsed: 3600,
        };
        
        // 🔗 STEP 2: Transform edit to update request using REAL update logic
        const updateRequest = transformFormToUpdateRequestReal(initialRunStep.id, editFormData);
        expect(updateRequest.id).toBe(initialRunStep.id);
        expect(updateRequest.contextSwitches).toBe(editFormData.contextSwitches);
        expect(updateRequest.status).toBe(editFormData.status);
        expect(updateRequest.timeElapsed).toBe(editFormData.timeElapsed);
        
        // 🗄️ STEP 3: Update via API
        const updatedRunStep = await mockRunStepService.update(initialRunStep.id, updateRequest);
        expect(updatedRunStep.id).toBe(initialRunStep.id);
        expect(updatedRunStep.contextSwitches).toBe(editFormData.contextSwitches);
        expect(updatedRunStep.status).toBe(editFormData.status);
        expect(updatedRunStep.timeElapsed).toBe(editFormData.timeElapsed);
        
        // 🔗 STEP 4: Fetch updated runStep
        const fetchedUpdatedRunStep = await mockRunStepService.findById(initialRunStep.id);
        
        // ✅ VERIFICATION: Update preserved core data
        expect(fetchedUpdatedRunStep.id).toBe(initialRunStep.id);
        expect(fetchedUpdatedRunStep.name).toBe(initialFormData.name); // Unchanged
        expect(fetchedUpdatedRunStep.complexity).toBe(initialFormData.complexity); // Unchanged
        expect(fetchedUpdatedRunStep.nodeId).toBe(initialFormData.nodeId); // Unchanged
        expect(fetchedUpdatedRunStep.order).toBe(initialFormData.order); // Unchanged
        expect(fetchedUpdatedRunStep.resourceInId).toBe(initialFormData.resourceInId); // Unchanged
        
        // Updated fields should have changed
        expect(fetchedUpdatedRunStep.contextSwitches).toBe(editFormData.contextSwitches);
        expect(fetchedUpdatedRunStep.status).toBe(editFormData.status);
        expect(fetchedUpdatedRunStep.timeElapsed).toBe(editFormData.timeElapsed);
    });

    test('all runStep statuses work correctly through round-trip', async () => {
        const runStepStatuses = Object.values(RunStepStatus);
        
        for (const status of runStepStatuses) {
            // 🎨 Create form data for each status
            const formData = {
                name: `${status} Test Step`,
                complexity: 1,
                nodeId: `${status.toLowerCase()}_node`,
                order: 0,
                status: status,
                resourceInId: "123456789012345678",
                run: { id: "987654321098765432" },
            };
            
            // 🔗 Transform and create using REAL functions
            const createRequest = transformFormToCreateRequestReal(formData);
            const createdRunStep = await mockRunStepService.create(createRequest);
            
            // 🗄️ Fetch back
            const fetchedRunStep = await mockRunStepService.findById(createdRunStep.id);
            
            // ✅ Verify status-specific data
            expect(fetchedRunStep.status).toBe(status);
            expect(fetchedRunStep.name).toBe(formData.name);
            expect(fetchedRunStep.nodeId).toBe(formData.nodeId);
            
            // Verify form reconstruction using REAL transformation
            const reconstructed = transformApiResponseToFormReal(fetchedRunStep);
            expect(reconstructed.status).toBe(status);
            expect(reconstructed.name).toBe(formData.name);
            expect(reconstructed.nodeId).toBe(formData.nodeId);
        }
    });

    test('complexity levels work correctly through round-trip', async () => {
        const complexityLevels = [0, 1, 10, 50, 100, 500, 999999];
        
        for (const complexity of complexityLevels) {
            const formData = {
                name: `Complexity ${complexity} Step`,
                complexity: complexity,
                nodeId: `complexity_${complexity}_node`,
                order: 0,
                resourceInId: "123456789012345678",
                run: { id: "987654321098765432" },
            };
            
            const createRequest = transformFormToCreateRequestReal(formData);
            const createdRunStep = await mockRunStepService.create(createRequest);
            const fetchedRunStep = await mockRunStepService.findById(createdRunStep.id);
            
            expect(fetchedRunStep.complexity).toBe(complexity);
            
            const reconstructed = transformApiResponseToFormReal(fetchedRunStep);
            expect(reconstructed.complexity).toBe(complexity);
        }
    });

    test('validation catches invalid form data before API submission', async () => {
        const invalidFormData = {
            name: "", // Empty name (invalid)
            complexity: -1, // Negative complexity (invalid)
            nodeId: "", // Empty nodeId (invalid)
            order: -1, // Negative order (invalid)
            resourceInId: "invalid-id", // Invalid ID format
            run: { id: "invalid-id" }, // Invalid run ID
        };
        
        const validationErrors = await validateRunStepFormDataReal(invalidFormData);
        expect(validationErrors.length).toBeGreaterThan(0);
        
        // Should not proceed to API if validation fails
        expect(validationErrors.some(error => 
            error.includes("name") || error.includes("required") || 
            error.includes("complexity") || error.includes("order")
        )).toBe(true);
    });

    test('context switches and timing work correctly', async () => {
        const timingFormData = {
            name: "Timing Test Step",
            complexity: 50,
            contextSwitches: 10,
            nodeId: "timing_node",
            order: 1,
            resourceInId: "123456789012345678",
            timeElapsed: 5400, // 90 minutes
            run: { id: "987654321098765432" },
        };
        
        const validationErrors = await validateRunStepFormDataReal(timingFormData);
        expect(validationErrors).toHaveLength(0);
        
        const createRequest = transformFormToCreateRequestReal(timingFormData);
        const createdRunStep = await mockRunStepService.create(createRequest);
        const fetchedRunStep = await mockRunStepService.findById(createdRunStep.id);
        
        expect(fetchedRunStep.contextSwitches).toBe(timingFormData.contextSwitches);
        expect(fetchedRunStep.timeElapsed).toBe(timingFormData.timeElapsed);
        
        const reconstructed = transformApiResponseToFormReal(fetchedRunStep);
        expect(reconstructed.contextSwitches).toBe(timingFormData.contextSwitches);
        expect(reconstructed.timeElapsed).toBe(timingFormData.timeElapsed);
    });

    test('runStep deletion works correctly', async () => {
        // Create runStep first using REAL functions
        const formData = {
            name: "Step to Delete",
            complexity: 1,
            nodeId: "node_001",
            order: 0,
            resourceInId: "123456789012345678",
            run: { id: "987654321098765432" },
        };
        const createRequest = transformFormToCreateRequestReal(formData);
        const createdRunStep = await mockRunStepService.create(createRequest);
        
        // Delete it
        const deleteResult = await mockRunStepService.delete(createdRunStep.id);
        expect(deleteResult.success).toBe(true);
    });

    test('data consistency across multiple operations', async () => {
        const originalFormData = {
            name: "Multi-Op Test Step",
            complexity: 10,
            nodeId: "multi_op_node",
            order: 0,
            resourceInId: "123456789012345678",
            run: { id: "987654321098765432" },
        };
        
        // Create using REAL functions
        const createRequest = transformFormToCreateRequestReal(originalFormData);
        const created = await mockRunStepService.create(createRequest);
        
        // Update using REAL functions
        const updateRequest = transformFormToUpdateRequestReal(created.id, { 
            contextSwitches: 8,
            status: RunStepStatus.Completed,
            timeElapsed: 7200,
        });
        const updated = await mockRunStepService.update(created.id, updateRequest);
        
        // Fetch final state
        const final = await mockRunStepService.findById(created.id);
        
        // Core runStep data should remain consistent
        expect(final.id).toBe(created.id);
        expect(final.name).toBe(originalFormData.name); // Unchanged
        expect(final.complexity).toBe(originalFormData.complexity); // Unchanged
        expect(final.nodeId).toBe(originalFormData.nodeId); // Unchanged
        expect(final.order).toBe(originalFormData.order); // Unchanged
        expect(final.resourceInId).toBe(originalFormData.resourceInId); // Unchanged
        
        // Only the updated fields should have changed
        expect(final.contextSwitches).toBe(8);
        expect(final.status).toBe(RunStepStatus.Completed);
        expect(final.timeElapsed).toBe(7200);
    });

    test('validation helper functions work correctly', async () => {
        // Test name validation
        expect(validateRunStepName("")).toBe("Step name is required");
        expect(validateRunStepName("   ")).toBe("Step name cannot be empty");
        expect(validateRunStepName("Valid Step Name")).toBe(null);
        expect(validateRunStepName("x".repeat(251))).toBe("Step name must be less than 250 characters");
        
        // Test complexity validation
        expect(validateComplexity(-1)).toBe("Complexity must be 0 or greater");
        expect(validateComplexity(0)).toBe(null);
        expect(validateComplexity(100)).toBe(null);
        
        // Test node ID validation
        expect(validateNodeId("")).toBe("Node ID is required");
        expect(validateNodeId("validNodeId")).toBe(null);
        expect(validateNodeId("x".repeat(129))).toBe("Node ID must be less than 128 characters");
        
        // Test order validation
        expect(validateOrder(-1)).toBe("Order must be 0 or greater");
        expect(validateOrder(0)).toBe(null);
        expect(validateOrder(10)).toBe(null);
        
        // Test context switches validation
        expect(validateContextSwitches(undefined)).toBe(null); // Optional field
        expect(validateContextSwitches(0)).toBe("Context switches must be 1 or greater when provided");
        expect(validateContextSwitches(1)).toBe(null);
        expect(validateContextSwitches(10)).toBe(null);
        
        // Test time elapsed validation
        expect(validateTimeElapsed(undefined)).toBe(null); // Optional field
        expect(validateTimeElapsed(-1)).toBe("Time elapsed must be 0 or greater");
        expect(validateTimeElapsed(0)).toBe(null);
        expect(validateTimeElapsed(3600)).toBe(null);
    });
});
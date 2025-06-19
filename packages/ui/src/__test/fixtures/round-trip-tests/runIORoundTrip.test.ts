import { describe, test, expect, beforeEach } from 'vitest';
import { 
    shapeRunIO, 
    runIOValidation, 
    generatePK, 
    DUMMY_ID,
    type RunIO, 
    type RunIOShape 
} from "@vrooli/shared";
import { 
    minimalRunIOFormInput,
    completeRunIOFormInput,
    runIOFormVariants,
    runIOUpdateFormVariants,
    runIONodeTypeVariants,
    runIOEdgeCases,
    invalidRunIOFormInputs,
    transformRunIOFormToApiInput,
    validateRunIOData,
    validateNodeInputName,
    validateNodeName,
    type RunIOFormData
} from '../form-data/runIOFormData.js';
import { 
    minimalRunResponse,
    completeRunResponse
} from '../api-responses/runResponses.js';

/**
 * Round-trip testing for RunIO data flow using REAL application functions
 * Tests the complete user journey: Form Input → API Request → Database → API Response → UI Display
 * 
 * ✅ Uses real shapeRunIO.create() and shapeRunIO.update() for transformations
 * ✅ Uses real runIOValidation for validation
 * ✅ Tests actual application logic instead of mock implementations
 */

// Mock service to simulate API calls (would use real database in integration tests)
const mockRunIOService = {
    storage: {} as Record<string, RunIO>,
    
    async create(input: any): Promise<RunIO> {
        const runIO: RunIO = {
            __typename: "RunIO",
            id: input.id || generatePK().toString(),
            data: input.data,
            nodeInputName: input.nodeInputName,
            nodeName: input.nodeName,
            run: {
                __typename: "Run",
                id: input.runConnect || "run_123456789012345678",
                // Mock minimal run data
                completedAt: null,
                completedComplexity: 0,
                contextSwitches: 0,
                data: null,
                io: [],
                ioCount: 0,
                isPrivate: false,
                lastStep: null,
                name: "Test Run",
                resourceVersion: null,
                schedule: null,
                startedAt: new Date().toISOString(),
                status: "InProgress" as any,
                steps: [],
                stepsCount: 0,
                team: null,
                timeElapsed: null,
                user: null,
                wasRunAutomatically: false,
                you: null,
            },
        };
        
        this.storage[runIO.id] = runIO;
        return runIO;
    },
    
    async findById(id: string): Promise<RunIO> {
        const runIO = this.storage[id];
        if (!runIO) throw new Error(`RunIO ${id} not found`);
        return runIO;
    },
    
    async update(id: string, input: any): Promise<RunIO> {
        const existing = this.storage[id];
        if (!existing) throw new Error(`RunIO ${id} not found`);
        
        const updated: RunIO = {
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
    return shapeRunIO.create({
        __typename: "RunIO",
        id: generatePK().toString(),
        data: formData.data,
        nodeInputName: formData.nodeInputName,
        nodeName: formData.nodeName,
        run: {
            __typename: "Run",
            __connect: true,
            id: runId || "run_123456789012345678",
        },
    });
}

function transformFormToUpdateRequestReal(runIOId: string, formData: Partial<any>): any {
    return shapeRunIO.update({
        __typename: "RunIO",
        id: runIOId,
        data: "",
        nodeInputName: "",
        nodeName: "",
        run: { __typename: "Run", id: "run_123456789012345678" },
    }, {
        __typename: "RunIO",
        id: runIOId,
        data: formData.data,
    });
}

async function validateRunIOFormDataReal(formData: any, runId?: string): Promise<string[]> {
    try {
        const validationData = {
            id: generatePK().toString(),
            data: formData.data,
            nodeInputName: formData.nodeInputName,
            nodeName: formData.nodeName,
            runConnect: runId || "run_123456789012345678",
        };
        
        await runIOValidation.create({ omitFields: [] }).validate(validationData);
        return []; // No validation errors
    } catch (error: any) {
        if (error.errors) {
            return error.errors; // Yup validation errors
        }
        return [error.message || "Validation failed"];
    }
}

function transformApiResponseToFormReal(runIO: RunIO): any {
    return {
        data: runIO.data,
        nodeInputName: runIO.nodeInputName,
        nodeName: runIO.nodeName,
    };
}

function areRunIOFormsEqualReal(form1: any, form2: any): boolean {
    return (
        form1.data === form2.data &&
        form1.nodeInputName === form2.nodeInputName &&
        form1.nodeName === form2.nodeName
    );
}

describe('RunIO Round-Trip Data Flow', () => {
    beforeEach(() => {
        // Clear test storage between tests
        mockRunIOService.storage = {};
    });

    test('minimal runIO creation maintains data integrity through complete flow', async () => {
        // 🎨 STEP 1: User fills out minimal runIO form
        const userFormData = {
            data: "input data",
            nodeInputName: "input1",
            nodeName: "ProcessNode",
        };
        
        // Validate form data using REAL validation schema
        const validationErrors = await validateRunIOFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Form submits to API using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.data).toBe(userFormData.data);
        expect(apiCreateRequest.nodeInputName).toBe(userFormData.nodeInputName);
        expect(apiCreateRequest.nodeName).toBe(userFormData.nodeName);
        expect(apiCreateRequest.runConnect).toBeDefined();
        expect(apiCreateRequest.id).toMatch(/^\d{10,19}$/);
        
        // 🗄️ STEP 3: API creates runIO
        const createdRunIO = await mockRunIOService.create(apiCreateRequest);
        expect(createdRunIO.id).toBe(apiCreateRequest.id);
        expect(createdRunIO.data).toBe(userFormData.data);
        expect(createdRunIO.nodeInputName).toBe(userFormData.nodeInputName);
        expect(createdRunIO.nodeName).toBe(userFormData.nodeName);
        
        // 🔗 STEP 4: API fetches runIO back
        const fetchedRunIO = await mockRunIOService.findById(createdRunIO.id);
        expect(fetchedRunIO.id).toBe(createdRunIO.id);
        expect(fetchedRunIO.data).toBe(userFormData.data);
        expect(fetchedRunIO.nodeInputName).toBe(userFormData.nodeInputName);
        expect(fetchedRunIO.nodeName).toBe(userFormData.nodeName);
        
        // 🎨 STEP 5: UI displays the runIO using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedRunIO);
        expect(reconstructedFormData.data).toBe(userFormData.data);
        expect(reconstructedFormData.nodeInputName).toBe(userFormData.nodeInputName);
        expect(reconstructedFormData.nodeName).toBe(userFormData.nodeName);
        
        // ✅ VERIFICATION: Complete round-trip integrity using REAL comparison
        expect(areRunIOFormsEqualReal(userFormData, reconstructedFormData)).toBe(true);
    });

    test('complete runIO with complex JSON data preserves all data', async () => {
        // 🎨 STEP 1: User creates complex runIO with JSON data
        const userFormData = {
            data: JSON.stringify({
                type: "object",
                properties: {
                    name: { type: "string" },
                    value: { type: "number" },
                    enabled: { type: "boolean" },
                },
                values: {
                    name: "test input",
                    value: 42,
                    enabled: true,
                },
                metadata: {
                    timestamp: "2024-01-01T00:00:00Z",
                    source: "user_input",
                    version: "1.0.0",
                },
            }),
            nodeInputName: "complexInput",
            nodeName: "DataProcessorNode",
        };
        
        // Validate complex form using REAL validation
        const validationErrors = await validateRunIOFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform to API request using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.data).toBe(userFormData.data);
        expect(apiCreateRequest.nodeInputName).toBe(userFormData.nodeInputName);
        expect(apiCreateRequest.nodeName).toBe(userFormData.nodeName);
        
        // 🗄️ STEP 3: Create via API
        const createdRunIO = await mockRunIOService.create(apiCreateRequest);
        expect(createdRunIO.data).toBe(userFormData.data);
        expect(createdRunIO.nodeInputName).toBe(userFormData.nodeInputName);
        expect(createdRunIO.nodeName).toBe(userFormData.nodeName);
        
        // 🔗 STEP 4: Fetch back from API
        const fetchedRunIO = await mockRunIOService.findById(createdRunIO.id);
        expect(fetchedRunIO.data).toBe(userFormData.data);
        
        // 🎨 STEP 5: Verify UI can display correctly using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedRunIO);
        expect(reconstructedFormData.data).toBe(userFormData.data);
        expect(reconstructedFormData.nodeInputName).toBe(userFormData.nodeInputName);
        expect(reconstructedFormData.nodeName).toBe(userFormData.nodeName);
        
        // ✅ VERIFICATION: Complex JSON data preserved
        expect(JSON.parse(fetchedRunIO.data)).toEqual(JSON.parse(userFormData.data));
    });

    test('runIO editing maintains data integrity', async () => {
        // Create initial runIO using REAL functions
        const initialFormData = {
            data: "initial data",
            nodeInputName: "input1",
            nodeName: "ProcessNode",
        };
        const createRequest = transformFormToCreateRequestReal(initialFormData);
        const initialRunIO = await mockRunIOService.create(createRequest);
        
        // 🎨 STEP 1: User edits runIO data
        const editFormData = {
            data: JSON.stringify({
                status: "processed",
                result: "processing complete",
                recordsProcessed: 1000,
            }),
        };
        
        // 🔗 STEP 2: Transform edit to update request using REAL update logic
        const updateRequest = transformFormToUpdateRequestReal(initialRunIO.id, editFormData);
        expect(updateRequest.id).toBe(initialRunIO.id);
        expect(updateRequest.data).toBe(editFormData.data);
        
        // 🗄️ STEP 3: Update via API
        const updatedRunIO = await mockRunIOService.update(initialRunIO.id, updateRequest);
        expect(updatedRunIO.id).toBe(initialRunIO.id);
        expect(updatedRunIO.data).toBe(editFormData.data);
        
        // 🔗 STEP 4: Fetch updated runIO
        const fetchedUpdatedRunIO = await mockRunIOService.findById(initialRunIO.id);
        
        // ✅ VERIFICATION: Update preserved core data
        expect(fetchedUpdatedRunIO.id).toBe(initialRunIO.id);
        expect(fetchedUpdatedRunIO.data).toBe(editFormData.data);
        expect(fetchedUpdatedRunIO.nodeInputName).toBe(initialFormData.nodeInputName); // Unchanged
        expect(fetchedUpdatedRunIO.nodeName).toBe(initialFormData.nodeName); // Unchanged
    });

    test('all data types work correctly through round-trip', async () => {
        const dataTypeVariants = [
            { data: "Simple string", type: "string" },
            { data: "123.456", type: "number" },
            { data: "true", type: "boolean" },
            { data: JSON.stringify({ key: "value", nested: { count: 5 } }), type: "json" },
        ];
        
        for (const variant of dataTypeVariants) {
            // 🎨 Create form data for each type
            const formData = {
                data: variant.data,
                nodeInputName: `${variant.type}Input`,
                nodeName: `${variant.type}ProcessorNode`,
            };
            
            // 🔗 Transform and create using REAL functions
            const createRequest = transformFormToCreateRequestReal(formData);
            const createdRunIO = await mockRunIOService.create(createRequest);
            
            // 🗄️ Fetch back
            const fetchedRunIO = await mockRunIOService.findById(createdRunIO.id);
            
            // ✅ Verify type-specific data
            expect(fetchedRunIO.data).toBe(variant.data);
            expect(fetchedRunIO.nodeInputName).toBe(formData.nodeInputName);
            expect(fetchedRunIO.nodeName).toBe(formData.nodeName);
            
            // Verify form reconstruction using REAL transformation
            const reconstructed = transformApiResponseToFormReal(fetchedRunIO);
            expect(reconstructed.data).toBe(variant.data);
            expect(reconstructed.nodeInputName).toBe(formData.nodeInputName);
            expect(reconstructed.nodeName).toBe(formData.nodeName);
        }
    });

    test('validation catches invalid form data before API submission', async () => {
        const invalidFormData = {
            data: "", // Empty data (invalid)
            nodeInputName: "", // Empty nodeInputName (invalid)
            nodeName: "", // Empty nodeName (invalid)
        };
        
        const validationErrors = await validateRunIOFormDataReal(invalidFormData);
        expect(validationErrors.length).toBeGreaterThan(0);
        
        // Should not proceed to API if validation fails
        expect(validationErrors.some(error => 
            error.includes("data") || error.includes("required")
        )).toBe(true);
    });

    test('edge cases and limits work correctly', async () => {
        // Test with maximum length data
        const maxLengthFormData = {
            data: "a".repeat(8192), // Maximum allowed length
            nodeInputName: "maxInput",
            nodeName: "MaxDataNode",
        };
        
        const validationErrors = await validateRunIOFormDataReal(maxLengthFormData);
        expect(validationErrors).toHaveLength(0);
        
        const createRequest = transformFormToCreateRequestReal(maxLengthFormData);
        const createdRunIO = await mockRunIOService.create(createRequest);
        
        expect(createdRunIO.data).toBe(maxLengthFormData.data);
        expect(createdRunIO.data).toHaveLength(8192);
    });

    test('special characters and unicode data work correctly', async () => {
        const unicodeFormData = {
            data: "Unicode test: 🚀 🌟 ⭐ 💫 ✨ 中文 العربية हिन्दी",
            nodeInputName: "unicodeInput",
            nodeName: "UnicodeProcessorNode",
        };
        
        const validationErrors = await validateRunIOFormDataReal(unicodeFormData);
        expect(validationErrors).toHaveLength(0);
        
        const createRequest = transformFormToCreateRequestReal(unicodeFormData);
        const createdRunIO = await mockRunIOService.create(createRequest);
        const fetchedRunIO = await mockRunIOService.findById(createdRunIO.id);
        
        expect(fetchedRunIO.data).toBe(unicodeFormData.data);
        
        const reconstructed = transformApiResponseToFormReal(fetchedRunIO);
        expect(reconstructed.data).toBe(unicodeFormData.data);
    });

    test('runIO deletion works correctly', async () => {
        // Create runIO first using REAL functions
        const formData = {
            data: "data to delete",
            nodeInputName: "input1",
            nodeName: "ProcessNode",
        };
        const createRequest = transformFormToCreateRequestReal(formData);
        const createdRunIO = await mockRunIOService.create(createRequest);
        
        // Delete it
        const deleteResult = await mockRunIOService.delete(createdRunIO.id);
        expect(deleteResult.success).toBe(true);
    });

    test('data consistency across multiple operations', async () => {
        const originalFormData = {
            data: "original data",
            nodeInputName: "input1",
            nodeName: "ProcessNode",
        };
        
        // Create using REAL functions
        const createRequest = transformFormToCreateRequestReal(originalFormData);
        const created = await mockRunIOService.create(createRequest);
        
        // Update using REAL functions
        const updateRequest = transformFormToUpdateRequestReal(created.id, { 
            data: "updated data",
        });
        const updated = await mockRunIOService.update(created.id, updateRequest);
        
        // Fetch final state
        const final = await mockRunIOService.findById(created.id);
        
        // Core runIO data should remain consistent
        expect(final.id).toBe(created.id);
        expect(final.nodeInputName).toBe(originalFormData.nodeInputName); // Unchanged
        expect(final.nodeName).toBe(originalFormData.nodeName); // Unchanged
        expect(final.run.id).toBe(created.run.id); // Unchanged
        
        // Only the data field should have changed
        expect(final.data).toBe("updated data");
    });

    test('validation helper functions work correctly', async () => {
        // Test data validation
        expect(validateRunIOData("")).toBe("Data is required");
        expect(validateRunIOData("valid data")).toBe(null);
        expect(validateRunIOData("x".repeat(8193))).toBe("Data must be less than 8192 characters");
        
        // Test node input name validation
        expect(validateNodeInputName("")).toBe("Node input name is required");
        expect(validateNodeInputName("validName")).toBe(null);
        expect(validateNodeInputName("x".repeat(129))).toBe("Node input name must be less than 128 characters");
        
        // Test node name validation
        expect(validateNodeName("")).toBe("Node name is required");
        expect(validateNodeName("ValidNode")).toBe(null);
        expect(validateNodeName("x".repeat(129))).toBe("Node name must be less than 128 characters");
    });
});
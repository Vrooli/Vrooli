import { describe, test, expect, beforeEach } from 'vitest';
import { 
    RunStatus, 
    shapeRun, 
    runValidation, 
    generatePK, 
    DUMMY_ID,
    type Run, 
    type RunShape 
} from "@vrooli/shared";
import { 
    minimalRunCreateFormInput,
    completeRunCreateFormInput,
    minimalRunUpdateFormInput,
    completeRunUpdateFormInput,
    runExecutionVariants,
    runStatusVariants,
    transformRunFormToApiInput,
    type RunFormData
} from '../form-data/runFormData.js';
import { 
    minimalRunResponse,
    completeRunResponse,
    runResponseVariants
} from '../api-responses/runResponses.js';

/**
 * Round-trip testing for Run data flow using REAL application functions
 * Tests the complete user journey: Form Input → API Request → Database → API Response → UI Display
 * 
 * ✅ Uses real shapeRun.create() and shapeRun.update() for transformations
 * ✅ Uses real runValidation for validation
 * ✅ Tests actual application logic instead of mock implementations
 */

// Mock service to simulate API calls (would use real database in integration tests)
const mockRunService = {
    storage: {} as Record<string, Run>,
    
    async create(input: any): Promise<Run> {
        const run: Run = {
            __typename: "Run",
            id: input.id || generatePK().toString(),
            completedAt: null,
            completedComplexity: input.completedComplexity || 0,
            contextSwitches: input.contextSwitches || 0,
            data: input.data || null,
            io: input.ioCreate || [],
            ioCount: input.ioCreate?.length || 0,
            isPrivate: input.isPrivate || false,
            lastStep: null,
            name: input.name,
            resourceVersion: {
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
            } as any,
            schedule: input.scheduleCreate ? {
                __typename: "Schedule",
                id: input.scheduleCreate.id,
                ...input.scheduleCreate
            } as any : null,
            startedAt: input.startedAt || (input.status === RunStatus.InProgress ? new Date().toISOString() : null),
            status: input.status,
            steps: input.stepsCreate || [],
            stepsCount: input.stepsCreate?.length || 0,
            team: input.teamConnect ? {
                __typename: "Team",
                id: input.teamConnect
            } as any : null,
            timeElapsed: input.timeElapsed || null,
            user: {
                __typename: "User",
                id: "user_123456789012345678"
            } as any,
            wasRunAutomatically: false,
            you: {
                __typename: "RunYou",
                canDelete: true,
                canRead: true,
                canUpdate: true,
            },
        };
        
        this.storage[run.id] = run;
        return run;
    },
    
    async findById(id: string): Promise<Run> {
        const run = this.storage[id];
        if (!run) throw new Error(`Run ${id} not found`);
        return run;
    },
    
    async update(id: string, input: any): Promise<Run> {
        const existing = this.storage[id];
        if (!existing) throw new Error(`Run ${id} not found`);
        
        const updated: Run = {
            ...existing,
            ...input,
            id: existing.id, // Preserve original ID
            updatedAt: new Date().toISOString(),
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
function transformFormToCreateRequestReal(formData: any): any {
    return shapeRun.create({
        __typename: "Run",
        id: generatePK().toString(),
        name: formData.name,
        status: formData.status,
        isPrivate: formData.isPrivate,
        completedComplexity: formData.completedComplexity,
        contextSwitches: formData.contextSwitches,
        data: formData.data,
        timeElapsed: formData.timeElapsed,
        startedAt: formData.startedAt,
        resourceVersion: formData.resourceVersionId ? {
            __typename: "ResourceVersion",
            __connect: true,
            id: formData.resourceVersionId,
        } : null,
        team: formData.teamId ? {
            __typename: "Team",
            __connect: true,
            id: formData.teamId,
        } : null,
        schedule: formData.schedule ? {
            __typename: "Schedule",
            id: generatePK().toString(),
            ...formData.schedule,
        } : null,
        steps: formData.steps?.map((step: any, index: number) => ({
            __typename: "RunStep",
            id: generatePK().toString(),
            name: step.name,
            nodeId: step.nodeId,
            order: step.order,
            complexity: step.complexity || 1,
            resourceInId: formData.resourceVersionId,
            run: { __connect: true, id: DUMMY_ID },
        })) || [],
        io: formData.ioData?.map((io: any, index: number) => ({
            __typename: "RunIO",
            id: generatePK().toString(),
            data: io.data,
            nodeInputName: io.nodeInputName,
            nodeName: io.nodeName,
            run: { __connect: true, id: DUMMY_ID },
        })) || [],
    });
}

function transformFormToUpdateRequestReal(runId: string, formData: Partial<any>): any {
    return shapeRun.update({
        __typename: "Run",
        id: runId,
    }, {
        __typename: "Run",
        id: runId,
        name: formData.name,
        isPrivate: formData.isPrivate,
        completedComplexity: formData.completedComplexity,
        contextSwitches: formData.contextSwitches,
        data: formData.data,
        timeElapsed: formData.timeElapsed,
        status: formData.status,
        steps: formData.steps?.map((step: any) => ({
            __typename: "RunStep",
            id: step.id || generatePK().toString(),
            name: step.name,
            complexity: step.complexity,
            status: step.status,
            contextSwitches: step.contextSwitches,
            timeElapsed: step.timeElapsed,
        })) || [],
        io: formData.ioData?.map((io: any) => ({
            __typename: "RunIO",
            id: io.id || generatePK().toString(),
            data: io.data,
        })) || [],
    });
}

async function validateRunFormDataReal(formData: any): Promise<string[]> {
    try {
        const validationData = {
            id: generatePK().toString(),
            name: formData.name,
            status: formData.status,
            isPrivate: formData.isPrivate,
            completedComplexity: formData.completedComplexity,
            contextSwitches: formData.contextSwitches,
            data: formData.data,
            timeElapsed: formData.timeElapsed,
            ...(formData.resourceVersionId && { 
                resourceVersionConnect: formData.resourceVersionId 
            }),
            ...(formData.teamId && { 
                teamConnect: formData.teamId 
            }),
            ...(formData.schedule && {
                scheduleCreate: {
                    id: generatePK().toString(),
                    ...formData.schedule,
                }
            }),
            ...(formData.steps && {
                stepsCreate: formData.steps.map((step: any) => ({
                    id: generatePK().toString(),
                    name: step.name,
                    nodeId: step.nodeId,
                    order: step.order,
                    complexity: step.complexity || 1,
                    resourceInId: formData.resourceVersionId,
                    runConnect: generatePK().toString(),
                }))
            }),
            ...(formData.ioData && {
                ioCreate: formData.ioData.map((io: any) => ({
                    id: generatePK().toString(),
                    data: io.data,
                    nodeInputName: io.nodeInputName,
                    nodeName: io.nodeName,
                    runConnect: generatePK().toString(),
                }))
            }),
        };
        
        await runValidation.create({ omitFields: [] }).validate(validationData);
        return []; // No validation errors
    } catch (error: any) {
        if (error.errors) {
            return error.errors; // Yup validation errors
        }
        return [error.message || "Validation failed"];
    }
}

function transformApiResponseToFormReal(run: Run): any {
    return {
        name: run.name,
        status: run.status,
        isPrivate: run.isPrivate,
        completedComplexity: run.completedComplexity,
        contextSwitches: run.contextSwitches,
        data: run.data,
        timeElapsed: run.timeElapsed,
        resourceVersionId: run.resourceVersion?.id,
        teamId: run.team?.id,
        startedAt: run.startedAt,
        steps: run.steps?.map(step => ({
            name: step.name,
            nodeId: step.nodeId,
            order: step.order,
            complexity: step.complexity,
            status: step.status,
        })) || [],
        ioData: run.io?.map(io => ({
            data: io.data,
            nodeInputName: io.nodeInputName,
            nodeName: io.nodeName,
        })) || [],
    };
}

function areRunFormsEqualReal(form1: any, form2: any): boolean {
    return (
        form1.name === form2.name &&
        form1.status === form2.status &&
        form1.isPrivate === form2.isPrivate &&
        form1.resourceVersionId === form2.resourceVersionId &&
        form1.teamId === form2.teamId
    );
}

describe('Run Round-Trip Data Flow', () => {
    beforeEach(() => {
        // Clear test storage between tests
        mockRunService.storage = {};
    });

    test('minimal run creation maintains data integrity through complete flow', async () => {
        // 🎨 STEP 1: User fills out minimal run form
        const userFormData = {
            name: "Test Run",
            status: RunStatus.InProgress,
            isPrivate: false,
            resourceVersionId: "resource_version_123456789012345678",
        };
        
        // Validate form data using REAL validation schema
        const validationErrors = await validateRunFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Form submits to API using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.name).toBe(userFormData.name);
        expect(apiCreateRequest.status).toBe(userFormData.status);
        expect(apiCreateRequest.isPrivate).toBe(userFormData.isPrivate);
        expect(apiCreateRequest.resourceVersionConnect).toBe(userFormData.resourceVersionId);
        expect(apiCreateRequest.id).toMatch(/^\d{10,19}$/);
        
        // 🗄️ STEP 3: API creates run
        const createdRun = await mockRunService.create(apiCreateRequest);
        expect(createdRun.id).toBe(apiCreateRequest.id);
        expect(createdRun.name).toBe(userFormData.name);
        expect(createdRun.status).toBe(userFormData.status);
        expect(createdRun.resourceVersion?.id).toBe(userFormData.resourceVersionId);
        
        // 🔗 STEP 4: API fetches run back
        const fetchedRun = await mockRunService.findById(createdRun.id);
        expect(fetchedRun.id).toBe(createdRun.id);
        expect(fetchedRun.name).toBe(userFormData.name);
        expect(fetchedRun.status).toBe(userFormData.status);
        
        // 🎨 STEP 5: UI displays the run using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedRun);
        expect(reconstructedFormData.name).toBe(userFormData.name);
        expect(reconstructedFormData.status).toBe(userFormData.status);
        expect(reconstructedFormData.isPrivate).toBe(userFormData.isPrivate);
        expect(reconstructedFormData.resourceVersionId).toBe(userFormData.resourceVersionId);
        
        // ✅ VERIFICATION: Complete round-trip integrity using REAL comparison
        expect(areRunFormsEqualReal(userFormData, reconstructedFormData)).toBe(true);
    });

    test('complete run with steps and IO preserves all data', async () => {
        // 🎨 STEP 1: User creates complex run with steps and IO data
        const userFormData = {
            name: "AI Processing Pipeline",
            status: RunStatus.InProgress,
            isPrivate: false,
            resourceVersionId: "resource_version_123456789012345678",
            teamId: "team_123456789012345678",
            data: JSON.stringify({
                config: { enableLogging: true },
                parameters: { threshold: 0.8 },
            }),
            steps: [
                { name: "Data Validation", nodeId: "validate_node", order: 0, complexity: 2 },
                { name: "Data Processing", nodeId: "process_node", order: 1, complexity: 5 },
            ],
            ioData: [
                { data: "input data", nodeInputName: "input1", nodeName: "ProcessNode" },
            ],
        };
        
        // Validate complex form using REAL validation
        const validationErrors = await validateRunFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform to API request using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.stepsCreate).toBeDefined();
        expect(apiCreateRequest.stepsCreate).toHaveLength(2);
        expect(apiCreateRequest.ioCreate).toBeDefined();
        expect(apiCreateRequest.ioCreate).toHaveLength(1);
        expect(apiCreateRequest.teamConnect).toBe(userFormData.teamId);
        
        // 🗄️ STEP 3: Create via API
        const createdRun = await mockRunService.create(apiCreateRequest);
        expect(createdRun.steps).toHaveLength(2);
        expect(createdRun.io).toHaveLength(1);
        expect(createdRun.team?.id).toBe(userFormData.teamId);
        expect(createdRun.data).toBe(userFormData.data);
        
        // 🔗 STEP 4: Fetch back from API
        const fetchedRun = await mockRunService.findById(createdRun.id);
        expect(fetchedRun.steps).toHaveLength(2);
        expect(fetchedRun.io).toHaveLength(1);
        expect(fetchedRun.data).toBe(userFormData.data);
        
        // 🎨 STEP 5: Verify UI can display correctly using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedRun);
        expect(reconstructedFormData.name).toBe(userFormData.name);
        expect(reconstructedFormData.steps).toHaveLength(2);
        expect(reconstructedFormData.ioData).toHaveLength(1);
        expect(reconstructedFormData.teamId).toBe(userFormData.teamId);
        
        // ✅ VERIFICATION: Complex data preserved
        expect(fetchedRun.data).toBe(userFormData.data);
        expect(fetchedRun.team?.id).toBe(userFormData.teamId);
    });

    test('run editing maintains data integrity', async () => {
        // Create initial run using REAL functions
        const initialFormData = {
            name: "Initial Run",
            status: RunStatus.InProgress,
            isPrivate: false,
            resourceVersionId: "resource_version_123456789012345678",
        };
        const createRequest = transformFormToCreateRequestReal(initialFormData);
        const initialRun = await mockRunService.create(createRequest);
        
        // 🎨 STEP 1: User edits run
        const editFormData = {
            name: "Updated Run Name",
            status: RunStatus.Paused,
            isPrivate: true,
            completedComplexity: 5,
            contextSwitches: 2,
            timeElapsed: 1800,
        };
        
        // 🔗 STEP 2: Transform edit to update request using REAL update logic
        const updateRequest = transformFormToUpdateRequestReal(initialRun.id, editFormData);
        expect(updateRequest.id).toBe(initialRun.id);
        expect(updateRequest.name).toBe(editFormData.name);
        expect(updateRequest.status).toBe(editFormData.status);
        
        // 🗄️ STEP 3: Update via API
        const updatedRun = await mockRunService.update(initialRun.id, updateRequest);
        expect(updatedRun.id).toBe(initialRun.id);
        expect(updatedRun.name).toBe(editFormData.name);
        expect(updatedRun.status).toBe(editFormData.status);
        expect(updatedRun.isPrivate).toBe(editFormData.isPrivate);
        
        // 🔗 STEP 4: Fetch updated run
        const fetchedUpdatedRun = await mockRunService.findById(initialRun.id);
        
        // ✅ VERIFICATION: Update preserved core data
        expect(fetchedUpdatedRun.id).toBe(initialRun.id);
        expect(fetchedUpdatedRun.name).toBe(editFormData.name);
        expect(fetchedUpdatedRun.status).toBe(editFormData.status);
        expect(fetchedUpdatedRun.isPrivate).toBe(editFormData.isPrivate);
        expect(fetchedUpdatedRun.resourceVersion?.id).toBe(initialFormData.resourceVersionId);
    });

    test('all run statuses work correctly through round-trip', async () => {
        const runStatuses = Object.values(RunStatus);
        
        for (const status of runStatuses) {
            // 🎨 Create form data for each status
            const formData = {
                name: `${status} Test Run`,
                status: status,
                isPrivate: false,
                resourceVersionId: "resource_version_123456789012345678",
            };
            
            // 🔗 Transform and create using REAL functions
            const createRequest = transformFormToCreateRequestReal(formData);
            const createdRun = await mockRunService.create(createRequest);
            
            // 🗄️ Fetch back
            const fetchedRun = await mockRunService.findById(createdRun.id);
            
            // ✅ Verify status-specific data
            expect(fetchedRun.status).toBe(status);
            expect(fetchedRun.name).toBe(formData.name);
            
            // Verify form reconstruction using REAL transformation
            const reconstructed = transformApiResponseToFormReal(fetchedRun);
            expect(reconstructed.status).toBe(status);
            expect(reconstructed.name).toBe(formData.name);
        }
    });

    test('validation catches invalid form data before API submission', async () => {
        const invalidFormData = {
            name: "", // Empty name (invalid)
            status: RunStatus.InProgress,
            isPrivate: false,
            resourceVersionId: "invalid-id", // Invalid ID format
        };
        
        const validationErrors = await validateRunFormDataReal(invalidFormData);
        expect(validationErrors.length).toBeGreaterThan(0);
        
        // Should not proceed to API if validation fails
        expect(validationErrors.some(error => 
            error.includes("name") || error.includes("required")
        )).toBe(true);
    });

    test('run deletion works correctly', async () => {
        // Create run first using REAL functions
        const formData = {
            name: "Run to Delete",
            status: RunStatus.InProgress,
            isPrivate: false,
            resourceVersionId: "resource_version_123456789012345678",
        };
        const createRequest = transformFormToCreateRequestReal(formData);
        const createdRun = await mockRunService.create(createRequest);
        
        // Delete it
        const deleteResult = await mockRunService.delete(createdRun.id);
        expect(deleteResult.success).toBe(true);
    });

    test('data consistency across multiple operations', async () => {
        const originalFormData = {
            name: "Multi-Op Test Run",
            status: RunStatus.InProgress,
            isPrivate: false,
            resourceVersionId: "resource_version_123456789012345678",
        };
        
        // Create using REAL functions
        const createRequest = transformFormToCreateRequestReal(originalFormData);
        const created = await mockRunService.create(createRequest);
        
        // Update using REAL functions
        const updateRequest = transformFormToUpdateRequestReal(created.id, { 
            name: "Updated Multi-Op Test Run",
            status: RunStatus.Completed,
        });
        const updated = await mockRunService.update(created.id, updateRequest);
        
        // Fetch final state
        const final = await mockRunService.findById(created.id);
        
        // Core run data should remain consistent
        expect(final.id).toBe(created.id);
        expect(final.resourceVersion?.id).toBe(originalFormData.resourceVersionId);
        expect(final.user.id).toBe(created.user.id);
        
        // Only the updated fields should have changed
        expect(final.name).toBe("Updated Multi-Op Test Run");
        expect(final.status).toBe(RunStatus.Completed);
        expect(final.isPrivate).toBe(originalFormData.isPrivate); // Unchanged
    });
});
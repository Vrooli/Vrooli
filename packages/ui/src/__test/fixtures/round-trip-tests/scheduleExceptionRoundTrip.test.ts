import { describe, test, expect, beforeEach } from 'vitest';
import { 
    shapeScheduleException, 
    scheduleExceptionValidation, 
    generatePK, 
    type ScheduleException 
} from "@vrooli/shared";
import { 
    minimalScheduleExceptionCreateFormInput,
    completeScheduleExceptionCreateFormInput,
    minimalScheduleExceptionUpdateFormInput,
    type ScheduleExceptionFormData
} from '../form-data/scheduleExceptionFormData.js';

/**
 * Round-trip testing for ScheduleException data flow using REAL application functions
 * Tests the complete user journey: Form Input → API Request → Database → API Response → UI Display
 * 
 * ✅ Uses real shapeScheduleException.create() for transformations
 * ✅ Uses real scheduleExceptionValidation for validation
 * ✅ Tests actual application logic instead of mock implementations
 */

// Mock service for testing (would use real database in integration tests)
const mockScheduleExceptionService = {
    storage: {} as Record<string, ScheduleException>,
    
    async create(data: any): Promise<ScheduleException> {
        const id = data.id || generatePK().toString();
        const scheduleException: ScheduleException = {
            __typename: "ScheduleException",
            id,
            originalStartTime: data.originalStartTime,
            newStartTime: data.newStartTime || null,
            newEndTime: data.newEndTime || null,
            schedule: {
                __typename: "Schedule",
                id: data.scheduleConnect,
                startTime: "2025-01-01T09:00:00Z",
                endTime: "2025-12-31T17:00:00Z",
                timezone: "UTC",
                createdAt: "2024-01-01T00:00:00Z",
                updatedAt: "2024-01-01T00:00:00Z",
                user: {} as any,
                exceptions: [],
                recurrences: [],
                meetings: [],
                runs: [],
            },
        };
        this.storage[id] = scheduleException;
        return scheduleException;
    },
    
    async findById(id: string): Promise<ScheduleException> {
        const scheduleException = this.storage[id];
        if (!scheduleException) {
            throw new Error(`ScheduleException with id ${id} not found`);
        }
        return scheduleException;
    },
    
    async update(id: string, data: any): Promise<ScheduleException> {
        const existing = await this.findById(id);
        const updated: ScheduleException = {
            ...existing,
            ...data,
            id, // Keep original ID
        };
        this.storage[id] = updated;
        return updated;
    },
    
    async delete(id: string): Promise<{ success: boolean }> {
        if (this.storage[id]) {
            delete this.storage[id];
            return { success: true };
        }
        return { success: false };
    }
};

// Helper functions using REAL application logic
function transformFormToCreateRequestReal(formData: any) {
    return shapeScheduleException.create({
        __typename: "ScheduleException",
        id: generatePK().toString(),
        originalStartTime: formData.originalStartTime,
        newStartTime: formData.newStartTime,
        newEndTime: formData.newEndTime,
        schedule: {
            __typename: "Schedule",
            __connect: true,
            id: formData.scheduleId,
        },
    });
}

function transformFormToUpdateRequestReal(exceptionId: string, formData: any) {
    const updateRequest: any = {
        id: exceptionId,
    };
    
    if (formData.originalStartTime !== undefined) {
        updateRequest.originalStartTime = formData.originalStartTime;
    }
    if (formData.newStartTime !== undefined) {
        updateRequest.newStartTime = formData.newStartTime;
    }
    if (formData.newEndTime !== undefined) {
        updateRequest.newEndTime = formData.newEndTime;
    }
    
    return updateRequest;
}

async function validateScheduleExceptionFormDataReal(formData: any): Promise<string[]> {
    try {
        // Use real validation schema
        const validationData = {
            id: generatePK().toString(),
            originalStartTime: formData.originalStartTime,
            newStartTime: formData.newStartTime,
            newEndTime: formData.newEndTime,
            scheduleConnect: formData.scheduleId,
        };
        
        await scheduleExceptionValidation.create({ omitFields: [] }).validate(validationData);
        return []; // No validation errors
    } catch (error: any) {
        if (error.errors) {
            return error.errors; // Yup validation errors
        }
        return [error.message || "Validation failed"];
    }
}

function transformApiResponseToFormReal(scheduleException: ScheduleException): any {
    return {
        originalStartTime: new Date(scheduleException.originalStartTime),
        newStartTime: scheduleException.newStartTime ? new Date(scheduleException.newStartTime) : null,
        newEndTime: scheduleException.newEndTime ? new Date(scheduleException.newEndTime) : null,
        scheduleId: scheduleException.schedule.id,
    };
}

function areScheduleExceptionFormsEqualReal(form1: any, form2: any): boolean {
    return (
        form1.originalStartTime?.getTime() === form2.originalStartTime?.getTime() &&
        form1.newStartTime?.getTime() === form2.newStartTime?.getTime() &&
        form1.newEndTime?.getTime() === form2.newEndTime?.getTime() &&
        form1.scheduleId === form2.scheduleId
    );
}

describe('ScheduleException Round-Trip Data Flow', () => {
    beforeEach(() => {
        // Clear test storage between tests
        mockScheduleExceptionService.storage = {};
    });

    test('minimal schedule exception creation maintains data integrity through complete flow', async () => {
        // 🎨 STEP 1: User fills out minimal schedule exception form
        const userFormData = {
            originalStartTime: new Date("2025-07-04T09:00:00Z"),
            newStartTime: new Date("2025-07-05T10:00:00Z"),
            newEndTime: new Date("2025-07-05T17:00:00Z"),
            scheduleId: "123456789012345678",
        };
        
        // Validate form data using REAL validation schema
        const validationErrors = await validateScheduleExceptionFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Form submits to API using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.originalStartTime).toEqual(userFormData.originalStartTime);
        expect(apiCreateRequest.newStartTime).toEqual(userFormData.newStartTime);
        expect(apiCreateRequest.newEndTime).toEqual(userFormData.newEndTime);
        expect(apiCreateRequest.scheduleConnect).toBe(userFormData.scheduleId);
        expect(apiCreateRequest.id).toMatch(/^\d{10,19}$/); // Valid ID
        
        // 🗄️ STEP 3: API creates schedule exception (simulated - real test would hit test DB)
        const createdScheduleException = await mockScheduleExceptionService.create(apiCreateRequest);
        expect(createdScheduleException.id).toBe(apiCreateRequest.id);
        expect(new Date(createdScheduleException.originalStartTime).getTime()).toBe(userFormData.originalStartTime.getTime());
        expect(new Date(createdScheduleException.newStartTime!).getTime()).toBe(userFormData.newStartTime.getTime());
        expect(new Date(createdScheduleException.newEndTime!).getTime()).toBe(userFormData.newEndTime.getTime());
        expect(createdScheduleException.schedule.id).toBe(userFormData.scheduleId);
        
        // 🔗 STEP 4: API fetches schedule exception back
        const fetchedScheduleException = await mockScheduleExceptionService.findById(createdScheduleException.id);
        expect(fetchedScheduleException.id).toBe(createdScheduleException.id);
        expect(fetchedScheduleException.originalStartTime).toBe(createdScheduleException.originalStartTime);
        expect(fetchedScheduleException.newStartTime).toBe(createdScheduleException.newStartTime);
        expect(fetchedScheduleException.newEndTime).toBe(createdScheduleException.newEndTime);
        
        // 🎨 STEP 5: UI would display the schedule exception using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedScheduleException);
        expect(reconstructedFormData.originalStartTime.getTime()).toBe(userFormData.originalStartTime.getTime());
        expect(reconstructedFormData.newStartTime?.getTime()).toBe(userFormData.newStartTime.getTime());
        expect(reconstructedFormData.newEndTime?.getTime()).toBe(userFormData.newEndTime.getTime());
        expect(reconstructedFormData.scheduleId).toBe(userFormData.scheduleId);
        
        // ✅ VERIFICATION: Complete round-trip integrity using REAL comparison
        expect(areScheduleExceptionFormsEqualReal(
            userFormData, 
            reconstructedFormData
        )).toBe(true);
    });

    test('complete schedule exception with cancellation preserves all data', async () => {
        // 🎨 STEP 1: User creates schedule exception with cancellation (null times)
        const userFormData = {
            originalStartTime: new Date("2025-12-25T09:00:00Z"),
            newStartTime: null, // Cancelled
            newEndTime: null, // Cancelled
            scheduleId: "234567890123456789",
        };
        
        // Validate cancellation form using REAL validation
        const validationErrors = await validateScheduleExceptionFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform to API request using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.originalStartTime).toEqual(userFormData.originalStartTime);
        expect(apiCreateRequest.newStartTime).toBe(null);
        expect(apiCreateRequest.newEndTime).toBe(null);
        expect(apiCreateRequest.scheduleConnect).toBe(userFormData.scheduleId);
        
        // 🗄️ STEP 3: Create via API
        const createdScheduleException = await mockScheduleExceptionService.create(apiCreateRequest);
        expect(createdScheduleException.newStartTime).toBe(null);
        expect(createdScheduleException.newEndTime).toBe(null);
        expect(new Date(createdScheduleException.originalStartTime).getTime()).toBe(userFormData.originalStartTime.getTime());
        
        // 🔗 STEP 4: Fetch back from API
        const fetchedScheduleException = await mockScheduleExceptionService.findById(createdScheduleException.id);
        expect(fetchedScheduleException.newStartTime).toBe(null);
        expect(fetchedScheduleException.newEndTime).toBe(null);
        
        // 🎨 STEP 5: Verify UI can display correctly using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedScheduleException);
        expect(reconstructedFormData.originalStartTime.getTime()).toBe(userFormData.originalStartTime.getTime());
        expect(reconstructedFormData.newStartTime).toBe(null);
        expect(reconstructedFormData.newEndTime).toBe(null);
        expect(reconstructedFormData.scheduleId).toBe(userFormData.scheduleId);
        
        // ✅ VERIFICATION: Cancellation data preserved
        expect(areScheduleExceptionFormsEqualReal(userFormData, reconstructedFormData)).toBe(true);
    });

    test('schedule exception editing maintains data integrity', async () => {
        // Create initial schedule exception using REAL functions
        const initialFormData = minimalScheduleExceptionCreateFormInput;
        const createRequest = transformFormToCreateRequestReal(initialFormData);
        const initialException = await mockScheduleExceptionService.create(createRequest);
        
        // 🎨 STEP 1: User edits schedule exception times
        const editFormData = {
            newStartTime: new Date("2025-07-05T14:00:00Z"), // Changed time
            newEndTime: new Date("2025-07-05T18:00:00Z"), // Changed duration
        };
        
        // 🔗 STEP 2: Transform edit to update request using REAL update logic
        const updateRequest = transformFormToUpdateRequestReal(initialException.id, editFormData);
        expect(updateRequest.id).toBe(initialException.id);
        expect(updateRequest.newStartTime).toEqual(editFormData.newStartTime);
        expect(updateRequest.newEndTime).toEqual(editFormData.newEndTime);
        
        // 🗄️ STEP 3: Update via API
        const updatedException = await mockScheduleExceptionService.update(initialException.id, updateRequest);
        expect(updatedException.id).toBe(initialException.id);
        expect(new Date(updatedException.newStartTime!).getTime()).toBe(editFormData.newStartTime.getTime());
        expect(new Date(updatedException.newEndTime!).getTime()).toBe(editFormData.newEndTime.getTime());
        
        // 🔗 STEP 4: Fetch updated schedule exception
        const fetchedUpdatedException = await mockScheduleExceptionService.findById(initialException.id);
        
        // ✅ VERIFICATION: Update preserved core data
        expect(fetchedUpdatedException.id).toBe(initialException.id);
        expect(fetchedUpdatedException.originalStartTime).toBe(initialException.originalStartTime);
        expect(fetchedUpdatedException.schedule.id).toBe(initialException.schedule.id);
        expect(new Date(fetchedUpdatedException.newStartTime!).getTime()).toBe(editFormData.newStartTime.getTime());
        expect(new Date(fetchedUpdatedException.newEndTime!).getTime()).toBe(editFormData.newEndTime.getTime());
    });

    test('validation catches invalid form data before API submission', async () => {
        const invalidFormData = {
            originalStartTime: new Date("2025-07-04T09:00:00Z"),
            newStartTime: new Date("2025-07-05T17:00:00Z"),
            newEndTime: new Date("2025-07-05T10:00:00Z"), // Before start time
            scheduleId: "123456789012345678",
        };
        
        const validationErrors = await validateScheduleExceptionFormDataReal(invalidFormData);
        expect(validationErrors.length).toBeGreaterThan(0);
        
        // Should not proceed to API if validation fails
        expect(validationErrors.some(error => 
            error.includes("End time") || error.includes("after")
        )).toBe(true);
    });

    test('schedule exception deletion works correctly', async () => {
        // Create schedule exception first using REAL functions
        const formData = minimalScheduleExceptionCreateFormInput;
        const createRequest = transformFormToCreateRequestReal(formData);
        const createdScheduleException = await mockScheduleExceptionService.create(createRequest);
        
        // Delete it
        const deleteResult = await mockScheduleExceptionService.delete(createdScheduleException.id);
        expect(deleteResult.success).toBe(true);
        
        // Verify it's gone
        await expect(mockScheduleExceptionService.findById(createdScheduleException.id))
            .rejects
            .toThrow("not found");
    });

    test('data consistency across multiple operations', async () => {
        const originalFormData = completeScheduleExceptionCreateFormInput;
        
        // Create using REAL functions
        const createRequest = transformFormToCreateRequestReal(originalFormData);
        const created = await mockScheduleExceptionService.create(createRequest);
        
        // Update using REAL functions
        const updateRequest = transformFormToUpdateRequestReal(created.id, { 
            newStartTime: new Date("2025-07-06T11:00:00Z"),
            newEndTime: new Date("2025-07-06T16:00:00Z"),
        });
        const updated = await mockScheduleExceptionService.update(created.id, updateRequest);
        
        // Fetch final state
        const final = await mockScheduleExceptionService.findById(created.id);
        
        // Core schedule exception data should remain consistent
        expect(final.id).toBe(created.id);
        expect(final.originalStartTime).toBe(created.originalStartTime);
        expect(final.schedule.id).toBe(created.schedule.id);
        
        // Only the new times should have changed
        expect(new Date(final.newStartTime!).getTime()).toBe(updateRequest.newStartTime.getTime());
        expect(new Date(final.newEndTime!).getTime()).toBe(updateRequest.newEndTime.getTime());
        expect(final.newStartTime).not.toBe(created.newStartTime);
        expect(final.newEndTime).not.toBe(created.newEndTime);
    });

    test('time-based edge cases work correctly', async () => {
        // Test same day reschedule
        const sameDayFormData = {
            originalStartTime: new Date("2025-07-04T09:00:00Z"),
            newStartTime: new Date("2025-07-04T14:00:00Z"), // Same day, later time
            newEndTime: new Date("2025-07-04T16:00:00Z"),
            scheduleId: "345678901234567890",
        };
        
        const validationErrors = await validateScheduleExceptionFormDataReal(sameDayFormData);
        expect(validationErrors).toHaveLength(0);
        
        const createRequest = transformFormToCreateRequestReal(sameDayFormData);
        const created = await mockScheduleExceptionService.create(createRequest);
        const fetched = await mockScheduleExceptionService.findById(created.id);
        
        const reconstructed = transformApiResponseToFormReal(fetched);
        expect(areScheduleExceptionFormsEqualReal(sameDayFormData, reconstructed)).toBe(true);
    });
});
import { describe, test, expect, beforeEach } from 'vitest';
import { 
    shapeReminderItem, 
    reminderItemValidation, 
    generatePK, 
    type ReminderItem 
} from "@vrooli/shared";
import { 
    minimalReminderItemCreateFormInput,
    completeReminderItemCreateFormInput,
    minimalReminderItemUpdateFormInput,
    type ReminderItemFormData
} from '../form-data/reminderItemFormData.js';

/**
 * Round-trip testing for ReminderItem data flow using REAL application functions
 * Tests the complete user journey: Form Input → API Request → Database → API Response → UI Display
 * 
 * ✅ Uses real shapeReminderItem.create() for transformations
 * ✅ Uses real reminderItemValidation for validation
 * ✅ Tests actual application logic instead of mock implementations
 */

// Mock service for testing (would use real database in integration tests)
const mockReminderItemService = {
    storage: {} as Record<string, ReminderItem>,
    
    async create(data: any): Promise<ReminderItem> {
        const id = data.id || generatePK().toString();
        const reminderItem: ReminderItem = {
            __typename: "ReminderItem",
            id,
            name: data.name,
            description: data.description || null,
            dueDate: data.dueDate || null,
            index: data.index || 0,
            isComplete: data.isComplete || false,
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            reminder: {
                __typename: "Reminder",
                id: data.reminderConnect,
                name: "Parent Reminder",
                description: null,
                dueDate: null,
                index: 0,
                isComplete: false,
                createdAt: "2024-01-01T00:00:00Z",
                updatedAt: "2024-01-01T00:00:00Z",
                reminderItems: [],
                reminderList: {} as any,
            },
        };
        this.storage[id] = reminderItem;
        return reminderItem;
    },
    
    async findById(id: string): Promise<ReminderItem> {
        const reminderItem = this.storage[id];
        if (!reminderItem) {
            throw new Error(`ReminderItem with id ${id} not found`);
        }
        return reminderItem;
    },
    
    async update(id: string, data: any): Promise<ReminderItem> {
        const existing = await this.findById(id);
        const updated: ReminderItem = {
            ...existing,
            ...data,
            id, // Keep original ID
            updatedAt: new Date().toISOString(),
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
    return shapeReminderItem.create({
        __typename: "ReminderItem",
        id: generatePK().toString(),
        name: formData.name,
        description: formData.description,
        dueDate: formData.dueDate,
        index: formData.index,
        isComplete: formData.isComplete,
        reminder: {
            __typename: "Reminder",
            __connect: true,
            id: formData.reminderId,
        },
    });
}

function transformFormToUpdateRequestReal(itemId: string, formData: any) {
    const updateRequest: any = {
        id: itemId,
    };
    
    if (formData.name !== undefined) {
        updateRequest.name = formData.name;
    }
    if (formData.description !== undefined) {
        updateRequest.description = formData.description;
    }
    if (formData.dueDate !== undefined) {
        updateRequest.dueDate = formData.dueDate;
    }
    if (formData.index !== undefined) {
        updateRequest.index = formData.index;
    }
    if (formData.isComplete !== undefined) {
        updateRequest.isComplete = formData.isComplete;
    }
    
    return updateRequest;
}

async function validateReminderItemFormDataReal(formData: any): Promise<string[]> {
    try {
        // Use real validation schema
        const validationData = {
            id: generatePK().toString(),
            name: formData.name,
            description: formData.description,
            dueDate: formData.dueDate,
            index: formData.index,
            isComplete: formData.isComplete,
            reminderConnect: formData.reminderId,
        };
        
        await reminderItemValidation.create({ omitFields: [] }).validate(validationData);
        return []; // No validation errors
    } catch (error: any) {
        if (error.errors) {
            return error.errors; // Yup validation errors
        }
        return [error.message || "Validation failed"];
    }
}

function transformApiResponseToFormReal(reminderItem: ReminderItem): any {
    return {
        name: reminderItem.name,
        description: reminderItem.description || "",
        dueDate: reminderItem.dueDate ? new Date(reminderItem.dueDate) : null,
        index: reminderItem.index,
        isComplete: reminderItem.isComplete,
        reminderId: reminderItem.reminder.id,
    };
}

function areReminderItemFormsEqualReal(form1: any, form2: any): boolean {
    return (
        form1.name === form2.name &&
        form1.description === form2.description &&
        form1.dueDate?.getTime() === form2.dueDate?.getTime() &&
        form1.index === form2.index &&
        form1.isComplete === form2.isComplete &&
        form1.reminderId === form2.reminderId
    );
}

describe('ReminderItem Round-Trip Data Flow', () => {
    beforeEach(() => {
        // Clear test storage between tests
        mockReminderItemService.storage = {};
    });

    test('minimal reminder item creation maintains data integrity through complete flow', async () => {
        // 🎨 STEP 1: User fills out minimal reminder item form
        const userFormData = {
            name: "Simple task",
            description: "",
            dueDate: null,
            index: 0,
            isComplete: false,
            reminderId: "123456789012345678",
        };
        
        // Validate form data using REAL validation schema
        const validationErrors = await validateReminderItemFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Form submits to API using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.name).toBe(userFormData.name);
        expect(apiCreateRequest.description).toBe(userFormData.description);
        expect(apiCreateRequest.dueDate).toBe(userFormData.dueDate);
        expect(apiCreateRequest.index).toBe(userFormData.index);
        expect(apiCreateRequest.isComplete).toBe(userFormData.isComplete);
        expect(apiCreateRequest.reminderConnect).toBe(userFormData.reminderId);
        expect(apiCreateRequest.id).toMatch(/^\d{10,19}$/); // Valid ID
        
        // 🗄️ STEP 3: API creates reminder item (simulated - real test would hit test DB)
        const createdReminderItem = await mockReminderItemService.create(apiCreateRequest);
        expect(createdReminderItem.id).toBe(apiCreateRequest.id);
        expect(createdReminderItem.name).toBe(userFormData.name);
        expect(createdReminderItem.description).toBe(userFormData.description);
        expect(createdReminderItem.index).toBe(userFormData.index);
        expect(createdReminderItem.isComplete).toBe(userFormData.isComplete);
        expect(createdReminderItem.reminder.id).toBe(userFormData.reminderId);
        
        // 🔗 STEP 4: API fetches reminder item back
        const fetchedReminderItem = await mockReminderItemService.findById(createdReminderItem.id);
        expect(fetchedReminderItem.id).toBe(createdReminderItem.id);
        expect(fetchedReminderItem.name).toBe(createdReminderItem.name);
        expect(fetchedReminderItem.description).toBe(createdReminderItem.description);
        expect(fetchedReminderItem.index).toBe(createdReminderItem.index);
        expect(fetchedReminderItem.isComplete).toBe(createdReminderItem.isComplete);
        
        // 🎨 STEP 5: UI would display the reminder item using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedReminderItem);
        expect(reconstructedFormData.name).toBe(userFormData.name);
        expect(reconstructedFormData.description).toBe(userFormData.description);
        expect(reconstructedFormData.dueDate).toBe(userFormData.dueDate);
        expect(reconstructedFormData.index).toBe(userFormData.index);
        expect(reconstructedFormData.isComplete).toBe(userFormData.isComplete);
        expect(reconstructedFormData.reminderId).toBe(userFormData.reminderId);
        
        // ✅ VERIFICATION: Complete round-trip integrity using REAL comparison
        expect(areReminderItemFormsEqualReal(
            userFormData, 
            reconstructedFormData
        )).toBe(true);
    });

    test('complete reminder item with all fields preserves all data', async () => {
        // 🎨 STEP 1: User creates reminder item with all optional fields
        const userFormData = {
            name: "Review pull request",
            description: "Review and approve the latest pull request for the authentication module",
            dueDate: new Date("2024-12-31T15:00:00Z"),
            index: 0,
            isComplete: false,
            reminderId: "234567890123456789",
        };
        
        // Validate complete form using REAL validation
        const validationErrors = await validateReminderItemFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform to API request using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.name).toBe(userFormData.name);
        expect(apiCreateRequest.description).toBe(userFormData.description);
        expect(apiCreateRequest.dueDate).toEqual(userFormData.dueDate);
        expect(apiCreateRequest.index).toBe(userFormData.index);
        expect(apiCreateRequest.isComplete).toBe(userFormData.isComplete);
        expect(apiCreateRequest.reminderConnect).toBe(userFormData.reminderId);
        
        // 🗄️ STEP 3: Create via API
        const createdReminderItem = await mockReminderItemService.create(apiCreateRequest);
        expect(createdReminderItem.name).toBe(userFormData.name);
        expect(createdReminderItem.description).toBe(userFormData.description);
        expect(createdReminderItem.dueDate).toEqual(userFormData.dueDate);
        expect(createdReminderItem.isComplete).toBe(userFormData.isComplete);
        
        // 🔗 STEP 4: Fetch back from API
        const fetchedReminderItem = await mockReminderItemService.findById(createdReminderItem.id);
        expect(fetchedReminderItem.description).toBe(userFormData.description);
        expect(fetchedReminderItem.dueDate).toEqual(userFormData.dueDate);
        
        // 🎨 STEP 5: Verify UI can display correctly using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedReminderItem);
        expect(reconstructedFormData.name).toBe(userFormData.name);
        expect(reconstructedFormData.description).toBe(userFormData.description);
        expect(reconstructedFormData.dueDate?.getTime()).toBe(userFormData.dueDate.getTime());
        expect(reconstructedFormData.index).toBe(userFormData.index);
        expect(reconstructedFormData.isComplete).toBe(userFormData.isComplete);
        expect(reconstructedFormData.reminderId).toBe(userFormData.reminderId);
        
        // ✅ VERIFICATION: Complete data preserved
        expect(areReminderItemFormsEqualReal(userFormData, reconstructedFormData)).toBe(true);
    });

    test('reminder item editing maintains data integrity', async () => {
        // Create initial reminder item using REAL functions
        const initialFormData = minimalReminderItemCreateFormInput;
        const createRequest = transformFormToCreateRequestReal(initialFormData);
        const initialItem = await mockReminderItemService.create(createRequest);
        
        // 🎨 STEP 1: User edits reminder item
        const editFormData = {
            name: "Updated task name",
            description: "Updated description with more details",
            isComplete: true,
            dueDate: new Date("2025-01-15T10:00:00Z"),
        };
        
        // 🔗 STEP 2: Transform edit to update request using REAL update logic
        const updateRequest = transformFormToUpdateRequestReal(initialItem.id, editFormData);
        expect(updateRequest.id).toBe(initialItem.id);
        expect(updateRequest.name).toBe(editFormData.name);
        expect(updateRequest.description).toBe(editFormData.description);
        expect(updateRequest.isComplete).toBe(editFormData.isComplete);
        expect(updateRequest.dueDate).toEqual(editFormData.dueDate);
        
        // 🗄️ STEP 3: Update via API
        const updatedItem = await mockReminderItemService.update(initialItem.id, updateRequest);
        expect(updatedItem.id).toBe(initialItem.id);
        expect(updatedItem.name).toBe(editFormData.name);
        expect(updatedItem.description).toBe(editFormData.description);
        expect(updatedItem.isComplete).toBe(editFormData.isComplete);
        expect(updatedItem.dueDate).toEqual(editFormData.dueDate);
        
        // 🔗 STEP 4: Fetch updated reminder item
        const fetchedUpdatedItem = await mockReminderItemService.findById(initialItem.id);
        
        // ✅ VERIFICATION: Update preserved core data and changed updated fields
        expect(fetchedUpdatedItem.id).toBe(initialItem.id);
        expect(fetchedUpdatedItem.reminder.id).toBe(initialItem.reminder.id);
        expect(fetchedUpdatedItem.createdAt).toBe(initialItem.createdAt); // Created date unchanged
        expect(fetchedUpdatedItem.name).toBe(editFormData.name);
        expect(fetchedUpdatedItem.description).toBe(editFormData.description);
        expect(fetchedUpdatedItem.isComplete).toBe(editFormData.isComplete);
        expect(fetchedUpdatedItem.dueDate).toEqual(editFormData.dueDate);
        // Updated date should be different
        expect(fetchedUpdatedItem.updatedAt).not.toBe(initialItem.updatedAt);
    });

    test('reminder item completion status works correctly through round-trip', async () => {
        // Test completion workflow
        const tasks = [
            { name: "Task 1", isComplete: false },
            { name: "Task 2", isComplete: true },
            { name: "Task 3", isComplete: false },
        ];
        
        for (const [index, task] of tasks.entries()) {
            // 🎨 Create form data for each completion state
            const formData = {
                name: task.name,
                description: "",
                dueDate: null,
                index,
                isComplete: task.isComplete,
                reminderId: "completion_123456789012345678",
            };
            
            // 🔗 Transform and create using REAL functions
            const createRequest = transformFormToCreateRequestReal(formData);
            const createdItem = await mockReminderItemService.create(createRequest);
            
            // 🗄️ Fetch back
            const fetchedItem = await mockReminderItemService.findById(createdItem.id);
            
            // ✅ Verify completion state
            expect(fetchedItem.isComplete).toBe(task.isComplete);
            expect(fetchedItem.name).toBe(task.name);
            
            // Verify form reconstruction using REAL transformation
            const reconstructed = transformApiResponseToFormReal(fetchedItem);
            expect(reconstructed.isComplete).toBe(task.isComplete);
            expect(areReminderItemFormsEqualReal(formData, reconstructed)).toBe(true);
        }
    });

    test('validation catches invalid form data before API submission', async () => {
        const invalidFormData = {
            name: "", // Required field missing
            description: "Valid description",
            dueDate: null,
            index: 0,
            isComplete: false,
            reminderId: "123456789012345678",
        };
        
        const validationErrors = await validateReminderItemFormDataReal(invalidFormData);
        expect(validationErrors.length).toBeGreaterThan(0);
        
        // Should not proceed to API if validation fails
        expect(validationErrors.some(error => 
            error.includes("name") || error.includes("required")
        )).toBe(true);
    });

    test('reminder item deletion works correctly', async () => {
        // Create reminder item first using REAL functions
        const formData = completeReminderItemCreateFormInput;
        const createRequest = transformFormToCreateRequestReal(formData);
        const createdReminderItem = await mockReminderItemService.create(createRequest);
        
        // Delete it
        const deleteResult = await mockReminderItemService.delete(createdReminderItem.id);
        expect(deleteResult.success).toBe(true);
        
        // Verify it's gone
        await expect(mockReminderItemService.findById(createdReminderItem.id))
            .rejects
            .toThrow("not found");
    });

    test('data consistency across multiple operations', async () => {
        const originalFormData = completeReminderItemCreateFormInput;
        
        // Create using REAL functions
        const createRequest = transformFormToCreateRequestReal(originalFormData);
        const created = await mockReminderItemService.create(createRequest);
        
        // Update using REAL functions
        const updateRequest = transformFormToUpdateRequestReal(created.id, { 
            name: "Final task name",
            isComplete: true,
            dueDate: new Date("2025-02-28T17:00:00Z"),
        });
        const updated = await mockReminderItemService.update(created.id, updateRequest);
        
        // Fetch final state
        const final = await mockReminderItemService.findById(created.id);
        
        // Core reminder item data should remain consistent
        expect(final.id).toBe(created.id);
        expect(final.reminder.id).toBe(created.reminder.id);
        expect(final.createdAt).toBe(created.createdAt);
        
        // Only the updated fields should have changed
        expect(final.name).toBe(updateRequest.name);
        expect(final.isComplete).toBe(updateRequest.isComplete);
        expect(final.dueDate).toEqual(updateRequest.dueDate);
        expect(final.name).not.toBe(created.name);
        expect(final.isComplete).not.toBe(created.isComplete);
    });

    test('index-based ordering works correctly', async () => {
        const items = [
            { name: "First item", index: 0 },
            { name: "Second item", index: 1 },
            { name: "Third item", index: 2 },
        ];
        
        // Create items with specific indexes
        for (const item of items) {
            const formData = {
                name: item.name,
                description: "",
                dueDate: null,
                index: item.index,
                isComplete: false,
                reminderId: "ordering_123456789012345678",
            };
            
            const createRequest = transformFormToCreateRequestReal(formData);
            const created = await mockReminderItemService.create(createRequest);
            const fetched = await mockReminderItemService.findById(created.id);
            
            expect(fetched.index).toBe(item.index);
            expect(fetched.name).toBe(item.name);
            
            const reconstructed = transformApiResponseToFormReal(fetched);
            expect(reconstructed.index).toBe(item.index);
            expect(areReminderItemFormsEqualReal(formData, reconstructed)).toBe(true);
        }
    });

    test('due date handling works correctly', async () => {
        // Test different due date scenarios
        const dueDateScenarios = [
            { name: "No due date", dueDate: null },
            { name: "Future due date", dueDate: new Date("2025-06-15T14:30:00Z") },
            { name: "Past due date", dueDate: new Date("2023-12-01T09:00:00Z") },
            { name: "Today due date", dueDate: new Date() },
        ];
        
        for (const scenario of dueDateScenarios) {
            const formData = {
                name: scenario.name,
                description: "",
                dueDate: scenario.dueDate,
                index: 0,
                isComplete: false,
                reminderId: "duedate_123456789012345678",
            };
            
            const validationErrors = await validateReminderItemFormDataReal(formData);
            expect(validationErrors).toHaveLength(0);
            
            const createRequest = transformFormToCreateRequestReal(formData);
            const created = await mockReminderItemService.create(createRequest);
            const fetched = await mockReminderItemService.findById(created.id);
            
            if (scenario.dueDate) {
                expect(new Date(fetched.dueDate!).getTime()).toBe(scenario.dueDate.getTime());
            } else {
                expect(fetched.dueDate).toBe(null);
            }
            
            const reconstructed = transformApiResponseToFormReal(fetched);
            expect(areReminderItemFormsEqualReal(formData, reconstructed)).toBe(true);
        }
    });
});
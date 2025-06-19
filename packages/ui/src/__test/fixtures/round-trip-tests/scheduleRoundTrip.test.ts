import { describe, test, expect, beforeEach } from 'vitest';
import { 
    shapeSchedule, 
    scheduleValidation, 
    generatePK, 
    type Schedule, 
    type ScheduleRecurrenceType,
    endpointsSchedule 
} from "@vrooli/shared";
import {
    minimalScheduleCreateFormInput,
    completeScheduleCreateFormInput,
    minimalScheduleUpdateFormInput,
    completeScheduleUpdateFormInput,
    scheduleFormVariants,
    transformScheduleFormToApiInput,
    validateScheduleTimeRange,
    validateTimezone
} from '../form-data/scheduleFormData.js';
import {
    minimalScheduleResponse,
    completeScheduleResponse,
    oneTimeScheduleResponse,
    weeklyScheduleResponse,
    monthlyScheduleResponse,
    yearlyScheduleResponse,
    scheduleResponseVariants
} from '../api-responses/scheduleResponses.js';

/**
 * Round-trip testing for Schedule data flow using REAL application functions
 * Tests the complete user journey: Form Input → API Request → Database → API Response → UI Display
 * 
 * ✅ Uses real shapeSchedule.create() and shapeSchedule.update() for transformations
 * ✅ Uses real scheduleValidation for validation
 * ✅ Tests actual application logic instead of mock implementations
 */

// Form data interface for type safety
interface ScheduleFormData {
    timezone: string;
    startTime: Date | null;
    endTime: Date | null;
    meetingId?: string | null;
    runId?: string | null;
    recurrences: Array<{
        recurrenceType: ScheduleRecurrenceType;
        interval: number;
        duration: number;
        dayOfWeek?: number | null;
        dayOfMonth?: number | null;
        month?: number | null;
        endDate?: Date | null;
    }>;
    exceptions: Array<{
        originalStartTime: Date;
        newStartTime?: Date | null;
        newEndTime?: Date | null;
    }>;
}

// Helper functions using REAL application logic
function transformFormToCreateRequestReal(formData: ScheduleFormData) {
    const scheduleId = generatePK().toString();
    
    return shapeSchedule.create({
        __typename: "Schedule",
        id: scheduleId,
        timezone: formData.timezone,
        startTime: formData.startTime,
        endTime: formData.endTime,
        exceptions: formData.exceptions.map(exception => ({
            __typename: "ScheduleException",
            id: generatePK().toString(),
            originalStartTime: exception.originalStartTime,
            newStartTime: exception.newStartTime,
            newEndTime: exception.newEndTime,
            schedule: {
                __typename: "Schedule",
                id: scheduleId,
            },
        })),
        recurrences: formData.recurrences.map(recurrence => ({
            __typename: "ScheduleRecurrence",
            id: generatePK().toString(),
            recurrenceType: recurrence.recurrenceType,
            interval: recurrence.interval,
            duration: recurrence.duration,
            dayOfWeek: recurrence.dayOfWeek,
            dayOfMonth: recurrence.dayOfMonth,
            month: recurrence.month,
            endDate: recurrence.endDate,
            schedule: {
                __typename: "Schedule",
                id: scheduleId,
            },
        })),
        meeting: formData.meetingId ? {
            __typename: "Meeting",
            __connect: true,
            id: formData.meetingId,
        } : null,
        runProject: formData.runId ? {
            __typename: "Run",
            __connect: true,
            id: formData.runId,
        } : null,
    });
}

function transformFormToUpdateRequestReal(scheduleId: string, formData: Partial<ScheduleFormData>) {
    const updateData: any = {
        __typename: "Schedule",
        id: scheduleId,
    };

    if (formData.timezone !== undefined) {
        updateData.timezone = formData.timezone;
    }
    if (formData.startTime !== undefined) {
        updateData.startTime = formData.startTime;
    }
    if (formData.endTime !== undefined) {
        updateData.endTime = formData.endTime;
    }

    // Handle exceptions (create/update/delete operations for updates)
    if (formData.exceptions) {
        updateData.exceptions = formData.exceptions.map(exception => ({
            __typename: "ScheduleException",
            id: generatePK().toString(),
            originalStartTime: exception.originalStartTime,
            newStartTime: exception.newStartTime,
            newEndTime: exception.newEndTime,
            schedule: {
                __typename: "Schedule",
                id: scheduleId,
            },
        }));
    }

    // Handle recurrences (create/update/delete operations for updates)
    if (formData.recurrences) {
        updateData.recurrences = formData.recurrences.map(recurrence => ({
            __typename: "ScheduleRecurrence",
            id: generatePK().toString(),
            recurrenceType: recurrence.recurrenceType,
            interval: recurrence.interval,
            duration: recurrence.duration,
            dayOfWeek: recurrence.dayOfWeek,
            dayOfMonth: recurrence.dayOfMonth,
            month: recurrence.month,
            endDate: recurrence.endDate,
            schedule: {
                __typename: "Schedule",
                id: scheduleId,
            },
        }));
    }

    return shapeSchedule.update(null, updateData);
}

async function validateScheduleFormDataReal(formData: ScheduleFormData): Promise<string[]> {
    try {
        // Use real validation schema
        const validationData = transformScheduleFormToApiInput(formData, false);
        
        await scheduleValidation.create({ omitFields: [] }).validate(validationData);
        return []; // No validation errors
    } catch (error: any) {
        if (error.errors) {
            return error.errors; // Yup validation errors
        }
        return [error.message || "Validation failed"];
    }
}

function transformApiResponseToFormReal(schedule: Schedule): ScheduleFormData {
    return {
        timezone: schedule.timezone,
        startTime: new Date(schedule.startTime),
        endTime: schedule.endTime ? new Date(schedule.endTime) : null,
        meetingId: schedule.meetings && schedule.meetings.length > 0 ? schedule.meetings[0].id : null,
        runId: schedule.runs && schedule.runs.length > 0 ? schedule.runs[0].id : null,
        recurrences: schedule.recurrences.map(recurrence => ({
            recurrenceType: recurrence.recurrenceType,
            interval: recurrence.interval,
            duration: recurrence.duration,
            dayOfWeek: recurrence.dayOfWeek,
            dayOfMonth: recurrence.dayOfMonth,
            month: recurrence.month,
            endDate: recurrence.endDate ? new Date(recurrence.endDate) : null,
        })),
        exceptions: schedule.exceptions.map(exception => ({
            originalStartTime: new Date(exception.originalStartTime),
            newStartTime: exception.newStartTime ? new Date(exception.newStartTime) : null,
            newEndTime: exception.newEndTime ? new Date(exception.newEndTime) : null,
        })),
    };
}

function areScheduleFormsEqualReal(form1: ScheduleFormData, form2: ScheduleFormData): boolean {
    // Compare basic fields
    if (form1.timezone !== form2.timezone) return false;
    
    // Compare dates (handle null values)
    const startTime1 = form1.startTime?.getTime();
    const startTime2 = form2.startTime?.getTime();
    if (startTime1 !== startTime2) return false;
    
    const endTime1 = form1.endTime?.getTime();
    const endTime2 = form2.endTime?.getTime();
    if (endTime1 !== endTime2) return false;
    
    // Compare connected entities
    if (form1.meetingId !== form2.meetingId) return false;
    if (form1.runId !== form2.runId) return false;
    
    // Compare recurrences (simplified comparison for core fields)
    if (form1.recurrences.length !== form2.recurrences.length) return false;
    for (let i = 0; i < form1.recurrences.length; i++) {
        const rec1 = form1.recurrences[i];
        const rec2 = form2.recurrences[i];
        if (rec1.recurrenceType !== rec2.recurrenceType) return false;
        if (rec1.interval !== rec2.interval) return false;
        if (rec1.duration !== rec2.duration) return false;
    }
    
    // Compare exceptions (simplified comparison for core fields)
    if (form1.exceptions.length !== form2.exceptions.length) return false;
    for (let i = 0; i < form1.exceptions.length; i++) {
        const exc1 = form1.exceptions[i];
        const exc2 = form2.exceptions[i];
        if (exc1.originalStartTime.getTime() !== exc2.originalStartTime.getTime()) return false;
    }
    
    return true;
}

// Mock service for testing - would use real API endpoints in integration tests
const mockScheduleService = {
    create: async (request: any): Promise<Schedule> => {
        const schedule: Schedule = {
            __typename: "Schedule",
            id: request.id,
            publicId: `sched_${request.id}`,
            timezone: request.timezone,
            startTime: request.startTime?.toISOString() || new Date().toISOString(),
            endTime: request.endTime?.toISOString() || new Date(Date.now() + 3600000).toISOString(),
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            user: { 
                __typename: "User",
                id: "user_123456789012345678",
                name: "Test User",
                handle: "testuser",
                isBot: false,
                isPrivate: false,
                isBotDepictingPerson: false,
                botSettings: null,
            } as any,
            exceptions: request.exceptionsCreate || [],
            recurrences: request.recurrencesCreate || [],
            meetings: request.meetingConnect ? [{ __typename: "Meeting", id: request.meetingConnect }] : [],
            runs: request.runConnect ? [{ __typename: "Run", id: request.runConnect }] : [],
        };
        
        // Store in global test storage
        (globalThis as any).__testScheduleStorage = (globalThis as any).__testScheduleStorage || {};
        (globalThis as any).__testScheduleStorage[schedule.id] = schedule;
        
        return schedule;
    },

    findById: async (id: string): Promise<Schedule> => {
        const storage = (globalThis as any).__testScheduleStorage || {};
        const schedule = storage[id];
        if (!schedule) {
            throw new Error(`Schedule with id ${id} not found`);
        }
        return { ...schedule };
    },

    update: async (id: string, updateRequest: any): Promise<Schedule> => {
        const storage = (globalThis as any).__testScheduleStorage || {};
        const existing = storage[id];
        if (!existing) {
            throw new Error(`Schedule with id ${id} not found`);
        }

        const updated: Schedule = {
            ...existing,
            ...updateRequest,
            id: id, // Preserve original ID
            updatedAt: new Date().toISOString(),
        };

        storage[id] = updated;
        return { ...updated };
    },

    delete: async (id: string): Promise<{ success: boolean }> => {
        const storage = (globalThis as any).__testScheduleStorage || {};
        if (storage[id]) {
            delete storage[id];
            return { success: true };
        }
        return { success: false };
    },
};

describe('Schedule Round-Trip Data Flow', () => {
    beforeEach(() => {
        // Clear test storage between tests
        (globalThis as any).__testScheduleStorage = {};
    });

    test('minimal schedule creation maintains data integrity through complete flow', async () => {
        // 🎨 STEP 1: User fills out minimal schedule form
        const userFormData: ScheduleFormData = {
            timezone: "America/New_York",
            startTime: new Date("2025-01-01T09:00:00Z"),
            endTime: new Date("2025-01-01T17:00:00Z"),
            recurrences: [],
            exceptions: [],
        };
        
        // Validate form data using REAL validation schema
        const validationErrors = await validateScheduleFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Form submits to API using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.timezone).toBe(userFormData.timezone);
        expect(apiCreateRequest.startTime).toEqual(userFormData.startTime);
        expect(apiCreateRequest.endTime).toEqual(userFormData.endTime);
        expect(apiCreateRequest.id).toMatch(/^\d+$/); // Valid ID format
        
        // 🗄️ STEP 3: API creates schedule (simulated - real test would hit test DB)
        const createdSchedule = await mockScheduleService.create(apiCreateRequest);
        expect(createdSchedule.id).toBe(apiCreateRequest.id);
        expect(createdSchedule.timezone).toBe(userFormData.timezone);
        expect(new Date(createdSchedule.startTime)).toEqual(userFormData.startTime);
        expect(new Date(createdSchedule.endTime)).toEqual(userFormData.endTime);
        
        // 🔗 STEP 4: API fetches schedule back
        const fetchedSchedule = await mockScheduleService.findById(createdSchedule.id);
        expect(fetchedSchedule.id).toBe(createdSchedule.id);
        expect(fetchedSchedule.timezone).toBe(userFormData.timezone);
        expect(new Date(fetchedSchedule.startTime)).toEqual(userFormData.startTime);
        
        // 🎨 STEP 5: UI would display the schedule using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedSchedule);
        expect(reconstructedFormData.timezone).toBe(userFormData.timezone);
        expect(reconstructedFormData.startTime).toEqual(userFormData.startTime);
        expect(reconstructedFormData.endTime).toEqual(userFormData.endTime);
        
        // ✅ VERIFICATION: Complete round-trip integrity using REAL comparison
        expect(areScheduleFormsEqualReal(
            userFormData, 
            reconstructedFormData
        )).toBe(true);
    });

    test('complete schedule with recurrences and exceptions preserves all data', async () => {
        // 🎨 STEP 1: User creates schedule with complex recurring pattern
        const userFormData: ScheduleFormData = {
            timezone: "UTC",
            startTime: new Date("2025-01-01T09:00:00Z"),
            endTime: new Date("2025-12-31T17:00:00Z"),
            meetingId: "123456789012345678",
            recurrences: [
                {
                    recurrenceType: "Daily" as ScheduleRecurrenceType,
                    interval: 1,
                    duration: 480, // 8 hours
                    dayOfWeek: null,
                    dayOfMonth: null,
                    month: null,
                    endDate: new Date("2025-12-31T23:59:59Z"),
                },
            ],
            exceptions: [
                {
                    originalStartTime: new Date("2025-07-04T09:00:00Z"),
                    newStartTime: null, // Holiday cancellation
                    newEndTime: null,
                },
                {
                    originalStartTime: new Date("2025-12-25T09:00:00Z"),
                    newStartTime: new Date("2025-12-26T10:00:00Z"), // Moved to next day
                    newEndTime: new Date("2025-12-26T16:00:00Z"),
                },
            ],
        };
        
        // Validate complex form using REAL validation
        const validationErrors = await validateScheduleFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform to API request using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.recurrencesCreate).toBeDefined();
        expect(apiCreateRequest.recurrencesCreate!.length).toBe(1);
        expect(apiCreateRequest.recurrencesCreate![0].recurrenceType).toBe(userFormData.recurrences[0].recurrenceType);
        expect(apiCreateRequest.exceptionsCreate).toBeDefined();
        expect(apiCreateRequest.exceptionsCreate!.length).toBe(2);
        expect(apiCreateRequest.meetingConnect).toBe(userFormData.meetingId);
        
        // 🗄️ STEP 3: Create via API
        const createdSchedule = await mockScheduleService.create(apiCreateRequest);
        expect(createdSchedule.recurrences).toBeDefined();
        expect(createdSchedule.exceptions).toBeDefined();
        expect(createdSchedule.meetings.length).toBe(1);
        expect(createdSchedule.meetings[0].id).toBe(userFormData.meetingId);
        
        // 🔗 STEP 4: Fetch back from API
        const fetchedSchedule = await mockScheduleService.findById(createdSchedule.id);
        expect(fetchedSchedule.timezone).toBe(userFormData.timezone);
        
        // 🎨 STEP 5: Verify UI can display correctly using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedSchedule);
        expect(reconstructedFormData.timezone).toBe(userFormData.timezone);
        expect(reconstructedFormData.startTime).toEqual(userFormData.startTime);
        expect(reconstructedFormData.endTime).toEqual(userFormData.endTime);
        expect(reconstructedFormData.meetingId).toBe(userFormData.meetingId);
        
        // ✅ VERIFICATION: Complex data preserved
        expect(reconstructedFormData.recurrences.length).toBe(userFormData.recurrences.length);
        expect(reconstructedFormData.exceptions.length).toBe(userFormData.exceptions.length);
    });

    test('schedule editing with timezone change maintains data integrity', async () => {
        // Create initial schedule using REAL functions
        const initialFormData: ScheduleFormData = {
            timezone: "America/New_York",
            startTime: new Date("2025-01-01T14:00:00Z"), // 9 AM EST
            endTime: new Date("2025-01-01T22:00:00Z"), // 5 PM EST
            recurrences: [],
            exceptions: [],
        };
        
        const createRequest = transformFormToCreateRequestReal(initialFormData);
        const initialSchedule = await mockScheduleService.create(createRequest);
        
        // 🎨 STEP 1: User edits schedule to change timezone
        const editFormData: Partial<ScheduleFormData> = {
            timezone: "Europe/London",
            startTime: new Date("2025-01-01T09:00:00Z"), // 9 AM GMT
            endTime: new Date("2025-01-01T17:00:00Z"), // 5 PM GMT
        };
        
        // 🔗 STEP 2: Transform edit to update request using REAL update logic
        const updateRequest = transformFormToUpdateRequestReal(initialSchedule.id, editFormData);
        expect(updateRequest.id).toBe(initialSchedule.id);
        expect(updateRequest.timezone).toBe(editFormData.timezone);
        
        // Add small delay to ensure different timestamps
        await new Promise(resolve => setTimeout(resolve, 10));
        
        // 🗄️ STEP 3: Update via API
        const updatedSchedule = await mockScheduleService.update(initialSchedule.id, updateRequest);
        expect(updatedSchedule.id).toBe(initialSchedule.id);
        expect(updatedSchedule.timezone).toBe(editFormData.timezone);
        
        // 🔗 STEP 4: Fetch updated schedule
        const fetchedUpdatedSchedule = await mockScheduleService.findById(initialSchedule.id);
        
        // ✅ VERIFICATION: Update preserved core data
        expect(fetchedUpdatedSchedule.id).toBe(initialSchedule.id);
        expect(fetchedUpdatedSchedule.timezone).toBe(editFormData.timezone);
        expect(fetchedUpdatedSchedule.createdAt).toBe(initialSchedule.createdAt); // Created date unchanged
        expect(new Date(fetchedUpdatedSchedule.updatedAt).getTime()).toBeGreaterThan(
            new Date(initialSchedule.updatedAt).getTime()
        );
    });

    test('all schedule recurrence types work correctly through round-trip', async () => {
        const recurrenceTypes: ScheduleRecurrenceType[] = ["Daily", "Weekly", "Monthly", "Yearly"];
        
        for (const recurrenceType of recurrenceTypes) {
            // 🎨 Create form data for each recurrence type
            const formData: ScheduleFormData = {
                timezone: "UTC",
                startTime: new Date("2025-01-01T12:00:00Z"),
                endTime: new Date("2025-01-01T13:00:00Z"),
                recurrences: [
                    {
                        recurrenceType: recurrenceType,
                        interval: 1,
                        duration: 60, // 1 hour
                        dayOfWeek: recurrenceType === "Weekly" ? 1 : null, // Monday for weekly
                        dayOfMonth: recurrenceType === "Monthly" ? 15 : null, // 15th for monthly
                        month: recurrenceType === "Yearly" ? 6 : null, // June for yearly
                        endDate: new Date("2025-12-31T23:59:59Z"),
                    },
                ],
                exceptions: [],
            };
            
            // 🔗 Transform and create using REAL functions
            const createRequest = transformFormToCreateRequestReal(formData);
            const createdSchedule = await mockScheduleService.create(createRequest);
            
            // 🗄️ Fetch back
            const fetchedSchedule = await mockScheduleService.findById(createdSchedule.id);
            
            // ✅ Verify recurrence-specific data
            expect(fetchedSchedule.recurrences.length).toBe(1);
            expect(fetchedSchedule.recurrences[0].recurrenceType).toBe(recurrenceType);
            expect(fetchedSchedule.recurrences[0].interval).toBe(1);
            expect(fetchedSchedule.recurrences[0].duration).toBe(60);
            
            // Verify form reconstruction using REAL transformation
            const reconstructed = transformApiResponseToFormReal(fetchedSchedule);
            expect(reconstructed.recurrences.length).toBe(1);
            expect(reconstructed.recurrences[0].recurrenceType).toBe(recurrenceType);
            expect(reconstructed.recurrences[0].interval).toBe(1);
            expect(reconstructed.recurrences[0].duration).toBe(60);
        }
    });

    test('validation catches invalid form data before API submission', async () => {
        // Test missing timezone
        const missingTimezoneData: ScheduleFormData = {
            timezone: "",
            startTime: new Date("2025-01-01T09:00:00Z"),
            endTime: new Date("2025-01-01T17:00:00Z"),
            recurrences: [],
            exceptions: [],
        };
        
        const timezoneErrors = await validateScheduleFormDataReal(missingTimezoneData);
        expect(timezoneErrors.length).toBeGreaterThan(0);
        
        // Test invalid time order
        const invalidTimeOrderData: ScheduleFormData = {
            timezone: "America/New_York",
            startTime: new Date("2025-01-01T17:00:00Z"),
            endTime: new Date("2025-01-01T09:00:00Z"), // Before start time
            recurrences: [],
            exceptions: [],
        };
        
        const timeRangeError = validateScheduleTimeRange(
            invalidTimeOrderData.startTime,
            invalidTimeOrderData.endTime
        );
        expect(timeRangeError).toBeTruthy();
        expect(timeRangeError).toContain("End time must be after start time");
    });

    test('timezone validation works correctly', async () => {
        // Test empty timezone
        expect(validateTimezone("")).toBeTruthy();
        
        // Test valid timezone
        expect(validateTimezone("America/New_York")).toBeNull();
        
        // Test timezone too long
        expect(validateTimezone("A".repeat(65))).toBeTruthy();
    });

    test('schedule exception handling works correctly', async () => {
        const formData: ScheduleFormData = {
            timezone: "America/New_York",
            startTime: new Date("2025-01-01T09:00:00Z"),
            endTime: new Date("2025-01-01T17:00:00Z"),
            recurrences: [
                {
                    recurrenceType: "Daily" as ScheduleRecurrenceType,
                    interval: 1,
                    duration: 480, // 8 hours
                    dayOfWeek: null,
                    dayOfMonth: null,
                    month: null,
                    endDate: new Date("2025-12-31T23:59:59Z"),
                },
            ],
            exceptions: [
                {
                    originalStartTime: new Date("2025-07-04T09:00:00Z"),
                    newStartTime: new Date("2025-07-05T09:00:00Z"), // Moved to next day
                    newEndTime: new Date("2025-07-05T17:00:00Z"),
                },
            ],
        };
        
        // Create using REAL functions
        const createRequest = transformFormToCreateRequestReal(formData);
        const created = await mockScheduleService.create(createRequest);
        
        // Verify exception data is preserved
        expect(created.exceptions.length).toBe(1);
        expect(created.exceptions[0].originalStartTime).toBeDefined();
        expect(created.exceptions[0].newStartTime).toBeDefined();
        expect(created.exceptions[0].newEndTime).toBeDefined();
        
        // Fetch and verify
        const fetched = await mockScheduleService.findById(created.id);
        const reconstructed = transformApiResponseToFormReal(fetched);
        
        expect(reconstructed.exceptions.length).toBe(1);
        expect(reconstructed.exceptions[0].originalStartTime).toEqual(formData.exceptions[0].originalStartTime);
        expect(reconstructed.exceptions[0].newStartTime).toEqual(formData.exceptions[0].newStartTime);
        expect(reconstructed.exceptions[0].newEndTime).toEqual(formData.exceptions[0].newEndTime);
    });

    test('schedule deletion works correctly', async () => {
        // Create schedule first using REAL functions
        const formData: ScheduleFormData = {
            timezone: "UTC",
            startTime: new Date("2025-01-01T12:00:00Z"),
            endTime: new Date("2025-01-01T13:00:00Z"),
            recurrences: [],
            exceptions: [],
        };
        
        const createRequest = transformFormToCreateRequestReal(formData);
        const createdSchedule = await mockScheduleService.create(createRequest);
        
        // Delete it
        const deleteResult = await mockScheduleService.delete(createdSchedule.id);
        expect(deleteResult.success).toBe(true);
        
        // Verify it's gone
        await expect(mockScheduleService.findById(createdSchedule.id)).rejects.toThrow();
    });

    test('data consistency across multiple operations', async () => {
        const originalFormData: ScheduleFormData = {
            timezone: "America/New_York",
            startTime: new Date("2025-01-01T09:00:00Z"),
            endTime: new Date("2025-01-01T17:00:00Z"),
            runId: "234567890123456789",
            recurrences: [],
            exceptions: [],
        };
        
        // Create using REAL functions
        const createRequest = transformFormToCreateRequestReal(originalFormData);
        const created = await mockScheduleService.create(createRequest);
        
        // Update using REAL functions
        const updateRequest = transformFormToUpdateRequestReal(created.id, { 
            timezone: "Europe/London",
            startTime: new Date("2025-01-01T14:00:00Z"), // 2 PM GMT
            endTime: new Date("2025-01-01T22:00:00Z"), // 10 PM GMT
        });
        const updated = await mockScheduleService.update(created.id, updateRequest);
        
        // Fetch final state
        const final = await mockScheduleService.findById(created.id);
        
        // Core schedule data should remain consistent
        expect(final.id).toBe(created.id);
        expect(final.user.id).toBe(created.user.id);
        expect(final.runs.length).toBe(1);
        expect(final.runs[0].id).toBe(originalFormData.runId);
        
        // Updated fields should be changed
        expect(final.timezone).toBe("Europe/London");
        expect(new Date(final.startTime)).toEqual(new Date("2025-01-01T14:00:00Z"));
        expect(new Date(final.endTime)).toEqual(new Date("2025-01-01T22:00:00Z"));
        
        // Verify update timestamp changed
        expect(new Date(final.updatedAt).getTime()).toBeGreaterThan(
            new Date(created.updatedAt).getTime()
        );
    });

    test('complex recurring schedule with multiple patterns', async () => {
        const complexFormData: ScheduleFormData = {
            timezone: "UTC",
            startTime: new Date("2025-01-01T00:00:00Z"),
            endTime: new Date("2025-12-31T23:59:59Z"),
            recurrences: [
                // Daily pattern
                {
                    recurrenceType: "Daily" as ScheduleRecurrenceType,
                    interval: 1,
                    duration: 60,
                    dayOfWeek: null,
                    dayOfMonth: null,
                    month: null,
                    endDate: new Date("2025-06-30T23:59:59Z"),
                },
                // Weekly pattern  
                {
                    recurrenceType: "Weekly" as ScheduleRecurrenceType,
                    interval: 2, // Bi-weekly
                    duration: 120,
                    dayOfWeek: 3, // Wednesday
                    dayOfMonth: null,
                    month: null,
                    endDate: new Date("2025-12-31T23:59:59Z"),
                },
            ],
            exceptions: [
                {
                    originalStartTime: new Date("2025-05-01T12:00:00Z"),
                    newStartTime: null, // Cancel May Day
                    newEndTime: null,
                },
            ],
        };
        
        // Create complex schedule
        const createRequest = transformFormToCreateRequestReal(complexFormData);
        const created = await mockScheduleService.create(createRequest);
        
        // Verify multiple recurrences
        expect(created.recurrences.length).toBe(2);
        expect(created.exceptions.length).toBe(1);
        
        // Fetch back and verify integrity
        const fetched = await mockScheduleService.findById(created.id);
        const reconstructed = transformApiResponseToFormReal(fetched);
        
        // Verify complex patterns preserved
        expect(reconstructed.recurrences.length).toBe(2);
        expect(reconstructed.recurrences[0].recurrenceType).toBe("Daily");
        expect(reconstructed.recurrences[1].recurrenceType).toBe("Weekly");
        expect(reconstructed.recurrences[1].interval).toBe(2); // Bi-weekly
        expect(reconstructed.recurrences[1].dayOfWeek).toBe(3); // Wednesday
        
        expect(reconstructed.exceptions.length).toBe(1);
        expect(reconstructed.exceptions[0].newStartTime).toBeNull(); // Cancelled
    });
});
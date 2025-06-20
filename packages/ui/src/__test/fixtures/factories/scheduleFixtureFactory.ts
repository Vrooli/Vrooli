/* c8 ignore start */
/**
 * Schedule fixture factory implementation
 * 
 * This implements the complete pattern for Schedule object fixtures with full
 * type safety and integration with @vrooli/shared.
 */

import {
    type Schedule,
    type ScheduleCreateInput,
    type ScheduleUpdateInput,
    ScheduleRecurrenceType
} from "@vrooli/shared";
import { shapeSchedule } from "@vrooli/shared";
import { scheduleValidation } from "@vrooli/shared";
import { generatePK } from "@vrooli/shared";
import { BaseFormFixtureFactory } from "../BaseFormFixtureFactory.js";
import { BaseRoundTripOrchestrator } from "../BaseRoundTripOrchestrator.js";
import { BaseMSWHandlerFactory } from "../BaseMSWHandlerFactory.js";
import { createValidationAdapter } from "../utils/integration.js";
import type {
    UIFixtureFactory,
    FormFixtureFactory,
    RoundTripOrchestrator,
    MSWHandlerFactory,
    UIStateFixtureFactory,
    ComponentTestUtils,
    TestAPIClient,
    DatabaseVerifier
} from "../types.js";
import { registerFixture } from "./index.js";

/**
 * Schedule form data type
 * 
 * This includes UI-specific fields that don't exist in the API input type.
 */
export interface ScheduleFormData extends Record<string, unknown> {
    name?: string;
    startTime: string;
    endTime?: string;
    timezone: string;
    focusMode?: boolean;
    recurrence?: {
        type: "daily" | "weekly" | "monthly" | "yearly" | "none";
        interval?: number;
        daysOfWeek?: number[];
        dayOfMonth?: number;
        month?: number;
        endDate?: string;
        duration?: number;
    };
    exceptions?: Array<{
        id?: string;
        originalStartTime: string;
        newStartTime?: string;
        newEndTime?: string;
        isDeleted?: boolean;
    }>;
    isActive?: boolean; // UI-specific
    reminderBefore?: number; // UI-specific - minutes before start
    colorCode?: string; // UI-specific - calendar color
}

/**
 * Schedule UI state type
 */
export interface ScheduleUIState {
    isLoading: boolean;
    schedule: Schedule | null;
    error: string | null;
    isEditing: boolean;
    hasUnsavedChanges: boolean;
    validationErrors: Record<string, string>;
    recurrencePreview: Array<{ start: Date; end: Date }> | null;
}

/**
 * Schedule form fixture factory
 */
class ScheduleFormFixtureFactory extends BaseFormFixtureFactory<ScheduleFormData, ScheduleCreateInput> {
    constructor() {
        super({
            scenarios: {
                minimal: {
                    startTime: "2025-01-01T09:00",
                    timezone: "America/New_York"
                },
                complete: {
                    name: "Daily Standup Meeting",
                    startTime: "2025-01-01T09:00",
                    endTime: "2025-01-01T10:00",
                    timezone: "America/New_York",
                    focusMode: true,
                    recurrence: {
                        type: "daily",
                        interval: 1,
                        duration: 60, // 1 hour
                        endDate: "2025-12-31T23:59"
                    },
                    exceptions: [
                        {
                            originalStartTime: "2025-07-04T09:00",
                            newStartTime: "2025-07-04T10:00",
                            newEndTime: "2025-07-04T11:00"
                        }
                    ],
                    isActive: true,
                    reminderBefore: 15,
                    colorCode: "#4285f4"
                },
                invalid: {
                    startTime: "invalid-date",
                    timezone: "", // Empty timezone
                    endTime: "2025-01-01T08:00", // Before start time
                    recurrence: {
                        type: "weekly",
                        interval: 0 // Invalid interval
                    }
                },
                recurring: {
                    name: "Weekly Team Meeting",
                    startTime: "2025-01-01T14:00",
                    endTime: "2025-01-01T15:30",
                    timezone: "America/New_York",
                    recurrence: {
                        type: "weekly",
                        interval: 1,
                        daysOfWeek: [1, 3], // Monday and Wednesday
                        duration: 90,
                        endDate: "2025-06-30T23:59"
                    },
                    isActive: true,
                    reminderBefore: 30
                },
                oneTime: {
                    name: "Project Kickoff",
                    startTime: "2025-03-15T10:00",
                    endTime: "2025-03-15T12:00",
                    timezone: "UTC",
                    recurrence: {
                        type: "none"
                    },
                    isActive: true,
                    reminderBefore: 60,
                    colorCode: "#34a853"
                },
                withExceptions: {
                    name: "Daily Check-in",
                    startTime: "2025-01-01T08:30",
                    endTime: "2025-01-01T09:00",
                    timezone: "Europe/London",
                    recurrence: {
                        type: "daily",
                        interval: 1,
                        duration: 30
                    },
                    exceptions: [
                        {
                            originalStartTime: "2025-12-25T08:30",
                            isDeleted: true // Christmas - cancelled
                        },
                        {
                            originalStartTime: "2025-01-01T08:30",
                            newStartTime: "2025-01-01T10:00",
                            newEndTime: "2025-01-01T10:30"
                        }
                    ],
                    isActive: true
                },
                meeting: {
                    name: "Board Meeting",
                    startTime: "2025-02-15T14:00",
                    endTime: "2025-02-15T16:00",
                    timezone: "America/New_York",
                    recurrence: {
                        type: "monthly",
                        interval: 1,
                        dayOfMonth: 15,
                        duration: 120
                    },
                    focusMode: true,
                    isActive: true,
                    reminderBefore: 24 * 60, // 1 day before
                    colorCode: "#ea4335"
                }
            },
            
            validate: createValidationAdapter<ScheduleFormData>(
                async (data: ScheduleFormData) => {
                    // Additional UI validation
                    const errors: string[] = [];
                    
                    if (data.endTime && data.startTime) {
                        const startDate = new Date(data.startTime as string);
                        const endDate = new Date(data.endTime as string);
                        
                        if (endDate <= startDate) {
                            errors.push("endTime: End time must be after start time");
                        }
                    }
                    
                    if (data.recurrence && typeof data.recurrence === 'object') {
                        const recurrence = data.recurrence as ScheduleFormData['recurrence'];
                        if (recurrence?.type !== "none" && recurrence?.interval && recurrence.interval < 1) {
                            errors.push("recurrence.interval: Interval must be at least 1");
                        }
                    }
                    
                    if (typeof data.reminderBefore === 'number' && data.reminderBefore < 0) {
                        errors.push("reminderBefore: Reminder time cannot be negative");
                    }
                    
                    if (errors.length > 0) {
                        return { isValid: false, errors };
                    }
                    
                    // Use shared validation for the rest
                    const apiInput = this.shapeToAPI!(data);
                    return scheduleValidation.create.validate(apiInput);
                }
            ),
            
            shapeToAPI: (formData) => {
                // Transform to ScheduleCreateInput
                const startTime = formData.startTime ? new Date(formData.startTime) : undefined;
                const endTime = formData.endTime ? new Date(formData.endTime) : undefined;
                
                const createInput: ScheduleCreateInput = {
                    id: generatePK().toString(),
                    timezone: formData.timezone,
                    ...(startTime && { startTime }),
                    ...(endTime && { endTime })
                };
                
                // Add recurrence if specified
                if (formData.recurrence && formData.recurrence.type !== "none") {
                    const recurrenceTypeMap = {
                        daily: ScheduleRecurrenceType.Daily,
                        weekly: ScheduleRecurrenceType.Weekly,
                        monthly: ScheduleRecurrenceType.Monthly,
                        yearly: ScheduleRecurrenceType.Yearly
                    };
                    
                    createInput.recurrencesCreate = [{
                        id: generatePK().toString(),
                        recurrenceType: recurrenceTypeMap[formData.recurrence.type],
                        interval: formData.recurrence.interval || 1,
                        duration: formData.recurrence.duration || 60,
                        dayOfWeek: formData.recurrence.daysOfWeek?.[0] || null,
                        dayOfMonth: formData.recurrence.dayOfMonth || null,
                        month: formData.recurrence.month || null,
                        endDate: formData.recurrence.endDate ? new Date(formData.recurrence.endDate) : null,
                        scheduleConnect: createInput.id
                    }];
                }
                
                // Add exceptions if specified
                if (formData.exceptions && formData.exceptions.length > 0) {
                    createInput.exceptionsCreate = formData.exceptions
                        .filter(exc => !exc.isDeleted)
                        .map(exc => ({
                            id: exc.id || generatePK().toString(),
                            originalStartTime: new Date(exc.originalStartTime),
                            newStartTime: exc.newStartTime ? new Date(exc.newStartTime) : new Date(exc.originalStartTime),
                            newEndTime: exc.newEndTime ? new Date(exc.newEndTime) : undefined,
                            scheduleConnect: createInput.id
                        }));
                }
                
                return createInput;
            }
        });
    }
    
    /**
     * Create schedule update form data
     */
    createUpdateFormData(scenario: "minimal" | "complete" = "minimal"): Partial<ScheduleFormData> {
        if (scenario === "minimal") {
            return {
                startTime: "2025-01-02T10:00"
            };
        }
        
        return {
            name: "Updated Meeting Name",
            startTime: "2025-01-02T10:00",
            endTime: "2025-01-02T11:00",
            timezone: "Europe/London",
            reminderBefore: 30,
            colorCode: "#ff9800"
        };
    }
}

/**
 * Schedule MSW handler factory
 */
class ScheduleMSWHandlerFactory extends BaseMSWHandlerFactory<ScheduleCreateInput, ScheduleUpdateInput, Schedule> {
    constructor() {
        super({
            baseUrl: "/api",
            endpoints: {
                create: "/schedule",
                update: "/schedule",
                delete: "/schedule",
                find: "/schedule",
                list: "/schedules"
            },
            successResponses: {
                create: (input) => ({
                    id: input.id,
                    timezone: input.timezone,
                    startTime: input.startTime || new Date("2025-01-01T09:00:00Z"),
                    endTime: input.endTime || new Date("2025-01-01T10:00:00Z"),
                    publicId: `schedule_${input.id}`,
                    createdAt: new Date(),
                    updatedAt: new Date(),
                    exceptions: [],
                    meetings: [],
                    recurrences: [],
                    runs: [],
                    user: {} as any
                }) as Schedule,
                update: (input) => ({
                    id: input.id,
                    timezone: input.timezone || "UTC",
                    startTime: input.startTime || new Date("2025-01-01T09:00:00Z"),
                    endTime: input.endTime,
                    publicId: `schedule_${input.id}`,
                    createdAt: new Date(),
                    updatedAt: new Date(),
                    exceptions: [],
                    meetings: [],
                    recurrences: [],
                    runs: [],
                    user: {} as any,
                    ...input
                }) as Schedule,
                find: (id) => ({
                    id,
                    timezone: "UTC",
                    startTime: new Date("2025-01-01T09:00:00Z"),
                    endTime: new Date("2025-01-01T10:00:00Z"),
                    publicId: `schedule_${id}`,
                    createdAt: new Date(),
                    updatedAt: new Date(),
                    exceptions: [],
                    meetings: [],
                    recurrences: [],
                    runs: [],
                    user: {} as any
                }) as Schedule
            },
            validate: {
                create: (input) => {
                    const errors: string[] = [];
                    
                    if (!input.timezone) {
                        errors.push("Timezone is required");
                    }
                    
                    if (input.endTime && input.startTime && input.endTime <= input.startTime) {
                        errors.push("End time must be after start time");
                    }
                    
                    return {
                        isValid: errors.length === 0,
                        errors
                    };
                }
            }
        });
    }
    
    /**
     * Create recurring schedule handlers
     */
    createRecurringHandlers() {
        return this.createCustomHandler({
            method: "GET",
            path: "/schedule/recurring/:id",
            response: {
                id: generatePK().toString(),
                timezone: "UTC",
                startTime: new Date("2025-01-01T09:00:00Z"),
                endTime: new Date("2025-01-01T10:00:00Z"),
                publicId: `schedule_${generatePK()}`,
                createdAt: new Date(),
                updatedAt: new Date(),
                exceptions: [],
                meetings: [],
                runs: [],
                user: {} as any,
                recurrences: [{
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceType.Daily,
                    interval: 1,
                    duration: 60
                }]
            }
        });
    }
}

/**
 * Schedule UI state fixture factory
 */
class ScheduleUIStateFixtureFactory implements UIStateFixtureFactory<ScheduleUIState> {
    createLoadingState(context?: { type: string }): ScheduleUIState {
        return {
            isLoading: true,
            schedule: null,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false,
            validationErrors: {},
            recurrencePreview: null
        };
    }
    
    createErrorState(error: { message: string }): ScheduleUIState {
        return {
            isLoading: false,
            schedule: null,
            error: error.message,
            isEditing: false,
            hasUnsavedChanges: false,
            validationErrors: {},
            recurrencePreview: null
        };
    }
    
    createSuccessState(data: Schedule): ScheduleUIState {
        return {
            isLoading: false,
            schedule: data,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false,
            validationErrors: {},
            recurrencePreview: null
        };
    }
    
    createEmptyState(): ScheduleUIState {
        return {
            isLoading: false,
            schedule: null,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false,
            validationErrors: {},
            recurrencePreview: null
        };
    }
    
    transitionToLoading(currentState: ScheduleUIState): ScheduleUIState {
        return {
            ...currentState,
            isLoading: true,
            error: null
        };
    }
    
    transitionToSuccess(currentState: ScheduleUIState, data: Schedule): ScheduleUIState {
        return {
            ...currentState,
            isLoading: false,
            schedule: data,
            error: null,
            hasUnsavedChanges: false,
            validationErrors: {}
        };
    }
    
    transitionToError(currentState: ScheduleUIState, error: { message: string }): ScheduleUIState {
        return {
            ...currentState,
            isLoading: false,
            error: error.message
        };
    }
    
    /**
     * Create editing state with validation errors
     */
    createEditingState(schedule: Schedule, validationErrors: Record<string, string> = {}): ScheduleUIState {
        return {
            isLoading: false,
            schedule,
            error: null,
            isEditing: true,
            hasUnsavedChanges: Object.keys(validationErrors).length === 0,
            validationErrors,
            recurrencePreview: null
        };
    }
    
    /**
     * Create state with recurrence preview
     */
    createWithRecurrencePreview(schedule: Schedule, preview: Array<{ start: Date; end: Date }>): ScheduleUIState {
        return {
            isLoading: false,
            schedule,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false,
            validationErrors: {},
            recurrencePreview: preview
        };
    }
}

/**
 * Complete Schedule fixture factory
 */
export class ScheduleFixtureFactory implements UIFixtureFactory<
    ScheduleFormData,
    ScheduleCreateInput,
    ScheduleUpdateInput,
    Schedule,
    ScheduleUIState
> {
    readonly objectType = "schedule";
    
    form: ScheduleFormFixtureFactory;
    roundTrip: RoundTripOrchestrator<ScheduleFormData, Schedule>;
    handlers: ScheduleMSWHandlerFactory;
    states: ScheduleUIStateFixtureFactory;
    componentUtils: ComponentTestUtils<any>;
    
    constructor(apiClient: TestAPIClient, dbVerifier: DatabaseVerifier) {
        this.form = new ScheduleFormFixtureFactory();
        this.handlers = new ScheduleMSWHandlerFactory();
        this.states = new ScheduleUIStateFixtureFactory();
        
        // Initialize round-trip orchestrator
        this.roundTrip = new BaseRoundTripOrchestrator<ScheduleFormData, Schedule, ScheduleCreateInput>({
            apiClient,
            dbVerifier,
            formFixture: this.form,
            endpoints: {
                create: "/api/schedule",
                update: "/api/schedule",
                delete: "/api/schedule",
                find: "/api/schedule"
            },
            tableName: "schedule",
            fieldMappings: {
                startTime: "startTime",
                endTime: "endTime",
                timezone: "timezone"
            }
        });
        
        // Component utils would be initialized here
        this.componentUtils = {} as any; // Placeholder
    }
    
    createFormData(scenario: "minimal" | "complete" | string = "minimal"): ScheduleFormData {
        return this.form.createFormData(scenario);
    }
    
    createAPIInput(formData: ScheduleFormData): ScheduleCreateInput {
        return this.form.transformToAPIInput(formData);
    }
    
    createMockResponse(overrides?: Partial<Schedule>): Schedule {
        return {
            id: generatePK().toString(),
            timezone: "UTC",
            startTime: new Date("2025-01-01T09:00:00Z"),
            endTime: new Date("2025-01-01T10:00:00Z"),
            publicId: `schedule_${generatePK()}`,
            createdAt: new Date(),
            updatedAt: new Date(),
            exceptions: [],
            meetings: [],
            recurrences: [],
            runs: [],
            user: {} as any,
            ...overrides
        } as Schedule;
    }
    
    setupMSW(scenario: "success" | "error" | "loading" = "success"): void {
        // This would integrate with the MSW server
        // For now, it's a placeholder
        console.log(`Setting up MSW handlers for scenario: ${scenario}`);
    }
    
    async testCreateFlow(formData?: ScheduleFormData): Promise<Schedule> {
        const data = formData || this.createFormData("minimal");
        const result = await this.roundTrip.testCreateFlow(data);
        
        if (!result.success) {
            throw new Error(`Create flow failed: ${result.errors?.join(", ")}`);
        }
        
        return result.metadata?.id as unknown as Schedule;
    }
    
    async testUpdateFlow(id: string, updates: Partial<ScheduleFormData>): Promise<Schedule> {
        const result = await this.roundTrip.testUpdateFlow(id, updates);
        
        if (!result.success) {
            throw new Error(`Update flow failed: ${result.errors?.join(", ")}`);
        }
        
        return result.metadata?.updatedData as Schedule;
    }
    
    async testDeleteFlow(id: string): Promise<boolean> {
        const result = await this.roundTrip.testDeleteFlow(id);
        return result.success;
    }
    
    async testRoundTrip(formData?: ScheduleFormData) {
        const data = formData || this.createFormData("complete");
        const result = await this.roundTrip.executeFullCycle({
            formData: data,
            validateEachStep: true
        });
        
        if (!result.success) {
            throw new Error(`Round trip failed: ${result.errors?.join(", ")}`);
        }
        
        return {
            success: result.success,
            formData: data,
            apiResponse: result.data!.apiResponse,
            uiState: this.states.createSuccessState(result.data!.apiResponse)
        };
    }
    
    /**
     * Create a recurring schedule scenario
     */
    createRecurringSchedule(type: "daily" | "weekly" | "monthly" = "daily"): ScheduleFormData {
        const baseData = this.createFormData("recurring");
        if (baseData.recurrence) {
            baseData.recurrence.type = type;
        }
        return baseData;
    }
    
    /**
     * Create a schedule with specific exceptions
     */
    createScheduleWithExceptions(exceptionCount: number = 2): ScheduleFormData {
        const data = this.createFormData("withExceptions");
        if (data.exceptions) {
            data.exceptions = Array.from({ length: exceptionCount }, (_, i) => ({
                originalStartTime: `2025-0${i + 1}-15T09:00`,
                newStartTime: `2025-0${i + 1}-15T10:00`,
                newEndTime: `2025-0${i + 1}-15T11:00`
            }));
        }
        return data;
    }
}

// Register in the global registry
// This would normally be done after creating the API client and DB verifier
// registerFixture("schedule", new ScheduleFixtureFactory(apiClient, dbVerifier));
/* c8 ignore stop */
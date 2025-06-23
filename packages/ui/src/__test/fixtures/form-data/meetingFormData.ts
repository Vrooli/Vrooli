/**
 * Meeting Form Data Fixtures
 * 
 * Provides comprehensive form data fixtures for meeting creation, scheduling,
 * and management with React Hook Form integration.
 */

import { useForm, type UseFormReturn, type FieldErrors } from "react-hook-form";
import { yupResolver } from "@hookform/resolvers/yup";
import * as yup from "yup";
import { act, waitFor } from "@testing-library/react";
import type { 
    MeetingCreateInput,
    MeetingUpdateInput,
    Meeting,
    MeetingInviteCreateInput,
} from "@vrooli/shared";
import { 
    meetingValidation,
    meetingInviteValidation,
} from "@vrooli/shared";

/**
 * UI-specific form data for meeting creation
 */
export interface MeetingFormData {
    // Basic info
    name: string;
    description?: string;
    
    // Scheduling
    startTime: Date;
    endTime: Date;
    timezone?: string;
    
    // Location/Link
    location?: string;
    meetingLink?: string;
    
    // Recurrence (UI-specific)
    isRecurring?: boolean;
    recurrencePattern?: {
        type: "daily" | "weekly" | "monthly";
        interval: number;
        endDate?: Date;
        maxOccurrences?: number;
    };
    
    // Participants (UI-specific)
    inviteParticipants?: Array<{
        userId?: string;
        email?: string;
        handle?: string;
        role: "host" | "participant" | "observer";
        isRequired?: boolean;
    }>;
    
    // Settings
    requiresApproval?: boolean;
    allowsRecording?: boolean;
    isPrivate?: boolean;
    maxParticipants?: number;
    
    // Agenda items (UI-specific)
    agendaItems?: Array<{
        title: string;
        description?: string;
        duration?: number; // minutes
        presenter?: string;
    }>;
    
    // Notifications
    reminderSettings?: {
        enabled: boolean;
        reminderTimes: number[]; // minutes before meeting
    };
}

/**
 * Extended form state with meeting-specific properties
 */
export interface MeetingFormState {
    values: MeetingFormData;
    errors: Record<string, string>;
    touched: Record<string, boolean>;
    isDirty: boolean;
    isSubmitting: boolean;
    isValid: boolean;
    isValidating?: boolean;
    
    // Meeting-specific state
    meetingState?: {
        currentMeeting?: Partial<Meeting>;
        isCheckingAvailability?: boolean;
        conflictingMeetings?: Array<{
            id: string;
            name: string;
            startTime: Date;
            endTime: Date;
        }>;
        suggestedTimes?: Array<{
            startTime: Date;
            endTime: Date;
            score: number; // availability score
        }>;
    };
    
    // Participant state
    participantState?: {
        totalInvited?: number;
        responded?: number;
        accepted?: number;
        declined?: number;
        pending?: number;
    };
    
    // Calendar integration
    calendarState?: {
        isConnected?: boolean;
        selectedCalendar?: string;
        availableCalendars?: Array<{
            id: string;
            name: string;
            provider: string;
        }>;
    };
}

/**
 * Meeting form data factory with React Hook Form integration
 */
export class MeetingFormDataFactory {
    /**
     * Create validation schema for meetings
     */
    private createMeetingSchema(): yup.ObjectSchema<MeetingFormData> {
        return yup.object({
            name: yup
                .string()
                .required("Meeting name is required")
                .min(1, "Meeting name cannot be empty")
                .max(255, "Meeting name is too long"),
                
            description: yup
                .string()
                .max(2000, "Description is too long")
                .optional(),
                
            startTime: yup
                .date()
                .required("Start time is required")
                .min(new Date(), "Start time cannot be in the past"),
                
            endTime: yup
                .date()
                .required("End time is required")
                .min(yup.ref("startTime"), "End time must be after start time"),
                
            timezone: yup
                .string()
                .optional(),
                
            location: yup
                .string()
                .max(500, "Location is too long")
                .optional(),
                
            meetingLink: yup
                .string()
                .url("Invalid meeting link URL")
                .optional(),
                
            isRecurring: yup
                .boolean()
                .optional(),
                
            recurrencePattern: yup.object({
                type: yup.string().oneOf(["daily", "weekly", "monthly"]).required(),
                interval: yup.number().min(1, "Interval must be at least 1").required(),
                endDate: yup.date().optional(),
                maxOccurrences: yup.number().min(1, "Must have at least 1 occurrence").optional(),
            }).when("isRecurring", {
                is: true,
                then: (schema) => schema.required("Recurrence pattern is required for recurring meetings"),
                otherwise: (schema) => schema.optional(),
            }),
            
            inviteParticipants: yup.array(
                yup.object({
                    userId: yup.string().optional(),
                    email: yup.string().email("Invalid email").optional(),
                    handle: yup.string().optional(),
                    role: yup.string().oneOf(["host", "participant", "observer"]).required(),
                    isRequired: yup.boolean().optional(),
                }).test("participant-identification", "Participant identification is required", function(value) {
                    return !!(value?.userId || value?.email || value?.handle);
                }),
            ).optional(),
            
            requiresApproval: yup.boolean().optional(),
            allowsRecording: yup.boolean().optional(),
            isPrivate: yup.boolean().optional(),
            
            maxParticipants: yup
                .number()
                .min(1, "Must allow at least 1 participant")
                .max(1000, "Maximum 1000 participants allowed")
                .optional(),
                
            agendaItems: yup.array(
                yup.object({
                    title: yup.string().required("Agenda item title is required"),
                    description: yup.string().max(1000, "Description too long").optional(),
                    duration: yup.number().min(1, "Duration must be at least 1 minute").optional(),
                    presenter: yup.string().optional(),
                }),
            ).optional(),
            
            reminderSettings: yup.object({
                enabled: yup.boolean().required(),
                reminderTimes: yup.array(yup.number().min(0))
                    .when("enabled", {
                        is: true,
                        then: (schema) => schema.min(1, "At least one reminder time is required"),
                        otherwise: (schema) => schema.optional(),
                    }),
            }).optional(),
        }).defined();
    }
    
    /**
     * Generate unique ID for testing
     */
    private generateId(): string {
        return `meeting_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create meeting form data for different scenarios
     */
    createFormData(
        scenario: "empty" | "minimal" | "complete" | "invalid" | "recurring" | "withAgenda" | 
                 "privateTeamMeeting" | "largeMeeting" | "quickStandup" | "interview",
    ): MeetingFormData {
        const now = new Date();
        const tomorrow = new Date(now.getTime() + 24 * 60 * 60 * 1000);
        const nextWeek = new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000);
        
        switch (scenario) {
            case "empty":
                return {
                    name: "",
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 60 * 60 * 1000), // 1 hour later
                };
                
            case "minimal":
                return {
                    name: "Team Sync",
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 30 * 60 * 1000), // 30 minutes
                };
                
            case "complete":
                return {
                    name: "Project Planning Session",
                    description: "Quarterly planning session to discuss roadmap and priorities",
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 2 * 60 * 60 * 1000), // 2 hours
                    timezone: "America/New_York",
                    location: "Conference Room A",
                    meetingLink: "https://zoom.us/j/123456789",
                    requiresApproval: false,
                    allowsRecording: true,
                    isPrivate: false,
                    maxParticipants: 20,
                    inviteParticipants: [
                        {
                            handle: "alice",
                            role: "host",
                            isRequired: true,
                        },
                        {
                            email: "bob@example.com",
                            role: "participant",
                            isRequired: true,
                        },
                        {
                            handle: "charlie",
                            role: "observer",
                            isRequired: false,
                        },
                    ],
                    agendaItems: [
                        {
                            title: "Q1 Review",
                            description: "Review of Q1 achievements and metrics",
                            duration: 30,
                            presenter: "alice",
                        },
                        {
                            title: "Q2 Planning",
                            description: "Discussion of Q2 goals and priorities",
                            duration: 60,
                            presenter: "bob",
                        },
                        {
                            title: "Resource Allocation",
                            description: "Planning team assignments and resources",
                            duration: 30,
                            presenter: "charlie",
                        },
                    ],
                    reminderSettings: {
                        enabled: true,
                        reminderTimes: [1440, 60, 15], // 1 day, 1 hour, 15 minutes before
                    },
                };
                
            case "invalid":
                return {
                    name: "", // Empty name
                    startTime: new Date(now.getTime() - 60 * 60 * 1000), // Past time
                    endTime: tomorrow, // End before start
                    meetingLink: "not-a-url", // Invalid URL
                    maxParticipants: 0, // Invalid count
                    inviteParticipants: [
                        {
                            // Missing identification
                            role: "participant",
                        } as any,
                    ],
                    agendaItems: [
                        {
                            title: "", // Empty title
                            duration: -10, // Invalid duration
                        },
                    ],
                };
                
            case "recurring":
                return {
                    name: "Weekly Team Standup",
                    description: "Regular weekly standup to sync on progress",
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 30 * 60 * 1000),
                    isRecurring: true,
                    recurrencePattern: {
                        type: "weekly",
                        interval: 1,
                        endDate: new Date(now.getTime() + 90 * 24 * 60 * 60 * 1000), // 3 months
                    },
                    meetingLink: "https://meet.google.com/abc-defg-hij",
                    inviteParticipants: [
                        { handle: "team-dev", role: "participant", isRequired: true },
                        { handle: "manager", role: "host", isRequired: true },
                    ],
                    reminderSettings: {
                        enabled: true,
                        reminderTimes: [15], // 15 minutes before
                    },
                };
                
            case "withAgenda":
                return {
                    name: "Architecture Review",
                    description: "Technical architecture review for the new microservice",
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 90 * 60 * 1000), // 1.5 hours
                    agendaItems: [
                        {
                            title: "Current Architecture Overview",
                            description: "High-level overview of existing system",
                            duration: 20,
                            presenter: "tech-lead",
                        },
                        {
                            title: "Proposed Changes",
                            description: "Detailed walkthrough of proposed architecture",
                            duration: 40,
                            presenter: "architect",
                        },
                        {
                            title: "Discussion & Q&A",
                            description: "Open discussion and questions",
                            duration: 20,
                        },
                        {
                            title: "Next Steps",
                            description: "Define action items and timeline",
                            duration: 10,
                            presenter: "project-manager",
                        },
                    ],
                    inviteParticipants: [
                        { handle: "tech-lead", role: "host", isRequired: true },
                        { handle: "architect", role: "participant", isRequired: true },
                        { handle: "dev-team", role: "participant", isRequired: false },
                    ],
                };
                
            case "privateTeamMeeting":
                return {
                    name: "Confidential Strategy Discussion",
                    description: "Private discussion about sensitive strategic matters",
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 60 * 60 * 1000),
                    isPrivate: true,
                    requiresApproval: true,
                    allowsRecording: false,
                    maxParticipants: 5,
                    inviteParticipants: [
                        { handle: "ceo", role: "host", isRequired: true },
                        { handle: "cto", role: "participant", isRequired: true },
                        { handle: "director", role: "participant", isRequired: true },
                    ],
                };
                
            case "largeMeeting":
                return {
                    name: "All-Hands Company Meeting",
                    description: "Monthly company-wide meeting with updates and announcements",
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 60 * 60 * 1000),
                    meetingLink: "https://teams.microsoft.com/l/meetup-join/xyz",
                    allowsRecording: true,
                    maxParticipants: 500,
                    inviteParticipants: [
                        { handle: "all-company", role: "participant", isRequired: false },
                    ],
                    reminderSettings: {
                        enabled: true,
                        reminderTimes: [1440, 60], // 1 day and 1 hour before
                    },
                };
                
            case "quickStandup":
                return {
                    name: "Daily Standup",
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 15 * 60 * 1000), // 15 minutes
                    isRecurring: true,
                    recurrencePattern: {
                        type: "daily",
                        interval: 1,
                        maxOccurrences: 30, // 1 month
                    },
                    reminderSettings: {
                        enabled: true,
                        reminderTimes: [5], // 5 minutes before
                    },
                };
                
            case "interview":
                return {
                    name: "Technical Interview - Senior Developer",
                    description: "Technical interview for senior developer position",
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 60 * 60 * 1000),
                    meetingLink: "https://zoom.us/j/interview123",
                    isPrivate: true,
                    maxParticipants: 5,
                    inviteParticipants: [
                        { email: "candidate@example.com", role: "participant", isRequired: true },
                        { handle: "hiring-manager", role: "host", isRequired: true },
                        { handle: "tech-interviewer", role: "participant", isRequired: true },
                    ],
                    agendaItems: [
                        {
                            title: "Introduction",
                            duration: 5,
                            presenter: "hiring-manager",
                        },
                        {
                            title: "Technical Discussion",
                            duration: 40,
                            presenter: "tech-interviewer",
                        },
                        {
                            title: "Candidate Questions",
                            duration: 10,
                        },
                        {
                            title: "Next Steps",
                            duration: 5,
                            presenter: "hiring-manager",
                        },
                    ],
                };
                
            default:
                throw new Error(`Unknown meeting scenario: ${scenario}`);
        }
    }
    
    /**
     * Create form state for different scenarios
     */
    createFormState(
        scenario: "pristine" | "dirty" | "submitting" | "withErrors" | "valid" | "checkingAvailability" | "withConflicts",
    ): MeetingFormState {
        const baseFormData = this.createFormData("complete");
        
        switch (scenario) {
            case "pristine":
                return {
                    values: this.createFormData("empty"),
                    errors: {},
                    touched: {},
                    isDirty: false,
                    isSubmitting: false,
                    isValid: false,
                };
                
            case "dirty":
                return {
                    values: baseFormData,
                    errors: {},
                    touched: { name: true, startTime: true, endTime: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                };
                
            case "submitting":
                return {
                    values: baseFormData,
                    errors: {},
                    touched: Object.keys(baseFormData).reduce((acc, key) => ({
                        ...acc,
                        [key]: true,
                    }), {}),
                    isDirty: true,
                    isSubmitting: true,
                    isValid: true,
                };
                
            case "withErrors":
                return {
                    values: this.createFormData("invalid"),
                    errors: {
                        name: "Meeting name is required",
                        startTime: "Start time cannot be in the past",
                        endTime: "End time must be after start time",
                        meetingLink: "Invalid meeting link URL",
                        maxParticipants: "Must allow at least 1 participant",
                    },
                    touched: { 
                        name: true, 
                        startTime: true, 
                        endTime: true,
                        meetingLink: true,
                        maxParticipants: true,
                    },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: false,
                };
                
            case "valid":
                return {
                    values: baseFormData,
                    errors: {},
                    touched: Object.keys(baseFormData).reduce((acc, key) => ({
                        ...acc,
                        [key]: true,
                    }), {}),
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                    participantState: {
                        totalInvited: 3,
                        responded: 2,
                        accepted: 2,
                        declined: 0,
                        pending: 1,
                    },
                };
                
            case "checkingAvailability":
                return {
                    values: baseFormData,
                    errors: {},
                    touched: { startTime: true, endTime: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: false,
                    isValidating: true,
                    meetingState: {
                        isCheckingAvailability: true,
                    },
                };
                
            case "withConflicts":
                return {
                    values: baseFormData,
                    errors: {},
                    touched: { startTime: true, endTime: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                    meetingState: {
                        conflictingMeetings: [
                            {
                                id: "meeting1",
                                name: "Existing Meeting",
                                startTime: baseFormData.startTime,
                                endTime: new Date(baseFormData.startTime.getTime() + 30 * 60 * 1000),
                            },
                        ],
                        suggestedTimes: [
                            {
                                startTime: new Date(baseFormData.startTime.getTime() + 2 * 60 * 60 * 1000),
                                endTime: new Date(baseFormData.startTime.getTime() + 4 * 60 * 60 * 1000),
                                score: 0.9,
                            },
                        ],
                    },
                };
                
            default:
                throw new Error(`Unknown form state scenario: ${scenario}`);
        }
    }
    
    /**
     * Create React Hook Form instance
     */
    createFormInstance(
        initialData?: Partial<MeetingFormData>,
    ): UseFormReturn<MeetingFormData> {
        const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000);
        const defaultValues: MeetingFormData = {
            name: "",
            startTime: tomorrow,
            endTime: new Date(tomorrow.getTime() + 60 * 60 * 1000),
            inviteParticipants: [],
            agendaItems: [],
            ...initialData,
        };
        
        return useForm<MeetingFormData>({
            mode: "onChange",
            reValidateMode: "onChange",
            shouldFocusError: true,
            defaultValues,
            resolver: yupResolver(this.createMeetingSchema()),
        });
    }
    
    /**
     * Validate form data using real validation
     */
    async validateFormData(
        formData: MeetingFormData,
    ): Promise<{
        isValid: boolean;
        errors?: Record<string, string>;
        apiInput?: MeetingCreateInput;
    }> {
        try {
            // Validate with form schema
            await this.createMeetingSchema().validate(formData, { abortEarly: false });
            
            // Transform to API input
            const apiInput = this.transformToAPIInput(formData);
            
            // Validate with real API validation
            await meetingValidation.create.validate(apiInput);
            
            return {
                isValid: true,
                apiInput,
            };
        } catch (error: any) {
            const errors: Record<string, string> = {};
            
            if (error.inner) {
                error.inner.forEach((err: any) => {
                    if (err.path) {
                        errors[err.path] = err.message;
                    }
                });
            } else if (error.message) {
                errors.general = error.message;
            }
            
            return {
                isValid: false,
                errors,
            };
        }
    }
    
    /**
     * Transform form data to API input
     */
    private transformToAPIInput(formData: MeetingFormData): MeetingCreateInput {
        const input: MeetingCreateInput = {
            id: this.generateId(),
            openToAnyoneWithInvite: !formData.requiresApproval,
            restrictedToRoles: [],
            meetingUrl: formData.meetingLink,
            translationsCreate: [{
                id: this.generateId(),
                language: "en",
                name: formData.name,
                description: formData.description,
            }],
            schedulesCreate: [{
                id: this.generateId(),
                startTime: formData.startTime.toISOString(),
                endTime: formData.endTime.toISOString(),
                timezone: formData.timezone || "UTC",
            }],
        };
        
        // Add participant invites
        if (formData.inviteParticipants && formData.inviteParticipants.length > 0) {
            input.invitesCreate = formData.inviteParticipants.map(participant => ({
                id: this.generateId(),
                userConnect: participant.userId || participant.handle || participant.email || "",
                willAttend: participant.isRequired,
            }));
        }
        
        return input;
    }
    
    /**
     * Calculate meeting duration in minutes
     */
    calculateDuration(startTime: Date, endTime: Date): number {
        return Math.round((endTime.getTime() - startTime.getTime()) / (1000 * 60));
    }
    
    /**
     * Validate time slot availability (mock implementation)
     */
    async checkTimeSlotAvailability(
        startTime: Date,
        endTime: Date,
        participants: string[],
    ): Promise<{
        isAvailable: boolean;
        conflicts: Array<{ participant: string; conflictingMeeting: string }>;
    }> {
        // Mock implementation - in real app would check calendar APIs
        await new Promise(resolve => setTimeout(resolve, 500));
        
        // Simulate some conflicts
        const conflicts = participants.slice(0, 1).map(participant => ({
            participant,
            conflictingMeeting: "Existing Team Meeting",
        }));
        
        return {
            isAvailable: conflicts.length === 0,
            conflicts,
        };
    }
}

/**
 * Form interaction simulator for meeting forms
 */
export class MeetingFormInteractionSimulator {
    private interactionDelay = 100;
    
    constructor(delay?: number) {
        this.interactionDelay = delay || 100;
    }
    
    /**
     * Simulate meeting scheduling workflow
     */
    async simulateMeetingScheduling(
        formInstance: UseFormReturn<MeetingFormData>,
        formData: MeetingFormData,
    ): Promise<void> {
        // Type meeting name
        await this.simulateTyping(formInstance, "name", formData.name);
        
        // Set date/time
        await this.fillField(formInstance, "startTime", formData.startTime);
        await this.fillField(formInstance, "endTime", formData.endTime);
        
        // Add description if present
        if (formData.description) {
            await this.simulateTyping(formInstance, "description", formData.description);
        }
        
        // Add participants
        if (formData.inviteParticipants) {
            for (let i = 0; i < formData.inviteParticipants.length; i++) {
                await this.addParticipant(formInstance, i, formData.inviteParticipants[i]);
            }
        }
        
        // Add agenda items
        if (formData.agendaItems) {
            for (let i = 0; i < formData.agendaItems.length; i++) {
                await this.addAgendaItem(formInstance, i, formData.agendaItems[i]);
            }
        }
        
        // Set meeting settings
        if (formData.meetingLink) {
            await this.fillField(formInstance, "meetingLink", formData.meetingLink);
        }
        
        if (formData.isRecurring) {
            await this.fillField(formInstance, "isRecurring", true);
            if (formData.recurrencePattern) {
                await this.fillField(formInstance, "recurrencePattern.type", formData.recurrencePattern.type);
                await this.fillField(formInstance, "recurrencePattern.interval", formData.recurrencePattern.interval);
            }
        }
    }
    
    /**
     * Add participant to meeting
     */
    private async addParticipant(
        formInstance: UseFormReturn<MeetingFormData>,
        index: number,
        participant: MeetingFormData["inviteParticipants"][0],
    ): Promise<void> {
        if (participant.userId) {
            await this.fillField(formInstance, `inviteParticipants.${index}.userId` as any, participant.userId);
        } else if (participant.email) {
            await this.fillField(formInstance, `inviteParticipants.${index}.email` as any, participant.email);
        } else if (participant.handle) {
            await this.fillField(formInstance, `inviteParticipants.${index}.handle` as any, participant.handle);
        }
        
        await this.fillField(formInstance, `inviteParticipants.${index}.role` as any, participant.role);
        
        if (participant.isRequired !== undefined) {
            await this.fillField(formInstance, `inviteParticipants.${index}.isRequired` as any, participant.isRequired);
        }
        
        await new Promise(resolve => setTimeout(resolve, 200));
    }
    
    /**
     * Add agenda item to meeting
     */
    private async addAgendaItem(
        formInstance: UseFormReturn<MeetingFormData>,
        index: number,
        item: MeetingFormData["agendaItems"][0],
    ): Promise<void> {
        await this.simulateTyping(formInstance, `agendaItems.${index}.title` as any, item.title);
        
        if (item.description) {
            await this.simulateTyping(formInstance, `agendaItems.${index}.description` as any, item.description);
        }
        
        if (item.duration) {
            await this.fillField(formInstance, `agendaItems.${index}.duration` as any, item.duration);
        }
        
        if (item.presenter) {
            await this.fillField(formInstance, `agendaItems.${index}.presenter` as any, item.presenter);
        }
        
        await new Promise(resolve => setTimeout(resolve, 200));
    }
    
    /**
     * Simulate typing
     */
    private async simulateTyping(
        formInstance: UseFormReturn<any>,
        fieldName: string,
        text: string,
    ): Promise<void> {
        for (let i = 1; i <= text.length; i++) {
            await this.fillField(formInstance, fieldName, text.substring(0, i));
            await new Promise(resolve => setTimeout(resolve, 50));
        }
    }
    
    /**
     * Fill field helper
     */
    private async fillField(
        formInstance: UseFormReturn<any>,
        fieldName: string,
        value: any,
    ): Promise<void> {
        act(() => {
            formInstance.setValue(fieldName, value, {
                shouldDirty: true,
                shouldTouch: true,
                shouldValidate: true,
            });
        });
        
        await waitFor(() => {
            expect(formInstance.formState.isValidating).toBe(false);
        });
    }
}

// Export factory instances
export const meetingFormFactory = new MeetingFormDataFactory();
export const meetingFormSimulator = new MeetingFormInteractionSimulator();

// Export pre-configured scenarios
export const meetingFormScenarios = {
    // Basic scenarios
    emptyMeeting: () => meetingFormFactory.createFormState("pristine"),
    validMeeting: () => meetingFormFactory.createFormState("valid"),
    meetingWithErrors: () => meetingFormFactory.createFormState("withErrors"),
    submittingMeeting: () => meetingFormFactory.createFormState("submitting"),
    
    // Meeting types
    minimalMeeting: () => meetingFormFactory.createFormData("minimal"),
    completeMeeting: () => meetingFormFactory.createFormData("complete"),
    recurringMeeting: () => meetingFormFactory.createFormData("recurring"),
    meetingWithAgenda: () => meetingFormFactory.createFormData("withAgenda"),
    privateTeamMeeting: () => meetingFormFactory.createFormData("privateTeamMeeting"),
    largeMeeting: () => meetingFormFactory.createFormData("largeMeeting"),
    quickStandup: () => meetingFormFactory.createFormData("quickStandup"),
    interview: () => meetingFormFactory.createFormData("interview"),
    
    // Form states
    checkingAvailability: () => meetingFormFactory.createFormState("checkingAvailability"),
    meetingWithConflicts: () => meetingFormFactory.createFormState("withConflicts"),
    
    // Interactive workflows
    async quickMeetingSetup(formInstance: UseFormReturn<MeetingFormData>) {
        const simulator = new MeetingFormInteractionSimulator();
        const formData = meetingFormFactory.createFormData("quickStandup");
        await simulator.simulateMeetingScheduling(formInstance, formData);
        return formData;
    },
    
    async completeMeetingWorkflow(formInstance: UseFormReturn<MeetingFormData>) {
        const simulator = new MeetingFormInteractionSimulator();
        const formData = meetingFormFactory.createFormData("complete");
        await simulator.simulateMeetingScheduling(formInstance, formData);
        return formData;
    },
    
    // Utility functions
    calculateMeetingDuration(startTime: Date, endTime: Date) {
        return meetingFormFactory.calculateDuration(startTime, endTime);
    },
    
    async checkAvailability(startTime: Date, endTime: Date, participants: string[]) {
        return meetingFormFactory.checkTimeSlotAvailability(startTime, endTime, participants);
    },
};

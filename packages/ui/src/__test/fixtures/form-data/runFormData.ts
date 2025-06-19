/**
 * Form data fixtures for run-related forms
 * These represent data as it appears in form state before submission
 */

import { RunStatus } from "@vrooli/shared";

/**
 * Run creation form data
 */
export const minimalRunCreateFormInput = {
    name: "Test Run",
    status: RunStatus.InProgress,
    isPrivate: false,
    resourceVersionId: "resource_version_123456789",
    runNow: true,
};

export const completeRunCreateFormInput = {
    name: "AI Processing Pipeline Run",
    status: RunStatus.Scheduled,
    isPrivate: false,
    resourceVersionId: "resource_version_123456789",
    teamId: "team_123456789",
    runNow: false,
    scheduledFor: "2024-12-25T10:00:00Z",
    data: JSON.stringify({
        inputs: {
            sourceData: "user_input_data.json",
            parameters: {
                threshold: 0.8,
                iterations: 100,
            },
        },
        config: {
            enableLogging: true,
            maxRetries: 3,
        },
    }),
    steps: [
        {
            name: "Data Validation",
            nodeId: "validate_node",
            order: 0,
            inputConnections: [],
        },
        {
            name: "Data Processing",
            nodeId: "process_node", 
            order: 1,
            inputConnections: ["validate_node"],
        },
        {
            name: "Result Analysis",
            nodeId: "analyze_node",
            order: 2,
            inputConnections: ["process_node"],
        },
    ],
    schedule: {
        startTime: "2024-12-25T10:00:00Z",
        endTime: "2024-12-25T12:00:00Z",
        timezone: "UTC",
        recurrence: {
            recurrenceType: "Weekly",
            interval: 1,
            dayOfWeek: 2, // Tuesday
        },
    },
};

/**
 * Run update form data
 */
export const minimalRunUpdateFormInput = {
    name: "Updated Test Run",
    status: RunStatus.Paused,
};

export const completeRunUpdateFormInput = {
    name: "Enhanced AI Processing Pipeline",
    status: RunStatus.InProgress,
    isPrivate: true,
    data: JSON.stringify({
        inputs: {
            sourceData: "enhanced_input_data.json",
            parameters: {
                threshold: 0.9,
                iterations: 150,
                useAdvancedAlgorithm: true,
            },
        },
        config: {
            enableLogging: true,
            enableDebugMode: true,
            maxRetries: 5,
            timeout: 3600,
        },
    }),
    completedComplexity: 8,
    contextSwitches: 3,
    timeElapsed: 1800,
    steps: [
        {
            name: "Enhanced Data Validation",
            nodeId: "validate_node_v2",
            order: 0,
            status: "Completed",
            complexity: 2,
        },
        {
            name: "Advanced Data Processing",
            nodeId: "process_node_v2",
            order: 1,
            status: "InProgress",
            complexity: 5,
        },
        {
            name: "Deep Result Analysis",
            nodeId: "analyze_node_v2",
            order: 2,
            status: "InProgress",
            complexity: 3,
        },
    ],
    ioData: [
        {
            nodeInputName: "input_data",
            nodeName: "validate_node_v2",
            data: JSON.stringify({ sourceFile: "data.csv", recordCount: 10000 }),
        },
        {
            nodeInputName: "processed_data",
            nodeName: "process_node_v2",
            data: JSON.stringify({ processedRecords: 8500, errors: 0 }),
        },
    ],
};

/**
 * Run execution form data variants
 */
export const runExecutionVariants = {
    quickRun: {
        name: "Quick Test Run",
        status: RunStatus.InProgress,
        isPrivate: false,
        resourceVersionId: "resource_version_123456789",
        runNow: true,
        data: JSON.stringify({ quickTest: true }),
    },
    batchRun: {
        name: "Batch Processing Run",
        status: RunStatus.Scheduled,
        isPrivate: false,
        resourceVersionId: "resource_version_batch_123",
        runNow: false,
        scheduledFor: "2024-12-25T02:00:00Z",
        data: JSON.stringify({
            batchSize: 1000,
            processingMode: "parallel",
            priority: "low",
        }),
    },
    debugRun: {
        name: "Debug Analysis Run",
        status: RunStatus.InProgress,
        isPrivate: true,
        resourceVersionId: "resource_version_debug_456",
        runNow: true,
        data: JSON.stringify({
            debugMode: true,
            verboseLogging: true,
            breakpoints: ["step_2", "step_5"],
        }),
    },
    teamRun: {
        name: "Team Collaboration Run",
        status: RunStatus.InProgress,
        isPrivate: false,
        resourceVersionId: "resource_version_team_789",
        teamId: "team_collaboration_123",
        runNow: true,
        data: JSON.stringify({
            collaborative: true,
            teamMembers: ["user_1", "user_2", "user_3"],
            permissions: { canPause: true, canStop: false },
        }),
    },
};

/**
 * Run status variants
 */
export const runStatusVariants = {
    scheduled: {
        name: "Scheduled Processing Run",
        status: RunStatus.Scheduled,
        isPrivate: false,
        resourceVersionId: "resource_version_123456789",
        runNow: false,
        scheduledFor: "2024-12-25T15:30:00Z",
    },
    inProgress: {
        name: "Active Processing Run",
        status: RunStatus.InProgress,
        isPrivate: false,
        resourceVersionId: "resource_version_123456789",
        runNow: true,
        startedAt: new Date().toISOString(),
    },
    paused: {
        name: "Paused Analysis Run",
        status: RunStatus.Paused,
        isPrivate: false,
        resourceVersionId: "resource_version_123456789",
        runNow: false,
        timeElapsed: 900,
    },
    completed: {
        name: "Completed Data Run",
        status: RunStatus.Completed,
        isPrivate: false,
        resourceVersionId: "resource_version_123456789",
        runNow: false,
        completedComplexity: 10,
        timeElapsed: 3600,
    },
    failed: {
        name: "Failed Processing Run",
        status: RunStatus.Failed,
        isPrivate: false,
        resourceVersionId: "resource_version_123456789",
        runNow: false,
        data: JSON.stringify({ 
            error: "Processing timeout",
            lastStep: "data_validation",
            retryCount: 3,
        }),
    },
    cancelled: {
        name: "Cancelled Batch Run",
        status: RunStatus.Cancelled,
        isPrivate: false,
        resourceVersionId: "resource_version_123456789",
        runNow: false,
        data: JSON.stringify({ 
            cancelledBy: "user_123",
            reason: "Resource constraints",
            partialResults: true,
        }),
    },
};

/**
 * Run scheduling form data
 */
export const runScheduleFormInput = {
    startTime: "2024-12-25T09:00:00Z",
    endTime: "2024-12-25T17:00:00Z",
    timezone: "America/New_York",
    recurrence: {
        recurrenceType: "Daily",
        interval: 1,
        dayOfMonth: null,
        dayOfWeek: null,
        month: null,
    },
    exceptions: [
        {
            originalStartTime: "2024-12-25T09:00:00Z",
            newStartTime: "2024-12-25T10:00:00Z",
            newEndTime: "2024-12-25T18:00:00Z",
        },
    ],
};

export const weeklyRecurrenceFormInput = {
    startTime: "2024-12-25T14:00:00Z",
    endTime: "2024-12-25T16:00:00Z",
    timezone: "UTC",
    recurrence: {
        recurrenceType: "Weekly",
        interval: 2,
        dayOfWeek: 5, // Friday
    },
};

export const monthlyRecurrenceFormInput = {
    startTime: "2024-12-01T12:00:00Z",
    endTime: "2024-12-01T13:00:00Z",
    timezone: "Europe/London",
    recurrence: {
        recurrenceType: "Monthly",
        interval: 1,
        dayOfMonth: 15, // 15th of each month
    },
};

/**
 * Run step management form data
 */
export const addRunStepFormInput = {
    name: "New Processing Step",
    nodeId: "new_step_node",
    order: 3,
    complexity: 4,
    inputConnections: ["previous_step_node"],
    configuration: {
        timeout: 300,
        retryPolicy: {
            maxRetries: 2,
            backoffStrategy: "exponential",
        },
    },
};

export const updateRunStepFormInput = {
    name: "Updated Analysis Step",  
    complexity: 6,
    status: "InProgress",
    configuration: {
        timeout: 600,
        enableCaching: true,
        cacheExpiry: 3600,
    },
};

/**
 * Run IO management form data
 */
export const addRunIOFormInput = {
    nodeInputName: "data_input",
    nodeName: "processing_node",
    data: JSON.stringify({
        inputFile: "dataset.json",
        format: "json",
        size: 1024000,
        checksum: "abc123def456",
    }),
};

export const updateRunIOFormInput = {
    data: JSON.stringify({
        outputFile: "results.json",
        format: "json",
        recordsProcessed: 50000,
        processingTime: 1200,
        success: true,
    }),
};

/**
 * Form validation states
 */
export const runFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            name: "", // Required but empty
            status: "", // Required but empty
            resourceVersionId: "", // Required but empty
            scheduledFor: "invalid-date", // Invalid date format
        },
        errors: {
            name: "Run name is required",
            status: "Run status is required",
            resourceVersionId: "Resource version is required",
            scheduledFor: "Please enter a valid date and time",
        },
        touched: {
            name: true,
            status: true,
            resourceVersionId: true,
            scheduledFor: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalRunCreateFormInput,
        errors: {},
        touched: {
            name: true,
            status: true,
            isPrivate: true,
            resourceVersionId: true,
            runNow: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: completeRunCreateFormInput,
        errors: {},
        touched: {},
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create run form initial values
 */
export const createRunFormInitialValues = (runData?: Partial<any>) => ({
    name: runData?.name || "",
    status: runData?.status || RunStatus.InProgress,
    isPrivate: runData?.isPrivate || false,
    resourceVersionId: runData?.resourceVersionId || "",
    teamId: runData?.teamId || "",
    runNow: runData?.runNow ?? true,
    scheduledFor: runData?.scheduledFor || "",
    data: runData?.data || "",
    completedComplexity: runData?.completedComplexity || 0,
    contextSwitches: runData?.contextSwitches || 0,
    timeElapsed: runData?.timeElapsed || 0,
    ...runData,
});

/**
 * Helper function to validate run name
 */
export const validateRunName = (name: string): string | null => {
    if (!name) return "Run name is required";
    if (name.length < 3) return "Run name must be at least 3 characters";
    if (name.length > 128) return "Run name must be less than 128 characters";
    return null;
};

/**
 * Helper function to validate scheduled time
 */
export const validateScheduledTime = (scheduledFor: string, runNow: boolean): string | null => {
    if (runNow) return null; // No validation needed for immediate runs
    if (!scheduledFor) return "Scheduled time is required when not running immediately";
    
    const scheduledDate = new Date(scheduledFor);
    if (isNaN(scheduledDate.getTime())) {
        return "Please enter a valid date and time";
    }
    
    if (scheduledDate <= new Date()) {
        return "Scheduled time must be in the future";
    }
    
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformRunFormToApiInput = (formData: any) => ({
    ...formData,
    // Convert resourceVersionId to connect format
    resourceVersionConnect: formData.resourceVersionId,
    // Convert teamId to connect format if provided
    ...(formData.teamId && { teamConnect: formData.teamId }),
    // Convert steps array to create format
    ...(formData.steps && {
        stepsCreate: formData.steps.map((step: any, index: number) => ({
            id: `step_${Date.now()}_${index}`,
            name: step.name,
            nodeId: step.nodeId,
            order: step.order,
            complexity: step.complexity || 1,
            status: step.status || "InProgress",
            runConnect: formData.id || `run_${Date.now()}`,
        })),
    }),
    // Convert IO data to create format
    ...(formData.ioData && {
        ioCreate: formData.ioData.map((io: any, index: number) => ({
            id: `io_${Date.now()}_${index}`,
            nodeInputName: io.nodeInputName,
            nodeName: io.nodeName,
            data: io.data,
            runConnect: formData.id || `run_${Date.now()}`,
        })),
    }),
    // Convert schedule data to create format
    ...(formData.schedule && {
        scheduleCreate: {
            id: `schedule_${Date.now()}`,
            startTime: formData.schedule.startTime,
            endTime: formData.schedule.endTime,
            timezone: formData.schedule.timezone,
            ...(formData.schedule.recurrence && {
                recurrencesCreate: [{
                    id: `recurrence_${Date.now()}`,
                    ...formData.schedule.recurrence,
                }],
            }),
        },
    }),
    // Remove form-specific fields
    resourceVersionId: undefined,
    teamId: undefined,
    runNow: undefined,
    scheduledFor: undefined,
    steps: undefined,
    ioData: undefined,
    schedule: undefined,
});

/**
 * Mock resource version options for form selects
 */
export const mockResourceVersionOptions = [
    { 
        value: "resource_version_123456789", 
        label: "Data Processing Pipeline v1.0.0",
        description: "Basic data processing and analysis pipeline",
        resourceType: "Routine",
    },
    { 
        value: "resource_version_987654321", 
        label: "ML Training Workflow v2.1.0",
        description: "Machine learning model training and evaluation",
        resourceType: "Routine",
    },
    { 
        value: "resource_version_456789123", 
        label: "Report Generator v1.5.0",
        description: "Automated report generation system",
        resourceType: "SmartContract",
    },
    { 
        value: "resource_version_789123456", 
        label: "Data Validation Suite v3.0.0",
        description: "Comprehensive data validation and cleaning",
        resourceType: "DataConverter",
    },
];

/**
 * Mock team options for form selects
 */
export const mockTeamOptions = [
    { value: "team_123456789", label: "Data Science Team", memberCount: 8 },
    { value: "team_987654321", label: "AI Research Group", memberCount: 12 },
    { value: "team_456789123", label: "Analytics Department", memberCount: 15 },
    { value: "team_789123456", label: "Development Team", memberCount: 6 },
];

/**
 * Mock status options for form selects
 */
export const mockStatusOptions = [
    { value: RunStatus.Scheduled, label: "Scheduled", description: "Run is scheduled for later execution" },
    { value: RunStatus.InProgress, label: "In Progress", description: "Run is currently executing" },
    { value: RunStatus.Paused, label: "Paused", description: "Run is temporarily paused" },
    { value: RunStatus.Completed, label: "Completed", description: "Run finished successfully" },
    { value: RunStatus.Failed, label: "Failed", description: "Run encountered an error" },
    { value: RunStatus.Cancelled, label: "Cancelled", description: "Run was cancelled by user" },
];

/**
 * Mock timezone options
 */
export const mockTimezoneOptions = [
    { value: "UTC", label: "UTC (Coordinated Universal Time)" },
    { value: "America/New_York", label: "Eastern Time" },
    { value: "America/Chicago", label: "Central Time" },
    { value: "America/Denver", label: "Mountain Time" },
    { value: "America/Los_Angeles", label: "Pacific Time" },
    { value: "Europe/London", label: "GMT (Greenwich Mean Time)" },
    { value: "Europe/Paris", label: "CET (Central European Time)" },
    { value: "Asia/Tokyo", label: "JST (Japan Standard Time)" },
    { value: "Asia/Shanghai", label: "CST (China Standard Time)" },
    { value: "Australia/Sydney", label: "AEDT (Australian Eastern Daylight Time)" },
];
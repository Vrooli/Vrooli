import { 
    type Run, 
    type RunIO, 
    type RunStep, 
    type Schedule, 
    type ScheduleRecurrence, 
    RunStatus, 
    RunStepStatus, 
    ScheduleRecurrenceType 
} from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse, botUserResponse } from "./userResponses.js";
import { minimalTeamResponse } from "./teamResponses.js";
import { minimalRoutineResponse, completeRoutineResponse } from "./routineResponses.js";

/**
 * API response fixtures for runs
 * These represent what components receive from API calls
 */

/**
 * Mock run IO data
 */
const inputIO: RunIO = {
    __typename: "RunIO",
    id: "runio_input_123456789",
    data: JSON.stringify({
        userInput: "Generate a report on quarterly sales data",
        additionalParams: {
            format: "PDF",
            includeCharts: true
        }
    }),
    nodeInputName: "userInput",
    nodeName: "inputNode",
    run: {} as Run, // Will be filled by parent run
};

const outputIO: RunIO = {
    __typename: "RunIO",
    id: "runio_output_123456789",
    data: JSON.stringify({
        generatedReport: "https://example.com/reports/q4-2024.pdf",
        reportMetadata: {
            pages: 25,
            generatedAt: "2024-01-15T14:45:00Z",
            fileSize: "2.4MB"
        }
    }),
    nodeInputName: "result",
    nodeName: "outputNode",
    run: {} as Run, // Will be filled by parent run
};

/**
 * Mock schedule data
 */
const basicSchedule: Schedule = {
    __typename: "Schedule",
    id: "schedule_123456789012345",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-15T08:00:00Z",
    startTime: "2024-01-15T09:00:00Z",
    endTime: "2024-01-15T17:00:00Z",
    timezone: "America/New_York",
    publicId: "sched_pub_123456789",
    exceptions: [],
    recurrences: [],
    meetings: [],
    runs: [],
    user: {
        __typename: "User",
        ...minimalUserResponse,
    },
};

/**
 * Mock run step data
 */
const completedStep: RunStep = {
    __typename: "RunStep",
    id: "runstep_completed_123456",
    completedAt: "2024-01-15T10:30:00Z",
    complexity: 2,
    contextSwitches: 1,
    name: "Data Processing",
    nodeId: "node_processing_123",
    order: 0,
    startedAt: "2024-01-15T10:15:00Z",
    status: RunStepStatus.Completed,
    resourceInId: "routine_123456789012345",
    resourceVersion: null,
    timeElapsed: 900, // 15 minutes in seconds
};

const inProgressStep: RunStep = {
    __typename: "RunStep",
    id: "runstep_inprogress_123456",
    completedAt: null,
    complexity: 3,
    contextSwitches: 0,
    name: "Report Generation",
    nodeId: "node_generation_123",
    order: 1,
    startedAt: "2024-01-15T10:30:00Z",
    status: RunStepStatus.InProgress,
    resourceInId: "routine_123456789012345",
    resourceVersion: null,
    timeElapsed: 300, // 5 minutes elapsed so far
};

const pendingStep: RunStep = {
    __typename: "RunStep",
    id: "runstep_pending_123456",
    completedAt: null,
    complexity: 1,
    contextSwitches: 0,
    name: "Email Delivery",
    nodeId: "node_email_123",
    order: 2,
    startedAt: null,
    status: RunStepStatus.InProgress, // Using InProgress as there's no Pending status
    resourceInId: "routine_123456789012345",
    resourceVersion: null,
    timeElapsed: null,
};

/**
 * Minimal run API response
 */
export const minimalRunResponse: Run = {
    __typename: "Run",
    id: "run_123456789012345678",
    completedAt: null,
    completedComplexity: 0,
    contextSwitches: 0,
    data: null,
    io: [],
    ioCount: 0,
    isPrivate: false,
    lastStep: null,
    name: "Simple Task Execution",
    resourceVersion: minimalRoutineResponse.versions[0],
    schedule: null,
    startedAt: "2024-01-15T09:00:00Z",
    status: RunStatus.InProgress,
    steps: [],
    stepsCount: 0,
    team: null,
    timeElapsed: 300, // 5 minutes
    user: {
        __typename: "User",
        ...minimalUserResponse,
    },
    wasRunAutomatically: false,
    you: {
        __typename: "RunYou",
        canDelete: false,
        canRead: true,
        canUpdate: false,
    },
};

/**
 * Complete run API response with all fields
 */
export const completeRunResponse: Run = {
    __typename: "Run",
    id: "run_987654321098765432",
    completedAt: null, // Still in progress
    completedComplexity: 2, // One step completed
    contextSwitches: 1,
    data: JSON.stringify({
        runConfig: {
            priority: "high",
            notifications: true,
            retryPolicy: "exponential"
        },
        metadata: {
            source: "web_ui",
            userAgent: "Mozilla/5.0 (Chrome/120.0.0.0)"
        }
    }),
    io: [inputIO, outputIO],
    ioCount: 2,
    isPrivate: false,
    lastStep: [1], // Currently on step 1 (0-indexed)
    name: "Quarterly Report Generation",
    resourceVersion: completeRoutineResponse.versions[0],
    schedule: basicSchedule,
    startedAt: "2024-01-15T10:15:00Z",
    status: RunStatus.InProgress,
    steps: [completedStep, inProgressStep, pendingStep],
    stepsCount: 3,
    team: {
        __typename: "Team",
        ...minimalTeamResponse,
    },
    timeElapsed: 1200, // 20 minutes total
    user: null, // Team-owned run
    wasRunAutomatically: false,
    you: {
        __typename: "RunYou",
        canDelete: true,
        canRead: true,
        canUpdate: true,
    },
};

/**
 * Completed run response
 */
export const completedRunResponse: Run = {
    __typename: "Run",
    id: "run_completed_123456789",
    completedAt: "2024-01-15T11:45:00Z",
    completedComplexity: 6, // All steps completed
    contextSwitches: 2,
    data: JSON.stringify({
        runConfig: {
            priority: "normal",
            notifications: true
        },
        results: {
            success: true,
            outputFiles: ["report.pdf", "data.csv"],
            executionTime: 2700
        }
    }),
    io: [
        {
            ...inputIO,
            id: "runio_input_completed_123",
            data: JSON.stringify({ prompt: "Generate monthly summary" })
        },
        {
            ...outputIO,
            id: "runio_output_completed_123",
            data: JSON.stringify({ 
                result: "Monthly summary generated successfully",
                fileUrl: "https://example.com/monthly-summary.pdf"
            })
        }
    ],
    ioCount: 2,
    isPrivate: false,
    lastStep: [2], // Completed step 2 (final step)
    name: "Monthly Summary Generation",
    resourceVersion: minimalRoutineResponse.versions[0],
    schedule: null,
    startedAt: "2024-01-15T11:00:00Z",
    status: RunStatus.Completed,
    steps: [
        {
            ...completedStep,
            id: "runstep_completed_1_123",
            name: "Data Collection",
            order: 0,
            status: RunStepStatus.Completed,
            completedAt: "2024-01-15T11:15:00Z",
            timeElapsed: 900
        },
        {
            ...completedStep,
            id: "runstep_completed_2_123",
            name: "Report Generation",
            order: 1,
            status: RunStepStatus.Completed,
            completedAt: "2024-01-15T11:30:00Z",
            timeElapsed: 900
        },
        {
            ...completedStep,
            id: "runstep_completed_3_123",
            name: "File Export",
            order: 2,
            status: RunStepStatus.Completed,
            completedAt: "2024-01-15T11:45:00Z",
            timeElapsed: 900
        }
    ],
    stepsCount: 3,
    team: null,
    timeElapsed: 2700, // 45 minutes total
    user: {
        __typename: "User",
        ...completeUserResponse,
    },
    wasRunAutomatically: false,
    you: {
        __typename: "RunYou",
        canDelete: true,
        canRead: true,
        canUpdate: false, // Can't update completed runs
    },
};

/**
 * Failed run response
 */
export const failedRunResponse: Run = {
    __typename: "Run",
    id: "run_failed_123456789012",
    completedAt: "2024-01-15T10:22:00Z", // Failed timestamp
    completedComplexity: 1, // Only first step completed
    contextSwitches: 3, // High context switches might indicate issues
    data: JSON.stringify({
        runConfig: {
            priority: "high",
            retryAttempts: 3
        },
        errorInfo: {
            lastError: "API rate limit exceeded",
            failedStep: "data_fetch",
            retryCount: 3
        }
    }),
    io: [
        {
            ...inputIO,
            id: "runio_failed_input_123",
            data: JSON.stringify({ 
                query: "SELECT * FROM large_dataset WHERE date > '2024-01-01'"
            })
        }
    ],
    ioCount: 1,
    isPrivate: false,
    lastStep: [0], // Failed on first step
    name: "Large Dataset Analysis",
    resourceVersion: completeRoutineResponse.versions[0],
    schedule: null,
    startedAt: "2024-01-15T10:15:00Z",
    status: RunStatus.Failed,
    steps: [
        {
            ...completedStep,
            id: "runstep_failed_123",
            name: "Database Query",
            order: 0,
            status: RunStepStatus.Completed,
            completedAt: "2024-01-15T10:20:00Z",
            timeElapsed: 300
        },
        {
            ...inProgressStep,
            id: "runstep_failed_processing_123",
            name: "Data Processing",
            order: 1,
            status: RunStepStatus.InProgress, // Was in progress when failed
            completedAt: null,
            startedAt: "2024-01-15T10:20:00Z",
            timeElapsed: 120 // Only 2 minutes before failure
        }
    ],
    stepsCount: 2,
    team: null,
    timeElapsed: 420, // 7 minutes total
    user: {
        __typename: "User",
        ...minimalUserResponse,
    },
    wasRunAutomatically: true, // Automatic run that failed
    you: {
        __typename: "RunYou",
        canDelete: true,
        canRead: true,
        canUpdate: false,
    },
};

/**
 * Cancelled run response
 */
export const cancelledRunResponse: Run = {
    __typename: "Run",
    id: "run_cancelled_123456789",
    completedAt: "2024-01-15T10:35:00Z", // Cancellation timestamp
    completedComplexity: 0, // No steps completed
    contextSwitches: 1,
    data: JSON.stringify({
        runConfig: {
            priority: "normal"
        },
        cancellationInfo: {
            reason: "user_cancelled",
            cancelledBy: "user_123456789012345678",
            cancelledAt: "2024-01-15T10:35:00Z"
        }
    }),
    io: [
        {
            ...inputIO,
            id: "runio_cancelled_input_123",
            data: JSON.stringify({ 
                task: "Long running analysis that was cancelled"
            })
        }
    ],
    ioCount: 1,
    isPrivate: true, // Private run
    lastStep: null, // Never progressed beyond initial setup
    name: "Cancelled Analysis Task",
    resourceVersion: minimalRoutineResponse.versions[0],
    schedule: null,
    startedAt: "2024-01-15T10:30:00Z",
    status: RunStatus.Cancelled,
    steps: [
        {
            ...inProgressStep,
            id: "runstep_cancelled_123",
            name: "Initial Setup",
            order: 0,
            status: RunStepStatus.InProgress, // Was setting up when cancelled
            completedAt: null,
            startedAt: "2024-01-15T10:30:00Z",
            timeElapsed: 300 // 5 minutes before cancellation
        }
    ],
    stepsCount: 1,
    team: null,
    timeElapsed: 300, // 5 minutes total
    user: {
        __typename: "User",
        ...completeUserResponse,
    },
    wasRunAutomatically: false,
    you: {
        __typename: "RunYou",
        canDelete: true,
        canRead: true,
        canUpdate: false,
    },
};

/**
 * Scheduled run response
 */
export const scheduledRunResponse: Run = {
    __typename: "Run",
    id: "run_scheduled_123456789",
    completedAt: null,
    completedComplexity: 0,
    contextSwitches: 0,
    data: JSON.stringify({
        runConfig: {
            priority: "low",
            scheduledExecution: true,
            recurrence: "daily"
        },
        scheduleInfo: {
            nextRun: "2024-01-16T09:00:00Z",
            recurrencePattern: "0 9 * * *"
        }
    }),
    io: [
        {
            ...inputIO,
            id: "runio_scheduled_input_123",
            data: JSON.stringify({ 
                reportType: "daily_summary",
                recipients: ["admin@example.com", "team@example.com"]
            })
        }
    ],
    ioCount: 1,
    isPrivate: false,
    lastStep: null,
    name: "Daily Automated Report",
    resourceVersion: completeRoutineResponse.versions[0],
    schedule: {
        ...basicSchedule,
        id: "schedule_daily_123456789",
        startTime: "2024-01-16T09:00:00Z",
        endTime: "2024-01-16T09:30:00Z",
        recurrences: [
            {
                __typename: "ScheduleRecurrence",
                id: "recur_daily_123456789",
                recurrenceType: ScheduleRecurrenceType.Daily,
                interval: 1,
                duration: 30, // 30 minutes duration
                dayOfWeek: null,
                dayOfMonth: null,
                month: null,
                endDate: null,
                schedule: basicSchedule, // Reference to parent schedule
            }
        ]
    },
    startedAt: null, // Not started yet
    status: RunStatus.Scheduled,
    steps: [],
    stepsCount: 0,
    team: {
        __typename: "Team",
        ...minimalTeamResponse,
    },
    timeElapsed: null,
    user: null, // Team-owned scheduled run
    wasRunAutomatically: true,
    you: {
        __typename: "RunYou",
        canDelete: true,
        canRead: true,
        canUpdate: true,
    },
};

/**
 * Paused run response
 */
export const pausedRunResponse: Run = {
    __typename: "Run",
    id: "run_paused_123456789012",
    completedAt: null,
    completedComplexity: 1, // One step completed before pause
    contextSwitches: 2,
    data: JSON.stringify({
        runConfig: {
            priority: "normal",
            allowPause: true
        },
        pauseInfo: {
            pausedAt: "2024-01-15T11:15:00Z",
            pausedBy: "user_123456789012345678",
            reason: "user_requested"
        }
    }),
    io: [
        {
            ...inputIO,
            id: "runio_paused_input_123",
            data: JSON.stringify({ 
                operation: "Complex data transformation that can be paused"
            })
        }
    ],
    ioCount: 1,
    isPrivate: false,
    lastStep: [0], // Paused after first step
    name: "Pausable Data Transformation",
    resourceVersion: completeRoutineResponse.versions[0],
    schedule: null,
    startedAt: "2024-01-15T11:00:00Z",
    status: RunStatus.Paused,
    steps: [
        {
            ...completedStep,
            id: "runstep_paused_completed_123",
            name: "Data Validation",
            order: 0,
            status: RunStepStatus.Completed,
            completedAt: "2024-01-15T11:10:00Z",
            timeElapsed: 600
        },
        {
            ...inProgressStep,
            id: "runstep_paused_next_123",
            name: "Data Transformation",
            order: 1,
            status: RunStepStatus.InProgress, // Will resume from here
            completedAt: null,
            startedAt: null, // Not started yet when paused
            timeElapsed: null
        }
    ],
    stepsCount: 2,
    team: null,
    timeElapsed: 900, // 15 minutes before pause
    user: {
        __typename: "User",
        ...completeUserResponse,
    },
    wasRunAutomatically: false,
    you: {
        __typename: "RunYou",
        canDelete: true,
        canRead: true,
        canUpdate: true, // Can resume paused runs
    },
};

/**
 * Run variant states for testing
 */
export const runResponseVariants = {
    minimal: minimalRunResponse,
    complete: completeRunResponse,
    completed: completedRunResponse,
    failed: failedRunResponse,
    cancelled: cancelledRunResponse,
    scheduled: scheduledRunResponse,
    paused: pausedRunResponse,
    private: {
        ...minimalRunResponse,
        id: "run_private_123456789",
        name: "Private Task",
        isPrivate: true,
        you: {
            __typename: "RunYou",
            canDelete: false, // No access to private run
            canRead: false,
            canUpdate: false,
        },
    },
    automated: {
        ...completeRunResponse,
        id: "run_automated_123456789",
        name: "Automated System Task",
        wasRunAutomatically: true,
        user: null, // System run
        team: null,
    },
    longRunning: {
        ...completeRunResponse,
        id: "run_longrunning_123456789",
        name: "Long Running Analysis",
        startedAt: "2024-01-15T06:00:00Z", // Started 6 hours ago
        timeElapsed: 21600, // 6 hours
        contextSwitches: 12, // Many context switches over long period
        steps: [
            ...completeRunResponse.steps,
            {
                ...completedStep,
                id: "runstep_long_1_123",
                name: "Phase 1: Data Collection",
                order: 3,
                timeElapsed: 7200, // 2 hours
            },
            {
                ...completedStep,
                id: "runstep_long_2_123",
                name: "Phase 2: Processing",
                order: 4,
                timeElapsed: 10800, // 3 hours
            },
            {
                ...inProgressStep,
                id: "runstep_long_3_123",
                name: "Phase 3: Analysis",
                order: 5,
                timeElapsed: 3600, // 1 hour so far
            }
        ],
        stepsCount: 6,
    },
} as const;

/**
 * Run search response
 */
export const runSearchResponse = {
    __typename: "RunSearchResult",
    edges: [
        {
            __typename: "RunEdge",
            cursor: "cursor_1",
            node: runResponseVariants.complete,
        },
        {
            __typename: "RunEdge",
            cursor: "cursor_2",
            node: runResponseVariants.completed,
        },
        {
            __typename: "RunEdge",
            cursor: "cursor_3",
            node: runResponseVariants.scheduled,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: true,
        hasPreviousPage: false,
        startCursor: "cursor_1",
        endCursor: "cursor_3",
    },
};

/**
 * Loading and error states for UI testing
 */
export const runUIStates = {
    loading: null,
    error: {
        code: "RUN_NOT_FOUND",
        message: "The requested run could not be found",
    },
    executionError: {
        code: "RUN_EXECUTION_FAILED",
        message: "Run execution failed due to an internal error. Please try again.",
    },
    permissionError: {
        code: "RUN_ACCESS_DENIED",
        message: "You don't have permission to access this run",
    },
    cancelError: {
        code: "RUN_CANCEL_FAILED",
        message: "Failed to cancel the run. It may have already completed.",
    },
    empty: {
        edges: [],
        pageInfo: {
            __typename: "PageInfo",
            hasNextPage: false,
            hasPreviousPage: false,
            startCursor: null,
            endCursor: null,
        },
    },
};
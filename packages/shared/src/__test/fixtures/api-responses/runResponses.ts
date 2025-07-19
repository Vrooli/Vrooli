/* c8 ignore start */
/**
 * Run API Response Fixtures
 * 
 * Comprehensive fixtures for routine execution endpoints including
 * run creation, status management, and execution monitoring.
 */

import type {
    RoutineVersion,
    Run,
    RunCreateInput,
    RunStatus,
    RunStep,
    RunUpdateInput,
} from "../../../api/types.js";
import { generatePK } from "../../../id/index.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MIN_NAME_LENGTH = 1;
const MAX_NAME_LENGTH = 200;
const DEFAULT_COMPLEXITY = 5;
const EXECUTION_TIME_MS = 3600000; // 1 hour

/**
 * Run API response factory
 */
export class RunResponseFactory extends BaseAPIResponseFactory<
    Run,
    RunCreateInput,
    RunUpdateInput
> {
    protected readonly entityName = "run";

    /**
     * Create mock run data
     */
    createMockData(options?: MockDataOptions): Run {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const runId = options?.overrides?.id || generatePK().toString();

        const baseRun: Run = {
            __typename: "Run",
            id: runId,
            createdAt: now,
            updatedAt: now,
            isPrivate: false,
            completedComplexity: 0,
            contextSwitches: 0,
            status: "Scheduled",
            startedAt: null,
            completedAt: null,
            timeElapsed: null,
            title: "Test Run",
            name: "Test Run",
            steps: [],
            stepsCount: 0,
            inputs: [],
            inputsCount: 0,
            outputs: [],
            outputsCount: 0,
            routineVersion: this.createMockRoutineVersion(),
            you: {
                canDelete: true,
                canUpdate: true,
                canRead: true,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            const startedAt = new Date(Date.now().toISOString() - EXECUTION_TIME_MS).toISOString();
            return {
                ...baseRun,
                status: "Completed",
                title: "Advanced Data Processing Pipeline",
                name: "Advanced Data Processing Pipeline",
                startedAt,
                completedAt: now,
                timeElapsed: EXECUTION_TIME_MS / 1000, // in seconds
                completedComplexity: 8,
                contextSwitches: 3,
                stepsCount: 5,
                inputsCount: 2,
                outputsCount: 1,
                steps: this.createMockRunSteps(5),
                routineVersion: this.createMockRoutineVersion({
                    complexity: 8,
                    versionLabel: "2.1.0",
                    isComplete: true,
                    translations: [{
                        __typename: "RoutineVersionTranslation",
                        id: generatePK().toString(),
                        language: "en",
                        name: "Advanced Data Processing",
                        description: "Comprehensive data processing pipeline with validation and transformation steps",
                    }],
                }),
                you: {
                    canDelete: true,
                    canUpdate: false, // Can't update completed runs
                    canRead: true,
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseRun,
            ...options?.overrides,
        };
    }

    /**
     * Create run from input
     */
    createFromInput(input: RunCreateInput): Run {
        const now = new Date().toISOString();
        const runId = generatePK().toString();

        return {
            __typename: "Run",
            id: runId,
            createdAt: now,
            updatedAt: now,
            isPrivate: input.isPrivate || false,
            completedComplexity: 0,
            contextSwitches: 0,
            status: "Scheduled",
            startedAt: null,
            completedAt: null,
            timeElapsed: null,
            title: input.name || "Untitled Run",
            name: input.name || "Untitled Run",
            steps: [],
            stepsCount: 0,
            inputs: input.inputsCreate?.map((inputData, index) => ({
                __typename: "RunIO" as const,
                id: generatePK().toString(),
                index,
                data: inputData.data || {},
                name: inputData.name || `Input ${index + 1}`,
            })) || [],
            inputsCount: input.inputsCreate?.length || 0,
            outputs: [],
            outputsCount: 0,
            routineVersion: {
                __typename: "RoutineVersion",
                id: input.routineVersionConnect,
            } as RoutineVersion,
            you: {
                canDelete: true,
                canUpdate: true,
                canRead: true,
            },
        };
    }

    /**
     * Update run from input
     */
    updateFromInput(existing: Run, input: RunUpdateInput): Run {
        const updates: Partial<Run> = {
            updatedAt: new Date().toISOString(),
        };

        if (input.name !== undefined) {
            updates.name = input.name;
            updates.title = input.name;
        }
        if (input.isPrivate !== undefined) updates.isPrivate = input.isPrivate;
        if (input.status !== undefined) {
            updates.status = input.status;

            // Update timestamps based on status
            if (input.status === "InProgress" && !existing.startedAt) {
                updates.startedAt = new Date().toISOString();
            }
            if (["Completed", "Failed", "Cancelled"].includes(input.status) && !existing.completedAt) {
                updates.completedAt = new Date().toISOString();
                if (existing.startedAt) {
                    const startTime = new Date(existing.startedAt).toISOString().getTime();
                    const endTime = new Date(updates.completedAt).toISOString().getTime();
                    updates.timeElapsed = Math.floor((endTime - startTime) / 1000);
                }
            }
        }

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: RunCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.routineVersionConnect) {
            errors.routineVersionConnect = "Routine version is required";
        }

        if (input.name !== undefined) {
            if (input.name.length < MIN_NAME_LENGTH) {
                errors.name = "Name cannot be empty";
            } else if (input.name.length > MAX_NAME_LENGTH) {
                errors.name = "Name must be 200 characters or less";
            }
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: RunUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.name !== undefined) {
            if (input.name.length < MIN_NAME_LENGTH) {
                errors.name = "Name cannot be empty";
            } else if (input.name.length > MAX_NAME_LENGTH) {
                errors.name = "Name must be 200 characters or less";
            }
        }

        if (input.status !== undefined) {
            const validStatuses: RunStatus[] = ["Scheduled", "InProgress", "Completed", "Failed", "Cancelled"];
            if (!validStatuses.includes(input.status)) {
                errors.status = "Invalid status";
            }
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create mock routine version
     */
    private createMockRoutineVersion(overrides?: Partial<RoutineVersion>): RoutineVersion {
        const now = new Date().toISOString();
        const versionId = generatePK().toString();

        return {
            __typename: "RoutineVersion",
            id: versionId,
            createdAt: now,
            updatedAt: now,
            versionLabel: "1.0.0",
            versionNotes: null,
            complexity: DEFAULT_COMPLEXITY,
            isAutomatable: true,
            isComplete: true,
            isDeleted: false,
            isLatest: true,
            isPrivate: false,
            translations: [{
                __typename: "RoutineVersionTranslation",
                id: generatePK().toString(),
                language: "en",
                name: "Test Routine",
                description: "A sample routine for testing",
            }],
            translationsCount: 1,
            root: {
                __typename: "Routine",
                id: generatePK().toString(),
                createdAt: now,
                updatedAt: now,
                isInternal: false,
                isPrivate: false,
                completedAt: null,
                hasCompleteVersion: true,
                score: 85,
                bookmarks: 12,
                views: 150,
                you: {
                    canComment: true,
                    canDelete: false,
                    canBookmark: true,
                    canUpdate: false,
                    canRead: true,
                    canReact: true,
                    isBookmarked: false,
                    isViewed: true,
                    reaction: null,
                },
            },
            you: {
                canComment: true,
                canDelete: false,
                canRead: true,
                canReport: false,
                canUpdate: false,
                isBookmarked: false,
                isViewed: true,
                reaction: null,
            },
            ...overrides,
        } as RoutineVersion;
    }

    /**
     * Create mock run steps
     */
    private createMockRunSteps(count: number): RunStep[] {
        const now = new Date().toISOString();
        const steps: RunStep[] = [];

        for (let i = 0; i < count; i++) {
            steps.push({
                __typename: "RunStep",
                id: generatePK().toString(),
                order: i,
                status: i < count - 1 ? "Completed" : "InProgress",
                timeElapsed: i < count - 1 ? 600 + (i * 100) : null,
                startedAt: new Date(Date.now().toISOString() - (count - i) * 600000).toISOString(),
                completedAt: i < count - 1 ? new Date(Date.now().toISOString() - (count - i - 1) * 600000).toISOString() : null,
                name: `Step ${i + 1}`,
                contextSwitches: Math.floor(Math.random() * 3),
            });
        }

        return steps;
    }

    /**
     * Create runs with different statuses
     */
    createRunsWithAllStatuses(): Run[] {
        const statuses: RunStatus[] = ["Scheduled", "InProgress", "Completed", "Failed", "Cancelled"];
        return statuses.map((status, index) => {
            const isStarted = ["InProgress", "Completed", "Failed", "Cancelled"].includes(status);
            const isCompleted = ["Completed", "Failed", "Cancelled"].includes(status);
            const startedAt = isStarted ? new Date(Date.now().toISOString() - EXECUTION_TIME_MS).toISOString() : null;
            const completedAt = isCompleted ? new Date().toISOString() : null;

            return this.createMockData({
                overrides: {
                    id: `run_${status.toLowerCase()}_${index}`,
                    status,
                    name: `${status} Run Example`,
                    title: `${status} Run Example`,
                    startedAt,
                    completedAt,
                    timeElapsed: isCompleted ? EXECUTION_TIME_MS / 1000 : null,
                    completedComplexity: status === "Completed" ? DEFAULT_COMPLEXITY : 0,
                    you: {
                        canDelete: true,
                        canUpdate: !isCompleted, // Can't update completed runs
                        canRead: true,
                    },
                },
            });
        });
    }

    /**
     * Create long-running run
     */
    createLongRunningRun(): Run {
        const LONG_EXECUTION_TIME = 7200000; // 2 hours
        const startedAt = new Date(Date.now().toISOString() - LONG_EXECUTION_TIME).toISOString();

        return this.createMockData({
            scenario: "complete",
            overrides: {
                status: "InProgress",
                name: "Large Dataset Analysis",
                title: "Large Dataset Analysis",
                startedAt,
                timeElapsed: LONG_EXECUTION_TIME / 1000,
                completedComplexity: 6,
                contextSwitches: 8,
                stepsCount: 12,
                steps: this.createMockRunSteps(12),
            },
        });
    }

    /**
     * Create failed run with error details
     */
    createFailedRun(): Run {
        const startedAt = new Date(Date.now().toISOString() - 1800000).toISOString(); // 30 minutes ago
        const completedAt = new Date().toISOString();

        return this.createMockData({
            scenario: "complete",
            overrides: {
                status: "Failed",
                name: "Data Validation Pipeline",
                title: "Data Validation Pipeline",
                startedAt,
                completedAt,
                timeElapsed: 1800, // 30 minutes
                completedComplexity: 3, // Partial completion
                contextSwitches: 5,
                stepsCount: 8,
                steps: this.createMockRunSteps(8).map((step, index) => ({
                    ...step,
                    status: index < 5 ? "Completed" : index === 5 ? "Failed" : "Cancelled",
                })),
                you: {
                    canDelete: true,
                    canUpdate: false, // Can't update failed runs
                    canRead: true,
                },
            },
        });
    }

    /**
     * Create run execution conflict error response
     */
    createExecutionConflictErrorResponse(currentStatus: RunStatus, requestedAction: string) {
        return this.createBusinessErrorResponse("conflict", {
            field: "status",
            currentValue: currentStatus,
            requestedAction,
            message: `Cannot ${requestedAction} run with status ${currentStatus}`,
            validActions: this.getValidActionsForStatus(currentStatus),
        });
    }

    /**
     * Create resource limit error response
     */
    createResourceLimitErrorResponse(resourceType: string, limit: number) {
        return this.createBusinessErrorResponse("limit", {
            resource: resourceType,
            limit,
            current: limit,
            message: `${resourceType} limit reached`,
            upgradeRequired: true,
        });
    }

    /**
     * Get valid actions for a run status
     */
    private getValidActionsForStatus(status: RunStatus): string[] {
        const actionMap: Record<RunStatus, string[]> = {
            "Scheduled": ["start", "cancel", "delete"],
            "InProgress": ["cancel"],
            "Completed": ["delete"],
            "Failed": ["restart", "delete"],
            "Cancelled": ["restart", "delete"],
        };
        return actionMap[status] || [];
    }
}

/**
 * Pre-configured run response scenarios
 */
export const runResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<RunCreateInput>) => {
        const factory = new RunResponseFactory();
        const defaultInput: RunCreateInput = {
            routineVersionConnect: generatePK().toString(),
            name: "Test Run",
            isPrivate: false,
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (run?: Run) => {
        const factory = new RunResponseFactory();
        return factory.createSuccessResponse(
            run || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new RunResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: Run, updates?: Partial<RunUpdateInput>) => {
        const factory = new RunResponseFactory();
        const run = existing || factory.createMockData({ scenario: "complete" });
        const input: RunUpdateInput = {
            id: run.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(run, input),
        );
    },

    startSuccess: (runId?: string) => {
        const factory = new RunResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    id: runId,
                    status: "InProgress",
                    startedAt: new Date().toISOString(),
                },
            }),
        );
    },

    completeSuccess: (runId?: string) => {
        const factory = new RunResponseFactory();
        const startedAt = new Date(Date.now().toISOString() - EXECUTION_TIME_MS).toISOString();
        const completedAt = new Date().toISOString();
        return factory.createSuccessResponse(
            factory.createMockData({
                scenario: "complete",
                overrides: {
                    id: runId,
                    status: "Completed",
                    startedAt,
                    completedAt,
                    timeElapsed: EXECUTION_TIME_MS / 1000,
                    completedComplexity: DEFAULT_COMPLEXITY,
                },
            }),
        );
    },

    cancelSuccess: (runId?: string) => {
        const factory = new RunResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    id: runId,
                    status: "Cancelled",
                    completedAt: new Date().toISOString(),
                },
            }),
        );
    },

    longRunningSuccess: () => {
        const factory = new RunResponseFactory();
        return factory.createSuccessResponse(
            factory.createLongRunningRun(),
        );
    },

    failedRunSuccess: () => {
        const factory = new RunResponseFactory();
        return factory.createSuccessResponse(
            factory.createFailedRun(),
        );
    },

    listSuccess: (runs?: Run[]) => {
        const factory = new RunResponseFactory();
        return factory.createPaginatedResponse(
            runs || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: runs?.length || DEFAULT_COUNT },
        );
    },

    statusFilteredSuccess: (status: RunStatus) => {
        const factory = new RunResponseFactory();
        const runs = factory.createRunsWithAllStatuses()
            .filter(run => run.status === status);
        return factory.createPaginatedResponse(
            runs,
            { page: 1, totalCount: runs.length },
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new RunResponseFactory();
        return factory.createValidationErrorResponse({
            routineVersionConnect: "Routine version is required",
            name: "Name must be 200 characters or less",
        });
    },

    notFoundError: (runId?: string) => {
        const factory = new RunResponseFactory();
        return factory.createNotFoundErrorResponse(
            runId || "non-existent-run",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new RunResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "execute",
            ["run:execute"],
        );
    },

    executionConflictError: (currentStatus?: RunStatus, action?: string) => {
        const factory = new RunResponseFactory();
        return factory.createExecutionConflictErrorResponse(
            currentStatus || "Completed",
            action || "start",
        );
    },

    resourceLimitError: (resourceType = "concurrent runs", limit = 5) => {
        const factory = new RunResponseFactory();
        return factory.createResourceLimitErrorResponse(resourceType, limit);
    },

    routineNotFoundError: (routineVersionId?: string) => {
        const factory = new RunResponseFactory();
        return factory.createNotFoundErrorResponse(
            routineVersionId || "non-existent-routine-version",
        );
    },

    // MSW handlers
    handlers: {
        success: () => new RunResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new RunResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new RunResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const runResponseFactory = new RunResponseFactory();

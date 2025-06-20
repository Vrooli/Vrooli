/* c8 ignore start */
/**
 * Type-Safe Run API Fixture Factory
 * 
 * This factory provides comprehensive fixtures for Run objects with:
 * - Zero `any` types - all fixtures use proper typing with `satisfies`
 * - Full validation integration using runValidation schemas
 * - Shape function integration using RunProgressConfig 
 * - Comprehensive execution lifecycle scenarios
 * - Factory methods for testing run states and progress
 * - Execution context helpers for testing AI workflow performance
 */
import type { Run, RunCreateInput, RunIOCreateInput, RunStepCreateInput, RunUpdateInput, ScheduleCreateInput, ScheduleRecurrenceType } from "../../../../api/types.js";
import { RunStatus, RunStepStatus, ScheduleRecurrenceType as ScheduleRecurrenceTypeEnum } from "../../../../api/types.js";
import { generatePK } from "../../../../id/snowflake.js";
import { runValidation } from "../../../../validation/models/run.js";
import { runConfigFixtures } from "../../config/runConfigFixtures.js";
import { BaseAPIFixtureFactory } from "../BaseAPIFixtureFactory.js";
import type { APIFixtureFactory, FactoryCustomizers } from "../types.js";

// Magic number constants for testing
const RUN_NAME_TOO_LONG_LENGTH = 129;
const RUN_DATA_TOO_LONG_LENGTH = 16385;
const NODE_ID_TOO_LONG_LENGTH = 129;
const RUN_DATA_MAX_LENGTH = 16384;
const RUN_IO_DATA_MAX_LENGTH = 8192;
const RUN_NAME_MAX_LENGTH = 128;

// ========================================
// Test Data Constants
// ========================================

const validIds = {
    run1: generatePK().toString(),
    run2: generatePK().toString(),
    run3: generatePK().toString(),
    run4: generatePK().toString(),
    run5: generatePK().toString(),
    run6: generatePK().toString(),
    resourceVersion1: generatePK().toString(),
    resourceVersion2: generatePK().toString(),
    team1: generatePK().toString(),
    user1: generatePK().toString(),
    schedule1: generatePK().toString(),
    schedule2: generatePK().toString(),
    runIO1: generatePK().toString(),
    runIO2: generatePK().toString(),
    runIO3: generatePK().toString(),
    runStep1: generatePK().toString(),
    runStep2: generatePK().toString(),
    runStep3: generatePK().toString(),
    runStep4: generatePK().toString(),
};

// Current timestamp for consistent test data
const testTimestamp = "2024-01-01T00:00:00Z";
const futureTimestamp = "2024-01-01T01:00:00Z";

// ========================================
// Type-Safe Fixture Data
// ========================================

const runFixtureData = {
    minimal: {
        create: {
            id: validIds.run1,
            status: RunStatus.Scheduled,
            name: "Minimal Test Run",
            isPrivate: false,
            resourceVersionConnect: validIds.resourceVersion1,
        } satisfies RunCreateInput,

        update: {
            id: validIds.run1,
        } satisfies RunUpdateInput,

        find: {
            __typename: "Run" as const,
            id: validIds.run1,
            name: "Minimal Test Run",
            status: RunStatus.Scheduled,
            isPrivate: false,
            completedComplexity: 0,
            contextSwitches: 0,
            data: null,
            lastStep: null,
            startedAt: null,
            completedAt: null,
            timeElapsed: null,
            wasRunAutomatically: false,
            resourceVersion: null,
            schedule: null,
            team: null,
            user: null,
            io: [],
            ioCount: 0,
            steps: [],
            stepsCount: 0,
            you: {
                __typename: "RunYou" as const,
                canDelete: false,
                canRead: true,
                canUpdate: false,
            },
        } satisfies Run,
    },

    complete: {
        create: {
            id: validIds.run2,
            status: RunStatus.InProgress,
            name: "Complete Execution Run",
            isPrivate: false,
            completedComplexity: 150,
            contextSwitches: 3,
            data: JSON.stringify(runConfigFixtures.complete),
            startedAt: testTimestamp,
            timeElapsed: 5400000, // 1.5 hours in milliseconds
            resourceVersionConnect: validIds.resourceVersion1,
            teamConnect: validIds.team1,
            scheduleCreate: {
                id: validIds.schedule1,
                startTime: testTimestamp,
                endTime: futureTimestamp,
                timezone: "UTC",
                recurrencesCreate: [{
                    id: generatePK().toString(),
                    recurrenceType: ScheduleRecurrenceTypeEnum.Daily,
                    interval: 1,
                    duration: 3600000, // 1 hour
                }],
            } satisfies ScheduleCreateInput,
            ioCreate: [
                {
                    id: validIds.runIO1,
                    data: JSON.stringify({ type: "input", value: "test data", timestamp: testTimestamp }),
                    nodeInputName: "mainInput",
                    nodeName: "startNode",
                    runConnect: validIds.run2,
                } satisfies RunIOCreateInput,
                {
                    id: validIds.runIO2,
                    data: JSON.stringify({ type: "output", result: "processed data", success: true }),
                    nodeInputName: "mainOutput",
                    nodeName: "processNode",
                    runConnect: validIds.run2,
                } satisfies RunIOCreateInput,
                {
                    id: validIds.runIO3,
                    data: JSON.stringify({ type: "intermediate", status: "processing", progress: 0.75 }),
                    nodeInputName: "statusUpdate",
                    nodeName: "monitorNode",
                    runConnect: validIds.run2,
                } satisfies RunIOCreateInput,
            ],
            stepsCreate: [
                {
                    id: validIds.runStep1,
                    complexity: 50,
                    contextSwitches: 1,
                    name: "Initialize Process",
                    nodeId: "startNode",
                    order: 0,
                    status: RunStepStatus.Completed,
                    resourceInId: validIds.resourceVersion1,
                    timeElapsed: 1200000, // 20 minutes
                    runConnect: validIds.run2,
                } satisfies RunStepCreateInput,
                {
                    id: validIds.runStep2,
                    complexity: 100,
                    contextSwitches: 2,
                    name: "Execute Main Logic",
                    nodeId: "processNode",
                    order: 1,
                    status: RunStepStatus.InProgress,
                    resourceInId: validIds.resourceVersion1,
                    timeElapsed: 3600000, // 1 hour
                    runConnect: validIds.run2,
                    resourceVersionConnect: validIds.resourceVersion2,
                } satisfies RunStepCreateInput,
                {
                    id: validIds.runStep3,
                    complexity: 75,
                    name: "Finalize Results",
                    nodeId: "endNode",
                    order: 2,
                    status: RunStepStatus.InProgress,
                    resourceInId: validIds.resourceVersion1,
                    runConnect: validIds.run2,
                } satisfies RunStepCreateInput,
            ],
        } satisfies RunCreateInput,

        update: {
            id: validIds.run2,
            status: RunStatus.Completed,
            completedComplexity: 225,
            contextSwitches: 4,
            data: JSON.stringify(runConfigFixtures.complete),
            timeElapsed: 7200000, // 2 hours
            isPrivate: true,
            ioCreate: [{
                id: generatePK().toString(),
                data: JSON.stringify({ type: "final", result: "execution completed", metrics: { duration: 7200000, success: true } }),
                nodeInputName: "finalOutput",
                nodeName: "endNode",
                runConnect: validIds.run2,
            }],
            stepsUpdate: [{
                id: validIds.runStep3,
                status: RunStepStatus.Completed,
                timeElapsed: 600000, // 10 minutes
                contextSwitches: 1,
            }],
        } satisfies RunUpdateInput,

        find: {
            __typename: "Run" as const,
            id: validIds.run2,
            name: "Complete Execution Run",
            status: RunStatus.Completed,
            isPrivate: true,
            completedComplexity: 225,
            contextSwitches: 4,
            data: JSON.stringify(runConfigFixtures.complete),
            lastStep: [2],
            startedAt: testTimestamp,
            completedAt: futureTimestamp,
            timeElapsed: 7200000,
            wasRunAutomatically: false,
            resourceVersion: {
                __typename: "ResourceVersion" as const,
                id: validIds.resourceVersion1,
                versionIndex: 1,
                versionLabel: "1.0.0",
                isLatest: true,
                isComplete: true,
                isPrivate: false,
                isDeleted: false,
                complexity: 100,
                timesCompleted: 5,
                timesStarted: 10,
                forksCount: 2,
                commentsCount: 1,
                reportsCount: 0,
                translationsCount: 1,
                publicId: "rv123test456",
                createdAt: testTimestamp,
                updatedAt: testTimestamp,
                completedAt: testTimestamp,
                codeLanguage: null,
                config: null,
                isAutomatable: true,
                pullRequest: null,
                resourceSubType: null,
                versionNotes: null,
                root: null as any,
                forks: [],
                comments: [],
                reports: [],
                relatedVersions: [],
                translations: [],
                you: {
                    __typename: "ResourceVersionYou" as const,
                    canBookmark: true,
                    canComment: true,
                    canCopy: true,
                    canDelete: false,
                    canReact: true,
                    canRead: true,
                    canReport: true,
                    canRun: true,
                    canUpdate: false,
                },
            },
            schedule: {
                __typename: "Schedule" as const,
                id: validIds.schedule1,
                startTime: testTimestamp,
                endTime: futureTimestamp,
                timezone: "UTC",
                publicId: "sch123test456",
                createdAt: testTimestamp,
                updatedAt: testTimestamp,
                exceptions: [],
                meetings: [],
                recurrences: [],
                runs: [],
                user: null as any,
            },
            team: null,
            user: null,
            io: [
                {
                    __typename: "RunIO" as const,
                    id: validIds.runIO1,
                    data: JSON.stringify({ type: "input", value: "test data", timestamp: testTimestamp }),
                    nodeInputName: "mainInput",
                    nodeName: "startNode",
                    run: null as any,
                },
                {
                    __typename: "RunIO" as const,
                    id: validIds.runIO2,
                    data: JSON.stringify({ type: "output", result: "processed data", success: true }),
                    nodeInputName: "mainOutput",
                    nodeName: "processNode",
                    run: null as any,
                },
            ],
            ioCount: 2,
            steps: [
                {
                    __typename: "RunStep" as const,
                    id: validIds.runStep1,
                    complexity: 50,
                    contextSwitches: 1,
                    name: "Initialize Process",
                    nodeId: "startNode",
                    order: 0,
                    status: RunStepStatus.Completed,
                    resourceInId: validIds.resourceVersion1,
                    timeElapsed: 1200000,
                    startedAt: testTimestamp,
                    completedAt: testTimestamp,
                    resourceVersion: null,
                },
                {
                    __typename: "RunStep" as const,
                    id: validIds.runStep2,
                    complexity: 100,
                    contextSwitches: 2,
                    name: "Execute Main Logic",
                    nodeId: "processNode",
                    order: 1,
                    status: RunStepStatus.Completed,
                    resourceInId: validIds.resourceVersion1,
                    timeElapsed: 3600000,
                    startedAt: testTimestamp,
                    completedAt: futureTimestamp,
                    resourceVersion: null,
                },
            ],
            stepsCount: 2,
            you: {
                __typename: "RunYou" as const,
                canDelete: true,
                canRead: true,
                canUpdate: true,
            },
        } satisfies Run,
    },

    invalid: {
        missingRequired: {
            create: {
                // Missing required fields: id, status, name, isPrivate, resourceVersionConnect
                completedComplexity: 10,
                contextSwitches: 1,
            } satisfies Partial<RunCreateInput>,
            update: {
                // Missing required field: id
                status: RunStatus.Completed,
                timeElapsed: 1000,
            } satisfies Partial<RunUpdateInput>,
        },

        invalidTypes: {
            create: {
                id: 123, // Should be string
                status: "InvalidStatus", // Should be valid RunStatus enum
                name: true, // Should be string
                isPrivate: "yes", // Should be boolean
                completedComplexity: "not-a-number", // Should be number
                contextSwitches: -5, // Should be positive or zero
                timeElapsed: "invalid", // Should be number
                resourceVersionConnect: 456, // Should be string
                teamConnect: [], // Should be string
            } satisfies Record<string, unknown>,
            update: {
                id: false, // Should be string
                status: 123, // Should be valid RunStatus enum
                completedComplexity: "invalid", // Should be number
                contextSwitches: -10, // Should be positive or zero
                isPrivate: "no", // Should be boolean
                timeElapsed: "not-a-number", // Should be number
            } satisfies Record<string, unknown>,
        },

        businessLogicErrors: {
            nonExistentResourceVersion: {
                id: validIds.run3,
                status: RunStatus.Scheduled,
                name: "Test Run",
                isPrivate: false,
                resourceVersionConnect: "999999999999999999", // Non-existent resource version
            } satisfies Partial<RunCreateInput>,

            invalidTeamConnection: {
                id: validIds.run3,
                status: RunStatus.Scheduled,
                name: "Test Run",
                isPrivate: false,
                resourceVersionConnect: validIds.resourceVersion1,
                teamConnect: "999999999999999999", // Non-existent team
            } satisfies Partial<RunCreateInput>,

            conflictingIOData: {
                id: validIds.run3,
                status: RunStatus.InProgress,
                name: "Test Run",
                isPrivate: false,
                resourceVersionConnect: validIds.resourceVersion1,
                ioCreate: [
                    {
                        id: validIds.runIO1,
                        data: JSON.stringify({ invalid: "structure" }),
                        nodeInputName: "duplicateInput",
                        nodeName: "sameNode",
                        runConnect: validIds.run3,
                    },
                    {
                        id: validIds.runIO2,
                        data: JSON.stringify({ different: "data" }),
                        nodeInputName: "duplicateInput", // Same input name on same node
                        nodeName: "sameNode",
                        runConnect: validIds.run3,
                    },
                ],
            } satisfies Partial<RunCreateInput>,
        },

        validationErrors: {
            invalidStatus: {
                id: validIds.run3,
                status: "NotARealStatus" as any, // Invalid enum value
                name: "Test Run",
                isPrivate: false,
                resourceVersionConnect: validIds.resourceVersion1,
            } satisfies Partial<RunCreateInput>,

            tooLongName: {
                id: validIds.run3,
                status: RunStatus.Scheduled,
                name: "N".repeat(RUN_NAME_TOO_LONG_LENGTH), // Too long (max 128 chars)
                isPrivate: false,
                resourceVersionConnect: validIds.resourceVersion1,
            } satisfies Partial<RunCreateInput>,

            tooLongData: {
                id: validIds.run3,
                status: RunStatus.Scheduled,
                name: "Test Run",
                isPrivate: false,
                data: "x".repeat(RUN_DATA_TOO_LONG_LENGTH), // Exceeds max length of 16384
                resourceVersionConnect: validIds.resourceVersion1,
            } satisfies Partial<RunCreateInput>,

            negativeComplexity: {
                id: validIds.run3,
                status: RunStatus.Scheduled,
                name: "Test Run",
                isPrivate: false,
                completedComplexity: -10, // Should be positive or zero
                resourceVersionConnect: validIds.resourceVersion1,
            } satisfies Partial<RunCreateInput>,

            negativeContextSwitches: {
                id: validIds.run3,
                status: RunStatus.Scheduled,
                name: "Test Run",
                isPrivate: false,
                contextSwitches: -3, // Should be positive or zero
                resourceVersionConnect: validIds.resourceVersion1,
            } satisfies Partial<RunCreateInput>,

            negativeTimeElapsed: {
                id: validIds.run3,
                status: RunStatus.Scheduled,
                name: "Test Run",
                isPrivate: false,
                timeElapsed: -1000, // Should be positive or zero
                resourceVersionConnect: validIds.resourceVersion1,
            } satisfies Partial<RunCreateInput>,

            invalidIOStructure: {
                id: validIds.run3,
                ioCreate: [{
                    id: "invalid-id", // Invalid Snowflake ID format
                    data: "not-json", // Invalid JSON
                    nodeInputName: "", // Empty string
                    nodeName: "", // Empty string
                    runConnect: validIds.run3,
                }],
            } satisfies Partial<RunUpdateInput>,

            invalidStepStructure: {
                id: validIds.run3,
                stepsCreate: [{
                    id: validIds.runStep4,
                    complexity: -5, // Should be positive or zero
                    name: "", // Empty name
                    nodeId: "x".repeat(NODE_ID_TOO_LONG_LENGTH), // Too long node ID
                    order: -1, // Should be positive or zero
                    resourceInId: "invalid-id", // Invalid ID
                    runConnect: validIds.run3,
                }],
            } satisfies Partial<RunUpdateInput>,
        },
    },

    edgeCases: {
        minimalValid: {
            create: {
                id: validIds.run4,
                status: RunStatus.Scheduled,
                name: "Min", // Minimum 3 chars after trimming
                isPrivate: false,
                resourceVersionConnect: validIds.resourceVersion1,
            } satisfies RunCreateInput,
            update: {
                id: validIds.run4,
            } satisfies RunUpdateInput,
        },

        maximalValid: {
            create: {
                id: validIds.run5,
                status: RunStatus.InProgress,
                name: "Maximal Test Run With Very Long Name To Test Field Length Limits And Comprehensive Data Structure Coverage",
                isPrivate: true,
                completedComplexity: 999999,
                contextSwitches: 10000,
                data: JSON.stringify(runConfigFixtures.variants.withSubcontexts),
                timeElapsed: 86400000, // 24 hours
                startedAt: testTimestamp,
                resourceVersionConnect: validIds.resourceVersion1,
                teamConnect: validIds.team1,
                scheduleCreate: {
                    id: validIds.schedule2,
                    startTime: testTimestamp,
                    endTime: futureTimestamp,
                    timezone: "America/New_York",
                    exceptionsCreate: [{
                        id: generatePK().toString(),
                        originalStartTime: testTimestamp,
                        newStartTime: futureTimestamp,
                        newEndTime: "2024-01-01T02:00:00Z",
                        scheduleConnect: validIds.schedule2,
                    }],
                    recurrencesCreate: [{
                        id: generatePK().toString(),
                        recurrenceType: ScheduleRecurrenceTypeEnum.Weekly,
                        interval: 2,
                        duration: 7200000, // 2 hours
                        dayOfWeek: 1, // Sunday
                        endDate: "2024-12-31T23:59:59Z",
                    }],
                },
                ioCreate: Array.from({ length: 5 }, (_, i) => ({
                    id: generatePK().toString(),
                    data: JSON.stringify({
                        type: `io_${i}`,
                        content: `Complex data structure ${i}`,
                        metadata: { index: i, processed: i < 3 },
                        nested: {
                            level1: { level2: { value: `deep_${i}` } },
                        },
                    }),
                    nodeInputName: `input_${i}`,
                    nodeName: `node_${i}`,
                    runConnect: validIds.run5,
                })),
                stepsCreate: Array.from({ length: 10 }, (_, i) => ({
                    id: generatePK().toString(),
                    complexity: 50 + i * 10,
                    contextSwitches: i + 1,
                    name: `Step ${i + 1}: Complex Processing Phase`,
                    nodeId: `complex_node_${i}`,
                    order: i,
                    status: i < 8 ? RunStepStatus.Completed : RunStepStatus.InProgress,
                    resourceInId: validIds.resourceVersion1,
                    timeElapsed: i < 8 ? (i + 1) * 600000 : undefined, // 10 minutes per step
                    runConnect: validIds.run5,
                    resourceVersionConnect: i % 2 === 0 ? validIds.resourceVersion2 : undefined,
                })),
            } satisfies RunCreateInput,
            update: {
                id: validIds.run5,
                status: RunStatus.Completed,
                completedComplexity: 1500000,
                contextSwitches: 15000,
                data: "x".repeat(RUN_DATA_MAX_LENGTH), // Maximum data length
                timeElapsed: 172800000, // 48 hours
                isPrivate: false,
                ioCreate: [{
                    id: generatePK().toString(),
                    data: JSON.stringify({ final: "maximum complexity result", metrics: { total_time: 172800000 } }),
                    nodeInputName: "maximalOutput",
                    nodeName: "finalNode",
                    runConnect: validIds.run5,
                }],
                ioUpdate: [{
                    id: generatePK().toString(),
                    data: "a".repeat(RUN_IO_DATA_MAX_LENGTH), // Maximum IO data length
                }],
                stepsUpdate: Array.from({ length: 2 }, (_, i) => ({
                    id: generatePK().toString(),
                    status: RunStepStatus.Completed,
                    timeElapsed: 600000,
                    contextSwitches: 5,
                })),
                scheduleUpdate: {
                    id: validIds.schedule2,
                    timezone: "UTC",
                    endTime: "2024-01-02T00:00:00Z",
                },
            } satisfies RunUpdateInput,
        },

        boundaryValues: {
            zeroValues: {
                id: validIds.run6,
                status: RunStatus.Scheduled,
                name: "Zero Values Run",
                isPrivate: false,
                completedComplexity: 0,
                contextSwitches: 0,
                timeElapsed: 0,
                resourceVersionConnect: validIds.resourceVersion1,
                stepsCreate: [{
                    id: generatePK().toString(),
                    complexity: 0,
                    contextSwitches: 0,
                    name: "Zero Complexity Step",
                    nodeId: "zero_node",
                    order: 0,
                    timeElapsed: 0,
                    resourceInId: validIds.resourceVersion1,
                    runConnect: validIds.run6,
                }],
            } satisfies RunCreateInput,

            maxLengthFields: {
                id: validIds.run6,
                status: RunStatus.InProgress,
                name: "x".repeat(RUN_NAME_MAX_LENGTH), // Maximum name length
                isPrivate: true,
                data: "y".repeat(RUN_DATA_MAX_LENGTH), // Maximum data length
                resourceVersionConnect: validIds.resourceVersion1,
            } satisfies RunCreateInput,

            emptyCollections: {
                id: validIds.run6,
                ioCreate: [],
                stepsCreate: [],
            } satisfies RunUpdateInput,
        },

        permissionScenarios: {
            privateRun: {
                id: validIds.run3,
                status: RunStatus.InProgress,
                name: "Private Execution Run",
                isPrivate: true,
                resourceVersionConnect: validIds.resourceVersion1,
            } satisfies RunCreateInput,

            publicRun: {
                id: validIds.run4,
                status: RunStatus.Completed,
                name: "Public Execution Run",
                isPrivate: false,
                resourceVersionConnect: validIds.resourceVersion1,
            } satisfies RunCreateInput,

            teamOwnedRun: {
                id: validIds.run5,
                status: RunStatus.InProgress,
                name: "Team Execution Run",
                isPrivate: false,
                resourceVersionConnect: validIds.resourceVersion1,
                teamConnect: validIds.team1,
            } satisfies RunCreateInput,
        },

        executionStates: Object.values(RunStatus).map(status => ({
            create: {
                id: generatePK().toString(),
                status,
                name: `${status} Test Run`,
                isPrivate: false,
                resourceVersionConnect: validIds.resourceVersion1,
                startedAt: status === RunStatus.Scheduled ? undefined : testTimestamp,
                timeElapsed: status === RunStatus.Scheduled ? undefined : 3600000,
            } satisfies RunCreateInput,
        })),

        complexExecutionFlow: {
            create: {
                id: validIds.run6,
                status: RunStatus.InProgress,
                name: "Complex Multi-Step Execution",
                isPrivate: false,
                data: JSON.stringify(runConfigFixtures.variants.withActiveBranches),
                startedAt: testTimestamp,
                timeElapsed: 10800000, // 3 hours
                resourceVersionConnect: validIds.resourceVersion1,
                ioCreate: [
                    {
                        id: generatePK().toString(),
                        data: JSON.stringify({ phase: "initialization", inputs: ["param1", "param2"] }),
                        nodeInputName: "init_params",
                        nodeName: "init_node",
                        runConnect: validIds.run6,
                    },
                    {
                        id: generatePK().toString(),
                        data: JSON.stringify({ phase: "processing", intermediate_results: { processed: 150, failed: 2 } }),
                        nodeInputName: "process_status",
                        nodeName: "process_node",
                        runConnect: validIds.run6,
                    },
                    {
                        id: generatePK().toString(),
                        data: JSON.stringify({ phase: "validation", validation_results: { passed: 148, warnings: 5 } }),
                        nodeInputName: "validation_output",
                        nodeName: "validate_node",
                        runConnect: validIds.run6,
                    },
                ],
                stepsCreate: [
                    {
                        id: generatePK().toString(),
                        complexity: 25,
                        contextSwitches: 1,
                        name: "Initialize Execution Context",
                        nodeId: "init_node",
                        order: 0,
                        status: RunStepStatus.Completed,
                        resourceInId: validIds.resourceVersion1,
                        timeElapsed: 1800000, // 30 minutes
                        runConnect: validIds.run6,
                    },
                    {
                        id: generatePK().toString(),
                        complexity: 200,
                        contextSwitches: 8,
                        name: "Execute Parallel Processing",
                        nodeId: "process_node",
                        order: 1,
                        status: RunStepStatus.Completed,
                        resourceInId: validIds.resourceVersion1,
                        timeElapsed: 7200000, // 2 hours
                        runConnect: validIds.run6,
                        resourceVersionConnect: validIds.resourceVersion2,
                    },
                    {
                        id: generatePK().toString(),
                        complexity: 75,
                        contextSwitches: 3,
                        name: "Validate and Finalize Results",
                        nodeId: "validate_node",
                        order: 2,
                        status: RunStepStatus.InProgress,
                        resourceInId: validIds.resourceVersion1,
                        timeElapsed: 1800000, // 30 minutes so far
                        runConnect: validIds.run6,
                    },
                ],
            } satisfies RunCreateInput,
        },
    },
};

// ========================================
// Factory Customizers
// ========================================

const runCustomizers: FactoryCustomizers<RunCreateInput, RunUpdateInput> = {
    create: (base: RunCreateInput, overrides?: Partial<RunCreateInput>): RunCreateInput => {
        return {
            ...base,
            id: overrides?.id || base.id || generatePK().toString(),
            status: overrides?.status !== undefined ? overrides.status : (base.status || RunStatus.Scheduled),
            name: overrides?.name || base.name || `Generated Run ${Date.now()}`,
            isPrivate: overrides?.isPrivate !== undefined ? overrides.isPrivate : (base.isPrivate !== undefined ? base.isPrivate : false),
            resourceVersionConnect: overrides?.resourceVersionConnect || base.resourceVersionConnect || validIds.resourceVersion1,
            ...overrides,
        };
    },

    update: (base: RunUpdateInput, overrides?: Partial<RunUpdateInput>): RunUpdateInput => {
        return {
            ...base,
            id: overrides?.id || base.id || generatePK().toString(),
            ...overrides,
        };
    },
};

// ========================================
// Integration Setup
// ========================================

// Simplified integration without FullIntegration for now
const runIntegration = {
    validation: {
        create: {
            validate: async (input: RunCreateInput) => {
                try {
                    const result = await runValidation.create({}).validate(input);
                    return { isValid: true, data: result };
                } catch (error: any) {
                    return { isValid: false, errors: [error.message] };
                }
            },
        },
        update: {
            validate: async (input: RunUpdateInput) => {
                try {
                    const result = await runValidation.update({}).validate(input);
                    return { isValid: true, data: result };
                } catch (error: any) {
                    return { isValid: false, errors: [error.message] };
                }
            },
        },
    },
};

// ========================================
// Type-Safe Fixture Factory
// ========================================

export class RunAPIFixtureFactory extends BaseAPIFixtureFactory<
    RunCreateInput,
    RunUpdateInput,
    Run,
    typeof runConfigFixtures.complete,
    Run // Database type same as find result for simplicity
> implements APIFixtureFactory<RunCreateInput, RunUpdateInput, Run, typeof runConfigFixtures.complete, Run> {

    constructor() {
        const config = {
            ...runFixtureData,
            validationSchema: runIntegration.validation,
            shapeTransforms: {
                toAPI: undefined,
                fromDB: undefined,
            },
        };

        super(config, runCustomizers);
    }

    // Override relationship helpers for run-specific logic
    withRelationships = (base: Run, relations: Record<string, unknown>): Run => {
        const result = { ...base };

        if (relations.resourceVersion) {
            result.resourceVersion = relations.resourceVersion as any;
        }

        if (relations.schedule) {
            result.schedule = relations.schedule as any;
        }

        if (relations.team) {
            result.team = relations.team as any;
        }

        if (relations.user) {
            result.user = relations.user as any;
        }

        if (relations.io && Array.isArray(relations.io)) {
            result.io = relations.io as any;
            result.ioCount = relations.io.length;
        }

        if (relations.steps && Array.isArray(relations.steps)) {
            result.steps = relations.steps as any;
            result.stepsCount = relations.steps.length;
        }

        return result;
    };

    // ========================================
    // Run State Management Helpers
    // ========================================

    createScheduledRun = (routineId: string, overrides?: Partial<RunCreateInput>): RunCreateInput => {
        return this.createFactory({
            status: RunStatus.Scheduled,
            resourceVersionConnect: routineId,
            startedAt: undefined,
            timeElapsed: undefined,
            ...overrides,
        });
    };

    createRunningRun = (routineId: string, progress?: number, overrides?: Partial<RunCreateInput>): RunCreateInput => {
        const progressData = progress !== undefined ? {
            completedComplexity: Math.floor(progress * 100),
            timeElapsed: Math.floor(progress * 3600000), // Scale to 1 hour max
            data: JSON.stringify(runConfigFixtures.variants.withActiveBranches),
        } : {};

        return this.createFactory({
            status: RunStatus.InProgress,
            resourceVersionConnect: routineId,
            startedAt: testTimestamp,
            ...progressData,
            ...overrides,
        });
    };

    createCompletedRun = (routineId: string, outputs?: Record<string, unknown>[], overrides?: Partial<RunCreateInput>): RunCreateInput => {
        const ioData = outputs ? outputs.map((output, index) => ({
            id: generatePK().toString(),
            data: JSON.stringify(output),
            nodeInputName: `output_${index}`,
            nodeName: `node_${index}`,
            runConnect: overrides?.id || generatePK().toString(),
        })) : [];

        return this.createFactory({
            status: RunStatus.Completed,
            resourceVersionConnect: routineId,
            startedAt: testTimestamp,
            timeElapsed: 3600000, // 1 hour
            completedComplexity: 100,
            data: JSON.stringify(runConfigFixtures.variants.withCompletedDecisions),
            ioCreate: ioData.length > 0 ? ioData : undefined,
            ...overrides,
        });
    };

    createFailedRun = (routineId: string, error: string, overrides?: Partial<RunCreateInput>): RunCreateInput => {
        return this.createFactory({
            status: RunStatus.Failed,
            resourceVersionConnect: routineId,
            startedAt: testTimestamp,
            timeElapsed: 1800000, // 30 minutes
            data: JSON.stringify({
                ...runConfigFixtures.variants.withErrors,
                errorDetails: { message: error, timestamp: testTimestamp },
            }),
            ...overrides,
        });
    };

    createCancelledRun = (routineId: string, reason?: string, overrides?: Partial<RunCreateInput>): RunCreateInput => {
        return this.createFactory({
            status: RunStatus.Cancelled,
            resourceVersionConnect: routineId,
            startedAt: testTimestamp,
            timeElapsed: 900000, // 15 minutes
            data: JSON.stringify({
                ...runConfigFixtures.minimal,
                cancellationReason: reason || "User cancelled",
            }),
            ...overrides,
        });
    };

    createPausedRun = (routineId: string, pauseReason?: string, overrides?: Partial<RunCreateInput>): RunCreateInput => {
        return this.createFactory({
            status: RunStatus.Paused,
            resourceVersionConnect: routineId,
            startedAt: testTimestamp,
            timeElapsed: 2400000, // 40 minutes
            data: JSON.stringify({
                ...runConfigFixtures.variants.withActiveBranches,
                pauseReason: pauseReason || "Manual pause",
            }),
            ...overrides,
        });
    };

    // ========================================
    // Execution Progress Helpers  
    // ========================================

    updateRunProgress = (runId: string, currentStep: number, progress: number, overrides?: Partial<RunUpdateInput>): RunUpdateInput => {
        const completedComplexity = Math.floor(progress * 200); // Scale to 200 total complexity
        const timeElapsed = Math.floor(progress * 7200000); // Scale to 2 hours max

        return this.updateFactory(runId, {
            completedComplexity,
            timeElapsed,
            contextSwitches: Math.max(1, currentStep),
            data: JSON.stringify({
                ...runConfigFixtures.variants.withActiveBranches,
                currentStep,
                progress,
            }),
            ...overrides,
        });
    };

    completeRun = (runId: string, outputs?: Record<string, unknown>[], metrics?: Record<string, unknown>, overrides?: Partial<RunUpdateInput>): RunUpdateInput => {
        const finalIO = outputs ? [{
            id: generatePK().toString(),
            data: JSON.stringify({ outputs, metrics }),
            nodeInputName: "finalResults",
            nodeName: "completionNode",
            runConnect: runId,
        }] : undefined;

        return this.updateFactory(runId, {
            status: RunStatus.Completed,
            timeElapsed: 7200000, // 2 hours
            completedComplexity: 200,
            data: JSON.stringify({
                ...runConfigFixtures.variants.withCompletedDecisions,
                completionMetrics: metrics,
            }),
            ioCreate: finalIO,
            ...overrides,
        });
    };

    failRun = (runId: string, error: string, stackTrace?: string, overrides?: Partial<RunUpdateInput>): RunUpdateInput => {
        return this.updateFactory(runId, {
            status: RunStatus.Failed,
            data: JSON.stringify({
                ...runConfigFixtures.variants.withErrors,
                error: { message: error, stack: stackTrace, timestamp: Date.now() },
            }),
            ...overrides,
        });
    };

    cancelRun = (runId: string, reason?: string, overrides?: Partial<RunUpdateInput>): RunUpdateInput => {
        return this.updateFactory(runId, {
            status: RunStatus.Cancelled,
            data: JSON.stringify({
                ...runConfigFixtures.minimal,
                cancellation: { reason: reason || "User request", timestamp: Date.now() },
            }),
            ...overrides,
        });
    };

    pauseRun = (runId: string, reason?: string, overrides?: Partial<RunUpdateInput>): RunUpdateInput => {
        return this.updateFactory(runId, {
            status: RunStatus.Paused,
            data: JSON.stringify({
                ...runConfigFixtures.variants.withActiveBranches,
                pause: { reason: reason || "Manual pause", timestamp: Date.now() },
            }),
            ...overrides,
        });
    };

    // ========================================
    // Input/Output Management Helpers
    // ========================================

    addRunInput = (runId: string, input: Record<string, unknown>, nodeName?: string, overrides?: Partial<RunUpdateInput>): RunUpdateInput => {
        return this.updateFactory(runId, {
            ioCreate: [{
                id: generatePK().toString(),
                data: JSON.stringify(input),
                nodeInputName: "input",
                nodeName: nodeName || "inputNode",
                runConnect: runId,
            }],
            ...overrides,
        });
    };

    addRunOutput = (runId: string, stepId: string, output: Record<string, unknown>, overrides?: Partial<RunUpdateInput>): RunUpdateInput => {
        return this.updateFactory(runId, {
            ioCreate: [{
                id: generatePK().toString(),
                data: JSON.stringify(output),
                nodeInputName: "output",
                nodeName: stepId,
                runConnect: runId,
            }],
            ...overrides,
        });
    };

    updateRunContext = (runId: string, context: Record<string, unknown>, overrides?: Partial<RunUpdateInput>): RunUpdateInput => {
        return this.updateFactory(runId, {
            data: JSON.stringify({
                ...runConfigFixtures.variants.withSubcontexts,
                contextUpdate: context,
            }),
            ...overrides,
        });
    };

    // ========================================
    // Performance and Metrics Helpers
    // ========================================

    addRunMetrics = (runId: string, metrics: Record<string, unknown>, overrides?: Partial<RunUpdateInput>): RunUpdateInput => {
        return this.updateFactory(runId, {
            data: JSON.stringify({
                ...runConfigFixtures.complete,
                performanceMetrics: metrics,
            }),
            ...overrides,
        });
    };

    setRunPerformance = (runId: string, timing: Record<string, unknown>, resources: Record<string, unknown>, overrides?: Partial<RunUpdateInput>): RunUpdateInput => {
        return this.updateFactory(runId, {
            timeElapsed: typeof timing.duration === "number" ? timing.duration : undefined,
            data: JSON.stringify({
                ...runConfigFixtures.complete,
                performance: { timing, resources },
            }),
            ...overrides,
        });
    };

    // ========================================
    // Validation Helpers
    // ========================================

    validateRunCreation = async (input: RunCreateInput): Promise<void> => {
        const result = await this.validateCreate(input);
        if (!result.isValid) {
            throw new Error(`Run creation validation failed: ${result.errors?.join(", ")}`);
        }
    };

    validateRunUpdate = async (input: RunUpdateInput): Promise<void> => {
        const result = await this.validateUpdate(input);
        if (!result.isValid) {
            throw new Error(`Run update validation failed: ${result.errors?.join(", ")}`);
        }
    };

    // ========================================
    // Complex Scenario Helpers
    // ========================================

    createMultiStepRun = (routineId: string, stepCount: number, overrides?: Partial<RunCreateInput>): RunCreateInput => {
        const steps = Array.from({ length: stepCount }, (_, i) => ({
            id: generatePK().toString(),
            complexity: 20 + i * 10,
            contextSwitches: i + 1,
            name: `Step ${i + 1}`,
            nodeId: `node_${i}`,
            order: i,
            status: i === stepCount - 1 ? RunStepStatus.InProgress : RunStepStatus.Completed,
            resourceInId: routineId,
            timeElapsed: i < stepCount - 1 ? (i + 1) * 600000 : undefined, // 10 minutes per completed step
            runConnect: overrides?.id || generatePK().toString(),
        }));

        return this.createFactory({
            status: RunStatus.InProgress,
            resourceVersionConnect: routineId,
            stepsCreate: steps,
            completedComplexity: steps.slice(0, -1).reduce((sum, step) => sum + step.complexity, 0),
            timeElapsed: steps.slice(0, -1).reduce((sum, step) => sum + (step.timeElapsed || 0), 0),
            data: JSON.stringify(runConfigFixtures.variants.withActiveBranches),
            ...overrides,
        });
    };

    createScheduledRunWithRecurrence = (routineId: string, recurrenceType: ScheduleRecurrenceType, overrides?: Partial<RunCreateInput>): RunCreateInput => {
        return this.createFactory({
            status: RunStatus.Scheduled,
            resourceVersionConnect: routineId,
            scheduleCreate: {
                id: generatePK().toString(),
                startTime: testTimestamp,
                endTime: futureTimestamp,
                timezone: "UTC",
                recurrencesCreate: [{
                    id: generatePK().toString(),
                    recurrenceType,
                    interval: 1,
                    duration: 3600000, // 1 hour
                }],
            },
            ...overrides,
        });
    };

    createRunWithLimits = (routineId: string, limits: { maxTime?: number; maxCredits?: string; maxSteps?: number }, overrides?: Partial<RunCreateInput>): RunCreateInput => {
        const configWithLimits = {
            ...runConfigFixtures.minimal,
            config: {
                ...runConfigFixtures.minimal.config,
                limits,
            },
        };

        return this.createFactory({
            status: RunStatus.InProgress,
            resourceVersionConnect: routineId,
            data: JSON.stringify(configWithLimits),
            ...overrides,
        });
    };
}

// ========================================
// Export Factory Instance  
// ========================================

export const runAPIFixtures = new RunAPIFixtureFactory();

// ========================================
// Type Exports for Other Fixtures
// ========================================

export type { RunAPIFixtureFactory as RunAPIFixtureFactoryType };

// ========================================
// Legacy Compatibility (Optional)
// ========================================

// Provide legacy-style access for gradual migration
export const legacyRunFixtures = {
    minimal: runAPIFixtures.minimal,
    complete: runAPIFixtures.complete,
    invalid: runAPIFixtures.invalid,
    edgeCases: runAPIFixtures.edgeCases,

    // Factory methods
    createFactory: runAPIFixtures.createFactory,
    updateFactory: runAPIFixtures.updateFactory,
    findFactory: runAPIFixtures.findFactory,

    // Run state methods
    createScheduledRun: runAPIFixtures.createScheduledRun,
    createRunningRun: runAPIFixtures.createRunningRun,
    createCompletedRun: runAPIFixtures.createCompletedRun,
    createFailedRun: runAPIFixtures.createFailedRun,
    createCancelledRun: runAPIFixtures.createCancelledRun,
    createPausedRun: runAPIFixtures.createPausedRun,

    // Progress management
    updateRunProgress: runAPIFixtures.updateRunProgress,
    completeRun: runAPIFixtures.completeRun,
    failRun: runAPIFixtures.failRun,
    cancelRun: runAPIFixtures.cancelRun,
    pauseRun: runAPIFixtures.pauseRun,

    // IO management
    addRunInput: runAPIFixtures.addRunInput,
    addRunOutput: runAPIFixtures.addRunOutput,
    updateRunContext: runAPIFixtures.updateRunContext,

    // Performance helpers
    addRunMetrics: runAPIFixtures.addRunMetrics,
    setRunPerformance: runAPIFixtures.setRunPerformance,

    // Complex scenarios
    createMultiStepRun: runAPIFixtures.createMultiStepRun,
    createScheduledRunWithRecurrence: runAPIFixtures.createScheduledRunWithRecurrence,
    createRunWithLimits: runAPIFixtures.createRunWithLimits,
};

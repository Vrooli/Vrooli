import type { RunCreateInput, RunUpdateInput } from "../../../api/types.js";
import { RunStatus, RunStepStatus } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures, testValues } from "../../../validation/models/__test/validationTestUtils.js";
import { runValidation } from "../../../validation/models/run.js";

// Generate consistent test IDs
const validIds = {
    runId1: testValues.snowflakeId(),
    runId2: testValues.snowflakeId(),
    runId3: testValues.snowflakeId(),
    resourceVersionId1: testValues.snowflakeId(),
    teamId1: testValues.snowflakeId(),
    runIOId1: testValues.snowflakeId(),
    runIOId2: testValues.snowflakeId(),
    scheduleId1: testValues.snowflakeId(),
    stepId1: testValues.snowflakeId(),
    stepId2: testValues.snowflakeId(),
};

export const runFixtures: ModelTestFixtures<RunCreateInput, RunUpdateInput> = {
    minimal: {
        create: {
            id: validIds.runId1,
            status: RunStatus.InProgress,
            name: "Test Run",
            isPrivate: false,
            resourceVersionConnect: validIds.resourceVersionId1,
        },
        update: {
            id: validIds.runId1,
        },
    },
    complete: {
        create: {
            id: validIds.runId2,
            completedComplexity: 5,
            contextSwitches: 2,
            data: JSON.stringify({ testData: "value", config: { setting: true } }),
            isPrivate: false,
            status: RunStatus.InProgress,
            name: "Complete Test Run",
            timeElapsed: 3600,
            ioCreate: [
                {
                    id: validIds.runIOId1,
                    data: JSON.stringify({ input: "value1" }),
                    nodeInputName: "input1",
                    nodeName: "node1",
                    runConnect: validIds.runId2,
                },
                {
                    id: validIds.runIOId2,
                    data: JSON.stringify({ output: "result1" }),
                    nodeInputName: "output1",
                    nodeName: "node2",
                    runConnect: validIds.runId2,
                },
            ],
            resourceVersionConnect: validIds.resourceVersionId1,
            scheduleCreate: {
                id: validIds.scheduleId1,
                startTime: new Date().toISOString(),
                endTime: new Date(Date.now() + 86400000).toISOString(),
                timezone: "UTC",
            },
            stepsCreate: [
                {
                    id: validIds.stepId1,
                    complexity: 1,
                    name: "Step 1",
                    nodeId: "node1",
                    order: 0,
                    resourceInId: validIds.resourceVersionId1,
                    status: RunStepStatus.Completed,
                    runConnect: validIds.runId2,
                },
                {
                    id: validIds.stepId2,
                    complexity: 2,
                    name: "Step 2", 
                    nodeId: "node2",
                    order: 1,
                    resourceInId: validIds.resourceVersionId1,
                    status: RunStepStatus.InProgress,
                    runConnect: validIds.runId2,
                },
            ],
            teamConnect: validIds.teamId1,
        },
        update: {
            id: validIds.runId2,
            completedComplexity: 10,
            contextSwitches: 3,
            data: JSON.stringify({ updatedData: "newValue" }),
            isPrivate: true,
            timeElapsed: 7200,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required fields: id, status, name, isPrivate, resourceVersionConnect
                completedComplexity: 5,
            },
            update: {
                // Missing required field: id
                completedComplexity: 10,
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                status: "InvalidStatus", // Should be valid RunStatus enum
                name: 123, // Should be string
                completedComplexity: "not a number", // Should be number
                contextSwitches: -5, // Should be positive or zero
                isPrivate: "not a boolean", // Should be boolean
                timeElapsed: -100, // Should be positive or zero
                resourceVersionConnect: 123, // Should be string
            },
            update: {
                id: 123, // Should be string
                completedComplexity: "not a number", // Should be number
                contextSwitches: -5, // Should be positive or zero
                isPrivate: "not a boolean", // Should be boolean
                timeElapsed: -100, // Should be positive or zero
            },
        },
        invalidId: {
            create: {
                id: "invalid-id", // Invalid Snowflake ID format
                status: RunStatus.InProgress,
                name: "Test Run",
                isPrivate: false,
                resourceVersionConnect: validIds.resourceVersionId1,
            },
            update: {
                id: "invalid-id", // Invalid Snowflake ID format
            },
        },
        invalidStatus: {
            create: {
                id: validIds.runId3,
                status: "NotAValidStatus",
                name: "Test Run",
                isPrivate: false,
                resourceVersionConnect: validIds.resourceVersionId1,
            },
        },
        tooLongData: {
            create: {
                id: validIds.runId3,
                status: RunStatus.InProgress,
                name: "Test Run",
                isPrivate: false,
                data: "x".repeat(16385), // Exceeds max length of 16384
                resourceVersionConnect: validIds.resourceVersionId1,
            },
        },
        tooLongName: {
            create: {
                id: validIds.runId3,
                status: RunStatus.InProgress,
                name: "x".repeat(129), // Exceeds max length of 128
                isPrivate: false,
                resourceVersionConnect: validIds.resourceVersionId1,
            },
        },
    },
    edgeCases: {
        emptyData: {
            create: {
                id: validIds.runId3,
                status: RunStatus.InProgress,
                name: "Test Run",
                isPrivate: false,
                data: "", // Empty string should be allowed
                resourceVersionConnect: validIds.resourceVersionId1,
            },
        },
        allStatuses: Object.values(RunStatus).map(status => ({
            create: {
                id: testValues.snowflakeId(),
                status,
                name: `Test Run - ${status}`,
                isPrivate: false,
                resourceVersionConnect: validIds.resourceVersionId1,
            },
        })),
        zeroValues: {
            create: {
                id: validIds.runId3,
                status: RunStatus.InProgress,
                name: "Test Run",
                isPrivate: false,
                completedComplexity: 0,
                contextSwitches: 0,
                timeElapsed: 0,
                resourceVersionConnect: validIds.resourceVersionId1,
            },
        },
        maxData: {
            create: {
                id: validIds.runId3,
                status: RunStatus.InProgress,
                name: "Test Run",
                isPrivate: false,
                data: "x".repeat(16384), // Exactly at max length
                resourceVersionConnect: validIds.resourceVersionId1,
            },
        },
    },
};

// Customizers with proper type annotations
const customizers = {
    create: (base: RunCreateInput): RunCreateInput => ({
        ...base,
        name: testValues.shortString("run"),
        resourceVersionConnect: testValues.snowflakeId(),
    }),
    update: (base: RunUpdateInput): RunUpdateInput => ({
        ...base,
        id: testValues.snowflakeId(),
    }),
};

export const runTestDataFactory = new TypedTestDataFactory(runFixtures, runValidation, customizers);
export const typedRunFixtures = createTypedFixtures(runFixtures, runValidation);

// Backward compatibility - keep the old factory available
export const legacyRunTestDataFactory = new TestDataFactory(runFixtures, customizers);

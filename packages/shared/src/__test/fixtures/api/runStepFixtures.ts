import type { RunStepCreateInput, RunStepUpdateInput } from "../../../api/types.js";
import { RunStepStatus } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { runRoutineStepValidation } from "../../../validation/models/runStep.js";

// Constants for validation limits
const NODE_ID_MAX_LENGTH = 128;
const NODE_ID_INVALID_LENGTH = NODE_ID_MAX_LENGTH + 1;

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
    id6: "123456789012345683",
    id7: "123456789012345684",
    id8: "123456789012345685",
};

// Shared runStep test fixtures
export const runStepFixtures: ModelTestFixtures<RunStepCreateInput, RunStepUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            complexity: 1,
            name: "Minimal Step",
            nodeId: "node_001",
            order: 0,
            resourceInId: validIds.id2,
            runConnect: validIds.id3,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            complexity: 150,
            contextSwitches: 5,
            name: "Complex Processing Step",
            nodeId: "advanced_processor_node_001",
            order: 3,
            status: RunStepStatus.InProgress,
            resourceInId: validIds.id2,
            timeElapsed: 2500,
            runConnect: validIds.id3,
            resourceVersionConnect: validIds.id4,
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 123,
            unknownField3: true,
        } as any,
        update: {
            id: validIds.id1,
            contextSwitches: 8,
            status: RunStepStatus.Completed,
            timeElapsed: 4200,
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 456,
        } as any,
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id, complexity, name, nodeId, order, resourceInId, and runConnect
                status: RunStepStatus.InProgress,
            },
            update: {
                // Missing required id
                status: RunStepStatus.Completed,
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                complexity: "not-a-number", // Should be number
                contextSwitches: "invalid", // Should be number
                name: 456, // Should be string
                nodeId: 789, // Should be string
                order: "not-a-number", // Should be number
                status: "InvalidStatus", // Invalid enum value
                resourceInId: 101112, // Should be string
                timeElapsed: "invalid", // Should be number
                runConnect: 131415, // Should be string
                resourceVersionConnect: 161718, // Should be string
            },
            update: {
                id: validIds.id1,
                contextSwitches: "invalid", // Should be number
                status: "InvalidStatus", // Invalid enum value
                timeElapsed: "invalid", // Should be number
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                complexity: 1,
                name: "Test Step",
                nodeId: "node_001",
                order: 0,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
            update: {
                id: "invalid-id",
            },
        },
        missingComplexity: {
            create: {
                id: validIds.id1,
                name: "Test Step",
                nodeId: "node_001",
                order: 0,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
                // Missing required complexity
            },
        },
        missingName: {
            create: {
                id: validIds.id1,
                complexity: 1,
                nodeId: "node_001",
                order: 0,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
                // Missing required name
            },
        },
        missingNodeId: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "Test Step",
                order: 0,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
                // Missing required nodeId
            },
        },
        missingOrder: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "Test Step",
                nodeId: "node_001",
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
                // Missing required order
            },
        },
        missingResourceInId: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "Test Step",
                nodeId: "node_001",
                order: 0,
                runConnect: validIds.id3,
                // Missing required resourceInId
            },
        },
        missingRunConnect: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "Test Step",
                nodeId: "node_001",
                order: 0,
                resourceInId: validIds.id2,
                // Missing required runConnect
            },
        },
        negativeComplexity: {
            create: {
                id: validIds.id1,
                complexity: -1, // Should be positive or zero
                name: "Test Step",
                nodeId: "node_001",
                order: 0,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        negativeOrder: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "Test Step",
                nodeId: "node_001",
                order: -1, // Should be positive or zero
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        zeroContextSwitches: {
            create: {
                id: validIds.id1,
                complexity: 1,
                contextSwitches: 0, // Should be positive (minimum 1)
                name: "Test Step",
                nodeId: "node_001",
                order: 0,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        negativeTimeElapsed: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "Test Step",
                nodeId: "node_001",
                order: 0,
                resourceInId: validIds.id2,
                timeElapsed: -100, // Should be positive or zero
                runConnect: validIds.id3,
            },
        },
        longNodeId: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "Test Step",
                nodeId: "x".repeat(NODE_ID_INVALID_LENGTH), // Too long (exceeds max length)
                order: 0,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        emptyName: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "", // Empty string becomes undefined
                nodeId: "node_001",
                order: 0,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        emptyNodeId: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "Test Step",
                nodeId: "", // Empty string becomes undefined
                order: 0,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
    },
    edgeCases: {
        zeroComplexity: {
            create: {
                id: validIds.id1,
                complexity: 0, // Valid minimum
                name: "Simple Step",
                nodeId: "simple_node",
                order: 0,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        highComplexity: {
            create: {
                id: validIds.id1,
                complexity: 999999, // Very high complexity
                name: "Complex Step",
                nodeId: "complex_node",
                order: 0,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        singleContextSwitch: {
            create: {
                id: validIds.id1,
                complexity: 50,
                contextSwitches: 1, // Minimum valid value
                name: "Context Switch Step",
                nodeId: "context_node",
                order: 0,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        manyContextSwitches: {
            create: {
                id: validIds.id1,
                complexity: 100,
                contextSwitches: 50, // High context switches
                name: "Multi Context Step",
                nodeId: "multi_context_node",
                order: 0,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        maxLengthNodeId: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "Max NodeId Step",
                nodeId: "a".repeat(NODE_ID_MAX_LENGTH), // Maximum length
                order: 0,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        highOrderStep: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "High Order Step",
                nodeId: "high_order_node",
                order: 999, // High order number
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        withoutStatus: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "No Status Step",
                nodeId: "no_status_node",
                order: 0,
                // status is optional
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        inProgressStatus: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "In Progress Step",
                nodeId: "progress_node",
                order: 0,
                status: RunStepStatus.InProgress,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        completedStatus: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "Completed Step",
                nodeId: "completed_node",
                order: 0,
                status: RunStepStatus.Completed,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        skippedStatus: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "Skipped Step",
                nodeId: "skipped_node",
                order: 0,
                status: RunStepStatus.Skipped,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        statusInAllCases: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "Status Test Step",
                nodeId: "status_test_node",
                order: 0,
                status: RunStepStatus.InProgress, // Test all valid statuses in the test
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
            },
        },
        withResourceVersion: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "Versioned Step",
                nodeId: "versioned_node",
                order: 0,
                resourceInId: validIds.id2,
                runConnect: validIds.id3,
                resourceVersionConnect: validIds.id4,
            },
        },
        longRunningStep: {
            create: {
                id: validIds.id1,
                complexity: 500,
                name: "Long Running Step",
                nodeId: "long_running_node",
                order: 0,
                resourceInId: validIds.id2,
                timeElapsed: 300000, // 5 minutes in milliseconds
                runConnect: validIds.id3,
            },
        },
        quickStep: {
            create: {
                id: validIds.id1,
                complexity: 1,
                name: "Quick Step",
                nodeId: "quick_node",
                order: 0,
                resourceInId: validIds.id2,
                timeElapsed: 50, // 50 milliseconds
                runConnect: validIds.id3,
            },
        },
        updateContextSwitches: {
            update: {
                id: validIds.id1,
                contextSwitches: 3,
            },
        },
        updateStatus: {
            update: {
                id: validIds.id1,
                status: RunStepStatus.Completed,
            },
        },
        updateTimeElapsed: {
            update: {
                id: validIds.id1,
                timeElapsed: 1500,
            },
        },
        updateAllFields: {
            update: {
                id: validIds.id1,
                contextSwitches: 10,
                status: RunStepStatus.Skipped,
                timeElapsed: 5000,
            },
        },
        updateOnlyId: {
            update: {
                id: validIds.id1,
                // Only required field
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers = {
    create: (base: RunStepCreateInput): RunStepCreateInput => ({
        ...base,
        id: base.id || validIds.id1,
        complexity: base.complexity !== undefined ? base.complexity : 1,
        name: base.name || "Default Step",
        nodeId: base.nodeId || "default_node",
        order: base.order !== undefined ? base.order : 0,
        resourceInId: base.resourceInId || validIds.id2,
        runConnect: base.runConnect || validIds.id3,
    }),
    update: (base: RunStepUpdateInput): RunStepUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const runStepTestDataFactory = new TypedTestDataFactory(runStepFixtures, runRoutineStepValidation, customizers);
export const typedRunStepFixtures = createTypedFixtures(runStepFixtures, runRoutineStepValidation);

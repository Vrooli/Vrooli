import { type ModelTestFixtures, TestDataFactory } from "../../../validation/models/__test/validationTestUtils.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
};

// Shared reminder item test fixtures
export const reminderItemFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            name: "Complete task",
            reminderConnect: validIds.id2,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            name: "Review pull request",
            description: "Review and approve the latest pull request for the authentication module",
            dueDate: new Date("2024-12-31T15:00:00Z"),
            index: 0,
            isComplete: false,
            reminderConnect: validIds.id2,
        },
        update: {
            id: validIds.id1,
            name: "Updated task name",
            description: "Updated description with more details",
            dueDate: new Date("2025-01-15T10:00:00Z"),
            index: 1,
            isComplete: true,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                id: validIds.id1,
                // Missing required name and reminderConnect
                description: "Task without a name",
            },
            update: {
                // Missing required id
                name: "Updated task",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                name: true, // Should be string
                description: 123, // Should be string
                dueDate: "not-a-date", // Should be Date
                index: "zero", // Should be number
                isComplete: "yes", // Should be boolean
                reminderConnect: validIds.id2,
            },
            update: {
                id: validIds.id1,
                name: 123, // Should be string
                isComplete: 1, // Should be boolean
                index: -1, // Should be non-negative
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                name: "Invalid ID task",
                reminderConnect: validIds.id2,
            },
            update: {
                id: "invalid-id",
            },
        },
        missingReminderConnect: {
            create: {
                id: validIds.id1,
                name: "Task without reminder connection",
                // Missing required reminderConnect
            },
        },
        nameTooShort: {
            create: {
                id: validIds.id1,
                name: "", // Too short (min 1 char)
                reminderConnect: validIds.id2,
            },
        },
        nameTooLong: {
            create: {
                id: validIds.id1,
                name: "A".repeat(51), // Too long (max 50 chars)
                reminderConnect: validIds.id2,
            },
        },
        descriptionTooLong: {
            create: {
                id: validIds.id1,
                name: "Valid name",
                description: "A".repeat(2049), // Too long (max 2048 chars)
                reminderConnect: validIds.id2,
            },
        },
        negativeIndex: {
            create: {
                id: validIds.id1,
                name: "Task with negative index",
                index: -1, // Should be non-negative
                reminderConnect: validIds.id2,
            },
        },
    },
    edgeCases: {
        emptyDescription: {
            create: {
                id: validIds.id1,
                name: "Task with empty description",
                description: "",
                reminderConnect: validIds.id2,
            },
            update: {
                id: validIds.id1,
                description: "",
            },
        },
        whitespaceStrings: {
            create: {
                id: validIds.id1,
                name: "   Trimmed task name   ", // Should be trimmed
                description: "   Trimmed description   ", // Should be trimmed
                reminderConnect: validIds.id2,
            },
        },
        minimalName: {
            create: {
                id: validIds.id1,
                name: "A", // Minimum valid name (1 char)
                reminderConnect: validIds.id2,
            },
        },
        maxLengthName: {
            create: {
                id: validIds.id1,
                name: "A".repeat(50), // Maximum valid name (50 chars)
                reminderConnect: validIds.id2,
            },
        },
        maxLengthDescription: {
            create: {
                id: validIds.id1,
                name: "Valid task",
                description: "A".repeat(2048), // Maximum valid description (2048 chars)
                reminderConnect: validIds.id2,
            },
        },
        zeroIndex: {
            create: {
                id: validIds.id1,
                name: "First task",
                index: 0, // Valid minimum index
                reminderConnect: validIds.id2,
            },
        },
        largeIndex: {
            create: {
                id: validIds.id1,
                name: "Task with large index",
                index: 999999,
                reminderConnect: validIds.id2,
            },
        },
        pastDueDate: {
            create: {
                id: validIds.id1,
                name: "Overdue task",
                dueDate: new Date("2020-01-01T00:00:00Z"),
                reminderConnect: validIds.id2,
            },
        },
        futureDueDate: {
            create: {
                id: validIds.id1,
                name: "Future task",
                dueDate: new Date("2030-12-31T23:59:59Z"),
                reminderConnect: validIds.id2,
            },
        },
        completedTask: {
            create: {
                id: validIds.id1,
                name: "Already completed task",
                isComplete: true,
                reminderConnect: validIds.id2,
            },
            update: {
                id: validIds.id1,
                isComplete: false, // Marking as incomplete
            },
        },
        incompleteTask: {
            create: {
                id: validIds.id1,
                name: "Incomplete task",
                isComplete: false,
                reminderConnect: validIds.id2,
            },
            update: {
                id: validIds.id1,
                isComplete: true, // Marking as complete
            },
        },
        booleanConversions: {
            create: {
                id: validIds.id1,
                name: "Task with boolean string",
                isComplete: "true", // String to boolean
                reminderConnect: validIds.id2,
            },
            update: {
                id: validIds.id1,
                isComplete: "false", // String to boolean
            },
        },
        floatIndex: {
            create: {
                id: validIds.id1,
                name: "Task with float index",
                index: 3.14, // Should be converted to integer or rejected
                reminderConnect: validIds.id2,
            },
        },
        differentReminderId: {
            create: {
                id: validIds.id1,
                name: "Task for different reminder",
                reminderConnect: validIds.id5,
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
        name: base.name || "Default task name",
        reminderConnect: base.reminderConnect || validIds.id2,
    }),
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const reminderItemTestDataFactory = new TestDataFactory(reminderItemFixtures, customizers);

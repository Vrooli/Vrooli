import { type ModelTestFixtures, TestDataFactory } from "../../../validation/models/__test/validationTestUtils.js";

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

// Shared reminder list test fixtures
export const reminderListFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            remindersCreate: [
                {
                    id: validIds.id2,
                    name: "First reminder",
                    description: "Description of first reminder",
                    index: 0,
                },
                {
                    id: validIds.id3,
                    name: "Second reminder",
                    dueDate: new Date("2025-12-31T17:00:00Z"),
                    index: 1,
                },
                {
                    id: validIds.id4,
                    name: "Third reminder",
                    reminderItemsCreate: [
                        {
                            id: validIds.id5,
                            name: "Subtask 1",
                            reminderConnect: validIds.id4,
                        },
                        {
                            id: validIds.id6,
                            name: "Subtask 2",
                            isComplete: true,
                            reminderConnect: validIds.id4,
                        },
                    ],
                    index: 2,
                },
            ],
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 123,
            unknownField3: true,
        },
        update: {
            id: validIds.id1,
            remindersCreate: [{
                id: validIds.id7,
                name: "New reminder in update",
            }],
            remindersUpdate: [{
                id: validIds.id2,
                name: "Updated first reminder",
                description: "Updated description",
            }],
            remindersDelete: [validIds.id3],
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 456,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id
                remindersCreate: [{
                    id: validIds.id2,
                    name: "Reminder without list ID",
                }],
            },
            update: {
                // Missing required id
                remindersUpdate: [{
                    id: validIds.id2,
                    name: "Updated reminder",
                }],
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                remindersCreate: "not-an-array", // Should be array
            },
            update: {
                id: validIds.id1,
                remindersUpdate: "not-an-array", // Should be array
                remindersDelete: 123, // Should be array of strings
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
            },
            update: {
                id: "invalid-id",
            },
        },
        invalidReminders: {
            create: {
                id: validIds.id1,
                remindersCreate: [{
                    id: validIds.id2,
                    // Missing required name
                    description: "Reminder without name",
                }],
            },
            update: {
                id: validIds.id1,
                remindersCreate: [{
                    // Missing required id for reminder
                    name: "Reminder without ID",
                }],
            },
        },
        invalidReminderDelete: {
            update: {
                id: validIds.id1,
                remindersDelete: ["not-a-valid-id", 123], // Should be valid IDs
            },
        },
    },
    edgeCases: {
        emptyList: {
            create: {
                id: validIds.id1,
                // No reminders - valid as they're optional
            },
            update: {
                id: validIds.id1,
                // No operations - valid
            },
        },
        singleReminder: {
            create: {
                id: validIds.id1,
                remindersCreate: [{
                    id: validIds.id2,
                    name: "Only reminder in list",
                }],
            },
        },
        manyReminders: {
            create: {
                id: validIds.id1,
                remindersCreate: Array.from({ length: 10 }, (_, i) => ({
                    id: `12345678901234567${i}`,
                    name: `Reminder ${i + 1}`,
                    index: i,
                })),
            },
        },
        complexUpdate: {
            update: {
                id: validIds.id1,
                remindersCreate: [
                    {
                        id: validIds.id2,
                        name: "New reminder 1",
                    },
                    {
                        id: validIds.id3,
                        name: "New reminder 2",
                    },
                ],
                remindersUpdate: [
                    {
                        id: validIds.id4,
                        name: "Updated reminder 1",
                    },
                    {
                        id: validIds.id5,
                        name: "Updated reminder 2",
                        isComplete: true,
                    },
                ],
                remindersDelete: [validIds.id6, validIds.id7],
            },
        },
        nestedReminderItems: {
            create: {
                id: validIds.id1,
                remindersCreate: [{
                    id: validIds.id2,
                    name: "Reminder with many items",
                    reminderItemsCreate: Array.from({ length: 5 }, (_, i) => ({
                        id: `98765432109876543${i}`,
                        name: `Task ${i + 1}`,
                        index: i,
                        reminderConnect: validIds.id2,
                    })),
                }],
            },
        },
        circularListReference: {
            create: {
                id: validIds.id1,
                remindersCreate: [{
                    id: validIds.id2,
                    name: "Reminder in same list",
                    reminderListConnect: validIds.id1, // References the list being created
                }],
            },
        },
        reminderWithAllFields: {
            create: {
                id: validIds.id1,
                remindersCreate: [{
                    id: validIds.id2,
                    name: "Complete reminder",
                    description: "A reminder with all possible fields",
                    dueDate: new Date("2025-06-15T14:30:00Z"),
                    index: 0,
                    reminderListConnect: validIds.id1,
                    reminderItemsCreate: [{
                        id: validIds.id3,
                        name: "Complete subtask",
                        description: "Subtask with all fields",
                        dueDate: new Date("2025-06-14T14:30:00Z"),
                        index: 0,
                        isComplete: false,
                        reminderConnect: validIds.id2,
                    }],
                }],
            },
        },
        updateOnlyOperations: {
            update: {
                id: validIds.id1,
                // Only update operations, no create or delete
                remindersUpdate: [
                    {
                        id: validIds.id2,
                        name: "Only updating name",
                    },
                    {
                        id: validIds.id3,
                        description: "Only updating description",
                    },
                    {
                        id: validIds.id4,
                        dueDate: new Date("2025-07-01T00:00:00Z"),
                    },
                ],
            },
        },
        deleteOnlyOperations: {
            update: {
                id: validIds.id1,
                // Only delete operations
                remindersDelete: [validIds.id2, validIds.id3, validIds.id4],
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const reminderListTestDataFactory = new TestDataFactory(reminderListFixtures, customizers);

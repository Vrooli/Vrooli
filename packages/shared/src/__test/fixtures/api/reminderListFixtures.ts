import type { ReminderListCreateInput, ReminderListUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { reminderListValidation } from "../../../validation/models/reminderList.js";

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
export const reminderListFixtures: ModelTestFixtures<ReminderListCreateInput, ReminderListUpdateInput> = {
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
                            index: 0,
                            reminderConnect: validIds.id4,
                        },
                        {
                            id: validIds.id6,
                            name: "Subtask 2",
                            index: 1,
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
        } as ReminderListCreateInput,
        update: {
            id: validIds.id1,
            remindersCreate: [{
                id: validIds.id7,
                name: "New reminder in update",
                index: 0,
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
        } as ReminderListUpdateInput,
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id
                remindersCreate: [{
                    id: validIds.id2,
                    name: "Reminder without list ID",
                    index: 0,
                }],
            } as ReminderListCreateInput,
            update: {
                // Missing required id
                remindersUpdate: [{
                    id: validIds.id2,
                    name: "Updated reminder",
                }],
            } as ReminderListUpdateInput,
        },
        invalidTypes: {
            create: {
                // @ts-expect-error - Testing invalid type for id (number instead of string)
                id: 123, // Should be string
                // @ts-expect-error - Testing invalid type for remindersCreate (string instead of array)
                remindersCreate: "not-an-array", // Should be array
            } as unknown as ReminderListCreateInput,
            update: {
                id: validIds.id1,
                // @ts-expect-error - Testing invalid type for remindersUpdate (string instead of array)
                remindersUpdate: "not-an-array", // Should be array
                // @ts-expect-error - Testing invalid type for remindersDelete (number instead of array)
                remindersDelete: 123, // Should be array of strings
            } as unknown as ReminderListUpdateInput,
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
            } as ReminderListCreateInput,
            update: {
                id: "invalid-id",
            } as ReminderListUpdateInput,
        },
        invalidReminders: {
            create: {
                id: validIds.id1,
                remindersCreate: [{
                    id: validIds.id2,
                    // Missing required name
                    description: "Reminder without name",
                    index: 0,
                }],
            } as ReminderListCreateInput,
            update: {
                id: validIds.id1,
                remindersCreate: [{
                    // Missing required id for reminder
                    name: "Reminder without ID",
                    index: 0,
                }],
            } as ReminderListUpdateInput,
        },
        invalidReminderDelete: {
            update: {
                id: validIds.id1,
                // @ts-expect-error - Testing invalid type in array (number instead of string)
                remindersDelete: ["not-a-valid-id", 123], // Should be valid IDs
            } as unknown as ReminderListUpdateInput,
        },
    },
    edgeCases: {
        emptyList: {
            create: {
                id: validIds.id1,
                // No reminders - valid as they're optional
            } as ReminderListCreateInput,
            update: {
                id: validIds.id1,
                // No operations - valid
            } as ReminderListUpdateInput,
        },
        singleReminder: {
            create: {
                id: validIds.id1,
                remindersCreate: [{
                    id: validIds.id2,
                    name: "Only reminder in list",
                    index: 0,
                }],
            } as ReminderListCreateInput,
        },
        manyReminders: {
            create: {
                id: validIds.id1,
                remindersCreate: Array.from({ length: 10 }, (_, i) => ({
                    id: `12345678901234567${i}`,
                    name: `Reminder ${i + 1}`,
                    index: i,
                })),
            } as ReminderListCreateInput,
        },
        complexUpdate: {
            update: {
                id: validIds.id1,
                remindersCreate: [
                    {
                        id: validIds.id2,
                        name: "New reminder 1",
                        index: 0,
                    },
                    {
                        id: validIds.id3,
                        name: "New reminder 2",
                        index: 1,
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
            } as ReminderListUpdateInput,
        },
        nestedReminderItems: {
            create: {
                id: validIds.id1,
                remindersCreate: [{
                    id: validIds.id2,
                    name: "Reminder with many items",
                    index: 0,
                    reminderItemsCreate: Array.from({ length: 5 }, (_, i) => ({
                        id: `98765432109876543${i}`,
                        name: `Task ${i + 1}`,
                        index: i,
                        reminderConnect: validIds.id2,
                    })),
                }],
            } as ReminderListCreateInput,
        },
        circularListReference: {
            create: {
                id: validIds.id1,
                remindersCreate: [{
                    id: validIds.id2,
                    name: "Reminder in same list",
                    index: 0,
                    reminderListConnect: validIds.id1, // References the list being created
                }],
            } as ReminderListCreateInput,
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
            } as ReminderListCreateInput,
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
            } as ReminderListUpdateInput,
        },
        deleteOnlyOperations: {
            update: {
                id: validIds.id1,
                // Only delete operations
                remindersDelete: [validIds.id2, validIds.id3, validIds.id4],
            } as ReminderListUpdateInput,
        },
    },
};

// Custom factory that always generates valid IDs
const customizers: {
    create: (base: ReminderListCreateInput) => ReminderListCreateInput;
    update: (base: ReminderListUpdateInput) => ReminderListUpdateInput;
} = {
    create: (base: ReminderListCreateInput): ReminderListCreateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
    update: (base: ReminderListUpdateInput): ReminderListUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const reminderListTestDataFactory = new TypedTestDataFactory(reminderListFixtures, reminderListValidation, customizers);

// Export typed fixtures with optional validation methods
export const typedReminderListFixtures = createTypedFixtures(reminderListFixtures, reminderListValidation);

// Maintain backward compatibility - keep the old factory export
export { reminderListTestDataFactory as reminderListFactory };

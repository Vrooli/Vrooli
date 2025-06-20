import type { ReminderCreateInput, ReminderUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { reminderValidation } from "../../../validation/models/reminder.js";

// Magic number constants for testing
const NAME_TOO_LONG_LENGTH = 51;
const DESCRIPTION_TOO_LONG_LENGTH = 2049;
const NAME_MAX_LENGTH = 50;
const DESCRIPTION_MAX_LENGTH = 2048;

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
    id6: "123456789012345683",
};

// Shared reminder test fixtures
export const reminderFixtures: ModelTestFixtures<ReminderCreateInput, ReminderUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            name: "Buy groceries",
            index: 0,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            name: "Complete project milestone",
            description: "Finish implementing the authentication module and write tests",
            dueDate: new Date("2024-12-31T17:00:00Z"),
            index: 0,
            reminderListConnect: validIds.id2,
            reminderItemsCreate: [
                {
                    id: validIds.id3,
                    name: "Write unit tests",
                    description: "Create comprehensive test coverage",
                    isComplete: false,
                    index: 0,
                    reminderConnect: validIds.id1,
                },
                {
                    id: validIds.id4,
                    name: "Update documentation",
                    isComplete: false,
                    index: 1,
                    reminderConnect: validIds.id1,
                },
            ],
        },
        update: {
            id: validIds.id1,
            name: "Updated project milestone",
            description: "Updated description with new requirements",
            dueDate: new Date("2025-01-15T17:00:00Z"),
            index: 1,
            reminderItemsCreate: [{
                id: validIds.id5,
                name: "New reminder item",
                index: 2,
                reminderConnect: validIds.id1,
            }],
            reminderItemsUpdate: [{
                id: validIds.id3,
                isComplete: true,
            }],
            reminderItemsDelete: [validIds.id4],
        },
    },
    invalid: {
        missingRequired: {
            create: {
                id: validIds.id1,
                // Missing required name
                description: "This reminder has no name",
            } as ReminderCreateInput,
            update: {
                // Missing required id
                name: "Updated reminder",
            } as ReminderUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                name: true, // Should be string
                description: 123, // Should be string
                dueDate: "not-a-date", // Should be Date
                index: "zero", // Should be number
            } as unknown as ReminderCreateInput,
            update: {
                id: validIds.id1,
                name: 123, // Should be string
                dueDate: "invalid-date", // Should be Date
                index: -1, // Should be non-negative
            } as unknown as ReminderUpdateInput,
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                name: "Invalid ID reminder",
            } as ReminderCreateInput,
            update: {
                id: "invalid-id",
            } as ReminderUpdateInput,
        },
        nameTooShort: {
            create: {
                id: validIds.id1,
                name: "", // Too short (min 1 char)
            } as ReminderCreateInput,
        },
        nameTooLong: {
            create: {
                id: validIds.id1,
                name: "A".repeat(NAME_TOO_LONG_LENGTH), // Too long (max 50 chars)
            } as ReminderCreateInput,
        },
        descriptionTooLong: {
            create: {
                id: validIds.id1,
                name: "Valid name",
                description: "A".repeat(DESCRIPTION_TOO_LONG_LENGTH), // Too long (max 2048 chars)
            } as ReminderCreateInput,
        },
        negativeIndex: {
            create: {
                id: validIds.id1,
                name: "Reminder with negative index",
                index: -1, // Should be non-negative
            } as ReminderCreateInput,
        },
        invalidReminderItem: {
            create: {
                id: validIds.id1,
                name: "Reminder with invalid item",
                index: 0,
                reminderItemsCreate: [{
                    id: validIds.id3,
                    // Missing required name, index, and reminderConnect
                    description: "Item without name",
                }],
            } as unknown as ReminderCreateInput,
        },
        conflictingListConnections: {
            create: {
                id: validIds.id1,
                name: "Conflicting connections",
                index: 0,
                reminderListConnect: validIds.id2,
                reminderListCreate: {
                    id: validIds.id3,
                }, // Can't have both connect and create
            } as ReminderCreateInput,
        },
    },
    edgeCases: {
        emptyDescription: {
            create: {
                id: validIds.id1,
                name: "Reminder with empty description",
                description: "",
                index: 0,
            },
            update: {
                id: validIds.id1,
                description: "",
            },
        },
        whitespaceStrings: {
            create: {
                id: validIds.id1,
                name: "   Trimmed name   ", // Should be trimmed
                description: "   Trimmed description   ", // Should be trimmed
                index: 0,
            },
        },
        minimalName: {
            create: {
                id: validIds.id1,
                name: "ABC", // Minimum valid name (3 chars)
                index: 0,
            },
        },
        maxLengthName: {
            create: {
                id: validIds.id1,
                name: "A".repeat(NAME_MAX_LENGTH), // Maximum valid name (50 chars)
                index: 0,
            },
        },
        maxLengthDescription: {
            create: {
                id: validIds.id1,
                name: "Valid reminder",
                description: "A".repeat(DESCRIPTION_MAX_LENGTH), // Maximum valid description (2048 chars)
                index: 0,
            },
        },
        zeroIndex: {
            create: {
                id: validIds.id1,
                name: "First reminder",
                index: 0, // Valid minimum index
            },
        },
        largeIndex: {
            create: {
                id: validIds.id1,
                name: "Reminder with large index",
                index: 999999,
            },
        },
        pastDueDate: {
            create: {
                id: validIds.id1,
                name: "Overdue reminder",
                dueDate: new Date("2020-01-01T00:00:00Z"),
                index: 0,
            },
        },
        futureDueDate: {
            create: {
                id: validIds.id1,
                name: "Future reminder",
                dueDate: new Date("2030-12-31T23:59:59Z"),
                index: 0,
            },
        },
        withReminderList: {
            create: {
                id: validIds.id1,
                name: "Reminder in a list",
                reminderListConnect: validIds.id2,
                index: 0,
            },
        },
        createReminderList: {
            create: {
                id: validIds.id1,
                name: "Reminder creating a list",
                index: 0,
                reminderListCreate: {
                    id: validIds.id2,
                    remindersCreate: [{
                        id: validIds.id3,
                        name: "Another reminder in same list",
                        index: 1,
                    }],
                },
            },
        },
        multipleReminderItems: {
            create: {
                id: validIds.id1,
                name: "Reminder with many items",
                reminderItemsCreate: [
                    {
                        id: validIds.id3,
                        name: "First item",
                        index: 0,
                        reminderConnect: validIds.id1,
                    },
                    {
                        id: validIds.id4,
                        name: "Second item",
                        index: 1,
                        reminderConnect: validIds.id1,
                    },
                    {
                        id: validIds.id5,
                        name: "Third item",
                        index: 2,
                        isComplete: true,
                        reminderConnect: validIds.id1,
                    },
                ],
            },
        },
        floatIndex: {
            create: {
                id: validIds.id1,
                name: "Reminder with float index",
                index: 3.14, // Should be converted to integer
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: ReminderCreateInput): ReminderCreateInput => ({
        ...base,
        id: base.id || validIds.id1,
        name: base.name || "Default reminder name",
        index: base.index !== undefined ? base.index : 0,
    }),
    update: (base: ReminderUpdateInput): ReminderUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export enhanced type-safe factories
export const reminderTestDataFactory = new TypedTestDataFactory(reminderFixtures, reminderValidation, customizers);

// Export type-safe fixtures with validation capabilities
export const typedReminderFixtures = createTypedFixtures(reminderFixtures, reminderValidation);

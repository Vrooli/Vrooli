import { type ModelTestFixtures, TestDataFactory } from "../../../validation/models/__test/validationTestUtils.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
};

// Shared chatParticipant test fixtures
export const chatParticipantFixtures: ModelTestFixtures = {
    minimal: {
        create: {}, // No create operation defined
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {}, // No create operation defined
        update: {
            id: validIds.id2,
            extraField1: "test1", // Will be stripped by omitFields
            extraField2: "test2", // Will be stripped by omitFields  
            extraField3: "test3", // Will be stripped by omitFields
        },
    },
    invalid: {
        missingRequired: {
            create: {}, // No create operation defined
            update: {
                // Missing required id
            },
        },
        invalidTypes: {
            create: {}, // No create operation defined
            update: {
                id: 123, // Should be string
            },
        },
        invalidId: {
            update: {
                id: "not-a-valid-snowflake",
            },
        },
        emptyId: {
            update: {
                id: "",
            },
        },
        nullId: {
            update: {
                id: null,
            },
        },
        undefinedId: {
            update: {
                id: undefined,
            },
        },
    },
    edgeCases: {
        maxLengthId: {
            update: {
                id: "999999999999999999", // Max valid snowflake ID
            },
        },
        minLengthId: {
            update: {
                id: "100000000000000000", // Min valid snowflake ID
            },
        },
        extraFields: {
            update: {
                id: validIds.id1,
                unknownField1: "should be stripped",
                unknownField2: 123,
                unknownField3: { nested: "object" },
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: any) => ({
        ...base,
        // No create operation
    }),
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const chatParticipantTestDataFactory = new TestDataFactory(chatParticipantFixtures, customizers);

import { type ModelTestFixtures, TestDataFactory } from "../../../validation/models/__test/validationTestUtils.js";

// Valid Snowflake IDs for testing
const validIds = {
    id1: "300000000000000001",
    id2: "300000000000000002",
    id3: "300000000000000003",
};

// Shared phone test fixtures
export const phoneFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            phoneNumber: "+1234567890",
        },
        update: {
            // Phone model doesn't support updates
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            phoneNumber: "+12025551234", // US format
        },
        update: {
            // Phone model doesn't support updates
            id: validIds.id2,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing both id and phoneNumber
            },
            update: {
                // Not applicable - no update operation
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                phoneNumber: 1234567890, // Should be string
            },
            update: {
                // Not applicable
            },
        },
        tooLongPhone: {
            create: {
                id: validIds.id1,
                phoneNumber: "+123456789012345678901", // Exceeds 16 char limit
            },
        },
    },
    edgeCases: {
        shortPhone: {
            create: {
                id: validIds.id1,
                phoneNumber: "+1", // Very short but valid format
            },
        },
        phoneWithSpaces: {
            create: {
                id: validIds.id1,
                phoneNumber: "+1 234 567 8900",
            },
        },
        phoneWithDashes: {
            create: {
                id: validIds.id1,
                phoneNumber: "+1-234-567-8900",
            },
        },
        phoneWithParentheses: {
            create: {
                id: validIds.id1,
                phoneNumber: "+1(234)567-8900",
            },
        },
        phoneWithDots: {
            create: {
                id: validIds.id1,
                phoneNumber: "+1.234.567.8900",
            },
        },
        internationalFormat: {
            create: {
                id: validIds.id1,
                phoneNumber: "+44 20 7946 0958", // UK format
            },
        },
        maxLengthPhone: {
            create: {
                id: validIds.id1,
                phoneNumber: "+12345678901234", // 16 characters exactly
            },
        },
        emptyString: {
            create: {
                id: validIds.id1,
                phoneNumber: "",
            },
        },
        whitespaceOnly: {
            create: {
                id: validIds.id1,
                phoneNumber: "   ",
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
export const phoneTestDataFactory = new TestDataFactory(phoneFixtures, customizers);

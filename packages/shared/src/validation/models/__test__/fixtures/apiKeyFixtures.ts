import { ModelTestFixtures, TestDataFactory, testValues } from "../validationTestUtils.js";

// Valid Snowflake IDs for testing
const validIds = {
    id1: "500000000000000001",
    id2: "500000000000000002",
    id3: "500000000000000003",
};

// Shared apiKey test fixtures
export const apiKeyFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            limitHard: "1000000",
            name: "Test API Key",
            stopAtLimit: true,
            absoluteMax: 100000,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            disabled: false,
            limitHard: "5000000",
            limitSoft: "4000000",
            name: "Production API Key",
            stopAtLimit: false,
            absoluteMax: 500000,
            permissions: JSON.stringify({ read: true, write: true, delete: false }),
        },
        update: {
            id: validIds.id2,
            disabled: true,
            limitHard: "6000000",
            limitSoft: "5000000",
            name: "Updated API Key",
            stopAtLimit: true,
            absoluteMax: 600000,
            permissions: JSON.stringify({ read: true, write: true, delete: true }),
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing all required fields: id, limitHard, name, stopAtLimit, absoluteMax
                disabled: false,
            },
            update: {
                // Missing id
                name: "Missing ID",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                limitHard: 1000, // Should be string
                name: 123, // Should be string
                stopAtLimit: "yes", // Should be boolean
                absoluteMax: "100000", // Should be number
            },
            update: {
                id: true, // Should be string
                disabled: "no", // Should be boolean
                limitHard: true, // Should be string
                limitSoft: false, // Should be string
                name: [], // Should be string
                stopAtLimit: 1, // Should be boolean (but will be converted)
                absoluteMax: "not a number", // Should be number
                permissions: { read: true }, // Should be string
            },
        },
        nameTooShort: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "ab", // Less than 3 characters
                stopAtLimit: true,
                absoluteMax: 100000,
            },
        },
        nameTooLong: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "a".repeat(51), // Exceeds 50 char limit
                stopAtLimit: true,
                absoluteMax: 100000,
            },
        },
        negativeLimitHard: {
            create: {
                id: validIds.id1,
                limitHard: "-1000", // Should be positive
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 100000,
            },
        },
        invalidLimitHard: {
            create: {
                id: validIds.id1,
                limitHard: "abc", // Not a valid integer
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 100000,
            },
        },
        negativeAbsoluteMax: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: -100, // Should be >= 0
            },
        },
        absoluteMaxTooLarge: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 1000001, // Exceeds 1000000
            },
        },
        permissionsTooLong: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 100000,
                permissions: "a".repeat(4097), // Exceeds 4096 char limit
            },
        },
    },
    edgeCases: {
        minLengthName: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "abc", // Exactly 3 characters
                stopAtLimit: true,
                absoluteMax: 100000,
            },
        },
        maxLengthName: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "a".repeat(50), // Exactly 50 characters
                stopAtLimit: true,
                absoluteMax: 100000,
            },
        },
        zeroLimitHard: {
            create: {
                id: validIds.id1,
                limitHard: "0", // Edge case: zero limit
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 100000,
            },
        },
        zeroAbsoluteMax: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 0, // Edge case: zero
            },
        },
        maxAbsoluteMax: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 1000000, // Edge case: maximum value
            },
        },
        emptyPermissions: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 100000,
                permissions: "", // Edge case: empty string
            },
        },
        maxLengthPermissions: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 100000,
                permissions: "a".repeat(4096), // Exactly 4096 characters
            },
        },
        bigIntAsString: {
            create: {
                id: validIds.id1,
                limitHard: "9223372036854775807", // Max safe bigint
                limitSoft: "9223372036854775806",
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 100000,
            },
        },
        floatAbsoluteMax: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 100.5, // Should be rounded to integer
            },
        },
        booleanStringStopAtLimit: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: "true", // String boolean, should be converted
                absoluteMax: 100000,
            },
        },
        updateOnlyName: {
            update: {
                id: validIds.id1,
                name: "Updated Name Only",
            },
        },
        updateOnlyDisabled: {
            update: {
                id: validIds.id1,
                disabled: true,
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
        limitHard: base.limitHard || "1000000",
        name: base.name || "Generated API Key",
        stopAtLimit: base.stopAtLimit !== undefined ? base.stopAtLimit : true,
        absoluteMax: base.absoluteMax !== undefined ? base.absoluteMax : 100000,
    }),
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const apiKeyTestDataFactory = new TestDataFactory(apiKeyFixtures, customizers);
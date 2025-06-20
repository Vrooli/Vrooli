import type { ApiKeyCreateInput, ApiKeyUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { apiKeyValidation } from "../../../validation/models/apiKey.js";
import { API_KEY_PERMISSIONS_MAX_LENGTH, NAME_MAX_LENGTH, TEST_FIELD_TOO_LONG_MULTIPLIER } from "../../../validation/utils/validationConstants.js";

// Valid Snowflake IDs for testing
const validIds = {
    id1: "500000000000000001",
    id2: "500000000000000002",
    id3: "500000000000000003",
};

// Shared apiKey test fixtures
export const apiKeyFixtures: ModelTestFixtures<ApiKeyCreateInput, ApiKeyUpdateInput> = {
    minimal: {
        create: {
            limitHard: "1000000",
            name: "Test API Key",
            stopAtLimit: true,
            permissions: "{}",
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            disabled: false,
            limitHard: "5000000",
            limitSoft: "4000000",
            name: "Production API Key",
            stopAtLimit: false,
            permissions: JSON.stringify({ read: true, write: true, delete: false }),
        },
        update: {
            id: validIds.id2,
            disabled: true,
            limitHard: "6000000",
            limitSoft: "5000000",
            name: "Updated API Key",
            stopAtLimit: true,
            permissions: JSON.stringify({ read: true, write: true, delete: true }),
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing all required fields: id, limitHard, name, stopAtLimit, permissions
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
                permissions: 123, // Should be string
            },
            update: {
                id: true, // Should be string
                disabled: "no", // Should be boolean
                limitHard: true, // Should be string
                limitSoft: false, // Should be string
                name: [], // Should be string
                stopAtLimit: 1, // Should be boolean (but will be converted)
                permissions: { read: true }, // Should be string
            },
        },
        nameTooShort: {
            create: {
                limitHard: "1000000",
                name: "ab", // Less than 3 characters
                stopAtLimit: true,
                permissions: "{}",
            },
        },
        nameTooLong: {
            create: {
                limitHard: "1000000",
                name: "a".repeat(NAME_MAX_LENGTH + TEST_FIELD_TOO_LONG_MULTIPLIER), // Exceeds max char limit
                stopAtLimit: true,
                permissions: "{}",
            },
        },
        negativeLimitHard: {
            create: {
                limitHard: "-1000", // Should be positive
                name: "Test API Key",
                stopAtLimit: true,
                permissions: "{}",
            },
        },
        invalidLimitHard: {
            create: {
                limitHard: "abc", // Not a valid integer
                name: "Test API Key",
                stopAtLimit: true,
                permissions: "{}",
            },
        },
        permissionsTooLong: {
            create: {
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                permissions: "a".repeat(4097), // Exceeds 4096 char limit
            },
        },
    },
    edgeCases: {
        minLengthName: {
            create: {
                limitHard: "1000000",
                name: "abc", // Exactly 3 characters
                stopAtLimit: true,
                permissions: "{}",
            },
        },
        maxLengthName: {
            create: {
                limitHard: "1000000",
                name: "a".repeat(50), // Exactly 50 characters
                stopAtLimit: true,
                permissions: "{}",
            },
        },
        zeroLimitHard: {
            create: {
                limitHard: "0", // Edge case: zero limit
                name: "Test API Key",
                stopAtLimit: true,
                permissions: "{}",
            },
        },
        emptyPermissions: {
            create: {
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                permissions: "", // Edge case: empty string
            },
        },
        maxLengthPermissions: {
            create: {
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                permissions: "a".repeat(API_KEY_PERMISSIONS_MAX_LENGTH), // Exactly at max length
            },
        },
        bigIntAsString: {
            create: {
                limitHard: "9223372036854775807", // Max safe bigint
                limitSoft: "9223372036854775806",
                name: "Test API Key",
                stopAtLimit: true,
                permissions: "{}",
            },
        },
        booleanStringStopAtLimit: {
            create: {
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: "true", // String boolean, should be converted
                permissions: "{}",
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
    create: (base: ApiKeyCreateInput): ApiKeyCreateInput => ({
        ...base,
        limitHard: base.limitHard || "1000000",
        name: base.name || "Generated API Key",
        stopAtLimit: base.stopAtLimit !== undefined ? base.stopAtLimit : true,
        permissions: base.permissions || "{}",
    }),
    update: (base: ApiKeyUpdateInput): ApiKeyUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const apiKeyTestDataFactory = new TypedTestDataFactory(apiKeyFixtures, apiKeyValidation, customizers);
export const typedApiKeyFixtures = createTypedFixtures(apiKeyFixtures, apiKeyValidation);

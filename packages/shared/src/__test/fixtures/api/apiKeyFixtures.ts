import type { ApiKeyCreateInput, ApiKeyUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { apiKeyValidation } from "../../../validation/models/apiKey.js";
import { API_KEY_PERMISSIONS_MAX_LENGTH, NAME_MAX_LENGTH, TEST_FIELD_TOO_LONG_MULTIPLIER } from "../../../validation/utils/validationConstants.js";

// Magic number constants for testing
const PERMISSIONS_TOO_LONG_LENGTH = 4097;
const MAX_LENGTH_NAME = 50;

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
            id: validIds.id1,
            limitHard: "1000000",
            name: "Test API Key",
            stopAtLimit: true,
            absoluteMax: 100000,
            permissions: "{}",
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
            } as ApiKeyCreateInput,
            update: {
                // Missing id
                name: "Missing ID",
            } as ApiKeyUpdateInput,
        },
        invalidTypes: {
            create: {
                // @ts-expect-error - Testing invalid types
                id: 123, // Should be string
                // @ts-expect-error - Testing invalid types
                limitHard: 1000, // Should be string
                // @ts-expect-error - Testing invalid types
                name: 123, // Should be string
                // @ts-expect-error - Testing invalid types
                stopAtLimit: "yes", // Should be boolean
                // @ts-expect-error - Testing invalid types
                permissions: 123, // Should be string
            } as unknown as ApiKeyCreateInput,
            update: {
                // @ts-expect-error - Testing invalid types
                id: true, // Should be string
                // @ts-expect-error - Testing invalid types
                disabled: "no", // Should be boolean
                // @ts-expect-error - Testing invalid types
                limitHard: true, // Should be string
                // @ts-expect-error - Testing invalid types
                limitSoft: false, // Should be string
                // @ts-expect-error - Testing invalid types
                name: [], // Should be string
                // @ts-expect-error - Testing invalid types
                stopAtLimit: 1, // Should be boolean (but will be converted)
                // @ts-expect-error - Testing invalid types
                permissions: { read: true }, // Should be string
            } as unknown as ApiKeyUpdateInput,
        },
        nameTooShort: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "ab", // Less than 3 characters
                stopAtLimit: true,
                absoluteMax: 100000,
                permissions: "{}",
            },
        },
        nameTooLong: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "a".repeat(NAME_MAX_LENGTH + TEST_FIELD_TOO_LONG_MULTIPLIER), // Exceeds max char limit
                stopAtLimit: true,
                absoluteMax: 100000,
                permissions: "{}",
            },
        },
        negativeLimitHard: {
            create: {
                id: validIds.id1,
                limitHard: "-1000", // Should be positive
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 100000,
                permissions: "{}",
            },
        },
        invalidLimitHard: {
            create: {
                id: validIds.id1,
                limitHard: "abc", // Not a valid integer
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 100000,
                permissions: "{}",
            },
        },
        permissionsTooLong: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 100000,
                permissions: "a".repeat(PERMISSIONS_TOO_LONG_LENGTH), // Exceeds 4096 char limit
            },
        },
        negativeAbsoluteMax: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: -1, // Negative value
                permissions: "{}",
            },
        },
        absoluteMaxTooLarge: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 1000001, // Over 1,000,000 limit
                permissions: "{}",
            },
        },
        floatAbsoluteMax: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 123.45, // Float value - should be rejected
                permissions: "{}",
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
                permissions: "{}",
            },
        },
        maxLengthName: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "a".repeat(MAX_LENGTH_NAME), // Exactly 50 characters
                stopAtLimit: true,
                absoluteMax: 100000,
                permissions: "{}",
            },
        },
        zeroLimitHard: {
            create: {
                id: validIds.id1,
                limitHard: "0", // Edge case: zero limit
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 100000,
                permissions: "{}",
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
                permissions: "a".repeat(API_KEY_PERMISSIONS_MAX_LENGTH), // Exactly at max length
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
                permissions: "{}",
            },
        },
        booleanStringStopAtLimit: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                // @ts-expect-error - Testing string boolean conversion
                stopAtLimit: "true", // String boolean, should be converted
                absoluteMax: 100000,
                permissions: "{}",
            } as unknown as ApiKeyCreateInput,
        },
        zeroAbsoluteMax: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 0, // Edge case: zero absolute max
                permissions: "{}",
            },
        },
        maxAbsoluteMax: {
            create: {
                id: validIds.id1,
                limitHard: "1000000",
                name: "Test API Key",
                stopAtLimit: true,
                absoluteMax: 1000000, // Edge case: maximum absolute max
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
const customizers: {
    create: (base: Partial<ApiKeyCreateInput>) => ApiKeyCreateInput;
    update: (base: Partial<ApiKeyUpdateInput>) => ApiKeyUpdateInput;
} = {
    create: (base: Partial<ApiKeyCreateInput>): ApiKeyCreateInput => ({
        id: validIds.id1,
        limitHard: "1000000",
        name: "Generated API Key",
        stopAtLimit: true,
        absoluteMax: 100000,
        permissions: "{}",
        ...base,
    }),
    update: (base: Partial<ApiKeyUpdateInput>): ApiKeyUpdateInput => ({
        id: validIds.id1,
        ...base,
    }),
};

// Export a factory for creating test data programmatically
export const apiKeyTestDataFactory = new TypedTestDataFactory(apiKeyFixtures, apiKeyValidation, customizers);
export const typedApiKeyFixtures = createTypedFixtures(apiKeyFixtures, apiKeyValidation);

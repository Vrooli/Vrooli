import type { ApiKeyExternalCreateInput, ApiKeyExternalUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures, testValues } from "../../../validation/models/__test/validationTestUtils.js";
import { apiKeyExternalValidation } from "../../../validation/models/apiKeyExternal.js";
import { API_KEY_EXTERNAL_MAX_LENGTH, API_KEY_SERVICE_MAX_LENGTH, NAME_MAX_LENGTH } from "../../../validation/utils/validationConstants.js";

// Extend create type to include id (required by validation but not API)
type ApiKeyExternalCreateInputWithId = ApiKeyExternalCreateInput & { id: string };

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
};

// Shared apiKeyExternal test fixtures
export const apiKeyExternalFixtures: ModelTestFixtures<ApiKeyExternalCreateInputWithId, ApiKeyExternalUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            key: "sk-test-1234567890abcdef",
            name: "Test API Key",
            service: "OpenAI",
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            disabled: false,
            key: "sk-test-complete-1234567890abcdef1234567890abcdef",
            name: "Complete Test API Key",
            service: "Anthropic",
        },
        update: {
            id: validIds.id2,
            disabled: true,
            key: "sk-test-updated-1234567890abcdef",
            name: "Updated API Key",
            service: "Google",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id, key, name, service
                disabled: true,
            },
            update: {
                // Missing id
                name: "Missing ID",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                disabled: "yes", // Should be boolean
                key: true, // Should be string
                name: null, // Should be string
                service: 456, // Should be string
            },
            update: {
                id: validIds.id3,
                disabled: "no", // Should be boolean
                key: {}, // Should be string
                name: [], // Should be string
                service: false, // Should be string
            },
        },
        tooLongKey: {
            create: {
                id: validIds.id1,
                key: testValues.longString(300), // Exceeds max length (255)
                name: "Valid Name",
                service: "OpenAI",
            },
        },
        tooLongService: {
            create: {
                id: validIds.id1,
                key: "sk-valid-key",
                name: "Valid Name",
                service: testValues.longString(150), // Exceeds max length (128)
            },
        },
        tooLongName: {
            create: {
                id: validIds.id1,
                key: "sk-valid-key",
                name: testValues.longString(100), // Exceeds max length (50)
                service: "OpenAI",
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                key: "sk-valid-key",
                name: "Valid Name",
                service: "OpenAI",
            },
        },
    },
    edgeCases: {
        emptyStrings: {
            create: {
                id: validIds.id1,
                key: "",
                name: "",
                service: "",
            },
        },
        whitespaceStrings: {
            create: {
                id: validIds.id1,
                key: "   ",
                name: "   ",
                service: "   ",
            },
        },
        maxLengthStrings: {
            create: {
                id: validIds.id1,
                key: "a".repeat(API_KEY_EXTERNAL_MAX_LENGTH), // Exactly at max length
                name: "a".repeat(NAME_MAX_LENGTH), // At name max length
                service: "a".repeat(API_KEY_SERVICE_MAX_LENGTH), // Exactly at max length
            },
        },
        differentServices: {
            create: {
                id: validIds.id1,
                key: "test-key-123",
                name: "Multi Service Key",
                service: "Mistral",
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: Partial<ApiKeyExternalCreateInputWithId>): ApiKeyExternalCreateInputWithId => ({
        id: validIds.id1,
        key: "sk-default-key",
        name: "Default API Key",
        service: "OpenAI",
        ...base,
    }),
    update: (base: Partial<ApiKeyExternalUpdateInput>): ApiKeyExternalUpdateInput => ({
        id: validIds.id1,
        ...base,
    }),
};

// Export a factory for creating test data programmatically
export const apiKeyExternalTestDataFactory = new TypedTestDataFactory(apiKeyExternalFixtures, apiKeyExternalValidation, customizers);
export const typedApiKeyExternalFixtures = createTypedFixtures(apiKeyExternalFixtures, apiKeyExternalValidation);

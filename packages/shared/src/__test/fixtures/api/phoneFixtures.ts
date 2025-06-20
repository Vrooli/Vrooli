import type { PhoneCreateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { phoneValidation } from "../../../validation/models/phone.js";

// Valid Snowflake IDs for testing
const validIds = {
    id1: "300000000000000001",
    id2: "300000000000000002",
    id3: "300000000000000003",
};

// Shared phone test fixtures
// Phone model only supports create operations (no updates allowed)
// Using empty object type for update since Phone doesn't support updates
export const phoneFixtures: ModelTestFixtures<PhoneCreateInput, {}> = {
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
    create: (base: PhoneCreateInput): PhoneCreateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
    update: (base: {}): {} => ({
        ...base,
    }),
};

// Export a factory for creating test data programmatically
export const phoneTestDataFactory = new TestDataFactory(phoneFixtures, customizers);

// Export typed factory and fixtures for better type safety
export const typedPhoneTestDataFactory = new TypedTestDataFactory(phoneFixtures, phoneValidation, customizers);
export const typedPhoneFixtures = createTypedFixtures(phoneFixtures, phoneValidation);

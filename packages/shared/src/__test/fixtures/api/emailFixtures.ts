import type { EmailCreateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { emailValidation } from "../../../validation/models/email.js";

// Valid Snowflake IDs for testing
const validIds = {
    id1: "200000000000000001",
    id2: "200000000000000002",
    id3: "200000000000000003",
};

// Shared email test fixtures (Email has no update operation)
export const emailFixtures: ModelTestFixtures<EmailCreateInput, never> = {
    minimal: {
        create: {
            id: validIds.id1,
            emailAddress: "test@example.com",
        },
        update: null, // Email model doesn't support updates
    },
    complete: {
        create: {
            id: validIds.id2,
            emailAddress: "user.name+tag@example.co.uk",
        },
        update: {
            // Email model doesn't support updates
            id: validIds.id2,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing both id and emailAddress
            },
            update: null, // Not applicable - no update operation
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                emailAddress: true, // Should be string
            },
            update: null, // Not applicable
        },
        invalidEmail: {
            create: {
                id: validIds.id1,
                emailAddress: "not-an-email",
            },
        },
        tooLongEmail: {
            create: {
                id: validIds.id1,
                emailAddress: `${"a".repeat(250)}@example.com`, // Exceeds 256 char limit
            },
        },
    },
    edgeCases: {
        minimalEmail: {
            create: {
                id: validIds.id1,
                emailAddress: "a@b.c", // Shortest valid email
            },
        },
        emailWithPlus: {
            create: {
                id: validIds.id1,
                emailAddress: "test+filter@gmail.com",
            },
        },
        emailWithDots: {
            create: {
                id: validIds.id1,
                emailAddress: "first.last@company.com",
            },
        },
        emailWithNumbers: {
            create: {
                id: validIds.id1,
                emailAddress: "user123@example456.com",
            },
        },
        emailWithHyphens: {
            create: {
                id: validIds.id1,
                emailAddress: "test@my-company.com",
            },
        },
        emptyString: {
            create: {
                id: validIds.id1,
                emailAddress: "",
            },
        },
        whitespaceOnly: {
            create: {
                id: validIds.id1,
                emailAddress: "   ",
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: Partial<EmailCreateInput>): EmailCreateInput => ({
        id: validIds.id1,
        emailAddress: "default@example.com",
        ...base,
    }),
    update: (_base: any): never => null as never, // No update operation
};

// Export a factory for creating test data programmatically
export const emailTestDataFactory = new TypedTestDataFactory(emailFixtures, emailValidation, customizers);
export const typedEmailFixtures = createTypedFixtures(emailFixtures, emailValidation);

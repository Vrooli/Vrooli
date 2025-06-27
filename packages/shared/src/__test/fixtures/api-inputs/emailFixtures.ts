import type { EmailCreateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { emailValidation } from "../../../validation/models/email.js";

// Magic number constants for testing
const EMAIL_TOO_LONG_LENGTH = 250;

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
            } as EmailCreateInput,
            update: null, // Not applicable - no update operation
        },
        invalidTypes: {
            // @ts-expect-error - Testing invalid types
            create: {
                id: 123, // Should be string
                emailAddress: true, // Should be string
            } as unknown as EmailCreateInput,
            update: null, // Not applicable
        },
        invalidEmail: {
            // @ts-expect-error - Testing invalid email format
            create: {
                id: validIds.id1,
                emailAddress: "not-an-email",
            },
        },
        tooLongEmail: {
            // @ts-expect-error - Testing email length validation
            create: {
                id: validIds.id1,
                emailAddress: `${"a".repeat(EMAIL_TOO_LONG_LENGTH)}@example.com`, // Exceeds 256 char limit
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
            // @ts-expect-error - Testing empty email validation
            create: {
                id: validIds.id1,
                emailAddress: "",
            },
        },
        whitespaceOnly: {
            // @ts-expect-error - Testing whitespace-only email validation
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
    update: (_base: unknown): never => null as never, // No update operation
};

// Export a factory for creating test data programmatically
export const emailTestDataFactory = new TypedTestDataFactory(emailFixtures, emailValidation, customizers);
export const typedEmailFixtures = createTypedFixtures(emailFixtures, emailValidation);

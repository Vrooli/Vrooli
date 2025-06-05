import { ModelTestFixtures, TestDataFactory, testValues } from "../validationTestUtils.js";

// Valid Snowflake IDs for testing
const validIds = {
    id1: "200000000000000001",
    id2: "200000000000000002",
    id3: "200000000000000003",
};

// Shared email test fixtures
export const emailFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            emailAddress: "test@example.com",
        },
        update: {
            // Email model doesn't support updates
            id: validIds.id1,
        },
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
            update: {
                // Not applicable - no update operation
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                emailAddress: true, // Should be string
            },
            update: {
                // Not applicable
            },
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
export const emailTestDataFactory = new TestDataFactory(emailFixtures, customizers);
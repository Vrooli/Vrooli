import { type ModelTestFixtures, TestDataFactory, testValues } from "../validationTestUtils.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
};

// Shared tag test fixtures that can be imported by API tests, UI tests, etc.
export const tagFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            tag: "javascript",
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            tag: "javascript",
            anonymous: true,
            translations: {
                create: [{
                    language: "en",
                    description: "JavaScript programming language",
                }],
            },
        },
        update: {
            id: validIds.id2,
            tag: "typescript",
            anonymous: false,
            translations: {
                create: [{
                    language: "es",
                    description: "Lenguaje de programación TypeScript",
                }],
                update: [{
                    id: validIds.id3,
                    language: "en",
                    description: "TypeScript programming language",
                }],
                delete: [validIds.id4],
            },
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing both id and tag
                anonymous: true,
            },
            update: {
                // Missing id
                tag: "python",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                tag: true, // Should be string
            },
            update: {
                id: validIds.id5,
                anonymous: "yes", // Should be boolean
            },
        },
        tooLongTag: {
            create: {
                id: validIds.id1,
                tag: testValues.longString(150), // Exceeds max length
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                tag: "valid-tag",
            },
        },
    },
    edgeCases: {
        emptyTag: {
            create: {
                id: validIds.id1,
                tag: "",
            },
        },
        whitespaceTag: {
            create: {
                id: validIds.id1,
                tag: "   ",
            },
        },
        specialCharacters: {
            create: {
                id: validIds.id1,
                tag: "c++",
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
export const tagTestDataFactory = new TestDataFactory(tagFixtures, customizers);

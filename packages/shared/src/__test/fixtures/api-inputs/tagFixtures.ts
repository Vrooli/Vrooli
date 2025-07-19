import type { TagCreateInput, TagUpdateInput, TagTranslationCreateInput, TagTranslationUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures, testValues } from "../../../validation/models/__test/validationTestUtils.js";
import { tagValidation } from "../../../validation/models/tag.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
};

// Magic number constants for testing
const TAG_MAX_LENGTH_EXCEEDED = 150;
const TRANSLATION_DESCRIPTION_MAX_LENGTH_EXCEEDED = 1000;

// Shared tag test fixtures that can be imported by API tests, UI tests, etc.
export const tagFixtures: ModelTestFixtures<TagCreateInput, TagUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            tag: "javascript",
        },
        update: {
            id: validIds.id1,
            tag: "javascript",
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            tag: "javascript",
            anonymous: true,
            translationsCreate: [{
                id: validIds.id3,
                language: "en",
                description: "JavaScript programming language",
            }],
        },
        update: {
            id: validIds.id2,
            tag: "typescript",
            anonymous: false,
            translationsCreate: [{
                id: validIds.id4,
                language: "es",
                description: "Lenguaje de programación TypeScript",
            }],
            translationsUpdate: [{
                id: validIds.id3,
                language: "en",
                description: "TypeScript programming language",
            }],
            translationsDelete: [validIds.id4],
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing both id and tag
                anonymous: true,
            } as TagCreateInput,
            update: {
                // Missing id - tag is required by TypeScript type
                tag: "python",
            } as TagUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                tag: true, // Should be string
            } as unknown as TagCreateInput,
            update: {
                id: validIds.id5,
                anonymous: "yes", // Should be boolean
            } as unknown as TagUpdateInput,
        },
        tooLongTag: {
            create: {
                id: validIds.id1,
                tag: testValues.longString(TAG_MAX_LENGTH_EXCEEDED), // Exceeds max length
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

// Tag translation fixtures for testing translations separately
export const tagTranslationFixtures: ModelTestFixtures<TagTranslationCreateInput, TagTranslationUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            language: "en",
        },
        update: {
            id: validIds.id1,
            language: "en",
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            language: "en",
            description: "Complete description for testing",
        },
        update: {
            id: validIds.id2,
            language: "en",
            description: "Updated description for testing",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id and language
                description: "Description without required fields",
            } as TagTranslationCreateInput,
            update: {
                // Missing id and language
                description: "Updated description without required fields",
            } as TagTranslationUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                language: true, // Should be string
                description: [], // Should be string
            } as unknown as TagTranslationCreateInput,
            update: {
                id: false, // Should be string
                language: 456, // Should be string
                description: {}, // Should be string
            } as unknown as TagTranslationUpdateInput,
        },
    },
    edgeCases: {
        emptyDescription: {
            create: {
                id: validIds.id3,
                language: "en",
                description: "",
            },
        },
        longDescription: {
            create: {
                id: validIds.id4,
                language: "en",
                description: testValues.longString(TRANSLATION_DESCRIPTION_MAX_LENGTH_EXCEEDED),
            },
        },
        multipleLanguages: {
            create: {
                id: validIds.id5,
                language: "fr",
                description: "Description en français",
            },
        },
    },
};

// Custom factory that always generates valid IDs with proper typing
const customizers = {
    create: (base: TagCreateInput): TagCreateInput => ({
        ...base,
        id: base.id || testValues.snowflakeId(),
        tag: base.tag || testValues.shortString("tag"),
    }),
    update: (base: TagUpdateInput): TagUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export enhanced type-safe factories
export const tagTestDataFactory = new TypedTestDataFactory(tagFixtures, tagValidation, customizers);
export const tagTranslationTestDataFactory = new TestDataFactory(tagTranslationFixtures);

// Export type-safe fixtures with validation capabilities
export const typedTagFixtures = createTypedFixtures(tagFixtures, tagValidation);

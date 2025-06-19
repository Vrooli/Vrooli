import { CommentFor } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, testValues } from "../../../validation/models/__test/validationTestUtils.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
    forId1: "123456789012345683",
    forId2: "123456789012345684",
    parentId1: "123456789012345685",
    parentId2: "123456789012345686",
    translationId1: "123456789012345687",
    translationId2: "123456789012345688",
};

// Shared comment test fixtures
export const commentFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            createdFor: CommentFor.Issue,
            forConnect: validIds.forId1,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            createdFor: CommentFor.ResourceVersion,
            forConnect: validIds.forId2,
            parentConnect: validIds.parentId1,
            translationsCreate: [{
                id: validIds.translationId1,
                language: "en",
                text: "This is a detailed comment about the resource version.",
            }],
        },
        update: {
            id: validIds.id2,
            translationsCreate: [{
                id: validIds.translationId2,
                language: "es",
                text: "Este es un comentario en español.",
            }],
            translationsUpdate: [{
                id: validIds.translationId1,
                language: "en",
                text: "Updated comment text in English.",
            }],
            translationsDelete: [validIds.translationId2],
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id, createdFor, and forConnect
                parentConnect: validIds.parentId1,
            },
            update: {
                // Missing id
                translationsCreate: [{
                    id: validIds.translationId1,
                    language: "en",
                    text: "Some text",
                }],
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                createdFor: "InvalidType", // Should be valid CommentFor enum
                forConnect: 456, // Should be string
            },
            update: {
                id: validIds.id3,
                translationsCreate: "not-an-array", // Should be array
            },
        },
        invalidCreatedFor: {
            create: {
                id: validIds.id1,
                createdFor: "NotAValidEnum",
                forConnect: validIds.forId1,
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                createdFor: CommentFor.Issue,
                forConnect: validIds.forId1,
            },
        },
        missingForConnection: {
            create: {
                id: validIds.id1,
                createdFor: CommentFor.Issue,
                // Missing required 'forConnect' relationship
            },
        },
        invalidTranslationText: {
            create: {
                id: validIds.id1,
                createdFor: CommentFor.Issue,
                forConnect: validIds.forId1,
                translationsCreate: [{
                    id: validIds.translationId1,
                    language: "en",
                    text: "", // Empty text should be invalid
                }],
            },
        },
        tooLongTranslationText: {
            create: {
                id: validIds.id1,
                createdFor: CommentFor.Issue,
                forConnect: validIds.forId1,
                translationsCreate: [{
                    id: validIds.translationId1,
                    language: "en",
                    text: testValues.longString(40000), // Exceeds max length (32768)
                }],
            },
        },
    },
    edgeCases: {
        allCreatedForTypes: [
            {
                id: validIds.id1,
                createdFor: CommentFor.Issue,
                forConnect: validIds.forId1,
            },
            {
                id: validIds.id2,
                createdFor: CommentFor.PullRequest,
                forConnect: validIds.forId1,
            },
            {
                id: validIds.id3,
                createdFor: CommentFor.ResourceVersion,
                forConnect: validIds.forId1,
            },
        ],
        withParentComment: {
            create: {
                id: validIds.id1,
                createdFor: CommentFor.Issue,
                forConnect: validIds.forId1,
                parentConnect: validIds.parentId1,
            },
        },
        multipleTranslations: {
            create: {
                id: validIds.id1,
                createdFor: CommentFor.Issue,
                forConnect: validIds.forId1,
                translationsCreate: [
                    {
                        id: validIds.translationId1,
                        language: "en",
                        text: "English comment text.",
                    },
                    {
                        id: validIds.translationId2,
                        language: "es",
                        text: "Texto del comentario en español.",
                    },
                ],
            },
        },
        maxLengthTranslationText: {
            create: {
                id: validIds.id1,
                createdFor: CommentFor.Issue,
                forConnect: validIds.forId1,
                translationsCreate: [{
                    id: validIds.translationId1,
                    language: "en",
                    text: "a".repeat(32768), // Exactly at max length
                }],
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
export const commentTestDataFactory = new TestDataFactory(commentFixtures, customizers);

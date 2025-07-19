import type { CommentCreateInput, CommentUpdateInput } from "../../../api/types.js";
import { CommentFor } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures, testValues } from "../../../validation/models/__test/validationTestUtils.js";
import { commentValidation } from "../../../validation/models/comment.js";

// Magic number constants for testing
const TEXT_TOO_LONG_LENGTH = 40000;
const TEXT_MAX_LENGTH = 32768;

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
export const commentFixtures: ModelTestFixtures<CommentCreateInput, CommentUpdateInput> = {
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
            } as CommentCreateInput,
            update: {
                // Missing id
                translationsCreate: [{
                    id: validIds.translationId1,
                    language: "en",
                    text: "Some text",
                }],
            } as CommentUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123,
                createdFor: "InvalidType",
                forConnect: 456,
            } as unknown as CommentCreateInput,
            update: {
                id: validIds.id3,
                translationsCreate: "not-an-array",
            } as unknown as CommentUpdateInput,
        },
        invalidCreatedFor: {
            create: {
                id: validIds.id1,
                createdFor: "NotAValidEnum",
                forConnect: validIds.forId1,
            } as unknown as CommentCreateInput,
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
            } as CommentCreateInput,
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
                    text: testValues.longString(TEXT_TOO_LONG_LENGTH), // Exceeds max length (32768)
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
                    text: "a".repeat(TEXT_MAX_LENGTH), // Exactly at max length
                }],
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: Partial<CommentCreateInput>): Partial<CommentCreateInput> => ({
        ...base,
        id: base.id || validIds.id1,
    }),
    update: (base: Partial<CommentUpdateInput>): Partial<CommentUpdateInput> => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const commentTestDataFactory = new TypedTestDataFactory(commentFixtures, commentValidation, customizers);
export const typedCommentFixtures = createTypedFixtures(commentFixtures, commentValidation);

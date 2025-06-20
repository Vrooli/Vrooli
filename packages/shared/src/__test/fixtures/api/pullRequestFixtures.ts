import type { PullRequestCreateInput, PullRequestUpdateInput, PullRequestTranslationCreateInput, PullRequestTranslationUpdateInput } from "../../../api/types.js";
import { PullRequestToObjectType, PullRequestFromObjectType, PullRequestStatus } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures, testValues } from "../../../validation/models/__test/validationTestUtils.js";
import { pullRequestValidation, pullRequestTranslationValidation } from "../../../validation/models/pullRequest.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
    id6: "123456789012345683",
};

// Shared pullRequest test fixtures
export const pullRequestFixtures: ModelTestFixtures<PullRequestCreateInput, PullRequestUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            toObjectType: PullRequestToObjectType.Resource,
            toConnect: validIds.id2,
            fromObjectType: PullRequestFromObjectType.ResourceVersion,
            fromConnect: validIds.id3,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            toObjectType: PullRequestToObjectType.Resource,
            toConnect: validIds.id2,
            fromObjectType: PullRequestFromObjectType.ResourceVersion,
            fromConnect: validIds.id3,
            translationsCreate: [
                {
                    id: validIds.id4,
                    language: "en",
                    text: "This pull request adds a new feature to improve user experience. It includes enhanced validation and better error handling.",
                },
                {
                    id: validIds.id5,
                    language: "es",
                    text: "Esta solicitud de extracción añade una nueva característica para mejorar la experiencia del usuario.",
                },
            ],
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 123,
            unknownField3: true,
        },
        update: {
            id: validIds.id1,
            status: PullRequestStatus.Open,
            translationsCreate: [
                {
                    id: validIds.id6,
                    language: "fr",
                    text: "Cette demande de tirage ajoute une nouvelle fonctionnalité.",
                },
            ],
            translationsUpdate: [
                {
                    id: validIds.id4,
                    language: "en",
                    text: "Updated pull request description with more details about the implementation.",
                },
            ],
            translationsDelete: [validIds.id5],
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 456,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id, toObjectType, toConnect, and fromConnect
                translationsCreate: [
                    {
                        id: validIds.id4,
                        language: "en",
                        text: "Incomplete pull request",
                    },
                ],
            },
            update: {
                // Missing required id
                status: PullRequestStatus.Draft,
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                toObjectType: "InvalidType", // Invalid enum value
                toConnect: 456, // Should be string
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: 789, // Should be string
                translationsCreate: [
                    {
                        id: validIds.id4,
                        language: "en",
                        text: "Invalid types test",
                    },
                ],
            },
            update: {
                id: validIds.id1,
                status: "InvalidStatus", // Invalid enum value
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                toObjectType: PullRequestToObjectType.Resource,
                toConnect: validIds.id2,
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: validIds.id3,
            },
            update: {
                id: "invalid-id",
            },
        },
        invalidToObjectType: {
            create: {
                id: validIds.id1,
                toObjectType: "UnknownType", // Not a valid enum value
                toConnect: validIds.id2,
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: validIds.id3,
            },
        },
        missingTo: {
            create: {
                id: validIds.id1,
                toObjectType: PullRequestToObjectType.Resource,
                // Missing required toConnect
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: validIds.id3,
            },
        },
        missingFrom: {
            create: {
                id: validIds.id1,
                toObjectType: PullRequestToObjectType.Resource,
                toConnect: validIds.id2,
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                // Missing required fromConnect
            },
        },
        invalidToConnect: {
            create: {
                id: validIds.id1,
                toObjectType: PullRequestToObjectType.Resource,
                toConnect: "invalid-to-id",
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: validIds.id3,
            },
        },
        invalidFromConnect: {
            create: {
                id: validIds.id1,
                toObjectType: PullRequestToObjectType.Resource,
                toConnect: validIds.id2,
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: "invalid-from-id",
            },
        },
        invalidTranslations: {
            create: {
                id: validIds.id1,
                toObjectType: PullRequestToObjectType.Resource,
                toConnect: validIds.id2,
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: validIds.id3,
                translationsCreate: [
                    {
                        id: validIds.id4,
                        language: "en",
                        text: "", // Empty text (should fail because it becomes undefined)
                    },
                ],
            },
        },
        longText: {
            create: {
                id: validIds.id1,
                toObjectType: PullRequestToObjectType.Resource,
                toConnect: validIds.id2,
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: validIds.id3,
                translationsCreate: [
                    {
                        id: validIds.id4,
                        language: "en",
                        text: "x".repeat(32769), // Exceeds max length
                    },
                ],
            },
        },
        emptyTextTranslation: {
            create: {
                id: validIds.id1,
                toObjectType: PullRequestToObjectType.Resource,
                toConnect: validIds.id2,
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: validIds.id3,
                translationsCreate: [
                    {
                        id: validIds.id4,
                        language: "en",
                        text: "", // Empty text becomes undefined, should fail as required
                    },
                ],
            },
        },
    },
    edgeCases: {
        withoutTranslations: {
            create: {
                id: validIds.id1,
                toObjectType: PullRequestToObjectType.Resource,
                toConnect: validIds.id2,
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: validIds.id3,
                // No translations
            },
        },
        withSingleTranslation: {
            create: {
                id: validIds.id1,
                toObjectType: PullRequestToObjectType.Resource,
                toConnect: validIds.id2,
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: validIds.id3,
                translationsCreate: [
                    {
                        id: validIds.id4,
                        language: "en",
                        text: "Single translation pull request.",
                    },
                ],
            },
        },
        withMultipleTranslations: {
            create: {
                id: validIds.id1,
                toObjectType: PullRequestToObjectType.Resource,
                toConnect: validIds.id2,
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: validIds.id3,
                translationsCreate: [
                    {
                        id: validIds.id4,
                        language: "en",
                        text: "English description of the pull request.",
                    },
                    {
                        id: validIds.id5,
                        language: "es",
                        text: "Descripción en español de la solicitud de extracción.",
                    },
                    {
                        id: validIds.id6,
                        language: "fr",
                        text: "Description française de la demande de tirage.",
                    },
                ],
            },
        },
        statusDraft: {
            update: {
                id: validIds.id1,
                status: PullRequestStatus.Draft,
            },
        },
        statusOpen: {
            update: {
                id: validIds.id1,
                status: PullRequestStatus.Open,
            },
        },
        statusMerged: {
            update: {
                id: validIds.id1,
                status: PullRequestStatus.Merged,
            },
        },
        statusRejected: {
            update: {
                id: validIds.id1,
                status: PullRequestStatus.Rejected,
            },
        },
        statusCanceled: {
            update: {
                id: validIds.id1,
                status: PullRequestStatus.Canceled,
            },
        },
        updateOnlyId: {
            update: {
                id: validIds.id1,
                // Only required field
            },
        },
        updateWithTranslationOperations: {
            update: {
                id: validIds.id1,
                status: PullRequestStatus.Open,
                translationsCreate: [
                    {
                        id: validIds.id4,
                        language: "de",
                        text: "Deutsche Beschreibung der Pull-Anfrage.",
                    },
                ],
                translationsUpdate: [
                    {
                        id: validIds.id5,
                        language: "en",
                        text: "Updated text for existing translation.",
                    },
                ],
                translationsDelete: [validIds.id6],
            },
        },
        differentIds: {
            create: {
                id: validIds.id6,
                toObjectType: PullRequestToObjectType.Resource,
                toConnect: validIds.id5,
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: validIds.id4,
            },
        },
        maxLengthText: {
            create: {
                id: validIds.id1,
                toObjectType: PullRequestToObjectType.Resource,
                toConnect: validIds.id2,
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: validIds.id3,
                translationsCreate: [
                    {
                        id: validIds.id4,
                        language: "en",
                        text: "x".repeat(32768), // Exactly max length
                    },
                ],
            },
        },
        minLengthText: {
            create: {
                id: validIds.id1,
                toObjectType: PullRequestToObjectType.Resource,
                toConnect: validIds.id2,
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: validIds.id3,
                translationsCreate: [
                    {
                        id: validIds.id4,
                        language: "en",
                        text: "x", // Exactly min length (1)
                    },
                ],
            },
        },
        longDescription: {
            create: {
                id: validIds.id1,
                toObjectType: PullRequestToObjectType.Resource,
                toConnect: validIds.id2,
                fromObjectType: PullRequestFromObjectType.ResourceVersion,
                fromConnect: validIds.id3,
                translationsCreate: [
                    {
                        id: validIds.id4,
                        language: "en",
                        text: "This is a comprehensive pull request that implements multiple features including enhanced validation, improved error handling, better user experience, optimized performance, and extensive documentation. The changes affect various components of the system and require thorough testing to ensure compatibility and functionality.",
                    },
                ],
            },
        },
    },
};

// Pull request translation fixtures
export const pullRequestTranslationFixtures: ModelTestFixtures<PullRequestTranslationCreateInput, PullRequestTranslationUpdateInput> = {
    minimal: {
        create: {
            id: "400000000000000001",
            language: "en",
            text: "Minimal pull request description",
        },
        update: {
            id: "400000000000000001",
            language: "en",
        },
    },
    complete: {
        create: {
            id: "400000000000000002", 
            language: "en",
            text: "A comprehensive pull request description with detailed information about the changes and implementation details",
        },
        update: {
            id: "400000000000000002",
            language: "en",
            text: "Updated pull request description with new information",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing language, id, and text
            },
            update: {
                // Missing id and language
                text: "Updated description without required fields",
            },
        },
        invalidTypes: {
            create: {
                id: "400000000000000003",
                language: 123, // Should be string
                text: true, // Should be string
            },
            update: {
                id: 456, // Should be string
                language: "en",
                text: [], // Should be string
            },
        },
    },
    edgeCases: {
        emptyText: {
            create: {
                id: "400000000000000004",
                language: "en",
                text: "",
            },
        },
        longText: {
            create: {
                id: "400000000000000005",
                language: "en",
                text: "x".repeat(32768), // Max length text
            },
        },
        multipleLanguages: {
            create: {
                id: "400000000000000006",
                language: "fr",
                text: "Description française de la demande de tirage",
            },
        },
        minimalText: {
            create: {
                id: "400000000000000007",
                language: "en",
                text: "x", // Minimum length (1 character)
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers = {
    create: (base: PullRequestCreateInput): PullRequestCreateInput => ({
        ...base,
        id: base.id || testValues.snowflakeId(),
        toObjectType: base.toObjectType || PullRequestToObjectType.Resource,
        toConnect: base.toConnect || testValues.snowflakeId(),
        fromObjectType: base.fromObjectType || PullRequestFromObjectType.ResourceVersion,
        fromConnect: base.fromConnect || testValues.snowflakeId(),
    }),
    update: (base: PullRequestUpdateInput): PullRequestUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export enhanced type-safe factories
export const pullRequestTestDataFactory = new TypedTestDataFactory(pullRequestFixtures, pullRequestValidation, customizers);
export const pullRequestTranslationTestDataFactory = new TestDataFactory(pullRequestTranslationFixtures);

// Export type-safe fixtures with validation capabilities
export const typedPullRequestFixtures = createTypedFixtures(pullRequestFixtures, pullRequestValidation);

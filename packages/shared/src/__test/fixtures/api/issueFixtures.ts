import { IssueFor, type IssueCreateInput, type IssueTranslationCreateInput, type IssueUpdateInput } from "../../../api/types.js";
import { TestDataFactory, TypedTestDataFactory, createTypedFixtures, type ModelTestFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { issueValidation } from "../../../validation/models/issue.js";

// Magic number constants for testing
const NAME_MAX_LENGTH = 50;
const DESCRIPTION_LONG_LENGTH = 1000;

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    forId1: "123456789012345682",
    forId2: "123456789012345683",
    translationId1: "123456789012345684",
    translationId2: "123456789012345685",
};

// Shared issue test fixtures
export const issueFixtures: ModelTestFixtures<IssueCreateInput, IssueUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            issueFor: IssueFor.Resource,
            forConnect: validIds.forId1,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            issueFor: IssueFor.Team,
            forConnect: validIds.forId2,
            translationsCreate: [
                {
                    id: validIds.translationId1,
                    language: "en",
                    name: "Bug in feature X",
                    description: "Detailed description of the bug",
                },
            ],
        },
        update: {
            id: validIds.id2,
            translationsUpdate: [
                {
                    id: validIds.translationId1,
                    language: "en",
                    name: "Updated bug title",
                    description: "Updated description",
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id, issueFor, and forConnect
                translationsCreate: [],
            } as IssueCreateInput,
            update: {
                // Missing id
                translationsCreate: [],
            } as IssueUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                issueFor: "InvalidType", // Should be valid IssueFor enum
                forConnect: 456, // Should be string
            } as unknown as IssueCreateInput,
            update: {
                id: validIds.id3,
                translationsCreate: "not-an-array", // Should be array
            } as unknown as IssueUpdateInput,
        },
        invalidIssueFor: {
            create: {
                id: validIds.id1,
                issueFor: "InvalidIssueType", // Should be valid IssueFor enum
                forConnect: validIds.forId1,
            } as unknown as IssueCreateInput,
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                issueFor: IssueFor.Resource,
                forConnect: validIds.forId1,
            },
        },
        invalidForConnect: {
            create: {
                id: validIds.id1,
                issueFor: IssueFor.Resource,
                forConnect: "not-a-valid-snowflake",
            },
        },
        missingForConnect: {
            create: {
                id: validIds.id1,
                issueFor: IssueFor.Resource,
                // Missing required forConnect
            } as IssueCreateInput,
        },
        invalidTranslationName: {
            create: {
                id: validIds.id1,
                issueFor: IssueFor.Resource,
                forConnect: validIds.forId1,
                translationsCreate: [
                    {
                        id: validIds.translationId1,
                        language: "en",
                        name: "", // Empty name should fail
                        description: "Valid description",
                    },
                ],
            },
        },
        missingTranslationLanguage: {
            create: {
                id: validIds.id1,
                issueFor: IssueFor.Resource,
                forConnect: validIds.forId1,
                translationsCreate: [
                    {
                        id: validIds.translationId1,
                        // Missing required language field (added by transRel)
                        name: "Valid name",
                        description: "Valid description",
                    } as IssueTranslationCreateInput,
                ],
            } as IssueCreateInput,
        },
    },
    edgeCases: {
        allIssueForTypes: [
            {
                id: validIds.id1,
                issueFor: IssueFor.Resource,
                forConnect: validIds.forId1,
            },
            {
                id: validIds.id2,
                issueFor: IssueFor.Team,
                forConnect: validIds.forId2,
            },
        ],
        translationWithoutDescription: {
            create: {
                id: validIds.id1,
                issueFor: IssueFor.Resource,
                forConnect: validIds.forId1,
                translationsCreate: [
                    {
                        id: validIds.translationId1,
                        language: "en",
                        name: "Issue without description",
                        // description is optional
                    },
                ],
            },
        },
        multipleTranslations: {
            create: {
                id: validIds.id1,
                issueFor: IssueFor.Resource,
                forConnect: validIds.forId1,
                translationsCreate: [
                    {
                        id: validIds.translationId1,
                        language: "en",
                        name: "English issue title",
                        description: "English description",
                    },
                    {
                        id: validIds.translationId2,
                        language: "es",
                        name: "Título del problema en español",
                        description: "Descripción en español",
                    },
                ],
            },
        },
        longNameAndDescription: {
            create: {
                id: validIds.id1,
                issueFor: IssueFor.Resource,
                forConnect: validIds.forId1,
                translationsCreate: [
                    {
                        id: validIds.translationId1,
                        language: "en",
                        name: "x".repeat(NAME_MAX_LENGTH), // Test long name
                        description: "x".repeat(DESCRIPTION_LONG_LENGTH), // Test long description
                    },
                ],
            },
        },
        updateWithAllTranslationOperations: {
            update: {
                id: validIds.id1,
                translationsCreate: [
                    {
                        id: validIds.translationId1,
                        language: "en",
                        name: "New translation",
                    },
                ],
                translationsUpdate: [
                    {
                        id: validIds.translationId2,
                        language: "en",
                        name: "Updated translation",
                    },
                ],
                translationsDelete: [validIds.translationId1],
            },
        },
        updateOptionalTranslationName: {
            update: {
                id: validIds.id1,
                translationsUpdate: [
                    {
                        id: validIds.translationId1,
                        language: "en",
                        // name is optional in update
                        description: "Updated description only",
                    },
                ],
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: Partial<IssueCreateInput>): IssueCreateInput => ({
        ...base,
        id: base.id || validIds.id1,
        issueFor: base.issueFor || IssueFor.Resource,
        forConnect: base.forConnect || validIds.forId1,
    } as IssueCreateInput),
    update: (base: Partial<IssueUpdateInput>): IssueUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    } as IssueUpdateInput),
};

// Export a factory for creating test data programmatically
export const issueTestDataFactory = new TypedTestDataFactory(issueFixtures, issueValidation, customizers);

// Export typed fixtures for direct use
export const typedIssueFixtures = createTypedFixtures(issueFixtures, issueValidation);

// For backward compatibility, also export a regular TestDataFactory if needed
// Note: The TypedTestDataFactory extends TestDataFactory, so this should work the same
export const issueTestDataFactoryLegacy = new TestDataFactory(issueFixtures, customizers);

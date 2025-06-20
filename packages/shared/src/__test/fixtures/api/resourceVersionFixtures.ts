import type { ResourceSubType, ResourceVersionCreateInput, ResourceVersionTranslationCreateInput, ResourceVersionTranslationUpdateInput, ResourceVersionUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { resourceVersionValidation } from "../../../validation/models/resourceVersion.js";

// Magic number constants for testing
const CODE_LANGUAGE_TOO_LONG_LENGTH = 129;
const VERSION_NOTES_TOO_LONG_LENGTH = 4093;
const NAME_MAX_LENGTH = 256;
const DESCRIPTION_MAX_LENGTH = 2048;
const DETAILS_MAX_LENGTH = 16384;
const INSTRUCTIONS_MAX_LENGTH = 8192;

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
    id6: "123456789012345683",
    id7: "123456789012345684",
    id8: "123456789012345685",
};

// Valid public IDs for testing (10-12 character alphanumeric)
const validPublicIds = {
    pub1: "abc1234567", // 10 chars
    pub2: "def6789012", // 10 chars  
    pub3: "ghi1357924", // 10 chars
    pub4: "jkl2468013", // 10 chars
};

// Shared resourceVersion test fixtures
export const resourceVersionFixtures: ModelTestFixtures<ResourceVersionCreateInput, ResourceVersionUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            versionLabel: "1.0.0",
            isPrivate: false,
            rootConnect: validIds.id2,
            translationsCreate: [
                {
                    id: validIds.id3,
                    language: "en",
                    name: "Minimal Resource Version",
                },
            ],
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            publicId: validPublicIds.pub1,
            codeLanguage: "typescript",
            config: { __version: "1.0", resources: [] },
            isAutomatable: true,
            isComplete: true,
            // isInternal: false, // Not part of ResourceVersionCreateInput
            isPrivate: false,
            resourceSubType: "RoutineApi" as ResourceSubType,
            versionLabel: "2.1.0",
            versionNotes: "Major update with new features and bug fixes.",
            rootConnect: validIds.id2,
            relatedVersionsCreate: [
                {
                    id: validIds.id4,
                    labels: ["dependency", "upgrade"],
                    toVersionConnect: validIds.id5,
                },
                {
                    id: validIds.id6,
                    labels: ["replaces"],
                    toVersionConnect: validIds.id7,
                },
            ],
            translationsCreate: [
                {
                    id: validIds.id3,
                    language: "en",
                    name: "Complete Resource Version",
                    description: "A comprehensive resource version with all features enabled.",
                    details: "This version includes advanced configuration options and automation capabilities.",
                    instructions: "Follow the setup guide carefully and ensure all dependencies are installed.",
                },
                {
                    id: validIds.id8,
                    language: "es",
                    name: "Versión Completa del Recurso",
                    description: "Una versión integral del recurso con todas las características habilitadas.",
                },
            ],
            // Add some extra fields that will be stripped
            // @ts-expect-error Testing unknown fields
            unknownField1: "should be stripped",
            unknownField2: 123,
            unknownField3: true,
        },
        update: {
            id: validIds.id1,
            codeLanguage: "javascript",
            config: { __version: "1.0", resources: [] },
            isAutomatable: false,
            isComplete: true,
            // isInternal: true, // Not part of ResourceVersionUpdateInput
            isPrivate: true,
            versionLabel: "2.1.1",
            versionNotes: "Patch release with minor fixes.",
            rootUpdate: {
                id: validIds.id2,
                isPrivate: true,
            },
            relatedVersionsCreate: [
                {
                    id: validIds.id4,
                    labels: ["hotfix"],
                    toVersionConnect: validIds.id5,
                },
            ],
            relatedVersionsUpdate: [
                {
                    id: validIds.id6,
                    labels: ["updated", "verified"],
                },
            ],
            relatedVersionsDisconnect: [validIds.id7],
            translationsCreate: [
                {
                    id: validIds.id3,
                    language: "fr",
                    name: "Version Mise à Jour",
                },
            ],
            translationsUpdate: [
                {
                    id: validIds.id8,
                    language: "en",
                    name: "Updated Complete Version",
                    description: "Updated comprehensive resource version.",
                },
            ],
            translationsDelete: [validIds.id4],
            // Add some extra fields that will be stripped
            // @ts-expect-error Testing unknown fields
            unknownField1: "should be stripped",
            unknownField2: 456,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id, versionLabel, and root connection
                codeLanguage: "python",
                isComplete: true,
            } as ResourceVersionCreateInput,
            update: {
                // Missing required id
                versionLabel: "1.0.1",
            } as ResourceVersionUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                publicId: 456, // Should be string
                codeLanguage: 789, // Should be string
                config: "not-an-object", // Should be object
                isAutomatable: "true", // Should be boolean
                isComplete: "false", // Should be boolean
                // isInternal: "yes", // Not part of ResourceVersionCreateInput
                isPrivate: "no", // Should be boolean
                resourceSubType: "InvalidType", // Invalid enum value
                versionLabel: 123, // Should be string
                versionNotes: 456, // Should be string
                rootConnect: 789, // Should be string
            } as unknown as ResourceVersionCreateInput,
            update: {
                id: validIds.id1,
                codeLanguage: 123, // Should be string
                config: "not-an-object", // Should be object
                isAutomatable: "maybe", // Should be boolean
                isComplete: "yes", // Should be boolean
                // isInternal: 1, // Not part of ResourceVersionUpdateInput
                isPrivate: 0, // Should be boolean
                versionLabel: true, // Should be string
                versionNotes: {}, // Should be string
            } as unknown as ResourceVersionUpdateInput,
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                versionLabel: "1.0.0",
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Invalid ID Version",
                    },
                ],
            } as ResourceVersionCreateInput,
            update: {
                id: "invalid-id",
            } as ResourceVersionUpdateInput,
        },
        invalidVersionLabel: {
            create: {
                id: validIds.id1,
                versionLabel: "", // Empty string
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Invalid Version Label",
                    },
                ],
            } as ResourceVersionCreateInput,
        },
        invalidResourceSubType: {
            create: {
                id: validIds.id1,
                resourceSubType: "UnknownSubType" as ResourceSubType, // Not a valid enum value
                versionLabel: "1.0.0",
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Invalid SubType Version",
                    },
                ],
            } as ResourceVersionCreateInput,
        },
        missingRoot: {
            create: {
                id: validIds.id1,
                versionLabel: "1.0.0",
                isPrivate: false,
                // Missing both rootConnect and rootCreate
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "No Root Version",
                    },
                ],
            } as ResourceVersionCreateInput,
        },
        bothRoots: {
            create: {
                id: validIds.id1,
                versionLabel: "1.0.0",
                isPrivate: false,
                rootConnect: validIds.id2,
                rootCreate: {
                    id: validIds.id3,
                    resourceType: "Note",
                    isPrivate: false,
                    ownedByUserConnect: validIds.id4,
                },
                translationsCreate: [
                    {
                        id: validIds.id5,
                        language: "en",
                        name: "Dual Root Version",
                    },
                ],
            },
        },
        invalidCodeLanguage: {
            create: {
                id: validIds.id1,
                codeLanguage: "x".repeat(CODE_LANGUAGE_TOO_LONG_LENGTH), // Too long
                versionLabel: "1.0.0",
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Long Language Version",
                    },
                ],
            } as ResourceVersionCreateInput,
        },
        invalidConfig: {
            create: {
                id: validIds.id1,
                config: "not-an-object",
                versionLabel: "1.0.0",
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Invalid Config Version",
                    },
                ],
            } as unknown as ResourceVersionCreateInput,
        },
        longVersionNotes: {
            create: {
                id: validIds.id1,
                versionLabel: "1.0.0",
                versionNotes: "x".repeat(VERSION_NOTES_TOO_LONG_LENGTH), // Too long (exceeds 4092)
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Long Notes Version",
                    },
                ],
            } as ResourceVersionCreateInput,
        },
    },
    edgeCases: {
        withPublicId: {
            create: {
                id: validIds.id1,
                publicId: validPublicIds.pub2,
                versionLabel: "1.0.0",
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Public ID Version",
                    },
                ],
            },
        },
        codeVersion: {
            create: {
                id: validIds.id1,
                codeLanguage: "python",
                resourceSubType: "CodeDataConverter" as ResourceSubType,
                versionLabel: "1.0.0",
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Python Code Version",
                    },
                ],
            },
        },
        routineVersion: {
            create: {
                id: validIds.id1,
                resourceSubType: "RoutineGenerate" as ResourceSubType,
                isAutomatable: true,
                versionLabel: "1.0.0",
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Generate Routine Version",
                    },
                ],
            },
        },
        standardVersion: {
            create: {
                id: validIds.id1,
                resourceSubType: "StandardPrompt" as ResourceSubType,
                versionLabel: "1.0.0",
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Prompt Standard Version",
                    },
                ],
            },
        },
        completeVersion: {
            create: {
                id: validIds.id1,
                isComplete: true,
                versionLabel: "1.0.0",
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Complete Version",
                    },
                ],
            },
        },
        automatedVersion: {
            create: {
                id: validIds.id1,
                isAutomatable: true,
                versionLabel: "1.0.0",
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Automated Version",
                    },
                ],
            },
        },
        internalVersion: {
            create: {
                id: validIds.id1,
                // isInternal: true, // Not part of ResourceVersionUpdateInput
                versionLabel: "1.0.0",
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Internal Version",
                    },
                ],
            },
        },
        privateVersion: {
            create: {
                id: validIds.id1,
                isPrivate: true,
                versionLabel: "1.0.0",
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Private Version",
                    },
                ],
            },
        },
        withConfiguration: {
            create: {
                id: validIds.id1,
                config: {
                    __version: "2.0",
                    resources: [
                        {
                            link: "https://example.com/docs",
                            usedFor: "Learning",
                            translations: [
                                {
                                    language: "en",
                                    name: "Documentation",
                                    description: "API documentation",
                                },
                            ],
                        },
                    ],
                },
                versionLabel: "1.0.0",
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Configured Version",
                    },
                ],
            },
        },
        withVersionNotes: {
            create: {
                id: validIds.id1,
                versionLabel: "1.2.0",
                versionNotes: "Added new authentication methods and improved error handling.",
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Documented Version",
                    },
                ],
            },
        },
        withRelatedVersions: {
            create: {
                id: validIds.id1,
                versionLabel: "1.0.0",
                isPrivate: false,
                rootConnect: validIds.id2,
                relatedVersionsCreate: [
                    {
                        id: validIds.id3,
                        labels: ["dependency"],
                        toVersionConnect: validIds.id4,
                    },
                    {
                        id: validIds.id5,
                        labels: ["successor", "upgrade"],
                        toVersionConnect: validIds.id6,
                    },
                ],
                translationsCreate: [
                    {
                        id: validIds.id7,
                        language: "en",
                        name: "Related Versions",
                    },
                ],
            },
        },
        multipleTranslations: {
            create: {
                id: validIds.id1,
                versionLabel: "1.0.0",
                isPrivate: false,
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Multi-language Version",
                        description: "A version available in multiple languages.",
                        details: "Comprehensive details in English.",
                        instructions: "Step-by-step instructions in English.",
                    },
                    {
                        id: validIds.id4,
                        language: "es",
                        name: "Versión Multiidioma",
                        description: "Una versión disponible en múltiples idiomas.",
                        details: "Detalles comprensivos en español.",
                    },
                    {
                        id: validIds.id5,
                        language: "fr",
                        name: "Version Multilingue",
                        description: "Une version disponible en plusieurs langues.",
                    },
                ],
            },
        },
        rootCreateVersion: {
            create: {
                id: validIds.id1,
                versionLabel: "1.0.0",
                isPrivate: false,
                rootCreate: {
                    id: validIds.id2,
                    resourceType: "Project",
                    isPrivate: false,
                    ownedByUserConnect: validIds.id3,
                    versionsCreate: [],
                },
                translationsCreate: [
                    {
                        id: validIds.id4,
                        language: "en",
                        name: "Root Create Version",
                    },
                ],
            },
        },
        updateOnlyId: {
            update: {
                id: validIds.id1,
                // Only required field
            },
        },
        updateVersionLabel: {
            update: {
                id: validIds.id1,
                versionLabel: "1.0.1",
            },
        },
        updateConfiguration: {
            update: {
                id: validIds.id1,
                config: { __version: "1.0", resources: [] },
                isComplete: true,
            },
        },
        updateBooleanFields: {
            update: {
                id: validIds.id1,
                isAutomatable: false,
                isComplete: true,
                // isInternal: false, // Not part of ResourceVersionUpdateInput
                isPrivate: true,
            },
        },
        updateWithRelationships: {
            update: {
                id: validIds.id1,
                rootUpdate: {
                    id: validIds.id2,
                    isPrivate: false,
                },
                relatedVersionsCreate: [
                    {
                        id: validIds.id3,
                        labels: ["patch"],
                        toVersionConnect: validIds.id4,
                    },
                ],
                translationsUpdate: [
                    {
                        id: validIds.id5,
                        language: "en",
                        name: "Updated Version Name",
                    },
                ],
            },
        },
        booleanConversions: {
            create: {
                id: validIds.id1,
                isAutomatable: "true",
                isComplete: "false",
                // isInternal: "1", // Not part of ResourceVersionCreateInput
                isPrivate: "0",
                versionLabel: "1.0.0",
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Boolean Conversion Version",
                    },
                ],
            } as unknown as ResourceVersionCreateInput,
        },
    },
};

// ResourceVersion translation fixtures
export const resourceVersionTranslationFixtures: ModelTestFixtures<ResourceVersionTranslationCreateInput, ResourceVersionTranslationUpdateInput> = {
    minimal: {
        create: {
            id: "400000000000000001",
            language: "en",
            name: "Minimal Translation",
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
            name: "Complete Translation",
            description: "A comprehensive description of the resource version.",
            details: "Detailed information about the resource version functionality.",
            instructions: "Step-by-step instructions for using this resource version.",
        },
        update: {
            id: "400000000000000002",
            language: "en",
            name: "Updated Translation",
            description: "Updated description of the resource version.",
            details: "Updated detailed information.",
            instructions: "Updated instructions for using this resource version.",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id, language, and name
                description: "Description without required fields",
            } as ResourceVersionTranslationCreateInput,
            update: {
                // Missing id and language
                name: "Name without required fields",
            } as ResourceVersionTranslationUpdateInput,
        },
        invalidTypes: {
            create: {
                id: "400000000000000003",
                language: 123, // Should be string
                name: true, // Should be string
                description: [], // Should be string
            } as unknown as ResourceVersionTranslationCreateInput,
            update: {
                id: 456, // Should be string
                language: "en",
                name: {}, // Should be string
                details: false, // Should be string
            } as unknown as ResourceVersionTranslationUpdateInput,
        },
    },
    edgeCases: {
        emptyName: {
            create: {
                id: "400000000000000004",
                language: "en",
                name: "",
            },
        },
        multipleLanguages: {
            create: {
                id: "400000000000000005",
                language: "es",
                name: "Versión en Español",
                description: "Descripción en español del recurso.",
            },
        },
        maxLengthFields: {
            create: {
                id: "400000000000000006",
                language: "en",
                name: "N".repeat(NAME_MAX_LENGTH), // Max length name
                description: "D".repeat(DESCRIPTION_MAX_LENGTH), // Max length description
                details: "D".repeat(DETAILS_MAX_LENGTH), // Max length details
                instructions: "I".repeat(INSTRUCTIONS_MAX_LENGTH), // Max length instructions
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers: {
    create: (base: ResourceVersionCreateInput) => ResourceVersionCreateInput;
    update: (base: ResourceVersionUpdateInput) => ResourceVersionUpdateInput;
} = {
    create: (base: ResourceVersionCreateInput): ResourceVersionCreateInput => ({
        ...base,
        id: base.id || validIds.id1,
        versionLabel: base.versionLabel || "1.0.0",
        isPrivate: base.isPrivate !== undefined ? base.isPrivate : false,
        rootConnect: base.rootConnect || (base.rootCreate ? undefined : validIds.id2),
        translationsCreate: base.translationsCreate || [
            {
                id: validIds.id3,
                language: "en",
                name: "Default Version",
            },
        ],
    }),
    update: (base: ResourceVersionUpdateInput): ResourceVersionUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export enhanced type-safe factories
export const resourceVersionTestDataFactory = new TypedTestDataFactory(resourceVersionFixtures, resourceVersionValidation, customizers);
export const resourceVersionTranslationTestDataFactory = new TestDataFactory(resourceVersionTranslationFixtures);

// Export type-safe fixtures with validation capabilities
export const typedResourceVersionFixtures = createTypedFixtures(resourceVersionFixtures, resourceVersionValidation);

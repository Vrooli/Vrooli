import { type ModelTestFixtures, TestDataFactory } from "../validationTestUtils.js";

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
export const resourceVersionFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            versionLabel: "1.0.0",
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
            config: { theme: "dark", autoSave: true },
            isAutomatable: true,
            isComplete: true,
            isInternal: false,
            isPrivate: false,
            resourceSubType: "RoutineApi",
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
            unknownField1: "should be stripped",
            unknownField2: 123,
            unknownField3: true,
        },
        update: {
            id: validIds.id1,
            codeLanguage: "javascript",
            config: { theme: "light", autoSave: false },
            isAutomatable: false,
            isComplete: true,
            isInternal: true,
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
            relatedVersionsDelete: [validIds.id7],
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
            },
            update: {
                // Missing required id
                versionLabel: "1.0.1",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                publicId: 456, // Should be string
                codeLanguage: 789, // Should be string
                config: "not-an-object", // Should be object
                isAutomatable: "true", // Should be boolean
                isComplete: "false", // Should be boolean
                isInternal: "yes", // Should be boolean
                isPrivate: "no", // Should be boolean
                resourceSubType: "InvalidType", // Invalid enum value
                versionLabel: 123, // Should be string
                versionNotes: 456, // Should be string
                rootConnect: 789, // Should be string
            },
            update: {
                id: validIds.id1,
                codeLanguage: 123, // Should be string
                config: "not-an-object", // Should be object
                isAutomatable: "maybe", // Should be boolean
                isComplete: "yes", // Should be boolean
                isInternal: 1, // Should be boolean
                isPrivate: 0, // Should be boolean
                versionLabel: true, // Should be string
                versionNotes: {}, // Should be string
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                versionLabel: "1.0.0",
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Invalid ID Version",
                    },
                ],
            },
            update: {
                id: "invalid-id",
            },
        },
        invalidVersionLabel: {
            create: {
                id: validIds.id1,
                versionLabel: "", // Empty string
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Invalid Version Label",
                    },
                ],
            },
        },
        invalidResourceSubType: {
            create: {
                id: validIds.id1,
                resourceSubType: "UnknownSubType", // Not a valid enum value
                versionLabel: "1.0.0",
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Invalid SubType Version",
                    },
                ],
            },
        },
        missingRoot: {
            create: {
                id: validIds.id1,
                versionLabel: "1.0.0",
                // Missing both rootConnect and rootCreate
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "No Root Version",
                    },
                ],
            },
        },
        bothRoots: {
            create: {
                id: validIds.id1,
                versionLabel: "1.0.0",
                rootConnect: validIds.id2,
                rootCreate: {
                    id: validIds.id3,
                    resourceType: "Note",
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
                codeLanguage: "x".repeat(129), // Too long
                versionLabel: "1.0.0",
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Long Language Version",
                    },
                ],
            },
        },
        invalidConfig: {
            create: {
                id: validIds.id1,
                config: "not-an-object",
                versionLabel: "1.0.0",
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Invalid Config Version",
                    },
                ],
            },
        },
        longVersionNotes: {
            create: {
                id: validIds.id1,
                versionLabel: "1.0.0",
                versionNotes: "x".repeat(4093), // Too long (exceeds 4092)
                rootConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "Long Notes Version",
                    },
                ],
            },
        },
    },
    edgeCases: {
        withPublicId: {
            create: {
                id: validIds.id1,
                publicId: validPublicIds.pub2,
                versionLabel: "1.0.0",
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
                resourceSubType: "CodeDataConverter",
                versionLabel: "1.0.0",
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
                resourceSubType: "RoutineGenerate",
                isAutomatable: true,
                versionLabel: "1.0.0",
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
                resourceSubType: "StandardPrompt",
                versionLabel: "1.0.0",
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
                isInternal: true,
                versionLabel: "1.0.0",
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
                    version: "2.0",
                    features: ["automation", "validation"],
                    settings: { timeout: 30000, retries: 3 },
                },
                versionLabel: "1.0.0",
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
                rootCreate: {
                    id: validIds.id2,
                    resourceType: "Project",
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
                config: { mode: "production" },
                isComplete: true,
            },
        },
        updateBooleanFields: {
            update: {
                id: validIds.id1,
                isAutomatable: false,
                isComplete: true,
                isInternal: false,
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
                isInternal: "1",
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
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers = {
    create: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
        versionLabel: base.versionLabel || "1.0.0",
        rootConnect: base.rootConnect || (base.rootCreate ? undefined : validIds.id2),
        translationsCreate: base.translationsCreate || [
            {
                id: validIds.id3,
                language: "en",
                name: "Default Version",
            },
        ],
    }),
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const resourceVersionTestDataFactory = new TestDataFactory(resourceVersionFixtures, customizers);

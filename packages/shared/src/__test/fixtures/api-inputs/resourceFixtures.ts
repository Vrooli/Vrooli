import type { ResourceCreateInput, ResourceUpdateInput, ResourceVersionTranslationCreateInput, ResourceVersionTranslationUpdateInput } from "../../../api/types.js";
import { ResourceType } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { resourceValidation } from "../../../validation/models/resource.js";

// Magic number constants for testing
const LONG_DESCRIPTION_LENGTH = 1000;
const LONG_DETAILS_LENGTH = 2000;
const LONG_INSTRUCTIONS_LENGTH = 1500;

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
    pub1: "abc1234567",
    pub2: "def6789012",
    pub3: "ghi1357913",
    pub4: "jkl2468024",
};

// Shared resource test fixtures
export const resourceFixtures: ModelTestFixtures<ResourceCreateInput, ResourceUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            resourceType: ResourceType.Note,
            isPrivate: false,
            ownedByUserConnect: validIds.id2,
            versionsCreate: [
                {
                    id: validIds.id3,
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    translationsCreate: [
                        {
                            id: validIds.id4,
                            language: "en",
                            name: "Minimal Resource",
                        },
                    ],
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
            isInternal: false,
            isPrivate: true,
            permissions: JSON.stringify(["read", "write", "admin"]),
            resourceType: ResourceType.Project,
            ownedByTeamConnect: validIds.id2,
            parentConnect: validIds.id3,
            tagsConnect: [validIds.id4, validIds.id5],
            tagsCreate: [
                {
                    id: validIds.id6,
                    tag: "test-resource",
                },
                {
                    id: validIds.id7,
                    tag: "validation",
                },
            ],
            versionsCreate: [
                {
                    id: validIds.id8,
                    versionLabel: "1.0.0",
                    isComplete: true,
                    isPrivate: false,
                    translationsCreate: [
                        {
                            id: validIds.id3,
                            language: "en",
                            name: "Complete Test Resource",
                            description: "A comprehensive test resource with all features enabled.",
                        },
                        {
                            id: validIds.id4,
                            language: "es",
                            name: "Recurso de Prueba Completo",
                            description: "Un recurso de prueba integral con todas las características habilitadas.",
                        },
                    ],
                },
            ],
        },
        update: {
            id: validIds.id1,
            isInternal: true,
            isPrivate: false,
            permissions: JSON.stringify(["read", "write"]),
            ownedByUserConnect: validIds.id5,
            tagsConnect: [validIds.id6],
            tagsDisconnect: [validIds.id4],
            tagsCreate: [
                {
                    id: validIds.id7,
                    tag: "updated-tag",
                },
            ],
            versionsCreate: [
                {
                    id: validIds.id8,
                    versionLabel: "1.1.0",
                    isPrivate: false,
                    translationsCreate: [
                        {
                            id: validIds.id2,
                            language: "en",
                            name: "Updated Resource Version",
                        },
                    ],
                },
            ],
            versionsUpdate: [
                {
                    id: validIds.id3,
                    versionLabel: "1.0.1",
                },
            ],
            versionsDelete: [validIds.id4],
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id, resourceType, versions, and owner
                publicId: validPublicIds.pub1,
                isPrivate: true,
            } as ResourceCreateInput,
            update: {
                // Missing required id
                isPrivate: false,
            } as ResourceUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                publicId: 456, // Should be string
                isInternal: "true", // Should be boolean
                isPrivate: "false", // Should be boolean
                permissions: ["read", "write"], // Should be JSON string
                resourceType: "InvalidType", // Invalid enum value
                ownedByUserConnect: 789, // Should be string
            } as unknown as ResourceCreateInput,
            update: {
                id: validIds.id1,
                isInternal: "yes", // Should be boolean
                isPrivate: "no", // Should be boolean
                permissions: { read: true }, // Should be JSON string
            } as unknown as ResourceUpdateInput,
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                resourceType: ResourceType.Note,
                ownedByUserConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "Invalid ID Resource",
                            },
                        ],
                    },
                ],
            },
            update: {
                id: "invalid-id",
            },
        },
        invalidResourceType: {
            create: {
                id: validIds.id1,
                resourceType: "UnknownType", // Not a valid enum value
                ownedByUserConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "Invalid Type Resource",
                            },
                        ],
                    },
                ],
            } as unknown as ResourceCreateInput,
        },
        missingOwner: {
            create: {
                id: validIds.id1,
                resourceType: ResourceType.Note,
                // Missing both ownedByUserConnect and ownedByTeamConnect
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "No Owner Resource",
                            },
                        ],
                    },
                ],
            } as ResourceCreateInput,
        },
        bothOwners: {
            create: {
                id: validIds.id1,
                resourceType: ResourceType.Note,
                ownedByUserConnect: validIds.id2,
                ownedByTeamConnect: validIds.id3, // Should only have one owner
                versionsCreate: [
                    {
                        id: validIds.id4,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id5,
                                language: "en",
                                name: "Dual Owner Resource",
                            },
                        ],
                    },
                ],
            },
        },
        missingVersions: {
            create: {
                id: validIds.id1,
                resourceType: ResourceType.Note,
                ownedByUserConnect: validIds.id2,
                // Missing required versionsCreate
            } as ResourceCreateInput,
        },
        invalidPublicId: {
            create: {
                id: validIds.id1,
                publicId: "invalid-public-id!", // Invalid characters
                resourceType: ResourceType.Note,
                ownedByUserConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "Invalid Public ID Resource",
                            },
                        ],
                    },
                ],
            },
        },
        invalidPermissions: {
            create: {
                id: validIds.id1,
                permissions: "not-valid-json", // Invalid JSON string
                resourceType: ResourceType.Note,
                ownedByUserConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "Invalid Permissions Resource",
                            },
                        ],
                    },
                ],
            },
        },
    },
    edgeCases: {
        userOwnedResource: {
            create: {
                id: validIds.id1,
                resourceType: ResourceType.Api,
                ownedByUserConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "User Owned API",
                            },
                        ],
                    },
                ],
            },
        },
        teamOwnedResource: {
            create: {
                id: validIds.id1,
                resourceType: ResourceType.Routine,
                ownedByTeamConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "Team Owned Routine",
                            },
                        ],
                    },
                ],
            },
        },
        codeResource: {
            create: {
                id: validIds.id1,
                resourceType: ResourceType.Code,
                ownedByUserConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "Code Resource",
                            },
                        ],
                    },
                ],
            },
        },
        standardResource: {
            create: {
                id: validIds.id1,
                resourceType: ResourceType.Standard,
                ownedByUserConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "Standard Resource",
                            },
                        ],
                    },
                ],
            },
        },
        projectResource: {
            create: {
                id: validIds.id1,
                resourceType: ResourceType.Project,
                ownedByUserConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "Project Resource",
                            },
                        ],
                    },
                ],
            },
        },
        noteResource: {
            create: {
                id: validIds.id1,
                resourceType: ResourceType.Note,
                ownedByUserConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "Note Resource",
                            },
                        ],
                    },
                ],
            },
        },
        withPublicId: {
            create: {
                id: validIds.id1,
                publicId: validPublicIds.pub2,
                resourceType: ResourceType.Api,
                ownedByUserConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "Resource with Public ID",
                            },
                        ],
                    },
                ],
            },
        },
        internalResource: {
            create: {
                id: validIds.id1,
                isInternal: true,
                resourceType: ResourceType.Code,
                ownedByUserConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "Internal Resource",
                            },
                        ],
                    },
                ],
            },
        },
        privateResource: {
            create: {
                id: validIds.id1,
                isPrivate: true,
                resourceType: ResourceType.Note,
                ownedByUserConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: true,
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "Private Resource",
                            },
                        ],
                    },
                ],
            },
        },
        withPermissions: {
            create: {
                id: validIds.id1,
                permissions: JSON.stringify(["read", "write", "delete", "admin"]),
                resourceType: ResourceType.Project,
                ownedByUserConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "Resource with Permissions",
                            },
                        ],
                    },
                ],
            },
        },
        withParent: {
            create: {
                id: validIds.id1,
                resourceType: ResourceType.Note,
                ownedByUserConnect: validIds.id2,
                parentConnect: validIds.id3,
                versionsCreate: [
                    {
                        id: validIds.id4,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id5,
                                language: "en",
                                name: "Child Resource",
                            },
                        ],
                    },
                ],
            },
        },
        withTags: {
            create: {
                id: validIds.id1,
                resourceType: ResourceType.Routine,
                ownedByUserConnect: validIds.id2,
                tagsConnect: [validIds.id3, validIds.id4],
                tagsCreate: [
                    {
                        id: validIds.id5,
                        tag: "automation",
                    },
                    {
                        id: validIds.id6,
                        tag: "workflow",
                    },
                ],
                versionsCreate: [
                    {
                        id: validIds.id7,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id8,
                                language: "en",
                                name: "Tagged Resource",
                            },
                        ],
                    },
                ],
            },
        },
        multipleVersions: {
            create: {
                id: validIds.id1,
                resourceType: ResourceType.Api,
                ownedByUserConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "Multi Version API v1",
                            },
                        ],
                    },
                    {
                        id: validIds.id5,
                        versionLabel: "1.1.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id6,
                                language: "en",
                                name: "Multi Version API v1.1",
                            },
                        ],
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
        updateOwnership: {
            update: {
                id: validIds.id1,
                ownedByTeamConnect: validIds.id2,
            },
        },
        updateWithTagOperations: {
            update: {
                id: validIds.id1,
                tagsConnect: [validIds.id2],
                tagsDisconnect: [validIds.id3],
                tagsCreate: [
                    {
                        id: validIds.id4,
                        tag: "updated-tag",
                    },
                ],
            },
        },
        updateWithVersionOperations: {
            update: {
                id: validIds.id1,
                versionsCreate: [
                    {
                        id: validIds.id2,
                        versionLabel: "2.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id3,
                                language: "en",
                                name: "New Version",
                            },
                        ],
                    },
                ],
                versionsUpdate: [
                    {
                        id: validIds.id4,
                        versionLabel: "1.0.1",
                    },
                ],
                versionsDelete: [validIds.id5],
            },
        },
        booleanConversions: {
            create: {
                id: validIds.id1,
                isInternal: "true",
                isPrivate: "false",
                resourceType: ResourceType.Note,
                ownedByUserConnect: validIds.id2,
                versionsCreate: [
                    {
                        id: validIds.id3,
                        versionLabel: "1.0.0",
                        isComplete: true,
                        isPrivate: false,
                        translationsCreate: [
                            {
                                id: validIds.id4,
                                language: "en",
                                name: "Boolean Conversion Resource",
                            },
                        ],
                    },
                ],
            } as unknown as ResourceCreateInput,
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers = {
    create: (base: ResourceCreateInput): ResourceCreateInput => ({
        ...base,
        id: base.id || validIds.id1,
        resourceType: base.resourceType || ResourceType.Note,
        ownedByUserConnect: base.ownedByUserConnect || (base.ownedByTeamConnect ? undefined : validIds.id2),
        versionsCreate: base.versionsCreate || [
            {
                id: validIds.id3,
                versionLabel: "1.0.0",
                isPrivate: false,
                translationsCreate: [
                    {
                        id: validIds.id4,
                        language: "en",
                        name: "Default Resource",
                    },
                ],
            },
        ],
    }),
    update: (base: ResourceUpdateInput): ResourceUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Resource version translation fixtures
export const resourceVersionTranslationFixtures: ModelTestFixtures<ResourceVersionTranslationCreateInput, ResourceVersionTranslationUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            language: "en",
            name: "Minimal Resource",
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
            name: "Complete Resource Translation",
            description: "A detailed description of the resource",
            details: "Additional details about the resource implementation",
            instructions: "Step-by-step instructions for using this resource",
        },
        update: {
            id: validIds.id2,
            language: "en",
            name: "Updated Resource Translation",
            description: "Updated description",
            details: "Updated details",
            instructions: "Updated instructions",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id, language, and name
                description: "Resource without required fields",
            } as ResourceVersionTranslationCreateInput,
            update: {
                // Missing required id and language
                name: "Updated name",
            } as ResourceVersionTranslationUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                language: true, // Should be string
                name: [], // Should be string
            } as unknown as ResourceVersionTranslationCreateInput,
            update: {
                id: true, // Should be string
                language: 789, // Should be string
                name: {}, // Should be string
            } as unknown as ResourceVersionTranslationUpdateInput,
        },
    },
    edgeCases: {
        emptyOptionalFields: {
            create: {
                id: validIds.id3,
                language: "en",
                name: "Resource with Empty Fields",
                description: "",
                details: "",
                instructions: "",
            },
        },
        multipleLanguages: {
            create: {
                id: validIds.id4,
                language: "es",
                name: "Recurso en Español",
                description: "Descripción del recurso",
            },
        },
        longContent: {
            create: {
                id: validIds.id5,
                language: "en",
                name: "Resource with Long Content",
                description: "D".repeat(LONG_DESCRIPTION_LENGTH), // Long description
                details: "T".repeat(LONG_DETAILS_LENGTH), // Long details
                instructions: "I".repeat(LONG_INSTRUCTIONS_LENGTH), // Long instructions
            },
        },
    },
};

// Export enhanced type-safe factories
export const resourceTestDataFactory = new TypedTestDataFactory(resourceFixtures, resourceValidation, customizers);
export const resourceVersionTranslationTestDataFactory = new TestDataFactory(resourceVersionTranslationFixtures);

// Export type-safe fixtures with validation capabilities
export const typedResourceFixtures = createTypedFixtures(resourceFixtures, resourceValidation);

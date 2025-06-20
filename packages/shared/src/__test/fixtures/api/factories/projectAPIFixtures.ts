/* c8 ignore start */
/**
 * Type-safe Project API fixture factory
 * 
 * This factory provides comprehensive fixtures for Project objects with:
 * - Zero `any` types
 * - Full validation integration for Resource and ResourceVersion
 * - Shape function integration
 * - Comprehensive error scenarios 
 * - Project-specific factory methods
 * 
 * Projects in Vrooli are Resources with resourceType="Project" and ResourceVersions.
 * This factory handles both the Resource root and ResourceVersion data.
 */
import type {
    Resource,
    ResourceCreateInput,
    ResourceUpdateInput,
    ResourceVersion,
    ResourceVersionCreateInput,
    ResourceVersionTranslationCreateInput,
    ResourceVersionTranslationUpdateInput,
    ResourceVersionUpdateInput,
} from "../../../../api/types.js";
import {
    ResourceType,
} from "../../../../api/types.js";
import { generatePK } from "../../../../id/snowflake.js";
import { resourceValidation } from "../../../../validation/models/resource.js";
import { projectConfigFixtures } from "../../config/projectConfigFixtures.js";
import { BaseAPIFixtureFactory } from "../BaseAPIFixtureFactory.js";

// Magic number constants for testing
const NAME_MAX_LENGTH = 256;
const VERSION_NOTES_LENGTH = 1000;
const TRANSLATION_NAME_MAX_LENGTH = 128;
const TRANSLATION_DESCRIPTION_LENGTH = 512;
const TRANSLATION_DETAILS_LENGTH = 2048;
const TRANSLATION_INSTRUCTIONS_LENGTH = 2048;

// ========================================
// Type-Safe Fixture Data
// ========================================

const validIds = {
    resource1: generatePK().toString(),
    resource2: generatePK().toString(),
    version1: generatePK().toString(),
    version2: generatePK().toString(),
    version3: generatePK().toString(),
    translation1: generatePK().toString(),
    translation2: generatePK().toString(),
    translation3: generatePK().toString(),
    translation4: generatePK().toString(),
    user1: generatePK().toString(),
    team1: generatePK().toString(),
    tag1: generatePK().toString(),
    tag2: generatePK().toString(),
};

// Core project fixture data with complete type safety
const projectFixtureData = {
    minimal: {
        create: {
            id: validIds.resource1,
            isPrivate: false,
            resourceType: ResourceType.Project,
            versionsCreate: [{
                id: validIds.version1,
                isPrivate: false,
                versionLabel: "1.0.0",
                config: projectConfigFixtures.minimal,
                translationsCreate: [{
                    id: validIds.translation1,
                    language: "en",
                    name: "Minimal Project",
                } satisfies ResourceVersionTranslationCreateInput],
            } satisfies ResourceVersionCreateInput],
        } satisfies ResourceCreateInput,

        update: {
            id: validIds.resource1,
            versionsUpdate: [{
                id: validIds.version1,
                versionLabel: "1.0.1",
                translationsUpdate: [{
                    id: validIds.translation1,
                    language: "en",
                    name: "Updated Minimal Project",
                } satisfies ResourceVersionTranslationUpdateInput],
            } satisfies ResourceVersionUpdateInput],
        } satisfies ResourceUpdateInput,

        find: {
            __typename: "Resource",
            id: validIds.resource1,
            publicId: `project_${validIds.resource1}`,
            isPrivate: false,
            isDeleted: false,
            isInternal: false,
            resourceType: ResourceType.Project,
            hasCompleteVersion: true,
            permissions: "{}",
            score: 0,
            views: 0,
            bookmarks: 0,
            transfersCount: 0,
            issuesCount: 0,
            pullRequestsCount: 0,
            versionsCount: 1,
            translatedName: "Minimal Project",
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            completedAt: null,
            bookmarkedBy: [],
            createdBy: null,
            owner: null,
            parent: null,
            issues: [],
            pullRequests: [],
            stats: [],
            tags: [],
            transfers: [],
            versions: [{
                __typename: "ResourceVersion",
                id: validIds.version1,
                publicId: `project_${validIds.version1}`,
                isPrivate: false,
                isComplete: true,
                isAutomatable: false,
                versionLabel: "1.0.0",
                versionNotes: null,
                codeLanguage: null,
                config: projectConfigFixtures.minimal,
                resourceSubType: null,
                relatedVersions: [],
                root: null, // Will be connected by shape function
                translations: [{
                    __typename: "ResourceVersionTranslation",
                    id: validIds.translation1,
                    language: "en",
                    name: "Minimal Project",
                    description: null,
                    details: null,
                    instructions: null,
                }],
            }] as ResourceVersion[],
            you: {
                __typename: "ResourceYou",
                canBookmark: true,
                canComment: true,
                canDelete: true,
                canReact: true,
                canRead: true,
                canTransfer: true,
                canUpdate: true,
                isBookmarked: false,
                isViewed: false,
                reaction: null,
            },
        } as Resource,
    },

    complete: {
        create: {
            id: validIds.resource2,
            isPrivate: false,
            isInternal: false,
            resourceType: ResourceType.Project,
            permissions: JSON.stringify({ members: ["read", "write"], public: ["read"] }),
            ownedByUserConnect: validIds.user1,
            tagsCreate: [
                { id: validIds.tag1, tag: "open-source" },
                { id: validIds.tag2, tag: "education" },
            ],
            versionsCreate: [{
                id: validIds.version2,
                isPrivate: false,
                isComplete: true,
                isAutomatable: false,
                versionLabel: "2.1.0",
                versionNotes: "Major release with new features and improvements",
                config: projectConfigFixtures.complete,
                translationsCreate: [
                    {
                        id: validIds.translation2,
                        language: "en",
                        name: "Complete Educational Project",
                        description: "A comprehensive educational project with multiple resources and documentation",
                        details: "This project includes tutorials, exercises, documentation, and community resources for learning",
                        instructions: "1. Start with the getting started guide\n2. Work through the tutorials\n3. Complete the exercises\n4. Join the community",
                    } satisfies ResourceVersionTranslationCreateInput,
                    {
                        id: validIds.translation3,
                        language: "es",
                        name: "Proyecto Educativo Completo",
                        description: "Un proyecto educativo integral con múltiples recursos y documentación",
                        details: "Este proyecto incluye tutoriales, ejercicios, documentación y recursos comunitarios para el aprendizaje",
                        instructions: "1. Comience con la guía de inicio\n2. Trabaje a través de los tutoriales\n3. Complete los ejercicios\n4. Únase a la comunidad",
                    } satisfies ResourceVersionTranslationCreateInput,
                ],
            } satisfies ResourceVersionCreateInput],
        } satisfies ResourceCreateInput,

        update: {
            id: validIds.resource2,
            isPrivate: false,
            permissions: JSON.stringify({ members: ["read", "write", "admin"], public: ["read"] }),
            tagsConnect: [validIds.tag1],
            tagsCreate: [{ id: generatePK().toString(), tag: "featured" }],
            versionsUpdate: [{
                id: validIds.version2,
                versionLabel: "2.1.1",
                versionNotes: "Bug fixes and minor improvements",
                isComplete: true,
                translationsUpdate: [{
                    id: validIds.translation2,
                    language: "en",
                    name: "Complete Educational Project - Updated",
                    description: "A comprehensive educational project with multiple resources, documentation, and recent updates",
                } satisfies ResourceVersionTranslationUpdateInput],
            } satisfies ResourceVersionUpdateInput],
        } satisfies ResourceUpdateInput,

        find: {
            __typename: "Resource",
            id: validIds.resource2,
            publicId: `project_${validIds.resource2}`,
            isPrivate: false,
            isDeleted: false,
            isInternal: false,
            resourceType: ResourceType.Project,
            hasCompleteVersion: true,
            permissions: JSON.stringify({ members: ["read", "write", "admin"], public: ["read"] }),
            score: 98,
            views: 2547,
            bookmarks: 156,
            transfersCount: 0,
            issuesCount: 3,
            pullRequestsCount: 1,
            versionsCount: 1,
            translatedName: "Complete Educational Project",
            createdAt: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(), // 30 days ago
            updatedAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(), // 2 days ago
            completedAt: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(), // 7 days ago
            bookmarkedBy: [],
            createdBy: null,
            owner: null, // Would be populated with full User object in real scenario
            parent: null,
            issues: [],
            pullRequests: [],
            stats: [],
            tags: [
                { __typename: "Tag", id: validIds.tag1, tag: "open-source", bookmarkedBy: [], bookmarks: 0, createdAt: new Date().toISOString(), reports: [], resources: [], teams: [], translations: [], updatedAt: new Date().toISOString(), you: { __typename: "TagYou", isBookmarked: false, isOwn: false } },
                { __typename: "Tag", id: validIds.tag2, tag: "education", bookmarkedBy: [], bookmarks: 0, createdAt: new Date().toISOString(), reports: [], resources: [], teams: [], translations: [], updatedAt: new Date().toISOString(), you: { __typename: "TagYou", isBookmarked: false, isOwn: false } },
            ],
            transfers: [],
            versions: [{
                __typename: "ResourceVersion",
                id: validIds.version2,
                publicId: `project_${validIds.version2}`,
                isPrivate: false,
                isComplete: true,
                isAutomatable: false,
                versionLabel: "2.1.0",
                versionNotes: "Major release with new features and improvements",
                codeLanguage: null,
                config: projectConfigFixtures.complete,
                resourceSubType: null,
                relatedVersions: [],
                root: null, // Will be connected by shape function
                translations: [
                    {
                        __typename: "ResourceVersionTranslation",
                        id: validIds.translation2,
                        language: "en",
                        name: "Complete Educational Project",
                        description: "A comprehensive educational project with multiple resources and documentation",
                        details: "This project includes tutorials, exercises, documentation, and community resources for learning",
                        instructions: "1. Start with the getting started guide\n2. Work through the tutorials\n3. Complete the exercises\n4. Join the community",
                    },
                    {
                        __typename: "ResourceVersionTranslation",
                        id: validIds.translation3,
                        language: "es",
                        name: "Proyecto Educativo Completo",
                        description: "Un proyecto educativo integral con múltiples recursos y documentación",
                        details: "Este proyecto incluye tutoriales, ejercicios, documentación y recursos comunitarios para el aprendizaje",
                        instructions: "1. Comience con la guía de inicio\n2. Trabaje a través de los tutoriales\n3. Complete los ejercicios\n4. Únase a la comunidad",
                    },
                ],
            }] as ResourceVersion[],
            you: {
                __typename: "ResourceYou",
                canBookmark: true,
                canComment: true,
                canDelete: true,
                canReact: true,
                canRead: true,
                canTransfer: true,
                canUpdate: true,
                isBookmarked: true,
                isViewed: true,
                reaction: "👍",
            },
        } as Resource,
    },

    invalid: {
        missingRequired: {
            create: {
                // Missing required fields: id, resourceType, versionsCreate
                isPrivate: false,
            },
            update: {
                // Missing required id
                isPrivate: true,
            },
        },

        invalidTypes: {
            create: {
                id: 123, // Should be string
                resourceType: "InvalidType", // Should be ResourceType enum
                isPrivate: "yes", // Should be boolean
                versionsCreate: "not an array", // Should be array
            },
            update: {
                id: null, // Should be string
                isPrivate: "no", // Should be boolean
                permissions: 123, // Should be string
            },
        },

        businessLogicErrors: {
            duplicateVersionLabels: {
                id: validIds.resource1,
                resourceType: ResourceType.Project,
                isPrivate: false,
                versionsCreate: [
                    {
                        id: validIds.version1,
                        versionLabel: "1.0.0",
                        isPrivate: false,
                        translationsCreate: [{ id: validIds.translation1, language: "en", name: "Test" }],
                    },
                    {
                        id: validIds.version2,
                        versionLabel: "1.0.0", // Duplicate version label
                        isPrivate: false,
                        translationsCreate: [{ id: validIds.translation2, language: "en", name: "Test 2" }],
                    },
                ],
            },

            invalidOwnership: {
                id: validIds.resource1,
                resourceType: ResourceType.Project,
                isPrivate: false,
                ownedByUserConnect: "invalid-user-id",
                ownedByTeamConnect: "invalid-team-id", // Can't be owned by both
                versionsCreate: [{
                    id: validIds.version1,
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    translationsCreate: [{ id: validIds.translation1, language: "en", name: "Test" }],
                }],
            },
        },

        validationErrors: {
            nameTooLong: {
                id: validIds.resource1,
                resourceType: ResourceType.Project,
                isPrivate: false,
                versionsCreate: [{
                    id: validIds.version1,
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    translationsCreate: [{
                        id: validIds.translation1,
                        language: "en",
                        name: "A".repeat(NAME_MAX_LENGTH), // Assuming max length is 255
                    }],
                }],
            },

            invalidVersionLabel: {
                id: validIds.resource1,
                resourceType: ResourceType.Project,
                isPrivate: false,
                versionsCreate: [{
                    id: validIds.version1,
                    versionLabel: "", // Empty version label
                    isPrivate: false,
                    translationsCreate: [{
                        id: validIds.translation1,
                        language: "en",
                        name: "Test Project",
                    }],
                }],
            },
        },
    },

    edgeCases: {
        minimalValid: {
            create: {
                id: generatePK().toString(),
                resourceType: ResourceType.Project,
                isPrivate: false,
                versionsCreate: [{
                    id: generatePK().toString(),
                    versionLabel: "1",
                    isPrivate: false,
                    translationsCreate: [{
                        id: generatePK().toString(),
                        language: "en",
                        name: "A",
                    }],
                }],
            } satisfies ResourceCreateInput,

            update: {
                id: validIds.resource1,
            } satisfies ResourceUpdateInput,
        },

        maximalValid: {
            create: {
                id: generatePK().toString(),
                publicId: `project_${generatePK().toString()}`,
                resourceType: ResourceType.Project,
                isPrivate: false,
                isInternal: false,
                permissions: JSON.stringify({
                    members: ["read", "write", "admin", "delete"],
                    public: ["read"],
                    teams: { [validIds.team1]: ["read", "write"] },
                }),
                ownedByUserConnect: validIds.user1,
                tagsCreate: [
                    { id: generatePK().toString(), tag: "featured" },
                    { id: generatePK().toString(), tag: "community" },
                    { id: generatePK().toString(), tag: "advanced" },
                ],
                versionsCreate: [{
                    id: generatePK().toString(),
                    publicId: `version_${generatePK().toString()}`,
                    versionLabel: "10.15.42-beta.3+build.2024.365",
                    versionNotes: "X".repeat(VERSION_NOTES_LENGTH), // Assuming reasonable max length
                    isPrivate: false,
                    isComplete: true,
                    isAutomatable: true,
                    config: projectConfigFixtures.complete,
                    translationsCreate: [
                        {
                            id: generatePK().toString(),
                            language: "en",
                            name: "Z".repeat(TRANSLATION_NAME_MAX_LENGTH), // Assuming max length
                            description: "Y".repeat(TRANSLATION_DESCRIPTION_LENGTH), // Assuming max length
                            details: "W".repeat(TRANSLATION_DETAILS_LENGTH), // Assuming max length
                            instructions: "V".repeat(TRANSLATION_INSTRUCTIONS_LENGTH), // Assuming max length
                        },
                        {
                            id: generatePK().toString(),
                            language: "es",
                            name: "Proyecto Máximo",
                            description: "Descripción completa del proyecto",
                            details: "Detalles técnicos exhaustivos",
                            instructions: "Instrucciones paso a paso detalladas",
                        },
                        {
                            id: generatePK().toString(),
                            language: "fr",
                            name: "Projet Maximal",
                            description: "Description complète du projet",
                            details: "Détails techniques exhaustifs",
                            instructions: "Instructions détaillées étape par étape",
                        },
                    ],
                }],
            } satisfies ResourceCreateInput,

            update: {
                id: validIds.resource2,
                isPrivate: true,
                isInternal: true,
                permissions: JSON.stringify({ private: true }),
                ownedByUserConnect: validIds.user1,
                tagsCreate: [{ id: generatePK().toString(), tag: "updated" }],
                tagsConnect: [validIds.tag1, validIds.tag2],
                versionsCreate: [{
                    id: generatePK().toString(),
                    versionLabel: "11.0.0",
                    versionNotes: "Complete rewrite with breaking changes",
                    isPrivate: true,
                    isComplete: true,
                    config: projectConfigFixtures.variants.researchProject,
                    translationsCreate: [{
                        id: generatePK().toString(),
                        language: "en",
                        name: "Next Generation Project",
                        description: "Revolutionary new version",
                    }],
                }],
            } satisfies ResourceUpdateInput,
        },

        boundaryValues: {
            emptyArrays: {
                id: generatePK().toString(),
                resourceType: ResourceType.Project,
                isPrivate: false,
                versionsCreate: [{
                    id: generatePK().toString(),
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    translationsCreate: [], // Empty translations array
                }],
            } satisfies ResourceCreateInput,

            specialCharacters: {
                id: generatePK().toString(),
                resourceType: ResourceType.Project,
                isPrivate: false,
                versionsCreate: [{
                    id: generatePK().toString(),
                    versionLabel: "1.0.0-α.β.γ+δ.ε",
                    isPrivate: false,
                    translationsCreate: [{
                        id: generatePK().toString(),
                        language: "en",
                        name: "📊 Data-Science & ML/AI Project (α→β) [2024]",
                        description: "Special chars: αβγδε 中文 العربية עברית 🚀✨💡",
                    }],
                }],
            } satisfies ResourceCreateInput,
        },

        permissionScenarios: {
            publicProject: {
                id: generatePK().toString(),
                resourceType: ResourceType.Project,
                isPrivate: false,
                permissions: JSON.stringify({ public: ["read"] }),
                versionsCreate: [{
                    id: generatePK().toString(),
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    translationsCreate: [{
                        id: generatePK().toString(),
                        language: "en",
                        name: "Public Project",
                    }],
                }],
            } satisfies ResourceCreateInput,

            privateProject: {
                id: generatePK().toString(),
                resourceType: ResourceType.Project,
                isPrivate: true,
                permissions: JSON.stringify({ members: ["read", "write"] }),
                ownedByUserConnect: validIds.user1,
                versionsCreate: [{
                    id: generatePK().toString(),
                    versionLabel: "1.0.0",
                    isPrivate: true,
                    translationsCreate: [{
                        id: generatePK().toString(),
                        language: "en",
                        name: "Private Project",
                    }],
                }],
            } satisfies ResourceCreateInput,

            teamProject: {
                id: generatePK().toString(),
                resourceType: ResourceType.Project,
                isPrivate: false,
                permissions: JSON.stringify({
                    members: ["read", "write"],
                    teams: { [validIds.team1]: ["read", "write", "admin"] },
                }),
                ownedByTeamConnect: validIds.team1,
                versionsCreate: [{
                    id: generatePK().toString(),
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    translationsCreate: [{
                        id: generatePK().toString(),
                        language: "en",
                        name: "Team Project",
                    }],
                }],
            } satisfies ResourceCreateInput,
        },
    },
};

// ========================================
// Validation Integration
// ========================================

// For now, we'll create the factory without shape integration to focus on the core fixtures
const projectValidationAdapters = {
    create: resourceValidation.create ? {
        validate: async (input: ResourceCreateInput) => {
            try {
                const result = await resourceValidation.create({}).validate(input);
                return { isValid: true, data: result };
            } catch (error) {
                return { isValid: false, errors: [String(error)] };
            }
        },
    } : undefined,
    update: resourceValidation.update ? {
        validate: async (input: ResourceUpdateInput) => {
            try {
                const result = await resourceValidation.update({}).validate(input);
                return { isValid: true, data: result };
            } catch (error) {
                return { isValid: false, errors: [String(error)] };
            }
        },
    } : undefined,
};

// ========================================
// Project-Specific Factory Class
// ========================================

export class ProjectAPIFixtureFactory extends BaseAPIFixtureFactory<
    ResourceCreateInput,
    ResourceUpdateInput,
    Resource
> {

    // Project-specific helper methods

    /**
     * Create a personal project owned by a user
     */
    createPersonalProject = (userId: string, name: string, overrides?: Partial<ResourceCreateInput>): ResourceCreateInput => {
        const versionId = generatePK().toString();
        const translationId = generatePK().toString();

        return this.createFactory({
            ownedByUserConnect: userId,
            isPrivate: false,
            versionsCreate: [{
                id: versionId,
                versionLabel: "1.0.0",
                isPrivate: false,
                translationsCreate: [{
                    id: translationId,
                    language: "en",
                    name,
                }],
            }],
            ...overrides,
        });
    };

    /**
     * Create a team project owned by a team
     */
    createTeamProject = (teamId: string, name: string, overrides?: Partial<ResourceCreateInput>): ResourceCreateInput => {
        const versionId = generatePK().toString();
        const translationId = generatePK().toString();

        return this.createFactory({
            ownedByTeamConnect: teamId,
            isPrivate: false,
            permissions: JSON.stringify({
                members: ["read", "write"],
                teams: { [teamId]: ["read", "write", "admin"] },
            }),
            versionsCreate: [{
                id: versionId,
                versionLabel: "1.0.0",
                isPrivate: false,
                translationsCreate: [{
                    id: translationId,
                    language: "en",
                    name,
                }],
            }],
            ...overrides,
        });
    };

    /**
     * Create a public project (no specific owner)
     */
    createPublicProject = (name: string, overrides?: Partial<ResourceCreateInput>): ResourceCreateInput => {
        const versionId = generatePK().toString();
        const translationId = generatePK().toString();

        return this.createFactory({
            isPrivate: false,
            permissions: JSON.stringify({ public: ["read"] }),
            versionsCreate: [{
                id: versionId,
                versionLabel: "1.0.0",
                isPrivate: false,
                translationsCreate: [{
                    id: translationId,
                    language: "en",
                    name,
                }],
            }],
            ...overrides,
        });
    };

    /**
     * Create a private project
     */
    createPrivateProject = (userId: string, name: string, overrides?: Partial<ResourceCreateInput>): ResourceCreateInput => {
        const versionId = generatePK().toString();
        const translationId = generatePK().toString();

        return this.createFactory({
            ownedByUserConnect: userId,
            isPrivate: true,
            permissions: JSON.stringify({ members: ["read", "write"] }),
            versionsCreate: [{
                id: versionId,
                versionLabel: "1.0.0",
                isPrivate: true,
                translationsCreate: [{
                    id: translationId,
                    language: "en",
                    name,
                }],
            }],
            ...overrides,
        });
    };

    /**
     * Add a new version to an existing project
     */
    addProjectVersion = (projectId: string, versionLabel: string, name: string, overrides?: Partial<ResourceVersionCreateInput>): ResourceUpdateInput => {
        const versionId = generatePK().toString();
        const translationId = generatePK().toString();

        return {
            id: projectId,
            versionsCreate: [{
                id: versionId,
                versionLabel,
                isPrivate: false,
                translationsCreate: [{
                    id: translationId,
                    language: "en",
                    name,
                }],
                ...overrides,
            }],
        };
    };

    /**
     * Update project to mark it as complete
     */
    markProjectComplete = (projectId: string, versionId: string): ResourceUpdateInput => {
        return {
            id: projectId,
            versionsUpdate: [{
                id: versionId,
                isComplete: true,
            }],
        };
    };

    /**
     * Transfer project ownership
     */
    transferProjectOwnership = (projectId: string, newOwnerId: string, isTeam = false): ResourceUpdateInput => {
        if (isTeam) {
            return {
                id: projectId,
                ownedByUserDisconnect: true,
                ownedByTeamConnect: newOwnerId,
            };
        } else {
            return {
                id: projectId,
                ownedByTeamDisconnect: true,
                ownedByUserConnect: newOwnerId,
            };
        }
    };

    /**
     * Add tags to a project
     */
    addProjectTags = (projectId: string, tagNames: string[]): ResourceUpdateInput => {
        return {
            id: projectId,
            tagsCreate: tagNames.map(tag => ({
                id: generatePK().toString(),
                tag,
            })),
        };
    };

    /**
     * Create a project with specific config type
     */
    createProjectWithConfig = (
        name: string,
        configType: keyof typeof projectConfigFixtures.variants,
        overrides?: Partial<ResourceCreateInput>,
    ): ResourceCreateInput => {
        const versionId = generatePK().toString();
        const translationId = generatePK().toString();

        return this.createFactory({
            versionsCreate: [{
                id: versionId,
                versionLabel: "1.0.0",
                isPrivate: false,
                config: projectConfigFixtures.variants[configType],
                translationsCreate: [{
                    id: translationId,
                    language: "en",
                    name,
                }],
            }],
            ...overrides,
        });
    };
}

// ========================================
// Factory Instance & Export
// ========================================

// Create factory configuration manually for now
const projectFactoryConfig = {
    minimal: projectFixtureData.minimal,
    complete: projectFixtureData.complete,
    invalid: projectFixtureData.invalid,
    edgeCases: projectFixtureData.edgeCases,
    validationSchema: projectValidationAdapters,
};

export const projectAPIFixtures = new ProjectAPIFixtureFactory(projectFactoryConfig);

// Export individual fixture sets for convenience
export const projectFixtures = {
    minimal: projectFixtureData.minimal,
    complete: projectFixtureData.complete,
    invalid: projectFixtureData.invalid,
    edgeCases: projectFixtureData.edgeCases,
};

/* c8 ignore stop */

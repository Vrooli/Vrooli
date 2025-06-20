import { generatePK, generatePublicId, nanoid, ResourceType } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface ProjectRelationConfig extends RelationConfig {
    owner?: { userId?: string; teamId?: string };
    versions?: Array<{
        versionLabel: string;
        isLatest?: boolean;
        isComplete?: boolean;
        isPrivate?: boolean;
        translations?: Array<{ language: string; name: string; description?: string; instructions?: string }>;
    }>;
    tags?: string[];
    translations?: Array<{ language: string; name: string; description?: string }>;
}

/**
 * Database fixture factory for Project model (implemented as Resource with Project subtype)
 * Handles projects with versions, translations, and ownership
 * 
 * NOTE: This is a placeholder implementation using the Resource model
 * When the actual Project model is added to the schema, update this factory accordingly
 */
export class ProjectDbFactory extends DatabaseFixtureFactory<
    Prisma.resource,
    Prisma.resourceCreateInput,
    Prisma.resourceInclude,
    Prisma.resourceUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('Resource', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.resource;
    }

    protected getMinimalData(overrides?: Partial<Prisma.resourceCreateInput>): Prisma.resourceCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            resourceType: ResourceType.Project, // Assuming Project is added as a ResourceType
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.resourceCreateInput>): Prisma.resourceCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            resourceType: ResourceType.Project,
            translations: {
                create: [
                    {
                        id: generatePK(),
                        language: "en",
                        name: "Complete Test Project",
                        description: "A comprehensive test project with all features",
                        instructions: "Follow these steps to use this project",
                    },
                    {
                        id: generatePK(),
                        language: "es",
                        name: "Proyecto de Prueba Completo",
                        description: "Un proyecto de prueba integral con todas las funcionalidades",
                        instructions: "Sigue estos pasos para usar este proyecto",
                    },
                ],
            },
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.resourceInclude {
        return {
            translations: true,
            owner: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
            team: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
            versions: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                    isLatest: true,
                    isComplete: true,
                    isPrivate: true,
                    createdAt: true,
                },
                orderBy: {
                    versionIndex: 'desc',
                },
            },
            tags: {
                select: {
                    tag: {
                        select: {
                            id: true,
                            tag: true,
                        },
                    },
                },
            },
            _count: {
                select: {
                    versions: true,
                    bookmarks: true,
                    views: true,
                    votes: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.resourceCreateInput,
        config: ProjectRelationConfig,
        tx: any
    ): Promise<Prisma.resourceCreateInput> {
        let data = { ...baseData };

        // Handle owner (user or team)
        if (config.owner) {
            if (config.owner.userId) {
                data.owner = {
                    connect: { id: config.owner.userId },
                };
            } else if (config.owner.teamId) {
                data.team = {
                    connect: { id: config.owner.teamId },
                };
            }
        }

        // Handle versions (would create ResourceVersions)
        if (config.versions && Array.isArray(config.versions)) {
            data.versions = {
                create: config.versions.map((version, index) => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    versionLabel: version.versionLabel,
                    versionIndex: index + 1,
                    isLatest: version.isLatest ?? (index === config.versions!.length - 1),
                    isComplete: version.isComplete ?? true,
                    isPrivate: version.isPrivate ?? false,
                    translations: version.translations ? {
                        create: version.translations.map(trans => ({
                            id: generatePK(),
                            ...trans,
                        })),
                    } : undefined,
                })),
            };
        }

        // Handle tags
        if (config.tags && Array.isArray(config.tags)) {
            data.tags = {
                create: config.tags.map(tagName => ({
                    id: generatePK(),
                    tag: {
                        connectOrCreate: {
                            where: { tag: tagName },
                            create: {
                                id: generatePK(),
                                tag: tagName,
                            },
                        },
                    },
                })),
            };
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: generatePK(),
                    ...trans,
                })),
            };
        }

        return data;
    }

    /**
     * Create a private project
     */
    async createPrivateProject(): Promise<Prisma.resource> {
        return this.createWithRelations({
            overrides: {
                isPrivate: true,
                resourceType: ResourceType.Project,
            },
            translations: [
                {
                    language: "en",
                    name: "Private Project",
                    description: "A private project for internal use",
                },
            ],
        });
    }

    /**
     * Create a public project with multiple versions
     */
    async createPublicProjectWithVersions(): Promise<Prisma.resource> {
        return this.createWithRelations({
            overrides: {
                isPrivate: false,
                resourceType: ResourceType.Project,
            },
            versions: [
                {
                    versionLabel: "1.0.0",
                    isComplete: true,
                    isLatest: false,
                    translations: [
                        {
                            language: "en",
                            name: "Project v1.0.0",
                            description: "Initial release",
                            instructions: "Getting started guide",
                        },
                    ],
                },
                {
                    versionLabel: "1.1.0",
                    isComplete: true,
                    isLatest: false,
                    translations: [
                        {
                            language: "en",
                            name: "Project v1.1.0",
                            description: "Feature updates",
                            instructions: "Updated instructions",
                        },
                    ],
                },
                {
                    versionLabel: "2.0.0",
                    isComplete: true,
                    isLatest: true,
                    translations: [
                        {
                            language: "en",
                            name: "Project v2.0.0",
                            description: "Major release",
                            instructions: "Latest instructions",
                        },
                    ],
                },
            ],
            tags: ["productivity", "automation", "workflow"],
            translations: [
                {
                    language: "en",
                    name: "Public Project",
                    description: "A public project for community use",
                },
            ],
        });
    }

    /**
     * Create a team-owned project
     */
    async createTeamProject(teamId: string): Promise<Prisma.resource> {
        return this.createWithRelations({
            owner: { teamId },
            overrides: {
                isPrivate: false,
                resourceType: ResourceType.Project,
            },
            versions: [
                {
                    versionLabel: "1.0.0",
                    isLatest: true,
                    isComplete: true,
                    translations: [
                        {
                            language: "en",
                            name: "Team Project v1.0.0",
                            description: "Team collaborative project",
                            instructions: "Team usage instructions",
                        },
                    ],
                },
            ],
            translations: [
                {
                    language: "en",
                    name: "Team Project",
                    description: "A project managed by a team",
                },
            ],
        });
    }

    protected async checkModelConstraints(record: Prisma.resource): Promise<string[]> {
        const violations: string[] = [];
        
        // Check that resource type is Project
        if (record.resourceType !== ResourceType.Project) {
            violations.push('Resource must be of type Project');
        }

        // Check that project has at least one version
        const versions = await this.prisma.resource_version.count({
            where: { resourceId: record.id },
        });
        
        if (versions === 0) {
            violations.push('Project must have at least one version');
        }

        // Check that only one version is marked as latest
        const latestVersions = await this.prisma.resource_version.count({
            where: {
                resourceId: record.id,
                isLatest: true,
            },
        });
        
        if (latestVersions > 1) {
            violations.push('Project can only have one latest version');
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, publicId
                isPrivate: false,
                resourceType: ResourceType.Project,
            },
            invalidTypes: {
                id: "not-a-snowflake",
                publicId: 123, // Should be string
                isPrivate: "yes", // Should be boolean
                resourceType: "INVALID", // Invalid enum
            },
            wrongResourceType: {
                id: generatePK(),
                publicId: generatePublicId(),
                isPrivate: false,
                resourceType: ResourceType.Code, // Wrong type for project
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.resourceCreateInput> {
        return {
            unicodeNameProject: {
                ...this.getCompleteData(),
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "项目 🚀", // Unicode characters
                        description: "Unicode project name",
                    }],
                },
            },
            multiLanguageProject: {
                ...this.getMinimalData(),
                translations: {
                    create: Array.from({ length: 5 }, (_, i) => ({
                        id: generatePK(),
                        language: ['en', 'es', 'fr', 'de', 'ja'][i],
                        name: `Project Name ${i}`,
                        description: `Project description in language ${i}`,
                        instructions: `Project instructions in language ${i}`,
                    })),
                },
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            versions: {
                include: {
                    translations: true,
                },
            },
            translations: true,
            tags: true,
            bookmarks: true,
            views: true,
            votes: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.resource,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // Delete in order of dependencies
        
        // Delete resource versions (cascade will handle their related records)
        if (record.versions?.length) {
            await tx.resource_version.deleteMany({
                where: { resourceId: record.id },
            });
        }

        // Delete bookmarks
        if (record.bookmarks?.length) {
            await tx.bookmark.deleteMany({
                where: { 
                    resourceId: record.id,
                },
            });
        }

        // Delete views
        if (record.views?.length) {
            await tx.view.deleteMany({
                where: { resourceId: record.id },
            });
        }

        // Delete votes/reactions
        if (record.votes?.length) {
            await tx.reaction.deleteMany({
                where: { resourceId: record.id },
            });
        }

        // Delete tag relationships
        if (record.tags?.length) {
            await tx.resource_tag.deleteMany({
                where: { resourceId: record.id },
            });
        }

        // Delete translations
        if (record.translations?.length) {
            await tx.resource_translation.deleteMany({
                where: { resourceId: record.id },
            });
        }
    }
}

// Export factory creator function
export const createProjectDbFactory = (prisma: PrismaClient) => new ProjectDbFactory(prisma);
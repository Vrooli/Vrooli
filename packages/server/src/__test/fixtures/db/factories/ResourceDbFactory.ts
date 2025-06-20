import { generatePK, generatePublicId, ResourceType } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface ResourceRelationConfig extends RelationConfig {
    owner?: { userId?: string; teamId?: string };
    versions?: Array<{
        name: string;
        versionLabel: string;
        isComplete?: boolean;
        isLatest?: boolean;
        link?: string;
        translations?: Array<{ language: string; name: string; description?: string; instructions?: string; details?: string }>;
    }>;
    tags?: string[];
}

/**
 * Database fixture factory for Resource model
 * Handles external links, documentation, tutorials, and versioned content
 */
export class ResourceDbFactory extends DatabaseFixtureFactory<
    Prisma.Resource,
    Prisma.ResourceCreateInput,
    Prisma.ResourceInclude,
    Prisma.ResourceUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('Resource', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.resource;
    }

    protected getMinimalData(overrides?: Partial<Prisma.ResourceCreateInput>): Prisma.ResourceCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            resourceType: ResourceType.Code,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.ResourceCreateInput>): Prisma.ResourceCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            isInternal: false,
            resourceType: ResourceType.Routine,
            permissions: JSON.stringify({
                canEdit: ["Owner", "Admin"],
                canView: ["Public"],
                canDelete: ["Owner"],
            }),
            versions: {
                create: [{
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isComplete: true,
                    isLatest: true,
                    isPrivate: false,
                    versionLabel: "1.0.0",
                    versionNotes: "Initial version",
                    complexity: 5,
                    link: "https://example.com/resource",
                    translations: {
                        create: [
                            {
                                id: generatePK(),
                                language: "en",
                                name: "Complete Resource",
                                description: "A comprehensive resource with all features",
                                instructions: "Follow these steps to use the resource",
                                details: "Detailed information about the resource",
                            },
                            {
                                id: generatePK(),
                                language: "es",
                                name: "Recurso Completo",
                                description: "Un recurso integral con todas las características",
                                instructions: "Sigue estos pasos para usar el recurso",
                                details: "Información detallada sobre el recurso",
                            },
                        ],
                    },
                }],
            },
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.ResourceInclude {
        return {
            versions: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                    isComplete: true,
                    isLatest: true,
                    isPrivate: true,
                    link: true,
                    translations: true,
                },
            },
            ownedByUser: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
            ownedByTeam: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
            tags: true,
            _count: {
                select: {
                    versions: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.ResourceCreateInput,
        config: ResourceRelationConfig,
        tx: any
    ): Promise<Prisma.ResourceCreateInput> {
        let data = { ...baseData };

        // Handle owner
        if (config.owner) {
            if (config.owner.userId) {
                data.ownedByUser = { connect: { id: config.owner.userId } };
            } else if (config.owner.teamId) {
                data.ownedByTeam = { connect: { id: config.owner.teamId } };
            }
        }

        // Handle versions
        if (config.versions && Array.isArray(config.versions)) {
            data.versions = {
                create: config.versions.map(version => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isComplete: version.isComplete ?? false,
                    isLatest: version.isLatest ?? false,
                    isPrivate: false,
                    versionLabel: version.versionLabel,
                    complexity: 1,
                    link: version.link,
                    translations: version.translations ? {
                        create: version.translations.map(trans => ({
                            id: generatePK(),
                            ...trans,
                        })),
                    } : {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: version.name,
                            description: `Version ${version.versionLabel} of ${version.name}`,
                        }],
                    },
                })),
            };
        }

        // Handle tags
        if (config.tags && Array.isArray(config.tags)) {
            data.tags = {
                connectOrCreate: config.tags.map(tag => ({
                    where: { tag },
                    create: {
                        id: generatePK(),
                        tag,
                        translations: {
                            create: [{
                                id: generatePK(),
                                language: "en",
                                description: `Tag for ${tag}`,
                            }],
                        },
                    },
                })),
            };
        }

        return data;
    }

    /**
     * Create documentation resource
     */
    async createDocumentation(ownerId: string): Promise<Prisma.Resource> {
        return this.createWithRelations({
            owner: { userId: ownerId },
            versions: [{
                name: "API Documentation",
                versionLabel: "1.0.0",
                isComplete: true,
                isLatest: true,
                link: "https://docs.example.com/api",
                translations: [{
                    language: "en",
                    name: "API Documentation",
                    description: "Complete API reference and guides",
                    instructions: "Navigate through the sections to find the information you need",
                    details: "This documentation covers all API endpoints, authentication, and best practices",
                }],
            }],
            tags: ["documentation", "api", "reference"],
            overrides: {
                resourceType: ResourceType.Documentation,
            },
        });
    }

    /**
     * Create video tutorial resource
     */
    async createVideoTutorial(ownerId: string): Promise<Prisma.Resource> {
        return this.createWithRelations({
            owner: { userId: ownerId },
            versions: [{
                name: "Getting Started Tutorial",
                versionLabel: "1.0.0",
                isComplete: true,
                isLatest: true,
                link: "https://videos.example.com/getting-started",
                translations: [{
                    language: "en",
                    name: "Getting Started Tutorial",
                    description: "Video walkthrough for beginners",
                    instructions: "Watch the video and follow along with the examples",
                    details: "Duration: 15 minutes. Covers basic setup and first steps.",
                }],
            }],
            tags: ["tutorial", "video", "beginner"],
            overrides: {
                resourceType: ResourceType.Tutorial,
            },
        });
    }

    /**
     * Create external tool resource
     */
    async createExternalTool(teamId: string): Promise<Prisma.Resource> {
        return this.createWithRelations({
            owner: { teamId },
            versions: [{
                name: "Code Formatter Tool",
                versionLabel: "2.3.0",
                isComplete: true,
                isLatest: true,
                link: "https://tools.example.com/formatter",
                translations: [{
                    language: "en",
                    name: "Code Formatter Tool",
                    description: "External tool for formatting code",
                    instructions: "Paste your code and select formatting options",
                    details: "Supports JavaScript, TypeScript, Python, and more",
                }],
            }],
            tags: ["tool", "formatter", "external"],
            overrides: {
                resourceType: ResourceType.ExternalTool,
                isInternal: false,
            },
        });
    }

    /**
     * Create resource with multiple versions
     */
    async createWithVersionHistory(ownerId: string): Promise<Prisma.Resource> {
        return this.createWithRelations({
            owner: { userId: ownerId },
            versions: [
                {
                    name: "Project Template v1",
                    versionLabel: "1.0.0",
                    isComplete: true,
                    isLatest: false,
                    link: "https://github.com/example/template/releases/v1.0.0",
                    translations: [{
                        language: "en",
                        name: "Project Template v1",
                        description: "Initial version of the project template",
                    }],
                },
                {
                    name: "Project Template v1.1",
                    versionLabel: "1.1.0",
                    isComplete: true,
                    isLatest: false,
                    link: "https://github.com/example/template/releases/v1.1.0",
                    translations: [{
                        language: "en",
                        name: "Project Template v1.1",
                        description: "Bug fixes and minor improvements",
                    }],
                },
                {
                    name: "Project Template v2",
                    versionLabel: "2.0.0",
                    isComplete: true,
                    isLatest: true,
                    link: "https://github.com/example/template/releases/v2.0.0",
                    translations: [{
                        language: "en",
                        name: "Project Template v2",
                        description: "Major update with new features and breaking changes",
                        instructions: "See migration guide for upgrading from v1.x",
                    }],
                },
            ],
            tags: ["template", "project", "github"],
            overrides: {
                resourceType: ResourceType.Code,
            },
        });
    }

    /**
     * Create private internal resource
     */
    async createPrivateInternal(teamId: string): Promise<Prisma.Resource> {
        return this.createWithRelations({
            owner: { teamId },
            versions: [{
                name: "Internal Development Guide",
                versionLabel: "1.0.0",
                isComplete: true,
                isLatest: true,
                link: "https://internal.example.com/dev-guide",
                translations: [{
                    language: "en",
                    name: "Internal Development Guide",
                    description: "Confidential development guidelines for team members",
                    instructions: "This guide is for internal use only. Do not share externally.",
                }],
            }],
            overrides: {
                isPrivate: true,
                isInternal: true,
                resourceType: ResourceType.Documentation,
                permissions: JSON.stringify({
                    canEdit: ["TeamOwner", "TeamAdmin"],
                    canView: ["TeamMember"],
                    canDelete: ["TeamOwner"],
                }),
            },
        });
    }

    protected async checkModelConstraints(record: Prisma.Resource): Promise<string[]> {
        const violations: string[] = [];

        // Check that resource has at least one version
        const versionCount = await this.prisma.resourceVersion.count({
            where: { rootId: record.id },
        });

        if (versionCount === 0) {
            violations.push('Resource must have at least one version');
        }

        // Check that there's exactly one latest version
        const latestVersions = await this.prisma.resourceVersion.count({
            where: {
                rootId: record.id,
                isLatest: true,
            },
        });

        if (latestVersions !== 1) {
            violations.push('Resource must have exactly one latest version');
        }

        // Check ownership
        if (!record.ownedByUserId && !record.ownedByTeamId) {
            violations.push('Resource must have an owner (user or team)');
        }

        if (record.ownedByUserId && record.ownedByTeamId) {
            violations.push('Resource cannot be owned by both user and team');
        }

        // Check private resource has owner
        if (record.isPrivate && !record.ownedByUserId && !record.ownedByTeamId) {
            violations.push('Private resource must have an owner');
        }

        // Check internal resources are team-owned
        if (record.isInternal && !record.ownedByTeamId) {
            violations.push('Internal resources should be team-owned');
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing required resourceType
                isPrivate: false,
            },
            invalidTypes: {
                id: "not-a-snowflake",
                publicId: 123, // Should be string
                isPrivate: "yes", // Should be boolean
                isInternal: "no", // Should be boolean
                resourceType: "InvalidType", // Not a valid ResourceType
                permissions: { invalid: "object" }, // Should be string
            },
            bothOwners: {
                id: generatePK(),
                publicId: generatePublicId(),
                isPrivate: false,
                resourceType: ResourceType.Code,
                ownedByUser: { connect: { id: "user123" } },
                ownedByTeam: { connect: { id: "team123" } }, // Can't have both
            },
            invalidResourceType: {
                id: generatePK(),
                publicId: generatePublicId(),
                isPrivate: false,
                resourceType: "NotAValidType" as any,
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.ResourceCreateInput> {
        return {
            maxComplexityResource: {
                ...this.getMinimalData(),
                resourceType: ResourceType.SmartContract,
                versions: {
                    create: [{
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isComplete: true,
                        isLatest: true,
                        isPrivate: false,
                        versionLabel: "1.0.0",
                        complexity: 10, // Maximum complexity
                        link: "https://complex.example.com",
                        translations: {
                            create: [{
                                id: generatePK(),
                                language: "en",
                                name: "Maximum Complexity Resource",
                                description: "A resource with the highest possible complexity rating",
                            }],
                        },
                    }],
                },
            },
            multiLanguageResource: {
                ...this.getMinimalData(),
                resourceType: ResourceType.Documentation,
                versions: {
                    create: [{
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isComplete: true,
                        isLatest: true,
                        isPrivate: false,
                        versionLabel: "1.0.0",
                        complexity: 5,
                        link: "https://multilang.example.com",
                        translations: {
                            create: [
                                { id: generatePK(), language: "en", name: "Global Resource", description: "International resource" },
                                { id: generatePK(), language: "es", name: "Recurso Global", description: "Recurso internacional" },
                                { id: generatePK(), language: "fr", name: "Ressource Globale", description: "Ressource internationale" },
                                { id: generatePK(), language: "de", name: "Globale Ressource", description: "Internationale Ressource" },
                                { id: generatePK(), language: "ja", name: "グローバルリソース", description: "国際的なリソース" },
                            ],
                        },
                    }],
                },
            },
            minimalConfigResource: {
                ...this.getMinimalData(),
                resourceType: ResourceType.Link,
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            versions: {
                include: {
                    translations: true,
                    relations: true,
                },
            },
            tags: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.Resource,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // Delete versions and their translations
        if (record.versions?.length) {
            for (const version of record.versions) {
                // Delete version relations
                if (version.relations?.length) {
                    await tx.resourceVersionRelation.deleteMany({
                        where: { resourceVersionId: version.id },
                    });
                }
                
                // Delete version translations
                if (version.translations?.length) {
                    await tx.resourceVersionTranslation.deleteMany({
                        where: { resourceVersionId: version.id },
                    });
                }
            }
            
            // Delete versions
            await tx.resourceVersion.deleteMany({
                where: { rootId: record.id },
            });
        }
    }
}

// Export factory creator function
export const createResourceDbFactory = (prisma: PrismaClient) => new ResourceDbFactory(prisma);
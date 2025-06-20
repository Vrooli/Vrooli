import { generatePK, generatePublicId, ResourceType, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

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
 * Enhanced database fixture factory for Resource model
 * Handles external links, documentation, tutorials, and versioned content
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for various resource types (Code, Documentation, Tutorial, etc.)
 * - Version management with ResourceVersion
 * - Owner relationships (user or team)
 * - Tag associations
 * - Predefined test scenarios
 * - Comprehensive validation
 */
export class ResourceDbFactory extends EnhancedDatabaseFactory<
    Prisma.Resource,
    Prisma.ResourceCreateInput,
    Prisma.ResourceInclude,
    Prisma.ResourceUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('Resource', prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.resource;
    }

    /**
     * Get complete test fixtures for Resource model
     */
    protected getFixtures(): DbTestFixtures<Prisma.ResourceCreateInput, Prisma.ResourceUpdateInput> {
        return {
            minimal: {
                id: generatePK().toString(),
                publicId: generatePublicId(),
                isPrivate: false,
                resourceType: ResourceType.Code,
            },
            complete: {
                id: generatePK().toString(),
                publicId: generatePublicId(),
                isPrivate: false,
                isInternal: false,
                resourceType: ResourceType.Documentation,
                permissions: JSON.stringify({
                    canEdit: ["Owner", "Admin"],
                    canView: ["Public"],
                    canDelete: ["Owner"],
                }),
            },
            invalid: {
                missingRequired: {
                    // Missing id, publicId, resourceType
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
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    resourceType: ResourceType.Code,
                    ownedByUser: { connect: { id: "user123" } },
                    ownedByTeam: { connect: { id: "team123" } }, // Can't have both
                },
                invalidResourceType: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    resourceType: "NotAValidType" as any,
                },
            },
            edgeCases: {
                maxComplexityResource: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    resourceType: ResourceType.SmartContract,
                },
                multiLanguageResource: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    resourceType: ResourceType.Documentation,
                },
                minimalConfigResource: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isPrivate: false,
                    resourceType: ResourceType.Link,
                },
                privateInternalResource: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isPrivate: true,
                    isInternal: true,
                    resourceType: ResourceType.Documentation,
                    permissions: JSON.stringify({
                        canEdit: ["TeamOwner", "TeamAdmin"],
                        canView: ["TeamMember"],
                        canDelete: ["TeamOwner"],
                    }),
                },
            },
            updates: {
                minimal: {
                    isPrivate: true,
                },
                complete: {
                    isPrivate: false,
                    isInternal: true,
                    resourceType: ResourceType.Tutorial,
                    permissions: JSON.stringify({
                        canEdit: ["Owner"],
                        canView: ["Member"],
                        canDelete: ["Owner"],
                    }),
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.ResourceCreateInput>): Prisma.ResourceCreateInput {
        return {
            id: generatePK().toString(),
            publicId: generatePublicId(),
            isPrivate: false,
            resourceType: ResourceType.Code,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.ResourceCreateInput>): Prisma.ResourceCreateInput {
        return {
            id: generatePK().toString(),
            publicId: generatePublicId(),
            isPrivate: false,
            isInternal: false,
            resourceType: ResourceType.Documentation,
            permissions: JSON.stringify({
                canEdit: ["Owner", "Admin"],
                canView: ["Public"],
                canDelete: ["Owner"],
            }),
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            documentation: {
                name: "documentation",
                description: "Documentation resource with complete metadata",
                config: {
                    overrides: {
                        resourceType: ResourceType.Documentation,
                    },
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
                },
            },
            videoTutorial: {
                name: "videoTutorial",
                description: "Video tutorial resource for learning",
                config: {
                    overrides: {
                        resourceType: ResourceType.Tutorial,
                    },
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
                },
            },
            externalTool: {
                name: "externalTool",
                description: "External tool resource for development",
                config: {
                    overrides: {
                        resourceType: ResourceType.ExternalTool,
                        isInternal: false,
                    },
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
                },
            },
            privateInternal: {
                name: "privateInternal",
                description: "Private internal resource for team use",
                config: {
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
                },
            },
            versionedResource: {
                name: "versionedResource",
                description: "Resource with multiple versions",
                config: {
                    overrides: {
                        resourceType: ResourceType.Code,
                    },
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
                },
            },
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
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    isComplete: version.isComplete ?? false,
                    isLatest: version.isLatest ?? false,
                    isPrivate: false,
                    versionLabel: version.versionLabel,
                    complexity: 1,
                    link: version.link,
                    translations: version.translations ? {
                        create: version.translations.map(trans => ({
                            id: generatePK().toString(),
                            ...trans,
                        })),
                    } : {
                        create: [{
                            id: generatePK().toString(),
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
                        id: generatePK().toString(),
                        tag,
                        translations: {
                            create: [{
                                id: generatePK().toString(),
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
        return this.seedScenario('versionedResource');
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
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete versions and their translations
        if (shouldDelete('versions') && record.versions?.length) {
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
export const createResourceDbFactory = (prisma: PrismaClient) => 
    ResourceDbFactory.getInstance('Resource', prisma);

// Export the class for type usage
export { ResourceDbFactory as ResourceDbFactoryClass };
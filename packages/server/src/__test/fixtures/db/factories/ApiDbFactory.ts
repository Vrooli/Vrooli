import { generatePK, generatePublicId, nanoid } from "../idHelpers.js";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface ApiRelationConfig extends RelationConfig {
    owner?: { userId?: string; teamId?: string };
    versions?: Array<{
        versionLabel: string;
        isLatest?: boolean;
        isComplete?: boolean;
        isPrivate?: boolean;
        apiType?: "REST" | "GraphQL" | "WebSocket" | "gRPC";
        translations?: Array<{ language: string; name: string; description?: string; summary?: string }>;
    }>;
    tags?: string[];
}

/**
 * Database fixture factory for Api model
 * Handles API definitions with versions, documentation, and schemas
 */
export class ApiDbFactory extends DatabaseFixtureFactory<
    Prisma.resource,
    Prisma.resourceCreateInput,
    Prisma.resourceInclude,
    Prisma.resourceUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("Api", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.resource;
    }

    protected getMinimalData(overrides?: Partial<Prisma.resourceCreateInput>): Prisma.resourceCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            resourceType: "Api",
            isPrivate: false,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.resourceCreateInput>): Prisma.resourceCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            resourceType: "Api",
            isPrivate: false,
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.resourceInclude {
        return {
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
            versions: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                    isLatest: true,
                    isComplete: true,
                    isPrivate: true,
                    createdAt: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                            description: true,
                        },
                    },
                },
                orderBy: {
                    versionIndex: "desc",
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
                    bookmarkedBy: true,
                    viewedBy: true,
                    reactions: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.resourceCreateInput,
        config: ApiRelationConfig,
        tx: any,
    ): Promise<Prisma.resourceCreateInput> {
        const data = { ...baseData };

        // Handle owner (user or team)
        if (config.owner) {
            if (config.owner.userId) {
                data.ownedByUser = {
                    connect: { id: config.owner.userId },
                };
            } else if (config.owner.teamId) {
                data.ownedByTeam = {
                    connect: { id: config.owner.teamId },
                };
            }
        }

        // Handle versions
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
                    config: {
                        apiType: version.apiType ?? "REST",
                    },
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

        // Note: Translations are handled at the version level, not resource level

        return data;
    }

    /**
     * Create a REST API
     */
    async createRestApi(): Promise<Prisma.resource> {
        return this.createWithRelations({
            overrides: {
                resourceType: "Api",
                isPrivate: false,
            },
            versions: [
                {
                    versionLabel: "1.0.0",
                    isLatest: true,
                    isComplete: true,
                    apiType: "REST",
                    translations: [
                        {
                            language: "en",
                            name: "REST API v1.0.0",
                            description: "RESTful API for data operations",
                            summary: "Complete REST API with CRUD operations",
                        },
                    ],
                },
            ],
            tags: ["rest", "api", "crud", "http"],
        });
    }

    /**
     * Create a GraphQL API
     */
    async createGraphQLApi(): Promise<Prisma.resource> {
        return this.createWithRelations({
            overrides: {
                resourceType: "Api",
                isPrivate: false,
            },
            versions: [
                {
                    versionLabel: "1.0.0",
                    isLatest: true,
                    isComplete: true,
                    apiType: "GraphQL",
                    translations: [
                        {
                            language: "en",
                            name: "GraphQL API v1.0.0",
                            description: "GraphQL API with flexible queries",
                            summary: "Modern GraphQL API with type safety",
                        },
                    ],
                },
            ],
            tags: ["graphql", "api", "query", "mutation", "subscription"],
        });
    }

    /**
     * Create a private API
     */
    async createPrivateApi(): Promise<Prisma.resource> {
        return this.createWithRelations({
            overrides: {
                resourceType: "Api",
                isPrivate: true,
            },
            versions: [
                {
                    versionLabel: "0.1.0-alpha",
                    isLatest: true,
                    isComplete: false,
                    isPrivate: true,
                    apiType: "REST",
                    translations: [
                        {
                            language: "en",
                            name: "Private API Alpha",
                            description: "Internal API under development",
                            summary: "Private alpha version for internal testing",
                        },
                    ],
                },
            ],
            tags: ["private", "internal", "alpha"],
        });
    }

    /**
     * Create an API with multiple versions
     */
    async createVersionedApi(): Promise<Prisma.resource> {
        return this.createWithRelations({
            overrides: {
                resourceType: "Api",
                isPrivate: false,
            },
            versions: [
                {
                    versionLabel: "1.0.0",
                    isLatest: false,
                    isComplete: true,
                    apiType: "REST",
                    translations: [
                        {
                            language: "en",
                            name: "API v1.0.0",
                            description: "Initial stable release",
                            summary: "First stable version",
                        },
                    ],
                },
                {
                    versionLabel: "1.1.0",
                    isLatest: false,
                    isComplete: true,
                    apiType: "REST",
                    translations: [
                        {
                            language: "en",
                            name: "API v1.1.0",
                            description: "Minor updates and bug fixes",
                            summary: "Incremental improvements",
                        },
                    ],
                },
                {
                    versionLabel: "2.0.0",
                    isLatest: true,
                    isComplete: true,
                    apiType: "REST",
                    translations: [
                        {
                            language: "en",
                            name: "API v2.0.0",
                            description: "Major version with breaking changes",
                            summary: "Complete redesign with new features",
                        },
                    ],
                },
            ],
            tags: ["versioned", "stable", "production"],
        });
    }

    /**
     * Create a team-owned API
     */
    async createTeamApi(teamId: string): Promise<Prisma.resource> {
        return this.createWithRelations({
            owner: { teamId },
            overrides: {
                resourceType: "Api",
                isPrivate: false,
            },
            versions: [
                {
                    versionLabel: "1.0.0",
                    isLatest: true,
                    isComplete: true,
                    apiType: "REST",
                    translations: [
                        {
                            language: "en",
                            name: "Team API v1.0.0",
                            description: "API developed by the team",
                            summary: "Collaborative API development",
                        },
                    ],
                },
            ],
            tags: ["team", "collaborative"],
        });
    }

    protected async checkModelConstraints(record: Prisma.resource): Promise<string[]> {
        const violations: string[] = [];
        
        // Check that API has at least one version
        const versions = await this.prisma.resource_version.count({
            where: { rootId: record.id },
        });
        
        if (versions === 0) {
            violations.push("API must have at least one version");
        }

        // Check that only one version is marked as latest
        const latestVersions = await this.prisma.resource_version.count({
            where: {
                rootId: record.id,
                isLatest: true,
            },
        });
        
        if (latestVersions > 1) {
            violations.push("API can only have one latest version");
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, publicId, resourceType
                isPrivate: false,
            },
            invalidTypes: {
                id: "not-a-snowflake",
                publicId: 123, // Should be string
                resourceType: true, // Should be string
                isPrivate: "yes", // Should be boolean
            },
            invalidResourceType: {
                id: generatePK(),
                publicId: generatePublicId(),
                resourceType: "InvalidType", // Invalid resource type
                isPrivate: false,
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.resourceCreateInput> {
        return {
            minimalApi: {
                ...this.getMinimalData(),
            },
            privateApi: {
                ...this.getMinimalData(),
                isPrivate: true,
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            versions: {
                include: {
                    translations: true,
                    comments: true,
                    reports: true,
                },
            },
            tags: true,
            bookmarkedBy: true,
            viewedBy: true,
            reactions: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.resource,
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // Delete in order of dependencies
        
        // Delete resource versions (cascade will handle their related records)
        if (record.versions?.length) {
            await tx.resource_version.deleteMany({
                where: { rootId: record.id },
            });
        }

        // Delete bookmarks
        if (record.bookmarkedBy?.length) {
            await tx.bookmark.deleteMany({
                where: { 
                    resourceId: record.id,
                },
            });
        }

        // Delete views
        if (record.viewedBy?.length) {
            await tx.view.deleteMany({
                where: { resourceId: record.id },
            });
        }

        // Delete votes/reactions
        if (record.reactions?.length) {
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

        // Note: Translations are handled at the version level, not resource level
    }
}

// Export factory creator function
export const createApiDbFactory = (prisma: PrismaClient) => new ApiDbFactory(prisma);

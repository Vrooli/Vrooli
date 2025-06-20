import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";
import { apiConfigFixtures } from "@vrooli/shared/__test/fixtures/config";

interface ApiRelationConfig extends RelationConfig {
    owner?: { userId?: string; teamId?: string };
    versions?: Array<{
        versionLabel: string;
        isLatest?: boolean;
        isComplete?: boolean;
        isPrivate?: boolean;
        apiType?: 'REST' | 'GraphQL' | 'WebSocket' | 'gRPC';
        translations?: Array<{ language: string; name: string; description?: string; summary?: string }>;
    }>;
    tags?: string[];
    translations?: Array<{ language: string; name: string; description?: string }>;
}

/**
 * Database fixture factory for Api model
 * Handles API definitions with versions, documentation, and schemas
 */
export class ApiDbFactory extends DatabaseFixtureFactory<
    Prisma.Api,
    Prisma.ApiCreateInput,
    Prisma.ApiInclude,
    Prisma.ApiUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('Api', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.api;
    }

    protected getMinimalData(overrides?: Partial<Prisma.ApiCreateInput>): Prisma.ApiCreateInput {
        const uniqueHandle = `api_${nanoid(8)}`;
        
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            handle: uniqueHandle,
            isPrivate: false,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.ApiCreateInput>): Prisma.ApiCreateInput {
        const uniqueHandle = `complete_api_${nanoid(8)}`;
        
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            handle: uniqueHandle,
            isPrivate: false,
            config: apiConfigFixtures.complete,
            translations: {
                create: [
                    {
                        id: generatePK(),
                        language: "en",
                        name: "Complete Test API",
                        description: "A comprehensive test API with all features",
                        summary: "Complete API for testing purposes",
                    },
                    {
                        id: generatePK(),
                        language: "es",
                        name: "API de Prueba Completa",
                        description: "Una API de prueba integral con todas las funcionalidades",
                        summary: "API completa para propósitos de prueba",
                    },
                ],
            },
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.ApiInclude {
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
                    apiType: true,
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
        baseData: Prisma.ApiCreateInput,
        config: ApiRelationConfig,
        tx: any
    ): Promise<Prisma.ApiCreateInput> {
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
                    apiType: version.apiType ?? 'REST',
                    config: apiConfigFixtures.create({
                        apiType: version.apiType ?? 'REST',
                    }),
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
     * Create a REST API
     */
    async createRestApi(): Promise<Prisma.Api> {
        return this.createWithRelations({
            overrides: {
                handle: `rest_api_${nanoid(8)}`,
                isPrivate: false,
            },
            versions: [
                {
                    versionLabel: "1.0.0",
                    isLatest: true,
                    isComplete: true,
                    apiType: 'REST',
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
            translations: [
                {
                    language: "en",
                    name: "REST API",
                    description: "A RESTful API for web services",
                },
            ],
        });
    }

    /**
     * Create a GraphQL API
     */
    async createGraphQLApi(): Promise<Prisma.Api> {
        return this.createWithRelations({
            overrides: {
                handle: `graphql_api_${nanoid(8)}`,
                isPrivate: false,
            },
            versions: [
                {
                    versionLabel: "1.0.0",
                    isLatest: true,
                    isComplete: true,
                    apiType: 'GraphQL',
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
            translations: [
                {
                    language: "en",
                    name: "GraphQL API",
                    description: "A GraphQL API for flexible data queries",
                },
            ],
        });
    }

    /**
     * Create a private API
     */
    async createPrivateApi(): Promise<Prisma.Api> {
        return this.createWithRelations({
            overrides: {
                isPrivate: true,
                handle: `private_api_${nanoid(8)}`,
            },
            versions: [
                {
                    versionLabel: "0.1.0-alpha",
                    isLatest: true,
                    isComplete: false,
                    isPrivate: true,
                    apiType: 'REST',
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
            translations: [
                {
                    language: "en",
                    name: "Private API",
                    description: "Internal API for private use",
                },
            ],
        });
    }

    /**
     * Create an API with multiple versions
     */
    async createVersionedApi(): Promise<Prisma.Api> {
        return this.createWithRelations({
            overrides: {
                handle: `versioned_api_${nanoid(8)}`,
                isPrivate: false,
            },
            versions: [
                {
                    versionLabel: "1.0.0",
                    isLatest: false,
                    isComplete: true,
                    apiType: 'REST',
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
                    apiType: 'REST',
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
                    apiType: 'REST',
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
            translations: [
                {
                    language: "en",
                    name: "Versioned API",
                    description: "API with multiple stable versions",
                },
            ],
        });
    }

    /**
     * Create a team-owned API
     */
    async createTeamApi(teamId: string): Promise<Prisma.Api> {
        return this.createWithRelations({
            owner: { teamId },
            overrides: {
                handle: `team_api_${nanoid(8)}`,
                isPrivate: false,
            },
            versions: [
                {
                    versionLabel: "1.0.0",
                    isLatest: true,
                    isComplete: true,
                    apiType: 'REST',
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
            translations: [
                {
                    language: "en",
                    name: "Team API",
                    description: "An API managed by a team",
                },
            ],
        });
    }

    protected async checkModelConstraints(record: Prisma.Api): Promise<string[]> {
        const violations: string[] = [];
        
        // Check handle uniqueness
        if (record.handle) {
            const duplicate = await this.prisma.api.findFirst({
                where: { 
                    handle: record.handle,
                    id: { not: record.id },
                },
            });
            if (duplicate) {
                violations.push('Handle must be unique');
            }
        }

        // Check handle format
        if (record.handle && !/^[a-zA-Z0-9_-]+$/.test(record.handle)) {
            violations.push('Handle contains invalid characters');
        }

        // Check that API has at least one version
        const versions = await this.prisma.apiVersion.count({
            where: { apiId: record.id },
        });
        
        if (versions === 0) {
            violations.push('API must have at least one version');
        }

        // Check that only one version is marked as latest
        const latestVersions = await this.prisma.apiVersion.count({
            where: {
                apiId: record.id,
                isLatest: true,
            },
        });
        
        if (latestVersions > 1) {
            violations.push('API can only have one latest version');
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, publicId, handle
                isPrivate: false,
            },
            invalidTypes: {
                id: "not-a-snowflake",
                publicId: 123, // Should be string
                handle: true, // Should be string
                isPrivate: "yes", // Should be boolean
            },
            duplicateHandle: {
                id: generatePK(),
                publicId: generatePublicId(),
                handle: "existing_api_handle", // Assumes this exists
                isPrivate: false,
            },
            invalidConfig: {
                id: generatePK(),
                publicId: generatePublicId(),
                handle: `invalid_config_${nanoid(8)}`,
                isPrivate: false,
                config: { invalid: "config" }, // Invalid config structure
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.ApiCreateInput> {
        return {
            maxLengthHandle: {
                ...this.getMinimalData(),
                handle: 'api_' + 'a'.repeat(45), // Max length handle
            },
            unicodeNameApi: {
                ...this.getCompleteData(),
                handle: `unicode_api_${nanoid(8)}`,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "API 接口 🚀", // Unicode characters
                        description: "Unicode API name",
                        summary: "API with unicode content",
                    }],
                },
            },
            complexConfigApi: {
                ...this.getMinimalData(),
                handle: `complex_config_${nanoid(8)}`,
                config: {
                    ...apiConfigFixtures.complete,
                    rateLimit: {
                        requests: 1000,
                        window: "1h",
                    },
                    authentication: {
                        type: "bearer",
                        required: true,
                    },
                },
            },
            multiLanguageApi: {
                ...this.getMinimalData(),
                handle: `multilang_api_${nanoid(8)}`,
                translations: {
                    create: Array.from({ length: 5 }, (_, i) => ({
                        id: generatePK(),
                        language: ['en', 'es', 'fr', 'de', 'ja'][i],
                        name: `API Name ${i}`,
                        description: `API description in language ${i}`,
                        summary: `API summary in language ${i}`,
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
                    endpoints: true,
                    resourceLists: true,
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
        record: Prisma.Api,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // Delete in order of dependencies
        
        // Delete API versions (cascade will handle their related records)
        if (record.versions?.length) {
            await tx.apiVersion.deleteMany({
                where: { apiId: record.id },
            });
        }

        // Delete bookmarks
        if (record.bookmarks?.length) {
            await tx.bookmark.deleteMany({
                where: { 
                    apiId: record.id,
                },
            });
        }

        // Delete views
        if (record.views?.length) {
            await tx.view.deleteMany({
                where: { apiId: record.id },
            });
        }

        // Delete votes/reactions
        if (record.votes?.length) {
            await tx.reaction.deleteMany({
                where: { apiId: record.id },
            });
        }

        // Delete tag relationships
        if (record.tags?.length) {
            await tx.apiTag.deleteMany({
                where: { apiId: record.id },
            });
        }

        // Delete translations
        if (record.translations?.length) {
            await tx.apiTranslation.deleteMany({
                where: { apiId: record.id },
            });
        }
    }
}

// Export factory creator function
export const createApiDbFactory = (prisma: PrismaClient) => new ApiDbFactory(prisma);
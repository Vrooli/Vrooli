import { generatePK, generatePublicId, nanoid } from "../idHelpers.js";
import { 
    type Prisma, 
    type PrismaClient,
    type resource_version,
    type resource_versionCreateInput
} from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface ApiVersionRelationConfig extends RelationConfig {
    root?: { rootId: string };
    fromRelations?: Array<{
        toVersionId: string;
        relationType: string;
        labels?: string[];
    }>;
    toRelations?: Array<{
        fromVersionId: string;
        relationType: string;
        labels?: string[];
    }>;
    translations?: Array<{ 
        language: string; 
        name: string; 
        description?: string; 
        summary?: string; 
        details?: string;
        instructions?: string;
    }>;
}

/**
 * Database fixture factory for resource_version model (API versions)
 * Handles versioned API content with config, schemas, and documentation
 */
export class ApiVersionDbFactory extends DatabaseFixtureFactory<
    resource_version,
    resource_versionCreateInput,
    Prisma.resource_versionInclude,
    Prisma.resource_versionUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("resource_version", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.resource_version;
    }

    protected getMinimalData(overrides?: Partial<resource_versionCreateInput>): resource_versionCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            rootId: generatePK(),
            versionLabel: "1.0.0",
            versionIndex: 1,
            isLatest: true,
            isComplete: true,
            isPrivate: false,
            config: { apiType: "REST" },
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<resource_versionCreateInput>): resource_versionCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            rootId: generatePK(),
            versionLabel: "1.0.0",
            versionIndex: 1,
            isLatest: true,
            isComplete: true,
            isPrivate: false,
            config: {
                apiType: "REST",
                openapi: "3.0.0",
                info: {
                    title: "Complete Test API",
                    version: "1.0.0",
                    description: "A comprehensive test API",
                },
                paths: {
                    "/users": {
                        get: {
                            summary: "List users",
                            responses: {
                                "200": {
                                    description: "Success",
                                },
                            },
                        },
                        post: {
                            summary: "Create user",
                            responses: {
                                "201": {
                                    description: "Created",
                                },
                            },
                        },
                    },
                },
            },
            translations: {
                create: [
                    {
                        id: generatePK(),
                        language: "en",
                        name: "Complete API Version",
                        description: "A comprehensive API version with all features",
                        summary: "Complete API documentation",
                        details: "Detailed API specifications and usage examples",
                        instructions: "Follow the API documentation for proper usage",
                    },
                    {
                        id: generatePK(),
                        language: "es",
                        name: "Versión Completa de API",
                        description: "Una versión de API integral con todas las funcionalidades",
                        summary: "Documentación completa de API",
                        details: "Especificaciones detalladas de API y ejemplos de uso",
                        instructions: "Siga la documentación de API para el uso adecuado",
                    },
                ],
            },
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.resource_versionInclude {
        return {
            translations: true,
            root: {
                select: {
                    id: true,
                    publicId: true,
                    resourceType: true,
                    isPrivate: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                        },
                    },
                },
            },
            relatedVersions: {
                select: {
                    id: true,
                    relationType: true,
                    labels: true,
                    toVersion: {
                        select: {
                            id: true,
                            publicId: true,
                            versionLabel: true,
                        },
                    },
                },
            },
            referencedBy: {
                select: {
                    id: true,
                    relationType: true,
                    labels: true,
                    fromVersion: {
                        select: {
                            id: true,
                            publicId: true,
                            versionLabel: true,
                        },
                    },
                },
            },
            _count: {
                select: {
                    relatedVersions: true,
                    referencedBy: true,
                    comments: true,
                    reports: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: resource_versionCreateInput,
        config: ApiVersionRelationConfig,
        tx: any,
    ): Promise<resource_versionCreateInput> {
        let data = { ...baseData };

        // Handle root resource connection
        if (config.root) {
            data.root = {
                connect: { id: config.root.rootId },
            };
        }

        // Handle related versions (outgoing relations)
        if (config.fromRelations && Array.isArray(config.fromRelations)) {
            data.relatedVersions = {
                create: config.fromRelations.map(relation => ({
                    id: generatePK(),
                    toVersionId: relation.toVersionId,
                    relationType: relation.relationType,
                    labels: relation.labels ?? [],
                })),
            };
        }

        // Handle referenced by (incoming relations)
        if (config.toRelations && Array.isArray(config.toRelations)) {
            data.referencedBy = {
                create: config.toRelations.map(relation => ({
                    id: generatePK(),
                    fromVersionId: relation.fromVersionId,
                    relationType: relation.relationType,
                    labels: relation.labels ?? [],
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
     * Create a REST API version
     */
    async createRestApiVersion(): Promise<resource_version> {
        return this.createWithRelations({
            overrides: {
                versionLabel: "1.0.0",
                isComplete: true,
                config: {
                    apiType: "REST",
                    openapi: "3.0.0",
                    info: {
                        title: "REST API",
                        version: "1.0.0",
                    },
                    paths: {
                        "/users": {
                            get: { summary: "List users" },
                            post: { summary: "Create user" },
                        },
                        "/users/{id}": {
                            get: { summary: "Get user" },
                            put: { summary: "Update user" },
                            delete: { summary: "Delete user" },
                        },
                    },
                },
            },
            translations: [
                {
                    language: "en",
                    name: "REST API v1.0.0",
                    description: "RESTful API with standard HTTP methods",
                    summary: "Complete REST API for user management",
                    details: "Full CRUD operations with proper HTTP status codes",
                    instructions: "Use standard HTTP methods for operations",
                },
            ],
        });
    }

    /**
     * Create a GraphQL API version
     */
    async createGraphQLApiVersion(): Promise<resource_version> {
        return this.createWithRelations({
            overrides: {
                versionLabel: "1.0.0",
                isComplete: true,
                config: {
                    apiType: "GraphQL",
                    type: "graphql",
                    schema: `
                        type User {
                            id: ID!
                            name: String!
                            email: String!
                        }
                        
                        type Query {
                            users: [User!]!
                            user(id: ID!): User
                        }
                        
                        type Mutation {
                            createUser(input: UserInput!): User!
                            updateUser(id: ID!, input: UserInput!): User!
                            deleteUser(id: ID!): Boolean!
                        }
                        
                        input UserInput {
                            name: String!
                            email: String!
                        }
                    `,
                },
            },
            translations: [
                {
                    language: "en",
                    name: "GraphQL API v1.0.0",
                    description: "GraphQL API with flexible queries",
                    summary: "Modern GraphQL API with type safety",
                    details: "Strongly typed GraphQL schema with introspection",
                    instructions: "Use GraphQL queries and mutations for data operations",
                },
            ],
        });
    }

    /**
     * Create a draft API version
     */
    async createDraftApiVersion(): Promise<resource_version> {
        return this.createWithRelations({
            overrides: {
                versionLabel: "0.1.0-draft",
                isComplete: false,
                isPrivate: true,
                isLatest: false,
                config: { apiType: "REST" },
            },
            translations: [
                {
                    language: "en",
                    name: "Draft API Version",
                    description: "Work in progress API version",
                    summary: "This is a draft version under development",
                    instructions: "This version is still being developed",
                },
            ],
        });
    }

    /**
     * Create a deprecated API version
     */
    async createDeprecatedApiVersion(): Promise<resource_version> {
        return this.createWithRelations({
            overrides: {
                versionLabel: "0.9.0",
                isComplete: true,
                isPrivate: false,
                isLatest: false,
                versionIndex: 1,
                config: { 
                    apiType: "REST",
                    deprecationNotice: "This version is deprecated. Please use v1.0.0 or later."
                },
            },
            translations: [
                {
                    language: "en",
                    name: "Deprecated API Version",
                    description: "This version is deprecated, use a newer version",
                    summary: "Please upgrade to the latest version",
                    instructions: "Migration guide available in documentation",
                },
            ],
        });
    }

    /**
     * Create a version with relations to other versions
     */
    async createRelatedApiVersion(rootId: string, relatedVersionId?: string): Promise<resource_version> {
        const config: any = {
            root: { rootId },
            overrides: {
                versionLabel: "1.1.0-related",
                isComplete: false,
                isPrivate: true,
                isLatest: false,
                versionIndex: 1,
                config: { apiType: "REST" },
            },
            translations: [
                {
                    language: "en",
                    name: "Related API Version",
                    description: "A version with relations to other versions",
                    summary: "Based on related versions with modifications",
                    instructions: "Check relations for dependencies",
                },
            ],
        };

        if (relatedVersionId) {
            config.fromRelations = [{
                toVersionId: relatedVersionId,
                relationType: "SUBROUTINE",
                labels: ["dependency"]
            }];
        }

        return this.createWithRelations(config);
    }

    protected async checkModelConstraints(record: resource_version): Promise<string[]> {
        const violations: string[] = [];
        
        // Check that rootId is valid
        if (record.rootId) {
            const root = await this.prisma.resource.findUnique({
                where: { id: record.rootId },
            });
            if (!root) {
                violations.push("resource_version must belong to a valid resource");
            }
        }

        // Check version label format
        if (record.versionLabel && !/^\d+\.\d+\.\d+(-[\w.-]+)?$/.test(record.versionLabel)) {
            violations.push("Version label must follow semantic versioning format");
        }

        // Check that only one version per resource is marked as latest
        if (record.isLatest && record.rootId) {
            const otherLatestVersions = await this.prisma.resource_version.count({
                where: {
                    rootId: record.rootId,
                    isLatest: true,
                    id: { not: record.id },
                },
            });
            
            if (otherLatestVersions > 0) {
                violations.push("Only one version per resource can be marked as latest");
            }
        }

        // Check that completed versions are not private if resource is public
        if (record.isComplete && !record.isPrivate && record.rootId) {
            const resource = await this.prisma.resource.findUnique({
                where: { id: record.rootId },
                select: { isPrivate: true },
            });
            
            if (resource && !resource.isPrivate && record.isPrivate) {
                violations.push("Complete versions of public resources should not be private");
            }
        }

        // Check config validity for API type
        if (record.config && typeof record.config === "object") {
            const config = record.config as any;
            if (config.apiType && !["REST", "GraphQL", "WebSocket", "gRPC"].includes(config.apiType)) {
                violations.push("API type must be one of: REST, GraphQL, WebSocket, gRPC");
            }

            // Check schema validity based on API type
            if (config.schema && config.apiType === "GraphQL") {
                if (typeof config.schema === "string" && config.schema.includes("type")) {
                    if (!config.schema.includes("type Query") && !config.schema.includes("type Mutation")) {
                        violations.push("GraphQL schema must contain at least Query or Mutation types");
                    }
                }
            }
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, publicId, versionLabel, versionIndex
                isLatest: true,
                isComplete: true,
                isPrivate: false,
                apiType: "REST",
            },
            invalidTypes: {
                id: "not-a-snowflake",
                publicId: 123, // Should be string
                versionLabel: true, // Should be string
                versionIndex: "1", // Should be number
                isLatest: "yes", // Should be boolean
                isComplete: 1, // Should be boolean
                isPrivate: "no", // Should be boolean
                apiType: "INVALID", // Invalid enum value
            },
            invalidVersionFormat: {
                id: generatePK(),
                publicId: generatePublicId(),
                versionLabel: "invalid-version", // Invalid format
                versionIndex: 1,
                isLatest: true,
                isComplete: true,
                isPrivate: false,
                apiType: "REST",
            },
            invalidApiType: {
                id: generatePK(),
                publicId: generatePublicId(),
                versionLabel: "1.0.0",
                versionIndex: 1,
                isLatest: true,
                isComplete: true,
                isPrivate: false,
                apiType: "INVALID_TYPE", // Invalid API type
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, resource_versionCreateInput> {
        return {
            prereleaseVersion: {
                ...this.getMinimalData(),
                versionLabel: "2.0.0-alpha.1",
                isComplete: false,
                isPrivate: true,
            },
            buildMetadataVersion: {
                ...this.getMinimalData(),
                versionLabel: "1.0.0+build.123",
                isComplete: true,
                isPrivate: false,
            },
            webSocketApiVersion: {
                ...this.getMinimalData(),
                config: {
                    apiType: "WebSocket",
                    type: "websocket",
                    events: {
                        connect: { description: "Client connected" },
                        disconnect: { description: "Client disconnected" },
                        message: { description: "Message received" },
                    },
                },
            },
            grpcApiVersion: {
                ...this.getMinimalData(),
                config: {
                    apiType: "gRPC",
                    type: "protobuf",
                    proto: `
                        syntax = "proto3";
                        
                        service UserService {
                            rpc GetUser(GetUserRequest) returns (User);
                            rpc CreateUser(CreateUserRequest) returns (User);
                        }
                        
                        message User {
                            string id = 1;
                            string name = 2;
                            string email = 3;
                        }
                        
                        message GetUserRequest {
                            string id = 1;
                        }
                        
                        message CreateUserRequest {
                            string name = 1;
                            string email = 2;
                        }
                    `,
                },
            },
            unicodeContent: {
                ...this.getMinimalData(),
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "zh",
                        name: "API版本 🚀",
                        description: "使用Unicode字符的API版本",
                        summary: "详细说明如何使用这个API",
                        details: "API的详细信息和功能描述",
                        instructions: "请查看详细文档",
                    }],
                },
            },
            complexSchema: {
                ...this.getMinimalData(),
                config: {
                    apiType: "REST",
                    openapi: "3.0.0",
                    info: {
                        title: "Complex API",
                        version: "1.0.0",
                        description: "A complex API with many endpoints",
                    },
                    paths: Object.fromEntries(
                        Array.from({ length: 20 }, (_, i) => [
                            `/endpoint-${i}`,
                            {
                                get: { summary: `Endpoint ${i}` },
                                post: { summary: `Create ${i}` },
                            },
                        ]),
                    ),
                },
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            fromRelations: {
                include: {
                    toVersion: true,
                },
            },
            toRelations: {
                include: {
                    fromVersion: true,
                },
            },
            bookmarks: true,
            views: true,
            reactions: true,
        };
    }

    protected async deleteRelatedRecords(
        record: resource_version,
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // Delete in order of dependencies
        
        // Delete version relations
        if (record.fromRelations?.length) {
            await tx.resource_version_relation.deleteMany({
                where: { fromVersionId: record.id },
            });
        }

        if (record.toRelations?.length) {
            await tx.resource_version_relation.deleteMany({
                where: { toVersionId: record.id },
            });
        }

        // Delete bookmarks
        if (record.bookmarks?.length) {
            await tx.bookmark.deleteMany({
                where: { 
                    resourceId: record.rootId,
                },
            });
        }

        // Delete views
        if (record.views?.length) {
            await tx.view.deleteMany({
                where: { resourceId: record.rootId },
            });
        }

        // Delete reactions
        if (record.reactions?.length) {
            await tx.reaction.deleteMany({
                where: { resourceId: record.rootId },
            });
        }

        // Delete translations
        if (record.translations?.length) {
            await tx.resource_translation.deleteMany({
                where: { resourceVersionId: record.id },
            });
        }
    }

    /**
     * Create version sequence for a resource
     */
    async createVersionSequence(
        rootId: string,
        versions: Array<{ label: string; isLatest?: boolean; isComplete?: boolean; apiType?: string }>,
    ): Promise<resource_version[]> {
        const createdVersions: resource_version[] = [];
        
        for (let i = 0; i < versions.length; i++) {
            const version = versions[i];
            const resourceVersion = await this.createWithRelations({
                root: { rootId },
                overrides: {
                    versionLabel: version.label,
                    versionIndex: i + 1,
                    isLatest: version.isLatest ?? (i === versions.length - 1),
                    isComplete: version.isComplete ?? true,
                    config: { apiType: version.apiType ?? "REST" },
                },
                translations: [
                    {
                        language: "en",
                        name: `API Version ${version.label}`,
                        description: `Version ${version.label} of the API`,
                        summary: `API v${version.label}`,
                        instructions: `Implementation guide for v${version.label}`,
                    },
                ],
            });
            createdVersions.push(resourceVersion);
        }
        
        return createdVersions;
    }
}

// Export factory creator function
export const createApiVersionDbFactory = (prisma: PrismaClient) => new ApiVersionDbFactory(prisma);

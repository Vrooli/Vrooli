import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";
import { apiConfigFixtures } from "@vrooli/shared/__test/fixtures/config";

interface ApiVersionRelationConfig extends RelationConfig {
    api?: { apiId: string };
    parent?: { parentId: string };
    endpoints?: Array<{
        path: string;
        method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
        description?: string;
        parameters?: any[];
        responses?: any[];
    }>;
    resourceLists?: Array<{
        name: string;
        description?: string;
        isPrivate?: boolean;
    }>;
    translations?: Array<{ language: string; name: string; description?: string; summary?: string; details?: string }>;
}

/**
 * Database fixture factory for ApiVersion model
 * Handles versioned API content with endpoints, schemas, and documentation
 */
export class ApiVersionDbFactory extends DatabaseFixtureFactory<
    Prisma.ApiVersion,
    Prisma.ApiVersionCreateInput,
    Prisma.ApiVersionInclude,
    Prisma.ApiVersionUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('ApiVersion', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.apiVersion;
    }

    protected getMinimalData(overrides?: Partial<Prisma.ApiVersionCreateInput>): Prisma.ApiVersionCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            versionLabel: "1.0.0",
            versionIndex: 1,
            isLatest: true,
            isComplete: true,
            isPrivate: false,
            apiType: 'REST',
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.ApiVersionCreateInput>): Prisma.ApiVersionCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            versionLabel: "1.0.0",
            versionIndex: 1,
            isLatest: true,
            isComplete: true,
            isPrivate: false,
            apiType: 'REST',
            config: apiConfigFixtures.complete,
            schema: {
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
                    },
                    {
                        id: generatePK(),
                        language: "es",
                        name: "Versión Completa de API",
                        description: "Una versión de API integral con todas las funcionalidades",
                        summary: "Documentación completa de API",
                        details: "Especificaciones detalladas de API y ejemplos de uso",
                    },
                ],
            },
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.ApiVersionInclude {
        return {
            translations: true,
            api: {
                select: {
                    id: true,
                    publicId: true,
                    handle: true,
                    isPrivate: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                        },
                    },
                },
            },
            parent: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                },
            },
            children: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                    versionIndex: true,
                },
                orderBy: {
                    versionIndex: 'asc',
                },
            },
            endpoints: {
                select: {
                    id: true,
                    path: true,
                    method: true,
                    description: true,
                    parameters: true,
                    responses: true,
                },
                orderBy: {
                    path: 'asc',
                },
            },
            resourceLists: {
                select: {
                    id: true,
                    publicId: true,
                    isPrivate: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                            description: true,
                        },
                    },
                },
            },
            _count: {
                select: {
                    children: true,
                    endpoints: true,
                    resourceLists: true,
                    bookmarks: true,
                    views: true,
                    votes: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.ApiVersionCreateInput,
        config: ApiVersionRelationConfig,
        tx: any
    ): Promise<Prisma.ApiVersionCreateInput> {
        let data = { ...baseData };

        // Handle API connection
        if (config.api) {
            data.api = {
                connect: { id: config.api.apiId },
            };
        }

        // Handle parent version
        if (config.parent) {
            data.parent = {
                connect: { id: config.parent.parentId },
            };
        }

        // Handle endpoints
        if (config.endpoints && Array.isArray(config.endpoints)) {
            data.endpoints = {
                create: config.endpoints.map(endpoint => ({
                    id: generatePK(),
                    path: endpoint.path,
                    method: endpoint.method,
                    description: endpoint.description ?? `${endpoint.method} ${endpoint.path}`,
                    parameters: endpoint.parameters ?? [],
                    responses: endpoint.responses ?? [
                        {
                            status: 200,
                            description: "Success",
                        },
                    ],
                })),
            };
        }

        // Handle resource lists
        if (config.resourceLists && Array.isArray(config.resourceLists)) {
            data.resourceLists = {
                create: config.resourceLists.map(resourceList => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    isPrivate: resourceList.isPrivate ?? false,
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: resourceList.name,
                            description: resourceList.description ?? resourceList.name,
                        }],
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
     * Create a REST API version
     */
    async createRestApiVersion(): Promise<Prisma.ApiVersion> {
        return this.createWithRelations({
            overrides: {
                apiType: 'REST',
                versionLabel: "1.0.0",
                isComplete: true,
                schema: {
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
            endpoints: [
                {
                    path: "/users",
                    method: 'GET',
                    description: "Retrieve a list of users",
                    parameters: [
                        { name: "limit", type: "integer", in: "query" },
                        { name: "offset", type: "integer", in: "query" },
                    ],
                    responses: [
                        { status: 200, description: "List of users" },
                        { status: 400, description: "Bad request" },
                    ],
                },
                {
                    path: "/users",
                    method: 'POST',
                    description: "Create a new user",
                    parameters: [
                        { name: "user", type: "object", in: "body" },
                    ],
                    responses: [
                        { status: 201, description: "User created" },
                        { status: 400, description: "Invalid input" },
                    ],
                },
                {
                    path: "/users/{id}",
                    method: 'GET',
                    description: "Get a specific user",
                    parameters: [
                        { name: "id", type: "integer", in: "path", required: true },
                    ],
                    responses: [
                        { status: 200, description: "User details" },
                        { status: 404, description: "User not found" },
                    ],
                },
            ],
            resourceLists: [
                {
                    name: "API Documentation",
                    description: "Complete API documentation and examples",
                    isPrivate: false,
                },
                {
                    name: "Code Examples",
                    description: "Sample code in various programming languages",
                    isPrivate: false,
                },
            ],
            translations: [
                {
                    language: "en",
                    name: "REST API v1.0.0",
                    description: "RESTful API with standard HTTP methods",
                    summary: "Complete REST API for user management",
                    details: "Full CRUD operations with proper HTTP status codes",
                },
            ],
        });
    }

    /**
     * Create a GraphQL API version
     */
    async createGraphQLApiVersion(): Promise<Prisma.ApiVersion> {
        return this.createWithRelations({
            overrides: {
                apiType: 'GraphQL',
                versionLabel: "1.0.0",
                isComplete: true,
                schema: {
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
            endpoints: [
                {
                    path: "/graphql",
                    method: 'POST',
                    description: "GraphQL endpoint for queries and mutations",
                    parameters: [
                        { name: "query", type: "string", in: "body", required: true },
                        { name: "variables", type: "object", in: "body" },
                    ],
                    responses: [
                        { status: 200, description: "GraphQL response" },
                        { status: 400, description: "GraphQL error" },
                    ],
                },
            ],
            resourceLists: [
                {
                    name: "GraphQL Schema",
                    description: "Complete GraphQL schema definition",
                    isPrivate: false,
                },
                {
                    name: "Query Examples",
                    description: "Example queries and mutations",
                    isPrivate: false,
                },
            ],
            translations: [
                {
                    language: "en",
                    name: "GraphQL API v1.0.0",
                    description: "GraphQL API with flexible queries",
                    summary: "Modern GraphQL API with type safety",
                    details: "Strongly typed GraphQL schema with introspection",
                },
            ],
        });
    }

    /**
     * Create a draft API version
     */
    async createDraftApiVersion(): Promise<Prisma.ApiVersion> {
        return this.createWithRelations({
            overrides: {
                versionLabel: "0.1.0-draft",
                isComplete: false,
                isPrivate: true,
                isLatest: false,
                apiType: 'REST',
            },
            translations: [
                {
                    language: "en",
                    name: "Draft API Version",
                    description: "Work in progress API version",
                    summary: "This is a draft version under development",
                },
            ],
        });
    }

    /**
     * Create a deprecated API version
     */
    async createDeprecatedApiVersion(): Promise<Prisma.ApiVersion> {
        return this.createWithRelations({
            overrides: {
                versionLabel: "0.9.0",
                isComplete: true,
                isPrivate: false,
                isLatest: false,
                versionIndex: 1,
                apiType: 'REST',
                deprecationNotice: "This version is deprecated. Please use v1.0.0 or later.",
            },
            translations: [
                {
                    language: "en",
                    name: "Deprecated API Version",
                    description: "This version is deprecated, use a newer version",
                    summary: "Please upgrade to the latest version",
                },
            ],
        });
    }

    /**
     * Create a child version (fork/branch)
     */
    async createChildApiVersion(parentId: string): Promise<Prisma.ApiVersion> {
        return this.createWithRelations({
            parent: { parentId },
            overrides: {
                versionLabel: "1.1.0-fork",
                isComplete: false,
                isPrivate: true,
                isLatest: false,
                versionIndex: 1,
                apiType: 'REST',
            },
            translations: [
                {
                    language: "en",
                    name: "Forked API Version",
                    description: "A fork of the parent API version",
                    summary: "Based on parent version with modifications",
                },
            ],
        });
    }

    protected async checkModelConstraints(record: Prisma.ApiVersion): Promise<string[]> {
        const violations: string[] = [];
        
        // Check that apiId is valid
        if (record.apiId) {
            const api = await this.prisma.api.findUnique({
                where: { id: record.apiId },
            });
            if (!api) {
                violations.push('ApiVersion must belong to a valid API');
            }
        }

        // Check version label format
        if (record.versionLabel && !/^\d+\.\d+\.\d+(-[\w.-]+)?$/.test(record.versionLabel)) {
            violations.push('Version label must follow semantic versioning format');
        }

        // Check that only one version per API is marked as latest
        if (record.isLatest && record.apiId) {
            const otherLatestVersions = await this.prisma.apiVersion.count({
                where: {
                    apiId: record.apiId,
                    isLatest: true,
                    id: { not: record.id },
                },
            });
            
            if (otherLatestVersions > 0) {
                violations.push('Only one version per API can be marked as latest');
            }
        }

        // Check that completed versions are not private if API is public
        if (record.isComplete && !record.isPrivate && record.apiId) {
            const api = await this.prisma.api.findUnique({
                where: { id: record.apiId },
                select: { isPrivate: true },
            });
            
            if (api && !api.isPrivate && record.isPrivate) {
                violations.push('Complete versions of public APIs should not be private');
            }
        }

        // Check API type is valid
        if (record.apiType && !['REST', 'GraphQL', 'WebSocket', 'gRPC'].includes(record.apiType)) {
            violations.push('API type must be one of: REST, GraphQL, WebSocket, gRPC');
        }

        // Check schema validity based on API type
        if (record.schema && record.apiType === 'GraphQL') {
            // Basic GraphQL schema validation
            if (typeof record.schema === 'object' && record.schema.schema) {
                const schema = record.schema.schema as string;
                if (!schema.includes('type Query') && !schema.includes('type Mutation')) {
                    violations.push('GraphQL schema must contain at least Query or Mutation types');
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
                apiType: 'REST',
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
                apiType: 'REST',
            },
            invalidApiType: {
                id: generatePK(),
                publicId: generatePublicId(),
                versionLabel: "1.0.0",
                versionIndex: 1,
                isLatest: true,
                isComplete: true,
                isPrivate: false,
                apiType: 'INVALID_TYPE', // Invalid API type
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.ApiVersionCreateInput> {
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
                apiType: 'WebSocket',
                schema: {
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
                apiType: 'gRPC',
                schema: {
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
                    }],
                },
            },
            complexSchema: {
                ...this.getMinimalData(),
                apiType: 'REST',
                schema: {
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
                        ])
                    ),
                },
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            endpoints: true,
            resourceLists: {
                include: {
                    translations: true,
                    resources: true,
                },
            },
            children: true,
            bookmarks: true,
            views: true,
            votes: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.ApiVersion,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // Delete in order of dependencies
        
        // Delete child versions first
        if (record.children?.length) {
            await tx.apiVersion.deleteMany({
                where: { parentId: record.id },
            });
        }

        // Delete endpoints
        if (record.endpoints?.length) {
            await tx.apiEndpoint.deleteMany({
                where: { apiVersionId: record.id },
            });
        }

        // Delete resource lists (cascade will handle their resources)
        if (record.resourceLists?.length) {
            await tx.resourceList.deleteMany({
                where: { apiVersionId: record.id },
            });
        }

        // Delete bookmarks
        if (record.bookmarks?.length) {
            await tx.bookmark.deleteMany({
                where: { apiVersionId: record.id },
            });
        }

        // Delete views
        if (record.views?.length) {
            await tx.view.deleteMany({
                where: { apiVersionId: record.id },
            });
        }

        // Delete votes/reactions
        if (record.votes?.length) {
            await tx.reaction.deleteMany({
                where: { apiVersionId: record.id },
            });
        }

        // Delete translations
        if (record.translations?.length) {
            await tx.apiVersionTranslation.deleteMany({
                where: { apiVersionId: record.id },
            });
        }
    }

    /**
     * Create version sequence for an API
     */
    async createVersionSequence(
        apiId: string,
        versions: Array<{ label: string; isLatest?: boolean; isComplete?: boolean; apiType?: string }>
    ): Promise<Prisma.ApiVersion[]> {
        const createdVersions: Prisma.ApiVersion[] = [];
        
        for (let i = 0; i < versions.length; i++) {
            const version = versions[i];
            const apiVersion = await this.createWithRelations({
                api: { apiId },
                overrides: {
                    versionLabel: version.label,
                    versionIndex: i + 1,
                    isLatest: version.isLatest ?? (i === versions.length - 1),
                    isComplete: version.isComplete ?? true,
                    apiType: version.apiType ?? 'REST',
                },
                translations: [
                    {
                        language: "en",
                        name: `API Version ${version.label}`,
                        description: `Version ${version.label} of the API`,
                        summary: `API v${version.label}`,
                    },
                ],
            });
            createdVersions.push(apiVersion);
        }
        
        return createdVersions;
    }
}

// Export factory creator function
export const createApiVersionDbFactory = (prisma: PrismaClient) => new ApiVersionDbFactory(prisma);
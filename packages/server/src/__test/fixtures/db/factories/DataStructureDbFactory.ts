import { generatePK, generatePublicId, nanoid } from "../idHelpers.js";
import { type resource, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

// DataStructure is implemented using the resource table with resourceType = "Standard"
// and resourceSubType = "StandardDataStructure" in resource_version

interface DataStructureRelationConfig extends RelationConfig {
    owner?: { userId?: string; teamId?: string };
    fields?: Array<{
        name: string;
        type: string;
        required?: boolean;
        description?: string;
        defaultValue?: any;
        validation?: any;
    }>;
    translations?: Array<{ language: string; name: string; description?: string; usage?: string }>;
}

/**
 * Database fixture factory for DataStructure model
 * Handles data structure definitions with fields, validation, and API relationships
 */
export class DataStructureDbFactory extends DatabaseFixtureFactory<
    resource,
    any,
    any,
    any
> {
    constructor(prisma: PrismaClient) {
        super("resource", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.resource;
    }

    protected getMinimalData(overrides?: Partial<any>): any {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            resourceType: "Standard",
            isPrivate: false,
            versions: {
                create: [{
                    id: generatePK(),
                    publicId: generatePublicId(),
                    versionIndex: 0,
                    isLatest: true,
                    resourceSubType: "StandardDataStructure",
                    config: JSON.stringify({
                        __version: "1.0",
                        resources: [],
                        schema: {
                            type: "object",
                            properties: {
                                id: {
                                    type: "string",
                                    description: "Unique identifier",
                                },
                            },
                            required: ["id"],
                        },
                    }),
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: `DataStructure_${nanoid(8)}`,
                            description: "Test data structure",
                        }],
                    },
                }],
            },
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<any>): any {
        const uniqueName = `complete_data_structure_${nanoid(8)}`;
        
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            resourceType: "Standard",
            isPrivate: false,
            versions: {
                create: [{
                    id: generatePK(),
                    publicId: generatePublicId(),
                    versionIndex: 0,
                    isLatest: true,
                    resourceSubType: "StandardDataStructure",
                    config: JSON.stringify({
                        __version: "1.0",
                        resources: [],
                        schema: {
                            type: "object",
                            title: "Complete Data Structure",
                            description: "A comprehensive data structure with all field types",
                            properties: {
                                id: {
                                    type: "string",
                                    description: "Unique identifier",
                                    pattern: "^[a-zA-Z0-9_-]+$",
                                },
                                name: {
                                    type: "string",
                                    description: "Name of the entity",
                                    minLength: 1,
                                    maxLength: 100,
                                },
                                email: {
                                    type: "string",
                                    format: "email",
                                    description: "Email address",
                                },
                                age: {
                                    type: "integer",
                                    minimum: 0,
                                    maximum: 150,
                                    description: "Age in years",
                                },
                                active: {
                                    type: "boolean",
                                    description: "Whether the entity is active",
                                    default: true,
                                },
                                tags: {
                                    type: "array",
                                    items: {
                                        type: "string",
                                    },
                                    description: "List of tags",
                                },
                                metadata: {
                                    type: "object",
                                    description: "Additional metadata",
                                    additionalProperties: true,
                                },
                                createdAt: {
                                    type: "string",
                                    format: "date-time",
                                    description: "Creation timestamp",
                                },
                            },
                            required: ["id", "name", "email"],
                            additionalProperties: false,
                        },
                        validation: {
                            strict: true,
                            validateRequired: true,
                            validateTypes: true,
                            validateFormats: true,
                        },
                        examples: [
                            {
                                id: "user_123",
                                name: "John Doe",
                                email: "john@example.com",
                                age: 30,
                                active: true,
                                tags: ["developer", "admin"],
                                metadata: {
                                    department: "Engineering",
                                    role: "Senior Developer",
                                },
                                createdAt: "2024-01-01T00:00:00Z",
                            },
                            {
                                id: "user_456",
                                name: "Jane Smith",
                                email: "jane@example.com",
                                age: 25,
                                active: false,
                                tags: ["designer"],
                                createdAt: "2024-01-02T00:00:00Z",
                            },
                        ],
                    }),
                    translations: {
                        create: [
                            {
                                id: generatePK(),
                                language: "en",
                                name: "Complete Data Structure",
                                description: "A comprehensive data structure with validation",
                                details: "Use this structure for complex data validation",
                            },
                            {
                                id: generatePK(),
                                language: "es",
                                name: "Estructura de Datos Completa",
                                description: "Una estructura de datos integral con validación",
                                details: "Usa esta estructura para validación de datos compleja",
                            },
                        ],
                    },
                }],
            },
            ...overrides,
        };
    }

    protected getDefaultInclude(): any {
        return {
            versions: {
                include: {
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
                    handle: true,
                },
            },
            bookmarkedBy: true,
            viewedBy: true,
            reactions: true,
        };
    }

    protected async applyRelationships(
        baseData: any,
        config: DataStructureRelationConfig,
        tx: any,
    ): Promise<any> {
        const data = { ...baseData };

        // Handle owner (user or team)
        if (config.owner) {
            if (config.owner.userId) {
                data.ownedByUser = {
                    connect: { id: BigInt(config.owner.userId) },
                };
            } else if (config.owner.teamId) {
                data.ownedByTeam = {
                    connect: { id: BigInt(config.owner.teamId) },
                };
            }
        }

        // API relationships are no longer supported in the resource model

        // Handle fields (add to schema in config)
        if (config.fields && Array.isArray(config.fields) && data.versions?.create) {
            const version = Array.isArray(data.versions.create) ? data.versions.create[0] : data.versions.create;
            const configObj = version.config ? JSON.parse(version.config) : { __version: "1.0", resources: [], schema: { type: "object", properties: {} } };
            const schema = configObj.schema || { type: "object", properties: {} };
            
            config.fields.forEach(field => {
                schema.properties[field.name] = {
                    type: field.type,
                    description: field.description || field.name,
                    ...(field.defaultValue !== undefined && { default: field.defaultValue }),
                    ...(field.validation && field.validation),
                };
                
                if (field.required) {
                    schema.required = schema.required || [];
                    if (!schema.required.includes(field.name)) {
                        schema.required.push(field.name);
                    }
                }
            });
            
            configObj.schema = schema;
            version.config = JSON.stringify(configObj) as any;
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations) && data.versions?.create) {
            const version = Array.isArray(data.versions.create) ? data.versions.create[0] : data.versions.create;
            version.translations = {
                create: config.translations.map(trans => ({
                    id: generatePK(),
                    ...trans,
                })),
            };
        }

        return data;
    }

    /**
     * Create a simple user data structure
     */
    async createUserDataStructure(): Promise<resource> {
        return this.createWithRelations({
            overrides: {
                versions: {
                    create: [{
                        id: generatePK(),
                        publicId: generatePublicId(),
                        versionIndex: 0,
                        isLatest: true,
                        resourceSubType: "StandardDataStructure",
                        config: JSON.stringify({
                            __version: "1.0",
                            resources: [],
                            schema: {
                                type: "object",
                                title: "User",
                                description: "User data structure",
                                properties: {
                                    id: {
                                        type: "string",
                                        description: "User ID",
                                    },
                                    username: {
                                        type: "string",
                                        description: "Username",
                                        minLength: 3,
                                        maxLength: 50,
                                    },
                                    email: {
                                        type: "string",
                                        format: "email",
                                        description: "Email address",
                                    },
                                    firstName: {
                                        type: "string",
                                        description: "First name",
                                    },
                                    lastName: {
                                        type: "string",
                                        description: "Last name",
                                    },
                                },
                                required: ["id", "username", "email"],
                            },
                        }),
                    }],
                },
            },
            translations: [
                {
                    language: "en",
                    name: "User Data Structure",
                    description: "Standard user entity structure",
                    details: "Use for user registration and profile data",
                },
            ],
        });
    }

    /**
     * Create a product data structure
     */
    async createProductDataStructure(): Promise<resource> {
        return this.createWithRelations({
            overrides: {
                versions: {
                    create: [{
                        id: generatePK(),
                        publicId: generatePublicId(),
                        versionIndex: 0,
                        isLatest: true,
                        resourceSubType: "StandardDataStructure",
                        config: JSON.stringify({
                            __version: "1.0",
                            resources: [],
                            schema: {
                                type: "object",
                                title: "Product",
                                description: "Product data structure",
                                properties: {
                                    id: {
                                        type: "string",
                                        description: "Product ID",
                                    },
                                    name: {
                                        type: "string",
                                        description: "Product name",
                                        minLength: 1,
                                        maxLength: 200,
                                    },
                                    description: {
                                        type: "string",
                                        description: "Product description",
                                    },
                                    price: {
                                        type: "number",
                                        minimum: 0,
                                        description: "Product price",
                                    },
                                    currency: {
                                        type: "string",
                                        enum: ["USD", "EUR", "GBP", "JPY"],
                                        description: "Price currency",
                                    },
                                    category: {
                                        type: "string",
                                        description: "Product category",
                                    },
                                    inStock: {
                                        type: "boolean",
                                        description: "Whether product is in stock",
                                        default: true,
                                    },
                                    tags: {
                                        type: "array",
                                        items: {
                                            type: "string",
                                        },
                                        description: "Product tags",
                                    },
                                },
                                required: ["id", "name", "price", "currency"],
                            },
                        }),
                    }],
                },
            },
            translations: [
                {
                    language: "en",
                    name: "Product Data Structure",
                    description: "E-commerce product entity structure",
                    details: "Use for product catalog and inventory management",
                },
            ],
        });
    }

    /**
     * Create a private data structure
     */
    async createPrivateDataStructure(): Promise<resource> {
        return this.createWithRelations({
            overrides: {
                isPrivate: true,
                versions: {
                    create: [{
                        id: generatePK(),
                        publicId: generatePublicId(),
                        versionIndex: 0,
                        isLatest: true,
                        resourceSubType: "StandardDataStructure",
                        config: JSON.stringify({
                            __version: "1.0",
                            resources: [],
                            schema: {
                                type: "object",
                                title: "Private Data",
                                description: "Internal data structure",
                                properties: {
                                    id: {
                                        type: "string",
                                        description: "Internal ID",
                                    },
                                    secretKey: {
                                        type: "string",
                                        description: "Secret key",
                                        pattern: "^[A-Za-z0-9+/]+=*$",
                                    },
                                },
                                required: ["id", "secretKey"],
                            },
                        }),
                    }],
                },
            },
            translations: [
                {
                    language: "en",
                    name: "Private Data Structure",
                    description: "Internal data structure for private use",
                    details: "For internal systems only",
                },
            ],
        });
    }

    /**
     * Create an API-linked data structure
     */
    async createApiLinkedDataStructure(apiId: string, apiVersionId?: string): Promise<resource> {
        return this.createWithRelations({
            overrides: {
                versions: {
                    create: [{
                        id: generatePK(),
                        publicId: generatePublicId(),
                        versionIndex: 0,
                        isLatest: true,
                        resourceSubType: "StandardDataStructure",
                        config: JSON.stringify({
                            __version: "1.0",
                            resources: [],
                            schema: {
                                type: "object",
                                title: "API Response",
                                description: "API response data structure",
                                properties: {
                                    id: {
                                        type: "string",
                                        description: "Resource ID",
                                    },
                                    type: {
                                        type: "string",
                                        description: "Resource type",
                                    },
                                    attributes: {
                                        type: "object",
                                        description: "Resource attributes",
                                        additionalProperties: true,
                                    },
                                    relationships: {
                                        type: "object",
                                        description: "Related resources",
                                        additionalProperties: true,
                                    },
                                },
                                required: ["id", "type"],
                            },
                        }),
                    }],
                },
            },
            translations: [
                {
                    language: "en",
                    name: "API Data Structure",
                    description: "Data structure for API response format",
                    details: "Use for API request/response validation",
                },
            ],
        });
    }

    /**
     * Create a nested data structure
     */
    async createNestedDataStructure(): Promise<resource> {
        return this.createWithRelations({
            overrides: {
                versions: {
                    create: [{
                        id: generatePK(),
                        publicId: generatePublicId(),
                        versionIndex: 0,
                        isLatest: true,
                        resourceSubType: "StandardDataStructure",
                        config: JSON.stringify({
                            __version: "1.0",
                            resources: [],
                            schema: {
                                type: "object",
                                title: "Order",
                                description: "Complex nested order structure",
                                properties: {
                                    id: {
                                        type: "string",
                                        description: "Order ID",
                                    },
                                    customer: {
                                        type: "object",
                                        description: "Customer information",
                                        properties: {
                                            id: { type: "string" },
                                            name: { type: "string" },
                                            email: { type: "string", format: "email" },
                                            address: {
                                                type: "object",
                                                properties: {
                                                    street: { type: "string" },
                                                    city: { type: "string" },
                                                    postalCode: { type: "string" },
                                                    country: { type: "string" },
                                                },
                                                required: ["street", "city", "country"],
                                            },
                                        },
                                        required: ["id", "name", "email"],
                                    },
                                    items: {
                                        type: "array",
                                        description: "Order items",
                                        items: {
                                            type: "object",
                                            properties: {
                                                productId: { type: "string" },
                                                quantity: { type: "integer", minimum: 1 },
                                                price: { type: "number", minimum: 0 },
                                            },
                                            required: ["productId", "quantity", "price"],
                                        },
                                    },
                                    total: {
                                        type: "number",
                                        minimum: 0,
                                        description: "Order total",
                                    },
                                    status: {
                                        type: "string",
                                        enum: ["pending", "processing", "shipped", "delivered", "cancelled"],
                                        description: "Order status",
                                    },
                                },
                                required: ["id", "customer", "items", "total", "status"],
                            },
                            validation: {
                                strict: true,
                                validateRequired: true,
                                validateTypes: true,
                                customValidators: [
                                    {
                                        field: "total",
                                        rule: "must equal sum of item prices",
                                    },
                                ],
                            },
                        }),
                    }],
                },
            },
            translations: [
                {
                    language: "en",
                    name: "Nested Order Structure",
                    description: "Complex order data structure with nested objects",
                    details: "Use for e-commerce order processing",
                },
            ],
        });
    }

    protected async checkModelConstraints(record: resource): Promise<string[]> {
        const violations: string[] = [];
        
        // Check publicId uniqueness
        if (record.publicId) {
            const duplicate = await this.prisma.resource.findFirst({
                where: {
                    publicId: record.publicId,
                    id: { not: record.id },
                    resourceType: "Standard",
                },
            });
            
            if (duplicate) {
                violations.push("Resource publicId must be unique");
            }
        }

        // Check publicId format
        if (record.publicId && !/^[a-zA-Z0-9_-]+$/.test(record.publicId)) {
            violations.push("Resource publicId contains invalid characters");
        }

        // Check schema validity in config
        // Note: Schema is now stored in the config field of resourceVersion

        // Check resource type consistency
        if (record.resourceType !== "Standard") {
            violations.push("DataStructure resources must have resourceType 'Standard'");
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
                resourceType: 123, // Should be string
                isPrivate: "yes", // Should be boolean
            },
            duplicatePublicId: {
                id: generatePK(),
                publicId: "existing_structure", // Assumes this exists
                resourceType: "Standard",
                isPrivate: false,
            },
            invalidResourceType: {
                id: generatePK(),
                publicId: generatePublicId(),
                resourceType: "Invalid", // Should be "Standard"
                isPrivate: false,
            },
            invalidPublicIdFormat: {
                id: generatePK(),
                publicId: "invalid public id with spaces", // Invalid characters
                resourceType: "Standard",
                isPrivate: false,
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, any> {
        return {
            maxPublicIdLength: {
                ...this.getMinimalData(),
                publicId: "structure_" + "a".repeat(40), // Max length publicId
            },
            unicodeContent: {
                ...this.getMinimalData(),
                versions: {
                    create: [{
                        id: generatePK(),
                        publicId: generatePublicId(),
                        versionIndex: 0,
                        isLatest: true,
                        resourceSubType: "StandardDataStructure",
                        config: JSON.stringify({
                            __version: "1.0",
                            resources: [],
                        }),
                        translations: {
                            create: [{
                                id: generatePK(),
                                language: "zh",
                                name: "数据结构 🗃️",
                                description: "使用Unicode字符的数据结构",
                                details: "用于处理国际化数据",
                            }],
                        },
                    }],
                },
            },
            complexValidation: {
                ...this.getMinimalData(),
                versions: {
                    create: [{
                        id: generatePK(),
                        publicId: generatePublicId(),
                        versionIndex: 0,
                        isLatest: true,
                        resourceSubType: "StandardDataStructure",
                        config: JSON.stringify({
                            __version: "1.0",
                            resources: [],
                            schema: {
                                type: "object",
                                properties: {
                                    email: {
                                        type: "string",
                                        format: "email",
                                        pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
                                    },
                                    phone: {
                                        type: "string",
                                        pattern: "^\\+?[1-9]\\d{1,14}$",
                                    },
                                    password: {
                                        type: "string",
                                        minLength: 8,
                                        pattern: "^(?=.*[a-z])(?=.*[A-Z])(?=.*\\d)(?=.*[@$!%*?&])[A-Za-z\\d@$!%*?&]",
                                    },
                                },
                                required: ["email", "phone", "password"],
                            },
                            validation: {
                                strict: true,
                                validateRequired: true,
                                validateTypes: true,
                                validateFormats: true,
                                validatePatterns: true,
                            },
                        }),
                    }],
                },
            },
            recursiveSchema: {
                ...this.getMinimalData(),
                versions: {
                    create: [{
                        id: generatePK(),
                        publicId: generatePublicId(),
                        versionIndex: 0,
                        isLatest: true,
                        resourceSubType: "StandardDataStructure",
                        config: JSON.stringify({
                            __version: "1.0",
                            resources: [],
                            schema: {
                                type: "object",
                                properties: {
                                    id: { type: "string" },
                                    name: { type: "string" },
                                    children: {
                                        type: "array",
                                        items: {
                                            $ref: "#", // Self-reference
                                        },
                                    },
                                },
                                required: ["id", "name"],
                            },
                        }),
                    }],
                },
            },
            arrayOfComplexObjects: {
                ...this.getMinimalData(),
                versions: {
                    create: [{
                        id: generatePK(),
                        publicId: generatePublicId(),
                        versionIndex: 0,
                        isLatest: true,
                        resourceSubType: "StandardDataStructure",
                        config: JSON.stringify({
                            __version: "1.0",
                            resources: [],
                            schema: {
                                type: "array",
                                items: {
                                    type: "object",
                                    properties: {
                                        id: { type: "string" },
                                        data: {
                                            type: "object",
                                            additionalProperties: {
                                                oneOf: [
                                                    { type: "string" },
                                                    { type: "number" },
                                                    { type: "boolean" },
                                                    { type: "array" },
                                                    { type: "object" },
                                                ],
                                            },
                                        },
                                    },
                                    required: ["id"],
                                },
                            },
                        }),
                    }],
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
            bookmarks: true,
            views: true,
            votes: true,
        };
    }

    protected async deleteRelatedRecords(
        record: resource,
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // Delete in order of dependencies
        
        // Delete bookmarks
        if ((record as any).bookmarkedBy?.length) {
            await tx.bookmark.deleteMany({
                where: { resourceId: record.id },
            });
        }

        // Delete views
        if ((record as any).viewedBy?.length) {
            await tx.view.deleteMany({
                where: { resourceId: record.id },
            });
        }

        // Delete votes/reactions
        if ((record as any).reactions?.length) {
            await tx.reaction.deleteMany({
                where: { resourceId: record.id },
            });
        }

        // Delete resource versions and their translations
        if ((record as any).versions?.length) {
            for (const version of (record as any).versions) {
                await tx.resourceVersionTranslation.deleteMany({
                    where: { resourceVersionId: version.id },
                });
            }
            await tx.resourceVersion.deleteMany({
                where: { resourceId: record.id },
            });
        }
    }
}

// Export factory creator function
export const createDataStructureDbFactory = (prisma: PrismaClient) => new DataStructureDbFactory(prisma);

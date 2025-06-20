import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface DataStructureRelationConfig extends RelationConfig {
    owner?: { userId?: string; teamId?: string };
    api?: { apiId?: string; apiVersionId?: string };
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
    Prisma.DataStructure,
    Prisma.DataStructureCreateInput,
    Prisma.DataStructureInclude,
    Prisma.DataStructureUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('DataStructure', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.dataStructure;
    }

    protected getMinimalData(overrides?: Partial<Prisma.DataStructureCreateInput>): Prisma.DataStructureCreateInput {
        const uniqueName = `data_structure_${nanoid(8)}`;
        
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            name: uniqueName,
            isPrivate: false,
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
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.DataStructureCreateInput>): Prisma.DataStructureCreateInput {
        const uniqueName = `complete_data_structure_${nanoid(8)}`;
        
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            name: uniqueName,
            isPrivate: false,
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
            translations: {
                create: [
                    {
                        id: generatePK(),
                        language: "en",
                        name: "Complete Data Structure",
                        description: "A comprehensive data structure with validation",
                        usage: "Use this structure for complex data validation",
                    },
                    {
                        id: generatePK(),
                        language: "es",
                        name: "Estructura de Datos Completa",
                        description: "Una estructura de datos integral con validación",
                        usage: "Usa esta estructura para validación de datos compleja",
                    },
                ],
            },
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.DataStructureInclude {
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
            api: {
                select: {
                    id: true,
                    publicId: true,
                    handle: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                        },
                    },
                },
            },
            apiVersion: {
                select: {
                    id: true,
                    publicId: true,
                    versionLabel: true,
                    apiType: true,
                },
            },
            usedByEndpoints: {
                select: {
                    id: true,
                    path: true,
                    method: true,
                },
            },
            _count: {
                select: {
                    usedByEndpoints: true,
                    bookmarks: true,
                    views: true,
                    votes: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.DataStructureCreateInput,
        config: DataStructureRelationConfig,
        tx: any
    ): Promise<Prisma.DataStructureCreateInput> {
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

        // Handle API relationships
        if (config.api) {
            if (config.api.apiId) {
                data.api = {
                    connect: { id: config.api.apiId },
                };
            }
            if (config.api.apiVersionId) {
                data.apiVersion = {
                    connect: { id: config.api.apiVersionId },
                };
            }
        }

        // Handle fields (add to schema)
        if (config.fields && Array.isArray(config.fields)) {
            const schema = data.schema as any || { type: "object", properties: {} };
            
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
            
            data.schema = schema;
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
     * Create a simple user data structure
     */
    async createUserDataStructure(): Promise<Prisma.DataStructure> {
        return this.createWithRelations({
            overrides: {
                name: `user_structure_${nanoid(8)}`,
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
            },
            translations: [
                {
                    language: "en",
                    name: "User Data Structure",
                    description: "Standard user entity structure",
                    usage: "Use for user registration and profile data",
                },
            ],
        });
    }

    /**
     * Create a product data structure
     */
    async createProductDataStructure(): Promise<Prisma.DataStructure> {
        return this.createWithRelations({
            overrides: {
                name: `product_structure_${nanoid(8)}`,
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
            },
            translations: [
                {
                    language: "en",
                    name: "Product Data Structure",
                    description: "E-commerce product entity structure",
                    usage: "Use for product catalog and inventory management",
                },
            ],
        });
    }

    /**
     * Create a private data structure
     */
    async createPrivateDataStructure(): Promise<Prisma.DataStructure> {
        return this.createWithRelations({
            overrides: {
                isPrivate: true,
                name: `private_structure_${nanoid(8)}`,
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
            },
            translations: [
                {
                    language: "en",
                    name: "Private Data Structure",
                    description: "Internal data structure for private use",
                    usage: "For internal systems only",
                },
            ],
        });
    }

    /**
     * Create an API-linked data structure
     */
    async createApiLinkedDataStructure(apiId: string, apiVersionId?: string): Promise<Prisma.DataStructure> {
        return this.createWithRelations({
            api: { apiId, apiVersionId },
            overrides: {
                name: `api_structure_${nanoid(8)}`,
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
            },
            translations: [
                {
                    language: "en",
                    name: "API Data Structure",
                    description: "Data structure linked to API endpoint",
                    usage: "Use for API request/response validation",
                },
            ],
        });
    }

    /**
     * Create a nested data structure
     */
    async createNestedDataStructure(): Promise<Prisma.DataStructure> {
        return this.createWithRelations({
            overrides: {
                name: `nested_structure_${nanoid(8)}`,
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
            },
            translations: [
                {
                    language: "en",
                    name: "Nested Order Structure",
                    description: "Complex order data structure with nested objects",
                    usage: "Use for e-commerce order processing",
                },
            ],
        });
    }

    protected async checkModelConstraints(record: Prisma.DataStructure): Promise<string[]> {
        const violations: string[] = [];
        
        // Check name uniqueness within owner scope
        if (record.name) {
            const whereClause: any = {
                name: record.name,
                id: { not: record.id },
            };
            
            if (record.ownerId) {
                whereClause.ownerId = record.ownerId;
            } else if (record.teamId) {
                whereClause.teamId = record.teamId;
            }
            
            const duplicate = await this.prisma.dataStructure.findFirst({
                where: whereClause,
            });
            
            if (duplicate) {
                violations.push('Data structure name must be unique within owner scope');
            }
        }

        // Check name format
        if (record.name && !/^[a-zA-Z0-9_-]+$/.test(record.name)) {
            violations.push('Data structure name contains invalid characters');
        }

        // Check schema validity
        if (record.schema) {
            try {
                const schema = record.schema as any;
                if (typeof schema !== 'object') {
                    violations.push('Schema must be a valid JSON object');
                } else {
                    // Basic JSON Schema validation
                    if (!schema.type) {
                        violations.push('Schema must have a type field');
                    }
                    
                    if (schema.type === 'object' && !schema.properties) {
                        violations.push('Object schema must have properties');
                    }
                }
            } catch (error) {
                violations.push('Schema is not valid JSON');
            }
        }

        // Check API relationship consistency
        if (record.apiVersionId && !record.apiId) {
            violations.push('ApiVersionId requires apiId to be set');
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, publicId, name
                isPrivate: false,
                schema: { type: "object" },
            },
            invalidTypes: {
                id: "not-a-snowflake",
                publicId: 123, // Should be string
                name: true, // Should be string
                isPrivate: "yes", // Should be boolean
                schema: "invalid", // Should be object
            },
            duplicateName: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "existing_structure", // Assumes this exists
                isPrivate: false,
                schema: { type: "object" },
            },
            invalidSchema: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: `invalid_schema_${nanoid(8)}`,
                isPrivate: false,
                schema: { invalid: "schema" }, // Missing type
            },
            invalidNameFormat: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "invalid name with spaces", // Invalid characters
                isPrivate: false,
                schema: { type: "object" },
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.DataStructureCreateInput> {
        return {
            maxNameLength: {
                ...this.getMinimalData(),
                name: 'structure_' + 'a'.repeat(40), // Max length name
            },
            unicodeContent: {
                ...this.getMinimalData(),
                name: `unicode_structure_${nanoid(8)}`,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "zh",
                        name: "数据结构 🗃️",
                        description: "使用Unicode字符的数据结构",
                        usage: "用于处理国际化数据",
                    }],
                },
            },
            complexValidation: {
                ...this.getMinimalData(),
                name: `complex_validation_${nanoid(8)}`,
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
            },
            recursiveSchema: {
                ...this.getMinimalData(),
                name: `recursive_structure_${nanoid(8)}`,
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
            },
            arrayOfComplexObjects: {
                ...this.getMinimalData(),
                name: `array_structure_${nanoid(8)}`,
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
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            usedByEndpoints: true,
            bookmarks: true,
            views: true,
            votes: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.DataStructure,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // Delete in order of dependencies
        
        // Remove references from API endpoints
        if (record.usedByEndpoints?.length) {
            await tx.apiEndpoint.updateMany({
                where: { 
                    OR: [
                        { requestStructureId: record.id },
                        { responseStructureId: record.id },
                    ],
                },
                data: { 
                    requestStructureId: null,
                    responseStructureId: null,
                },
            });
        }

        // Delete bookmarks
        if (record.bookmarks?.length) {
            await tx.bookmark.deleteMany({
                where: { dataStructureId: record.id },
            });
        }

        // Delete views
        if (record.views?.length) {
            await tx.view.deleteMany({
                where: { dataStructureId: record.id },
            });
        }

        // Delete votes/reactions
        if (record.votes?.length) {
            await tx.reaction.deleteMany({
                where: { dataStructureId: record.id },
            });
        }

        // Delete translations
        if (record.translations?.length) {
            await tx.dataStructureTranslation.deleteMany({
                where: { dataStructureId: record.id },
            });
        }
    }
}

// Export factory creator function
export const createDataStructureDbFactory = (prisma: PrismaClient) => new DataStructureDbFactory(prisma);
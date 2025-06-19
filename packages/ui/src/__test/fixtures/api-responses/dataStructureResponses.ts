import { type Resource, type ResourceVersion, type Label, type Tag, ResourceSubType } from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse } from "./userResponses.js";
import { minimalTeamResponse } from "./teamResponses.js";

/**
 * API response fixtures for data structures (Resources with StandardDataStructure subtype)
 * These represent what components receive from API calls
 */

/**
 * Mock tag data for data structures
 */
const schemaTag: Tag = {
    __typename: "Tag",
    id: "tag_schema_123456789",
    tag: "schema",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 42,
    translations: [
        {
            __typename: "TagTranslation",
            id: "tagtrans_schema_123",
            language: "en",
            description: "Data schemas and structure definitions",
        },
    ],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

const jsonSchemaTag: Tag = {
    __typename: "Tag",
    id: "tag_jsonschema_123456789",
    tag: "json-schema",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 28,
    translations: [
        {
            __typename: "TagTranslation",
            id: "tagtrans_jsonschema_123",
            language: "en",
            description: "JSON Schema specifications and validations",
        },
    ],
    you: {
        __typename: "TagYou",
        isBookmarked: true,
    },
};

const databaseTag: Tag = {
    __typename: "Tag",
    id: "tag_database_123456789",
    tag: "database",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 156,
    translations: [
        {
            __typename: "TagTranslation",
            id: "tagtrans_database_123",
            language: "en",
            description: "Database schemas and table structures",
        },
    ],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

const apiTag: Tag = {
    __typename: "Tag",
    id: "tag_api_123456789",
    tag: "api",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 89,
    translations: [],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

/**
 * Mock label data for data structures
 */
const stableLabel: Label = {
    __typename: "Label",
    id: "label_stable_123456789",
    label: "Stable",
    color: "#4caf50",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    you: {
        __typename: "LabelYou",
        canDelete: true,
        canUpdate: true,
    },
};

const draftLabel: Label = {
    __typename: "Label",
    id: "label_draft_123456789",
    label: "Draft",
    color: "#ff9800",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    you: {
        __typename: "LabelYou",
        canDelete: false,
        canUpdate: false,
    },
};

const deprecatedLabel: Label = {
    __typename: "Label",
    id: "label_deprecated_123456",
    label: "Deprecated",
    color: "#f44336",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    you: {
        __typename: "LabelYou",
        canDelete: false,
        canUpdate: false,
    },
};

/**
 * Minimal data structure version
 */
const minimalDataStructureVersion: ResourceVersion = {
    __typename: "ResourceVersion",
    id: "resver_123456789012345",
    versionLabel: "1.0.0",
    versionNotes: null,
    isLatest: true,
    isPrivate: false,
    isComplete: true,
    isAutomatable: false,
    resourceSubType: ResourceSubType.StandardDataStructure,
    codeLanguage: null,
    complexity: 2,
    simplicity: 8,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: "2024-01-01T00:00:00Z",
    comments: [],
    commentsCount: 0,
    forks: [],
    forksCount: 0,
    pulls: [],
    pullsCount: 0,
    issues: [],
    issuesCount: 0,
    labels: [],
    labelsCount: 0,
    reportsCount: 0,
    score: 0,
    bookmarks: 0,
    views: 0,
    transfersCount: 0,
    directoryListings: [],
    directoryListingsCount: 0,
    isInternal: false,
    root: null,
    timesCompleted: 0,
    timesStarted: 0,
    translations: [
        {
            __typename: "ResourceVersionTranslation",
            id: "resvtrans_123456789",
            language: "en",
            name: "User Profile Schema",
            description: "Basic user profile data structure",
            instructions: "Defines the standard user profile fields",
            details: null,
        },
    ],
    resourceList: null,
    you: {
        __typename: "ResourceVersionYou",
        canBookmark: true,
        canComment: true,
        canCopy: true,
        canDelete: false,
        canReact: true,
        canRead: true,
        canReport: true,
        canRun: false,
        canUpdate: false,
    },
};

/**
 * Complete data structure version with all features
 */
const completeDataStructureVersion: ResourceVersion = {
    __typename: "ResourceVersion",
    id: "resver_987654321098765",
    versionLabel: "2.3.0",
    versionNotes: "Added support for nested objects and improved validation rules",
    isLatest: true,
    isPrivate: false,
    isComplete: true,
    isAutomatable: true,
    resourceSubType: ResourceSubType.StandardDataStructure,
    codeLanguage: null,
    complexity: 7,
    simplicity: 5,
    createdAt: "2023-06-01T00:00:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    completedAt: "2024-01-10T00:00:00Z",
    comments: [],
    commentsCount: 8,
    forks: [],
    forksCount: 2,
    pulls: [],
    pullsCount: 0,
    issues: [],
    issuesCount: 0,
    labels: [stableLabel],
    labelsCount: 1,
    reportsCount: 0,
    score: 234,
    bookmarks: 67,
    views: 1543,
    transfersCount: 0,
    directoryListings: [],
    directoryListingsCount: 1,
    isInternal: false,
    root: null,
    timesCompleted: 0,
    timesStarted: 0,
    config: {
        schemaType: "json-schema",
        version: "draft-07",
        strict: true,
        allowAdditionalProperties: false,
        validateRequired: true,
        errorFormat: "detailed",
    },
    translations: [
        {
            __typename: "ResourceVersionTranslation",
            id: "resvtrans_987654321",
            language: "en",
            name: "E-commerce Product Schema",
            description: "Comprehensive product data structure for e-commerce applications with inventory, pricing, and metadata",
            instructions: "## Schema Usage\n\n1. **Product Information**: Core product details including name, description, SKU\n2. **Pricing**: Support for multiple currencies, discounts, and tax calculations\n3. **Inventory**: Stock levels, availability, and warehouse information\n4. **Categories**: Hierarchical category structure with tags\n5. **Metadata**: SEO fields, timestamps, and custom attributes\n\n## Validation Rules\n- All required fields must be present\n- Price must be positive number\n- SKU must be unique\n- Category hierarchy must be valid\n\n## Example Usage\n```json\n{\n  \"id\": \"prod_123\",\n  \"name\": \"Wireless Headphones\",\n  \"sku\": \"WH-001\",\n  \"price\": 99.99,\n  \"currency\": \"USD\"\n}\n```",
            details: "This schema supports complex product data including variants, bundles, and dynamic pricing. It includes built-in validation for common e-commerce scenarios and extensibility for custom fields.",
        },
        {
            __typename: "ResourceVersionTranslation",
            id: "resvtrans_876543210",
            language: "es",
            name: "Esquema de Producto E-commerce",
            description: "Estructura de datos completa para productos de aplicaciones e-commerce con inventario, precios y metadatos",
            instructions: "## Uso del Esquema\n\n1. **Información del Producto**: Detalles principales incluyendo nombre, descripción, SKU",
            details: null,
        },
    ],
    resourceList: {
        __typename: "ResourceList",
        id: "reslist_schema_123",
        usedFor: "Display",
        resources: [],
    },
    you: {
        __typename: "ResourceVersionYou",
        canBookmark: true,
        canComment: true,
        canCopy: true,
        canDelete: true,
        canReact: true,
        canRead: true,
        canReport: false,
        canRun: false,
        canUpdate: true,
    },
};

/**
 * Minimal data structure API response
 */
export const minimalDataStructureResponse: Resource = {
    __typename: "Resource",
    id: "resource_123456789012345",
    isDeleted: false,
    isInternal: false,
    isPrivate: false,
    resourceType: "Standard",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: "2024-01-01T00:00:00Z",
    score: 0,
    bookmarks: 0,
    views: 0,
    bookmarkedBy: [],
    owner: {
        __typename: "User",
        ...minimalUserResponse,
    },
    hasCompleteVersion: true,
    labels: [],
    tags: [],
    permissions: JSON.stringify(["Read"]),
    versions: [minimalDataStructureVersion],
    versionsCount: 1,
    forksCount: 0,
    issues: [],
    issuesCount: 0,
    pullsCount: 0,
    reportsCount: 0,
    transfersCount: 0,
    you: {
        __typename: "ResourceYou",
        canBookmark: true,
        canComment: true,
        canDelete: false,
        canReact: true,
        canRead: true,
        canTransfer: false,
        canUpdate: false,
        isBookmarked: false,
        isViewed: false,
        reaction: null,
    },
};

/**
 * Complete data structure API response with all fields
 */
export const completeDataStructureResponse: Resource = {
    __typename: "Resource",
    id: "resource_987654321098765",
    isDeleted: false,
    isInternal: false,
    isPrivate: false,
    resourceType: "Standard",
    createdAt: "2023-06-01T00:00:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    completedAt: "2024-01-10T00:00:00Z",
    score: 234,
    bookmarks: 67,
    views: 1543,
    bookmarkedBy: [],
    owner: {
        __typename: "Team",
        ...minimalTeamResponse,
    },
    hasCompleteVersion: true,
    labels: [stableLabel],
    tags: [schemaTag, jsonSchemaTag, apiTag],
    permissions: JSON.stringify(["Read", "Copy", "Fork"]),
    versions: [completeDataStructureVersion],
    versionsCount: 5,
    forksCount: 2,
    issues: [],
    issuesCount: 0,
    pullsCount: 0,
    reportsCount: 0,
    transfersCount: 0,
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
        reaction: "like",
    },
};

/**
 * Private data structure response
 */
export const privateDataStructureResponse: Resource = {
    __typename: "Resource",
    id: "resource_private_123456",
    isDeleted: false,
    isInternal: false,
    isPrivate: true,
    resourceType: "Standard",
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: null,
    score: 0,
    bookmarks: 0,
    views: 0,
    bookmarkedBy: [],
    owner: {
        __typename: "User",
        ...completeUserResponse,
    },
    hasCompleteVersion: false,
    labels: [draftLabel],
    tags: [],
    permissions: JSON.stringify([]),
    versions: [{
        ...minimalDataStructureVersion,
        id: "resver_private_123456",
        isPrivate: true,
        isComplete: false,
        translations: [
            {
                __typename: "ResourceVersionTranslation",
                id: "resvtrans_private_123",
                language: "en",
                name: "Internal API Schema",
                description: "Private data structure for internal API endpoints",
                instructions: "Internal use only - access restricted",
                details: null,
            },
        ],
    }],
    versionsCount: 1,
    forksCount: 0,
    issues: [],
    issuesCount: 0,
    pullsCount: 0,
    reportsCount: 0,
    transfersCount: 0,
    you: {
        __typename: "ResourceYou",
        canBookmark: false,
        canComment: false,
        canDelete: false,
        canReact: false,
        canRead: false,
        canTransfer: false,
        canUpdate: false,
        isBookmarked: false,
        isViewed: false,
        reaction: null,
    },
};

/**
 * Data structure variant states for testing
 */
export const dataStructureResponseVariants = {
    minimal: minimalDataStructureResponse,
    complete: completeDataStructureResponse,
    private: privateDataStructureResponse,
    databaseSchema: {
        ...completeDataStructureResponse,
        id: "resource_dbschema_123456789",
        versions: [{
            ...completeDataStructureVersion,
            id: "resver_dbschema_123456789",
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_dbschema_123456",
                    language: "en",
                    name: "PostgreSQL Database Schema",
                    description: "Relational database schema for user management system with tables, relationships, and constraints",
                    instructions: "## Database Schema Structure\n\n### Tables\n- **users**: User account information\n- **profiles**: Extended user profile data\n- **sessions**: Authentication sessions\n- **roles**: User roles and permissions\n\n### Relationships\n- One-to-one: user -> profile\n- One-to-many: user -> sessions\n- Many-to-many: users <-> roles\n\n### Constraints\n- Primary keys on all tables\n- Foreign key constraints\n- Unique constraints on email, username\n- Check constraints for data validation",
                    details: "Includes migration scripts, seed data, and performance optimization indexes. Compatible with PostgreSQL 12+.",
                },
            ],
            config: {
                schemaType: "sql",
                database: "postgresql",
                version: "12",
                includeIndexes: true,
                includeTriggers: true,
                enforceConstraints: true,
            },
        }],
        tags: [databaseTag, schemaTag],
    },
    jsonSchema: {
        ...minimalDataStructureResponse,
        id: "resource_jsonschema_123456789",
        versions: [{
            ...minimalDataStructureVersion,
            id: "resver_jsonschema_123456789",
            complexity: 4,
            simplicity: 7,
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_jsonschema_123456",
                    language: "en",
                    name: "REST API Response Schema",
                    description: "JSON Schema for standardized REST API responses with error handling",
                    instructions: "Standard response format for all API endpoints with success/error states and metadata",
                    details: "Includes HTTP status codes, error messages, pagination, and rate limiting headers",
                },
            ],
            config: {
                schemaType: "json-schema",
                version: "draft-07",
                strict: true,
                allowAdditionalProperties: false,
                includeExamples: true,
            },
        }],
        tags: [jsonSchemaTag, apiTag],
    },
    graphqlSchema: {
        ...completeDataStructureResponse,
        id: "resource_graphql_123456",
        versions: [{
            ...completeDataStructureVersion,
            id: "resver_graphql_123456",
            complexity: 9,
            simplicity: 3,
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_graphql_123",
                    language: "en",
                    name: "GraphQL Schema Definition",
                    description: "Complete GraphQL schema with types, queries, mutations, and subscriptions",
                    instructions: "## GraphQL Schema Components\n\n### Types\n- Scalar types: ID, String, Int, Float, Boolean\n- Object types: User, Product, Order\n- Input types: CreateUserInput, UpdateProductInput\n\n### Operations\n- Queries: getUser, listProducts, searchOrders\n- Mutations: createUser, updateProduct, deleteOrder\n- Subscriptions: userUpdated, orderCreated\n\n### Directives\n- @auth: Authentication requirement\n- @rateLimit: Rate limiting\n- @deprecated: Deprecated fields",
                    details: "Includes resolver implementations, authentication directives, and performance optimizations",
                },
            ],
            config: {
                schemaType: "graphql",
                version: "15",
                includeDirectives: true,
                includeSubscriptions: true,
                enableIntrospection: true,
            },
        }],
        tags: [apiTag, schemaTag],
    },
    xmlSchema: {
        ...minimalDataStructureResponse,
        id: "resource_xmlschema_123456",
        versions: [{
            ...minimalDataStructureVersion,
            id: "resver_xmlschema_123456",
            complexity: 6,
            simplicity: 4,
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_xmlschema_123",
                    language: "en",
                    name: "XML Schema Definition (XSD)",
                    description: "XML Schema for document validation and structure definition",
                    instructions: "Defines XML document structure with elements, attributes, and data types",
                    details: "Includes namespaces, complex types, and validation rules for XML documents",
                },
            ],
            config: {
                schemaType: "xsd",
                version: "1.1",
                targetNamespace: "http://example.com/schema",
                elementFormDefault: "qualified",
                includeAnnotations: true,
            },
        }],
        tags: [schemaTag],
    },
    deprecated: {
        ...completeDataStructureResponse,
        id: "resource_deprecated_123456",
        versions: [{
            ...completeDataStructureVersion,
            id: "resver_deprecated_123456",
            isLatest: false,
            versionLabel: "1.0.0",
            versionNotes: "This version is deprecated. Please use v2.0.0 or higher.",
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_deprecated_123",
                    language: "en",
                    name: "Legacy User Schema",
                    description: "Deprecated user data structure - use UserV2 schema instead",
                    instructions: "This schema is no longer maintained. Migrate to the new version.",
                    details: "Migration guide available in documentation",
                },
            ],
        }],
        labels: [deprecatedLabel],
        tags: [schemaTag],
    },
    inDevelopment: {
        ...minimalDataStructureResponse,
        id: "resource_dev_123456789",
        hasCompleteVersion: false,
        versions: [{
            ...minimalDataStructureVersion,
            id: "resver_dev_123456789",
            isComplete: false,
            versionLabel: "0.3.0-beta",
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_dev_123456",
                    language: "en",
                    name: "Next-Gen Schema Format",
                    description: "Work in progress - new schema format with AI-powered validation",
                    instructions: "Schema under development - structure may change",
                    details: "This schema will support automatic validation and smart defaults using AI",
                },
            ],
        }],
        labels: [draftLabel],
    },
    popular: {
        ...completeDataStructureResponse,
        id: "resource_popular_123456",
        score: 9999,
        bookmarks: 2500,
        views: 75000,
        versions: [{
            ...completeDataStructureVersion,
            id: "resver_popular_123456",
            score: 9999,
            bookmarks: 2500,
            views: 75000,
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_popular_123",
                    language: "en",
                    name: "Universal Data Schema",
                    description: "The most popular and widely-used data structure template for modern applications",
                    instructions: "Industry-standard schema used by thousands of projects worldwide",
                    details: "Comprehensive, battle-tested schema with extensive community support and documentation",
                },
            ],
        }],
        tags: [schemaTag, jsonSchemaTag, apiTag, databaseTag],
    },
} as const;

/**
 * Data structure search response
 */
export const dataStructureSearchResponse = {
    __typename: "ResourceSearchResult",
    edges: [
        {
            __typename: "ResourceEdge",
            cursor: "cursor_1",
            node: dataStructureResponseVariants.complete,
        },
        {
            __typename: "ResourceEdge",
            cursor: "cursor_2",
            node: dataStructureResponseVariants.databaseSchema,
        },
        {
            __typename: "ResourceEdge",
            cursor: "cursor_3",
            node: dataStructureResponseVariants.jsonSchema,
        },
        {
            __typename: "ResourceEdge",
            cursor: "cursor_4",
            node: dataStructureResponseVariants.popular,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: true,
        hasPreviousPage: false,
        startCursor: "cursor_1",
        endCursor: "cursor_4",
    },
};

/**
 * Loading and error states for UI testing
 */
export const dataStructureUIStates = {
    loading: null,
    error: {
        code: "DATA_STRUCTURE_NOT_FOUND",
        message: "The requested data structure could not be found",
    },
    validationError: {
        code: "SCHEMA_VALIDATION_FAILED",
        message: "Schema validation failed. Please check your data structure definition.",
    },
    formatError: {
        code: "UNSUPPORTED_SCHEMA_FORMAT",
        message: "The schema format is not supported. Please use JSON Schema, XSD, or GraphQL.",
    },
    versionError: {
        code: "SCHEMA_VERSION_INCOMPATIBLE",
        message: "The schema version is incompatible with the current system.",
    },
    empty: {
        edges: [],
        pageInfo: {
            __typename: "PageInfo",
            hasNextPage: false,
            hasPreviousPage: false,
            startCursor: null,
            endCursor: null,
        },
    },
};
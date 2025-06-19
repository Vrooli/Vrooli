import { type Resource, type ResourceVersion, type ResourceVersionTranslation, type Label, type Tag, ResourceSubType } from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse } from "./userResponses.js";
import { minimalTeamResponse } from "./teamResponses.js";

/**
 * API response fixtures for data converters (Resources with CodeDataConverter subtype)
 * These represent what components receive from API calls
 */

/**
 * Mock tag data for data converters
 */
const dataProcessingTag: Tag = {
    __typename: "Tag",
    id: "tag_dataproc_123456789",
    tag: "data-processing",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 15,
    translations: [
        {
            __typename: "TagTranslation",
            id: "tagtrans_dataproc_123",
            language: "en",
            description: "Tools and utilities for processing and transforming data",
        },
    ],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

const jsonTag: Tag = {
    __typename: "Tag",
    id: "tag_json_123456789",
    tag: "json",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 32,
    translations: [
        {
            __typename: "TagTranslation",
            id: "tagtrans_json_123456",
            language: "en",
            description: "JavaScript Object Notation data format processing",
        },
    ],
    you: {
        __typename: "TagYou",
        isBookmarked: true,
    },
};

const csvTag: Tag = {
    __typename: "Tag",
    id: "tag_csv_123456789",
    tag: "csv",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 28,
    translations: [],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

/**
 * Mock label data for data converters
 */
const betaLabel: Label = {
    __typename: "Label",
    id: "label_beta_123456789",
    label: "Beta",
    color: "#ff9800",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    you: {
        __typename: "LabelYou",
        canDelete: false,
        canUpdate: false,
    },
};

const productionLabel: Label = {
    __typename: "Label",
    id: "label_prod_123456789",
    label: "Production Ready",
    color: "#4caf50",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    you: {
        __typename: "LabelYou",
        canDelete: true,
        canUpdate: true,
    },
};

/**
 * Minimal data converter version
 */
const minimalDataConverterVersion: ResourceVersion = {
    __typename: "ResourceVersion",
    id: "resver_123456789012345",
    versionLabel: "1.0.0",
    versionNotes: null,
    isLatest: true,
    isPrivate: false,
    isComplete: true,
    isAutomatable: false,
    resourceSubType: ResourceSubType.CodeDataConverter,
    codeLanguage: "javascript",
    complexity: 3,
    simplicity: 7,
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
    translations: [
        {
            __typename: "ResourceVersionTranslation",
            id: "resvtrans_123456789",
            language: "en",
            name: "JSON to CSV Converter",
            description: "Simple converter that transforms JSON data into CSV format",
            instructions: "Upload your JSON file and click convert",
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
        canRun: true,
        canUpdate: false,
    },
};

/**
 * Complete data converter version with all features
 */
const completeDataConverterVersion: ResourceVersion = {
    __typename: "ResourceVersion",
    id: "resver_987654321098765",
    versionLabel: "3.2.1",
    versionNotes: "Enhanced performance with batch processing support and error handling improvements",
    isLatest: true,
    isPrivate: false,
    isComplete: true,
    isAutomatable: true,
    resourceSubType: ResourceSubType.CodeDataConverter,
    codeLanguage: "python",
    complexity: 8,
    simplicity: 4,
    createdAt: "2023-06-01T00:00:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    completedAt: "2024-01-10T00:00:00Z",
    comments: [],
    commentsCount: 12,
    forks: [],
    forksCount: 3,
    pulls: [],
    pullsCount: 1,
    issues: [],
    issuesCount: 0,
    labels: [productionLabel],
    labelsCount: 1,
    reportsCount: 0,
    score: 89,
    bookmarks: 156,
    views: 2847,
    transfersCount: 0,
    directoryListings: [],
    directoryListingsCount: 2,
    isInternal: false,
    root: null,
    config: {
        supportedFormats: ["json", "csv", "xml", "yaml"],
        batchSize: 1000,
        timeout: 30000,
        validation: true,
        preserveOrder: true,
    },
    translations: [
        {
            __typename: "ResourceVersionTranslation",
            id: "resvtrans_987654321",
            language: "en",
            name: "Advanced Data Format Converter",
            description: "Professional-grade data converter supporting multiple formats with batch processing, validation, and error recovery",
            instructions: "## Usage Instructions\n\n1. **Select Input Format**: Choose from JSON, CSV, XML, or YAML\n2. **Upload File**: Select your data file (max 50MB)\n3. **Choose Output Format**: Pick your desired output format\n4. **Configure Options**: Set batch size, validation rules, and format-specific options\n5. **Process Data**: Click convert and monitor progress\n6. **Download Results**: Get your converted file and processing report\n\n## Advanced Features\n- Batch processing for large files\n- Schema validation\n- Error reporting and recovery\n- Custom transformation rules\n- Format-specific optimizations",
            details: "This converter uses optimized parsing algorithms and supports schema-based validation. It can handle files up to 50MB and processes data in configurable batches to ensure memory efficiency.",
        },
        {
            __typename: "ResourceVersionTranslation",
            id: "resvtrans_876543210",
            language: "es",
            name: "Convertidor Avanzado de Formatos de Datos",
            description: "Convertidor de datos de grado profesional que soporta múltiples formatos con procesamiento por lotes, validación y recuperación de errores",
            instructions: "## Instrucciones de Uso\n\n1. **Seleccionar Formato de Entrada**: Elige entre JSON, CSV, XML o YAML\n2. **Subir Archivo**: Selecciona tu archivo de datos (máx 50MB)",
            details: null,
        },
    ],
    resourceList: {
        __typename: "ResourceList",
        id: "reslist_converter_123",
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
        canReport: false, // Own resource
        canRun: true,
        canUpdate: true,
    },
};

/**
 * Minimal data converter API response
 */
export const minimalDataConverterResponse: Resource = {
    __typename: "Resource",
    id: "resource_123456789012345",
    isDeleted: false,
    isInternal: false,
    isPrivate: false,
    resourceType: "Code",
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
    permissions: JSON.stringify(["Read", "Run"]),
    versions: [minimalDataConverterVersion],
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
 * Complete data converter API response with all fields
 */
export const completeDataConverterResponse: Resource = {
    __typename: "Resource",
    id: "resource_987654321098765",
    isDeleted: false,
    isInternal: false,
    isPrivate: false,
    resourceType: "Code",
    createdAt: "2023-06-01T00:00:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    completedAt: "2024-01-10T00:00:00Z",
    score: 89,
    bookmarks: 156,
    views: 2847,
    bookmarkedBy: [],
    owner: {
        __typename: "Team",
        ...minimalTeamResponse,
    },
    hasCompleteVersion: true,
    labels: [productionLabel],
    tags: [dataProcessingTag, jsonTag, csvTag],
    permissions: JSON.stringify(["Read", "Run", "Fork"]),
    versions: [completeDataConverterVersion],
    versionsCount: 8,
    forksCount: 3,
    issues: [],
    issuesCount: 0,
    pullsCount: 1,
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
 * Private data converter response
 */
export const privateDataConverterResponse: Resource = {
    __typename: "Resource",
    id: "resource_private_123456",
    isDeleted: false,
    isInternal: false,
    isPrivate: true,
    resourceType: "Code",
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
    labels: [betaLabel],
    tags: [],
    permissions: JSON.stringify([]),
    versions: [{
        ...minimalDataConverterVersion,
        id: "resver_private_123456",
        isPrivate: true,
        isComplete: false,
        translations: [
            {
                __typename: "ResourceVersionTranslation",
                id: "resvtrans_private_123",
                language: "en",
                name: "Personal Data Processor",
                description: "Private data converter for internal use",
                instructions: "Internal tool - access restricted",
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
        canRead: false, // No access to private converter
        canTransfer: false,
        canUpdate: false,
        isBookmarked: false,
        isViewed: false,
        reaction: null,
    },
};

/**
 * Data converter variant states for testing
 */
export const dataConverterResponseVariants = {
    minimal: minimalDataConverterResponse,
    complete: completeDataConverterResponse,
    private: privateDataConverterResponse,
    xmlConverter: {
        ...completeDataConverterResponse,
        id: "resource_xml_123456789",
        versions: [{
            ...completeDataConverterVersion,
            id: "resver_xml_123456789",
            codeLanguage: "java",
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_xml_123456",
                    language: "en",
                    name: "XML Data Transformer",
                    description: "Specialized converter for XML to various formats with XPath support",
                    instructions: "Supports XPath queries and schema validation",
                    details: "Enterprise-grade XML processing with XSLT support",
                },
            ],
            config: {
                supportedFormats: ["xml", "json", "yaml"],
                xpathSupport: true,
                schemaValidation: true,
                xsltTransforms: true,
            },
        }],
        tags: [dataProcessingTag],
    },
    csvProcessor: {
        ...minimalDataConverterResponse,
        id: "resource_csv_123456789",
        versions: [{
            ...minimalDataConverterVersion,
            id: "resver_csv_123456789",
            codeLanguage: "python",
            complexity: 5,
            simplicity: 6,
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_csv_123456",
                    language: "en",
                    name: "CSV Batch Processor",
                    description: "High-performance CSV processor with filtering and aggregation",
                    instructions: "Upload CSV, set filters, choose output format",
                    details: "Optimized for large datasets with memory-efficient processing",
                },
            ],
            config: {
                supportedFormats: ["csv", "json", "parquet"],
                batchSize: 5000,
                filterSupport: true,
                aggregations: ["sum", "avg", "count", "group"],
            },
        }],
        tags: [csvTag, dataProcessingTag],
    },
    jsonValidator: {
        ...completeDataConverterResponse,
        id: "resource_jsonval_123456",
        versions: [{
            ...completeDataConverterVersion,
            id: "resver_jsonval_123456",
            codeLanguage: "typescript",
            isAutomatable: false,
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_jsonval_123",
                    language: "en",
                    name: "JSON Schema Validator & Converter",
                    description: "Validate JSON against schemas and convert to other formats",
                    instructions: "Provide JSON data and optional schema for validation",
                    details: "Supports JSON Schema draft-07 and custom validation rules",
                },
            ],
            config: {
                supportedFormats: ["json", "yaml", "xml"],
                schemaValidation: true,
                customRules: true,
                errorReporting: "detailed",
            },
        }],
        tags: [jsonTag, dataProcessingTag],
    },
    inDevelopment: {
        ...minimalDataConverterResponse,
        id: "resource_dev_123456789",
        hasCompleteVersion: false,
        versions: [{
            ...minimalDataConverterVersion,
            id: "resver_dev_123456789",
            isComplete: false,
            versionLabel: "0.1.0-alpha",
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_dev_123456",
                    language: "en",
                    name: "Next-Gen Data Converter",
                    description: "Work in progress - advanced AI-powered data transformation",
                    instructions: "Feature under development",
                    details: "This converter will use machine learning to automatically detect and convert between data formats",
                },
            ],
        }],
        labels: [betaLabel],
    },
    popular: {
        ...completeDataConverterResponse,
        id: "resource_popular_123456",
        score: 9999,
        bookmarks: 5000,
        views: 50000,
        versions: [{
            ...completeDataConverterVersion,
            id: "resver_popular_123456",
            score: 9999,
            bookmarks: 5000,
            views: 50000,
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_popular_123",
                    language: "en",
                    name: "Universal Data Converter",
                    description: "The most popular data converter supporting 20+ formats",
                    instructions: "Drag and drop any data file to get started",
                    details: "Industry standard converter trusted by thousands of developers worldwide",
                },
            ],
        }],
        tags: [dataProcessingTag, jsonTag, csvTag],
    },
} as const;

/**
 * Data converter search response
 */
export const dataConverterSearchResponse = {
    __typename: "ResourceSearchResult",
    edges: [
        {
            __typename: "ResourceEdge",
            cursor: "cursor_1",
            node: dataConverterResponseVariants.complete,
        },
        {
            __typename: "ResourceEdge",
            cursor: "cursor_2",
            node: dataConverterResponseVariants.xmlConverter,
        },
        {
            __typename: "ResourceEdge",
            cursor: "cursor_3",
            node: dataConverterResponseVariants.csvProcessor,
        },
        {
            __typename: "ResourceEdge",
            cursor: "cursor_4",
            node: dataConverterResponseVariants.popular,
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
export const dataConverterUIStates = {
    loading: null,
    error: {
        code: "DATA_CONVERTER_NOT_FOUND",
        message: "The requested data converter could not be found",
    },
    conversionError: {
        code: "DATA_CONVERSION_FAILED",
        message: "Failed to convert data. Please check your input format and try again.",
    },
    formatError: {
        code: "UNSUPPORTED_FORMAT",
        message: "The input data format is not supported by this converter.",
    },
    fileSizeError: {
        code: "FILE_SIZE_EXCEEDED",
        message: "File size exceeds the maximum limit of 50MB.",
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
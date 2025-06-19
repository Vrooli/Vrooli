import { type Routine, type RoutineVersion, type RoutineVersionInput, type RoutineVersionOutput, RoutineType } from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse } from "./userResponses.js";
import { minimalTeamResponse } from "./teamResponses.js";
import { aiTag, webDevTag } from "./projectResponses.js";

/**
 * API response fixtures for routines
 * These represent what components receive from API calls
 */

// Re-export tags for convenience
export { aiTag, webDevTag } from "./projectResponses.js";

/**
 * Mock routine input/output data
 */
const textInput: RoutineVersionInput = {
    __typename: "RoutineVersionInput",
    id: "routineinput_text_123",
    index: 0,
    isRequired: true,
    name: "userInput",
    translations: [
        {
            __typename: "RoutineVersionInputTranslation",
            id: "inputtrans_123456",
            language: "en",
            description: "Enter your text prompt",
            helpText: "Describe what you want to accomplish",
        },
    ],
};

const fileInput: RoutineVersionInput = {
    __typename: "RoutineVersionInput",
    id: "routineinput_file_123",
    index: 1,
    isRequired: false,
    name: "uploadedFile",
    translations: [
        {
            __typename: "RoutineVersionInputTranslation",
            id: "inputtrans_file_123",
            language: "en",
            description: "Upload a file (optional)",
            helpText: "Supported formats: PDF, TXT, CSV",
        },
    ],
};

const resultOutput: RoutineVersionOutput = {
    __typename: "RoutineVersionOutput",
    id: "routineoutput_result_123",
    index: 0,
    name: "result",
    translations: [
        {
            __typename: "RoutineVersionOutputTranslation",
            id: "outputtrans_123456",
            language: "en",
            description: "Generated result",
            helpText: "The processed output based on your input",
        },
    ],
};

/**
 * Minimal routine version
 */
const minimalRoutineVersion: RoutineVersion = {
    __typename: "RoutineVersion",
    id: "routver_123456789012345",
    versionLabel: "1.0.0",
    versionNotes: null,
    isLatest: true,
    isPrivate: false,
    isComplete: true,
    routineType: RoutineType.Api,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: "2024-01-01T00:00:00Z",
    complexity: 1,
    simplicity: 10,
    timesStarted: 0,
    timesCompleted: 0,
    smartContractCallId: null,
    apiCallId: null,
    buildCallId: null,
    dataStructureCallId: null,
    inputs: [],
    outputs: [],
    resourceList: null,
    suggestedNextRoutineVersions: [],
    translations: [
        {
            __typename: "RoutineVersionTranslation",
            id: "routvertrans_123456",
            language: "en",
            name: "Simple API Call",
            description: "A basic routine that makes an API call",
            instructions: "1. Run the routine\n2. Get results",
        },
    ],
    you: {
        __typename: "RoutineVersionYou",
        canComment: true,
        canCopy: true,
        canDelete: false,
        canReport: true,
        canRun: true,
        canUpdate: false,
        canRead: true,
    },
};

/**
 * Complete routine version with all fields
 */
const completeRoutineVersion: RoutineVersion = {
    __typename: "RoutineVersion",
    id: "routver_987654321098765",
    versionLabel: "2.5.0",
    versionNotes: "Enhanced AI capabilities and performance improvements",
    isLatest: true,
    isPrivate: false,
    isComplete: true,
    routineType: RoutineType.MultiStep,
    createdAt: "2023-06-01T00:00:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    completedAt: "2024-01-10T00:00:00Z",
    complexity: 8,
    simplicity: 3,
    timesStarted: 1250,
    timesCompleted: 1100,
    smartContractCallId: null,
    apiCallId: null,
    buildCallId: null,
    dataStructureCallId: null,
    inputs: [textInput, fileInput],
    outputs: [resultOutput],
    resourceList: {
        __typename: "ResourceList",
        id: "reslist_routine_123",
        usedFor: "Display",
        resources: [],
    },
    suggestedNextRoutineVersions: [],
    translations: [
        {
            __typename: "RoutineVersionTranslation",
            id: "routvertrans_987654",
            language: "en",
            name: "AI Content Generator",
            description: "Generate high-quality content using advanced AI models with customizable parameters",
            instructions: "## How to use this routine\n\n1. **Input your prompt** - Describe what content you want to generate\n2. **Optional: Upload reference file** - Provide additional context\n3. **Run the routine** - Click the Run button\n4. **Review results** - Check the generated content\n5. **Iterate if needed** - Adjust your prompt and run again",
            warnings: "This routine uses AI models. Results may vary and should be reviewed before use.",
        },
        {
            __typename: "RoutineVersionTranslation",
            id: "routvertrans_876543",
            language: "es",
            name: "Generador de Contenido IA",
            description: "Genera contenido de alta calidad usando modelos de IA avanzados con parámetros personalizables",
            instructions: "## Cómo usar esta rutina\n\n1. **Ingresa tu prompt** - Describe qué contenido quieres generar",
        },
    ],
    you: {
        __typename: "RoutineVersionYou",
        canComment: true,
        canCopy: true,
        canDelete: true,
        canReport: false,
        canRun: true,
        canUpdate: true,
        canRead: true,
    },
};

/**
 * Minimal routine API response
 */
export const minimalRoutineResponse: Routine = {
    __typename: "Routine",
    id: "routine_123456789012345",
    handle: "simple-api-call",
    isInternal: false,
    isPrivate: false,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    score: 0,
    views: 0,
    owner: {
        __typename: "User",
        ...minimalUserResponse,
    },
    hasCompleteVersion: true,
    tags: [],
    labels: [],
    permissions: JSON.stringify(["Read"]),
    versions: [minimalRoutineVersion],
    versionsCount: 1,
    you: {
        __typename: "RoutineYou",
        canComment: true,
        canDelete: false,
        canBookmark: true,
        canReact: true,
        canRead: true,
        canReport: true,
        canUpdate: false,
        canTransfer: false,
        isBookmarked: false,
        reaction: null,
        isReported: false,
        isViewed: false,
    },
};

/**
 * Complete routine API response with all fields
 */
export const completeRoutineResponse: Routine = {
    __typename: "Routine",
    id: "routine_987654321098765",
    handle: "ai-content-generator",
    isInternal: false,
    isPrivate: false,
    createdAt: "2023-06-01T00:00:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    score: 456,
    views: 8901,
    owner: {
        __typename: "Team",
        ...minimalTeamResponse,
    },
    hasCompleteVersion: true,
    tags: [aiTag, webDevTag],
    labels: [],
    permissions: JSON.stringify(["Read", "Run"]),
    versions: [completeRoutineVersion],
    versionsCount: 12,
    transfersCount: 0,
    you: {
        __typename: "RoutineYou",
        canComment: true,
        canDelete: false,
        canBookmark: true,
        canReact: true,
        canRead: true,
        canReport: true,
        canUpdate: false,
        canTransfer: false,
        isBookmarked: true,
        reaction: "like",
        isReported: false,
        isViewed: true,
    },
};

/**
 * Internal routine (system routine)
 */
export const internalRoutineResponse: Routine = {
    __typename: "Routine",
    id: "routine_internal_123456",
    handle: null, // Internal routines might not have handles
    isInternal: true,
    isPrivate: false,
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    score: 0,
    views: 0,
    owner: {
        __typename: "User",
        ...botUserResponse,
    },
    hasCompleteVersion: true,
    tags: [],
    labels: [],
    permissions: JSON.stringify(["Read", "Run"]),
    versions: [{
        ...minimalRoutineVersion,
        id: "routver_internal_123",
        routineType: RoutineType.Informational,
        translations: [
            {
                __typename: "RoutineVersionTranslation",
                id: "routvertrans_internal",
                language: "en",
                name: "System Health Check",
                description: "Internal routine for monitoring system health",
                instructions: "This routine runs automatically",
            },
        ],
    }],
    versionsCount: 1,
    you: {
        __typename: "RoutineYou",
        canComment: false,
        canDelete: false,
        canBookmark: false,
        canReact: false,
        canRead: true,
        canReport: false,
        canUpdate: false,
        canTransfer: false,
        isBookmarked: false,
        reaction: null,
        isReported: false,
        isViewed: false,
    },
};

/**
 * Routine variant states for testing
 */
export const routineResponseVariants = {
    minimal: minimalRoutineResponse,
    complete: completeRoutineResponse,
    internal: internalRoutineResponse,
    dataProcessing: {
        ...completeRoutineResponse,
        id: "routine_data_123456789",
        handle: "csv-processor",
        versions: [{
            ...completeRoutineVersion,
            id: "routver_data_123456",
            routineType: RoutineType.Data,
            translations: [
                {
                    __typename: "RoutineVersionTranslation",
                    id: "routvertrans_data",
                    language: "en",
                    name: "CSV Data Processor",
                    description: "Process and analyze CSV files",
                    instructions: "Upload a CSV file to begin",
                },
            ],
        }],
    },
    multiStep: {
        ...completeRoutineResponse,
        id: "routine_multi_123456",
        handle: "complex-workflow",
        versions: [{
            ...completeRoutineVersion,
            id: "routver_multi_123",
            routineType: RoutineType.MultiStep,
            complexity: 10,
            simplicity: 1,
        }],
    },
    smartContract: {
        ...minimalRoutineResponse,
        id: "routine_contract_123",
        handle: "blockchain-interact",
        versions: [{
            ...minimalRoutineVersion,
            id: "routver_contract_123",
            routineType: RoutineType.SmartContract,
            smartContractCallId: "contract_123456789",
        }],
    },
} as const;

/**
 * Routine search response
 */
export const routineSearchResponse = {
    __typename: "RoutineSearchResult",
    edges: [
        {
            __typename: "RoutineEdge",
            cursor: "cursor_1",
            node: routineResponseVariants.complete,
        },
        {
            __typename: "RoutineEdge",
            cursor: "cursor_2",
            node: routineResponseVariants.minimal,
        },
        {
            __typename: "RoutineEdge",
            cursor: "cursor_3",
            node: routineResponseVariants.dataProcessing,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: true,
        hasPreviousPage: false,
        startCursor: "cursor_1",
        endCursor: "cursor_3",
    },
};

/**
 * Loading and error states for UI testing
 */
export const routineUIStates = {
    loading: null,
    error: {
        code: "ROUTINE_NOT_FOUND",
        message: "The requested routine could not be found",
    },
    executionError: {
        code: "ROUTINE_EXECUTION_FAILED",
        message: "Failed to execute routine. Please check your inputs and try again.",
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
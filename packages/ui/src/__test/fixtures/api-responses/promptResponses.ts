import { type Resource, type ResourceVersion, type ResourceVersionTranslation, type Tag, ResourceSubType, ResourceType } from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse } from "./userResponses.js";
import { minimalTeamResponse } from "./teamResponses.js";

/**
 * API response fixtures for prompts (StandardPrompt resources)
 * These represent what components receive from API calls
 */

/**
 * Mock tag data for prompts
 */
const aiTag: Tag = {
    __typename: "Tag",
    id: "tag_ai_123456789",
    tag: "artificial-intelligence",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 0,
    translations: [
        {
            __typename: "TagTranslation",
            id: "tagtrans_ai_123456789",
            language: "en",
            description: "Related to artificial intelligence and machine learning",
        },
    ],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

const promptTag: Tag = {
    __typename: "Tag",
    id: "tag_prompt_123456789",
    tag: "prompt-engineering",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 0,
    translations: [
        {
            __typename: "TagTranslation",
            id: "tagtrans_prompt_123456789",
            language: "en",
            description: "Related to prompt engineering and LLM interactions",
        },
    ],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

const llmTag: Tag = {
    __typename: "Tag",
    id: "tag_llm_123456789",
    tag: "large-language-model",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 0,
    translations: [],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

/**
 * Minimal prompt version
 */
const minimalPromptVersion: ResourceVersion = {
    __typename: "ResourceVersion",
    id: "promptver_123456789012345",
    versionIndex: 1,
    versionLabel: "1.0.0",
    versionNotes: null,
    publicId: "prompt001",
    isLatest: true,
    isPrivate: false,
    isComplete: true,
    isDeleted: false,
    resourceSubType: ResourceSubType.StandardPrompt,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: "2024-01-01T00:00:00Z",
    complexity: 1,
    isAutomatable: false,
    timesStarted: 0,
    timesCompleted: 0,
    config: {
        __version: "1.0",
        resources: [],
        schema: JSON.stringify({
            type: "object",
            properties: {
                content: { type: "string", description: "The prompt content" },
                model: { type: "string", description: "Target AI model" },
            },
            required: ["content"],
        }),
        schemaLanguage: "json-schema",
        validation: {
            strictMode: false,
        },
    },
    translations: [
        {
            __typename: "ResourceVersionTranslation",
            id: "promptvertrans_123456",
            language: "en",
            name: "Simple Chat Prompt",
            description: "A basic prompt for general conversation",
            instructions: "Use this prompt to start a conversation with an AI assistant",
            details: null,
        },
    ] as ResourceVersionTranslation[],
    root: {} as Resource, // Will be filled by parent
    comments: [],
    commentsCount: 0,
    forks: [],
    forksCount: 0,
    pullRequest: null,
    relatedVersions: [],
    reports: [],
    reportsCount: 0,
    translationsCount: 1,
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
 * Complete prompt version with all fields
 */
const completePromptVersion: ResourceVersion = {
    __typename: "ResourceVersion",
    id: "promptver_987654321098765",
    versionIndex: 3,
    versionLabel: "2.1.0",
    versionNotes: "Enhanced with temperature controls and better examples",
    publicId: "prompt_advanced_001",
    isLatest: true,
    isPrivate: false,
    isComplete: true,
    isDeleted: false,
    resourceSubType: ResourceSubType.StandardPrompt,
    createdAt: "2023-06-01T00:00:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    completedAt: "2024-01-10T00:00:00Z",
    complexity: 5,
    isAutomatable: true,
    timesStarted: 450,
    timesCompleted: 425,
    config: {
        __version: "1.0",
        resources: [
            {
                link: "https://example.com/prompt-guide",
                translations: [
                    {
                        language: "en",
                        name: "Prompt Engineering Guide",
                    },
                ],
            },
        ],
        schema: JSON.stringify({
            type: "object",
            properties: {
                content: { 
                    type: "string", 
                    description: "The main prompt content",
                    minLength: 10,
                    maxLength: 5000,
                },
                systemPrompt: { 
                    type: "string", 
                    description: "System-level instructions" 
                },
                model: { 
                    type: "string", 
                    enum: ["gpt-4", "gpt-3.5-turbo", "claude-3", "llama-2"],
                    description: "Target AI model" 
                },
                temperature: { 
                    type: "number", 
                    minimum: 0, 
                    maximum: 2,
                    description: "Controls randomness in responses" 
                },
                maxTokens: { 
                    type: "integer", 
                    minimum: 1, 
                    maximum: 4000,
                    description: "Maximum response length" 
                },
            },
            required: ["content", "model"],
        }),
        schemaLanguage: "json-schema",
        validation: {
            strictMode: true,
            rules: {
                maxLength: 5000,
                minLength: 10,
            },
            errorMessages: {
                maxLength: "Prompt content too long",
                minLength: "Prompt content too short",
                required: "This field is required",
            },
        },
        format: {
            defaultFormat: "markdown",
            options: {
                highlightSyntax: true,
                lineNumbers: false,
            },
        },
        compatibility: {
            minimumRequirements: { 
                apiVersion: ">=2.0.0" 
            },
            knownIssues: [],
            compatibleWith: ["GPT-4", "Claude-3", "Llama-2"],
        },
        props: {
            category: "conversational",
            difficulty: "intermediate",
            estimatedTokens: 150,
        },
    },
    translations: [
        {
            __typename: "ResourceVersionTranslation",
            id: "promptvertrans_987654",
            language: "en",
            name: "Advanced Code Review Prompt",
            description: "Comprehensive prompt for AI-assisted code reviews with best practices and security checks",
            instructions: "## Usage Instructions\n\n1. **Input your code** - Paste the code you want reviewed\n2. **Specify language** - Mention the programming language\n3. **Set review focus** - Choose from security, performance, or general review\n4. **Review results** - Analyze the AI feedback and suggestions\n\n## Best Practices\n\n- Use specific, clear descriptions\n- Include context about the code's purpose\n- Specify any particular concerns or areas of focus",
            details: "This prompt is designed for professional code review scenarios. It includes guidelines for security analysis, performance optimization, and code quality assessment. The prompt has been tested with various programming languages and coding styles.",
        },
        {
            __typename: "ResourceVersionTranslation",
            id: "promptvertrans_876543",
            language: "es",
            name: "Prompt Avanzado de Revisión de Código",
            description: "Prompt integral para revisiones de código asistidas por IA con mejores prácticas y verificaciones de seguridad",
            instructions: "## Instrucciones de Uso\n\n1. **Ingresa tu código** - Pega el código que quieres revisar\n2. **Especifica el lenguaje** - Menciona el lenguaje de programación\n3. **Establece el enfoque de revisión** - Elige entre seguridad, rendimiento o revisión general",
            details: null,
        },
    ] as ResourceVersionTranslation[],
    root: {} as Resource, // Will be filled by parent
    comments: [],
    commentsCount: 0,
    forks: [],
    forksCount: 0,
    pullRequest: null,
    relatedVersions: [],
    reports: [],
    reportsCount: 0,
    translationsCount: 2,
    you: {
        __typename: "ResourceVersionYou",
        canBookmark: true,
        canComment: true,
        canCopy: true,
        canDelete: true,
        canReact: true,
        canRead: true,
        canReport: false,
        canRun: true,
        canUpdate: true,
    },
};

/**
 * System prompt version for internal use
 */
const systemPromptVersion: ResourceVersion = {
    __typename: "ResourceVersion",
    id: "promptver_system_123456",
    versionIndex: 1,
    versionLabel: "1.0.0",
    versionNotes: null,
    publicId: "sys_prompt_001",
    isLatest: true,
    isPrivate: true,
    isComplete: true,
    isDeleted: false,
    resourceSubType: ResourceSubType.StandardPrompt,
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: "2023-01-01T00:00:00Z",
    complexity: 3,
    isAutomatable: true,
    timesStarted: 1200,
    timesCompleted: 1150,
    config: {
        __version: "1.0",
        resources: [],
        schema: JSON.stringify({
            type: "object",
            properties: {
                role: { type: "string", enum: ["system", "assistant", "user"] },
                content: { type: "string", description: "System instructions" },
                constraints: { 
                    type: "array", 
                    items: { type: "string" },
                    description: "Behavioral constraints" 
                },
            },
            required: ["role", "content"],
        }),
        schemaLanguage: "json-schema",
        validation: {
            strictMode: true,
        },
        props: {
            category: "system",
            security: "high",
            internal: true,
        },
    },
    translations: [
        {
            __typename: "ResourceVersionTranslation",
            id: "promptvertrans_system",
            language: "en",
            name: "System Initialization Prompt",
            description: "Internal system prompt for AI assistant initialization",
            instructions: "This prompt is used internally by the system. Do not modify without authorization.",
            details: "Contains system-level instructions and behavioral constraints for AI assistants.",
        },
    ] as ResourceVersionTranslation[],
    root: {} as Resource, // Will be filled by parent
    comments: [],
    commentsCount: 0,
    forks: [],
    forksCount: 0,
    pullRequest: null,
    relatedVersions: [],
    reports: [],
    reportsCount: 0,
    translationsCount: 1,
    you: {
        __typename: "ResourceVersionYou",
        canBookmark: false,
        canComment: false,
        canCopy: false,
        canDelete: false,
        canReact: false,
        canRead: true,
        canReport: false,
        canRun: true,
        canUpdate: false,
    },
};

/**
 * Minimal prompt API response
 */
export const minimalPromptResponse: Resource = {
    __typename: "Resource",
    id: "prompt_123456789012345",
    publicId: "prompt001",
    isInternal: false,
    isPrivate: false,
    isDeleted: false,
    resourceType: ResourceType.Standard,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: "2024-01-01T00:00:00Z",
    score: 0,
    views: 0,
    bookmarks: 0,
    owner: {
        __typename: "User",
        ...minimalUserResponse,
    },
    createdBy: {
        __typename: "User",
        ...minimalUserResponse,
    },
    hasCompleteVersion: true,
    tags: [],
    permissions: JSON.stringify(["Read"]),
    versions: [{ ...minimalPromptVersion, root: {} as Resource }],
    versionsCount: 1,
    translatedName: "Simple Chat Prompt",
    bookmarkedBy: [],
    issues: [],
    issuesCount: 0,
    parent: null,
    pullRequests: [],
    pullRequestsCount: 0,
    stats: [],
    transfers: [],
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

// Fix circular reference
minimalPromptResponse.versions[0].root = minimalPromptResponse;

/**
 * Complete prompt API response with all fields
 */
export const completePromptResponse: Resource = {
    __typename: "Resource",
    id: "prompt_987654321098765",
    publicId: "prompt_advanced_001",
    isInternal: false,
    isPrivate: false,
    isDeleted: false,
    resourceType: ResourceType.Standard,
    createdAt: "2023-06-01T00:00:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    completedAt: "2024-01-10T00:00:00Z",
    score: 89,
    views: 1450,
    bookmarks: 67,
    owner: {
        __typename: "Team",
        ...minimalTeamResponse,
    },
    createdBy: {
        __typename: "User",
        ...completeUserResponse,
    },
    hasCompleteVersion: true,
    tags: [aiTag, promptTag, llmTag],
    permissions: JSON.stringify(["Read", "Run"]),
    versions: [{ ...completePromptVersion, root: {} as Resource }],
    versionsCount: 3,
    translatedName: "Advanced Code Review Prompt",
    bookmarkedBy: [],
    issues: [],
    issuesCount: 0,
    parent: null,
    pullRequests: [],
    pullRequestsCount: 0,
    stats: [],
    transfers: [],
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
        isBookmarked: true,
        isViewed: true,
        reaction: "like",
    },
};

// Fix circular reference
completePromptResponse.versions[0].root = completePromptResponse;

/**
 * System prompt response (internal)
 */
export const systemPromptResponse: Resource = {
    __typename: "Resource",
    id: "prompt_system_123456",
    publicId: "sys_prompt_001", 
    isInternal: true,
    isPrivate: true,
    isDeleted: false,
    resourceType: ResourceType.Standard,
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: "2023-01-01T00:00:00Z",
    score: 0,
    views: 0,
    bookmarks: 0,
    owner: {
        __typename: "User",
        ...minimalUserResponse,
        isBot: true,
        handle: "system",
        name: "System",
    },
    createdBy: null,
    hasCompleteVersion: true,
    tags: [],
    permissions: JSON.stringify(["Read"]),
    versions: [{ ...systemPromptVersion, root: {} as Resource }],
    versionsCount: 1,
    translatedName: "System Initialization Prompt",
    bookmarkedBy: [],
    issues: [],
    issuesCount: 0,
    parent: null,
    pullRequests: [],
    pullRequestsCount: 0,
    stats: [],
    transfers: [],
    transfersCount: 0,
    you: {
        __typename: "ResourceYou",
        canBookmark: false,
        canComment: false,
        canDelete: false,
        canReact: false,
        canRead: true,
        canTransfer: false,
        canUpdate: false,
        isBookmarked: false,
        isViewed: false,
        reaction: null,
    },
};

// Fix circular reference
systemPromptResponse.versions[0].root = systemPromptResponse;

/**
 * Private prompt response
 */
export const privatePromptResponse: Resource = {
    __typename: "Resource",
    id: "prompt_private_123456",
    publicId: "private_prompt_001",
    isInternal: false,
    isPrivate: true,
    isDeleted: false,
    resourceType: ResourceType.Standard,
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: "2024-01-01T00:00:00Z",
    score: 0,
    views: 5,
    bookmarks: 1,
    owner: {
        __typename: "User",
        ...completeUserResponse,
    },
    createdBy: {
        __typename: "User",
        ...completeUserResponse,
    },
    hasCompleteVersion: true,
    tags: [promptTag],
    permissions: JSON.stringify([]),
    versions: [{
        ...minimalPromptVersion,
        id: "promptver_private_123456",
        isPrivate: true,
        root: {} as Resource,
    }],
    versionsCount: 1,
    translatedName: "Private Development Prompt",
    bookmarkedBy: [],
    issues: [],
    issuesCount: 0,
    parent: null,
    pullRequests: [],
    pullRequestsCount: 0,
    stats: [],
    transfers: [],
    transfersCount: 0,
    you: {
        __typename: "ResourceYou",
        canBookmark: false,
        canComment: false,
        canDelete: false,
        canReact: false,
        canRead: false, // No access to private prompt
        canTransfer: false,
        canUpdate: false,
        isBookmarked: false,
        isViewed: false,
        reaction: null,
    },
};

// Fix circular reference
privatePromptResponse.versions[0].root = privatePromptResponse;

/**
 * Prompt variant states for testing
 */
export const promptResponseVariants = {
    minimal: minimalPromptResponse,
    complete: completePromptResponse,
    system: systemPromptResponse,
    private: privatePromptResponse,
    chatPrompt: {
        ...minimalPromptResponse,
        id: "prompt_chat_123456789",
        publicId: "chat_prompt_001",
        versions: [{
            ...minimalPromptVersion,
            id: "promptver_chat_123456",
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "promptvertrans_chat",
                    language: "en",
                    name: "Friendly Chat Assistant",
                    description: "A conversational prompt for casual interactions",
                    instructions: "Use this for general conversation and Q&A",
                    details: null,
                },
            ] as ResourceVersionTranslation[],
            config: {
                __version: "1.0",
                resources: [],
                props: {
                    category: "conversational",
                    tone: "friendly",
                    formality: "casual",
                },
            },
            root: {} as Resource,
        }],
        translatedName: "Friendly Chat Assistant",
    },
    codePrompt: {
        ...completePromptResponse,
        id: "prompt_code_123456789",
        publicId: "code_prompt_001",
        versions: [{
            ...completePromptVersion,
            id: "promptver_code_123456",
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "promptvertrans_code",
                    language: "en",
                    name: "Code Generation Assistant",
                    description: "Specialized prompt for generating and explaining code",
                    instructions: "Provide clear specifications for the code you need",
                    details: "Optimized for multiple programming languages with best practices",
                },
            ] as ResourceVersionTranslation[],
            config: {
                __version: "1.0",
                resources: [],
                props: {
                    category: "code-generation",
                    languages: ["python", "javascript", "typescript", "java"],
                    complexity: "intermediate",
                },
            },
            root: {} as Resource,
        }],
        tags: [aiTag, promptTag],
        translatedName: "Code Generation Assistant",
    },
    completionPrompt: {
        ...minimalPromptResponse,
        id: "prompt_completion_123456",
        publicId: "completion_prompt_001",
        versions: [{
            ...minimalPromptVersion,
            id: "promptver_completion_123456",
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "promptvertrans_completion",
                    language: "en",
                    name: "Text Completion Helper",
                    description: "Prompt for completing partial text or ideas",
                    instructions: "Provide the beginning of your text or idea",
                    details: null,
                },
            ] as ResourceVersionTranslation[],
            config: {
                __version: "1.0",
                resources: [],
                props: {
                    category: "text-completion",
                    maxLength: 1000,
                    style: "creative",
                },
            },
            root: {} as Resource,
        }],
        translatedName: "Text Completion Helper",
    },
} as const;

// Fix circular references for variants
promptResponseVariants.chatPrompt.versions[0].root = promptResponseVariants.chatPrompt as Resource;
promptResponseVariants.codePrompt.versions[0].root = promptResponseVariants.codePrompt as Resource;
promptResponseVariants.completionPrompt.versions[0].root = promptResponseVariants.completionPrompt as Resource;

/**
 * Prompt search response
 */
export const promptSearchResponse = {
    __typename: "ResourceSearchResult",
    edges: [
        {
            __typename: "ResourceEdge",
            cursor: "cursor_1",
            node: promptResponseVariants.complete,
        },
        {
            __typename: "ResourceEdge",
            cursor: "cursor_2",
            node: promptResponseVariants.chatPrompt,
        },
        {
            __typename: "ResourceEdge",
            cursor: "cursor_3",
            node: promptResponseVariants.codePrompt,
        },
        {
            __typename: "ResourceEdge",
            cursor: "cursor_4",
            node: promptResponseVariants.completionPrompt,
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
export const promptUIStates = {
    loading: null,
    error: {
        code: "PROMPT_NOT_FOUND",
        message: "The requested prompt could not be found",
    },
    versionError: {
        code: "PROMPT_VERSION_NOT_FOUND", 
        message: "The requested prompt version could not be found",
    },
    validationError: {
        code: "PROMPT_VALIDATION_FAILED",
        message: "Prompt validation failed. Please check your input format.",
    },
    executionError: {
        code: "PROMPT_EXECUTION_FAILED",
        message: "Failed to execute prompt. Please try again or contact support.",
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
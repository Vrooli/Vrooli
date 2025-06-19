import { type User, type UserYou, type Translation } from "@vrooli/shared";

/**
 * API response fixtures for bots
 * These represent what components receive from API calls
 * Note: Bot is a User type with isBot: true
 */

/**
 * Minimal bot API response
 */
export const minimalBotResponse: User = {
    __typename: "User",
    id: "bot_123456789012345678",
    handle: "helpbot",
    name: "Help Bot",
    isBot: true,
    isBotDepictingPerson: false,
    isPrivate: false,
    status: "Unlocked",
    bannerImage: null,
    profileImage: null,
    theme: "light",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    // Bot-specific settings
    botSettings: {
        __version: "1.0.0",
        model: "gpt-4",
        temperature: 0.7,
        maxTokens: 2048,
        systemPrompt: "You are a helpful assistant.",
    },
    // Counts (following userResponses.ts pattern)
    awardedCount: 0,
    bookmarksCount: 0,
    commentsCount: 0,
    issuesCount: 0,
    postsCount: 0,
    projectsCount: 0,
    pullRequestsCount: 0,
    questionsCount: 0,
    quizzesCount: 0,
    reportsReceivedCount: 0,
    routinesCount: 2, // Bots typically have some routines
    standardsCount: 0,
    teamsCount: 0,
    translations: [],
    you: {
        __typename: "UserYou",
        canDelete: false,
        canReport: true,
        canUpdate: false,
        isBookmarked: false,
        isReported: false,
        reaction: null,
    },
};

/**
 * Complete bot API response with all fields
 */
export const completeBotResponse: User = {
    __typename: "User",
    id: "bot_987654321098765432",
    handle: "smartbot",
    name: "Smart Assistant Bot",
    isBot: true,
    isBotDepictingPerson: false,
    isPrivate: false,
    status: "Unlocked",
    bannerImage: "https://example.com/bot-banner.jpg",
    profileImage: "https://example.com/bot-avatar.png",
    theme: "dark",
    createdAt: "2023-06-15T10:30:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    // Advanced bot settings
    botSettings: {
        __version: "1.2.0",
        model: "claude-3-sonnet",
        temperature: 0.5,
        maxTokens: 4096,
        systemPrompt: "You are an advanced AI assistant specialized in helping developers with coding tasks.",
        assistantId: "asst_abc123def456",
        customConfig: {
            codeReviewEnabled: true,
            suggestionsEnabled: true,
            learningMode: "adaptive",
            capabilities: ["code_analysis", "documentation", "testing"],
            preferences: {
                language: "TypeScript",
                framework: "React",
                testingFramework: "Jest",
            },
        },
    },
    // Bot activity metrics (higher than typical users)
    awardedCount: 8,
    bookmarksCount: 150,
    commentsCount: 2450, // High activity
    issuesCount: 12,
    postsCount: 35,
    projectsCount: 8,
    pullRequestsCount: 45,
    questionsCount: 180,
    quizzesCount: 15,
    reportsReceivedCount: 1,
    routinesCount: 25,
    standardsCount: 12,
    teamsCount: 3,
    translations: [
        {
            __typename: "UserTranslation",
            id: "bottrans_123456789012345678",
            language: "en",
            bio: "Advanced AI assistant specialized in software development and code optimization",
        },
        {
            __typename: "UserTranslation",
            id: "bottrans_987654321098765432",
            language: "es",
            bio: "Asistente de IA avanzado especializado en desarrollo de software y optimización de código",
        },
    ],
    you: {
        __typename: "UserYou",
        canDelete: false,
        canReport: true,
        canUpdate: false,
        isBookmarked: true,
        isReported: false,
        reaction: "like",
    },
};

/**
 * Bot depicting a person response
 */
export const personDepictingBotResponse: User = {
    __typename: "User",
    id: "bot_person_123456789012",
    handle: "virtualassistant",
    name: "Virtual Sarah",
    isBot: true,
    isBotDepictingPerson: true, // This bot represents a real person
    isPrivate: false,
    status: "Unlocked",
    bannerImage: null,
    profileImage: "https://example.com/sarah-avatar.jpg",
    theme: "light",
    createdAt: "2023-08-01T00:00:00Z",
    updatedAt: "2024-01-10T12:00:00Z",
    botSettings: {
        __version: "1.0.0",
        model: "gpt-4",
        temperature: 0.8,
        maxTokens: 1024,
        systemPrompt: "You are Sarah, a friendly virtual assistant. You represent a real person and should maintain that persona.",
        personality: "friendly",
        background: "I am a virtual representation of Sarah, here to help with customer support.",
    },
    awardedCount: 2,
    bookmarksCount: 45,
    commentsCount: 320,
    issuesCount: 3,
    postsCount: 12,
    projectsCount: 1,
    pullRequestsCount: 5,
    questionsCount: 85,
    quizzesCount: 2,
    reportsReceivedCount: 0,
    routinesCount: 8,
    standardsCount: 2,
    teamsCount: 1,
    translations: [
        {
            __typename: "UserTranslation",
            id: "bottrans_person_123456789",
            language: "en",
            bio: "Virtual assistant representing Sarah from customer support",
        },
    ],
    you: {
        __typename: "UserYou",
        canDelete: false,
        canReport: true,
        canUpdate: false,
        isBookmarked: false,
        isReported: false,
        reaction: null,
    },
};

/**
 * Private bot response
 */
export const privateBotResponse: User = {
    __typename: "User",
    id: "bot_private_123456789",
    handle: "secretbot",
    name: "Internal Bot",
    isBot: true,
    isBotDepictingPerson: false,
    isPrivate: true,
    status: "Unlocked",
    bannerImage: null,
    profileImage: null,
    theme: "light",
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    botSettings: {
        __version: "1.0.0",
        model: "gpt-3.5-turbo",
        temperature: 0.3,
        maxTokens: 1024,
        systemPrompt: "You are an internal company assistant with access to private information.",
    },
    // Private bots show limited metrics
    awardedCount: 0,
    bookmarksCount: 0,
    commentsCount: 0,
    issuesCount: 0,
    postsCount: 0,
    projectsCount: 0,
    pullRequestsCount: 0,
    questionsCount: 0,
    quizzesCount: 0,
    reportsReceivedCount: 0,
    routinesCount: 0,
    standardsCount: 0,
    teamsCount: 0,
    translations: [],
    you: {
        __typename: "UserYou",
        canDelete: false,
        canReport: false, // Can't report private bots
        canUpdate: false,
        isBookmarked: false,
        isReported: false,
        reaction: null,
    },
};

/**
 * Training bot response (bot in development/training mode)
 */
export const trainingBotResponse: User = {
    __typename: "User",
    id: "bot_training_123456789",
    handle: "traineebot",
    name: "Training Bot",
    isBot: true,
    isBotDepictingPerson: false,
    isPrivate: false,
    status: "Unlocked",
    bannerImage: null,
    profileImage: "https://example.com/training-bot.png",
    theme: "light",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-20T15:30:00Z",
    botSettings: {
        __version: "0.5.0",
        model: "gpt-4",
        temperature: 0.9, // Higher temperature for learning
        maxTokens: 512,
        systemPrompt: "You are a bot in training. Learn from interactions and improve responses.",
        trainingMode: true,
        learningSettings: {
            adaptiveResponses: true,
            feedbackCollection: true,
            errorCorrection: true,
        },
    },
    awardedCount: 0,
    bookmarksCount: 2,
    commentsCount: 15,
    issuesCount: 1,
    postsCount: 1,
    projectsCount: 0,
    pullRequestsCount: 0,
    questionsCount: 8,
    quizzesCount: 0,
    reportsReceivedCount: 0,
    routinesCount: 1,
    standardsCount: 0,
    teamsCount: 0,
    translations: [
        {
            __typename: "UserTranslation",
            id: "bottrans_training_123456789",
            language: "en",
            bio: "AI assistant currently in training mode - responses may vary as the bot learns",
        },
    ],
    you: {
        __typename: "UserYou",
        canDelete: false,
        canReport: true,
        canUpdate: false,
        isBookmarked: false,
        isReported: false,
        reaction: null,
    },
};

/**
 * Disabled/inactive bot response
 */
export const disabledBotResponse: User = {
    __typename: "User",
    id: "bot_disabled_123456789",
    handle: "oldbot",
    name: "Disabled Bot",
    isBot: true,
    isBotDepictingPerson: false,
    isPrivate: false,
    status: "SoftLocked", // Using AccountStatus enum value
    bannerImage: null,
    profileImage: null,
    theme: "light",
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2023-12-01T00:00:00Z",
    botSettings: {
        __version: "0.9.0",
        model: "gpt-3.5-turbo",
        temperature: 0.7,
        maxTokens: 1024,
        systemPrompt: "Bot is currently disabled and not responding.",
        enabled: false,
    },
    awardedCount: 1,
    bookmarksCount: 5,
    commentsCount: 85,
    issuesCount: 2,
    postsCount: 3,
    projectsCount: 0,
    pullRequestsCount: 0,
    questionsCount: 12,
    quizzesCount: 0,
    reportsReceivedCount: 3,
    routinesCount: 2,
    standardsCount: 0,
    teamsCount: 0,
    translations: [],
    you: {
        __typename: "UserYou",
        canDelete: false,
        canReport: false, // Can't report disabled bots
        canUpdate: false,
        isBookmarked: false,
        isReported: false,
        reaction: null,
    },
};

/**
 * High-activity specialist bot response
 */
export const specialistBotResponse: User = {
    __typename: "User",
    id: "bot_specialist_123456789",
    handle: "codeexpert",
    name: "Code Expert Bot",
    isBot: true,
    isBotDepictingPerson: false,
    isPrivate: false,
    status: "Unlocked",
    bannerImage: "https://example.com/code-expert-banner.jpg",
    profileImage: "https://example.com/code-expert-avatar.png",
    theme: "dark",
    createdAt: "2023-03-15T08:00:00Z",
    updatedAt: "2024-01-22T16:45:00Z",
    botSettings: {
        __version: "2.1.0",
        model: "claude-3-opus",
        temperature: 0.2, // Lower temperature for precise code assistance
        maxTokens: 8192,
        systemPrompt: "You are a code expert specializing in TypeScript, React, and Node.js. Provide accurate, well-commented code examples.",
        specialization: "software_development",
        expertise: ["TypeScript", "React", "Node.js", "GraphQL", "Testing"],
        capabilities: [
            "code_review",
            "debugging",
            "architecture_advice",
            "performance_optimization",
            "testing_strategies",
        ],
    },
    // High activity metrics for specialist
    awardedCount: 25,
    bookmarksCount: 892,
    commentsCount: 5420,
    issuesCount: 45,
    postsCount: 128,
    projectsCount: 32,
    pullRequestsCount: 234,
    questionsCount: 1250,
    quizzesCount: 48,
    reportsReceivedCount: 0,
    routinesCount: 156,
    standardsCount: 67,
    teamsCount: 12,
    translations: [
        {
            __typename: "UserTranslation",
            id: "bottrans_specialist_123456789",
            language: "en",
            bio: "Expert-level AI assistant specializing in modern web development technologies and best practices",
        },
        {
            __typename: "UserTranslation",
            id: "bottrans_specialist_987654321",
            language: "fr",
            bio: "Assistant IA expert spécialisé dans les technologies de développement web modernes et les meilleures pratiques",
        },
    ],
    you: {
        __typename: "UserYou",
        canDelete: false,
        canReport: true,
        canUpdate: false,
        isBookmarked: true,
        isReported: false,
        reaction: "like",
    },
};

/**
 * Bot variant states for testing
 */
export const botResponseVariants = {
    minimal: minimalBotResponse,
    complete: completeBotResponse,
    personDepicting: personDepictingBotResponse,
    private: privateBotResponse,
    training: trainingBotResponse,
    disabled: disabledBotResponse,
    specialist: specialistBotResponse,
    suspended: {
        ...minimalBotResponse,
        id: "bot_suspended_123456789",
        handle: "suspendedbot",
        name: "Suspended Bot",
        status: "HardLocked",
        you: {
            ...minimalBotResponse.you,
            canReport: false, // Can't report suspended bots
        },
    },
    deleted: {
        ...minimalBotResponse,
        id: "bot_deleted_123456789",
        handle: "[deleted]",
        name: "[Deleted Bot]",
        status: "Deleted",
        you: {
            ...minimalBotResponse.you,
            canReport: false,
        },
    },
} as const;

/**
 * Bot search response
 */
export const botSearchResponse = {
    __typename: "UserSearchResult",
    edges: [
        {
            __typename: "UserEdge",
            cursor: "cursor_1",
            node: botResponseVariants.complete,
        },
        {
            __typename: "UserEdge",
            cursor: "cursor_2",
            node: botResponseVariants.specialist,
        },
        {
            __typename: "UserEdge",
            cursor: "cursor_3",
            node: botResponseVariants.minimal,
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
export const botUIStates = {
    loading: null,
    error: {
        code: "BOT_NOT_FOUND",
        message: "The requested bot could not be found",
    },
    accessDenied: {
        code: "BOT_ACCESS_DENIED",
        message: "You don't have permission to view this bot",
    },
    botDisabled: {
        code: "BOT_DISABLED",
        message: "This bot is currently disabled and unavailable",
    },
    trainingMode: {
        code: "BOT_IN_TRAINING",
        message: "This bot is currently in training mode and responses may be limited",
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
import { type Chat, type ChatMessage, type ChatParticipant, type ChatYou, type ChatMessageYou } from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse, botUserResponse } from "./userResponses.js";
import { minimalTeamResponse } from "./teamResponses.js";

/**
 * API response fixtures for chats
 * These represent what components receive from API calls
 */

/**
 * Mock chat participant data
 */
const userParticipant: ChatParticipant = {
    __typename: "ChatParticipant",
    id: "participant_user_123456",
    user: minimalUserResponse,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
};

const botParticipant: ChatParticipant = {
    __typename: "ChatParticipant",
    id: "participant_bot_123456",
    user: botUserResponse,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
};

const currentUserParticipant: ChatParticipant = {
    __typename: "ChatParticipant",
    id: "participant_current_123",
    user: {
        ...completeUserResponse,
        id: "user_current_123456789",
        handle: "currentuser",
        name: "Current User",
    },
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
};

/**
 * Mock chat messages
 */
const textMessage: ChatMessage = {
    __typename: "ChatMessage",
    id: "message_text_123456789",
    created_at: "2024-01-20T10:00:00Z",
    updated_at: "2024-01-20T10:00:00Z",
    sequence: 1,
    user: minimalUserResponse,
    versionIndex: 0,
    parent: null,
    translations: [
        {
            __typename: "ChatMessageTranslation",
            id: "msgtrans_123456789",
            language: "en",
            text: "Hello! How can I help you today?",
        },
    ],
    score: 0,
    reportsCount: 0,
    you: {
        __typename: "ChatMessageYou",
        canDelete: false,
        canReply: true,
        canReport: true,
        canUpdate: false,
        canReact: true,
        isBookmarked: false,
        reaction: null,
        isReported: false,
    },
};

const botResponseMessage: ChatMessage = {
    __typename: "ChatMessage",
    id: "message_bot_123456789",
    created_at: "2024-01-20T10:01:00Z",
    updated_at: "2024-01-20T10:01:00Z",
    sequence: 2,
    user: botUserResponse,
    versionIndex: 0,
    parent: textMessage,
    translations: [
        {
            __typename: "ChatMessageTranslation",
            id: "msgtrans_bot_123456",
            language: "en",
            text: "I can help you with various tasks including:\n- Creating routines\n- Managing projects\n- Answering questions\n\nWhat would you like to do?",
        },
    ],
    score: 5,
    reportsCount: 0,
    you: {
        __typename: "ChatMessageYou",
        canDelete: false,
        canReply: true,
        canReport: true,
        canUpdate: false,
        canReact: true,
        isBookmarked: false,
        reaction: "like",
        isReported: false,
    },
};

/**
 * Minimal chat API response
 */
export const minimalChatResponse: Chat = {
    __typename: "Chat",
    id: "chat_123456789012345",
    openToAnyoneWithInvite: false,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-20T10:00:00Z",
    team: null,
    restrictedToRoles: [],
    participants: [currentUserParticipant, userParticipant],
    participantsCount: 2,
    messages: [],
    messagesCount: 0,
    translations: [
        {
            __typename: "ChatTranslation",
            id: "chattrans_123456789",
            language: "en",
            name: "General Chat",
            description: null,
        },
    ],
    you: {
        __typename: "ChatYou",
        canDelete: false,
        canInvite: true,
        canUpdate: false,
        canRead: true,
        isParticipant: true,
    },
};

/**
 * Complete chat API response with all fields
 */
export const completeChatResponse: Chat = {
    __typename: "Chat",
    id: "chat_987654321098765",
    openToAnyoneWithInvite: true,
    createdAt: "2023-12-01T00:00:00Z",
    updatedAt: "2024-01-20T15:30:00Z",
    team: minimalTeamResponse,
    restrictedToRoles: [],
    participants: [currentUserParticipant, userParticipant, botParticipant],
    participantsCount: 3,
    messages: [textMessage, botResponseMessage],
    messagesCount: 25,
    translations: [
        {
            __typename: "ChatTranslation",
            id: "chattrans_987654321",
            language: "en",
            name: "Project Discussion",
            description: "Discuss the AI Assistant project development",
        },
        {
            __typename: "ChatTranslation",
            id: "chattrans_876543210",
            language: "es",
            name: "Discusión del Proyecto",
            description: "Discutir el desarrollo del proyecto Asistente IA",
        },
    ],
    you: {
        __typename: "ChatYou",
        canDelete: true, // Chat owner
        canInvite: true,
        canUpdate: true,
        canRead: true,
        isParticipant: true,
    },
};

/**
 * Team chat with role restrictions
 */
export const teamChatResponse: Chat = {
    __typename: "Chat",
    id: "chat_team_123456789",
    openToAnyoneWithInvite: false,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-20T00:00:00Z",
    team: minimalTeamResponse,
    restrictedToRoles: [
        {
            __typename: "Role",
            id: "role_admin_123456",
            name: "Admin",
            permissions: ["Admin"],
            membersCount: 2,
            createdAt: "2024-01-01T00:00:00Z",
            updatedAt: "2024-01-01T00:00:00Z",
            translations: [],
        },
    ],
    participants: [currentUserParticipant],
    participantsCount: 5,
    messages: [],
    messagesCount: 150,
    translations: [
        {
            __typename: "ChatTranslation",
            id: "chattrans_team_123",
            language: "en",
            name: "Admin Discussion",
            description: "Private chat for team administrators",
        },
    ],
    you: {
        __typename: "ChatYou",
        canDelete: false,
        canInvite: false, // Restricted by roles
        canUpdate: false,
        canRead: true,
        isParticipant: true,
    },
};

/**
 * Direct message chat (1-on-1)
 */
export const directMessageChatResponse: Chat = {
    __typename: "Chat",
    id: "chat_dm_123456789",
    openToAnyoneWithInvite: false,
    createdAt: "2024-01-15T00:00:00Z",
    updatedAt: "2024-01-20T12:00:00Z",
    team: null,
    restrictedToRoles: [],
    participants: [currentUserParticipant, userParticipant],
    participantsCount: 2,
    messages: [
        {
            ...textMessage,
            id: "message_dm_1",
            user: currentUserParticipant.user,
            translations: [
                {
                    __typename: "ChatMessageTranslation",
                    id: "msgtrans_dm_1",
                    language: "en",
                    text: "Hey, quick question about the project",
                },
            ],
        },
    ],
    messagesCount: 10,
    translations: [], // DMs typically don't have names
    you: {
        __typename: "ChatYou",
        canDelete: false,
        canInvite: false, // Can't invite to DM
        canUpdate: false,
        canRead: true,
        isParticipant: true,
    },
};

/**
 * Chat variant states for testing
 */
export const chatResponseVariants = {
    minimal: minimalChatResponse,
    complete: completeChatResponse,
    teamChat: teamChatResponse,
    directMessage: directMessageChatResponse,
    largeGroup: {
        ...completeChatResponse,
        id: "chat_large_123456789",
        participantsCount: 50,
        messagesCount: 1000,
    },
    botOnly: {
        ...minimalChatResponse,
        id: "chat_bot_123456789",
        participants: [currentUserParticipant, botParticipant],
        participantsCount: 2,
        translations: [
            {
                __typename: "ChatTranslation",
                id: "chattrans_bot_chat",
                language: "en",
                name: "AI Assistant Chat",
                description: "Chat with the AI assistant",
            },
        ],
    },
} as const;

/**
 * Chat search response
 */
export const chatSearchResponse = {
    __typename: "ChatSearchResult",
    edges: [
        {
            __typename: "ChatEdge",
            cursor: "cursor_1",
            node: chatResponseVariants.complete,
        },
        {
            __typename: "ChatEdge",
            cursor: "cursor_2",
            node: chatResponseVariants.teamChat,
        },
        {
            __typename: "ChatEdge",
            cursor: "cursor_3",
            node: chatResponseVariants.directMessage,
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
export const chatUIStates = {
    loading: null,
    error: {
        code: "CHAT_NOT_FOUND",
        message: "The requested chat could not be found",
    },
    accessDenied: {
        code: "CHAT_ACCESS_DENIED",
        message: "You don't have permission to access this chat",
    },
    messageError: {
        code: "MESSAGE_SEND_FAILED",
        message: "Failed to send message. Please try again.",
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
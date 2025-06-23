import type { Meta, StoryObj } from "@storybook/react";
import { DUMMY_ID } from "@vrooli/shared";
import { ChatInviteListItem } from "./ChatInviteListItem.js";

const meta: Meta<typeof ChatInviteListItem> = {
    title: "Components/Chat/ChatInviteListItem",
    component: ChatInviteListItem,
    parameters: {
        layout: "centered",
    },
    tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Mock chat invite data
const mockChatInvite = {
    __typename: "ChatInvite" as const,
    id: DUMMY_ID,
    chat: {
        __typename: "Chat" as const,
        id: "chat-1",
        translations: [
            {
                __typename: "ChatTranslation" as const,
                id: DUMMY_ID,
                language: "en",
                name: "Project Planning Chat",
                description: "Discussion about the new website redesign",
            },
        ],
    },
    user: {
        __typename: "User" as const,
        id: "user-1",
        name: "John Doe",
        handle: "johndoe",
        profileImage: null,
        isBot: false,
        translations: [
            {
                __typename: "UserTranslation" as const,
                id: DUMMY_ID,
                language: "en",
                bio: "Software developer with expertise in React and Node.js",
            },
        ],
    },
    message: "Hi! I'd like to invite you to join our project planning discussion. We're discussing the new website redesign and could really use your input.",
    createdAt: new Date("2024-01-16T14:30:00Z").toISOString(),
    updatedAt: new Date("2024-01-16T14:30:00Z").toISOString(),
};

const mockChatInviteShortMessage = {
    __typename: "ChatInvite" as const,
    id: DUMMY_ID,
    chat: {
        __typename: "Chat" as const,
        id: "chat-2",
        translations: [
            {
                __typename: "ChatTranslation" as const,
                id: DUMMY_ID,
                language: "en",
                name: "Team Standup",
                description: "Daily team sync meeting chat",
            },
        ],
    },
    user: {
        __typename: "User" as const,
        id: "user-2",
        name: "Sarah Wilson",
        handle: "sarahw",
        profileImage: "/api/v1/file/avatar-sarah.jpg",
        isBot: false,
        translations: [
            {
                __typename: "UserTranslation" as const,
                id: DUMMY_ID,
                language: "en",
                bio: "Project manager and scrum master",
            },
        ],
    },
    message: "Join our daily standup!",
    createdAt: new Date("2024-01-16T09:00:00Z").toISOString(),
    updatedAt: new Date("2024-01-16T09:00:00Z").toISOString(),
};

const mockChatInviteNoMessage = {
    __typename: "ChatInvite" as const,
    id: DUMMY_ID,
    chat: {
        __typename: "Chat" as const,
        id: "chat-3",
        translations: [
            {
                __typename: "ChatTranslation" as const,
                id: DUMMY_ID,
                language: "en",
                name: "AI Research Group",
                description: "Discussion about AI and machine learning topics",
            },
        ],
    },
    user: {
        __typename: "User" as const,
        id: "user-3",
        name: "Dr. Alex Chen",
        handle: "alexchen",
        profileImage: null,
        isBot: false,
        translations: [
            {
                __typename: "UserTranslation" as const,
                id: DUMMY_ID,
                language: "en",
                bio: "AI researcher and data scientist",
            },
        ],
    },
    message: "",
    createdAt: new Date("2024-01-15T16:45:00Z").toISOString(),
    updatedAt: new Date("2024-01-15T16:45:00Z").toISOString(),
};

const mockBotInvite = {
    __typename: "ChatInvite" as const,
    id: DUMMY_ID,
    chat: {
        __typename: "Chat" as const,
        id: "chat-4",
        translations: [
            {
                __typename: "ChatTranslation" as const,
                id: DUMMY_ID,
                language: "en",
                name: "AI Assistant Support",
                description: "Get help with AI assistant features",
            },
        ],
    },
    user: {
        __typename: "User" as const,
        id: "bot-1",
        name: "Valyxa AI",
        handle: "valyxa",
        profileImage: null,
        isBot: true,
        translations: [
            {
                __typename: "UserTranslation" as const,
                id: DUMMY_ID,
                language: "en",
                bio: "Your intelligent AI assistant for productivity and automation",
            },
        ],
    },
    message: "I'm here to help you with any questions about using AI features in your projects. Join this chat to get personalized assistance!",
    createdAt: new Date("2024-01-16T12:00:00Z").toISOString(),
    updatedAt: new Date("2024-01-16T12:00:00Z").toISOString(),
};

export const Default: Story = {
    args: {
        data: mockChatInvite,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: false,
        isSelected: false,
    },
};

export const ShortMessage: Story = {
    args: {
        data: mockChatInviteShortMessage,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: false,
        isSelected: false,
    },
};

export const NoMessage: Story = {
    args: {
        data: mockChatInviteNoMessage,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: false,
        isSelected: false,
    },
};

export const FromBot: Story = {
    args: {
        data: mockBotInvite,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: false,
        isSelected: false,
    },
};

export const Selected: Story = {
    args: {
        data: mockChatInvite,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: true,
        isSelected: true,
    },
};

export const Selectable: Story = {
    args: {
        data: mockChatInvite,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: true,
        isSelected: false,
    },
};

import type { Meta, StoryObj } from "@storybook/react";
import { DUMMY_ID } from "@vrooli/shared";
import { ChatListItem } from "./ChatListItem.js";

const meta: Meta<typeof ChatListItem> = {
    title: "Components/Chat/ChatListItem",
    component: ChatListItem,
    parameters: {
        layout: "centered",
    },
    tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Mock chat data
const mockChat = {
    __typename: "Chat" as const,
    id: DUMMY_ID,
    publicId: "chat-123",
    isPrivate: false,
    participants: [
        {
            __typename: "ChatParticipant" as const,
            id: "participant-1",
            user: {
                __typename: "User" as const,
                id: "user-1",
                name: "John Doe",
                handle: "johndoe",
            },
        },
        {
            __typename: "ChatParticipant" as const,
            id: "participant-2",
            user: {
                __typename: "User" as const,
                id: "bot-1",
                name: "AI Assistant",
                handle: "ai-assistant",
                isBot: true,
            },
        },
    ],
    messages: [],
    translations: [
        {
            __typename: "ChatTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "Project Planning Discussion",
            description: "Chat about planning the new website redesign project",
        },
    ],
    createdAt: new Date("2024-01-15T10:30:00Z").toISOString(),
    updatedAt: new Date("2024-01-16T14:20:00Z").toISOString(),
};

const mockPrivateChat = {
    __typename: "Chat" as const,
    id: DUMMY_ID,
    publicId: "chat-456",
    isPrivate: true,
    participants: [
        {
            __typename: "ChatParticipant" as const,
            id: "participant-1",
            user: {
                __typename: "User" as const,
                id: "user-1",
                name: "Alice Johnson",
                handle: "alicej",
            },
        },
        {
            __typename: "ChatParticipant" as const,
            id: "participant-2",
            user: {
                __typename: "User" as const,
                id: "user-2",
                name: "Bob Smith",
                handle: "bobsmith",
            },
        },
    ],
    messages: [],
    translations: [
        {
            __typename: "ChatTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "Private Discussion",
            description: "Confidential conversation about project details",
        },
    ],
    createdAt: new Date("2024-01-14T09:15:00Z").toISOString(),
    updatedAt: new Date("2024-01-16T16:45:00Z").toISOString(),
};

const mockBotOnlyChat = {
    __typename: "Chat" as const,
    id: DUMMY_ID,
    publicId: "chat-789",
    isPrivate: false,
    participants: [
        {
            __typename: "ChatParticipant" as const,
            id: "participant-1",
            user: {
                __typename: "User" as const,
                id: "user-1",
                name: "Current User",
                handle: "currentuser",
            },
        },
        {
            __typename: "ChatParticipant" as const,
            id: "participant-2",
            user: {
                __typename: "User" as const,
                id: "bot-1",
                name: "Valyxa AI",
                handle: "valyxa",
                isBot: true,
            },
        },
    ],
    messages: [],
    translations: [
        {
            __typename: "ChatTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "AI Assistant Chat",
            description: "General purpose AI assistance and task automation",
        },
    ],
    createdAt: new Date("2024-01-16T08:00:00Z").toISOString(),
    updatedAt: new Date("2024-01-16T17:30:00Z").toISOString(),
};

const mockGroupChat = {
    __typename: "Chat" as const,
    id: DUMMY_ID,
    publicId: "chat-group-001",
    isPrivate: false,
    participants: [
        {
            __typename: "ChatParticipant" as const,
            id: "participant-1",
            user: {
                __typename: "User" as const,
                id: "user-1",
                name: "Team Lead",
                handle: "teamlead",
            },
        },
        {
            __typename: "ChatParticipant" as const,
            id: "participant-2",
            user: {
                __typename: "User" as const,
                id: "user-2",
                name: "Developer 1",
                handle: "dev1",
            },
        },
        {
            __typename: "ChatParticipant" as const,
            id: "participant-3",
            user: {
                __typename: "User" as const,
                id: "user-3",
                name: "Developer 2",
                handle: "dev2",
            },
        },
        {
            __typename: "ChatParticipant" as const,
            id: "participant-4",
            user: {
                __typename: "User" as const,
                id: "user-4",
                name: "Designer",
                handle: "designer",
            },
        },
    ],
    messages: [],
    translations: [
        {
            __typename: "ChatTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "Team Standup",
            description: "Daily standup discussions and sprint planning",
        },
    ],
    createdAt: new Date("2024-01-10T09:00:00Z").toISOString(),
    updatedAt: new Date("2024-01-16T11:15:00Z").toISOString(),
};

export const Default: Story = {
    args: {
        data: mockChat,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: false,
        isSelected: false,
        onAction: (action, id) => console.log("onAction", action, id),
    },
};

export const PrivateChat: Story = {
    args: {
        data: mockPrivateChat,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: false,
        isSelected: false,
        onAction: (action, id) => console.log("onAction", action, id),
    },
};

export const BotOnlyChat: Story = {
    args: {
        data: mockBotOnlyChat,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: false,
        isSelected: false,
        onAction: (action, id) => console.log("onAction", action, id),
    },
};

export const GroupChat: Story = {
    args: {
        data: mockGroupChat,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: false,
        isSelected: false,
        onAction: (action, id) => console.log("onAction", action, id),
    },
};

export const Selected: Story = {
    args: {
        data: mockChat,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: true,
        isSelected: true,
        onAction: (action, id) => console.log("onAction", action, id),
    },
};

export const Selectable: Story = {
    args: {
        data: mockChat,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: true,
        isSelected: false,
        onAction: (action, id) => console.log("onAction", action, id),
    },
};

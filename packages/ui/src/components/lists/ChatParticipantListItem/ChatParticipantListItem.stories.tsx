import type { Meta, StoryObj } from "@storybook/react";
import { DUMMY_ID } from "@vrooli/shared";
import { ChatParticipantListItem } from "./ChatParticipantListItem.js";

const meta: Meta<typeof ChatParticipantListItem> = {
    title: "Components/Chat/ChatParticipantListItem",
    component: ChatParticipantListItem,
    parameters: {
        layout: "centered",
    },
    tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Mock chat participant data
const mockChatParticipant = {
    __typename: "ChatParticipant" as const,
    id: DUMMY_ID,
    chat: {
        __typename: "Chat" as const,
        id: "chat-1",
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
                bio: "Software developer passionate about AI and automation",
            },
        ],
    },
    createdAt: new Date("2024-01-15T10:30:00Z").toISOString(),
    updatedAt: new Date("2024-01-15T10:30:00Z").toISOString(),
};

const mockBotParticipant = {
    __typename: "ChatParticipant" as const,
    id: DUMMY_ID,
    chat: {
        __typename: "Chat" as const,
        id: "chat-1",
    },
    user: {
        __typename: "User" as const,
        id: "bot-1",
        name: "AI Assistant",
        handle: "ai-assistant",
        profileImage: null,
        isBot: true,
        translations: [
            {
                __typename: "UserTranslation" as const,
                id: DUMMY_ID,
                language: "en",
                bio: "Your helpful AI assistant for managing projects and workflows",
            },
        ],
    },
    createdAt: new Date("2024-01-10T08:00:00Z").toISOString(),
    updatedAt: new Date("2024-01-10T08:00:00Z").toISOString(),
};

const mockOwnerParticipant = {
    __typename: "ChatParticipant" as const,
    id: DUMMY_ID,
    chat: {
        __typename: "Chat" as const,
        id: "chat-1",
    },
    user: {
        __typename: "User" as const,
        id: "owner-1",
        name: "Sarah Wilson",
        handle: "sarahw",
        profileImage: "/api/v1/file/avatar-sarah.jpg",
        isBot: false,
        translations: [
            {
                __typename: "UserTranslation" as const,
                id: DUMMY_ID,
                language: "en",
                bio: "Project manager and team lead",
            },
        ],
    },
    createdAt: new Date("2024-01-01T00:00:00Z").toISOString(),
    updatedAt: new Date("2024-01-01T00:00:00Z").toISOString(),
};

export const Default: Story = {
    args: {
        data: mockChatParticipant,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: false,
        isSelected: false,
    },
};

export const BotParticipant: Story = {
    args: {
        data: mockBotParticipant,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: false,
        isSelected: false,
    },
};

export const OwnerWithAvatar: Story = {
    args: {
        data: mockOwnerParticipant,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: false,
        isSelected: false,
    },
};

export const Selected: Story = {
    args: {
        data: mockChatParticipant,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: true,
        isSelected: true,
    },
};

export const Selectable: Story = {
    args: {
        data: mockChatParticipant,
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: true,
        isSelected: false,
    },
};

export const WithoutJoinDate: Story = {
    args: {
        data: {
            ...mockChatParticipant,
            createdAt: null,
        },
        handleToggleSelect: () => console.log("handleToggleSelect"),
        isSelecting: false,
        isSelected: false,
    },
};

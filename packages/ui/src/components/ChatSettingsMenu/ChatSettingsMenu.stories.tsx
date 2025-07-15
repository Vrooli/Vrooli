// AI_CHECK: TYPE_SAFETY=fixed-10-translation-key-types | LAST: 2025-07-01
// AI_CHECK: TYPE_SAFETY=fixed-ai-service-types | LAST: 2025-06-28
import type { Meta, StoryObj } from "@storybook/react";
import { DUMMY_ID, type AIServicesInfo, LlmServiceId, OpenAIModel, AnthropicModel, MistralModel, type TranslationKeyModel } from "@vrooli/shared";
import React, { useState, useEffect } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import { ChatSettingsMenu, type IntegrationSettings, type ModelConfig, type ShareSettings } from "./ChatSettingsMenu.js";
import { customWrapperDecorator } from "../../__test/helpers/storybookDecorators.tsx";

// Mock available models
const mockAvailableModels = [
    {
        name: "GPT-4",
        description: "Advanced language model with improved reasoning",
        value: "gpt-4",
        tier: "premium" as const,
    },
    {
        name: "GPT-3.5 Turbo", 
        description: "Fast and efficient language model",
        value: "gpt-3.5-turbo",
        tier: "standard" as const,
    },
    {
        name: "Claude 3.5 Sonnet",
        description: "Anthropic's advanced reasoning model",
        value: "claude-3.5-sonnet",
        tier: "premium" as const,
    },
];

// Mock AI service configuration
const mockAIServiceConfig: AIServicesInfo = {
    defaultService: LlmServiceId.OpenAI,
    services: {
        OpenAI: {
            enabled: true,
            name: "OpenAI",
            defaultModel: OpenAIModel.Gpt4,
            fallbackMaxTokens: 4096,
            models: {
                [OpenAIModel.Gpt4]: {
                    enabled: true,
                    name: "GPT_4_Name" as TranslationKeyModel,
                    descriptionShort: "GPT_4_Description" as TranslationKeyModel,
                    inputCost: 3000,
                    outputCost: 6000,
                    contextWindow: 8192,
                    maxOutputTokens: 8192,
                    features: {},
                    supportsReasoning: false,
                },
                [OpenAIModel.Gpt4o_Mini]: {
                    enabled: true,
                    name: "GPT_4o_Mini_Name" as TranslationKeyModel,
                    descriptionShort: "GPT_4o_Mini_Description" as TranslationKeyModel,
                    inputCost: 150,
                    outputCost: 200,
                    contextWindow: 16384,
                    maxOutputTokens: 4096,
                    features: {},
                    supportsReasoning: false,
                },
            },
            displayOrder: [OpenAIModel.Gpt4, OpenAIModel.Gpt4o_Mini],
        },
        Anthropic: {
            enabled: true,
            name: "Anthropic",
            defaultModel: AnthropicModel.Claude3_5_Sonnet,
            fallbackMaxTokens: 4096,
            models: {
                [AnthropicModel.Claude3_5_Sonnet]: {
                    enabled: true,
                    name: "Claude_3_5_Sonnet_Name" as TranslationKeyModel,
                    descriptionShort: "Claude_3_5_Sonnet_Description" as TranslationKeyModel,
                    inputCost: 300,
                    outputCost: 1500,
                    contextWindow: 200000,
                    maxOutputTokens: 8192,
                    features: {},
                    supportsReasoning: false,
                },
            },
            displayOrder: [AnthropicModel.Claude3_5_Sonnet],
        },
        Mistral: {
            enabled: false,
            name: "Mistral", 
            defaultModel: MistralModel.MistralNemo,
            fallbackMaxTokens: 4096,
            models: {},
            displayOrder: [],
        },
    },
    fallbacks: {},
};

// Helper component to set up localStorage mocks
const WithAIMocks: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    useEffect(() => {
        // Store the mock AI service config in localStorage
        const mockServiceData = {
            config: mockAIServiceConfig,
            timestamp: Date.now(),
        };
        localStorage.setItem("AI_SERVICE_CACHE", JSON.stringify(mockServiceData));
        
        // Clean up on unmount
        return () => {
            localStorage.removeItem("AI_SERVICE_CACHE");
        };
    }, []);
    
    return <>{children}</>;
};

const meta: Meta<typeof ChatSettingsMenu> = {
    title: "Components/Chat/ChatSettingsMenu",
    component: ChatSettingsMenu,
    parameters: {
        layout: "fullscreen",
        docs: {
            description: {
                component: "A comprehensive chat settings dialog with tabs for model configuration, integrations, participants, details, and sharing. In production, AI models are fetched from the API, but in Storybook we provide mock model configs.",
            },
        },
    },
    tags: ["autodocs"],
    decorators: [customWrapperDecorator(WithAIMocks)],
};

export default meta;
type Story = StoryObj<typeof meta>;


// Mock chat object
const mockChat = {
    __typename: "Chat" as const,
    id: DUMMY_ID,
    publicId: "chat-123",
    translations: [
        {
            __typename: "ChatTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            name: "AI Assistant Chat",
            description: "A conversation with an AI assistant",
        },
    ],
};

// Mock participants
const mockParticipants = [
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
            id: "user-2",
            name: "AI Assistant",
            handle: "ai-assistant",
        },
    },
];

// Mock chat invites
const mockInvites = [
    {
        __typename: "ChatInvite" as const,
        id: "invite-1",
        user: {
            __typename: "User" as const,
            id: "user-3",
            name: "Jane Smith",
            handle: "janesmith",
        },
        message: "Would you like to join this chat?",
    },
];

// Mock model configs
const defaultModelConfigs: ModelConfig[] = [
    {
        model: {
            name: "GPT_4_Name" as TranslationKeyModel,
            description: "GPT_4_Description" as TranslationKeyModel,
            value: OpenAIModel.Gpt4,
            tier: "premium" as const,
        },
        toolSettings: {
            siteActions: true,
            webSearch: true,
            fileRetrieval: true,
        },
        requireConfirmation: false,
    },
];

// Mock integration settings
const defaultIntegrationSettings: IntegrationSettings = {
    projects: [
        {
            __typename: "ProjectVersion" as const,
            id: "project-1",
            translations: [
                {
                    __typename: "ProjectVersionTranslation" as const,
                    id: DUMMY_ID,
                    language: "en",
                    name: "My Project",
                },
            ],
        },
    ],
    externalApps: [
        {
            id: "slack-1",
            name: "Slack",
            description: "Team communication and notifications",
            isConnected: true,
            authType: "oauth",
        },
        {
            id: "github-1",
            name: "GitHub",
            description: "Code repository integration",
            isConnected: false,
            authType: "oauth",
        },
    ],
};

// Mock share settings
const defaultShareSettings: ShareSettings = {
    enabled: false,
};

const enabledShareSettings: ShareSettings = {
    enabled: true,
    link: "https://vrooli.com/chat/chat-123",
};

// Mock handlers
const mockHandlers = {
    onClose: () => console.log("onClose"),
    onModelConfigChange: (configs: ModelConfig[]) => console.log("onModelConfigChange", configs),
    onAddParticipant: (id: string) => console.log("onAddParticipant", id),
    onRemoveParticipant: (id: string) => console.log("onRemoveParticipant", id),
    onCancelInvite: (inviteId: string) => console.log("onCancelInvite", inviteId),
    onUpdateDetails: (data: { name: string; description?: string }) => console.log("onUpdateDetails", data),
    onToggleShare: (enabled: boolean) => console.log("onToggleShare", enabled),
    onIntegrationSettingsChange: (settings: IntegrationSettings) => console.log("onIntegrationSettingsChange", settings),
    onDeleteChat: () => console.log("onDeleteChat"),
};

// Interactive story with open/close functionality
export const Default: Story = {
    render: () => {
        const [open, setOpen] = useState(false);
        const [shareSettings, setShareSettings] = useState(defaultShareSettings);
        const [modelConfigs, setModelConfigs] = useState(defaultModelConfigs);
        const [integrationSettings, setIntegrationSettings] = useState(defaultIntegrationSettings);
        
        return (
            <Box sx={{ p: 3, height: "100vh", bgcolor: "background.default" }}>
                <Button variant="contained" onClick={() => setOpen(true)}>
                    Open Chat Settings
                </Button>
                <ChatSettingsMenu
                    open={open}
                    onClose={() => setOpen(false)}
                    chat={mockChat}
                    participants={mockParticipants}
                    invites={[]}
                    integrationSettings={integrationSettings}
                    modelConfigs={modelConfigs}
                    shareSettings={shareSettings}
                    onModelConfigChange={setModelConfigs}
                    onToggleShare={(enabled) => setShareSettings(prev => ({ ...prev, enabled }))}
                    onIntegrationSettingsChange={setIntegrationSettings}
                    {...mockHandlers}
                />
            </Box>
        );
    },
};

export const WithInvites: Story = {
    render: () => {
        const [open, setOpen] = useState(false);
        
        return (
            <Box sx={{ p: 3, height: "100vh", bgcolor: "background.default" }}>
                <Button variant="contained" onClick={() => setOpen(true)}>
                    Open Settings (With Invites)
                </Button>
                <ChatSettingsMenu
                    open={open}
                    onClose={() => setOpen(false)}
                    chat={mockChat}
                    participants={mockParticipants}
                    invites={mockInvites}
                    integrationSettings={defaultIntegrationSettings}
                    modelConfigs={defaultModelConfigs}
                    shareSettings={defaultShareSettings}
                    {...mockHandlers}
                />
            </Box>
        );
    },
};

export const ShareEnabled: Story = {
    render: () => {
        const [open, setOpen] = useState(false);
        const [shareSettings, setShareSettings] = useState(enabledShareSettings);
        const [modelConfigs, setModelConfigs] = useState(defaultModelConfigs);
        const [integrationSettings, setIntegrationSettings] = useState(defaultIntegrationSettings);
        
        return (
            <Box sx={{ p: 3, height: "100vh", bgcolor: "background.default" }}>
                <Button variant="contained" onClick={() => setOpen(true)}>
                    Open Settings (Share Enabled)
                </Button>
                <ChatSettingsMenu
                    open={open}
                    onClose={() => setOpen(false)}
                    chat={mockChat}
                    participants={mockParticipants}
                    invites={[]}
                    integrationSettings={integrationSettings}
                    modelConfigs={modelConfigs}
                    shareSettings={shareSettings}
                    onModelConfigChange={setModelConfigs}
                    onToggleShare={(enabled) => setShareSettings(prev => ({ ...prev, enabled }))}
                    onIntegrationSettingsChange={setIntegrationSettings}
                    {...mockHandlers}
                />
            </Box>
        );
    },
};

export const NoModels: Story = {
    render: () => {
        const [open, setOpen] = useState(false);
        
        return (
            <Box sx={{ p: 3, height: "100vh", bgcolor: "background.default" }}>
                <Button variant="contained" onClick={() => setOpen(true)}>
                    Open Settings (No Models)
                </Button>
                <ChatSettingsMenu
                    open={open}
                    onClose={() => setOpen(false)}
                    chat={mockChat}
                    participants={mockParticipants}
                    invites={[]}
                    integrationSettings={defaultIntegrationSettings}
                    modelConfigs={[]}
                    shareSettings={defaultShareSettings}
                    {...mockHandlers}
                />
            </Box>
        );
    },
};

export const RequireConfirmation: Story = {
    render: () => {
        const [open, setOpen] = useState(false);
        
        return (
            <Box sx={{ p: 3, height: "100vh", bgcolor: "background.default" }}>
                <Button variant="contained" onClick={() => setOpen(true)}>
                    Open Settings (Require Confirmation)
                </Button>
                <ChatSettingsMenu
                    open={open}
                    onClose={() => setOpen(false)}
                    chat={mockChat}
                    participants={mockParticipants}
                    invites={[]}
                    integrationSettings={defaultIntegrationSettings}
                    modelConfigs={[
                        {
                            model: {
                                name: "GPT_4_Name" as TranslationKeyModel,
                                description: "GPT_4_Description" as TranslationKeyModel,
                                value: OpenAIModel.Gpt4,
                                tier: "premium" as const,
                            },
                            toolSettings: {
                                siteActions: false,
                                webSearch: false,
                                fileRetrieval: true,
                            },
                            requireConfirmation: true,
                        },
                    ]}
                    shareSettings={defaultShareSettings}
                    {...mockHandlers}
                />
            </Box>
        );
    },
};

export const NoIntegrations: Story = {
    render: () => {
        const [open, setOpen] = useState(false);
        
        return (
            <Box sx={{ p: 3, height: "100vh", bgcolor: "background.default" }}>
                <Button variant="contained" onClick={() => setOpen(true)}>
                    Open Settings (No Integrations)
                </Button>
                <ChatSettingsMenu
                    open={open}
                    onClose={() => setOpen(false)}
                    chat={mockChat}
                    participants={mockParticipants}
                    invites={[]}
                    integrationSettings={{ projects: [] }}
                    modelConfigs={defaultModelConfigs}
                    shareSettings={defaultShareSettings}
                    {...mockHandlers}
                />
            </Box>
        );
    },
};

export const WithDeleteOption: Story = {
    render: () => {
        const [open, setOpen] = useState(false);
        
        return (
            <Box sx={{ p: 3, height: "100vh", bgcolor: "background.default" }}>
                <Button variant="contained" onClick={() => setOpen(true)}>
                    Open Settings (With Delete)
                </Button>
                <ChatSettingsMenu
                    open={open}
                    onClose={() => setOpen(false)}
                    chat={mockChat}
                    participants={mockParticipants}
                    invites={[]}
                    integrationSettings={defaultIntegrationSettings}
                    modelConfigs={defaultModelConfigs}
                    shareSettings={defaultShareSettings}
                    {...mockHandlers}
                    onDeleteChat={() => console.log("onDeleteChat")}
                />
            </Box>
        );
    },
};

export const SingleParticipant: Story = {
    render: () => {
        const [open, setOpen] = useState(false);
        
        return (
            <Box sx={{ p: 3, height: "100vh", bgcolor: "background.default" }}>
                <Button variant="contained" onClick={() => setOpen(true)}>
                    Open Settings (Single Participant)
                </Button>
                <ChatSettingsMenu
                    open={open}
                    onClose={() => setOpen(false)}
                    chat={mockChat}
                    participants={[mockParticipants[0]]}
                    invites={[]}
                    integrationSettings={defaultIntegrationSettings}
                    modelConfigs={defaultModelConfigs}
                    shareSettings={defaultShareSettings}
                    {...mockHandlers}
                />
            </Box>
        );
    },
};

// Story that starts with the dialog open
export const AlwaysOpen: Story = {
    render: () => {
        return (
            <Box sx={{ p: 3, height: "100vh", bgcolor: "background.default" }}>
                <ChatSettingsMenu
                    open={true}
                    onClose={() => console.log("Close clicked - dialog stays open in this story")}
                    chat={mockChat}
                    participants={mockParticipants}
                    invites={mockInvites}
                    integrationSettings={defaultIntegrationSettings}
                    modelConfigs={defaultModelConfigs}
                    shareSettings={defaultShareSettings}
                    {...mockHandlers}
                />
            </Box>
        );
    },
};

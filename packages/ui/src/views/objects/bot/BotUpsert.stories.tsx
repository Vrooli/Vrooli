import type { Meta, StoryObj } from "@storybook/react";
import { action } from "@storybook/addon-actions";
import { DUMMY_ID, type User, endpointsUser, endpointsTask, generatePK } from "@vrooli/shared";
import { SessionContext } from "../../../contexts/session.js";
import { BotUpsert } from "./BotUpsert.js";
import { HttpResponse, http } from "msw";
import { API_URL, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { getMockUrl } from "../../../__test/helpers/storybookMocking.js";
import React from "react";

// Mock AI configuration for localStorage
const mockAIServiceConfig = {
    config: {
        defaultService: "openai",
        services: {
            openai: {
                enabled: true,
                defaultModel: "gpt-4",
                models: {
                    "gpt-4": {
                        enabled: true,
                        name: { "en": "GPT-4" },
                        descriptionShort: { "en": "Most capable GPT-4 model" },
                    },
                    "gpt-3.5-turbo": {
                        enabled: true,
                        name: { "en": "GPT-3.5 Turbo" },
                        descriptionShort: { "en": "Fast and efficient model" },
                    },
                },
            },
            anthropic: {
                enabled: true,
                defaultModel: "claude-3",
                models: {
                    "claude-3": {
                        enabled: true,
                        name: { "en": "Claude 3" },
                        descriptionShort: { "en": "Advanced reasoning model" },
                    },
                },
            },
        },
    },
    timestamp: Date.now(),
};

// Set up localStorage mocks before stories load
if (typeof window !== "undefined") {
    window.localStorage.setItem("AI_SERVICE_CACHE", JSON.stringify(mockAIServiceConfig));
}

// Use the standard session from storybookConsts but without credits to disable autofill
const mockSession = {
    ...signedInPremiumWithCreditsSession,
    users: signedInPremiumWithCreditsSession.users.map(user => ({
        ...user,
        credits: "0", // No credits to disable autofill
    })),
};

// Mock bot data for stories
const mockExistingBot: Partial<User> = {
    id: "bot-123",
    name: "Assistant Bot",
    handle: "assistantbot",
    isBot: true,
    isBotDepictingPerson: false,
    isPrivate: false,
    botSettings: {
        __version: "1.0",
        model: "gpt-4",
        persona: "You are a helpful AI assistant",
        creativity: 0.7,
        verbosity: 0.5,
    },
    translations: [{
        id: "trans-bot-123",
        language: "en",
        bio: "A helpful AI assistant bot designed to provide support and answer questions.",
        occupation: "AI Assistant",
        persona: "Friendly and knowledgeable helper",
        tone: "Professional yet approachable",
        keyPhrases: "help, assist, support, answer questions",
        domainKnowledge: "General knowledge, customer support, technical assistance",
        bias: "Helpful and solution-oriented",
        startingMessage: "Hello! I'm here to help you with any questions you might have.",
        __typename: "UserTranslation" as const,
    }],
    __typename: "User" as const,
};

// Add missing endpoint if it doesn't exist
if (!endpointsTask.startLlmTask) {
    (endpointsTask as any).startLlmTask = {
        endpoint: "/task/start/llm",
        method: "POST" as const,
    };
}

// Default MSW handlers for all stories
const defaultHandlers = [
    // Mock the LLM task endpoint to prevent autofill errors
    http.post(`${API_URL}/task/start/llm`, () => {
        return HttpResponse.json({ success: true });
    }),
    // Mock the swarm task endpoint
    http.post(getMockUrl(endpointsTask.startSwarmTask), () => {
        return HttpResponse.json({ success: true });
    }),
    // Mock the bot create endpoint
    http.post(getMockUrl(endpointsUser.botCreateOne), () => {
        return HttpResponse.json({
            data: {
                ...mockExistingBot,
                id: generatePK().toString(),
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
            },
        });
    }),
    // Mock the bot update endpoint
    http.put(getMockUrl(endpointsUser.botUpdateOne), () => {
        return HttpResponse.json({
            data: {
                ...mockExistingBot,
                updatedAt: new Date().toISOString(),
            },
        });
    }),
    // Mock the findOne endpoint
    http.get(getMockUrl(endpointsUser.findOne), () => {
        return HttpResponse.json({
            data: mockExistingBot,
        });
    }),
    // Mock any other potential endpoints that might be called
    http.get(`${API_URL}/*`, () => {
        return HttpResponse.json({ data: {} });
    }),
    http.post(`${API_URL}/*`, () => {
        return HttpResponse.json({ success: true });
    }),
];

const meta: Meta<typeof BotUpsert> = {
    title: "Views/Objects/Bot/BotUpsert",
    component: BotUpsert,
    parameters: {
        layout: "fullscreen",
        backgrounds: { disable: true },
        docs: {
            story: {
                inline: false,
                iframeHeight: 800,
            },
        },
        msw: {
            handlers: defaultHandlers,
        },
    },
    tags: ["autodocs"],
    decorators: [
        (Story) => (
            <SessionContext.Provider value={mockSession}>
                <Story />
            </SessionContext.Provider>
        ),
    ],
    argTypes: {
        display: {
            control: { type: "select" },
            options: ["Dialog", "Page"],
            description: "How the component is displayed",
        },
        isCreate: {
            control: { type: "boolean" },
            description: "Whether this is creating a new bot or editing existing",
        },
        isOpen: {
            control: { type: "boolean" },
            description: "Whether the dialog is open (only relevant for Dialog display)",
        },
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Create new bot in dialog
export const CreateBotDialog: Story = {
    args: {
        display: "Dialog",
        isCreate: true,
        isOpen: true,
        onClose: action("onClose"),
        onCompleted: action("onCompleted"),
        onDeleted: action("onDeleted"),
    },
};

// Create new bot in full page
export const CreateBotPage: Story = {
    args: {
        display: "Page",
        isCreate: true,
        onClose: action("onClose"),
        onCompleted: action("onCompleted"),
        onDeleted: action("onDeleted"),
    },
};

// Edit existing bot in dialog
export const EditBotDialog: Story = {
    args: {
        display: "Dialog",
        isCreate: false,
        isOpen: true,
        overrideObject: mockExistingBot,
        onClose: action("onClose"),
        onCompleted: action("onCompleted"),
        onDeleted: action("onDeleted"),
    },
};

// Edit existing bot in full page
export const EditBotPage: Story = {
    args: {
        display: "Page",
        isCreate: false,
        overrideObject: mockExistingBot,
        onClose: action("onClose"),
        onCompleted: action("onCompleted"),
        onDeleted: action("onDeleted"),
    },
};

// Bot depicting a person
export const BotDepictingPerson: Story = {
    args: {
        display: "Dialog",
        isCreate: false,
        isOpen: true,
        overrideObject: {
            ...mockExistingBot,
            name: "Celebrity Bot",
            handle: "celebritybot",
            isBotDepictingPerson: true,
            translations: [{
                ...mockExistingBot.translations![0],
                bio: "A bot that represents a real person's personality and knowledge.",
                persona: "Mimics the speaking style and knowledge of a specific individual",
            }],
        },
        onClose: action("onClose"),
        onCompleted: action("onCompleted"),
        onDeleted: action("onDeleted"),
    },
};

// Private bot
export const PrivateBot: Story = {
    args: {
        display: "Dialog",
        isCreate: false,
        isOpen: true,
        overrideObject: {
            ...mockExistingBot,
            name: "Private Assistant",
            handle: "privatebot",
            isPrivate: true,
            translations: [{
                ...mockExistingBot.translations![0],
                bio: "A private bot for personal use only.",
            }],
        },
        onClose: action("onClose"),
        onCompleted: action("onCompleted"),
        onDeleted: action("onDeleted"),
    },
};

// Bot with complex personality settings
export const ComplexPersonalityBot: Story = {
    args: {
        display: "Dialog",
        isCreate: false,
        isOpen: true,
        overrideObject: {
            ...mockExistingBot,
            name: "Expert Consultant",
            handle: "expertbot",
            botSettings: {
                __version: "1.0",
                model: "claude-3",
                persona: {
                    occupation: "Senior Technology Consultant",
                    persona: "Highly analytical, detail-oriented professional with deep technical knowledge",
                    tone: "Formal, authoritative, but patient in explanations",
                    keyPhrases: "best practices, industry standards, technical excellence, optimization",
                    domainKnowledge: "Software architecture, cloud computing, DevOps, security protocols, performance optimization",
                    bias: "Prefers proven, well-established solutions over experimental approaches",
                    startingMessage: "Good day. I'm here to provide expert consultation on your technical challenges. How may I assist you today?",
                    creativity: 0.3,
                    verbosity: 0.8,
                },
            },
            translations: [{
                ...mockExistingBot.translations![0],
                bio: "A specialized consulting bot with extensive domain expertise.",
            }],
        },
        onClose: action("onClose"),
        onCompleted: action("onCompleted"),
        onDeleted: action("onDeleted"),
    },
};

// Minimal bot configuration
export const MinimalBot: Story = {
    args: {
        display: "Dialog",
        isCreate: true,
        isOpen: true,
        overrideObject: {
            id: DUMMY_ID,
            name: "",
            isBot: true,
            isBotDepictingPerson: false,
            isPrivate: false,
            botSettings: {
                __version: "1.0",
                persona: "You are a helpful assistant",
                creativity: 0.5,
                verbosity: 0.5,
            },
            translations: [{
                id: DUMMY_ID,
                language: "en",
                bio: "",
                __typename: "UserTranslation" as const,
            }],
            __typename: "User" as const,
        },
        onClose: action("onClose"),
        onCompleted: action("onCompleted"),
        onDeleted: action("onDeleted"),
    },
};

// Multi-language bot
export const MultilingualBot: Story = {
    args: {
        display: "Dialog",
        isCreate: false,
        isOpen: true,
        overrideObject: {
            ...mockExistingBot,
            name: "Global Assistant",
            handle: "globalbot",
            translations: [
                {
                    id: "trans-en",
                    language: "en",
                    bio: "A multilingual assistant that can help in various languages.",
                    occupation: "International Assistant",
                    persona: "Culturally aware and adaptable",
                    tone: "Warm and inclusive",
                    keyPhrases: "global perspective, cultural sensitivity, multilingual support",
                    domainKnowledge: "International business, cultural norms, language learning",
                    bias: "Respectful of cultural differences",
                    startingMessage: "Hello! I can assist you in multiple languages. How can I help?",
                    __typename: "UserTranslation" as const,
                },
                {
                    id: "trans-es",
                    language: "es",
                    bio: "Un asistente multilingüe que puede ayudar en varios idiomas.",
                    occupation: "Asistente Internacional",
                    persona: "Consciente culturalmente y adaptable",
                    tone: "Cálido e inclusivo",
                    keyPhrases: "perspectiva global, sensibilidad cultural, soporte multilingüe",
                    domainKnowledge: "Negocios internacionales, normas culturales, aprendizaje de idiomas",
                    bias: "Respetuoso de las diferencias culturales",
                    startingMessage: "¡Hola! Puedo ayudarte en múltiples idiomas. ¿Cómo puedo ayudar?",
                    __typename: "UserTranslation" as const,
                },
            ],
        },
        onClose: action("onClose"),
        onCompleted: action("onCompleted"),
        onDeleted: action("onDeleted"),
    },
};

// Creative bot with high creativity settings
export const CreativeBot: Story = {
    args: {
        display: "Dialog",
        isCreate: false,
        isOpen: true,
        overrideObject: {
            ...mockExistingBot,
            name: "Creative Writer",
            handle: "creativebot",
            botSettings: {
                __version: "1.0",
                model: "gpt-4",
                creativity: 0.9,
                verbosity: 0.7,
            },
            translations: [{
                ...mockExistingBot.translations![0],
                bio: "A highly creative writing assistant that helps with storytelling and creative content.",
                occupation: "Creative Writing Assistant",
                persona: "Imaginative, inspiring, and encouraging creative expression",
                tone: "Enthusiastic and supportive",
                keyPhrases: "creativity, imagination, storytelling, artistic expression",
                domainKnowledge: "Creative writing, literature, poetry, screenwriting, character development",
                bias: "Encourages originality and creative risk-taking",
                startingMessage: "Welcome, fellow creator! I'm here to help bring your creative visions to life. What story shall we tell today?",
            }],
        },
        onClose: action("onClose"),
        onCompleted: action("onCompleted"),
        onDeleted: action("onDeleted"),
    },
};

// Analytical bot with low creativity settings
export const AnalyticalBot: Story = {
    args: {
        display: "Dialog",
        isCreate: false,
        isOpen: true,
        overrideObject: {
            ...mockExistingBot,
            name: "Data Analyst",
            handle: "analystbot",
            botSettings: {
                __version: "1.0",
                model: "claude-3",
                creativity: 0.2,
                verbosity: 0.6,
            },
            translations: [{
                ...mockExistingBot.translations![0],
                bio: "A precise analytical assistant focused on data analysis and logical reasoning.",
                occupation: "Senior Data Analyst",
                persona: "Methodical, precise, and data-driven",
                tone: "Professional and objective",
                keyPhrases: "data analysis, statistical significance, evidence-based, methodology",
                domainKnowledge: "Statistics, data science, business intelligence, research methodology",
                bias: "Prefers data-driven decisions over intuition",
                startingMessage: "I'm ready to help you analyze data and derive meaningful insights. What dataset or problem would you like to examine?",
            }],
        },
        onClose: action("onClose"),
        onCompleted: action("onCompleted"),
        onDeleted: action("onDeleted"),
    },
};

// Bot with custom JSON personality
export const CustomJSONPersonalityBot: Story = {
    args: {
        display: "Dialog",
        isCreate: false,
        isOpen: true,
        overrideObject: {
            ...mockExistingBot,
            name: "Custom Bot",
            handle: "custombot",
            botSettings: {
                __version: "1.0",
                model: "gpt-4",
                persona: {
                    role: "Game Master",
                    setting: "Fantasy Medieval World",
                    rules: ["Always describe scenes vividly", "Track player inventory", "Roll dice fairly"],
                    npcVoices: {
                        innkeeper: "Gruff but friendly",
                        wizard: "Mysterious and cryptic",
                        merchant: "Eager to make a deal",
                    },
                    specialAbilities: ["world building", "combat narration", "puzzle creation"],
                    responseFormat: "narrative",
                    creativity: 0.8,
                    verbosity: 0.9,
                },
            },
            translations: [{
                ...mockExistingBot.translations![0],
                bio: "A custom game master bot with unique personality configuration.",
            }],
        },
        onClose: action("onClose"),
        onCompleted: action("onCompleted"),
        onDeleted: action("onDeleted"),
    },
};

// Closed dialog state
export const ClosedDialog: Story = {
    args: {
        display: "Dialog",
        isCreate: true,
        isOpen: false,
        onClose: action("onClose"),
        onCompleted: action("onCompleted"),
        onDeleted: action("onDeleted"),
    },
};

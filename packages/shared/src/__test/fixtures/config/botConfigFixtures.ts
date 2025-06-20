import { type BotConfigObject, DEFAULT_PERSONA } from "../../../shape/configs/bot.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures, mergeWithBaseDefaults } from "./baseConfigFixtures.js";

// Constants to avoid magic numbers
const DEFAULT_MAX_TOKENS = 2048;

/**
 * Bot configuration fixtures for testing bot personality and settings
 */
export const botConfigFixtures: ConfigTestFixtures<BotConfigObject> = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
    },

    complete: {
        __version: LATEST_CONFIG_VERSION,
        model: "gpt-4",
        maxTokens: 2048,
        persona: {
            occupation: "AI Assistant",
            persona: "Helpful, knowledgeable, and professional",
            tone: "friendly yet formal",
            startingMessage: "Hello! How can I assist you today?",
            keyPhrases: "efficiency, accuracy, helpfulness",
            domainKnowledge: "general knowledge, programming, science",
            bias: "objective and factual",
            creativity: 0.5,
            verbosity: 0.5,
        },
        resources: [{
            link: "https://example.com/bot-docs",
            usedFor: "OfficialWebsite",
            translations: [{
                language: "en",
                name: "Bot Documentation",
                description: "Official documentation for this bot",
            }],
        }],
    },

    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
        persona: DEFAULT_PERSONA,
    },

    invalid: {
        missingVersion: {
            // Missing __version
            model: "gpt-4",
            maxTokens: 2048,
        },
        invalidVersion: {
            __version: "0.1", // Invalid version
            model: "gpt-4",
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            persona: "string instead of object", // Wrong type
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            maxTokens: 0,
            persona: {
                creativity: 2,
                verbosity: 2,
            },
        },
    },

    variants: {
        minimalPersona: {
            __version: LATEST_CONFIG_VERSION,
            persona: {
                occupation: "Developer",
            },
        },

        creativeBotHighTokens: {
            __version: LATEST_CONFIG_VERSION,
            model: "gpt-4",
            maxTokens: 4096,
            persona: {
                ...DEFAULT_PERSONA,
                creativity: 1,
                tone: "imaginative and playful",
                occupation: "Creative Writer",
                keyPhrases: "storytelling, imagination, creativity",
            },
        },

        technicalBotPrecise: {
            __version: LATEST_CONFIG_VERSION,
            model: "claude-3",
            maxTokens: 8192,
            persona: {
                occupation: "Software Engineer",
                persona: "Technical expert focused on accuracy and detail",
                tone: "technical and precise",
                startingMessage: "Ready to assist with technical questions.",
                keyPhrases: "algorithms, architecture, best practices",
                domainKnowledge: "programming, system design, algorithms, data structures",
                bias: "evidence-based and analytical",
                creativity: 0.2,
                verbosity: 0.8,
            },
        },

        customerServiceBot: {
            __version: LATEST_CONFIG_VERSION,
            model: "gpt-3.5-turbo",
            maxTokens: 1024,
            persona: {
                occupation: "Customer Service Representative",
                persona: "Patient, understanding, and solution-oriented",
                tone: "warm and empathetic",
                startingMessage: "Hi! I'm here to help. What can I do for you today?",
                keyPhrases: "help, support, solution, assistance",
                domainKnowledge: "customer service, problem resolution, communication",
                bias: "customer-focused and positive",
                creativity: 0.3,
                verbosity: 0.6,
            },
        },

        educationalBot: {
            __version: LATEST_CONFIG_VERSION,
            persona: {
                occupation: "Teacher",
                persona: "Encouraging educator who breaks down complex topics",
                tone: "educational and encouraging",
                startingMessage: "Let's learn something new today!",
                keyPhrases: "learning, understanding, example, explanation",
                domainKnowledge: "education, teaching methods, learning psychology",
                bias: "educational and growth-oriented",
                creativity: 0.6,
                verbosity: 0.7,
            },
        },

        researchBot: {
            __version: LATEST_CONFIG_VERSION,
            model: "claude-3",
            maxTokens: 16384,
            persona: {
                occupation: "Research Analyst",
                persona: "Thorough researcher who provides comprehensive analysis",
                tone: "analytical and thorough",
                startingMessage: "I'll help you research and analyze information.",
                keyPhrases: "research, analysis, data, evidence, sources",
                domainKnowledge: "research methodology, data analysis, critical thinking",
                bias: "data-driven and skeptical",
                creativity: 0.4,
                verbosity: 0.9,
            },
            resources: [
                {
                    link: "https://scholar.google.com",
                    usedFor: "Researching",
                    translations: [{
                        language: "en",
                        name: "Google Scholar",
                        description: "Academic research database",
                    }],
                },
                {
                    link: "https://arxiv.org",
                    usedFor: "Learning",
                    translations: [{
                        language: "en",
                        name: "arXiv",
                        description: "Open-access research papers",
                    }],
                },
            ],
        },
    },
};

/**
 * Create a bot config with specific persona traits
 */
export function createBotConfigWithPersona(
    occupation: string,
    traits: Partial<BotConfigObject["persona"]> = {},
): BotConfigObject {
    return mergeWithBaseDefaults<BotConfigObject>({
        persona: {
            ...DEFAULT_PERSONA,
            occupation,
            ...traits,
        },
    });
}

/**
 * Create a bot config for a specific model
 */
export function createBotConfigForModel(
    model: string,
    maxTokens = DEFAULT_MAX_TOKENS,
): BotConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        model,
        maxTokens,
        persona: DEFAULT_PERSONA,
    };
}

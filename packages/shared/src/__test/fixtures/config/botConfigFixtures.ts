import { ModelStrategy } from "../../../shape/configs/base.js";
import { type BotConfigObject } from "../../../shape/configs/bot.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures } from "./baseConfigFixtures.js";

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
        modelConfig: {
            strategy: ModelStrategy.FALLBACK,
            preferredModel: "gpt-4",
            offlineOnly: false,
        },
        maxTokens: 2048,
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
    },

    invalid: {
        missingVersion: {
            // Missing __version
            modelConfig: {
                strategy: ModelStrategy.FIXED,
                preferredModel: "gpt-4",
                offlineOnly: false,
            },
            maxTokens: 2048,
        },
        invalidVersion: {
            __version: "0.1", // Invalid version
            modelConfig: {
                strategy: ModelStrategy.FIXED,
                preferredModel: "gpt-4",
                offlineOnly: false,
            },
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            modelConfig: 123, // Wrong type
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            maxTokens: "invalid" as unknown as number, // Intentionally wrong type
        },
    },

    variants: {
        basicBot: {
            __version: LATEST_CONFIG_VERSION,
            modelConfig: {
                strategy: ModelStrategy.COST_OPTIMIZED,
                preferredModel: "gpt-3.5-turbo",
                offlineOnly: false,
            },
        },

        creativeBotHighTokens: {
            __version: LATEST_CONFIG_VERSION,
            modelConfig: {
                strategy: ModelStrategy.QUALITY_FIRST,
                preferredModel: "gpt-4",
                offlineOnly: false,
            },
            maxTokens: 4096,
        },

        technicalBotPrecise: {
            __version: LATEST_CONFIG_VERSION,
            modelConfig: {
                strategy: ModelStrategy.QUALITY_FIRST,
                preferredModel: "claude-3",
                offlineOnly: false,
            },
            maxTokens: 8192,
        },

        researchBot: {
            __version: LATEST_CONFIG_VERSION,
            modelConfig: {
                strategy: ModelStrategy.QUALITY_FIRST,
                preferredModel: "claude-3",
                offlineOnly: false,
            },
            maxTokens: 16384,
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
 * Create a bot config for a specific model
 */
export function createBotConfigForModel(
    model: string,
    maxTokens = DEFAULT_MAX_TOKENS,
): BotConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        modelConfig: {
            strategy: ModelStrategy.FIXED,
            preferredModel: model,
            offlineOnly: false,
        },
        maxTokens,
    };
}

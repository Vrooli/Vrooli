import { OpenAIModel, type AIServiceName, type AIServicesInfo } from "../ai/services";
import { type User, type UserTranslation } from "../api/generated/graphqlTypes";
import { type PassableLogger } from "../consts/commonTypes";
import { DUMMY_ID } from "../id/uuid";
import { type BotShape, type BotTranslationShape } from "../shape/models";
import { type TranslationFunc, type TranslationKeyService } from "../types";
import { toPosDouble } from "../validation/utils/builders/toPosDouble";

export type LlmModel = {
    name: TranslationKeyService,
    description?: TranslationKeyService,
    value: string,
};

export function getAvailableModels(aiServicesInfo: AIServicesInfo | null | undefined): LlmModel[] {
    const models: LlmModel[] = [];
    if (!aiServicesInfo) return models;
    const services = aiServicesInfo.services;
    for (const serviceKey in services) {
        const service = services[serviceKey as AIServiceName];
        if (service.enabled) {
            for (const modelKey of service.displayOrder) {
                const modelInfo = service.models[modelKey];
                if (modelInfo.enabled) {
                    models.push({
                        name: modelInfo.name, // This is a ServiceKey for i18next
                        description: modelInfo.descriptionShort, // Also a ServiceKey
                        value: modelKey,
                    });
                }
            }
        }
    }
    return models;
}

export function getModelName(option: LlmModel | null, t: TranslationFunc) {
    return option ? t(option.name, { ns: "service" }) : "";
}
export function getModelDescription(option: LlmModel, t: TranslationFunc) {
    return option && option.description ? t(option.description, { ns: "service" }) : "";
}

export const DEFAULT_MODEL = OpenAIModel.Gpt4o_Mini;

const DEFAULT_CREATIVITY = 0.5; // Must be between 0 and 1
const DEFAULT_VERBOSITY = 0.5; // Must be between 0 and 1

export type BotSettingsTranslation = {
    bias?: string;
    creativity?: number;
    domainKnowledge?: string;
    keyPhrases?: string;
    occupation?: string;
    persona?: string;
    startingMessage?: string;
    tone?: string;
    verbosity?: number;
}
export type BotSettings = {
    model?: string;
    maxTokens?: number;
    name: string;
    translations?: Record<string, BotSettingsTranslation>
};
export type ToBotSettingsPropBot = {
    name?: string | null | undefined;
    translations?: (Partial<UserTranslation> & { [key: string]: any })[] | null | undefined;
    botSettings?: Record<string, any> | string | null | undefined;
}

/**
 * Converts db bot info to BotSettings type
 * @param bot The bot object to convert
 * @param logger The logger to use for logging errors
 */
export function toBotSettings(
    bot: ToBotSettingsPropBot | null | undefined,
    logger: PassableLogger,
): BotSettings {
    if (!bot || typeof bot !== "object" || Array.isArray(bot)) {
        logger.error("Invalid data passed into 'toBotSettings'", { trace: "0408", bot });
        return { name: "" }; // Default return for invalid input
    }

    let result: BotSettings = { name: bot.name || "" };
    if (typeof bot.botSettings !== "string") return result;
    try {
        const botSettings = JSON.parse(bot.botSettings);
        if (typeof botSettings !== "object" || botSettings === null) return result;

        // Merge botSettings at the top level
        result = { ...botSettings, ...result };

        // If translations are missing in botSettings but present at the top level
        if (!botSettings.translations && bot.translations && Array.isArray(bot.translations)) {
            const translations: Record<string, BotSettingsTranslation> = {};
            bot.translations.forEach((translation) => {
                if (translation.language && typeof translation.language === "string") {
                    // Exclude __typename, id, and bio from the translation details
                    const { __typename, id, language, bio, ...translationDetails } = translation;
                    translations[translation.language] = translationDetails as BotSettingsTranslation;

                    // If creativity and verbosity are present, make sure they are numbers
                    if (typeof translations[translation.language]!.creativity === "string") translations[translation.language]!.creativity = toPosDouble(translations[translation.language]!.creativity as unknown as string);
                    if (typeof translations[translation.language]!.verbosity === "string") translations[translation.language]!.verbosity = toPosDouble(translations[translation.language]!.verbosity as unknown as string);
                }
            });
            if (Object.keys(translations).length > 0) {
                result.translations = translations;
            }
        }
    }
    // eslint-disable-next-line no-empty
    catch { }

    // If creativity and verbosity are present, make sure they are numbers
    if (typeof (result as { creativity?: unknown }).creativity === "string") (result as { creativity?: unknown }).creativity = toPosDouble((result as { creativity?: unknown }).creativity as string);
    if (typeof (result as { verbosity?: unknown }).verbosity === "string") (result as { verbosity?: unknown }).verbosity = toPosDouble((result as { verbosity?: unknown }).verbosity as string);

    return result;
}

export function findBotData(
    language: string,
    availableModels: LlmModel[],
    existing?: Partial<User> | BotShape | null | undefined,
): {
    creativity?: number | null,
    verbosity?: number | null,
    model: LlmModel["value"],
    translations: BotTranslationShape[],
} {
    const settings = toBotSettings(existing, console);
    const settingsTranslation = settings?.translations?.[language];
    const creativity = settingsTranslation?.creativity ?? (settings as { creativity?: number })?.creativity ?? DEFAULT_CREATIVITY;
    const verbosity = settingsTranslation?.verbosity ?? (settings as { verbosity?: number })?.verbosity ?? DEFAULT_VERBOSITY;

    // Default values so formik is happy
    const defaultTranslation = {
        bio: "",
        bias: settingsTranslation?.bias ?? "",
        domainKnowledge: settingsTranslation?.domainKnowledge ?? "",
        keyPhrases: settingsTranslation?.keyPhrases ?? "",
        occupation: settingsTranslation?.occupation ?? "",
        persona: settingsTranslation?.persona ?? "",
        startingMessage: settingsTranslation?.startingMessage ?? "",
        tone: settingsTranslation?.tone ?? "",
        __typename: "UserTranslation" as const,
        id: DUMMY_ID,
        language,
    };

    // Start by preparing translations from existing user data
    const translations = existing?.translations?.map((translation) => ({
        ...defaultTranslation,
        ...translation,
    })) ?? [];

    // If there are bot settings translations, integrate them.
    // This is done separately from the above in case it includes 
    // translations that don't exist in the existing user data.
    if (settings.translations) {
        Object.entries(settings.translations).forEach(([lang, botTranslation]) => {
            // Find if there's an existing translation for this language
            const existingTranslationIndex = translations.findIndex(t => t.language === lang);
            if (existingTranslationIndex !== -1) {
                // Merge bot settings translations into the existing one
                translations[existingTranslationIndex] = {
                    ...botTranslation, // Bot settings first so that they're overwritten by user translations if conflicts exist
                    ...translations[existingTranslationIndex],
                };
            } else {
                // If no existing translation, add a new one based on bot settings
                translations.push({
                    ...defaultTranslation,
                    ...botTranslation,
                    language: lang, // Ensure the language is set correctly
                });
            }
        });
    }

    // If requested language doesn't exist, add it with default values
    if (!translations.some(t => t.language === language)) {
        translations.push({
            ...defaultTranslation,
            language,
        });
    }

    return {
        creativity: creativity >= 0 && creativity <= 1 ? creativity : DEFAULT_CREATIVITY,
        verbosity: verbosity >= 0 && verbosity <= 1 ? verbosity : DEFAULT_VERBOSITY,
        model: typeof settings.model === "string" && availableModels.some((model) => model.value === settings.model) ? settings.model : DEFAULT_MODEL,
        translations,
    };
}

/**
 * Using the participants and the responding bot ID, parses the stringified bot information 
 * associated with the responding bot.
 * @param participants The participants in the chat
 * @param respondingBotId The ID of the bot that is responding
 * @param logger The logger to use for logging errors
 * @returns The bot settings for the responding bot, or null if the bot is not found
 */
export function parseBotInformation(
    participants: Record<string, { name: string, botSettings: string }>,
    respondingBotId: string,
    logger: { error: (message: string, data?: Record<string, any>) => unknown },
): BotSettings | null {
    const bot = participants[respondingBotId];
    return bot ? toBotSettings(bot, logger) : null;
}

import { User, UserTranslation } from "../api/generated/graphqlTypes";
import { PassableLogger } from "../consts/commonTypes";
import { AnthropicModel, OpenAIModel } from "../consts/llm";
import { DUMMY_ID } from "../id/uuid";
import { BotShape, BotTranslationShape } from "../shape/models/bot";
import { toPosDouble } from "../validation/utils/builders/toPosDouble";

export type LlmModel = {
    name: string,
    description?: string,
    value: string,
};
export const AVAILABLE_MODELS: LlmModel[] = [{
    name: "Claude 3.5 Sonnet",
    description: "Anthropic's fastest model (recommended)",
    value: AnthropicModel.Sonnet3_5,
}, {
    name: "Claude 3 Opus",
    description: "Anthropic's most advanced model",
    value: AnthropicModel.Opus3,
}, {
    name: "GPT-3.5 Turbo",
    description: "OpenAI's fastest model",
    value: OpenAIModel.Gpt3_5Turbo,
}, {
    name: "GPT-4o",
    description: "OpenAI's most advanced model",
    value: OpenAIModel.Gpt4o,
}, {
    name: "GPT-4",
    description: "One of OpenAI's advanced models",
    value: OpenAIModel.Gpt4,
    // }, {
    //     name: "Mistral 7b",
    //     description: "Mistral's fastest model",
    //     value: "open-mistral-7b",
    // }, {
    //     name: "Mistral 8x7b",
    //     description: "Mistral's most advanced model",
    //     value: "open-mixtral-8x7b",
    // }];
}];
export function getModelName(option: LlmModel) {
    return option.name;
}
export function getModelDescription(option: LlmModel) {
    return option.description;
}

export const DEFAULT_MODEL = AnthropicModel.Sonnet3_5;

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
export const toBotSettings = (
    bot: ToBotSettingsPropBot | null | undefined,
    logger: PassableLogger,
): BotSettings => {
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
};

export function findBotData(
    language: string,
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
        model: typeof settings.model === "string" && AVAILABLE_MODELS.some((model) => model.value === settings.model) ? settings.model : OpenAIModel.Gpt3_5Turbo,
        translations,
    };
}

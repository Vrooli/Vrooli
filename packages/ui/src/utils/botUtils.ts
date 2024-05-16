import { AnthropicModel, DUMMY_ID, OpenAIModel, User, toBotSettings } from "@local/shared";
import { BotShape, BotTranslationShape } from "./shape/models/bot";

export type LlmModel = {
    name: string,
    description?: string,
    value: string,
};
export const AVAILABLE_MODELS: LlmModel[] = [{
    name: "Claude 3 Sonnet",
    description: "Anthropic's fastest model (recommended)",
    value: AnthropicModel.Sonnet,
}, {
    name: "Claude 3 Opus",
    description: "Anthropic's most advanced model",
    value: AnthropicModel.Opus,
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

export const findBotData = (
    language: string,
    existing?: Partial<User> | BotShape | null | undefined,
): {
    creativity?: number | null,
    verbosity?: number | null,
    model: LlmModel["value"],
    translations: BotTranslationShape[],
} => {
    const settings = toBotSettings(existing, console);
    const settingsTranslation = settings?.translations?.[language];
    const creativity = settingsTranslation?.creativity ?? (settings as { creativity?: number })?.creativity ?? 0.5;
    const verbosity = settingsTranslation?.verbosity ?? (settings as { verbosity?: number })?.verbosity ?? 0.5;

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
        creativity: creativity >= 0 && creativity <= 1 ? creativity : 0.5,
        verbosity: verbosity >= 0 && verbosity <= 1 ? verbosity : 0.5,
        model: typeof settings.model === "string" && AVAILABLE_MODELS.some((model) => model.value === settings.model) ? settings.model : OpenAIModel.Gpt3_5Turbo,
        translations,
    };
};

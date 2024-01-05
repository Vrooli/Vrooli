import { DUMMY_ID, User } from "@local/shared";
import { BotShape, BotTranslationShape } from "./shape/models/bot";

export type LlmModel = {
    name: string,
    description?: string,
    value: string,
};
export const AVAILABLE_MODELS: LlmModel[] = [{
    name: "GPT-3.5 Turbo",
    description: "OpenAI's fastest model",
    value: "gpt-3.5-turbo",
}, {
    name: "GPT-4",
    description: "OpenAI's most advanced model",
    value: "gpt-4",
}];

export const parseBotSettings = (existing?: Partial<User> | BotShape | null | undefined): Record<string, string | number> => {
    let settings: Record<string, string | number> = {};
    try {
        settings = JSON.parse((existing as Partial<User>)?.botSettings ?? "{}");
    } catch (error) {
        console.error("Failed to parse settings", error);
    }
    return settings;
};

export const findBotData = (
    language: string,
    existing?: Partial<User> | BotShape | null | undefined,
): {
    creativity?: number | null,
    verbosity?: number | null,
    model: LlmModel["value"],
    translations: BotTranslationShape[],
} => {
    const settings = parseBotSettings(existing);

    // Default values so formik is happy
    const defaultTranslation = {
        bio: "",
        bias: settings.bias ?? "",
        domainKnowledge: settings.domainKnowledge ?? "",
        keyPhrases: settings.keyPhrases ?? "",
        occupation: settings.occupation ?? "",
        persona: settings.persona ?? "",
        startMessage: settings.startMessage ?? "",
        tone: settings.tone ?? "",
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

    const creativityNumber = +settings.creativity;
    const verbosityNumber = +settings.verbosity;
    return {
        creativity: creativityNumber >= 0 && creativityNumber <= 1 ? creativityNumber : 0.5,
        verbosity: verbosityNumber >= 0 && verbosityNumber <= 1 ? verbosityNumber : 0.5,
        model: typeof settings.model === "string" && AVAILABLE_MODELS.some((model) => model.value === settings.model) ? settings.model : "gpt-3.5-turbo",
        translations,
    };
};

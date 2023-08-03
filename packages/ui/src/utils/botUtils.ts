import { DUMMY_ID, User } from "@local/shared";
import { BotTranslationShape } from "./shape/models/bot";

export const parseBotSettings = (existing?: User | null | undefined): Record<string, string | number> => {
    let settings: Record<string, string | number> = {};
    try {
        settings = JSON.parse(existing?.botSettings ?? "{}");
    } catch (error) {
        console.error("Failed to parse settings", error);
    }
    return settings;
};

export const findBotData = (
    language: string,
    existing?: User | null | undefined,
): {
    creativity?: number | null,
    verbosity?: number | null,
    translations: BotTranslationShape[],
} => {
    const settings = parseBotSettings(existing);

    const defaultTranslation = {
        bio: settings.bio ?? "",
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

    const translations = existing?.translations?.map((translation) => ({
        ...defaultTranslation,
        ...settings.translations?.[translation.language],
        ...translation,
    })) ?? [defaultTranslation];

    const creativityNumber = +settings.creativity;
    const verbosityNumber = +settings.verbosity;
    return {
        creativity: creativityNumber >= 0 && creativityNumber <= 1 ? creativityNumber : 0.5,
        verbosity: verbosityNumber >= 0 && verbosityNumber <= 1 ? verbosityNumber : 0.5,
        translations,
    };
};

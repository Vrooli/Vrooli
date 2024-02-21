import { UserTranslation } from "../api/generated/graphqlTypes";
import { toPosDouble } from "../validation/utils/builders/toPosDouble";

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
    logger: { error: Function },
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
                if (translation.language && typeof translation.language === 'string') {
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
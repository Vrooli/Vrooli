/**
 * Finds the most preferred translation, or the first one if none of the 
 * preferred languages are found.
 * @param translations The translations to search through
 * @param languages The preferred languages
 * @returns The best translation
 */
export const bestTranslation = <
    Translation extends { language: string }
>(
    translations: Translation[],
    languages: string[],
): Translation | undefined => {
    if (!Array.isArray(translations) || translations.length === 0) return undefined;

    for (const language of (languages || [])) {
        const translation = translations.find(({ language: lang }) => lang === language);
        if (translation) return translation;
    }
    if (translations.length > 0) return translations[0];
    return undefined;
};

import { DEFAULT_LANGUAGE } from "../consts";

/**
 * Retrieves an object's translation for a given language code.
 * @param obj The object to retrieve the translation from.
 * @param languages The languages the user is requesting, in order of preference.
 * @param showAny If true, will default to returning the first language if no value is found
 * @returns The requested translation or an empty object if none is found
 */
export function getTranslation<
    Translation extends { language: string },
>(
    obj: { translations?: Translation[] | null | undefined } | null | undefined,
    languages: readonly string[] | null | undefined,
    showAny = true,
): Partial<Translation> {
    if (!obj || !Array.isArray(obj.translations)) return {};
    // Convert user languages to lowercase for case-insensitive comparison
    const lowerCaseLanguages = Array.isArray(languages) ? languages.map(lang => lang.toLowerCase()) : [DEFAULT_LANGUAGE];
    // Loop through user's preferred languages first
    for (const preferredLanguage of lowerCaseLanguages) {
        const foundTranslation = obj.translations.find(translation => translation.language.toLowerCase() === preferredLanguage);
        if (foundTranslation) {
            return foundTranslation;
        }
    }
    // If showAny is true and there's at least one translation, return the first one
    if (showAny && obj.translations.length > 0) return obj.translations[0] as Partial<Translation>;
    // If no translation matches the user's preferences, return an empty object
    return {};
}
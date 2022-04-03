import { Session } from "types";

/**
 * Retrieves a value from an object's translations
 * @param obj The object to retrieve the value from
 * @param field The field to retrieve the value from
 * @param languages The languages the user is requesting
 * @param showAny If true, will default to returning the first language if no value is found
 * @returns The value of the field in the object's translations
 */
export const getTranslation = (obj: any, field: string, languages: readonly string[], showAny: boolean = true): any => {
    if (!obj || !obj.translations) return undefined;
    // Loop through translations
    for (const translation of obj.translations) {
        // If this translation is one of the languages we're looking for, check for the field
        if (languages.includes(translation.language)) {
            if (translation[field]) return translation[field];
        }
    }
    if (showAny && obj.translations.length > 0) return obj.translations[0][field];
    // If we didn't find a translation, return undefined
    return undefined;
}

/**
 * Update a translation key/value pair for a specific language.
 * @param objectWithTranslation An object with a "translations" array
 * @param key The key to update
 * @param value The value to update
 * @param language 2 letter language code
 * @returns Updated translations array
 */
export const updateTranslationField = (objectWithTranslation: { [x: string]: any }, key: string, value: string, language: string): { [x: string]: any }[] => {
    if (!objectWithTranslation.translations) return [];
    let translationFound = false;
    let translations: any[] = []
    for (let translation of objectWithTranslation.translations) {
        if (translation.language === language) {
            translations.push({ ...translation, [key]: value });
            translationFound = true;
        } else {
            translations.push(translation);
        }
    }
    if (!translationFound) {
        translations.push({ language: language, [key]: value });
    }
    return translations;
}

/**
 * Update an entire translation object for a specific language.
 * @param objectWithTranslation An object with a "translations" array
 * @param translation The translation object to update, including at least the language code
 * @returns Updated translations array
 */
export const updateTranslation = (objectWithTranslation: { [x: string]: any }, translation: { language: string, [x: string]: any }): { [x: string]: any }[] => {
    if (!objectWithTranslation.translations) return [];
    let translationFound = false;
    let translations: any[] = []
    for (let existingTranslation of objectWithTranslation.translations) {
        if (existingTranslation.language === translation.language) {
            translations.push({ ...translation });
            translationFound = true;
        } else {
            translations.push(existingTranslation);
        }
    }
    if (!translationFound) {
        translations.push(translation);
    }
    return translations;
}

/**
 * Strips a language IETF code down to the subtag (e.g. en-US becomes en)
 * @param language IETF language code
 * @returns Subtag of language code
 */
export const getLanguageSubtag = (language: string): string => {
    if (!language) return "";
    const parts = language.split("-");
    return parts[0];
}

/**
 * Returns a list of user-preferred languages.
 * Priority order is the following: 
 * 1. Languages in session data
 * 2. Languages in browser (i.e. navigator.language)
 * 3. English
 * Strips languages so only the subtag is returned (e.g. en-US becomes en)
 * @param session Session data
 * @returns Array of user-preferred language subtags
 */
export const getUserLanguages = (session?: Session): string[] => {
    if (session?.languages && session.languages.length > 0) {
        return session.languages.map(getLanguageSubtag);
    }
    if (navigator.language) {
        return [getLanguageSubtag(navigator.language)];
    }
    return ["en"];
}
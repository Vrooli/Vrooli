/**
 * Finds the most preferred translation for a label, or the first one if none of the 
 * preferred languages are found.
 * @param translations The translations to search through
 * @param labelField The field to search for the label in
 * @param languages The preferred languages
 */
export const bestLabel = <
    LabelField extends string,
    Translation extends { [key in LabelField]: string | null } & { language: string }
>(
    translations: Translation[],
    labelField: LabelField,
    languages: string[],
): string => {
    for (const language of languages) {
        const translation = translations.find(({ language: lang }) => lang === language);
        if (translation) return translation[labelField] ?? '';
    }
    if (translations.length > 0) return translations[0][labelField] ?? '';
    return '';
}
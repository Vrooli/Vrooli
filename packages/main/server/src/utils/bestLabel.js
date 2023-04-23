export const bestLabel = (translations, labelField, languages) => {
    for (const language of languages) {
        const translation = translations.find(({ language: lang }) => lang === language);
        if (translation)
            return translation[labelField] ?? "";
    }
    if (translations.length > 0)
        return translations[0][labelField] ?? "";
    return "";
};
//# sourceMappingURL=bestLabel.js.map
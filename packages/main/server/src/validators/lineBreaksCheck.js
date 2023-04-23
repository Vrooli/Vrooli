import { CustomError } from "../events";
export const lineBreaksCheck = (input, fields, error, languages, k = 2) => {
    const checkTranslations = (translations, fields) => {
        translations.forEach((x) => {
            fields.forEach(field => {
                if (x[field] && x[field].split("\n").length > (k + 1)) {
                    throw new CustomError("0116", error, languages);
                }
            });
        });
    };
    if (input.translationsCreate)
        checkTranslations(input.translationsCreate, fields);
    if (input.translationsUpdate)
        checkTranslations(input.translationsUpdate, fields);
    fields.forEach(field => {
        if (input[field] && input[field].split("\n").length > (k + 1)) {
            throw new CustomError("0117", error, languages);
        }
    });
};
//# sourceMappingURL=lineBreaksCheck.js.map
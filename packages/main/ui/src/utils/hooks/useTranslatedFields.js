import { useField } from "formik";
import { useCallback, useEffect, useMemo, useState } from "react";
import { addEmptyTranslation, getFormikErrorsWithTranslations, removeTranslation } from "../display/translationTools";
export function useTranslatedFields({ defaultLanguage, fields, validationSchema, }) {
    const [language, setLanguage] = useState(defaultLanguage);
    const [field, meta, helpers] = useField("translations");
    const translationErrors = useMemo(() => getFormikErrorsWithTranslations(field, meta, validationSchema), [field, meta, validationSchema]);
    const languages = useMemo(() => field.value.map((t) => t.language), [field.value]);
    useEffect(() => {
        if (languages.length === 0 && field.value.length > 0) {
            setLanguage(field.value[0].language);
        }
    }, [field.value, languages.length, setLanguage]);
    const handleAddLanguage = useCallback((newLanguage) => {
        setLanguage(newLanguage);
        addEmptyTranslation(field, meta, helpers, newLanguage);
    }, [field, helpers, meta]);
    const handleDeleteLanguage = useCallback((language) => {
        const newLanguages = [...languages.filter(l => l !== language)];
        if (newLanguages.length === 0)
            return;
        setLanguage(newLanguages[0]);
        removeTranslation(field, meta, helpers, language);
    }, [field, helpers, languages, meta]);
    return {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        setLanguage,
        translationErrors,
    };
}
//# sourceMappingURL=useTranslatedFields.js.map
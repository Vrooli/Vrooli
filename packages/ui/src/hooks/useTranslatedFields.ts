import { useField } from "formik";
import { useCallback, useEffect, useMemo, useState } from "react";
import { addEmptyTranslation, getFormikErrorsWithTranslations, removeTranslation } from "utils/display/translationTools";
import * as yup from "yup";

/**
 * Hook to get translated fields, touched status, error messages, and other related data
 * @param defaultLanguage The default language to use
 * @param validationSchema The validation schema to use for error messages
 * @returns An object containing the translated fields, touched status, and error messages
 */
export function useTranslatedFields({
    defaultLanguage,
    validationSchema,
}: {
    defaultLanguage: string,
    validationSchema?: yup.AnyObjectSchema,
}) {
    // Language state
    const [language, setLanguage] = useState<string>(defaultLanguage);

    // Get the translated fields, touched status, and error messages
    const [field, meta, helpers] = useField("translations");
    const translationErrors = useMemo(() => validationSchema ? getFormikErrorsWithTranslations(field, validationSchema) : {}, [field, validationSchema]);

    // Find languages with translations
    const languages = useMemo(() => field.value?.map((t: { language: string }) => t.language) ?? [], [field.value]);

    // Set language to first language if it's empty
    useEffect(() => {
        if (languages.length === 0 && field.value?.length > 0) {
            setLanguage(field.value[0].language);
        }
    }, [field.value, languages.length, setLanguage]);

    // Functions for adding, removing, blurring, etc.
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguage(newLanguage);
        addEmptyTranslation(field, meta, helpers, newLanguage);
    }, [field, helpers, meta]);
    const handleDeleteLanguage = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)];
        if (newLanguages.length === 0) return;
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

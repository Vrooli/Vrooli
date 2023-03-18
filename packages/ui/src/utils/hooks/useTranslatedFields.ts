import { useField } from "formik";
import { useCallback, useEffect, useMemo, useState } from "react";
import { addEmptyTranslation, getFormikErrorsWithTranslations, getTranslationData, handleTranslationBlur, handleTranslationChange, removeTranslation } from "utils/display/translationTools";
import { booleanSpread, stringSpread } from "utils/shape/general";
import * as yup from 'yup';

/**
 * Hook to get translated fields, touched status, error messages, and other related data
 * @param defaultLanguage The default language to use
 * @param fields The fields to get the translated values for
 * @param formik The formik object
 * @param formikField The formik.values field which contains the translations
 * @param validationSchema The validation schema to use for error messages
 * @returns An object containing the translated fields, touched status, and error messages
 */
export function useTranslatedFields({
    defaultLanguage,
    fields,
    validationSchema,
}: {
    defaultLanguage: string,
    fields: readonly string[],
    validationSchema: yup.ObjectSchema<any>
}) {
    // Language state
    const [language, setLanguage] = useState<string>(defaultLanguage);

    // Get the translated fields, touched status, and error messages
    const [field, meta, helpers] = useField('translations');
    const translations = useMemo(() => {
        const { error, touched, value } = getTranslationData(field, meta, language);
        return {
            ...stringSpread(value, fields),
            ...stringSpread(error, fields, 'error'),
            ...booleanSpread(touched, fields, 'touched'),
            errorsWithTranslations: getFormikErrorsWithTranslations(field, meta, validationSchema),
        } as any
    }, [field, fields, language, meta, validationSchema]);

    // Find languages with translations
    const languages = useMemo(() => field.value.map((t: any) => t.language), [field.value]);

    // Set language to first language if it's empty
    useEffect(() => {
        if (languages.length === 0 && field.value.length > 0) {
            setLanguage(field.value[0].language);
        }
    }, [field.value, languages.length, setLanguage])

    // Functions for adding, removing, blurring, etc.
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguage(newLanguage);
        addEmptyTranslation(field, meta, helpers, newLanguage);
    }, [field, helpers, meta]);
    const handleDeleteLanguage = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        setLanguage(newLanguages[0]);
        removeTranslation(field, meta, helpers, language);
    }, [field, helpers, languages, meta]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
        handleTranslationBlur(field, meta, e, language)
    }, [field, language, meta]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(field, meta, helpers, e, language)
    }, [field, helpers, language, meta]);

    return {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        onTranslationBlur,
        onTranslationChange,
        setLanguage,
        translations,
    }
}

import { FormikProps } from "formik";
import { useCallback, useMemo, useState } from "react";
import { addEmptyTranslation, getFormikErrorsWithTranslations, getTranslationData, handleTranslationBlur, handleTranslationChange, removeTranslation } from "utils/display";
import { booleanSpread, stringSpread } from "utils/shape";
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
export function useTranslatedFields<
    FormikField extends keyof Object & string,
    Fields extends keyof Object[FormikField][0] & string,
    Object extends { [P1 in FormikField]: { [P2 in Fields]?: string | null | undefined }[] }
>({
    defaultLanguage,
    fields,
    formik,
    formikField,
    validationSchema,
}: {
    defaultLanguage: string,
    fields: readonly Fields[],
    formik: FormikProps<Object>,
    formikField: FormikField,
    validationSchema: yup.ObjectSchema<any>
}) {
    // Language state
    const [language, setLanguage] = useState<string>(defaultLanguage);

    // Get the translated fields, touched status, and error messages
    const translations = useMemo(() => {
        const { error, touched, value } = getTranslationData(formik as any, formikField, language);
        return {
            ...stringSpread(value, fields),
            ...stringSpread(error, fields, 'error'),
            ...booleanSpread(touched, fields, 'touched'),
            errorsWithTranslations: getFormikErrorsWithTranslations(formik as any, formikField, validationSchema),
        }
    }, [fields, formik, formikField, language, validationSchema]);

    // Find languages with translations
    const languages = useMemo(() => formik.values[formikField].map(t => (t as any).language), [formik.values, formikField]);

    // Functions for adding, removing, blurring, etc.
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguage(newLanguage);
        addEmptyTranslation(formik as any, formikField, newLanguage);
    }, [formik, formikField]);
    const handleDeleteLanguage = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        setLanguage(newLanguages[0]);
        removeTranslation(formik as any, formikField, language);
    }, [formik, formikField, languages]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
        handleTranslationBlur(formik as any, formikField, e, language)
    }, [formik, formikField, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik as any, formikField, e, language)
    }, [formik, formikField, language]);

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

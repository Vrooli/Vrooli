import { FormikProps } from "formik";
import { useMemo } from "react";
import { getFormikErrorsWithTranslations, getTranslationData } from "utils/display";
import { booleanSpread, stringSpread } from "utils/shape";
import * as yup from 'yup';

/**
 * Hook to get translated fields, touched status, and error messages
 * @param fields The fields to get the translated values for
 * @param formik The formik object
 * @param formikField The formik.values field which contains the translations
 * @param language The language to get the fields for
 * @param validationSchema The validation schema to use for error messages
 * @returns An object containing the translated fields, touched status, and error messages
 */
export function useTranslatedFields<
    FormikField extends keyof Object & string,
    Fields extends keyof Object[FormikField][0] & string,
    Object extends { [P1 in FormikField]: { [P2 in Fields]?: string | null | undefined }[] }
>({
    fields,
    formik,
    formikField,
    language,
    validationSchema,
}: {
    fields: readonly Fields[],
    formik: FormikProps<Object>,
    formikField: FormikField,
    language: string,
    validationSchema: yup.ObjectSchema<any>
}) {
    const data = useMemo(() => {
        const { error, touched, value } = getTranslationData(formik as any, formikField, language);
        return {
            ...stringSpread(value, fields),
            ...stringSpread(error, fields, 'error'),
            ...booleanSpread(touched, fields, 'touched'),
            errorsWithTranslations: getFormikErrorsWithTranslations(formik as any, formikField, validationSchema),
        }
    }, [fields, formik, formikField, language, validationSchema]);
    return data;
}

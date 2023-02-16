import { isObject } from "@shared/utils";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedLanguageInput = ({
    disabled,
    fieldData,
    formik,
    index,
    session,
    zIndex,
}: GeneratedInputComponentProps) => {
    console.log('rendering language input');
    let languages: string[] = [];
    if (isObject(formik.values) && Array.isArray(formik.values[fieldData.fieldName]) && fieldData.fieldName in formik.values) {
        languages = formik.values[fieldData.fieldName] as string[];
    }
    const addLanguage = (lang: string) => {
        formik.setFieldValue(fieldData.fieldName, [...languages, lang]);
    };
    const deleteLanguage = (lang: string) => {
        const newLanguages = [...languages.filter(l => l !== lang)]
        formik.setFieldValue(fieldData.fieldName, newLanguages);
    }
    return (
        <LanguageInput
            currentLanguage={languages.length > 0 ? languages[0] : ''} //TOOD
            disabled={disabled}
            handleAdd={addLanguage}
            handleDelete={deleteLanguage}
            handleCurrent={() => { }} //TODO
            session={session}
            translations={languages.map(language => ({ language }))}
            zIndex={zIndex}
        />
    )
}
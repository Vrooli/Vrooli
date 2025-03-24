import { LanguageFormInput, getFormikFieldName } from "@local/shared";
import { useField } from "formik";
import { LanguageInput } from "../LanguageInput/LanguageInput.js";
import { FormInputProps } from "./types.js";

export function FormInputLanguage({
    disabled,
    fieldData,
    fieldNamePrefix,
    index,
}: FormInputProps<LanguageFormInput>) {
    const [field, , helpers] = useField(getFormikFieldName(fieldData.fieldName, fieldNamePrefix));

    function addLanguage(lang: string) {
        helpers.setValue([...field.value, lang]);
    }
    function deleteLanguage(lang: string) {
        const newLanguages = [...field.value.filter(l => l !== lang)];
        helpers.setValue(newLanguages);
    }

    return (
        <LanguageInput
            currentLanguage={field.value.length > 0 ? field.value[0] : ""} //TOOD
            disabled={disabled}
            handleAdd={addLanguage}
            handleDelete={deleteLanguage}
            handleCurrent={() => { }} //TODO
            languages={field.value}
        />
    );
}

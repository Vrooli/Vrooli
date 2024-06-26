import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { useField } from "formik";
import { LanguageFormInput } from "forms/types";
import { FormInputProps } from "../types";

export function FormInputLanguage({
    disabled,
    fieldData,
    index,
}: FormInputProps<LanguageFormInput>) {
    const [field, , helpers] = useField(fieldData.fieldName);

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

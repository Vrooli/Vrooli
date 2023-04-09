import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { useField } from "formik";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedLanguageInput = ({
    disabled,
    fieldData,
    index,
    zIndex,
}: GeneratedInputComponentProps) => {
    console.log('rendering language input');
    const [field, , helpers] = useField(fieldData.fieldName);

    const addLanguage = (lang: string) => {
        helpers.setValue([...field.value, lang]);
    };
    const deleteLanguage = (lang: string) => {
        const newLanguages = [...field.value.filter(l => l !== lang)];
        helpers.setValue(newLanguages);
    }
    return (
        <LanguageInput
            currentLanguage={field.value.length > 0 ? field.value[0] : ''} //TOOD
            disabled={disabled}
            handleAdd={addLanguage}
            handleDelete={deleteLanguage}
            handleCurrent={() => { }} //TODO
            languages={field.value}
            zIndex={zIndex}
        />
    )
}
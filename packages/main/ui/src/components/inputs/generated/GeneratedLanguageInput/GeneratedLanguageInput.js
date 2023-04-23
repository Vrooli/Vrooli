import { jsx as _jsx } from "react/jsx-runtime";
import { useField } from "formik";
import { LanguageInput } from "../../LanguageInput/LanguageInput";
export const GeneratedLanguageInput = ({ disabled, fieldData, index, zIndex, }) => {
    console.log("rendering language input");
    const [field, , helpers] = useField(fieldData.fieldName);
    const addLanguage = (lang) => {
        helpers.setValue([...field.value, lang]);
    };
    const deleteLanguage = (lang) => {
        const newLanguages = [...field.value.filter(l => l !== lang)];
        helpers.setValue(newLanguages);
    };
    return (_jsx(LanguageInput, { currentLanguage: field.value.length > 0 ? field.value[0] : "", disabled: disabled, handleAdd: addLanguage, handleDelete: deleteLanguage, handleCurrent: () => { }, languages: field.value, zIndex: zIndex }));
};
//# sourceMappingURL=GeneratedLanguageInput.js.map
import { jsx as _jsx } from "react/jsx-runtime";
import { useField } from "formik";
import { getTranslationData, handleTranslationChange } from "../../../utils/display/translationTools";
import { MarkdownInputBase } from "../MarkdownInputBase/MarkdownInputBase";
export const TranslatedMarkdownInput = ({ language, name, ...props }) => {
    const [field, meta, helpers] = useField("translations");
    const { value, error, touched } = getTranslationData(field, meta, language);
    const handleBlur = (event) => {
        field.onBlur(event);
    };
    const handleChange = (newText) => {
        handleTranslationChange(field, meta, helpers, { target: { name, value: newText } }, language);
    };
    return (_jsx(MarkdownInputBase, { ...props, name: name, value: value?.[name] || "", error: touched?.[name] && Boolean(error?.[name]), helperText: touched?.[name] && error?.[name], onBlur: handleBlur, onChange: handleChange }));
};
//# sourceMappingURL=TranslatedMarkdownInput.js.map
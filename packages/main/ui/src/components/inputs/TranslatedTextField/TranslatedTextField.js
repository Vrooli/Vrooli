import { jsx as _jsx } from "react/jsx-runtime";
import { TextField } from "@mui/material";
import { useField } from "formik";
import { getTranslationData, handleTranslationChange } from "../../../utils/display/translationTools";
export const TranslatedTextField = ({ language, name, ...props }) => {
    const [field, meta, helpers] = useField("translations");
    const { value, error, touched } = getTranslationData(field, meta, language);
    const handleBlur = (event) => {
        field.onBlur(event);
    };
    const handleChange = (event) => {
        handleTranslationChange(field, meta, helpers, event, language);
    };
    return (_jsx(TextField, { ...props, id: name, name: name, value: value?.[name] || "", error: touched?.[name] && Boolean(error?.[name]), helperText: touched?.[name] && error?.[name], onBlur: handleBlur, onChange: handleChange }));
};
//# sourceMappingURL=TranslatedTextField.js.map
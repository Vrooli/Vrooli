import { jsx as _jsx } from "react/jsx-runtime";
import { useField } from "formik";
import { MarkdownInputBase } from "../MarkdownInputBase/MarkdownInputBase";
export const MarkdownInput = ({ name, ...props }) => {
    const [field, meta] = useField(name);
    return (_jsx(MarkdownInputBase, { ...props, name: name, value: field.value, error: meta.touched && !!meta.error, helperText: meta.touched && meta.error, onBlur: field.onBlur, onChange: field.onChange }));
};
//# sourceMappingURL=MarkdownInput.js.map
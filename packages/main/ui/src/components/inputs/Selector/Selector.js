import { jsx as _jsx } from "react/jsx-runtime";
import { useField } from "formik";
import { SelectorBase } from "../SelectorBase/SelectorBase";
export const Selector = ({ name, onChange, ...props }) => {
    const [field, meta] = useField(name);
    return (_jsx(SelectorBase, { ...props, name: name, value: field.value, error: meta.touched && !!meta.error, helperText: meta.touched && meta.error, onBlur: field.onBlur, onChange: (newValue) => {
            if (onChange)
                onChange(newValue);
            field.onChange(newValue);
        } }));
};
//# sourceMappingURL=Selector.js.map
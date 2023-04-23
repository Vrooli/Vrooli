import { jsx as _jsx } from "react/jsx-runtime";
import { TextField } from "@mui/material";
import { useField } from "formik";
import { useMemo } from "react";
export const GeneratedTextField = ({ fieldData, index, }) => {
    const [field, meta] = useField(fieldData.fieldName);
    const props = useMemo(() => fieldData.props, [fieldData]);
    const multiLineProps = props.maxRows ? { multiline: true, rows: props.maxRows } : {};
    const hasDescription = typeof fieldData.description === "string" && fieldData.description.trim().length > 0;
    console.log("rendering text field");
    return (_jsx(TextField, { autoComplete: props.autoComplete, autoFocus: index === 0, fullWidth: true, id: fieldData.fieldName, InputLabelProps: { shrink: true }, name: fieldData.fieldName, placeholder: hasDescription ? fieldData.description : `${fieldData.label}...`, required: fieldData.yup?.required, value: field.value, onBlur: field.onBlur, onChange: field.onChange, tabIndex: index, error: meta.touched && !!meta.error, helperText: meta.touched && meta.error, ...multiLineProps }, `field-${fieldData.fieldName}-${index}`));
};
//# sourceMappingURL=GeneratedTextField.js.map
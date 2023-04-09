import { TextField, TextFieldProps } from "@mui/material";
import { useField } from "formik";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedTextField = ({
    fieldData,
    index,
}: GeneratedInputComponentProps) => {
    const [field, meta] = useField(fieldData.fieldName);
    const props = useMemo(() => fieldData.props as TextFieldProps, [fieldData]);
    const multiLineProps = props.maxRows ? { multiline: true, rows: props.maxRows } : {};
    const hasDescription = typeof fieldData.description === 'string' && fieldData.description.trim().length > 0;

    console.log('rendering text field');

    return (
        <TextField
            key={`field-${fieldData.fieldName}-${index}`}
            autoComplete={props.autoComplete}
            autoFocus={index === 0}
            fullWidth
            id={fieldData.fieldName}
            InputLabelProps={{ shrink: true }}
            name={fieldData.fieldName}
            placeholder={hasDescription ? fieldData.description as string : `${fieldData.label}...`}
            required={fieldData.yup?.required}
            value={field.value}
            onBlur={field.onBlur}
            onChange={field.onChange}
            tabIndex={index}
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
            {...multiLineProps}
        />
    );
}
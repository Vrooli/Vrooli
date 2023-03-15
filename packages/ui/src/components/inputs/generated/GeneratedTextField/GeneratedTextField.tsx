import { TextField, TextFieldProps } from "@mui/material";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedTextField = ({
    fieldData,
    formik,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering text field');
    const props = useMemo(() => fieldData.props as TextFieldProps, [fieldData]);
    const multiLineProps = props.maxRows ? { multiline: true, rows: props.maxRows } : {};
    const hasDescription = typeof fieldData.description === 'string' && fieldData.description.trim().length > 0;
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
            value={formik.values[fieldData.fieldName]}
            onBlur={formik.handleBlur}
            onChange={formik.handleChange}
            tabIndex={index}
            error={formik.touched[fieldData.fieldName] && Boolean(formik.errors[fieldData.fieldName])}
            helperText={formik.touched[fieldData.fieldName] && formik.errors[fieldData.fieldName]}
            {...multiLineProps}
        />
    );
}
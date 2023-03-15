import { IntegerInput } from "components/inputs/IntegerInput/IntegerInput";
import { IntegerInputProps } from "forms/types";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedIntegerInput = ({
    fieldData,
    formik,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering integer input');
    const props = useMemo(() => fieldData.props as IntegerInputProps, [fieldData.props]);

    return (
        <IntegerInput
            autoFocus={index === 0}
            key={`field-${fieldData.fieldName}-${index}`}
            tabIndex={index}
            id={fieldData.fieldName}
            label={fieldData.label}
            min={props.min ?? 0}
            tooltip={props.tooltip ?? ''}
            value={formik.values[fieldData.fieldName]}
            onBlur={formik.handleBlur}
            handleChange={(value: number) => formik.setFieldValue(fieldData.fieldName, value)}
            error={formik.touched[fieldData.fieldName] && Boolean(formik.errors[fieldData.fieldName])}
            helperText={formik.touched[fieldData.fieldName] && formik.errors[fieldData.fieldName]}
        />
    );
}
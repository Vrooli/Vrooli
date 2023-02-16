import { QuantityBox } from "components/inputs/QuantityBox/QuantityBox";
import { QuantityBoxProps } from "forms/types";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedQuantityBox = ({
    fieldData,
    formik,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering quantity box');
    const props = useMemo(() => fieldData.props as QuantityBoxProps, [fieldData.props]);

    return (
        <QuantityBox
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
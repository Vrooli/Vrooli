import { JsonInput } from "components/inputs/JsonInput/JsonInput";
import { JsonProps } from "forms/types";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedJsonInput = ({
    disabled,
    fieldData,
    formik,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering json input');
    const props = useMemo(() => fieldData.props as JsonProps, [fieldData.props]);

    return (
        <JsonInput
            id={fieldData.fieldName}
            disabled={disabled}
            format={props.format}
            variables={props.variables}
            placeholder={props.placeholder ?? fieldData.label}
            value={formik.values[fieldData.fieldName]}
            minRows={props.minRows}
            onChange={(newText: string) => formik.setFieldValue(fieldData.fieldName, newText)}
            error={formik.touched[fieldData.fieldName] && Boolean(formik.errors[fieldData.fieldName])}
            helperText={formik.touched[fieldData.fieldName] && formik.errors[fieldData.fieldName]}
        />
    )
}
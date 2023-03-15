import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { MarkdownProps } from "forms/types";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedMarkdownInput = ({
    disabled,
    fieldData,
    formik,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering markdown input');
    const props = useMemo(() => fieldData.props as MarkdownProps, [fieldData.props]);

    return (
        <MarkdownInput
            id={fieldData.fieldName}
            disabled={disabled}
            placeholder={props.placeholder ?? fieldData.label}
            value={formik.values[fieldData.fieldName]}
            minRows={props.minRows}
            onChange={(newText: string) => formik.setFieldValue(fieldData.fieldName, newText)}
            error={formik.touched[fieldData.fieldName] && Boolean(formik.errors[fieldData.fieldName])}
            helperText={formik.touched[fieldData.fieldName] && formik.errors[fieldData.fieldName]}
        />
    )
}
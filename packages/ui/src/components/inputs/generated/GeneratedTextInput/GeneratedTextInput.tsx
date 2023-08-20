import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { Field } from "formik";
import { TextProps } from "forms/types";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedTextInput = ({
    disabled,
    fieldData,
    index,
}: GeneratedInputComponentProps) => {
    console.log("rendering markdown input");
    const props = useMemo(() => fieldData.props as TextProps, [fieldData.props]);

    if (props.isMarkdown) {
        return (
            <MarkdownInput
                disabled={disabled}
                name={fieldData.fieldName}
                placeholder={props.placeholder ?? fieldData.label}
                maxChars={props.maxChars}
                maxRows={props.maxRows}
                minRows={props.minRows}
            />
        );
    }

    const multiLineProps = props.maxRows ? { multiline: true, rows: props.maxRows } : {};
    const hasDescription = typeof fieldData.description === "string" && fieldData.description.trim().length > 0;
    return (
        <Field
            key={`field-${fieldData.fieldName}-${index}`}
            autoComplete={props.autoComplete}
            autoFocus={index === 0}
            fullWidth
            id={fieldData.fieldName}
            InputLabelProps={{ shrink: true }}
            name={fieldData.fieldName}
            placeholder={hasDescription ? fieldData.description as string : `${fieldData.label}...`}
            required={fieldData.yup?.required}
            tabIndex={index}
            // error={props.error}
            // helperText={props.helperText}
            {...multiLineProps}
        />
    )
};

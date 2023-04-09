import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { MarkdownProps } from "forms/types";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedMarkdownInput = ({
    disabled,
    fieldData,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering markdown input');
    const props = useMemo(() => fieldData.props as MarkdownProps, [fieldData.props]);

    return (
        <MarkdownInput
            disabled={disabled}
            name={fieldData.fieldName}
            placeholder={props.placeholder ?? fieldData.label}
            minRows={props.minRows}
        />
    )
}
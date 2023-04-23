import { useMemo } from "react";
import { MarkdownProps } from "../../../../forms/types";
import { MarkdownInput } from "../../MarkdownInput/MarkdownInput";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedMarkdownInput = ({
    disabled,
    fieldData,
    index,
}: GeneratedInputComponentProps) => {
    console.log("rendering markdown input");
    const props = useMemo(() => fieldData.props as MarkdownProps, [fieldData.props]);

    return (
        <MarkdownInput
            disabled={disabled}
            name={fieldData.fieldName}
            placeholder={props.placeholder ?? fieldData.label}
            minRows={props.minRows}
        />
    );
};

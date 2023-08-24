import { CodeInput } from "components/inputs/CodeInput/CodeInput";
import { CodeProps } from "forms/types";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedCodeInput = ({
    disabled,
    fieldData,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering code input');
    const props = useMemo(() => fieldData.props as CodeProps, [fieldData.props]);

    return (
        <CodeInput
            {...props}
        // name={fieldData.fieldName}
        />
    )
}
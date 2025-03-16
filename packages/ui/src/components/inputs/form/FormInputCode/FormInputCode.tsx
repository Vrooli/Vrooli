import { CodeFormInput } from "@local/shared";
import { CodeInput } from "components/inputs/CodeInput/CodeInput.js";
import { useMemo } from "react";
import { FormInputProps } from "../types.js";

export function FormInputCode({
    disabled,
    fieldData,
}: FormInputProps<CodeFormInput>) {
    const props = useMemo(() => fieldData.props, [fieldData.props]);

    return (
        <CodeInput
            disabled={disabled}
            name={fieldData.fieldName}
            // tabIndex={index}
            {...props}
        />
    );
}

import { type CodeFormInput } from "@local/shared";
import { useMemo } from "react";
import { CodeInput } from "../../inputs/CodeInput/CodeInput.js";
import { type FormInputProps } from "./types.js";

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

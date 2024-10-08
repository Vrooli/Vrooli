import { CodeFormInput } from "@local/shared";
import { CodeInput } from "components/inputs/CodeInput/CodeInput";
import { useMemo } from "react";
import { FormInputProps } from "../types";

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

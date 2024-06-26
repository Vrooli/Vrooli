import { CodeInput } from "components/inputs/CodeInput/CodeInput";
import { CodeFormInput } from "forms/types";
import { useMemo } from "react";
import { FormInputProps } from "../types";

export function FormInputCode({
    disabled,
    fieldData,
    index,
}: FormInputProps<CodeFormInput>) {
    const props = useMemo(() => fieldData.props, [fieldData.props]);

    return (
        <CodeInput
            {...props}
        // name={fieldData.fieldName}
        />
    );
}

import { useMemo } from "react";
import { IntegerInputProps } from "../../../../forms/types";
import { IntegerInput } from "../../IntegerInput/IntegerInput";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedIntegerInput = ({
    fieldData,
    index,
}: GeneratedInputComponentProps) => {
    console.log("rendering integer input");
    const props = useMemo(() => fieldData.props as IntegerInputProps, [fieldData.props]);

    return (
        <IntegerInput
            autoFocus={index === 0}
            key={`field-${fieldData.fieldName}-${index}`}
            tabIndex={index}
            label={fieldData.label}
            min={props.min ?? 0}
            name={fieldData.fieldName}
            tooltip={props.tooltip ?? ""}
        />
    );
};
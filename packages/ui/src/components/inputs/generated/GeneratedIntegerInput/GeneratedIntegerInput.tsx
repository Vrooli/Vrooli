import { IntegerInput } from "components/inputs/IntegerInput/IntegerInput";
import { IntegerInputProps } from "forms/types";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedIntegerInput = ({
    fieldData,
    index,
}: GeneratedInputComponentProps) => {
    const props = useMemo(() => fieldData.props as IntegerInputProps, [fieldData.props]);

    return (
        <IntegerInput
            autoFocus={index === 0}
            key={`field-${fieldData.fieldName}-${index}`}
            tabIndex={index}
            label={fieldData.label}
            min={props.min}
            name={fieldData.fieldName}
            tooltip={props.tooltip}
            zeroText={props.zeroText}
        />
    );
}
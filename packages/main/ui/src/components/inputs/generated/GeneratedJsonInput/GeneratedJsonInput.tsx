import { useMemo } from "react";
import { JsonProps } from "../../../../forms/types";
import { JsonInput } from "../../JsonInput/JsonInput";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedJsonInput = ({
    disabled,
    fieldData,
    index,
}: GeneratedInputComponentProps) => {
    console.log("rendering json input");
    const props = useMemo(() => fieldData.props as JsonProps, [fieldData.props]);

    return (
        <JsonInput
            disabled={disabled}
            format={props.format}
            variables={props.variables}
            placeholder={props.placeholder ?? fieldData.label}
            minRows={props.minRows}
            name={fieldData.fieldName}
        />
    );
};

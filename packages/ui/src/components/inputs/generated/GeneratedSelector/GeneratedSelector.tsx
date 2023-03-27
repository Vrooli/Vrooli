import { Selector } from "components/inputs/Selector/Selector";
import { SelectorProps } from "forms/types";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedSelector = ({
    disabled,
    fieldData,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering selector');
    const props = useMemo(() => fieldData.props as SelectorProps<any>, [fieldData.props]);

    return (
        <Selector
            key={`field-${fieldData.fieldName}-${index}`}
            autoFocus={index === 0}
            disabled={disabled}
            options={props.options}
            getOptionLabel={props.getOptionLabel}
            name={fieldData.fieldName}
            fullWidth
            inputAriaLabel={`select-input-${fieldData.fieldName}`}
            noneOption={props.noneOption}
            label={fieldData.label}
            tabIndex={index}
        />
    );
}
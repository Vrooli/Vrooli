import { Selector } from "components/inputs/Selector/Selector";
import { SelectorProps } from "forms/types";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedSelector = ({
    disabled,
    fieldData,
    formik,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering selector');
    const props = useMemo(() => fieldData.props as SelectorProps<any>, [fieldData.props]);

    return (
        <Selector
            key={`field-${fieldData.fieldName}-${index}`}
            disabled={disabled}
            options={props.options}
            getOptionLabel={props.getOptionLabel}
            selected={formik.values[fieldData.fieldName]}
            onBlur={formik.handleBlur}
            handleChange={formik.handleChange}
            fullWidth
            inputAriaLabel={`select-input-${fieldData.fieldName}`}
            noneOption={props.noneOption}
            label={fieldData.label}
            color={props.color}
            tabIndex={index}
        />
    );
}
import { Slider, SliderProps } from "@mui/material";
import { useField } from "formik";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedSlider = ({
    disabled,
    fieldData,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering slider');
    const [field] = useField(fieldData.fieldName);
    const props = useMemo(() => fieldData.props as SliderProps, [fieldData.props]);

    return (
        <Slider
            aria-label={fieldData.fieldName}
            disabled={disabled}
            key={`field-${fieldData.fieldName}-${index}`}
            min={props.min}
            max={props.max}
            name={fieldData.fieldName}
            step={props.step}
            valueLabelDisplay={props.valueLabelDisplay}
            value={field.value ?? props.defaultValue}
            onBlur={field.onBlur}
            onChange={field.onChange}
            tabIndex={index}
        />
    );
}
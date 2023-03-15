import { Slider, SliderProps } from "@mui/material";
import { useMemo } from "react";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedSlider = ({
    disabled,
    fieldData,
    formik,
    index,
}: GeneratedInputComponentProps) => {
    console.log('rendering slider');
    const props = useMemo(() => fieldData.props as SliderProps, [fieldData.props]);

    return (
        <Slider
            aria-label={fieldData.fieldName}
            disabled={disabled}
            key={`field-${fieldData.fieldName}-${index}`}
            min={props.min}
            max={props.max}
            step={props.step}
            valueLabelDisplay={props.valueLabelDisplay}
            value={formik.values[fieldData.fieldName]}
            onBlur={formik.handleBlur}
            onChange={formik.handleChange}
            tabIndex={index}
        />
    );
}
import { jsx as _jsx } from "react/jsx-runtime";
import { Slider } from "@mui/material";
import { useField } from "formik";
import { useMemo } from "react";
export const GeneratedSlider = ({ disabled, fieldData, index, }) => {
    console.log("rendering slider");
    const [field] = useField(fieldData.fieldName);
    const props = useMemo(() => fieldData.props, [fieldData.props]);
    return (_jsx(Slider, { "aria-label": fieldData.fieldName, disabled: disabled, min: props.min, max: props.max, name: fieldData.fieldName, step: props.step, valueLabelDisplay: props.valueLabelDisplay, value: field.value ?? props.defaultValue, onBlur: field.onBlur, onChange: field.onChange, tabIndex: index }, `field-${fieldData.fieldName}-${index}`));
};
//# sourceMappingURL=GeneratedSlider.js.map
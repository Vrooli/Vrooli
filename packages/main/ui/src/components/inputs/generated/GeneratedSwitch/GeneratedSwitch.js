import { jsx as _jsx } from "react/jsx-runtime";
import { FormControlLabel, FormGroup, Switch } from "@mui/material";
import { useField } from "formik";
import { useMemo } from "react";
export const GeneratedSwitch = ({ disabled, fieldData, index, }) => {
    console.log("rendering switch");
    const [field] = useField(fieldData.fieldName);
    const props = useMemo(() => fieldData.props, [fieldData.props]);
    return (_jsx(FormGroup, { children: _jsx(FormControlLabel, { tabIndex: index, control: (_jsx(Switch, { autoFocus: index === 0, disabled: disabled, size: props.size, color: props.color, tabIndex: index, checked: field.value ?? props.defaultValue, onBlur: field.onBlur, onChange: field.onChange, name: fieldData.fieldName, inputProps: { "aria-label": fieldData.fieldName } })), label: fieldData.fieldName }) }, `field-${fieldData.fieldName}-${index}`));
};
//# sourceMappingURL=GeneratedSwitch.js.map
import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { FormControl, FormControlLabel, FormLabel, Radio, RadioGroup } from "@mui/material";
import { useField } from "formik";
import { useMemo } from "react";
export const GeneratedRadio = ({ disabled, fieldData, index, }) => {
    console.log("rendering radio");
    const [field] = useField(fieldData.fieldName);
    const props = useMemo(() => fieldData.props, [fieldData.props]);
    return (_jsxs(FormControl, { component: "fieldset", disabled: disabled, sx: { paddingLeft: 1 }, children: [_jsx(FormLabel, { component: "legend", children: fieldData.label }), _jsx(RadioGroup, { "aria-label": fieldData.fieldName, row: props.row, id: fieldData.fieldName, name: fieldData.fieldName, value: field.value ?? props.defaultValue, defaultValue: props.defaultValue, onBlur: field.onBlur, onChange: field.onChange, tabIndex: index, children: props.options.map((option, index) => (_jsx(FormControlLabel, { value: option.value, control: _jsx(Radio, {}), label: option.label }, index))) })] }, `field-${fieldData.fieldName}-${index}`));
};
//# sourceMappingURL=GeneratedRadio.js.map
import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Checkbox, FormControl, FormControlLabel, FormGroup, FormHelperText, FormLabel } from "@mui/material";
import { useField } from "formik";
import { useMemo } from "react";
import { updateArray } from "../../../../utils/shape/general";
export const GeneratedCheckbox = ({ disabled, fieldData, index, }) => {
    const [field, meta] = useField(fieldData.fieldName);
    const props = useMemo(() => fieldData.props, [fieldData.props]);
    console.log("rendering checkbox");
    return (_jsxs(FormControl, { disabled: disabled, required: fieldData.yup?.required, error: meta.touched && !!meta.error, name: fieldData.fieldName, component: "fieldset", variant: "standard", sx: { m: 3 }, children: [_jsx(FormLabel, { component: "legend", children: fieldData.label }), _jsx(FormGroup, { row: props.row ?? false, children: props.options.map((option, index) => (_jsx(FormControlLabel, { control: _jsx(Checkbox, { checked: (Array.isArray(field.value?.[fieldData.fieldName]) && field.value[fieldData.fieldName].length > index) ? field.value[fieldData.fieldName][index] === props.options[index] : false, onChange: (event) => { field.onChange(updateArray(field.value[fieldData.fieldName], index, !props.options[index])); }, name: `${fieldData.fieldName}-${index}`, id: `${fieldData.fieldName}-${index}`, value: props.options[index] }), label: option.label }))) }), meta.touched && !!meta.error && _jsx(FormHelperText, { children: meta.error })] }, `field-${fieldData.fieldName}-${index}`));
};
//# sourceMappingURL=GeneratedCheckbox.js.map
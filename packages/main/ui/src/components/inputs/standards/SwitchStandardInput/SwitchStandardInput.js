import { jsx as _jsx } from "react/jsx-runtime";
import { FormControlLabel, FormGroup, Switch } from "@mui/material";
import { useField } from "formik";
export const SwitchStandardInput = ({ isEditing, }) => {
    const [defaultValueField, , defaulValueHelpers] = useField("defaultValue");
    return (_jsx(FormGroup, { children: _jsx(FormControlLabel, { control: (_jsx(Switch, { disabled: !isEditing, size: "medium", color: "secondary", checked: defaultValueField.value, onChange: (event) => {
                    defaulValueHelpers.setValue(event.target.checked);
                } })), label: "Default checked" }) }));
};
//# sourceMappingURL=SwitchStandardInput.js.map
import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { ListItem, Stack, Typography, useTheme } from "@mui/material";
import { useField } from "formik";
import { ToggleSwitch } from "../../inputs/ToggleSwitch/ToggleSwitch";
export const SettingsToggleListItem = ({ description, disabled, name, title, }) => {
    const { palette } = useTheme();
    const [field] = useField(name);
    return (_jsxs(ListItem, { disablePadding: true, component: "a", sx: {
            display: "flex",
            background: palette.background.paper,
            padding: "8px 16px",
            cursor: "pointer",
            borderBottom: `1px solid ${palette.divider}`,
        }, children: [_jsxs(Stack, { direction: "column", spacing: 0, sx: {
                    width: "100%",
                    color: disabled ? palette.text.disabled : palette.text.primary,
                }, children: [_jsx(Typography, { variant: "h6", component: "div", children: title }), _jsx(Typography, { variant: "body2", component: "div", children: description })] }), _jsx(ToggleSwitch, { checked: field.value, disabled: disabled, name: name, onChange: field.onChange })] }));
};
//# sourceMappingURL=SettingsToggleListItem.js.map
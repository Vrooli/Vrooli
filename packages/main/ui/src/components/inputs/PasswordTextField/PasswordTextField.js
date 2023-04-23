import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { InvisibleIcon, VisibleIcon } from "@local/icons";
import { FormControl, FormHelperText, IconButton, InputAdornment, InputLabel, OutlinedInput, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
export const PasswordTextField = ({ autoComplete = "current-password", autoFocus = false, fullWidth = true, label, name, ...props }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [field, meta] = useField(name);
    const [showPassword, setShowPassword] = useState(false);
    const handleClickShowPassword = useCallback(() => {
        setShowPassword(!showPassword);
    }, [showPassword]);
    return (_jsxs(FormControl, { fullWidth: fullWidth, variant: "outlined", ...props, children: [_jsx(InputLabel, { htmlFor: name, children: label ?? "Password" }), _jsx(OutlinedInput, { id: name, name: name, type: showPassword ? "text" : "password", value: field.value, onBlur: field.onBlur, onChange: field.onChange, autoComplete: autoComplete, autoFocus: autoFocus, error: meta.touched && !!meta.error, endAdornment: _jsx(InputAdornment, { position: "end", children: _jsx(IconButton, { "aria-label": "toggle password visibility", onClick: handleClickShowPassword, edge: "end", children: showPassword ?
                            _jsx(InvisibleIcon, { fill: palette.background.textSecondary }) :
                            _jsx(VisibleIcon, { fill: palette.background.textSecondary }) }) }), label: label ?? t("Password") }), _jsx(FormHelperText, { id: "adornment-password-error-text", sx: { color: palette.error.main }, children: meta.touched && meta.error })] }));
};
//# sourceMappingURL=PasswordTextField.js.map
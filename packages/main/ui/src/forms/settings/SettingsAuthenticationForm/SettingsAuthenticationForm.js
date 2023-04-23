import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Grid, TextField } from "@mui/material";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { PasswordTextField } from "../../../components/inputs/PasswordTextField/PasswordTextField";
import { BaseForm } from "../../BaseForm/BaseForm";
export const SettingsAuthenticationForm = ({ display, dirty, isLoading, onCancel, values, ...props }) => {
    const { t } = useTranslation();
    return (_jsxs(BaseForm, { dirty: dirty, isLoading: isLoading, style: {
            width: { xs: "100%", md: "min(100%, 500px)" },
            margin: "auto",
            display: "block",
        }, children: [_jsx(TextField, { name: "username", autoComplete: "username", sx: { display: "none" } }), _jsxs(Grid, { container: true, spacing: 1, children: [_jsx(Grid, { item: true, xs: 12, children: _jsx(PasswordTextField, { fullWidth: true, name: "currentPassword", label: t("PasswordCurrent"), autoComplete: "current-password" }) }), _jsx(Grid, { item: true, xs: 12, children: _jsx(PasswordTextField, { fullWidth: true, name: "newPassword", label: t("PasswordNew"), autoComplete: "new-password" }) }), _jsx(Grid, { item: true, xs: 12, children: _jsx(PasswordTextField, { fullWidth: true, name: "newPasswordConfirmation", autoComplete: "new-password", label: t("PasswordNewConfirm") }) })] }), _jsx(GridSubmitButtons, { display: display, errors: props.errors, isCreate: false, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }));
};
//# sourceMappingURL=SettingsAuthenticationForm.js.map
import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { uuidValidate } from "@local/uuid";
import { emailResetPasswordSchema } from "@local/validation";
import { Button, Grid } from "@mui/material";
import { Formik } from "formik";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { authEmailResetPassword } from "../../../api/generated/endpoints/auth_emailResetPassword";
import { useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { PasswordTextField } from "../../../components/inputs/PasswordTextField/PasswordTextField";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { PubSub } from "../../../utils/pubsub";
import { parseSearchParams, useLocation } from "../../../utils/route";
import { BaseForm } from "../../BaseForm/BaseForm";
import { formPaper, formSubmit } from "../../styles";
export const ResetPasswordForm = ({ onClose, }) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [emailResetPassword, { loading }] = useCustomMutation(authEmailResetPassword);
    const { userId, code } = useMemo(() => {
        const params = parseSearchParams();
        if (typeof params.code !== "string" || !params.code.includes(":"))
            return { userId: undefined, code: undefined };
        const [userId, code] = params.code.split(":");
        if (!uuidValidate(userId))
            return { userId: undefined, code: undefined };
        return { userId, code };
    }, []);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: "dialog", onClose: onClose, titleData: {
                    titleKey: "ResetPassword",
                } }), _jsx(Formik, { initialValues: {
                    newPassword: "",
                    confirmNewPassword: "",
                }, onSubmit: (values, helpers) => {
                    if (!userId || !code) {
                        PubSub.get().publishSnack({ messageKey: "InvalidResetPasswordUrl", severity: "Error" });
                        return;
                    }
                    mutationWrapper({
                        mutation: emailResetPassword,
                        input: { id: userId, code, newPassword: values.newPassword },
                        onSuccess: (data) => {
                            PubSub.get().publishSession(data);
                            setLocation(LINKS.Home);
                        },
                        successMessage: () => ({ key: "PasswordReset" }),
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }, validationSchema: emailResetPasswordSchema, children: (formik) => _jsxs(BaseForm, { dirty: formik.dirty, isLoading: loading, style: {
                        display: "block",
                        ...formPaper,
                    }, children: [_jsxs(Grid, { container: true, spacing: 2, children: [_jsx(Grid, { item: true, xs: 12, children: _jsx(PasswordTextField, { fullWidth: true, autoFocus: true, name: "newPassword", autoComplete: "new-password", label: t("PasswordNew") }) }), _jsx(Grid, { item: true, xs: 12, children: _jsx(PasswordTextField, { fullWidth: true, name: "confirmNewPassword", autoComplete: "new-password", label: t("PasswordNewConfirm") }) })] }), _jsx(Button, { fullWidth: true, disabled: loading, type: "submit", color: "secondary", sx: { ...formSubmit }, children: t("Submit") })] }) })] }));
};
//# sourceMappingURL=ResetPasswordForm.js.map
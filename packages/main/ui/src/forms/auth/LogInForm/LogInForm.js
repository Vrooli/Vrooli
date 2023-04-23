import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { emailLogInFormValidation } from "@local/validation";
import { Button, Grid, Link, TextField, Typography } from "@mui/material";
import { Field, Formik } from "formik";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { authEmailLogIn } from "../../../api/generated/endpoints/auth_emailLogIn";
import { useCustomMutation } from "../../../api/hooks";
import { errorToCode, hasErrorCode, mutationWrapper } from "../../../api/utils";
import { PasswordTextField } from "../../../components/inputs/PasswordTextField/PasswordTextField";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { clickSize } from "../../../styles";
import { Forms } from "../../../utils/consts";
import { PubSub } from "../../../utils/pubsub";
import { parseSearchParams, useLocation } from "../../../utils/route";
import { BaseForm } from "../../BaseForm/BaseForm";
import { formNavLink, formPaper, formSubmit } from "../../styles";
export const LogInForm = ({ onClose, onFormChange = () => { }, }) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const { redirect, verificationCode } = useMemo(() => {
        const params = parseSearchParams();
        return {
            redirect: typeof params.redirect === "string" ? params.redirect : undefined,
            verificationCode: typeof params.code === "string" ? params.code : undefined,
        };
    }, []);
    const [emailLogIn, { loading }] = useCustomMutation(authEmailLogIn);
    const toForgotPassword = () => onFormChange(Forms.ForgotPassword);
    const toSignUp = () => onFormChange(Forms.SignUp);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: "dialog", onClose: onClose, titleData: {
                    titleKey: "LogIn",
                } }), _jsx(Formik, { initialValues: {
                    email: "",
                    password: "",
                }, onSubmit: (values, helpers) => {
                    mutationWrapper({
                        mutation: emailLogIn,
                        input: { ...values, verificationCode },
                        successCondition: (data) => data !== null,
                        onSuccess: (data) => {
                            if (verificationCode)
                                PubSub.get().publishSnack({ messageKey: "EmailVerified", severity: "Success" });
                            PubSub.get().publishSession(data);
                            setLocation(redirect ?? LINKS.Home);
                        },
                        showDefaultErrorSnack: false,
                        onError: (response) => {
                            if (hasErrorCode(response, "MustResetPassword")) {
                                PubSub.get().publishAlertDialog({
                                    messageKey: "ChangePasswordBeforeLogin",
                                    buttons: [
                                        { labelKey: "Ok", onClick: () => { setLocation(redirect ?? LINKS.Home); } },
                                    ],
                                });
                            }
                            else if (hasErrorCode(response, "EmailNotFound")) {
                                PubSub.get().publishSnack({
                                    messageKey: "EmailNotFound",
                                    severity: "Error",
                                    buttonKey: "SignUp",
                                    buttonClicked: () => { toSignUp(); },
                                });
                            }
                            else {
                                PubSub.get().publishSnack({ messageKey: errorToCode(response), severity: "Error", data: response });
                            }
                            helpers.setSubmitting(false);
                        },
                    });
                }, validationSchema: emailLogInFormValidation, children: (formik) => _jsxs(BaseForm, { dirty: formik.dirty, isLoading: loading, style: {
                        display: "block",
                        ...formPaper,
                    }, children: [_jsxs(Grid, { container: true, spacing: 2, children: [_jsx(Grid, { item: true, xs: 12, children: _jsx(Field, { fullWidth: true, autoComplete: "email", name: "email", label: t("Email", { count: 1 }), as: TextField }) }), _jsx(Grid, { item: true, xs: 12, children: _jsx(PasswordTextField, { fullWidth: true, name: "password", autoComplete: "current-password" }) })] }), _jsx(Button, { fullWidth: true, disabled: loading, type: "submit", color: "secondary", sx: { ...formSubmit }, children: t("LogIn") }), _jsxs(Grid, { container: true, spacing: 2, children: [_jsx(Grid, { item: true, xs: 6, children: _jsx(Link, { onClick: toForgotPassword, children: _jsx(Typography, { sx: {
                                                ...clickSize,
                                                ...formNavLink,
                                            }, children: t("ForgotPassword") }) }) }), _jsx(Grid, { item: true, xs: 6, children: _jsx(Link, { onClick: toSignUp, children: _jsx(Typography, { sx: {
                                                ...clickSize,
                                                ...formNavLink,
                                                flexDirection: "row-reverse",
                                            }, children: t("DontHaveAccountSignUp") }) }) })] })] }) })] }));
};
//# sourceMappingURL=LogInForm.js.map
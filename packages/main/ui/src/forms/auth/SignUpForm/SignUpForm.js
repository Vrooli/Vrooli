import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { BUSINESS_NAME, LINKS } from "@local/consts";
import { emailSignUpFormValidation } from "@local/validation";
import { Button, Checkbox, FormControlLabel, Grid, Link, TextField, Typography, useTheme } from "@mui/material";
import { Field, Formik } from "formik";
import { useTranslation } from "react-i18next";
import { authEmailSignUp } from "../../../api/generated/endpoints/auth_emailSignUp";
import { useCustomMutation } from "../../../api/hooks";
import { hasErrorCode, mutationWrapper } from "../../../api/utils";
import { PasswordTextField } from "../../../components/inputs/PasswordTextField/PasswordTextField";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { clickSize } from "../../../styles";
import { Forms } from "../../../utils/consts";
import { PubSub } from "../../../utils/pubsub";
import { setupPush } from "../../../utils/push";
import { useLocation } from "../../../utils/route";
import { BaseForm } from "../../BaseForm/BaseForm";
import { formNavLink, formPaper, formSubmit } from "../../styles";
export const SignUpForm = ({ onClose, onFormChange = () => { }, }) => {
    const theme = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [emailSignUp, { loading }] = useCustomMutation(authEmailSignUp);
    const toLogIn = () => onFormChange(Forms.LogIn);
    const toForgotPassword = () => onFormChange(Forms.ForgotPassword);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: "dialog", onClose: onClose, titleData: {
                    titleKey: "SignUp",
                } }), _jsx(Formik, { initialValues: {
                    marketingEmails: true,
                    name: "",
                    email: "",
                    password: "",
                    confirmPassword: "",
                }, onSubmit: (values, helpers) => {
                    mutationWrapper({
                        mutation: emailSignUp,
                        input: {
                            ...values,
                            marketingEmails: Boolean(values.marketingEmails),
                            theme: theme.palette.mode ?? "light",
                        },
                        onSuccess: (data) => {
                            PubSub.get().publishSession(data);
                            PubSub.get().publishAlertDialog({
                                messageKey: "WelcomeVerifyEmail",
                                messageVariables: { appName: BUSINESS_NAME },
                                buttons: [{
                                        labelKey: "Ok", onClick: () => {
                                            setLocation(LINKS.Welcome);
                                            setupPush();
                                        },
                                    }],
                            });
                        },
                        onError: (response) => {
                            if (hasErrorCode(response, "EmailInUse")) {
                                PubSub.get().publishAlertDialog({
                                    messageKey: "EmailInUseWrongPassword",
                                    buttons: [
                                        { labelKey: "Yes", onClick: () => onFormChange(Forms.ForgotPassword) },
                                        { labelKey: "No" },
                                    ],
                                });
                            }
                            helpers.setSubmitting(false);
                        },
                    });
                }, validationSchema: emailSignUpFormValidation, children: (formik) => _jsxs(BaseForm, { dirty: formik.dirty, isLoading: loading, style: {
                        display: "block",
                        ...formPaper,
                    }, children: [_jsxs(Grid, { container: true, spacing: 2, children: [_jsx(Grid, { item: true, xs: 12, children: _jsx(Field, { fullWidth: true, autoComplete: "name", name: "name", label: t("Name"), as: TextField }) }), _jsx(Grid, { item: true, xs: 12, children: _jsx(Field, { fullWidth: true, autoComplete: "email", name: "email", label: t("Email", { count: 1 }), as: TextField }) }), _jsx(Grid, { item: true, xs: 12, children: _jsx(PasswordTextField, { fullWidth: true, name: "password", autoComplete: "new-password", label: t("Password") }) }), _jsx(Grid, { item: true, xs: 12, children: _jsx(PasswordTextField, { fullWidth: true, name: "confirmPassword", autoComplete: "new-password", label: t("PasswordConfirm") }) }), _jsx(Grid, { item: true, xs: 12, children: _jsx(FormControlLabel, { control: _jsx(Checkbox, { id: "marketingEmails", name: "marketingEmails", color: "secondary", checked: Boolean(formik.values.marketingEmails), onBlur: formik.handleBlur, onChange: formik.handleChange }), label: "I want to receive marketing promotions and updates via email." }) })] }), _jsx(Button, { fullWidth: true, disabled: loading, type: "submit", color: "secondary", sx: { ...formSubmit }, children: t("SignUp") }), _jsxs(Grid, { container: true, spacing: 2, children: [_jsx(Grid, { item: true, xs: 6, children: _jsx(Link, { onClick: toLogIn, children: _jsx(Typography, { sx: {
                                                ...clickSize,
                                                ...formNavLink,
                                            }, children: t("AlreadyHaveAccountLogIn") }) }) }), _jsx(Grid, { item: true, xs: 6, children: _jsx(Link, { onClick: toForgotPassword, children: _jsx(Typography, { sx: {
                                                ...clickSize,
                                                ...formNavLink,
                                                flexDirection: "row-reverse",
                                            }, children: t("ForgotPassword") }) }) })] })] }) })] }));
};
//# sourceMappingURL=SignUpForm.js.map
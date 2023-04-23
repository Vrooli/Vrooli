import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { emailRequestPasswordChangeSchema } from "@local/validation";
import { Button, Grid, Link, TextField, Typography } from "@mui/material";
import { Field, Formik } from "formik";
import { useTranslation } from "react-i18next";
import { authEmailRequestPasswordChange } from "../../../api/generated/endpoints/auth_emailRequestPasswordChange";
import { useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { clickSize } from "../../../styles";
import { Forms } from "../../../utils/consts";
import { useLocation } from "../../../utils/route";
import { BaseForm } from "../../BaseForm/BaseForm";
import { formNavLink, formPaper, formSubmit } from "../../styles";
export const ForgotPasswordForm = ({ onClose, onFormChange = () => { }, }) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [emailRequestPasswordChange, { loading }] = useCustomMutation(authEmailRequestPasswordChange);
    const toSignUp = () => onFormChange(Forms.SignUp);
    const toLogIn = () => onFormChange(Forms.LogIn);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: "dialog", onClose: onClose, titleData: {
                    titleKey: "ForgotPassword",
                } }), _jsx(Formik, { initialValues: {
                    email: "",
                }, onSubmit: (values, helpers) => {
                    mutationWrapper({
                        mutation: emailRequestPasswordChange,
                        input: { ...values },
                        successCondition: (data) => data.success === true,
                        onSuccess: () => setLocation(LINKS.Home),
                        onError: () => { helpers.setSubmitting(false); },
                        successMessage: () => ({ key: "RequestSentCheckEmail" }),
                    });
                }, validationSchema: emailRequestPasswordChangeSchema, children: (formik) => _jsxs(BaseForm, { dirty: formik.dirty, isLoading: loading, style: {
                        display: "block",
                        ...formPaper,
                    }, children: [_jsx(Grid, { container: true, spacing: 2, children: _jsx(Grid, { item: true, xs: 12, children: _jsx(Field, { fullWidth: true, autoComplete: "email", name: "email", label: t("Email", { count: 1 }), as: TextField }) }) }), _jsx(Button, { fullWidth: true, disabled: loading, type: "submit", color: "secondary", sx: { ...formSubmit }, children: t("Submit") }), _jsxs(Grid, { container: true, spacing: 2, children: [_jsx(Grid, { item: true, xs: 6, children: _jsx(Link, { onClick: toLogIn, children: _jsx(Typography, { sx: {
                                                ...clickSize,
                                                ...formNavLink,
                                            }, children: t("RememberLogBackIn") }) }) }), _jsx(Grid, { item: true, xs: 6, children: _jsx(Link, { onClick: toSignUp, children: _jsx(Typography, { sx: {
                                                ...clickSize,
                                                ...formNavLink,
                                                flexDirection: "row-reverse",
                                            }, children: t("DontHaveAccountSignUp") }) }) })] })] }) })] }));
};
//# sourceMappingURL=ForgotPasswordForm.js.map
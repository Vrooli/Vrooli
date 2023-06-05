import { emailLogInFormValidation, EmailLogInInput, endpointPostAuthEmailLogin, LINKS, parseSearchParams, Session, useLocation } from "@local/shared";
import { Button, Grid, Link, TextField, Typography } from "@mui/material";
import { errorToCode, fetchLazyWrapper, hasErrorCode } from "api";
import { PasswordTextField } from "components/inputs/PasswordTextField/PasswordTextField";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Field, Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { CSSProperties, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { clickSize } from "styles";
import { Forms } from "utils/consts";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { PubSub } from "utils/pubsub";
import { formNavLink, formPaper, formSubmit } from "../../styles";
import { LogInFormProps } from "../../types";

export const LogInForm = ({
    onClose,
    onFormChange = () => { },
}: LogInFormProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { redirect, verificationCode } = useMemo(() => {
        const params = parseSearchParams();
        return {
            redirect: typeof params.redirect === "string" ? params.redirect : undefined,
            verificationCode: typeof params.code === "string" ? params.code : undefined,
        };
    }, []);

    const [emailLogIn, { loading }] = useLazyFetch<EmailLogInInput, Session>(endpointPostAuthEmailLogin);

    const toForgotPassword = () => onFormChange(Forms.ForgotPassword);
    const toSignUp = () => onFormChange(Forms.SignUp);

    return (
        <>
            <TopBar
                display="dialog"
                onClose={onClose}
                titleData={{
                    titleKey: "LogIn",
                }}
            />
            <Formik
                initialValues={{
                    email: "",
                    password: "",
                }}
                onSubmit={(values, helpers) => {
                    fetchLazyWrapper<EmailLogInInput, Session>({
                        fetch: emailLogIn,
                        inputs: { ...values, verificationCode },
                        successCondition: (data) => data !== null,
                        onSuccess: (data) => {
                            if (verificationCode) PubSub.get().publishSnack({ messageKey: "EmailVerified", severity: "Success" });
                            PubSub.get().publishSession(data);
                            setLocation(redirect ?? LINKS.Home);
                        },
                        showDefaultErrorSnack: false,
                        onError: (response) => {
                            // Custom dialog for changing password
                            if (hasErrorCode(response, "MustResetPassword")) {
                                PubSub.get().publishAlertDialog({
                                    messageKey: "ChangePasswordBeforeLogin",
                                    buttons: [
                                        { labelKey: "Ok", onClick: () => { setLocation(redirect ?? LINKS.Home); } },
                                    ],
                                });
                            }
                            // Custom snack for invalid email, that has sign up link
                            else if (hasErrorCode(response, "EmailNotFound")) {
                                PubSub.get().publishSnack({
                                    messageKey: "EmailNotFound",
                                    severity: "Error",
                                    buttonKey: "SignUp",
                                    buttonClicked: () => { toSignUp(); },
                                });
                            } else {
                                PubSub.get().publishSnack({ messageKey: errorToCode(response), severity: "Error", data: response });
                            }
                            helpers.setSubmitting(false);
                        },
                    });
                }}
                validationSchema={emailLogInFormValidation}
            >
                {(formik) => <BaseForm
                    dirty={formik.dirty}
                    isLoading={loading}
                    style={{
                        display: "block",
                        ...formPaper,
                    }}
                >
                    <Grid container spacing={2}>
                        <Grid item xs={12}>
                            <Field
                                fullWidth
                                autoComplete="email"
                                name="email"
                                label={t("Email", { count: 1 })}
                                as={TextField}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <PasswordTextField
                                fullWidth
                                name="password"
                                autoComplete="current-password"
                            />
                        </Grid>
                    </Grid>
                    <Button
                        fullWidth
                        disabled={loading}
                        type="submit"
                        color="secondary"
                        sx={{ ...formSubmit }}
                    >
                        {t("LogIn")}
                    </Button>
                    <Grid container spacing={2}>
                        <Grid item xs={6}>
                            <Link onClick={toForgotPassword}>
                                <Typography
                                    sx={{
                                        ...clickSize,
                                        ...formNavLink,
                                    } as CSSProperties}
                                >
                                    {t("ForgotPassword")}
                                </Typography>
                            </Link>
                        </Grid>
                        <Grid item xs={6}>
                            <Link onClick={toSignUp}>
                                <Typography
                                    sx={{
                                        ...clickSize,
                                        ...formNavLink,
                                        flexDirection: "row-reverse",
                                    } as CSSProperties}
                                >
                                    {t("DontHaveAccountSignUp")}
                                </Typography>
                            </Link>
                        </Grid>
                    </Grid>
                </BaseForm>}
            </Formik>
        </>
    );
};

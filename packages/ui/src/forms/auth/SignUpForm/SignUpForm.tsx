import { BUSINESS_NAME, emailSignUpFormValidation, EmailSignUpInput, endpointPostAuthEmailSignup, LINKS, Session } from "@local/shared";
import { Button, Checkbox, FormControlLabel, Grid, Link, TextField, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper, hasErrorCode } from "api";
import { PasswordTextField } from "components/inputs/PasswordTextField/PasswordTextField";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Field, Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { clickSize } from "styles";
import { Forms } from "utils/consts";
import { PubSub } from "utils/pubsub";
import { setupPush } from "utils/push";
import { formNavLink, formPaper, formSubmit } from "../../styles";
import { SignUpFormProps } from "../../types";

export const SignUpForm = ({
    onClose,
    onFormChange = () => { },
}: SignUpFormProps) => {
    const theme = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [emailSignUp, { loading }] = useLazyFetch<EmailSignUpInput, Session>(endpointPostAuthEmailSignup);

    const toLogIn = () => onFormChange(Forms.LogIn);
    const toForgotPassword = () => onFormChange(Forms.ForgotPassword);

    return (
        <>
            <TopBar
                display="dialog"
                onClose={onClose}
                title={t("SignUp")}
            />
            <Formik
                initialValues={{
                    marketingEmails: true,
                    name: "",
                    email: "",
                    password: "",
                    confirmPassword: "",
                }}
                onSubmit={(values, helpers) => {
                    fetchLazyWrapper<EmailSignUpInput, Session>({
                        fetch: emailSignUp,
                        inputs: {
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
                                        setLocation(LINKS.Home);
                                        // Set up push notifications
                                        setupPush();
                                        // Start the tutorial
                                        PubSub.get().publishTutorial();
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
                }}
                validationSchema={emailSignUpFormValidation}
            >
                {(formik) => <BaseForm
                    dirty={formik.dirty}
                    display={"dialog"}
                    isLoading={loading}
                    style={{
                        ...formPaper,
                        paddingBottom: "unset",
                    }}
                >
                    <Grid container spacing={2}>
                        <Grid item xs={12}>
                            <Field
                                fullWidth
                                autoComplete="name"
                                name="name"
                                label={t("Name")}
                                as={TextField}
                            />
                        </Grid>
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
                                autoComplete="new-password"
                                label={t("Password")}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <PasswordTextField
                                fullWidth
                                name="confirmPassword"
                                autoComplete="new-password"
                                label={t("PasswordConfirm")}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <FormControlLabel
                                control={
                                    <Checkbox
                                        id="marketingEmails"
                                        name="marketingEmails"
                                        color="secondary"
                                        checked={Boolean(formik.values.marketingEmails)}
                                        onBlur={formik.handleBlur}
                                        onChange={formik.handleChange}
                                    />
                                }
                                label="I want to receive marketing promotions and updates via email."
                            />
                        </Grid>
                    </Grid>
                    <Button
                        fullWidth
                        disabled={loading}
                        type="submit"
                        color="secondary"
                        variant="contained"
                        sx={{ ...formSubmit }}
                    >
                        {t("SignUp")}
                    </Button>
                    <Grid container spacing={2}>
                        <Grid item xs={6}>
                            <Link onClick={toLogIn}>
                                <Typography
                                    sx={{
                                        ...clickSize,
                                        ...formNavLink,
                                    }}
                                >
                                    {t("AlreadyHaveAccountLogIn")}
                                </Typography>
                            </Link>
                        </Grid>
                        <Grid item xs={6}>
                            <Link onClick={toForgotPassword}>
                                <Typography
                                    sx={{
                                        ...clickSize,
                                        ...formNavLink,
                                        flexDirection: "row-reverse",
                                    }}
                                >
                                    {t("ForgotPassword")}
                                </Typography>
                            </Link>
                        </Grid>
                    </Grid>
                </BaseForm>}
            </Formik>
        </>
    );
};

import { emailLogInFormValidation, EmailLogInInput, endpointPostAuthEmailLogin, LINKS, Session } from "@local/shared";
import { Box, Button, Grid, InputAdornment, Link, Typography, useTheme } from "@mui/material";
import { errorToMessage, fetchLazyWrapper, hasErrorCode } from "api";
import { PasswordTextInput } from "components/inputs/PasswordTextInput/PasswordTextInput";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { formNavLink, formPaper, formSubmit } from "forms/styles";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useReactSearch } from "hooks/useReactSearch";
import { EmailIcon } from "icons";
import { useContext, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { clickSize } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { removeCookie } from "utils/cookies";
import { PubSub } from "utils/pubsub";
import { LoginViewProps } from "views/types";

interface LoginFormProps {
    onClose?: () => unknown;
}

const LoginForm = ({
    onClose,
}: LoginFormProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    const search = useReactSearch();
    const { redirect, verificationCode } = useMemo(() => ({
        redirect: typeof search.redirect === "string" ? search.redirect : undefined,
        verificationCode: typeof search.verificationCode === "string" ? search.verificationCode : undefined,
    }), [search]);

    const [emailLogIn, { loading }] = useLazyFetch<EmailLogInInput, Session>(endpointPostAuthEmailLogin);

    /**
     * If verification code supplied
     */
    useEffect(() => {
        if (verificationCode) {
            // If still logged in, call emailLogIn right away
            if (userId) {
                fetchLazyWrapper<EmailLogInInput, Session>({
                    fetch: emailLogIn,
                    inputs: { verificationCode },
                    onSuccess: (data) => {
                        PubSub.get().publish("snack", { messageKey: "EmailVerified", severity: "Success" });
                        PubSub.get().publish("session", data);
                        setLocation(redirect ?? LINKS.Home);
                    },
                    onError: (response) => {
                        if (hasErrorCode(response, "MustResetPassword")) {
                            PubSub.get().publish("alertDialog", {
                                messageKey: "ChangePasswordBeforeLogin",
                                buttons: [
                                    { labelKey: "Ok", onClick: () => { setLocation(redirect ?? LINKS.Home); } },
                                ],
                            });
                        }
                    },
                });
            }
        }
    }, [emailLogIn, verificationCode, redirect, setLocation, userId]);

    return (
        <>
            <TopBar
                display="dialog"
                onClose={onClose}
                title={t("LogIn")}
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
                            removeCookie("FormData"); // Clear old form data cache
                            if (verificationCode) PubSub.get().publish("snack", { messageKey: "EmailVerified", severity: "Success" });
                            PubSub.get().publish("session", data);
                            setLocation(redirect ?? LINKS.Home);
                        },
                        showDefaultErrorSnack: false,
                        onError: (response) => {
                            // Custom dialog for changing password
                            if (hasErrorCode(response, "MustResetPassword")) {
                                PubSub.get().publish("alertDialog", {
                                    messageKey: "ChangePasswordBeforeLogin",
                                    buttons: [
                                        { labelKey: "Ok", onClick: () => { setLocation(redirect ?? LINKS.Home); } },
                                    ],
                                });
                            }
                            // Custom snack for invalid email, that has sign up link
                            else if (hasErrorCode(response, "EmailNotFound")) {
                                PubSub.get().publish("snack", {
                                    messageKey: "EmailNotFound",
                                    severity: "Error",
                                    buttonKey: "SignUp",
                                    buttonClicked: () => { setLocation(LINKS.Signup); },
                                });
                            } else {
                                PubSub.get().publish("snack", { message: errorToMessage(response, ["en"]), severity: "Error", data: response });
                            }
                            helpers.setSubmitting(false);
                        },
                    });
                }}
                validationSchema={emailLogInFormValidation}
            >
                {(formik) => <BaseForm
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
                                autoComplete="email"
                                autoFocus
                                name="email"
                                label={t("Email", { count: 1 })}
                                placeholder={t("EmailPlaceholder")}
                                as={TextInput}
                                InputProps={{
                                    startAdornment: (
                                        <InputAdornment position="start">
                                            <EmailIcon />
                                        </InputAdornment>
                                    ),
                                }}
                                helperText={formik.touched.email && formik.errors.email}
                                error={formik.touched.email && Boolean(formik.errors.email)}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <PasswordTextInput
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
                        variant='contained'
                        sx={{ ...formSubmit }}
                    >
                        {t("LogIn")}
                    </Button>
                    <Box sx={{
                        display: "flex",
                        flexDirection: "row",
                        justifyContent: "space-between",
                        alignItems: "center",
                    }}>
                        <Link href={LINKS.ForgotPassword}>
                            <Typography
                                sx={{
                                    ...clickSize,
                                    ...formNavLink,
                                }}
                            >
                                {t("ForgotPassword")}
                            </Typography>
                        </Link>
                        <Link href={LINKS.Signup}>
                            <Typography
                                sx={{
                                    ...clickSize,
                                    ...formNavLink,
                                    flexDirection: "row-reverse",
                                }}
                            >
                                {t("SignUp")}
                            </Typography>
                        </Link>
                    </Box>
                </BaseForm>}
            </Formik>
        </>
    );
};

export const LoginView = ({
    display,
    onClose,
}: LoginViewProps) => {
    const { palette } = useTheme();

    return (
        <Box sx={{ maxHeight: "100vh", overflow: "hidden" }}>
            <TopBar
                display={display}
                onClose={onClose}
                hideTitleOnDesktop
            />
            <Box
                sx={{
                    position: "absolute",
                    top: "50%",
                    left: "50%",
                    transform: "translateX(-50%) translateY(-50%)",
                    width: "min(700px, 100%)",
                    background: palette.background.paper,
                    borderRadius: { xs: 0, sm: 2 },
                    overflow: "overlay",
                }}
            >
                <LoginForm onClose={onClose} />
            </Box>
        </Box>
    );
};

import { Box, Button } from "@mui/material";
import { LINKS, emailLogInFormValidation, endpointsAuth, type EmailLogInInput, type Session } from "@vrooli/shared";
import { Field, Formik, type FormikHelpers } from "formik";
import { useCallback, useContext, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { ServerResponseParser } from "../../api/responseParser.js";
import { SocketService } from "../../api/socket.js";
import { BreadcrumbsBase } from "../../components/breadcrumbs/BreadcrumbsBase.js";
import { PasswordTextInput } from "../../components/inputs/PasswordTextInput/PasswordTextInput.js";
import { TextInput } from "../../components/inputs/TextInput/TextInput.js";
import { TopBar } from "../../components/navigation/TopBar.js";
import { SessionContext } from "../../contexts/session.js";
import { InnerForm } from "../../forms/BaseForm/BaseForm.js";
import { formSubmit } from "../../forms/styles.js";
import { useLazyFetch } from "../../hooks/useFetch.js";
import { useReactSearch } from "../../hooks/useReactSearch.js";
import { IconFavicon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { CenteredContentPage, pagePaddingBottom } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { removeCookie } from "../../utils/localStorage.js";
import { PubSub } from "../../utils/pubsub.js";
import { type LoginViewProps } from "../../views/types.js";
import { AuthContainer, AuthFormContainer, FormSection, OAUTH_PROVIDERS_INFO, OAuthButton, OAuthContainer, OAuthSection, OrDivider, OuterAuthFormContainer, baseFormStyle, breadcrumbsStyle, emailStartAdornment, oAuthSpanStyle } from "./authStyles.js";

type FormValues = typeof initialValues;

const initialValues = {
    email: "",
    password: "",
};

const BREADCRUMB_PATHS = [
    {
        textKey: "ForgotPassword",
        link: LINKS.ForgotPassword,
    },
    {
        textKey: "SignUp",
        link: LINKS.Signup,
    },
] as const;

function LoginForm() {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    const search = useReactSearch();
    const { redirect, verificationCode } = useMemo(function parseUrlSearch() {
        return {
            redirect: typeof search.redirect === "string" ? search.redirect : undefined,
            verificationCode: typeof search.verificationCode === "string" ? search.verificationCode : undefined,
        } as const;
    }, [search]);

    // If already logged in and there's a redirect, show a snack to let user go to the redirect. 
    // This is particularly useful during development when the server is restarted.
    useEffect(function checkAlreadyLoggedIn() {
        if (userId && redirect) {
            PubSub.get().publish("snack", {
                message: "Logged in. Go to redirect?",
                severity: "Info",
                buttonKey: "GoBack",
                buttonClicked: () => {
                    setLocation(redirect, { replace: true });
                },
            });
        }
    }, [userId, redirect, setLocation]);

    const [emailLogIn, { loading }] = useLazyFetch<EmailLogInInput, Session>(endpointsAuth.emailLogin);

    useEffect(function handleVerificationCodeEffect() {
        if (!verificationCode) return;
        if (!userId) {
            console.warn("[LoginView] No user ID found while handling verification code. Will not attempt to verify email until user logs in.");
            return;
        }
        fetchLazyWrapper<EmailLogInInput, Session>({
            fetch: emailLogIn,
            inputs: { verificationCode },
            onSuccess: (data) => {
                SocketService.get().disconnect();
                removeCookie("FormData"); // Clear old form data cache
                PubSub.get().publish("snack", { messageKey: "EmailVerified", severity: "Success" });
                PubSub.get().publish("session", data);
                localStorage.setItem("isLoggedIn", "true");
                SocketService.get().connect();
                setLocation(redirect ?? LINKS.Home);
            },
            onError: (response) => {
                if (ServerResponseParser.hasErrorCode(response, "MustResetPassword")) {
                    PubSub.get().publish("alertDialog", {
                        messageKey: "ChangePasswordBeforeLogin",
                        buttons: [
                            { labelKey: "Ok", onClick: () => { setLocation(redirect ?? LINKS.Home); } },
                        ],
                    });
                }
            },
        });
    }, [emailLogIn, verificationCode, redirect, setLocation, userId]);

    const onSubmit = useCallback(function onSubmitCallback(values: FormValues, helpers: FormikHelpers<FormValues>) {
        fetchLazyWrapper<EmailLogInInput, Session>({
            fetch: emailLogIn,
            inputs: { ...values, verificationCode },
            successCondition: (data) => data !== null,
            onSuccess: (data) => {
                SocketService.get().disconnect();
                removeCookie("FormData"); // Clear old form data cache
                if (verificationCode) PubSub.get().publish("snack", { messageKey: "EmailVerified", severity: "Success" });
                PubSub.get().publish("session", data);
                localStorage.setItem("isLoggedIn", "true");
                SocketService.get().connect();
                setLocation(redirect ?? LINKS.Home);
            },
            showDefaultErrorSnack: false,
            onError: (response) => {
                // Custom dialog for changing password
                if (ServerResponseParser.hasErrorCode(response, "MustResetPassword")) {
                    PubSub.get().publish("alertDialog", {
                        messageKey: "ChangePasswordBeforeLogin",
                        buttons: [
                            { labelKey: "Ok", onClick: () => { setLocation(redirect ?? LINKS.Home); } },
                        ],
                    });
                }
                // Custom snack for signing up
                else if (ServerResponseParser.hasErrorCode(response, "InvalidCredentials")) {
                    PubSub.get().publish("snack", {
                        messageKey: "InvalidCredentials",
                        severity: "Error",
                        buttonKey: "SignUp",
                        buttonClicked: () => { setLocation(LINKS.Signup); },
                    });
                } else {
                    ServerResponseParser.displayErrors(response.errors);
                }
                helpers.setSubmitting(false);
            },
        });
    }, [emailLogIn, redirect, setLocation, verificationCode]);

    const breadcrumbPaths = useMemo(() => BREADCRUMB_PATHS.map(path => ({
        text: t(path.textKey),
        link: path.link,
    })), [t]);

    return (
        <OuterAuthFormContainer>
            <AuthContainer>
                <AuthFormContainer>
                    <Formik
                        initialValues={initialValues}
                        onSubmit={onSubmit}
                        validationSchema={emailLogInFormValidation}
                    >
                        {(formik) => (
                            <InnerForm
                                display="Dialog"
                                isLoading={loading}
                                style={baseFormStyle}
                            >
                                <FormSection>
                                    <Field
                                        fullWidth
                                        autoComplete="email"
                                        name="email"
                                        label={t("Email", { count: 1 })}
                                        placeholder={t("EmailPlaceholder")}
                                        as={TextInput}
                                        InputProps={emailStartAdornment}
                                        helperText={formik.touched.email && formik.errors.email}
                                        error={formik.touched.email && Boolean(formik.errors.email)}
                                    />
                                    <Box mt={2}>
                                        <PasswordTextInput
                                            fullWidth
                                            name="password"
                                            autoComplete="current-password"
                                        />
                                    </Box>
                                    <Box mt={3} display="flex" flexDirection="column">
                                        <Button
                                            fullWidth
                                            disabled={loading}
                                            type="submit"
                                            color="secondary"
                                            variant="contained"
                                            sx={formSubmit}
                                        >
                                            {t("LogIn")}
                                        </Button>
                                        <BreadcrumbsBase
                                            paths={breadcrumbPaths}
                                            separator="•"
                                            sx={breadcrumbsStyle}
                                        />
                                    </Box>
                                </FormSection>
                            </InnerForm>
                        )}
                    </Formik>
                </AuthFormContainer>
                <OAuthContainer>
                    <OrDivider>or</OrDivider>
                    <OAuthSection>
                        {OAUTH_PROVIDERS_INFO.map(({ name, url, site, style }) => {
                            function handleOAuthClick() {
                                window.location.href = url;
                            }

                            return (
                                <OAuthButton
                                    key={name}
                                    onClick={handleOAuthClick}
                                    variant="contained"
                                    fullWidth
                                    startIcon={<IconFavicon href={site} />}
                                    providerStyle={style}
                                >
                                    <span style={oAuthSpanStyle}>
                                        {t("SignInWith", { provider: name })}
                                    </span>
                                </OAuthButton>
                            );
                        })}
                    </OAuthSection>
                </OAuthContainer>
            </AuthContainer>
        </OuterAuthFormContainer>
    );
}

export function LoginView({
    display,
}: LoginViewProps) {
    return (
        <CenteredContentPage>
            <TopBar
                display={display}
            />
            <Box
                display="flex"
                flexDirection="column"
                gap={1}
                overflow="auto"
                padding={2}
                paddingBottom={pagePaddingBottom}
            >
                <LoginForm />
            </Box>
        </CenteredContentPage>
    );
}

import { emailLogInFormValidation, EmailLogInInput, endpointsAuth, getOAuthInitRoute, LINKS, OAUTH_PROVIDERS, Session } from "@local/shared";
import { Box, Button, Divider, InputAdornment, styled } from "@mui/material";
import { Field, Formik, FormikHelpers } from "formik";
import { useCallback, useContext, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { ServerResponseParser } from "../../api/responseParser.js";
import { SocketService } from "../../api/socket.js";
import AppleIcon from "../../assets/img/apple.svg";
import FacebookIcon from "../../assets/img/facebook.svg";
import GitHubIcon from "../../assets/img/github.svg";
import GoogleIcon from "../../assets/img/google.svg";
import XIcon from "../../assets/img/x.svg";
import { BreadcrumbsBase } from "../../components/breadcrumbs/BreadcrumbsBase/BreadcrumbsBase.js";
import { PasswordTextInput } from "../../components/inputs/PasswordTextInput/PasswordTextInput.js";
import { TextInput } from "../../components/inputs/TextInput/TextInput.js";
import { TopBar } from "../../components/navigation/TopBar/TopBar.js";
import { SessionContext } from "../../contexts.js";
import { InnerForm } from "../../forms/BaseForm/BaseForm.js";
import { formPaper, formSubmit } from "../../forms/styles.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { useReactSearch } from "../../hooks/useReactSearch.js";
import { EmailIcon } from "../../icons/common.js";
import { CenteredContentPage, CenteredContentPageWrap, CenteredContentPaper, FormContainer, FormSection } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { removeCookie } from "../../utils/localStorage.js";
import { PubSub } from "../../utils/pubsub.js";
import { LoginViewProps } from "../../views/types.js";

interface LoginFormProps {
    onClose?: () => unknown;
}
type FormValues = typeof initialValues;

// Ordered by appearance
const OAUTH_PROVIDERS_INFO = [
    {
        name: "X",
        url: getOAuthInitRoute(OAUTH_PROVIDERS.X),
        logo: XIcon,
    },
    {
        name: "Google",
        url: getOAuthInitRoute(OAUTH_PROVIDERS.Google),
        logo: GoogleIcon,
    },
    {
        name: "Apple",
        url: getOAuthInitRoute(OAUTH_PROVIDERS.Apple),
        logo: AppleIcon,
    },
    {
        name: "GitHub",
        url: getOAuthInitRoute(OAUTH_PROVIDERS.GitHub),
        logo: GitHubIcon,
    },
    {
        name: "Facebook",
        url: getOAuthInitRoute(OAUTH_PROVIDERS.Facebook),
        logo: FacebookIcon,
    },
    // Add more providers as needed
] as const;

const initialValues = {
    email: "",
    password: "",
};

const OrDivider = styled(Divider)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
    width: "100%",
}));
const OAuthButton = styled(Button)(({ theme }) => ({
    background: "white",
    color: "black",
    borderRadius: "16px",
    textTransform: "none",
    "& .MuiButton-icon": {
        marginRight: "auto",
    },
    "&:hover": {
        background: "#dcdcdc",
    },
}));

const baseFormStyle = {
    ...formPaper,
    paddingBottom: "unset",
    margin: "0px",
} as const;
const breadcrumbsStyle = {
    margin: "auto",
} as const;
const oAuthIconStyle = {
    height: "24px",
} as const;
const oAuthSpanStyle = {
    marginRight: "auto",
} as const;

const emailStartAdornment = {
    startAdornment: (
        <InputAdornment position="start">
            <EmailIcon />
        </InputAdornment>
    ),
};

function LoginForm({
    onClose,
}: LoginFormProps) {
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
            }
        }
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

    const breadcrumbPaths = [
        {
            text: t("ForgotPassword"),
            link: LINKS.ForgotPassword,
        },
        {
            text: t("SignUp"),
            link: LINKS.Signup,
        },
    ] as const;

    return (
        <Box>
            <TopBar
                display="dialog"
                onClose={onClose}
                title={t("LogIn")}
            />
            <Formik
                initialValues={initialValues}
                onSubmit={onSubmit}
                validationSchema={emailLogInFormValidation}
            >
                {(formik) => <InnerForm
                    display={"dialog"}
                    isLoading={loading}
                    style={baseFormStyle}
                >
                    <FormContainer width="unset" maxWidth="unset">
                        <FormSection variant="transparent">
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
                            <PasswordTextInput
                                fullWidth
                                name="password"
                                autoComplete="current-password"
                            />
                        </FormSection>
                    </FormContainer>
                    <Box width="100%" display="flex" flexDirection="column" p={2} pt={0}>
                        <Button
                            fullWidth
                            disabled={loading}
                            type="submit"
                            color="secondary"
                            variant='contained'
                            sx={formSubmit}
                        >
                            {t("LogIn")}
                        </Button>
                        <BreadcrumbsBase
                            paths={breadcrumbPaths}
                            separator={"•"}
                            sx={breadcrumbsStyle}
                        />
                    </Box>
                    <OrDivider>or</OrDivider>
                    <Box display="flex" flexDirection="column" width="100%" maxWidth="400px" gap={1} p={2}>
                        {OAUTH_PROVIDERS_INFO.map((provider) => {
                            function handleOAuthClick() {
                                window.location.href = provider.url;
                            }

                            return (
                                <OAuthButton
                                    key={provider.name}
                                    onClick={handleOAuthClick}
                                    variant="contained"
                                    fullWidth
                                    startIcon={<img src={provider.logo} alt={`${provider.name} logo`} style={oAuthIconStyle} />}
                                >
                                    <span style={oAuthSpanStyle}>
                                        {t("SignInWith", { provider: provider.name })}
                                    </span>
                                </OAuthButton>
                            );
                        })}
                    </Box>
                </InnerForm>}
            </Formik>
        </Box>
    );
}

export function LoginView({
    display,
    onClose,
}: LoginViewProps) {
    return (
        <CenteredContentPage>
            <TopBar
                display={display}
                onClose={onClose}
                titleBehaviorDesktop="ShowBelow"
            />
            <CenteredContentPageWrap>
                <CenteredContentPaper maxWidth={600}>
                    <LoginForm onClose={onClose} />
                </CenteredContentPaper>
            </CenteredContentPageWrap>
        </CenteredContentPage>
    );
}

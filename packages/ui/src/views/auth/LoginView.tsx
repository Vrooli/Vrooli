import { emailLogInFormValidation, EmailLogInInput, endpointsAuth, getOAuthInitRoute, LINKS, OAUTH_PROVIDERS, Session } from "@local/shared";
import { Box, Button, Divider, InputAdornment, keyframes, styled } from "@mui/material";
import { Field, Formik, FormikHelpers } from "formik";
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
import { formPaper, formSubmit } from "../../forms/styles.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { useReactSearch } from "../../hooks/useReactSearch.js";
import { IconCommon, IconFavicon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { CenteredContentPage, CenteredContentPageWrap, pagePaddingBottom } from "../../styles.js";
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
        site: "https://x.com",
        style: {
            background: "#000000",
            color: "#ffffff",
            hoverBackground: "#14171a",
            border: "none",
        },
    },
    {
        name: "Google",
        url: getOAuthInitRoute(OAUTH_PROVIDERS.Google),
        site: "https://google.com",
        style: {
            background: "#4285f4",
            color: "#ffffff",
            border: "none",
            hoverBackground: "#357ABD",
        },
    },
    {
        name: "Apple",
        url: getOAuthInitRoute(OAUTH_PROVIDERS.Apple),
        site: "https://apple.com",
        style: {
            background: "#000000",
            color: "#ffffff",
            hoverBackground: "#333333",
            border: "none",
        },
    },
    {
        name: "GitHub",
        url: getOAuthInitRoute(OAUTH_PROVIDERS.GitHub),
        site: "https://github.com",
        style: {
            background: "#171a21",
            color: "#ffffff",
            hoverBackground: "#2a475e",
            border: "none",
        },
    },
    {
        name: "Facebook",
        url: getOAuthInitRoute(OAUTH_PROVIDERS.Facebook),
        site: "https://facebook.com",
        style: {
            background: "#1877f2",
            color: "#ffffff",
            hoverBackground: "#0a66c2",
            border: "none",
        },
    },
    // Add more providers as needed
] as const;

const initialValues = {
    email: "",
    password: "",
};

// eslint-disable-next-line no-magic-numbers
const ANIMATION_DURATION = 0.5;

const fadeIn = keyframes`
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
`;

const SPACING_LARGE = 4;
const SPACING_MEDIUM = 2;

const LoginContainer = styled(Box)(({ theme }) => ({
    width: "100%",
    maxWidth: "900px",
    background: theme.palette.background.paper,
    borderRadius: theme.spacing(2),
    boxShadow: "0 8px 32px rgba(0, 0, 0, 0.1)",
    animation: `${fadeIn} ${ANIMATION_DURATION}s ease-out`,
    overflow: "hidden",
    margin: theme.spacing(SPACING_LARGE, SPACING_MEDIUM),
    display: "flex",
    flexDirection: "row",
    [theme.breakpoints.down("md")]: {
        maxWidth: "450px",
        flexDirection: "column",
    },
    [theme.breakpoints.down("sm")]: {
        maxWidth: "100%",
        margin: theme.spacing(2),
    },
}));

const LoginFormContainer = styled(Box)(({ theme }) => ({
    flex: "1 1 60%",
    borderRight: `1px solid ${theme.palette.divider}`,
    display: "flex",
    flexDirection: "column",
    margin: "auto",
    [theme.breakpoints.down("md")]: {
        flex: "1 1 auto",
        borderRight: "none",
        borderBottom: `1px solid ${theme.palette.divider}`,
    },
}));

const OAuthContainer = styled(Box)(({ theme }) => ({
    flex: "1 1 40%",
    display: "flex",
    flexDirection: "column",
    justifyContent: "center",
    padding: theme.spacing(SPACING_MEDIUM),
    [theme.breakpoints.down("md")]: {
        flex: "1 1 auto",
        padding: theme.spacing(SPACING_MEDIUM),
    },
}));

const OrDivider = styled(Divider)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
    width: "100%",
    margin: theme.spacing(2, 0),
    "&::before, &::after": {
        borderColor: theme.palette.divider,
    },
    [theme.breakpoints.up("md")]: {
        display: "none",
    },
}));

interface OAuthButtonProps {
    providerStyle: typeof OAUTH_PROVIDERS_INFO[number]["style"];
}

const OAuthButton = styled(Button, {
    shouldForwardProp: (prop) => prop !== "providerStyle",
})<OAuthButtonProps>(({ theme, providerStyle }) => ({
    background: providerStyle.background,
    color: providerStyle.color,
    borderRadius: theme.spacing(2),
    textTransform: "none",
    border: providerStyle.border,
    padding: theme.spacing(2),
    transition: "all 0.2s ease-in-out",
    "& .MuiButton-icon": {
        marginRight: "auto",
    },
    "&:hover": {
        background: providerStyle.hoverBackground,
        transform: "translateY(-1px)",
        boxShadow: "0 4px 8px rgba(0, 0, 0, 0.1)",
    },
}));

const FormSection = styled(Box)(({ theme }) => ({
    padding: theme.spacing(2),
    [theme.breakpoints.down("sm")]: {
        padding: theme.spacing(2),
    },
}));

const OAuthSection = styled(Box)(({ theme }) => ({
    padding: theme.spacing(0, 2, 2),
    display: "flex",
    flexDirection: "column",
    gap: theme.spacing(2),
    [theme.breakpoints.down("sm")]: {
        padding: theme.spacing(0, 2, 2),
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
const oAuthSpanStyle = {
    marginRight: "auto",
} as const;

const emailStartAdornment = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon name="Email" />
        </InputAdornment>
    ),
};

const FormContainer = styled(Box)(() => ({
    width: "100%",
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    marginBottom: pagePaddingBottom,
}));

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

    const breadcrumbPaths = useMemo(() => BREADCRUMB_PATHS.map(path => ({
        text: t(path.textKey),
        link: path.link,
    })), [t]);

    return (
        <FormContainer>
            <LoginContainer>
                <LoginFormContainer>
                    <Formik
                        initialValues={initialValues}
                        onSubmit={onSubmit}
                        validationSchema={emailLogInFormValidation}
                    >
                        {(formik) => (
                            <InnerForm
                                display="dialog"
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
                </LoginFormContainer>
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
            </LoginContainer>
        </FormContainer>
    );
}

const contentWrapStyle = {
    minHeight: "100%",
    alignItems: "flex-start",
} as const;

export function LoginView({
    display,
}: LoginViewProps) {
    return (
        <CenteredContentPage>
            <TopBar
                display={display}
                titleBehaviorDesktop="ShowBelow"
            />
            <CenteredContentPageWrap sx={contentWrapStyle}>
                <LoginForm />
            </CenteredContentPageWrap>
        </CenteredContentPage>
    );
}

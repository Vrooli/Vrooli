import { styled, useTheme } from "@mui/material";
import Box from "@mui/material/Box";
import FormControl from "@mui/material/FormControl";
import FormHelperText from "@mui/material/FormHelperText";
import Link from "@mui/material/Link";
import Typography from "@mui/material/Typography";
import { alpha } from "@mui/material/styles";
import { Button } from "../../components/buttons/Button.js";
import { Checkbox } from "../../components/inputs/Checkbox.js";
import { Divider } from "../../components/layout/Divider.js";
import { FormControlLabel } from "../../components/inputs/FormControlLabel.js";
import { BUSINESS_NAME, LINKS, emailSignUpFormValidation, endpointsAuth, type EmailSignUpInput, type Session } from "@vrooli/shared";
import { Field, Formik, type FormikHelpers, type FormikProps } from "formik";
import React, { useCallback, useContext, useEffect, useMemo, useState } from "react";
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
import { useLazyFetch } from "../../hooks/useFetch.js";
import { useReactSearch } from "../../hooks/useReactSearch.js";
import { IconFavicon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { CenteredContentPage, pagePaddingBottom } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { removeCookie } from "../../utils/localStorage.js";
import { PubSub } from "../../utils/pubsub.js";
import { setupPush } from "../../utils/push.js";
import { type SignupViewProps } from "../../views/types.js";
import { AuthContainer, AuthFormContainer, FormSection, OAUTH_PROVIDERS_INFO, OAuthButton, OAuthContainer, OAuthSection, OrDivider, OuterAuthFormContainer, baseFormStyle, breadcrumbsStyle, emailStartAdornment, nameStartAdornment, oAuthSpanStyle } from "./authStyles.js";

export type SignupViewFormInput = EmailSignUpInput & {
    agreeToTerms: boolean;
}

const initialValues: SignupViewFormInput = {
    agreeToTerms: false,
    marketingEmails: true,
    name: "",
    email: "",
    password: "",
    confirmPassword: "",
    theme: "",
};

const agreementTextStyle = {
    fontStyle: "italic",
    margin: 0,
} as const;
// Custom sign up button component using Tailwind Button
function StyledSignUpButton({ children, ...props }: React.ComponentProps<typeof Button>) {
    const theme = useTheme();
    const SHADOW_ALPHA = 0.3;
    const buttonStyle = {
        boxShadow: `0 4px 15px ${alpha(theme.palette.secondary.main, SHADOW_ALPHA)}`,
    };
    return (
        <Button
            {...props}
            variant="primary"
            size="lg"
            className="w-full bg-gradient-to-r from-secondary-500 to-primary-500 hover:from-secondary-600 hover:to-primary-600 text-lg font-semibold transition-all duration-300 hover:scale-105 hover:shadow-lg rounded-2xl"
            style={buttonStyle}
        >
            {children}
        </Button>
    );
}
const StyledOuterAuthFormContainer = styled(OuterAuthFormContainer)(({ theme }) => ({
    marginBottom: 0,
}));


const BREADCRUMB_PATHS = [
    {
        textKey: "LogIn",
        link: LINKS.Login,
    },
    {
        textKey: "ForgotPassword",
        link: LINKS.ForgotPassword,
    },
] as const;

/**
 * SignupFormFields Component
 * 
 * Extracted form fields for better testability.
 * This component contains just the form fields and their layout,
 * while SignupForm handles the Formik wrapper and submission logic.
 */
interface SignupFormFieldsProps {
    formik: FormikProps<SignupViewFormInput>;
    inputFieldStyle?: Record<string, unknown>;
}

export function SignupFormFields({
    formik,
    inputFieldStyle = {},
}: SignupFormFieldsProps) {
    const { t } = useTranslation();

    return (
        <>
            <Field
                fullWidth
                autoComplete="name"
                id="name"
                name="name"
                label={t("Name")}
                placeholder={t("NamePlaceholder")}
                as={TextInput}
                startAdornment={nameStartAdornment.startAdornment}
            />
            <Field
                fullWidth
                autoComplete="email"
                id="email"
                name="email"
                label={t("Email", { count: 1 })}
                placeholder={t("EmailPlaceholder")}
                as={TextInput}
                startAdornment={emailStartAdornment.startAdornment}
                helperText={formik.touched.email && formik.errors.email}
                error={formik.touched.email && Boolean(formik.errors.email)}
            />
            <Divider />
            <PasswordTextInput
                fullWidth
                name="password"
                autoComplete="new-password"
                label={t("Password")}
            />
            <PasswordTextInput
                fullWidth
                name="confirmPassword"
                autoComplete="new-password"
                label={t("PasswordConfirm")}
            />
            <Box>
                <FormControlLabel
                    control={
                        <Checkbox
                            id="marketingEmails"
                            name="marketingEmails"
                            color="secondary"
                            checked={Boolean(formik.values.marketingEmails)}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            aria-label="Send me updates and offers"
                        />
                    }
                    label={
                        <Typography variant="caption">
                            Send me updates and offers
                        </Typography>
                    }
                />
                <FormControl required error={!formik.values.agreeToTerms && formik.touched.agreeToTerms}>
                    <FormControlLabel
                        control={
                            <Checkbox
                                id="agreeToTerms"
                                name="agreeToTerms"
                                color="secondary"
                                checked={Boolean(formik.values.agreeToTerms)}
                                onBlur={formik.handleBlur}
                                onChange={formik.handleChange}
                                aria-label="I agree to the terms"
                            />
                        }
                        label={
                            <Typography variant="caption">
                                I agree to the{" "}
                                <Link
                                    href={LINKS.Terms}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    onClick={stopPropagation}
                                >
                                    terms
                                </Link>
                                .
                            </Typography>
                        }
                    />
                    {!formik.values.agreeToTerms && formik.touched.agreeToTerms && (
                        <FormHelperText component="p" sx={agreementTextStyle}>
                            Please accept the terms to continue
                        </FormHelperText>
                    )}
                </FormControl>
            </Box>
        </>
    );
}

function SignupForm() {
    const session = useContext(SessionContext);
    const theme = useTheme();
    const { palette } = theme;
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [emailSignUp, { loading }] = useLazyFetch<EmailSignUpInput, Session>(endpointsAuth.emailSignup);
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    const search = useReactSearch();
    const { redirect } = useMemo(function parseUrlSearch() {
        return {
            redirect: typeof search.redirect === "string" ? search.redirect : undefined,
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

    const breadcrumbPaths = useMemo(() => BREADCRUMB_PATHS.map(path => ({
        text: t(path.textKey),
        link: path.link,
    })), [t]);

    const onSubmit = useCallback(function onSubmitCallback(values: SignupViewFormInput, helpers: FormikHelpers<SignupViewFormInput>) {
        if (values.password !== values.confirmPassword) {
            PubSub.get().publish("snack", { messageKey: "PasswordsDontMatch", severity: "Error" });
            helpers.setSubmitting(false);
            return;
        }
        fetchLazyWrapper<EmailSignUpInput, Session>({
            fetch: emailSignUp,
            inputs: {
                name: values.name,
                email: values.email,
                password: values.password,
                confirmPassword: values.confirmPassword,
                marketingEmails: Boolean(values.marketingEmails),
                theme: palette.mode ?? "light",
            },
            onSuccess: (data) => {
                SocketService.get().disconnect();
                removeCookie("FormData"); // Clear old form data cache
                setupPush(false);
                PubSub.get().publish("session", data);
                PubSub.get().publish("celebration", { targetId: "sign-up-button" });
                PubSub.get().publish("alertDialog", {
                    messageKey: "WelcomeVerifyEmail",
                    messageVariables: { appName: BUSINESS_NAME },
                    buttons: [{
                        labelKey: "Ok", onClick: () => {
                            if (redirect) {
                                setLocation(redirect);
                            } else {
                                setLocation(LINKS.Home);
                                PubSub.get().publish("menu", { id: ELEMENT_IDS.Tutorial, isOpen: true });
                            }
                        },
                    }],
                });
                SocketService.get().connect();
            },
            onError: (response) => {
                if (ServerResponseParser.hasErrorCode(response, "EmailInUse")) {
                    PubSub.get().publish("alertDialog", {
                        messageKey: "EmailInUseWrongPassword",
                        buttons: [
                            { labelKey: "Yes", onClick: () => { setLocation(LINKS.ForgotPassword); } },
                            { labelKey: "No" },
                        ],
                    });
                }
                helpers.setSubmitting(false);
            },
        });
    }, [emailSignUp, palette.mode, redirect, setLocation]);

    const inputFieldStyle = {
        "& .MuiOutlinedInput-root": {
            backgroundColor: "transparent",
            "& fieldset": { border: `1px solid ${palette.divider}` },
            "&:hover fieldset": { border: `1px solid ${palette.divider}` },
            "&.Mui-focused fieldset": { border: `1px solid ${palette.divider}` },
        },
    } as const;

    return (
        <StyledOuterAuthFormContainer>
            <AuthContainer>
                <AuthFormContainer>
                    <Formik
                        initialValues={initialValues}
                        onSubmit={onSubmit}
                        validationSchema={emailSignUpFormValidation}
                    >
                        {(formik) => (
                            <InnerForm
                                display="Dialog"
                                isLoading={loading}
                                style={baseFormStyle}
                            >
                                <FormSection>
                                    <SignupFormFields
                                        formik={formik}
                                        inputFieldStyle={inputFieldStyle}
                                    />
                                    <Box display="flex" flexDirection="column">
                                        <StyledSignUpButton
                                            id="sign-up-button"
                                            disabled={loading}
                                            type="submit"
                                        >
                                            {t("SignUp")}
                                        </StyledSignUpButton>
                                        <BreadcrumbsBase
                                            paths={breadcrumbPaths}
                                            separator={"•"}
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
        </StyledOuterAuthFormContainer>
    );
}

function stopPropagation(event: React.MouseEvent) {
    event.stopPropagation();
}

/**
 * Component that cycles through persuasive sentences about Vrooli's features
 */
function FeatureCarousel() {
    const theme = useTheme();
    const [currentFeatureIndex, setCurrentFeatureIndex] = useState(0);
    const [isVisible, setIsVisible] = useState(true);

    // Constants to fix linter errors
    const ROTATION_INTERVAL_MS = 5000;
    const FADE_TRANSITION_MS = 400;
    const TEXT_SHADOW_ALPHA = 0.2;

    // A list of compelling features based on the README and LandingView content
    const featureMessages = useMemo(() => [
        "Create and customize autonomous agents to tackle any role in your business",
        "Build dynamic teams of bots and humans that collaborate toward specific goals",
        "Create powerful workflows that combine, APIs, code, prompts, and more for any purpose",
        "Run autonomous or supervised workflows with real-time chat support",
        "Share and improve prompts and workflows, promoting a collaborative ecosystem",
        "Engage with multiple bots and humans in the same conversation",
        "Schedule tasks to run at specific times or when conditions are met",
        "Build a beautiful and efficient automation system that evolves with your needs",
    ], []);

    // Change the displayed feature every interval
    useEffect(() => {
        const interval = setInterval(() => {
            // First fade out
            setIsVisible(false);

            // After fade out, change the text and fade back in
            setTimeout(() => {
                setCurrentFeatureIndex(prevIndex =>
                    prevIndex === featureMessages.length - 1 ? 0 : prevIndex + 1,
                );
                setIsVisible(true);
            }, FADE_TRANSITION_MS);

        }, ROTATION_INTERVAL_MS);

        return () => clearInterval(interval);
    }, [featureMessages.length, FADE_TRANSITION_MS, ROTATION_INTERVAL_MS]);

    // Styles as static objects to fix linter errors
    const boxStyle = {
        mt: 4,
        textAlign: "center",
        padding: 3,
        borderRadius: 2,
        maxWidth: "700px",
        margin: "0 auto",
        marginTop: 4,
        height: "90px",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
    } as const;

    const textStyle = {
        fontWeight: 400,
        color: theme.palette.background.textSecondary,
        textShadow: `0 0 10px ${alpha(theme.palette.primary.main, TEXT_SHADOW_ALPHA)}`,
        opacity: isVisible ? 1 : 0,
        transition: `opacity ${FADE_TRANSITION_MS}ms ease-in-out`,
    } as const;

    return (
        <Box sx={boxStyle}>
            <Typography variant="h6" sx={textStyle}>
                ✨ {featureMessages[currentFeatureIndex]}
            </Typography>
        </Box>
    );
}

export function SignupView({
    display,
}: SignupViewProps) {
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
                <SignupForm />
                <FeatureCarousel />
            </Box>
        </CenteredContentPage>
    );
}


// // Wallet provider popups
// const [connectOpen, setConnectOpen] = useState(false);
// const [installOpen, setInstallOpen] = useState(false);
// const openWalletConnectDialog = useCallback(() => { setConnectOpen(true); }, []);
// const openWalletInstallDialog = useCallback(() => { setInstallOpen(true); }, []);

// // Performs handshake to establish trust between site backend and user's wallet.
// // 1. Whitelist website on wallet
// // 2. Send public address to backend
// // 3. Store public address and nonce in database
// // 4. Sign human-readable message (which includes nonce) using wallet
// // 5. Send signed message to backend for verification
// // 6. Receive JWT and user session
// const walletLogin = useCallback(async (providerKey: string) => {
//     // Check if wallet extension installed
//     if (!hasWalletExtension(providerKey)) {
//         PubSub.get().publish("alertDialog", {
//             messageKey: "WalletProviderNotFoundDetails",
//             buttons: [
//                 { labelKey: "TryAgain", onClick: openWalletConnectDialog },
//                 { labelKey: "InstallWallet", onClick: openWalletInstallDialog },
//                 { labelKey: "EmailLogin", onClick: toEmailLogIn },
//             ],
//         });
//         return;
//     }
//     // Validate wallet
//     const walletCompleteResult = await validateWallet(providerKey);
//     if (walletCompleteResult?.session) {
//         PubSub.get().publish("snack", { messageKey: "WalletVerified", severity: "Success" });
//         PubSub.get().publish("session", walletCompleteResult.session);
//         // Redirect to main dashboard
//         setLocation(redirect ?? LINKS.Home);
//         // Set up push notifications
//         setupPush();
//     }
// }, [openWalletConnectDialog, openWalletInstallDialog, toEmailLogIn, setLocation, redirect]);

// const closeWalletConnectDialog = useCallback((providerKey: string | null) => {
//     setConnectOpen(false);
//     if (providerKey) {
//         walletLogin(providerKey);
//     }
// }, [walletLogin]);

// const closeWalletInstallDialog = useCallback(() => { setInstallOpen(false); }, []);

//  <WalletSelectDialog
//             handleOpenInstall={openWalletInstallDialog}
//             open={connectOpen}
//             onClose={closeWalletConnectDialog}
//         />
//         <WalletInstallDialog
//             open={installOpen}
//             onClose={closeWalletInstallDialog}
//         />

//          <Box sx={{
//                     display: "flex",
//                     justifyContent: "center",
//                     alignItems: "center",
//                     marginBottom: 2,
//                 }}>
//                     <Typography
//                         variant="h6"
//                         sx={{
//                             display: "inline-block",
//                         }}
//                     >
//                         {t("SelectLogInMethod")}
//                     </Typography>
//                     <HelpButton markdown={helpText} />
//                 </Box>
//                 <Stack
//                     direction="column"
//                     spacing={2}
//                 >
//                     <Button
//                         fullWidth
//                         onClick={openWalletConnectDialog}
//                         startIcon={<WalletIcon />}
//                         sx={{ ...buttonProps }}
//                     >{t("Wallet")}</Button>
//                     <Button
//                         fullWidth
//                         onClick={toEmailLogIn}
//                         startIcon={<EmailIcon />}
//                         sx={{ ...buttonProps }}
//                     >{t("Email")}</Button>
//                 </Stack>

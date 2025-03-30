import { BUSINESS_NAME, emailSignUpFormValidation, EmailSignUpInput, endpointsAuth, LINKS, Session, SignUpPageTabOption } from "@local/shared";
import { Box, BoxProps, Button, Checkbox, FormControl, FormControlLabel, FormHelperText, InputAdornment, Link, styled, useTheme } from "@mui/material";
import { Field, Formik, FormikHelpers } from "formik";
import { useCallback, useContext, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { ServerResponseParser } from "../../api/responseParser.js";
import { SocketService } from "../../api/socket.js";
import { BreadcrumbsBase } from "../../components/breadcrumbs/BreadcrumbsBase.js";
import { PasswordTextInput } from "../../components/inputs/PasswordTextInput/PasswordTextInput.js";
import { TextInput } from "../../components/inputs/TextInput/TextInput.js";
import { Footer } from "../../components/navigation/Footer.js";
import { TopBar } from "../../components/navigation/TopBar.js";
import { PageTabs } from "../../components/PageTabs/PageTabs.js";
import { SessionContext } from "../../contexts/session.js";
import { BaseForm } from "../../forms/BaseForm/BaseForm.js";
import { formSubmit } from "../../forms/styles.js";
import { useIsLeftHanded } from "../../hooks/subscriptions.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { useReactSearch } from "../../hooks/useReactSearch.js";
import { useTabs } from "../../hooks/useTabs.js";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { FormContainer, FormSection } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { removeCookie } from "../../utils/localStorage.js";
import { PubSub } from "../../utils/pubsub.js";
import { setupPush } from "../../utils/push.js";
import { signUpTabParams } from "../../utils/search/objectToSearch.js";
import { SignupViewProps } from "../../views/types.js";

type FormInput = EmailSignUpInput & {
    agreeToTerms: boolean;
}

const initialValues: FormInput = {
    agreeToTerms: false,
    marketingEmails: true,
    name: "",
    email: "",
    password: "",
    confirmPassword: "",
    theme: "",
};

const outerFormStyle = {
    height: "100vh",
    overflowY: "auto",
    overflowX: "hidden",
} as const;
const baseFormStyle = {
    paddingBottom: "unset",
    position: "sticky",
    margin: 0,
} as const;
const breadcrumbsStyle = {
    margin: "auto",
} as const;

const nameInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon name="User" />
        </InputAdornment>
    ),
} as const;

const emailInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon name="Email" />
        </InputAdornment>
    ),
} as const;

function SignupForm() {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
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

    const handleSubmit = useCallback(function handleSubmitCallback(values: FormInput, helpers: FormikHelpers<FormInput>) {
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
    }, [emailSignUp, palette.mode, setLocation]);

    const breadcrumbPaths = [
        {
            text: t("LogIn"),
            link: LINKS.Login,
        },
        {
            text: t("ForgotPassword"),
            link: LINKS.ForgotPassword,
        },
    ] as const;

    return (
        <Box sx={outerFormStyle}>
            <Formik
                initialValues={initialValues}
                onSubmit={handleSubmit}
                validationSchema={emailSignUpFormValidation}
            >
                {(formik) => <BaseForm
                    display={"dialog"}
                    isLoading={loading}
                    style={baseFormStyle}
                >
                    <FormContainer>
                        <FormSection variant="transparent">
                            <Field
                                fullWidth
                                autoComplete="name"
                                name="name"
                                label={t("Name")}
                                placeholder={t("NamePlaceholder")}
                                as={TextInput}
                                InputProps={nameInputProps}
                            />
                            <Field
                                fullWidth
                                autoComplete="email"
                                name="email"
                                label={t("Email", { count: 1 })}
                                placeholder={t("EmailPlaceholder")}
                                as={TextInput}
                                InputProps={emailInputProps}
                                helperText={formik.touched.email && formik.errors.email}
                                error={formik.touched.email && Boolean(formik.errors.email)}
                            />
                        </FormSection>
                        <FormSection variant="transparent">
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
                        </FormSection>
                        <FormSection variant="transparent">
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
                                label="I agree to receive marketing promotions and updates via email."
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
                                        />
                                    }
                                    label={
                                        <>
                                            I agree to the{" "}
                                            <Link
                                                href="/terms-and-conditions"
                                                target="_blank"
                                                rel="noopener noreferrer"
                                                onClick={stopPropagation}
                                            >
                                                terms and conditions
                                            </Link>
                                            .
                                        </>
                                    }
                                />
                                <FormHelperText>
                                    {formik.touched.agreeToTerms !== true && "You must agree to the terms and conditions"}
                                </FormHelperText>
                            </FormControl>
                        </FormSection>
                        <Box width="100%" display="flex" flexDirection="column" p={2}>
                            <Button
                                id="sign-up-button"
                                fullWidth
                                disabled={loading}
                                type="submit"
                                color="secondary"
                                variant="contained"
                                sx={formSubmit}
                            >
                                {t("SignUp")}
                            </Button>
                            <BreadcrumbsBase
                                paths={breadcrumbPaths}
                                separator={"•"}
                                sx={breadcrumbsStyle}
                            />
                        </Box>
                    </FormContainer>
                </BaseForm>}
            </Formik>
        </Box>
    );
}

interface ScreenshotProps extends BoxProps {
    isMobile: boolean;
}

const Screenshot = styled("img", {
    shouldForwardProp: (prop) => prop !== "isMobile",
})<ScreenshotProps>(({ isMobile }) => ({
    cursor: "pointer",
    width: isMobile ? "-webkit-fill-available" : "100%",
    maxWidth: isMobile ? "720px" : "800px",
    margin: "auto",
    borderRadius: "8px",
    display: "inline-block",
    maxHeight: isMobile ? "1280px" : "400px",
    zIndex: 5,
}));

const screenshots = [{
    src: "1-intro.webp",
    alt: "Chatting with bot from main page",
}, {
    src: "2-build.webp",
    alt: "Building a multi-step routine",
}, {
    src: "3-team.webp",
    alt: "Viewing a team",
}, {
    src: "4-search.webp",
    alt: "Searching for existing routines",
}];

function getFullSrc(src: string, isMobile: boolean) {
    return `${window.location.origin}/screenshots/${isMobile ? "narrow" : "wide"}-${src}`;
}

function stopPropagation(event: React.MouseEvent) {
    event.stopPropagation();
}

function Promo({
    isMobile,
}: {
    isMobile: boolean;
}) {
    const handleImageOpen = useCallback((index: number) => {
        if (index < 0 || index >= screenshots.length) {
            console.error("Invalid index", index);
            return;
        }
        const screenshot = screenshots[index];
        const src = getFullSrc(screenshot.src, isMobile);
        PubSub.get().publish("popupImage", { src, alt: screenshot.alt });
    }, [isMobile]);

    return (
        <>
            <Box display="flex" flexDirection="column" gap={8} p={2}>
                {screenshots.map(({ src, alt }, index) => {
                    const fullSrc = getFullSrc(src, isMobile);

                    function handleClick() {
                        console.log("opening image", index, src, alt, screenshots);
                        handleImageOpen(index);
                    }

                    return (
                        <Screenshot
                            key={src}
                            src={fullSrc}
                            alt={alt}
                            isMobile={isMobile}
                            onClick={handleClick}
                        />
                    );
                })}
            </Box>
            <Footer />
        </>
    );
}

const OuterBox = styled(Box)(({ theme }) => ({
    background: theme.palette.background.paper,
    height: "100%",
}));

export function SignupView({
    display,
    onClose,
}: SignupViewProps) {
    const { breakpoints, palette } = useTheme();
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isLeftHanded = useIsLeftHanded();

    const {
        currTab,
        handleTabChange,
        tabs,
    } = useTabs({ id: "sign-up-tabs", tabParams: signUpTabParams, disableHistory: true, display });

    // Side menus are not supported in this page due to the way it's styled
    useMemo(function hideMenusMemo() {
        PubSub.get().publish("menu", { id: ELEMENT_IDS.LeftDrawer, isOpen: false });
        PubSub.get().publish("menu", { id: ELEMENT_IDS.RightDrawer, isOpen: false });
    }, []);

    const innerBoxStyle = useMemo(function innerBoxStyleMemo() {
        if (isMobile) {
            return {
                display: "flex",
                flexDirection: "column",
                height: "100%",
                width: "100vw",
            } as const;
        } else {
            return {
                display: "flex",
                flexDirection: isLeftHanded ? "row-reverse" : "row",
                height: "100vh",
                width: "100vw",
                overflow: "hidden",
            } as const;
        }
    }, [isLeftHanded, isMobile]);

    const signUpBoxStyle = useMemo(function signUpBoxStyleMemo() {
        return {
            flexGrow: 1,
            maxWidth: isMobile ? "unset" : "min(400px, 30%)",
            background: palette.background.paper,
            display: isMobile && currTab.key !== SignUpPageTabOption.SignUp
                ? "none"
                : "block",
            overflow: "unset",
            position: "relative",
            zIndex: 3,
        } as const;
    }, [currTab, isMobile, palette.background.paper]);

    const promoBoxStyle = useMemo(function promoBoxStyleMemo() {
        return {
            background: palette.background.default,
            color: "white",
            display: isMobile && currTab.key !== SignUpPageTabOption.MoreInfo
                ? "none"
                : "block",
            overflowY: "auto",
            height: "100vh",
            width: "100%",
            position: "sticky",
        } as const;
    }, [currTab.key, isMobile, palette.background.default]);

    return (
        <OuterBox>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("SignUp")}
                titleBehaviorDesktop="ShowIn"
                below={isMobile && (
                    <PageTabs
                        ariaLabel="sign-up-tabs"
                        fullWidth
                        id="sign-up-tabs"
                        ignoreIcons
                        currTab={currTab}
                        onChange={handleTabChange}
                        tabs={tabs}
                    />
                )}
            />
            <Box sx={innerBoxStyle}>
                <Box sx={signUpBoxStyle}>
                    <SignupForm />
                </Box>
                <Box sx={promoBoxStyle}>
                    <Promo isMobile={isMobile} />
                </Box>
            </Box>
        </OuterBox>
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

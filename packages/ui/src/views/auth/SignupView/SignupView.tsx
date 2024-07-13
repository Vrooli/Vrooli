import { BUSINESS_NAME, emailSignUpFormValidation, EmailSignUpInput, endpointPostAuthEmailSignup, LINKS, Session } from "@local/shared";
import { Box, BoxProps, Button, Checkbox, FormControl, FormControlLabel, FormHelperText, Grid, IconButton, InputAdornment, Link, Modal, styled, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper, hasErrorCode } from "api";
import { PasswordTextInput } from "components/inputs/PasswordTextInput/PasswordTextInput";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { Field, Formik, FormikHelpers } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { formNavLink, formPaper, formSubmit } from "forms/styles";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useTabs } from "hooks/useTabs";
import { useWindowSize } from "hooks/useWindowSize";
import { CloseIcon, EmailIcon, UserIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { clickSize } from "styles";
import { removeCookie } from "utils/cookies";
import { PubSub } from "utils/pubsub";
import { setupPush } from "utils/push";
import { SignUpPageTabOption, signUpTabParams } from "utils/search/objectToSearch";
import { SignupViewProps } from "views/types";

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

const baseFormStyle = {
    ...formPaper,
    paddingBottom: "unset",
    position: "sticky",
    top: "80px",
} as const;

const logInLinkStyle = {
    ...clickSize,
    ...formNavLink,
} as const;

const forgotPasswordLinkStyle = {
    ...clickSize,
    ...formNavLink,
    flexDirection: "row-reverse",
} as const;

const nameInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <UserIcon />
        </InputAdornment>
    ),
} as const;

const emailInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <EmailIcon />
        </InputAdornment>
    ),
} as const;

const checkboxGridStyle = { display: "flex", justifyContent: "left" } as const;

function SignupForm() {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [emailSignUp, { loading }] = useLazyFetch<EmailSignUpInput, Session>(endpointPostAuthEmailSignup);

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
                removeCookie("FormData"); // Clear old form data cache
                setupPush(false);
                PubSub.get().publish("session", data);
                PubSub.get().publish("celebration", { targetId: "sign-up-button" });
                PubSub.get().publish("alertDialog", {
                    messageKey: "WelcomeVerifyEmail",
                    messageVariables: { appName: BUSINESS_NAME },
                    buttons: [{
                        labelKey: "Ok", onClick: () => {
                            setLocation(LINKS.Home);
                            PubSub.get().publish("tutorial");
                        },
                    }],
                });
            },
            onError: (response) => {
                if (hasErrorCode(response, "EmailInUse")) {
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

    return (
        <>
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
                    <Grid container spacing={2}>
                        <Grid item xs={12}>
                            <Field
                                fullWidth
                                autoComplete="name"
                                autoFocus
                                name="name"
                                label={t("Name")}
                                placeholder={t("NamePlaceholder")}
                                as={TextInput}
                                InputProps={nameInputProps}
                            />
                        </Grid>
                        <Grid item xs={12}>
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
                        </Grid>
                        <Grid item xs={12}>
                            <PasswordTextInput
                                fullWidth
                                name="password"
                                autoComplete="new-password"
                                label={t("Password")}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <PasswordTextInput
                                fullWidth
                                name="confirmPassword"
                                autoComplete="new-password"
                                label={t("PasswordConfirm")}
                            />
                        </Grid>
                        <Grid item xs={12} sx={checkboxGridStyle}>
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
                        </Grid>
                        <Grid item xs={12} sx={checkboxGridStyle}>
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
                        </Grid>
                    </Grid>
                    <Button
                        id="sign-up-button"
                        fullWidth
                        disabled={loading}
                        type="submit"
                        color="secondary"
                        variant="contained"
                        sx={{ ...formSubmit }}
                    >
                        {t("SignUp")}
                    </Button>
                    <Box sx={{
                        display: "flex",
                        flexDirection: "row",
                        justifyContent: "space-between",
                        alignItems: "center",
                    }}>
                        <Link href={LINKS.Login}>
                            <Typography
                                sx={logInLinkStyle}
                            >
                                {t("LogIn")}
                            </Typography>
                        </Link>
                        <Link href={LINKS.ForgotPassword}>
                            <Typography
                                sx={forgotPasswordLinkStyle}
                            >
                                {t("ForgotPassword")}
                            </Typography>
                        </Link>
                    </Box>
                </BaseForm>}
            </Formik>
        </>
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
    src: "1-intro.png",
    alt: "Chatting with bot from main page",
}, {
    src: "2-build.png",
    alt: "Building a multi-step routine",
}, {
    src: "3-team.png",
    alt: "Viewing a team",
}, {
    src: "4-search.png",
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
    const [openedImage, setOpenedImage] = useState<number | null>(null);
    const handleCloseImage = useCallback(() => { setOpenedImage(null); }, []);
    const handleImageOpen = useCallback((index: number) => { setOpenedImage(index); }, []);

    return (
        <>
            <Box sx={{
                display: "flex",
                flexDirection: "column",
                gap: 8,
            }}>
                {screenshots.map(({ src, alt }, index) => {
                    const fullSrc = getFullSrc(src, isMobile);

                    function handleClick() {
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
                <Modal
                    open={openedImage !== null}
                    onClose={handleCloseImage}
                    aria-labelledby="full-size-image"
                    aria-describedby="full-size-screenshot"
                    sx={{
                        backgroundColor: "black",
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                    }}
                >
                    <Box
                        sx={{
                            position: "relative",
                            width: "100%",
                            height: "100%",
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                        }}
                        onClick={handleCloseImage}
                    >
                        <img
                            src={getFullSrc(screenshots[openedImage ?? 0].src, isMobile)}
                            alt={screenshots[openedImage ?? 0].alt}
                            onClick={stopPropagation}
                            style={{
                                maxWidth: "90%",
                                maxHeight: "90%",
                                objectFit: "contain",
                            }}
                        />
                        <IconButton
                            aria-label="close"
                            onClick={handleCloseImage}
                            sx={{
                                position: "absolute",
                                right: 20,
                                top: 20,
                                color: "white",
                            }}
                        >
                            <CloseIcon />
                        </IconButton>
                    </Box>
                </Modal>
            </Box>
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
    } = useTabs({ id: "sign-up-tabs", tabParams: signUpTabParams, display });

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
                height: "100%",
                width: "100vw",
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
            padding: 2,
            background: palette.background.default,
            backgroundAttachment: "fixed",
            color: "white",
            display: isMobile && currTab.key !== SignUpPageTabOption.MoreInfo
                ? "none"
                : "block",
            overflow: "auto",
            width: "100%",
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

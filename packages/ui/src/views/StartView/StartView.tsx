// Provides 3 options for entering the main application:
// 1. Wallet authentication - Quickest and safest method, but requires an extension which supports it
// 2. Email authentication - Requires email and password. Pretty safe if using password manager, 
// but wallet must be connected before performing any blockchain-related activities
// 3. Guest pass - Those who don't want to make an account can still view and run routines, but will not
// be able to utilize the full functionality of the service
import { EmailLogInInput, endpointPostAuthEmailLogin, LINKS, Session } from "@local/shared";
import { Box, SxProps, useTheme } from "@mui/material";
import { fetchLazyWrapper, hasErrorCode } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { ForgotPasswordForm, LogInForm, ResetPasswordForm, SignUpForm } from "forms/auth";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getCurrentUser } from "utils/authentication/session";
import { hasWalletExtension, validateWallet } from "utils/authentication/walletIntegration";
import { Forms } from "utils/consts";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useReactSearch } from "utils/hooks/useReactSearch";
import { PubSub } from "utils/pubsub";
import { setupPush } from "utils/push";
import { SessionContext } from "utils/SessionContext";
import { StartViewProps } from "../types";

const helpText =
    "Logging in allows you to vote, save favorites, and contribute to the community.\n\nChoose **WALLET** if you are on a browser with a supported extension. This will not cost any money, but requires the signing of a message to verify that you own the wallet. Wallets will be utilized in the future to support user donations and execute routines tied to smart contracts.\n\nChoose **EMAIL** if you are on mobile or do not have a Nami account. A wallet can be associated with your account later.";

const buttonProps: SxProps = {
    height: "4em",
};

const emailTitleId = "email-login-dialog-title";

export const StartView = ({
    display = "page",
    onClose,
    zIndex,
}: StartViewProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    const search = useReactSearch();
    const { redirect, verificationCode } = useMemo(() => ({
        redirect: typeof search.redirect === "string" ? search.redirect : undefined,
        verificationCode: typeof search.verificationCode === "string" ? search.verificationCode : undefined,
    }), [search]);

    const [emailLogIn] = useLazyFetch<EmailLogInInput, Session>(endpointPostAuthEmailLogin);
    // Handles email authentication popup
    const [emailPopupOpen, setEmailPopupOpen] = useState(false);
    const [popupForm, setPopupForm] = useState<Forms>(Forms.LogIn);
    const handleFormChange = useCallback((type: Forms = Forms.LogIn) => type !== popupForm && setPopupForm(type), [popupForm]);
    const Form = useMemo(() => {
        switch (popupForm) {
            case Forms.ForgotPassword:
                return ForgotPasswordForm;
            case Forms.LogIn:
                return LogInForm;
            case Forms.ResetPassword:
                return ResetPasswordForm;
            case Forms.SignUp:
                return SignUpForm;
            default:
                return LogInForm;
        }
    }, [popupForm]);

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
                        PubSub.get().publishSnack({ messageKey: "EmailVerified", severity: "Success" });
                        PubSub.get().publishSession(data);
                        setLocation(redirect ?? LINKS.Home);
                    },
                    onError: (response) => {
                        if (hasErrorCode(response, "MustResetPassword")) {
                            PubSub.get().publishAlertDialog({
                                messageKey: "ChangePasswordBeforeLogin",
                                buttons: [
                                    { labelKey: "Ok", onClick: () => { setLocation(redirect ?? LINKS.Home); } },
                                ],
                            });
                        }
                    },
                });
            }
            // Otherwise, open log in form
            else {
                setEmailPopupOpen(true);
                setPopupForm(Forms.LogIn);
            }
        }
    }, [emailLogIn, verificationCode, redirect, setLocation, userId]);

    // Wallet provider popups
    const [connectOpen, setConnectOpen] = useState(false);
    const [installOpen, setInstallOpen] = useState(false);
    const openWalletConnectDialog = useCallback(() => { setConnectOpen(true); }, []);
    const openWalletInstallDialog = useCallback(() => { setInstallOpen(true); }, []);

    const toEmailLogIn = useCallback(() => {
        setPopupForm(Forms.LogIn);
        setEmailPopupOpen(true);
    }, []);

    const closeEmailPopup = useCallback(() => setEmailPopupOpen(false), []);

    // Performs handshake to establish trust between site backend and user's wallet.
    // 1. Whitelist website on wallet
    // 2. Send public address to backend
    // 3. Store public address and nonce in database
    // 4. Sign human-readable message (which includes nonce) using wallet
    // 5. Send signed message to backend for verification
    // 6. Receive JWT and user session
    const walletLogin = useCallback(async (providerKey: string) => {
        // Check if wallet extension installed
        if (!hasWalletExtension(providerKey)) {
            PubSub.get().publishAlertDialog({
                messageKey: "WalletProviderNotFoundDetails",
                buttons: [
                    { labelKey: "TryAgain", onClick: openWalletConnectDialog },
                    { labelKey: "InstallWallet", onClick: openWalletInstallDialog },
                    { labelKey: "EmailLogin", onClick: toEmailLogIn },
                ],
            });
            return;
        }
        // Validate wallet
        const walletCompleteResult = await validateWallet(providerKey);
        if (walletCompleteResult?.session) {
            PubSub.get().publishSnack({ messageKey: "WalletVerified", severity: "Success" });
            PubSub.get().publishSession(walletCompleteResult.session);
            // Redirect to main dashboard
            setLocation(redirect ?? LINKS.Home);
            // Set up push notifications
            setupPush();
        }
    }, [openWalletConnectDialog, openWalletInstallDialog, toEmailLogIn, setLocation, redirect]);

    const closeWalletConnectDialog = useCallback((providerKey: string | null) => {
        setConnectOpen(false);
        if (providerKey) {
            walletLogin(providerKey);
        }
    }, [walletLogin]);

    const closeWalletInstallDialog = useCallback(() => { setInstallOpen(false); }, []);

    return (
        <>
            {/* Dialogs */}
            {/* <WalletSelectDialog
                handleOpenInstall={openWalletInstallDialog}
                open={connectOpen}
                onClose={closeWalletConnectDialog}
                zIndex={zIndex + 1}
            />
            <WalletInstallDialog
                open={installOpen}
                onClose={closeWalletInstallDialog}
                zIndex={connectOpen ? 201 : 200}
            />
            <LargeDialog
                id="email-auth-dialog"
                isOpen={emailPopupOpen}
                onClose={closeEmailPopup}
                titleId={emailTitleId}
                zIndex={zIndex + 1}
            >
                <Form
                    onClose={closeEmailPopup}
                    onFormChange={handleFormChange}
                />
            </LargeDialog> */}
            {/* App bar */}
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Start")}
                hideTitleOnDesktop
                zIndex={zIndex}
            />
            {/* Main content */}
            <Box sx={{
                display: "flex",
                flexDirection: "column",
                justifyContent: "center",
                alignItems: "center",
                textAlign: "center",
                height: "calc(100vh - 128px)", // Minus double the app bar height
            }}>
                <Box sx={{
                    width: "min(calc(100vw - 16px), 600px)",
                    marginTop: 4,
                    background: palette.background.paper,
                    borderRadius: 2,
                    overflow: "overlay",
                }}>
                    {/* <Box sx={{
                        display: "flex",
                        justifyContent: "center",
                        alignItems: "center",
                        marginBottom: 2,
                    }}>
                        <Typography
                            variant="h6"
                            sx={{
                                display: "inline-block",
                            }}
                        >
                            {t("SelectLogInMethod")}
                        </Typography>
                        <HelpButton markdown={helpText} />
                    </Box>
                    <Stack
                        direction="column"
                        spacing={2}
                    >
                        <Button
                            fullWidth
                            onClick={openWalletConnectDialog}
                            startIcon={<WalletIcon />}
                            sx={{ ...buttonProps }}
                        >{t("Wallet")}</Button>
                        <Button
                            fullWidth
                            onClick={toEmailLogIn}
                            startIcon={<EmailIcon />}
                            sx={{ ...buttonProps }}
                        >{t("Email")}</Button>
                    </Stack> */}
                    <Form
                        onClose={closeEmailPopup}
                        onFormChange={handleFormChange}
                        zIndex={zIndex}
                    />
                </Box>
            </Box>
        </>
    );
};

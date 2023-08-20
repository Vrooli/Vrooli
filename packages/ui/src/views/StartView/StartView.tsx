// Provides 3 options for entering the main application:
// 1. Wallet authentication - Quickest and safest method, but requires an extension which supports it
// 2. Email authentication - Requires email and password. Pretty safe if using password manager, 
// but wallet must be connected before performing any blockchain-related activities
// 3. Guest pass - Those who don't want to make an account can still view and run routines, but will not
// be able to utilize the full functionality of the service
import { EmailLogInInput, endpointPostAuthEmailLogin, LINKS, Session } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { fetchLazyWrapper, hasErrorCode } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { ForgotPasswordForm, LogInForm, ResetPasswordForm, SignUpForm } from "forms/auth";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useReactSearch } from "hooks/useReactSearch";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { addSearchParams, parseSearchParams, useLocation } from "route";
import { getCurrentUser } from "utils/authentication/session";
import { Forms } from "utils/consts";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { StartViewProps } from "../types";

export const StartView = ({
    isOpen,
    onClose,
}: StartViewProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const display = toDisplay(isOpen);
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    const search = useReactSearch();
    const { redirect, verificationCode } = useMemo(() => ({
        redirect: typeof search.redirect === "string" ? search.redirect : undefined,
        verificationCode: typeof search.verificationCode === "string" ? search.verificationCode : undefined,
    }), [search]);

    const [emailLogIn] = useLazyFetch<EmailLogInInput, Session>(endpointPostAuthEmailLogin);
    const [formType, setFormType] = useState<Forms>(() => {
        // Default to log in form
        let form: Forms = Forms.LogIn;
        // Check search params
        const searchParams = parseSearchParams();
        if (searchParams.form && Object.values(Forms).includes(searchParams.form as Forms)) {
            form = searchParams.form as Forms;
        }
        return form;
    });
    const handleFormChange = useCallback((type: Forms = Forms.LogIn) => {
        if (type === formType) return;
        setFormType(type);
        addSearchParams(setLocation, { form: type });
    }, [formType, setLocation]);
    const Form = useMemo(() => {
        switch (formType) {
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
    }, [formType]);

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
                setFormType(Forms.LogIn);
            }
        }
    }, [emailLogIn, verificationCode, redirect, setLocation, userId]);

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
    //         PubSub.get().publishAlertDialog({
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
    //         PubSub.get().publishSnack({ messageKey: "WalletVerified", severity: "Success" });
    //         PubSub.get().publishSession(walletCompleteResult.session);
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

    return (
        <>
            {/* Dialogs */}
            {/* <WalletSelectDialog
                handleOpenInstall={openWalletInstallDialog}
                open={connectOpen}
                onClose={closeWalletConnectDialog}
            />
            <WalletInstallDialog
                open={installOpen}
                onClose={closeWalletInstallDialog}
            /> */}
            {/* App bar */}
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Start")}
                hideTitleOnDesktop
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
                        onFormChange={handleFormChange}
                    />
                </Box>
            </Box>
        </>
    );
};

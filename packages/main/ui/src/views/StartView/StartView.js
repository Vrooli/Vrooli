import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { EmailIcon, WalletIcon } from "@local/icons";
import { Box, Button, Stack, Typography } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { authEmailLogIn } from "../../api/generated/endpoints/auth_emailLogIn";
import { useCustomMutation } from "../../api/hooks";
import { hasErrorCode, mutationWrapper } from "../../api/utils";
import { HelpButton } from "../../components/buttons/HelpButton/HelpButton";
import { LargeDialog } from "../../components/dialogs/LargeDialog/LargeDialog";
import { WalletInstallDialog } from "../../components/dialogs/WalletInstallDialog/WalletInstallDialog";
import { WalletSelectDialog } from "../../components/dialogs/WalletSelectDialog/WalletSelectDialog";
import { TopBar } from "../../components/navigation/TopBar/TopBar";
import { ForgotPasswordForm, LogInForm, ResetPasswordForm, SignUpForm } from "../../forms/auth";
import { getCurrentUser } from "../../utils/authentication/session";
import { hasWalletExtension, validateWallet } from "../../utils/authentication/walletIntegration";
import { Forms } from "../../utils/consts";
import { useReactSearch } from "../../utils/hooks/useReactSearch";
import { PubSub } from "../../utils/pubsub";
import { setupPush } from "../../utils/push";
import { useLocation } from "../../utils/route";
import { SessionContext } from "../../utils/SessionContext";
const helpText = "Logging in allows you to vote, save favorites, and contribute to the community.\n\nChoose **WALLET** if you are on a browser with a supported extension. This will not cost any money, but requires the signing of a message to verify that you own the wallet. Wallets will be utilized in the future to support user donations and execute routines tied to smart contracts.\n\nChoose **EMAIL** if you are on mobile or do not have a Nami account. A wallet can be associated with your account later.";
const buttonProps = {
    height: "4em",
};
const emailTitleId = "email-login-dialog-title";
export const StartView = ({ display = "page", }) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const search = useReactSearch();
    const { redirect, verificationCode } = useMemo(() => ({
        redirect: typeof search.redirect === "string" ? search.redirect : undefined,
        verificationCode: typeof search.verificationCode === "string" ? search.verificationCode : undefined,
    }), [search]);
    const [emailLogIn] = useCustomMutation(authEmailLogIn);
    const [emailPopupOpen, setEmailPopupOpen] = useState(false);
    const [popupForm, setPopupForm] = useState(Forms.LogIn);
    const handleFormChange = useCallback((type = Forms.LogIn) => type !== popupForm && setPopupForm(type), [popupForm]);
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
    useEffect(() => {
        if (verificationCode) {
            if (userId) {
                mutationWrapper({
                    mutation: emailLogIn,
                    input: { verificationCode },
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
            else {
                setEmailPopupOpen(true);
                setPopupForm(Forms.LogIn);
            }
        }
    }, [emailLogIn, verificationCode, redirect, setLocation, userId]);
    const [connectOpen, setConnectOpen] = useState(false);
    const [installOpen, setInstallOpen] = useState(false);
    const openWalletConnectDialog = useCallback(() => { setConnectOpen(true); }, []);
    const openWalletInstallDialog = useCallback(() => { setInstallOpen(true); }, []);
    const toEmailLogIn = useCallback(() => {
        setPopupForm(Forms.LogIn);
        setEmailPopupOpen(true);
    }, []);
    const closeEmailPopup = useCallback(() => setEmailPopupOpen(false), []);
    const walletLogin = useCallback(async (providerKey) => {
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
        const walletCompleteResult = await validateWallet(providerKey);
        if (walletCompleteResult?.session) {
            PubSub.get().publishSnack({ messageKey: "WalletVerified", severity: "Success" });
            PubSub.get().publishSession(walletCompleteResult.session);
            setLocation(walletCompleteResult?.firstLogIn ? LINKS.Welcome : (redirect ?? LINKS.Home));
            setupPush();
        }
    }, [openWalletConnectDialog, openWalletInstallDialog, toEmailLogIn, setLocation, redirect]);
    const closeWalletConnectDialog = useCallback((providerKey) => {
        setConnectOpen(false);
        if (providerKey) {
            walletLogin(providerKey);
        }
    }, [walletLogin]);
    const closeWalletInstallDialog = useCallback(() => { setInstallOpen(false); }, []);
    return (_jsxs(_Fragment, { children: [_jsx(WalletSelectDialog, { handleOpenInstall: openWalletInstallDialog, open: connectOpen, onClose: closeWalletConnectDialog, zIndex: 200 }), _jsx(WalletInstallDialog, { open: installOpen, onClose: closeWalletInstallDialog, zIndex: connectOpen ? 201 : 200 }), _jsx(LargeDialog, { id: "email-auth-dialog", isOpen: emailPopupOpen, onClose: closeEmailPopup, titleId: emailTitleId, zIndex: 201, children: _jsx(Form, { onClose: closeEmailPopup, onFormChange: handleFormChange }) }), _jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Start",
                } }), _jsx(Box, { sx: {
                    display: "flex",
                    flexDirection: "column",
                    justifyContent: "center",
                    alignItems: "center",
                    textAlign: "center",
                    height: "calc(100vh - 128px)",
                    marginTop: 4,
                }, children: _jsxs(Box, { sx: {
                        width: "min(calc(100vw - 16px), 400px)",
                    }, children: [_jsxs(Box, { sx: {
                                display: "flex",
                                justifyContent: "center",
                                alignItems: "center",
                                marginBottom: 2,
                            }, children: [_jsx(Typography, { variant: "h6", sx: {
                                        display: "inline-block",
                                    }, children: t("SelectLogInMethod") }), _jsx(HelpButton, { markdown: helpText })] }), _jsxs(Stack, { direction: "column", spacing: 2, children: [_jsx(Button, { fullWidth: true, onClick: openWalletConnectDialog, startIcon: _jsx(WalletIcon, {}), sx: { ...buttonProps }, children: t("Wallet") }), _jsx(Button, { fullWidth: true, onClick: toEmailLogIn, startIcon: _jsx(EmailIcon, {}), sx: { ...buttonProps }, children: t("Email") })] })] }) })] }));
};
//# sourceMappingURL=StartView.js.map
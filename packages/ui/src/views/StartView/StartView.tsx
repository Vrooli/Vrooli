// Provides 3 options for entering the main application:
// 1. Wallet authentication - Quickest and safest method, but requires an extension which supports it
// 2. Email authentication - Requires email and password. Pretty safe if using password manager, 
// but wallet must be connected before performing any blockchain-related activities
// 3. Guest pass - Those who don't want to make an account can still view and run routines, but will not
// be able to utilize the full functionality of the service
import {
    Box,
    Button,
    Dialog,
    Stack,
    SxProps,
    Typography
} from '@mui/material';
import { EmailLogInInput, LINKS, Session } from '@shared/consts';
import { useLocation } from '@shared/route';
import { authEmailLogIn } from 'api/generated/endpoints/auth_emailLogIn';
import { authGuestLogIn } from 'api/generated/endpoints/auth_guestLogIn';
import { useCustomMutation } from 'api/hooks';
import { hasErrorCode, mutationWrapper } from 'api/utils';
import { HelpButton } from 'components/buttons/HelpButton/HelpButton';
import { DialogTitle } from 'components/dialogs/DialogTitle/DialogTitle';
import { WalletInstallDialog } from 'components/dialogs/WalletInstallDialog/WalletInstallDialog';
import { WalletSelectDialog } from 'components/dialogs/WalletSelectDialog/WalletSelectDialog';
import { TopBar } from 'components/navigation/TopBar/TopBar';
import { ForgotPasswordForm } from 'forms/ForgotPasswordForm';
import { LogInForm } from 'forms/LogInForm';
import { ResetPasswordForm } from 'forms/ResetPasswordForm';
import { SignUpForm } from 'forms/SignUpForm';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { subscribeUserToPush } from 'serviceWorkerRegistration';
import { getCurrentUser } from 'utils/authentication/session';
import { hasWalletExtension, validateWallet } from 'utils/authentication/walletIntegration';
import { Forms } from 'utils/consts';
import { useReactSearch } from 'utils/hooks/useReactSearch';
import { PubSub } from 'utils/pubsub';
import { StartViewProps } from '../types';

const helpText =
    `Logging in allows you to vote, save favorites, and contribute to the community.\n\nChoose **WALLET** if you are on a browser with a supported extension. This will not cost any money, but requires the signing of a message to verify that you own the wallet. Wallets will be utilized in the future to support user donations and execute routines tied to smart contracts.\n\nChoose **EMAIL** if you are on mobile or do not have a Nami account. A wallet can be associated with your account later.\n\nChoose **ENTER AS GUEST** if you only want to view the site or execute existing routines.`

const buttonProps: SxProps = {
    height: '4em',
}

const emailTitleId = 'email-login-dialog-title';

export const StartView = ({
    display = 'page',
    session,
}: StartViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    const search = useReactSearch();
    const { redirect, verificationCode } = useMemo(() => ({
        redirect: typeof search.redirect === 'string' ? search.redirect : undefined,
        verificationCode: typeof search.verificationCode === 'string' ? search.verificationCode : undefined,
    }), [search]);

    const [emailLogIn] = useCustomMutation<Session, EmailLogInInput>(authEmailLogIn);
    const [guestLogIn] = useCustomMutation<Session, undefined>(authGuestLogIn);
    // Handles email authentication popup
    const [emailPopupOpen, setEmailPopupOpen] = useState(false);
    const [popupForm, setPopupForm] = useState<Forms>(Forms.LogIn);
    const handleFormChange = useCallback((type: Forms = Forms.LogIn) => type !== popupForm && setPopupForm(type), [popupForm]);
    const [Form, formTitle] = useMemo(() => {
        switch (popupForm) {
            case Forms.ForgotPassword:
                return [ForgotPasswordForm, t('ForgotPassword')];
            case Forms.LogIn:
                return [LogInForm, t('LogIn')];
            case Forms.ResetPassword:
                return [ResetPasswordForm, t('ResetPassword')];
            case Forms.SignUp:
                return [SignUpForm, t('SignUp')];
            default:
                return [LogInForm, t('LogIn')];
        }
    }, [popupForm, t])

    /**
     * If verification code supplied
     */
    useEffect(() => {
        if (verificationCode) {
            // If still logged in, call emailLogIn right away
            if (userId) {
                mutationWrapper<Session, EmailLogInInput>({
                    mutation: emailLogIn,
                    input: { verificationCode },
                    onSuccess: (data) => {
                        PubSub.get().publishSnack({ messageKey: 'EmailVerified', severity: 'Success' });
                        PubSub.get().publishSession(data);
                        setLocation(redirect ?? LINKS.Home)
                    },
                    onError: (response) => {
                        if (hasErrorCode(response, 'MustResetPassword')) {
                            PubSub.get().publishAlertDialog({
                                messageKey: 'ChangePasswordBeforeLogin',
                                buttons: [
                                    { labelKey: 'Ok', onClick: () => { setLocation(redirect ?? LINKS.Home) } },
                                ]
                            });
                        }
                    }
                })
            }
            // Otherwise, open log in form
            else {
                setEmailPopupOpen(true);
                setPopupForm(Forms.LogIn);
            }
        }
    }, [emailLogIn, verificationCode, redirect, setLocation, userId])

    // Wallet provider popups
    const [connectOpen, setConnectOpen] = useState(false);
    const [installOpen, setInstallOpen] = useState(false);
    const openWalletConnectDialog = useCallback(() => { setConnectOpen(true) }, []);
    const openWalletInstallDialog = useCallback(() => { setInstallOpen(true) }, []);

    const toEmailLogIn = useCallback(() => {
        setPopupForm(Forms.LogIn);
        setEmailPopupOpen(true);
    }, [])

    const closeEmailPopup = useCallback(() => setEmailPopupOpen(false), [])

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
                messageKey: 'WalletProviderNotFoundDetails',
                buttons: [
                    { labelKey: 'TryAgain', onClick: openWalletConnectDialog },
                    { labelKey: 'InstallWallet', onClick: openWalletInstallDialog },
                    { labelKey: 'EmailLogin', onClick: toEmailLogIn },
                ]
            });
            return;
        }
        // Validate wallet
        const walletCompleteResult = await validateWallet(providerKey);
        if (walletCompleteResult?.session) {
            PubSub.get().publishSnack({ messageKey: 'WalletVerified', severity: 'Success' })
            PubSub.get().publishSession(walletCompleteResult.session)
            // Redirect to main dashboard
            setLocation(walletCompleteResult?.firstLogIn ? LINKS.Welcome : (redirect ?? LINKS.Home));
            // Request user to enable notifications
            subscribeUserToPush();
        }
    }, [openWalletConnectDialog, openWalletInstallDialog, toEmailLogIn, setLocation, redirect])

    const requestGuestToken = useCallback(() => {
        mutationWrapper<Session, never>({
            mutation: guestLogIn,
            onSuccess: (data) => {
                PubSub.get().publishSession(data)
                setLocation(redirect ?? LINKS.Welcome);
            },
        })
    }, [guestLogIn, setLocation, redirect]);

    const closeWalletConnectDialog = useCallback((providerKey: string | null) => {
        setConnectOpen(false);
        if (providerKey) {
            walletLogin(providerKey);
        }
    }, [walletLogin]);

    const closeWalletInstallDialog = useCallback(() => { setInstallOpen(false) }, []);

    return (
        <>
            {/* Dialogs */}
            <WalletSelectDialog
                handleOpenInstall={openWalletInstallDialog}
                open={connectOpen}
                onClose={closeWalletConnectDialog}
                zIndex={200}
            />
            <WalletInstallDialog
                open={installOpen}
                onClose={closeWalletInstallDialog}
                zIndex={connectOpen ? 201 : 200}
            />
            <Dialog
                open={emailPopupOpen}
                disableScrollLock={true}
                onClose={closeEmailPopup}
                aria-labelledby={emailTitleId}
            >
                <DialogTitle
                    id={emailTitleId}
                    title={formTitle}
                    onClose={closeEmailPopup}
                />
                <Box sx={{ padding: 1 }}>
                    <Form onFormChange={handleFormChange} />
                </Box>
            </Dialog>
            {/* App bar */}
            <TopBar
                display={display}
                onClose={() => { }}
                session={session}
                titleData={{
                    titleKey: 'Start',
                }}
            />
            {/* Main content */}
            <Box sx={{
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                textAlign: 'center',
                marginTop: 4,
            }}>
                <Box sx={{
                    width: 'min(calc(100vw - 16px), 500px)',
                }}>
                    <Box sx={{
                        display: 'flex',
                        justifyContent: 'center',
                        alignItems: 'center',
                    }}>
                        <Typography
                            variant="h6"
                            sx={{
                                display: 'inline-block',
                            }}
                        >
                            {t('PleaseSelectLogInMethod')}
                        </Typography>
                        <HelpButton markdown={helpText} />
                    </Box>
                    <Stack
                        direction="column"
                        spacing={2}
                    >
                        <Button fullWidth onClick={openWalletConnectDialog} sx={{ ...buttonProps }}>{t('Wallet')}</Button>
                        <Button fullWidth onClick={toEmailLogIn} sx={{ ...buttonProps }}>{t('Email')}</Button>
                        <Button fullWidth onClick={requestGuestToken} sx={{ ...buttonProps }}>{t('EnterAsGuest')}</Button>
                    </Stack>
                </Box>
            </Box>
        </>
    );
}
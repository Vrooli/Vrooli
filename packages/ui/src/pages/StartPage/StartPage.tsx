// Provides 3 options for entering the main application:
// 1. Wallet authentication - Quickest and safest method, but requires an extension which supports it
// 2. Email authentication - Requires email and password. Pretty safe if using password manager, 
// but wallet must be connected before performing any blockchain-related activities
// 3. Guest pass - Those who don't want to make an account can still view and run routines, but will not
// be able to utilize the full functionality of the service
import { useLocation } from '@shared/route';
import {
    Box,
    Button,
    Dialog,
    Stack,
    SxProps,
    Typography,
} from '@mui/material';
import { Forms, PubSub, useReactSearch } from 'utils';
import { APP_LINKS, CODE } from '@shared/consts';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { hasWalletExtension, validateWallet } from 'utils/authentication/walletIntegration';
import { DialogTitle, HelpButton, SnackSeverity, WalletInstallDialog, WalletSelectDialog } from 'components';
import {
    LogInForm,
    ForgotPasswordForm,
    SignUpForm,
    ResetPasswordForm,
} from 'forms';
import { emailLogInMutation, guestLogInMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
import { emailLogIn, emailLogInVariables } from 'graphql/generated/emailLogIn';
import { StartPageProps } from 'pages/types';
import { hasErrorCode } from 'graphql/utils';
import { guestLogIn } from 'graphql/generated/guestLogIn';

const helpText =
    `Logging in allows you to vote, save favorites, and contribute to the community.

Choose **WALLET** if you are on a browser with a supported extension. This will not cost any money, but requires the signing of a message to verify that you own the wallet. Wallets will be utilized in the future to support user donations and execute routines tied to smart contracts.

Choose **EMAIL** if you are on mobile or do not have a Nami account. A wallet can be associated with your account later.

Choose **ENTER AS GUEST** if you only want to view the site or execute existing routines.`

const buttonProps: SxProps = {
    height: '4em',
}

const emailTitleAria = 'email-login-dialog-title';

export const StartPage = ({
    session,
}: StartPageProps) => {
    const [, setLocation] = useLocation();
    const search = useReactSearch();
    const { redirect, verificationCode } = useMemo(() => ({
        redirect: typeof search.redirect === 'string' ? search.redirect : undefined,
        verificationCode: typeof search.verificationCode === 'string' ? search.verificationCode : undefined,
    }), [search]);

    const [emailLogIn] = useMutation<emailLogIn, emailLogInVariables>(emailLogInMutation);
    const [guestLogIn] = useMutation<guestLogIn, any>(guestLogInMutation);
    // Handles email authentication popup
    const [emailPopupOpen, setEmailPopupOpen] = useState(false);
    const [popupForm, setPopupForm] = useState<Forms>(Forms.LogIn);
    const handleFormChange = useCallback((type: Forms = Forms.LogIn) => type !== popupForm && setPopupForm(type), [popupForm]);
    const [Form, formTitle] = useMemo(() => {
        switch (popupForm) {
            case Forms.ForgotPassword:
                return [ForgotPasswordForm, 'Forgot Password'];
            case Forms.LogIn:
                return [LogInForm, 'Log In'];
            case Forms.ResetPassword:
                return [ResetPasswordForm, 'Reset Password'];
            case Forms.SignUp:
                return [SignUpForm, 'Sign Up'];
            default:
                return [LogInForm, 'Log In'];
        }
    }, [popupForm])

    /**
     * If verification code supplied
     */
    useEffect(() => {
        if (verificationCode) {
            // If session is already verified, call guest log in
            if (session.id) {
                mutationWrapper({
                    mutation: emailLogIn,
                    input: { verificationCode },
                    onSuccess: (data) => {
                        PubSub.get().publishSnack({ message: 'Email verified!', severity: SnackSeverity.Success });
                        PubSub.get().publishSession(data.emailLogIn);
                        setLocation(redirect ?? APP_LINKS.Home)
                    },
                    onError: (response) => {
                        if (hasErrorCode(response, CODE.MustResetPassword)) {
                            PubSub.get().publishAlertDialog({
                                message: 'Before signing in, please follow the link sent to your email to change your password.',
                                buttons: [
                                    { text: 'Ok', onClick: () => { setLocation(redirect ?? APP_LINKS.Home) } },
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
    }, [emailLogIn, verificationCode, redirect, session.id, setLocation])

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
                message: 'Wallet provider not found. Please verify that you are using a Chromium browser (e.g. Chrome, Brave), and that the Nami wallet extension is installed.',
                buttons: [
                    { text: 'Try Again', onClick: openWalletConnectDialog },
                    { text: 'Install Wallet', onClick: openWalletInstallDialog },
                    { text: 'Email Login', onClick: toEmailLogIn },
                ]
            });
            return;
        }
        // Validate wallet
        const walletCompleteResult = await validateWallet(providerKey);
        if (walletCompleteResult?.session) {
            PubSub.get().publishSnack({ message: 'Wallet verified.', severity: SnackSeverity.Success })
            // Set actor role
            PubSub.get().publishSession(walletCompleteResult.session)
            // Redirect to main dashboard
            setLocation(walletCompleteResult?.firstLogIn ? APP_LINKS.Welcome : (redirect ?? APP_LINKS.Home));
        }
    }, [openWalletConnectDialog, openWalletInstallDialog, toEmailLogIn, setLocation, redirect])

    const requestGuestToken = useCallback(() => {
        mutationWrapper({
            mutation: guestLogIn,
            onSuccess: () => {
                let theme: string = 'light';
                if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) theme = 'dark';
                PubSub.get().publishSession({
                    isLoggedIn: true,
                    theme,
                })
                setLocation(redirect ?? APP_LINKS.Welcome);
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
        <Box
            sx={{
                padding: '1em',
                paddingTop: '20vh',
                minHeight: '100vh', //Fullscreen
            }}
        >
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
            <Box
                sx={{
                    width: 'min(100%, 500px)',
                    margin: 'auto',
                    paddingTop: { xs: '5vh', sm: '20vh' },
                }}
            >
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
                        Please select your log in method
                    </Typography>
                    <HelpButton markdown={helpText} />
                </Box>
                <Stack
                    direction="column"
                    spacing={2}
                >
                    <Button fullWidth onClick={openWalletConnectDialog} sx={{ ...buttonProps }}>Wallet</Button>
                    <Button fullWidth onClick={toEmailLogIn} sx={{ ...buttonProps }}>Email</Button>
                    <Button fullWidth onClick={requestGuestToken} sx={{ ...buttonProps }}>Enter As Guest</Button>
                </Stack>
            </Box>
            <Dialog
                open={emailPopupOpen}
                disableScrollLock={true}
                onClose={closeEmailPopup}
                aria-labelledby={emailTitleAria}
            >
                <DialogTitle
                    ariaLabel={emailTitleAria}
                    title={formTitle}
                    onClose={closeEmailPopup}
                />
                <Box sx={{ padding: 1 }}>
                    <Form onFormChange={handleFormChange} />
                </Box>
            </Dialog>
        </Box>
    );
}
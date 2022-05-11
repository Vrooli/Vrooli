// Provides 3 options for entering the main application:
// 1. Wallet authentication - Quickest and safest method, but requires an extension which supports it
// 2. Email authentication - Requires email and password. Pretty safe if using password manager, 
// but wallet must be connected before performing any blockchain-related activities
// 3. Guest pass - Those who don't want to make an account can still view and run routines, but will not
// be able to utilize the full functionality of the service
import { useLocation } from 'wouter';
import {
    Box,
    Button,
    Dialog,
    ListItem,
    ListItemText,
    Stack,
    SxProps,
    Typography,
    useTheme,
} from '@mui/material';
import { Forms, Pubs, useReactSearch } from 'utils';
import { APP_LINKS, CODE } from '@local/shared';
import { MouseEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { hasWalletExtension, validateWallet, WalletProvider, walletProviderInfo } from 'utils/walletIntegration';
import { ROLES } from '@local/shared';
import { HelpButton } from 'components';
import {
    LogInForm,
    ForgotPasswordForm,
    SignUpForm,
    ResetPasswordForm,
} from 'forms';
import { emailLogInMutation, guestLogInMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { emailLogIn } from 'graphql/generated/emailLogIn';
import { StartPageProps } from 'pages/types';

const helpText =
    `Logging in allows you to vote, save favorites, and contribute to the community.

Choose **WALLET** if you are on a browser with a supported extension (currently [Eternl (aka CCVault.io)](https://ccvault.io/app/mainnet/faq), [Nami](https://namiwallet.io/), and [Yoroi](https://yoroi-wallet.com/#/)). This will not cost any money, but requires the signing of a message to verify that you own the wallet. Wallets will be utilized in the future to support user donations and execute routines tied to smart contracts.

Choose **EMAIL** if you are on mobile or do not have a Nami account. A wallet can be associated with your account later.

Choose **ENTER AS GUEST** if you only want to view the site or execute existing routines.`

const buttonProps: SxProps = {
    height: '4em',
}

export const StartPage = ({
    session,
}: StartPageProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { redirect, code: verificationCode } = useReactSearch();

    const [emailLogIn] = useMutation<emailLogIn>(emailLogInMutation);
    const [guestLogIn] = useMutation<any>(guestLogInMutation);
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
                    onSuccess: (response) => { 
                        PubSub.publish(Pubs.Snack, { message: 'Email verified!' });
                        PubSub.publish(Pubs.Session, response.data.emailLogIn); 
                        setLocation(redirect ?? APP_LINKS.Home) 
                    },
                    onError: (response) => {
                        if (Array.isArray(response.graphQLErrors) && response.graphQLErrors.some(e => e.extensions.code === CODE.MustResetPassword.code)) {
                            PubSub.publish(Pubs.AlertDialog, {
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

    // Wallet provider select popup
    const [providerOpen, setProviderPopupOpen] = useState(false);
    const [walletDialogFor, setWalletDialogFor] = useState<'connect' | 'download'>('connect');
    const openWalletConnectDialog = useCallback((ev: MouseEvent<HTMLButtonElement>) => {
        setWalletDialogFor('connect');
        setProviderPopupOpen(true);
        ev.preventDefault();
    }, []);
    const openWalletDownloadDialog = useCallback((ev: MouseEvent<HTMLButtonElement>) => {
        setWalletDialogFor('download');
        setProviderPopupOpen(true);
        ev.preventDefault();
    }, []);

    // Opens link to install wallet extension
    const downloadExtension = useCallback((provider: WalletProvider) => {
        const extensionUrl = walletProviderInfo[provider].extensionUrl;
        window.open(extensionUrl, '_blank', 'noopener,noreferrer');
    }, [])


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
    const walletLogin = useCallback(async (provider: WalletProvider) => {
        // Check if wallet extension installed
        if (!hasWalletExtension(provider)) {
            PubSub.publish(Pubs.AlertDialog, {
                message: 'Wallet provider not found. Please verify that you are using a Chromium browser (e.g. Chrome, Brave), and that the Nami wallet extension is installed.',
                buttons: [
                    { text: 'Try Again', onClick: walletLogin },
                    { text: 'Install Wallet', onClick: openWalletDownloadDialog },
                    { text: 'Email Login', onClick: toEmailLogIn },
                ]
            });
            return;
        }
        // Validate wallet
        const walletCompleteResult = await validateWallet(provider);
        if (walletCompleteResult) {
            PubSub.publish(Pubs.Snack, { message: 'Wallet verified.' })
            // Set actor role
            PubSub.publish(Pubs.Session, walletCompleteResult?.session)
            // Redirect to main dashboard
            setLocation(walletCompleteResult?.firstLogIn ? APP_LINKS.Welcome : (redirect ?? APP_LINKS.Home));
        }
    }, [openWalletDownloadDialog, setLocation, redirect, toEmailLogIn])

    const requestGuestToken = useCallback(() => {
        mutationWrapper({
            mutation: guestLogIn,
            onSuccess: () => {
                PubSub.publish(Pubs.Session, { roles: [{ role: { title: ROLES.Guest } }] })
                setLocation(redirect ?? APP_LINKS.Welcome);
            },
        })
    }, [guestLogIn, setLocation, redirect]);

    const handleProviderClose = useCallback(() => {
        setProviderPopupOpen(false);
    }, [])
    const handleWalletDialogSelect = useCallback((selected: WalletProvider) => {
        if (walletDialogFor === 'connect') {
            walletLogin(selected);
        } else if (walletDialogFor === 'download') {
            downloadExtension(selected);
        }
        handleProviderClose();
    }, [walletDialogFor, walletLogin, downloadExtension, handleProviderClose])

    return (
        <Box
            sx={{
                padding: '1em',
                paddingTop: '20vh',
                minHeight: '100vh', //Fullscreen
            }}
        >
            <Dialog
                open={providerOpen}
                disableScrollLock={true}
                onClose={handleProviderClose}
            >
                <Box
                    sx={{
                        width: '100',
                        borderRadius: '4px 4px 0 0',
                        padding: 1,
                        paddingLeft: 2,
                        paddingRight: 2,
                        background: palette.primary.dark,
                        color: 'white',
                    }}
                >
                    <Typography variant="h6" textAlign="center">Select Wallet</Typography>
                </Box>
                {Object.values(walletProviderInfo).map((o, index) => (
                    <ListItem button onClick={() => handleWalletDialogSelect(o.enum)} key={index}>
                        <ListItemText primary={o.label} />
                    </ListItem>
                ))}
            </Dialog>
            <Box
                sx={{
                    width: 'min(100%, 500px)',
                    margin: 'auto',
                    paddingTop: { xs: '5vh', sm: '20vh' },
                }}
            >
                <Box textAlign="center">
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
            >
                <Box
                    sx={{
                        width: '100',
                        borderRadius: '4px 4px 0 0',
                        padding: 1,
                        background: palette.primary.dark,
                        color: 'white',
                    }}
                >
                    <Typography variant="h6" textAlign="center">{formTitle}</Typography>
                </Box>
                <Box sx={{ padding: 1 }}>
                    <Form onFormChange={handleFormChange} />
                </Box>
            </Dialog>
        </Box>
    );
}
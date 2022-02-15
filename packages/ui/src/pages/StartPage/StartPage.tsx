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
} from '@mui/material';
import { Forms, Pubs } from 'utils';
import { APP_LINKS } from '@local/shared';
import { MouseEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { hasWalletExtension, validateWallet, WalletProvider, walletProviderInfo } from 'utils/walletIntegration';
import { CommonProps } from 'types';
import { ROLES } from '@local/shared';
import { HelpButton } from 'components';
import {
    LogInForm,
    ForgotPasswordForm,
    SignUpForm,
    ResetPasswordForm,
} from 'forms';
import { guestLogInMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils/wrappers';
import helpMarkdown from './startHelp.md';

const buttonProps: SxProps = {
    height: '4em',
}

export const StartPage = ({
    onSessionUpdate
}: Pick<CommonProps, 'onSessionUpdate'>) => {
    const [, setLocation] = useLocation();
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

    const [helpText, setHelpText] = useState<string>('');
    useEffect(() => {
        fetch(helpMarkdown).then((r) => r.text()).then((text) => { setHelpText(text) });
    }, []);

    // Wallet connect popup
    const [walletOptionPopupOpen, setWalletOptionPopupOpen] = useState(false);
    const [walletDialogFor, setWalletDialogFor] = useState<'connect' | 'download'>('connect');
    const openWalletConnectDialog = useCallback((ev: MouseEvent<HTMLButtonElement>) => {
        setWalletDialogFor('connect');
        setWalletOptionPopupOpen(true);
        ev.preventDefault();
    }, []);
    const openWalletDownloadDialog = useCallback((ev: MouseEvent<HTMLButtonElement>) => {
        setWalletDialogFor('download');
        setWalletOptionPopupOpen(true);
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
                message: 'Wallet not found. Please verify that you are using a Chromium browser (e.g. Chrome, Brave), and that the Nami wallet extension is installed.',
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
        console.log('wallet validation', walletCompleteResult?.session);
        if (walletCompleteResult) {
            PubSub.publish(Pubs.Snack, { message: 'Wallet verified.' })
            // Set actor role
            onSessionUpdate(walletCompleteResult?.session)
            // Redirect to main dashboard
            setLocation(walletCompleteResult?.firstLogIn ? APP_LINKS.Welcome : APP_LINKS.Home);
        }
    }, [downloadExtension, setLocation, onSessionUpdate, toEmailLogIn])

    const requestGuestToken = useCallback(() => {
        mutationWrapper({
            mutation: guestLogIn,
            onSuccess: () => {
                onSessionUpdate({ roles: [{ role: { title: ROLES.Guest } }] });
                setLocation(APP_LINKS.Welcome);
            },
        })
    }, [guestLogIn, setLocation, onSessionUpdate]);

    const handleWalletDialogSelect = useCallback((selected: WalletProvider) => {
        console.log('handleWalletDialogSelect', selected);
        if (walletDialogFor === 'connect') {
            walletLogin(selected);
        } else if (walletDialogFor === 'download') {
            downloadExtension(selected);
        }
        handleWalletDialogClose();
    }, []);
    const handleWalletDialogClose = useCallback(() => {
        setWalletOptionPopupOpen(false);
    }, []);

    return (
        <Box
            sx={{
                padding: '1em',
                paddingTop: '20vh',
                minHeight: '100vh', //Fullscreen
            }}
        >
            <Dialog
                open={walletOptionPopupOpen}
                disableScrollLock={true}
                onClose={handleWalletDialogClose}
            >
                <Box
                    sx={{
                        width: '100',
                        borderRadius: '4px 4px 0 0',
                        padding: 1,
                        paddingLeft: 2,
                        paddingRight: 2,
                        background: (t) => t.palette.primary.dark,
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
                        background: (t) => t.palette.primary.dark,
                        color: 'white',
                    }}
                >
                    <Typography variant="h6" textAlign="center">{formTitle}</Typography>
                </Box>
                <Box sx={{ padding: 1 }}>
                    <Form
                        onSessionUpdate={onSessionUpdate}
                        onFormChange={handleFormChange}
                    />
                </Box>
            </Dialog>
        </Box>
    );
}
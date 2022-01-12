// Provides 3 options for entering the main application:
// 1. Wallet authentication - Quickest and safest method, but requires Nami extension
// 2. Email authentication - Requires email and password. Pretty safe if using password manager, 
// but wallet must be connected before performing any blockchain-related activities
// 3. Guest pass - Those who don't want to make an account can still view and run routines, but will not
// be able to utilize the full functionality of the service
import { useLocation } from 'wouter';
import {
    Box,
    Button,
    Dialog,
    Stack,
    SxProps,
    Typography,
} from '@mui/material';
import { Forms, Pubs } from 'utils';
import { APP_LINKS } from '@local/shared';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { hasWalletExtension, validateWallet } from 'utils/walletIntegration';
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
import { centeredText } from 'styles';
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

    // Parse help text from markdown
    useEffect(() => {
        fetch(helpMarkdown)
            .then((response) => response.text())
            .then((text) => {
                setHelpText(text);
            });
    }, []);

    // Opens link to install wallet extension
    const downloadExtension = useCallback(() => {
        const extensionLink = `https://chrome.google.com/webstore/detail/nami-wallet/lpfcbjknijpeeillifnkikgncikgfhdo`;
        window.open(extensionLink, '_blank', 'noopener,noreferrer');
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
    const walletLogin = useCallback(async () => {
        // Check if wallet extension installed
        if (!hasWalletExtension()) {
            PubSub.publish(Pubs.AlertDialog, {
                message: 'Wallet not found. Please verify that you are using a Chromium browser (e.g. Chrome, Brave), and that the Nami wallet extension is installed.',
                buttons: [
                    { text: 'Try Again', onClick: walletLogin },
                    { text: 'Install Nami', onClick: downloadExtension },
                    { text: 'Email Login', onClick: toEmailLogIn },
                ]
            });
            return;
        }
        // Validate wallet
        const session = await validateWallet();
        console.log('wallet validation', session);
        if (session) {
            PubSub.publish(Pubs.Snack, { message: 'Wallet verified.' })
            // Set actor role
            onSessionUpdate(session)
            // Redirect to main dashboard
            setLocation(APP_LINKS.Home);
        }
    }, [downloadExtension, setLocation, onSessionUpdate, toEmailLogIn])

    const requestGuestToken = useCallback(() => {
        mutationWrapper({
            mutation: guestLogIn,
            onSuccess: () => {
                onSessionUpdate({ roles: [{ role: { title: ROLES.Guest } }] });
                setLocation(APP_LINKS.Home);
            },
        })
    }, [guestLogIn, setLocation, onSessionUpdate]);

    return (
        <Box
            sx={{
                padding: '1em',
                paddingTop: '20vh',
                border: '2px solid brown',
                minHeight: '100vh', //Fullscreen
            }}
        >
            <Box
                sx={{
                    width: 'min(100%, 500px)',
                    margin: 'auto',
                    paddingTop: { xs: '5vh', sm: '20vh' },
                }}
            >
                <Box sx={{...centeredText}}>
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
                    <Button fullWidth onClick={walletLogin} sx={{ ...buttonProps }}>Wallet (Nami)</Button>
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
                    <Typography variant="h6" sx={{ ...centeredText }}>{formTitle}</Typography>
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
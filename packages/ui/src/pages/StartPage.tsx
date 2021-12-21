// Provides 3 options for entering the main application:
// 1. Wallet authentication - Quickest and safest method, but requires Nami extension
// 2. Email authentication - Requires email and password. Pretty safe if using password manager, 
// but wallet must be connected before performing any blockchain-related activities
// 3. Guest pass - Those who don't want to make an account can still view and run routines, but will not
// be able to utilize the full functionality of the service
import { useNavigate } from 'react-router-dom';
import {
    Button,
    Dialog,
    Grid,
    Typography,
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { Theme } from '@material-ui/core';
import { FORMS, PUBS } from 'utils';
import { APP_LINKS } from '@local/shared';
import { useCallback, useMemo, useState } from 'react';
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

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        padding: '1em',
        paddingTop: '20vh',
        border: '2px solid brown',
        minHeight: '100vh', //Fullscreen
    },
    horizontal: {
        textAlign: 'center',
    },
    prompt: {
        display: 'inline-block',
    },
    buttonContainer: {
        maxWidth: '500px',
        margin: 'auto', //Horizontal align
    },
    option: {
        height: '4em',
    },
}));

export const StartPage = ({
    onSessionUpdate
}: Pick<CommonProps, 'onSessionUpdate'>) => {
    const classes = useStyles();
    const navigate = useNavigate();
    const [guestLogIn] = useMutation<any>(guestLogInMutation);
    // Handles email authentication popup
    const [emailPopupOpen, setEmailPopupOpen] = useState(false);
    const [popupForm, setPopupForm] = useState<FORMS>(FORMS.LogIn);
    const handleFormChange = useCallback((type: FORMS = FORMS.LogIn) => type !== popupForm && setPopupForm(type), [popupForm]);
    const Form = useMemo(() => {
        switch (popupForm) {
            case FORMS.ForgotPassword:
                return ForgotPasswordForm
            case FORMS.LogIn:
                return LogInForm
            case FORMS.ResetPassword:
                return ResetPasswordForm
            case FORMS.SignUp:
                return SignUpForm
            default:
                return LogInForm
        }
    }, [popupForm])

    // Opens link to install wallet extension
    const downloadExtension = useCallback(() => {
        const extensionLink = `https://chrome.google.com/webstore/detail/nami-wallet/lpfcbjknijpeeillifnkikgncikgfhdo`;
        window.open(extensionLink, '_blank', 'noopener,noreferrer');
    }, [])


    const toEmailLogIn = useCallback(() => {
        setPopupForm(FORMS.LogIn);
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
            PubSub.publish(PUBS.AlertDialog, {
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
            PubSub.publish(PUBS.Snack, { message: 'Wallet verified.' })
            // Set actor role
            onSessionUpdate(session)
            // Redirect to main dashboard
            navigate(APP_LINKS.Home);
        }
    }, [downloadExtension, navigate, onSessionUpdate, toEmailLogIn])

    const requestGuestToken = useCallback(() => {
        mutationWrapper({
            mutation: guestLogIn,
            onSuccess: () => {
                onSessionUpdate({ roles: [{ role: { title: ROLES.Guest } }] });
                navigate(APP_LINKS.Home);
            },
        })
    }, [guestLogIn, navigate, onSessionUpdate]);

    return (
        <div className={classes.root}>
            <div className={classes.horizontal}>
                <Typography className={classes.prompt} variant="h6">Please select your log in method</Typography>
                <HelpButton title={'boop'} />
            </div>
            <Grid className={classes.buttonContainer} container spacing={2}>
                <Grid item xs={12}>
                    <Button
                        className={classes.option}
                        fullWidth
                        onClick={walletLogin}
                    >Wallet (Nami)</Button>
                </Grid>
                <Grid item xs={12}>
                    <Button
                        className={classes.option}
                        fullWidth
                        onClick={toEmailLogIn}
                    >Email</Button>
                </Grid>
                <Grid item xs={12}>
                    <Button
                        className={classes.option}
                        fullWidth
                        onClick={requestGuestToken}
                    >Enter As Guest</Button>
                </Grid>
            </Grid>
            <Dialog open={emailPopupOpen} onClose={closeEmailPopup}>
                <Form onSessionUpdate={onSessionUpdate} onFormChange={handleFormChange} />
            </Dialog>
        </div>
    );
}
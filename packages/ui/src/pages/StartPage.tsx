import { useHistory } from 'react-router-dom';
import {
    Button,
    Container,
    Grid,
    Typography,
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { Theme } from '@material-ui/core';
import { LINKS, PUBS } from 'utils';
import React, { useCallback, useState } from 'react';
import { connectWallet } from 'utils/connectWallet';
import { CommonProps } from 'types';
import { ROLES } from '@local/shared';
import { HelpButton } from 'components';

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

const LOGIN_METHODS = {
    Wallet: 'wallet',
    Email: 'email',
}

export const StartPage = ({
    onSessionUpdate
}: Pick<CommonProps, 'onSessionUpdate'>) => {
    const classes = useStyles();
    const history = useHistory();
    const [loginMethod, setLoginMethod] = useState<string | null>(null);

    const downloadExtension = useCallback(() => {
        const extensionLink = `https://chrome.google.com/webstore/detail/nami-wallet/lpfcbjknijpeeillifnkikgncikgfhdo`;
        window.open(extensionLink, '_blank', 'noopener,noreferrer');
    }, [])

    const toEmailLogin = useCallback(() => setLoginMethod(LOGIN_METHODS.Email), [])

    const walletLogin = useCallback(async () => {
        console.log('[] useeffect', window.cardano);
        const success = await connectWallet();
        if (success) {
            // Set customer role
            onSessionUpdate({ roles: [{ role: ROLES.Customer }] })
            // Redirect to main dashboard
            history.push(LINKS.Home);
        } else {
            // Alert that login failed. Provide options to try again, download extension, or login via email
            PubSub.publish(PUBS.AlertDialog, {
                message: 'Wallet log in failed. Please verify that you are using a Chromium browser (e.g. Chrome, Brave), and that the Nami wallet extension is installed.',
                buttons: [
                    { text: 'Try Again', onClick: walletLogin },
                    { text: 'Install Nami', onClick: downloadExtension },
                    { text: 'Email Login', onClick: toEmailLogin },
                ]
            });
        }
    }, [downloadExtension, history, toEmailLogin])

    const guestLogin = useCallback(() => {
        // Set user role as guest
        onSessionUpdate({ roles: [{ role: ROLES.Guest }] })
        // Redirect to home dashboard
        history.push(LINKS.Home)
    }, [history, onSessionUpdate]);

    return (
        <div className={classes.root}>
            <div className={classes.horizontal}>
            <Typography className={classes.prompt} variant="h6">Please select your login method</Typography>
            <HelpButton title={'boop'}/>
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
                        onClick={toEmailLogin}
                    >Email</Button>
                </Grid>
                <Grid item xs={12}>
                    <Button
                        className={classes.option}
                        fullWidth
                        onClick={guestLogin}
                    >Enter As Guest</Button>
                </Grid>
            </Grid>
        </div>
    );
}
import { useHistory } from 'react-router-dom';
import {
    Button,
    Grid,
    Typography,
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { LINKS } from 'utils';
import { formStyles } from './styles';
import { CommonProps } from 'types';
import React, { useCallback, useState } from 'react';
import { connectWallet } from 'utils/connectWallet';

const useStyles = makeStyles(formStyles);

export const LogInForm = ({
    onSessionUpdate
}: Pick<CommonProps, 'onSessionUpdate'>) => {
    const classes = useStyles();
    const history = useHistory();
    const [loginMethod, setLoginMethod] = useState<string | null>(null);

    const downloadExtension = useCallback(() => {
        history.push(`https://chrome.google.com/webstore/detail/nami-wallet/lpfcbjknijpeeillifnkikgncikgfhdo`);
    }, [history])

    const connectToWallet = useCallback(async () => {
        console.log('[] useeffect', window.cardano);
        const success = await connectWallet();
        if (success) {
            history.push(LINKS.Home);
        }
    }, [history])

    const toWalletLogin = useCallback(() => setLoginMethod('wallet'), [])
    const toEmailLogin = useCallback(() => setLoginMethod('email'), [])

    let form;
    // Display login instructions and links for wallet login
    if (loginMethod === 'wallet') {
        form = <React.Fragment>
            <Grid item xs={12} sm={6}>
                <Button
                    fullWidth
                    color="secondary"
                    className={classes.submit}
                    onClick={downloadExtension}
                >
                    Download extension
                </Button>
            </Grid>
            <Grid item xs={12} sm={6}>
                <Button
                    fullWidth
                    type="submit"
                    color="secondary"
                    className={classes.submit}
                    onClick={connectToWallet}
                >
                    Connect wallet
                </Button>
            </Grid>
        </React.Fragment>
    }
    // Display traditional email login form
    else if (loginMethod === 'email') {

    }
    // Prompt for wallet or email login
    else {
        form = <React.Fragment>
            <Grid item xs={12}>
                <Button
                    fullWidth
                    color="secondary"
                    className={classes.submit}
                    onClick={toWalletLogin}
                >
                    Wallet Login
                </Button>
            </Grid>
            <Grid item xs={12}>
                <Typography>Or</Typography>
            </Grid>
            <Grid item xs={12}>
                <Button
                    fullWidth
                    color="secondary"
                    className={classes.submit}
                    onClick={toEmailLogin}
                >
                    Email Login
                </Button>
            </Grid>
        </React.Fragment>
    }

    return (
        <div>
            <Grid container spacing={2}>
                {form}
            </Grid>
        </div >
    );
}
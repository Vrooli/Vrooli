import { useHistory } from 'react-router-dom';
import {
    Button,
    Container,
    Grid,
    Typography,
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { Theme } from '@material-ui/core';
import { LINKS } from 'utils';
import React, { useCallback, useState } from 'react';
import { connectWallet } from 'utils/connectWallet';
import { CommonProps } from 'types';
import { ROLES } from '@local/shared';

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        padding: '1em',
        paddingTop: '20vh',
        color: 'red',
        border: '2px solid brown',
    },
    formHeader: {
        backgroundColor: theme.palette.primary.main,
        color: theme.palette.primary.contrastText,
        padding: '1em',
        textAlign: 'center'
    },
    container: {
        backgroundColor: theme.palette.background.paper,
        display: 'grid',
        position: 'relative',
        boxShadow: '0px 2px 4px -1px rgb(0 0 0 / 20%), 0px 4px 5px 0px rgb(0 0 0 / 14%), 0px 1px 10px 0px rgb(0 0 0 / 12%)',
        minWidth: '300px',
        maxWidth: 'min(100%, 700px)',
        borderRadius: '10px',
        overflow: 'hidden',
        left: '50%',
        transform: 'translateX(-50%)',
        marginBottom: '20px'
    },
    submit: {

    }
}));

export const StartPage = ({
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

    const toGuestLogin = useCallback(() => {
        // Set user role as guest
        onSessionUpdate({roles:[{role: ROLES.Guest}]})
        // Redirect to home dashboard
        history.push(LINKS.Home)
    }, [history, onSessionUpdate]);

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

    return (
        <div className={classes.root}>
            <div className={classes.container}>
                <Container className={classes.formHeader}>
                    <Typography variant="h3" >Start</Typography>
                </Container>
                <Grid container spacing={2}>
                    <Grid item xs={12}>
                        <Button
                            onClick={toWalletLogin}
                        >Connect via Wallet</Button>
                    </Grid>
                    <Grid item xs={12}>
                        <Button
                            onClick={toEmailLogin}
                        >Connect via Email</Button>
                    </Grid>
                    <Grid item xs={12}>
                        <Button
                            onClick={toGuestLogin}
                        >Enter as guest</Button>
                    </Grid>
                </Grid>
            </div>
        </div>
    );
}
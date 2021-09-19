import React, { useEffect, useState } from 'react';
import { IconButton, Button, Snackbar } from '@material-ui/core';
import { Close as CloseIcon } from '@material-ui/icons';
import { PUBS, PubSub } from 'utils';
import { makeStyles } from '@material-ui/styles';

const DEFAULT_STATE = {
    message: null,
    severity: 'default', // one of ('default', 'warning', 'error')
    data: null, // anything you'd like to print in development mode
    buttonText: null,
    buttonClicked: null,
    autoHideDuration: 5000,
    anchorOrigin: {
        vertical: 'bottom',
        horizontal: 'left',
    },
};

const useStyles = makeStyles((theme) => ({
    default: {
        background: theme.palette.primary.light,
        color: theme.palette.primary.contrastText,
    },
    warning: {
        background: theme.palette.warning.main,
        color: theme.palette.warning.contrastText,
    },
    error: {
        background: theme.palette.error.dark,
        color: theme.palette.error.contrastText,
    },
    button: {
        color: theme.palette.secondary.light,
    },
}));

function Snack() {
    const classes = useStyles();
    const [state, setState] = useState(DEFAULT_STATE)

    function getSnackClass(severity) {
        if (severity === 'error') return classes.error;
        if (severity === 'warning') return classes.warning;
        return classes.default;
    }
    let open = state.message !== null;

    useEffect(() => {
        let snackSub = PubSub.subscribe(PUBS.Snack, (_, o) => setState({ ...DEFAULT_STATE, ...o }));
        return () => PubSub.unsubscribe(snackSub);
    }, [])

    useEffect(() => {
        // Log snack errors if in development
        if (process.env.NODE_ENV === 'development' && state.data) {
            if (state.severity === 'error') console.error('Snack data', state.data);
            else console.info('Snack data', state.data);
        }
    }, [state])

    return (
        <Snackbar
        ContentProps={{
            classes: {
              root: getSnackClass(state.severity)
            }
          }}
            anchorOrigin={state.anchorOrigin}
            open={open}
            autoHideDuration={state.autoHideDuration}
            onClose={() => setState(DEFAULT_STATE)}
            message={Array.isArray(state.message) && state.message.length > 0 ? state.message[0] : state.message}
            action={
                <React.Fragment>
                    {state.buttonText ? <Button className={classes.button} variant="text" size="small" onClick={state.buttonClicked}>
                        {state.buttonText}
                    </Button> : null}
                    <IconButton size="small" aria-label="close" color="inherit" onClick={() => setState(DEFAULT_STATE)}>
                        <CloseIcon fontSize="small" />
                    </IconButton>
                </React.Fragment>
            }
        />
    );
}

export { Snack };
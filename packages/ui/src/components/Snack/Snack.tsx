import { useEffect, useState } from 'react';
import { IconButton, Button, Snackbar, Theme, SnackbarProps } from '@material-ui/core';
import { Close as CloseIcon } from '@material-ui/icons';
import { PUBS } from 'utils';
import PubSub from 'pubsub-js';
import { makeStyles } from '@material-ui/styles';
import { ValueOf } from '@local/shared';

export const SnackSeverity = {
    Default: "default",
    Warning: "warning",
    Error: "error",
}
// eslint-disable-next-line @typescript-eslint/no-redeclare
export type SnackSeverity = ValueOf<typeof SnackSeverity>;

class SnackState {
    message?: string;
    severity?: SnackSeverity = SnackSeverity.Default;
    data?: any = null; // anything you'd like to print in development mode
    buttonText?: string;
    buttonClicked?: (event?: any) => any = () => {};
    autoHideDuration?: number = 5000;
    anchorOrigin: SnackbarProps['anchorOrigin'] = {
        vertical: 'bottom',
        horizontal: 'left',
    };
}

const useStyles = makeStyles((theme: Theme) => ({
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
    const [state, setState] = useState<SnackState>(new SnackState());

    useEffect(() => {
        let snackSub = PubSub.subscribe(PUBS.Snack, (_, o) => setState({ ...(new SnackState()), ...o }));
        return () => { PubSub.unsubscribe(snackSub) };
    }, [])

    useEffect(() => {
        // Log snack errors if in development
        if (process.env.NODE_ENV === 'development' && state.data) {
            if (state.severity === SnackSeverity.Error) console.error('Snack data', state.data);
            else console.info('Snack data', state.data);
        }
    }, [state])

    const getSnackClass = (severity: SnackSeverity) => {
        if (severity === SnackSeverity.Error) return classes.error;
        if (severity === SnackSeverity.Warning) return classes.warning;
        return classes.default;
    }
    let open = Boolean(state.message);

    const resetState = () => setState(new SnackState());

    return (
        <Snackbar
        ContentProps={{
            classes: {
              root: getSnackClass(state.severity || SnackSeverity.Default)
            }
          }}
            anchorOrigin={state.anchorOrigin}
            open={open}
            autoHideDuration={state.autoHideDuration}
            onClose={resetState}
            message={Array.isArray(state.message) && state.message.length > 0 ? state.message[0] : state.message}
            action={
                <>
                    {state.buttonText ? <Button className={classes.button} variant="text" size="small" onClick={state.buttonClicked}>
                        {state.buttonText}
                    </Button> : null}
                    <IconButton size="small" aria-label="close" color="inherit" onClick={resetState}>
                        <CloseIcon fontSize="small" />
                    </IconButton>
                </>
            }
        />
    );
}

export { Snack };
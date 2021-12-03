import { useCallback, useEffect, useState } from 'react';
import {
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
    DialogTitle,
} from '@material-ui/core';
import { PUBS } from 'utils';
import PubSub from 'pubsub-js';

interface StateButton {
    text: string;
    onClick: (() => void) | null;
}

interface State {
    title?: string | null;
    message?: string | null;
    buttons: StateButton[];
}

const default_state: State = {
    title: null,
    message: null,
    buttons: [{text: 'Ok', onClick: null}],
};

const AlertDialog = () => {
    const [state, setState] = useState<State>(default_state)
    let open = state.title !== null || state.message !== null;

    useEffect(() => {
        let dialogSub = PubSub.subscribe(PUBS.AlertDialog, (_, o) => setState({...default_state, ...o}));
        return () => { PubSub.unsubscribe(dialogSub) };
    }, [])

    const handleClick = useCallback((action: (() => void) | null | undefined) => {
        if (action) action();
        setState(default_state);
    }, []);

    const resetState = useCallback(() => setState(default_state), []);

    return (
        <Dialog
            open={open}
            onClose={resetState}
            aria-labelledby="alert-dialog-title"
            aria-describedby="alert-dialog-description"
        >
            {state.title ? <DialogTitle id="alert-dialog-title">{state.title}</DialogTitle> : null}
            <DialogContent>
                <DialogContentText id="alert-dialog-description">
                    {state.message}
                </DialogContentText>
            </DialogContent>
            <DialogActions>
                {state?.buttons && state.buttons.length > 0 ? (
                    state.buttons.map((b:StateButton) => (
                        <Button onClick={() => handleClick(b.onClick)} color="secondary">
                            {b.text}
                        </Button>
                    ))
                ): null}
            </DialogActions>
        </Dialog>
    );
}

export { AlertDialog };
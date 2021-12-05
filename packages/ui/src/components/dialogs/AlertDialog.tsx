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
    onClick?: (() => void);
}

interface State {
    title?: string;
    message?: string;
    buttons: StateButton[];
}

const default_state: State = {
    buttons: [{ text: 'Ok' }],
};

const AlertDialog = () => {
    const [state, setState] = useState<State>(default_state)
    let open = Boolean(state.title) || Boolean(state.message);

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
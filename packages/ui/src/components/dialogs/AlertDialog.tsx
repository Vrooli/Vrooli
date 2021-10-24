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

interface State {
    title?: string | null;
    message?: string | null;
    firstButtonText?: string;
    firstButtonClicked?: (() => void) | null;
    secondButtonText?: string | null;
    secondButtonClicked?: (() => void) | null;
}

const default_state: State = {
    title: null,
    message: null,
    firstButtonText: 'Ok',
    firstButtonClicked: null,
    secondButtonText: null,
    secondButtonClicked: null,
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

    const clickFirst = useCallback(() => handleClick(state.firstButtonClicked), [handleClick, state.firstButtonClicked]);
    const clickSecond = useCallback(() => handleClick(state.secondButtonClicked), [handleClick, state.secondButtonClicked]);

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
                <Button onClick={clickFirst} color="secondary">
                    {state.firstButtonText}
                </Button>
                {state.secondButtonText ? (
                    <Button onClick={clickSecond} color="secondary">
                        {state.secondButtonText}
                    </Button>
                ) : null}
            </DialogActions>
        </Dialog>
    );
}

export { AlertDialog };
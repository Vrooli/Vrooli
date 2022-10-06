import { useCallback, useEffect, useState } from 'react';
import {
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
} from '@mui/material';
import { firstString, PubSub } from 'utils';
import { DialogTitle } from 'components';

interface StateButton {
    text: string;
    onClick?: (() => void);
}

export interface AlertDialogState {
    title?: string;
    message?: string;
    buttons: StateButton[];
}

const default_state: AlertDialogState = {
    buttons: [{ text: 'Ok' }],
};

const titleAria = 'alert-dialog-title';
const descriptionAria = 'alert-dialog-description';

const AlertDialog = () => {
    const [state, setState] = useState<AlertDialogState>(default_state)
    let open = Boolean(state.title) || Boolean(state.message);

    useEffect(() => {
        let dialogSub = PubSub.get().subscribeAlertDialog((o) => setState({ ...default_state, ...o }));
        return () => { PubSub.get().unsubscribe(dialogSub) };
    }, [])

    const handleClick = useCallback((event: any, action: ((e?: any) => void) | null | undefined) => {
        if (action) action(event);
        setState(default_state);
    }, []);

    const resetState = useCallback(() => setState(default_state), []);

    return (
        <Dialog
            open={open}
            disableScrollLock={true}
            onClose={resetState}
            aria-labelledby={titleAria}
            aria-describedby={descriptionAria}
        >
            <DialogTitle
                ariaLabel={titleAria}
                title={firstString(state.title)}
                onClose={resetState}
            />
            <DialogContent>
                <DialogContentText id={descriptionAria} sx={{
                    whiteSpace: 'pre-wrap',
                    wordWrap: 'break-word',
                    paddingTop: 2,
                }}>
                    {state.message}
                </DialogContentText>
            </DialogContent>
            {/* Actions */}
            <DialogActions>
                {state?.buttons && state.buttons.length > 0 ? (
                    state.buttons.map((b: StateButton, index) => (
                        <Button key={`alert-button-${index}`} onClick={(e) => handleClick(e, b.onClick)} color="secondary">
                            {b.text}
                        </Button>
                    ))
                ) : null}
            </DialogActions>
        </Dialog >
    );
}

export { AlertDialog };
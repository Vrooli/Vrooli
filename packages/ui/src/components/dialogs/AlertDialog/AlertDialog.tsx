import { useCallback, useEffect, useState } from 'react';
import {
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
    DialogTitle,
    IconButton,
    Typography,
    useTheme,
} from '@mui/material';
import {
    Close as CloseIcon,
} from '@mui/icons-material';
import { noSelect } from 'styles';
import { PubSub } from 'utils';

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

const AlertDialog = () => {
    const { palette } = useTheme();
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
            aria-labelledby="alert-dialog-title"
            aria-describedby="alert-dialog-description"
        >
            {/* Title with close icon */}
            <DialogTitle
                id="alert-dialog-title"
                sx={{
                    ...noSelect,
                    display: 'flex',
                    alignItems: 'center',
                    padding: 2,
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                }}
            >
                <Typography
                    variant="h6"
                    sx={{
                        width: '-webkit-fill-available',
                        textAlign: 'center',
                    }}
                >
                    {state.title}
                </Typography>
                <IconButton
                    aria-label="close"
                    edge="end"
                    onClick={resetState}
                >
                    <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                </IconButton>
            </DialogTitle>
            <DialogContent>
                <DialogContentText id="alert-dialog-description" sx={{
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
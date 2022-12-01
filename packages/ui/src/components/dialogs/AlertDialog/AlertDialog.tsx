import { useCallback, useEffect, useState } from 'react';
import {
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
} from '@mui/material';
import { firstString, PubSub, translateCommonKey, translateSnackMessage } from 'utils';
import { DialogTitle } from 'components';
import { AlertDialogProps } from '../types';

interface StateButton {
    label: string;
    onClick?: (() => void);
}

export interface AlertDialogState {
    title?: string;
    message?: string;
    buttons: StateButton[];
}

const defaultState = (languages: string[]): AlertDialogState => ({
    buttons: [{ label: translateCommonKey('Ok', undefined, languages) }],
});

const titleAria = 'alert-dialog-title';
const descriptionAria = 'alert-dialog-description';

const AlertDialog = ({
    languages
}: AlertDialogProps) => {
    const [state, setState] = useState<AlertDialogState>(defaultState(languages))
    let open = Boolean(state.title) || Boolean(state.message);

    useEffect(() => {
        let dialogSub = PubSub.get().subscribeAlertDialog((o) => setState({
            title: o.titleKey ? translateCommonKey(o.titleKey, o.titleVariables, languages) : undefined,
            message: o.messageKey ? translateSnackMessage(o.messageKey, o.messageVariables, languages).details : undefined,
            buttons: o.buttons.map((b) => ({
                label: translateCommonKey(b.labelKey, b.labelVariables, languages),
                onClick: b.onClick,
            })),
        }));
        return () => { PubSub.get().unsubscribe(dialogSub) };
    }, [languages])

    const handleClick = useCallback((event: any, action: ((e?: any) => void) | null | undefined) => {
        if (action) action(event);
        setState(defaultState(languages));
    }, [languages]);

    const resetState = useCallback(() => setState(defaultState(languages)), [languages]);

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
                            {b.label}
                        </Button>
                    ))
                ) : null}
            </DialogActions>
        </Dialog >
    );
}

export { AlertDialog };
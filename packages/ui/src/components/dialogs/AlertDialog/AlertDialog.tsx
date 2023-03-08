import { useCallback, useEffect, useState } from 'react';
import {
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
} from '@mui/material';
import { firstString, PubSub, translateSnackMessage } from 'utils';
import { DialogTitle } from 'components';
import i18next from 'i18next';
import { useTranslation } from 'react-i18next';

interface StateButton {
    label: string;
    onClick?: (() => void);
}

export interface AlertDialogState {
    title?: string;
    message?: string;
    buttons: StateButton[];
}

const defaultState = (): AlertDialogState => ({
    buttons: [{ label: i18next.t('Ok') }],
});

const titleId = 'alert-dialog-title';
const descriptionAria = 'alert-dialog-description';

const AlertDialog = () => {
    const { t } = useTranslation();

    const [state, setState] = useState<AlertDialogState>(defaultState())
    let open = Boolean(state.title) || Boolean(state.message);

    useEffect(() => {
        let dialogSub = PubSub.get().subscribeAlertDialog((o) => setState({
            title: o.titleKey ? t(o.titleKey, { ...o.titleVariables, defaultValue: o.titleKey }) : undefined,
            message: o.messageKey ? translateSnackMessage(o.messageKey, o.messageVariables).details : undefined,
            buttons: o.buttons.map((b) => ({
                label: t(b.labelKey, { ...b.labelVariables, defaultValue: b.labelKey }),
                onClick: b.onClick,
            })),
        }));
        return () => { PubSub.get().unsubscribe(dialogSub) };
    }, [t])

    const handleClick = useCallback((event: any, action: ((e?: any) => void) | null | undefined) => {
        if (action) action(event);
        setState(defaultState());
    }, []);

    const resetState = useCallback(() => setState(defaultState()), []);

    return (
        <Dialog
            open={open}
            disableScrollLock={true}
            onClose={resetState}
            aria-labelledby={titleId}
            aria-describedby={descriptionAria}
        >
            <DialogTitle
                id={titleId}
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
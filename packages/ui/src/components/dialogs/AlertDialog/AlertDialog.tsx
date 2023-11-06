import { Button, Dialog, DialogActions, DialogContent, DialogContentText, useTheme } from "@mui/material";
import { useZIndex } from "hooks/useZIndex";
import i18next from "i18next";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { translateSnackMessage } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { DialogTitle } from "../DialogTitle/DialogTitle";

interface StateButton {
    label: string;
    onClick?: (() => unknown);
}

export interface AlertDialogState {
    title?: string;
    message?: string;
    buttons: StateButton[];
}

const defaultState = (): AlertDialogState => ({
    buttons: [{ label: i18next.t("Ok") }],
});

const titleId = "alert-dialog-title";
const descriptionAria = "alert-dialog-description";

export const AlertDialog = () => {
    const { t } = useTranslation();
    const { palette } = useTheme();

    const [state, setState] = useState<AlertDialogState>(defaultState());
    const [open, setOpen] = useState(false);
    const zIndex = useZIndex(open, false, 1000);

    useEffect(() => {
        const dialogSub = PubSub.get().subscribeAlertDialog((o) => {
            setState({
                title: o.titleKey ? t(o.titleKey, { ...o.titleVariables, defaultValue: o.titleKey }) : undefined,
                message: o.messageKey ? translateSnackMessage(o.messageKey, o.messageVariables).details ?? translateSnackMessage(o.messageKey, o.messageVariables).message : undefined,
                buttons: o.buttons.map((b) => ({
                    label: t(b.labelKey, { ...b.labelVariables, defaultValue: b.labelKey }),
                    onClick: b.onClick,
                })),
            });
            setOpen(true);
        });
        return () => { PubSub.get().unsubscribe(dialogSub); };
    }, [t]);

    const handleClick = useCallback((event: any, action: ((e?: unknown) => unknown) | null | undefined) => {
        if (action) action(event);
        setOpen(false);
    }, []);

    const resetState = useCallback(() => setOpen(false), []);

    return (
        <Dialog
            open={open}
            disableScrollLock={true}
            onClose={resetState}
            aria-labelledby={state.title ? titleId : undefined}
            aria-describedby={descriptionAria}
            sx={{
                zIndex,
                "& > .MuiDialog-container": {
                    "& > .MuiPaper-root": {
                        zIndex,
                        borderRadius: 4,
                        background: palette.background.paper,
                    },
                },
            }}
        >
            {state.title && <DialogTitle
                id={titleId}
                title={state.title}
                onClose={resetState}
            />}
            <DialogContent>
                <DialogContentText id={descriptionAria} sx={{
                    whiteSpace: "pre-wrap",
                    wordWrap: "break-word",
                    paddingTop: 2,
                }}>
                    {state.message}
                </DialogContentText>
            </DialogContent>
            {/* Actions */}
            <DialogActions>
                {state?.buttons && state.buttons.length > 0 ? (
                    state.buttons.map((b: StateButton, index) => (
                        <Button
                            key={`alert-button-${index}`}
                            onClick={(e) => handleClick(e, b.onClick)}
                            variant="text"
                        >
                            {b.label}
                        </Button>
                    ))
                ) : null}
            </DialogActions>
        </Dialog >
    );
};

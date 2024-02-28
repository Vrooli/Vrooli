import { Button, Dialog, DialogActions, DialogContent, DialogContentText, Palette, useTheme } from "@mui/material";
import { useZIndex } from "hooks/useZIndex";
import i18next from "i18next";
import { ErrorIcon, InfoIcon, SuccessIcon, WarningIcon } from "icons";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { translateSnackMessage } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { DialogTitle } from "../DialogTitle/DialogTitle";

interface StateButton {
    label: string;
    onClick?: (() => unknown);
}

export enum AlertDialogSeverity {
    Error = "Error",
    Info = "Info",
    Success = "Success",
    Warning = "Warning",
}

export interface AlertDialogState {
    title?: string;
    message?: string;
    buttons: StateButton[];
    severity?: AlertDialogSeverity | `${AlertDialogSeverity}`;
}

const defaultState = (): AlertDialogState => ({
    buttons: [{ label: i18next.t("Ok") }],
});

const iconColor = (severity: AlertDialogSeverity | `${AlertDialogSeverity}` | undefined, palette: Palette) => {
    switch (severity) {
        case "Error":
            return palette.error.dark;
        case "Info":
            return palette.info.main;
        case "Success":
            return palette.success.main;
        case "Warning":
            return palette.warning.main;
        default:
            return palette.primary.light;
    }
};

const titleId = "alert-dialog-title";
const descriptionAria = "alert-dialog-description";

export const AlertDialog = () => {
    const { t } = useTranslation();
    const { palette } = useTheme();

    const [state, setState] = useState<AlertDialogState>(defaultState());
    const [open, setOpen] = useState(false);
    const zIndex = useZIndex(open, false, 1000);

    // Determine the icon based on severity
    const Icon = state.severity ? {
        [AlertDialogSeverity.Error]: ErrorIcon,
        [AlertDialogSeverity.Info]: InfoIcon,
        [AlertDialogSeverity.Success]: SuccessIcon,
        [AlertDialogSeverity.Warning]: WarningIcon,
    }[state.severity] : null;

    useEffect(() => {
        const unsubscribe = PubSub.get().subscribe("alertDialog", (o) => {
            setState({
                title: o.titleKey ? t(o.titleKey, { ...o.titleVariables, defaultValue: o.titleKey }) : undefined,
                message: o.messageKey ?
                    translateSnackMessage(o.messageKey, o.messageVariables, o.severity === "Error" ? "error" : undefined).details ??
                    translateSnackMessage(o.messageKey, o.messageVariables, o.severity === "Error" ? "error" : undefined).message :
                    undefined,
                buttons: o.buttons.map((b) => ({
                    label: t(b.labelKey, { ...b.labelVariables, defaultValue: b.labelKey }),
                    onClick: b.onClick,
                })),
                severity: o.severity,
            });
            setOpen(true);
        });
        return unsubscribe;
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
                startComponent={Icon ? <Icon fill={iconColor(state.severity, palette)} /> : undefined}
            />}
            <DialogContent>
                {!state.title && Icon !== null && <Icon fill={iconColor(state.severity, palette)} />}
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

import { Button, Dialog, DialogActions, DialogContent, DialogContentText, DialogProps, Palette, styled, useTheme } from "@mui/material";
import i18next from "i18next";
import { ErrorIcon, InfoIcon, SuccessIcon, WarningIcon } from "icons/common.js";
import React, { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Z_INDEX } from "utils/consts.js";
import { translateSnackMessage } from "utils/display/translationTools.js";
import { PubSub } from "utils/pubsub.js";
import { DialogTitle } from "../DialogTitle/DialogTitle.js";

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

function defaultState(): AlertDialogState {
    return {
        buttons: [{ label: i18next.t("Ok") }],
    } as const;
}

function iconColor(severity: AlertDialogSeverity | `${AlertDialogSeverity}` | undefined, palette: Palette) {
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
}

const titleId = "alert-dialog-title";
const descriptionAria = "alert-dialog-description";
interface StyledDialogProps extends Omit<DialogProps, "zIndex"> {
    zIndex: number;
}
const StyledDialog = styled(Dialog, {
    shouldForwardProp: (prop) => prop !== "zIndex",
})<StyledDialogProps>(({ theme, zIndex }) => ({
    zIndex,
    "& > .MuiDialog-container": {
        "& > .MuiPaper-root": {
            zIndex,
            borderRadius: 4,
            background: theme.palette.background.paper,
        },
    },
}));

const dialogContextTextStyle = {
    whiteSpace: "pre-wrap",
    wordWrap: "break-word",
    paddingTop: 2,
} as const;

export function AlertDialog() {
    const { t } = useTranslation();
    const { palette } = useTheme();

    const [state, setState] = useState<AlertDialogState>(defaultState());
    const [open, setOpen] = useState(false);

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

    const handleClick = useCallback((
        event: React.MouseEvent<HTMLButtonElement>,
        action: ((e?: unknown) => unknown) | null | undefined,
    ) => {
        if (action) action(event);
        setOpen(false);
    }, []);

    const resetState = useCallback(() => setOpen(false), []);

    return (
        <StyledDialog
            open={open}
            disableScrollLock={true}
            onClose={resetState}
            aria-labelledby={state.title ? titleId : undefined}
            aria-describedby={descriptionAria}
            zIndex={Z_INDEX.Popup}
        >
            {state.title && <DialogTitle
                id={titleId}
                title={state.title}
                onClose={resetState}
                startComponent={Icon ? <Icon fill={iconColor(state.severity, palette)} /> : undefined}
            />}
            <DialogContent>
                {!state.title && Icon !== null && <Icon fill={iconColor(state.severity, palette)} />}
                <DialogContentText id={descriptionAria} sx={dialogContextTextStyle}>
                    {state.message}
                </DialogContentText>
            </DialogContent>
            {/* Actions */}
            <DialogActions>
                {state?.buttons && state.buttons.length > 0 ? (
                    state.buttons.map((b: StateButton, index) => {
                        function onClick(e: React.MouseEvent<HTMLButtonElement>) {
                            handleClick(e, b.onClick);
                        }

                        return (
                            <Button
                                key={`alert-button-${index}`}
                                onClick={onClick}
                                variant="text"
                            >
                                {b.label}
                            </Button>
                        );
                    })
                ) : null}
            </DialogActions>
        </StyledDialog>
    );
}

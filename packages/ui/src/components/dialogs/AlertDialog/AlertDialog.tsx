import { Button, Dialog, DialogActions, DialogContent, DialogContentText, DialogProps, styled, useTheme } from "@mui/material";
import i18next from "i18next";
import React, { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Icon, IconInfo } from "../../../icons/Icons.js";
import { Z_INDEX } from "../../../utils/consts.js";
import { translateSnackMessage } from "../../../utils/display/translationTools.js";
import { PubSub } from "../../../utils/pubsub.js";
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
    const { iconInfo, iconFill }: { iconInfo: IconInfo | null, iconFill: string | null } = state.severity ? {
        [AlertDialogSeverity.Error]: { iconInfo: { name: "Error" as const, type: "Common" as const }, iconFill: palette.error.dark },
        [AlertDialogSeverity.Info]: { iconInfo: { name: "Info" as const, type: "Common" as const }, iconFill: palette.info.main },
        [AlertDialogSeverity.Success]: { iconInfo: { name: "Success" as const, type: "Common" as const }, iconFill: palette.success.main },
        [AlertDialogSeverity.Warning]: { iconInfo: { name: "Warning" as const, type: "Common" as const }, iconFill: palette.warning.main },
    }[state.severity] : { iconInfo: null, iconFill: null };

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
                startComponent={iconInfo ? <Icon
                    decorative={false}
                    fill={iconFill}
                    info={iconInfo}
                    size={24}
                /> : undefined}
            />}
            <DialogContent>
                {!state.title && iconInfo !== null && <Icon
                    decorative={false}
                    fill={iconFill}
                    info={iconInfo}
                    size={24}
                />}
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

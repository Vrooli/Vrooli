import Button from "@mui/material/Button";
import DialogContentText from "@mui/material/DialogContentText";
import { useTheme } from "@mui/material";
import i18next from "i18next";
import React, { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Icon, type IconInfo } from "../../../icons/Icons.js";
import { translateSnackMessage } from "../../../utils/display/translationTools.js";
import { PubSub } from "../../../utils/pubsub.js";
import { Dialog, DialogContent, DialogActions } from "../Dialog/Dialog.js";

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

const dialogContextTextStyle = {
    whiteSpace: "pre-wrap",
    wordWrap: "break-word",
    paddingTop: 2,
} as const;

// Global flag to ensure only one AlertDialog instance is active
let alertDialogInstance: string | null = null;

export function AlertDialog() {
    const { t } = useTranslation();
    const { palette } = useTheme();

    const [state, setState] = useState<AlertDialogState>(defaultState());
    const [open, setOpen] = useState(false);
    const [instanceId] = useState(() => Math.random().toString(36).substr(2, 9));

    // Determine the icon based on severity
    const { iconInfo, iconFill }: { iconInfo: IconInfo | null, iconFill: string | null } = state.severity ? {
        [AlertDialogSeverity.Error]: { iconInfo: { name: "Error" as const, type: "Common" as const }, iconFill: palette.error.dark },
        [AlertDialogSeverity.Info]: { iconInfo: { name: "Info" as const, type: "Common" as const }, iconFill: palette.info.main },
        [AlertDialogSeverity.Success]: { iconInfo: { name: "Success" as const, type: "Common" as const }, iconFill: palette.success.main },
        [AlertDialogSeverity.Warning]: { iconInfo: { name: "Warning" as const, type: "Common" as const }, iconFill: palette.warning.main },
    }[state.severity] : { iconInfo: null, iconFill: null };

    useEffect(() => {
        // Register this instance as the active one
        if (!alertDialogInstance) {
            alertDialogInstance = instanceId;
        }

        const unsubscribe = PubSub.get().subscribe("alertDialog", (o) => {
            // Only respond if this is the active instance
            if (alertDialogInstance === instanceId) {
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
            }
        });

        return () => {
            unsubscribe();
            // Clear the active instance if this was it
            if (alertDialogInstance === instanceId) {
                alertDialogInstance = null;
            }
        };
    }, [t, instanceId]);

    const handleClick = useCallback((
        event: React.MouseEvent<HTMLButtonElement>,
        action: ((e?: unknown) => unknown) | null | undefined,
    ) => {
        if (action) action(event);
        setOpen(false);
    }, []);

    const resetState = useCallback(() => setOpen(false), []);

    // Determine dialog variant based on severity
    const variant = state.severity ? {
        [AlertDialogSeverity.Error]: "danger" as const,
        [AlertDialogSeverity.Warning]: "default" as const,
        [AlertDialogSeverity.Success]: "success" as const,
        [AlertDialogSeverity.Info]: "default" as const,
    }[state.severity] : "default" as const;

    // Create title with icon
    const titleWithIcon = state.title ? (
        <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
            {iconInfo && (
                <Icon
                    decorative="false"
                    fill={iconFill}
                    info={iconInfo}
                    size={24}
                />
            )}
            <span>{state.title}</span>
        </div>
    ) : undefined;

    return (
        <Dialog
            isOpen={open}
            onClose={resetState}
            title={titleWithIcon}
            size="md"
            variant={variant}
            aria-labelledby={state.title ? titleId : undefined}
            aria-describedby={descriptionAria}
        >
            <DialogContent>
                {!state.title && iconInfo !== null && (
                    <div style={{ display: "flex", alignItems: "center", marginBottom: "16px" }}>
                        <Icon
                            decorative="false"
                            fill={iconFill}
                            info={iconInfo}
                            size={24}
                        />
                    </div>
                )}
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
        </Dialog>
    );
}

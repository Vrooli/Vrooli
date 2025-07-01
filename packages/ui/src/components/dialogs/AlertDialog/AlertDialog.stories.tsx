import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { action } from "@storybook/addon-actions";
import {
    Box,
    Card,
    CardContent,
    Typography,
    TextField,
    FormControl,
    InputLabel,
    Select,
    MenuItem,
    FormControlLabel,
    Checkbox,
    Alert,
    Stack,
} from "@mui/material";
import { Button } from "../../buttons/Button.js";
import { AlertDialog, AlertDialogSeverity } from "./AlertDialog.js";
import { PubSub } from "../../../utils/pubsub.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

const meta: Meta<typeof AlertDialog> = {
    title: "Components/Dialogs/AlertDialog",
    component: AlertDialog,
    parameters: {
        layout: "fullscreen",
        backgrounds: { disable: true },
        docs: {
            story: {
                inline: false,
                iframeHeight: 600,
            },
        },
    },
    tags: ["autodocs"],
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Showcase story with controls
export const Showcase: Story = {
    render: () => {
        const [title, setTitle] = useState("Alert Title");
        const [message, setMessage] = useState("This is an alert dialog message. It can contain important information for the user.");
        const [severity, setSeverity] = useState<AlertDialogSeverity>(AlertDialogSeverity.Info);
        const [confirmText, setConfirmText] = useState("OK");
        const [cancelText, setCancelText] = useState("Cancel");
        const [showCancelButton, setShowCancelButton] = useState(true);

        const handleOpen = () => {
            const confirmAction = action("confirm-clicked");
            const cancelAction = action("cancel-clicked");

            const buttons = showCancelButton ? [
                { 
                    labelKey: cancelText, 
                    onClick: () => {
                        cancelAction();
                    },
                },
                { 
                    labelKey: confirmText, 
                    onClick: () => {
                        confirmAction();
                    },
                },
            ] : [
                { 
                    labelKey: confirmText, 
                    onClick: () => {
                        confirmAction();
                    },
                },
            ];

            PubSub.get().publish("alertDialog", {
                titleKey: title,
                messageKey: message,
                severity,
                buttons,
            });
        };

        return (
            <Stack spacing={3} sx={{ maxWidth: 600 }}>
                {/* Control Panel */}
                <Card>
                    <CardContent>
                        <Typography variant="h6" gutterBottom>
                            AlertDialog Controls
                        </Typography>
                        
                        <Stack spacing={2}>
                            <TextField
                                label="Title"
                                value={title}
                                onChange={(e) => setTitle(e.target.value)}
                                fullWidth
                            />

                            <TextField
                                label="Message"
                                value={message}
                                onChange={(e) => setMessage(e.target.value)}
                                multiline
                                rows={3}
                                fullWidth
                            />

                            <FormControl fullWidth>
                                <InputLabel>Severity</InputLabel>
                                <Select
                                    value={severity}
                                    label="Severity"
                                    onChange={(e) => setSeverity(e.target.value as AlertDialogSeverity)}
                                >
                                    {Object.values(AlertDialogSeverity).map((sev) => (
                                        <MenuItem key={sev} value={sev}>{sev}</MenuItem>
                                    ))}
                                </Select>
                            </FormControl>

                            <TextField
                                label="Confirm Button Text"
                                value={confirmText}
                                onChange={(e) => setConfirmText(e.target.value)}
                                fullWidth
                            />

                            <FormControlLabel
                                control={
                                    <Checkbox
                                        checked={showCancelButton}
                                        onChange={(e) => setShowCancelButton(e.target.checked)}
                                    />
                                }
                                label="Show Cancel Button"
                            />

                            {showCancelButton && (
                                <TextField
                                    label="Cancel Button Text"
                                    value={cancelText}
                                    onChange={(e) => setCancelText(e.target.value)}
                                    fullWidth
                                />
                            )}
                        </Stack>
                    </CardContent>
                </Card>

                {/* Trigger Button */}
                <Box sx={{ textAlign: "center" }}>
                    <Button onClick={handleOpen} variant="primary" size="lg">
                        Open Alert Dialog
                    </Button>
                </Box>

                {/* Info about severity colors */}
                <Alert severity="info">
                    <Typography variant="body2">
                        <strong>Severity Icons & Colors:</strong>
                    </Typography>
                    <Box component="ul" sx={{ mt: 1, mb: 0, pl: 2 }}>
                        <li><strong>Info:</strong> Blue info icon</li>
                        <li><strong>Success:</strong> Green checkmark icon</li>
                        <li><strong>Warning:</strong> Orange warning icon</li>
                        <li><strong>Error:</strong> Red error icon</li>
                    </Box>
                </Alert>

                {/* Alert Dialog Component */}
                <AlertDialog key="showcase-alert" />
            </Stack>
        );
    },
};

// Example preset configurations
export const InfoAlert: Story = {
    render: () => {
        const handleOpen = () => {
            PubSub.get().publish("alertDialog", {
                titleKey: "Information",
                messageKey: "This is an informational message to keep you updated about the current status.",
                severity: AlertDialogSeverity.Info,
                buttons: [
                    { 
                        labelKey: "Got it", 
                        onClick: action("info-ok-clicked"),
                    },
                ],
            });
        };

        return (
            <>
                <Button onClick={handleOpen} variant="primary">
                    Show Info Alert
                </Button>
                <AlertDialog key="info-alert" />
            </>
        );
    },
};

export const SuccessAlert: Story = {
    render: () => {
        const handleOpen = () => {
            PubSub.get().publish("alertDialog", {
                titleKey: "Success!",
                messageKey: "Your operation completed successfully. All changes have been saved.",
                severity: AlertDialogSeverity.Success,
                buttons: [
                    { 
                        labelKey: "Great!", 
                        onClick: action("success-ok-clicked"),
                    },
                ],
            });
        };

        return (
            <>
                <Button onClick={handleOpen} variant="primary">
                    Show Success Alert
                </Button>
                <AlertDialog key="success-alert" />
            </>
        );
    },
};

export const WarningAlert: Story = {
    render: () => {
        const handleOpen = () => {
            PubSub.get().publish("alertDialog", {
                titleKey: "Warning",
                messageKey: "This action may have unintended consequences. Are you sure you want to proceed?",
                severity: AlertDialogSeverity.Warning,
                buttons: [
                    { 
                        labelKey: "Cancel", 
                        onClick: action("warning-cancel-clicked"),
                    },
                    { 
                        labelKey: "Proceed", 
                        onClick: action("warning-proceed-clicked"),
                    },
                ],
            });
        };

        return (
            <>
                <Button onClick={handleOpen} variant="primary">
                    Show Warning Alert
                </Button>
                <AlertDialog key="warning-alert" />
            </>
        );
    },
};

export const ErrorAlert: Story = {
    render: () => {
        const handleOpen = () => {
            PubSub.get().publish("alertDialog", {
                titleKey: "Error Occurred",
                messageKey: "An unexpected error has occurred. Please try again later or contact support if the problem persists.",
                severity: AlertDialogSeverity.Error,
                buttons: [
                    { 
                        labelKey: "Close", 
                        onClick: action("error-close-clicked"),
                    },
                    { 
                        labelKey: "Retry", 
                        onClick: action("error-retry-clicked"),
                    },
                ],
            });
        };

        return (
            <>
                <Button onClick={handleOpen} variant="danger">
                    Show Error Alert
                </Button>
                <AlertDialog key="error-alert" />
            </>
        );
    },
};

import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { action } from "@storybook/addon-actions";
import {
    Box,
    Card,
    CardContent,
    Typography,
    FormControlLabel,
    Checkbox,
    Alert,
    Stack,
} from "@mui/material";
import { Button } from "../../buttons/Button.js";
import { SubroutineCreateDialog } from "./SubroutineCreateDialog.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

const meta: Meta<typeof SubroutineCreateDialog> = {
    title: "Components/Dialogs/SubroutineCreateDialog",
    component: SubroutineCreateDialog,
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
        const [isOpen, setIsOpen] = useState(false);
        const [simulateSelection, setSimulateSelection] = useState(false);

        const handleOpen = () => {
            setIsOpen(true);
        };

        const handleClose = () => {
            action("dialog-closed")();
            setIsOpen(false);
        };

        return (
            <Stack spacing={3} sx={{ maxWidth: 600 }}>
                {/* Control Panel */}
                <Card>
                    <CardContent>
                        <Typography variant="h6" gutterBottom>
                            SubroutineCreateDialog Controls
                        </Typography>
                        
                        <Box>
                            <FormControlLabel
                                control={
                                    <Checkbox
                                        checked={simulateSelection}
                                        onChange={(e) => setSimulateSelection(e.target.checked)}
                                    />
                                }
                                label="Simulate Subroutine Type Selection"
                            />
                            <Typography variant="caption" color="text.secondary" display="block">
                                When enabled, shows what would happen after selecting a subroutine type (currently shows "TODO" placeholder)
                            </Typography>
                        </Box>
                    </CardContent>
                </Card>

                {/* Trigger Button */}
                <Box sx={{ textAlign: "center" }}>
                    <Button onClick={handleOpen} variant="primary" size="lg">
                        Open Create Subroutine Dialog
                    </Button>
                </Box>

                {/* Info about subroutine types */}
                <Alert severity="success">
                    <Typography variant="body2">
                        <strong>Available Subroutine Types:</strong>
                    </Typography>
                    <Box component="ul" sx={{ mt: 1, mb: 0, pl: 2 }}>
                        <li><strong>Prompt:</strong> AI prompt-based subroutines</li>
                        <li><strong>Data:</strong> Data processing subroutines</li>
                        <li><strong>Generate:</strong> Content generation subroutines</li>
                        <li><strong>API:</strong> External API integration subroutines</li>
                        <li><strong>Smart Contract:</strong> Blockchain smart contract subroutines</li>
                        <li><strong>Web Content:</strong> Web content extraction subroutines</li>
                        <li><strong>Code:</strong> Code execution subroutines</li>
                    </Box>
                </Alert>

                {/* Dialog Component */}
                <SubroutineCreateDialog
                    isOpen={isOpen}
                    onClose={handleClose}
                />
            </Stack>
        );
    },
};

// Simple open dialog
export const OpenDialog: Story = {
    render: () => {
        const handleClose = () => {
            action("dialog-closed")();
        };

        return (
            <SubroutineCreateDialog
                isOpen={true}
                onClose={handleClose}
            />
        );
    },
};

// Dialog showing selection state
export const SelectionState: Story = {
    render: () => {
        const handleClose = () => {
            action("dialog-closed")();
        };

        return (
            <Stack spacing={2}>
                <Alert severity="info">
                    <Typography variant="body2" sx={{ textAlign: "center" }}>
                        This shows the subroutine type selection interface
                    </Typography>
                </Alert>
                <SubroutineCreateDialog
                    isOpen={true}
                    onClose={handleClose}
                />
            </Stack>
        );
    },
};

// Closed dialog (for testing)
export const ClosedDialog: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);

        const handleOpen = () => {
            setIsOpen(true);
        };

        const handleClose = () => {
            action("dialog-closed")();
            setIsOpen(false);
        };

        return (
            <Box sx={{ textAlign: "center" }}>
                <Button onClick={handleOpen} variant="primary">
                    Open Dialog
                </Button>
                <SubroutineCreateDialog
                    isOpen={isOpen}
                    onClose={handleClose}
                />
                <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
                    Dialog is currently {isOpen ? "open" : "closed"}
                </Typography>
            </Box>
        );
    },
};

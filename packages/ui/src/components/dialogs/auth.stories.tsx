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
    Chip,
} from "@mui/material";
import { Button } from "../buttons/Button.js";
import { WalletInstallDialog, WalletSelectDialog } from "./auth.js";
import { centeredDecorator } from "../../__test/helpers/storybookDecorators.tsx";

const meta: Meta = {
    title: "Components/Dialogs/Auth",
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

// Mock wallet data for stories
const mockInstalledWallets = [
    { name: "Nami", icon: "data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDgiIGhlaWdodD0iNDgiIHZpZXdCb3g9IjAgMCA0OCA0OCIgZmlsbD0ibm9uZSI+PHJlY3Qgd2lkdGg9IjQ4IiBoZWlnaHQ9IjQ4IiByeD0iMjQiIGZpbGw9IiMzNDk4ZGIiLz48L3N2Zz4=" },
    { name: "Eternl", icon: "data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDgiIGhlaWdodD0iNDgiIHZpZXdCb3g9IjAgMCA0OCA0OCIgZmlsbD0ibm9uZSI+PHJlY3Qgd2lkdGg9IjQ4IiBoZWlnaHQ9IjQ4IiByeD0iMjQiIGZpbGw9IiNmZjU3MjIiLz48L3N2Zz4=" },
    { name: "Flint", icon: "data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDgiIGhlaWdodD0iNDgiIHZpZXdCb3g9IjAgMCA0OCA0OCIgZmlsbD0ibm9uZSI+PHJlY3Qgd2lkdGg9IjQ4IiBoZWlnaHQ9IjQ4IiByeD0iMjQiIGZpbGw9IiM2MzY2ZjEiLz48L3N2Zz4=" },
];

// Showcase story with controls for both dialogs
export const Showcase: Story = {
    render: () => {
        const [installDialogOpen, setInstallDialogOpen] = useState(false);
        const [selectDialogOpen, setSelectDialogOpen] = useState(false);
        const [simulateInstalledWallets, setSimulateInstalledWallets] = useState(true);
        const [selectedWallet, setSelectedWallet] = useState<string | null>(null);

        // Mock the wallet integration function
        React.useEffect(() => {
            // Mock getInstalledWalletProviders
            const originalGetInstalledWalletProviders = (window as any).getInstalledWalletProviders;
            
            (window as any).getInstalledWalletProviders = () => {
                if (simulateInstalledWallets) {
                    return mockInstalledWallets.map((wallet, index) => [
                        `wallet_${index}`,
                        wallet,
                    ]);
                } else {
                    return [];
                }
            };

            return () => {
                (window as any).getInstalledWalletProviders = originalGetInstalledWalletProviders;
            };
        }, [simulateInstalledWallets]);

        const handleOpenInstallDialog = () => {
            setInstallDialogOpen(true);
            action("install-dialog-opened")();
        };

        const handleCloseInstallDialog = () => {
            setInstallDialogOpen(false);
            action("install-dialog-closed")();
        };

        const handleOpenSelectDialog = () => {
            setSelectDialogOpen(true);
            action("select-dialog-opened")();
        };

        const handleCloseSelectDialog = (selectedKey: string | null) => {
            setSelectDialogOpen(false);
            setSelectedWallet(selectedKey);
            action("select-dialog-closed")(selectedKey);
        };

        const handleOpenInstallFromSelect = () => {
            setSelectDialogOpen(false);
            setInstallDialogOpen(true);
            action("opened-install-from-select")();
        };

        return (
            <Stack spacing={3} sx={{ maxWidth: 600 }}>
                {/* Control Panel */}
                <Card>
                    <CardContent>
                        <Typography variant="h6" gutterBottom>
                            Wallet Dialog Controls
                        </Typography>
                        
                        <Stack spacing={2}>
                            <Box>
                                <FormControlLabel
                                    control={
                                        <Checkbox
                                            checked={simulateInstalledWallets}
                                            onChange={(e) => setSimulateInstalledWallets(e.target.checked)}
                                        />
                                    }
                                    label="Simulate Installed Wallets"
                                />
                                <Typography variant="caption" color="text.secondary" display="block">
                                    When disabled, shows "no wallets installed" state
                                </Typography>
                            </Box>

                            {selectedWallet && (
                                <Alert severity="success">
                                    <Typography variant="body2">
                                        <strong>Last Selected Wallet:</strong> {selectedWallet}
                                    </Typography>
                                </Alert>
                            )}
                        </Stack>
                    </CardContent>
                </Card>

                {/* Action Buttons */}
                <Box sx={{ display: "flex", gap: 2, justifyContent: "center" }}>
                    <Button onClick={handleOpenSelectDialog} variant="primary" size="lg">
                        Open Wallet Select Dialog
                    </Button>
                    <Button onClick={handleOpenInstallDialog} variant="secondary" size="lg">
                        Open Wallet Install Dialog
                    </Button>
                </Box>

                {/* Info about wallet integration */}
                <Alert severity="success">
                    <Typography variant="body2">
                        <strong>Wallet Integration Features:</strong>
                    </Typography>
                    <Box component="ul" sx={{ mt: 1, mb: 0, pl: 2 }}>
                        <li><strong>WalletSelectDialog:</strong> Shows installed wallets that support CIP-0030</li>
                        <li><strong>WalletInstallDialog:</strong> Provides links to install popular wallets</li>
                        <li>Supports Chromium-based browsers with extension support</li>
                        <li>Recommends Nami wallet for new users</li>
                        <li>Includes fallback for when no wallets are installed</li>
                    </Box>
                </Alert>

                {/* Current state */}
                <Alert severity="info">
                    <Typography variant="body2">
                        <strong>Current State:</strong>
                    </Typography>
                    <Box component="ul" sx={{ mt: 1, mb: 0, pl: 2 }}>
                        <li>Installed Wallets Simulation: {simulateInstalledWallets ? "Enabled" : "Disabled"}</li>
                        <li>Available Wallets: {simulateInstalledWallets ? mockInstalledWallets.length : 0}</li>
                        <li>Install Dialog: {installDialogOpen ? "Open" : "Closed"}</li>
                        <li>Select Dialog: {selectDialogOpen ? "Open" : "Closed"}</li>
                    </Box>
                </Alert>

                {/* Dialog Components */}
                <WalletInstallDialog
                    open={installDialogOpen}
                    onClose={handleCloseInstallDialog}
                />

                <WalletSelectDialog
                    open={selectDialogOpen}
                    onClose={handleCloseSelectDialog}
                    handleOpenInstall={handleOpenInstallFromSelect}
                />
            </Stack>
        );
    },
};

// WalletSelectDialog with wallets available
export const WalletSelectWithWallets: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);
        const [selectedWallet, setSelectedWallet] = useState<string | null>(null);

        // Mock installed wallets
        React.useEffect(() => {
            (window as any).getInstalledWalletProviders = () => {
                return mockInstalledWallets.map((wallet, index) => [
                    `wallet_${index}`,
                    wallet,
                ]);
            };
        }, []);

        const handleClose = (selectedKey: string | null) => {
            setSelectedWallet(selectedKey);
            action("wallet-selected")(selectedKey);
            // Keep dialog open for demo purposes
        };

        const handleOpenInstall = () => {
            action("open-install-clicked")();
        };

        return (
            <Stack spacing={2}>
                <Alert severity="info" sx={{ textAlign: "center" }}>
                    <Typography variant="body2">
                        <strong>WalletSelectDialog with Installed Wallets</strong>
                    </Typography>
                    {selectedWallet && (
                        <Typography variant="body2" color="primary" sx={{ mt: 1 }}>
                            Selected: {selectedWallet}
                        </Typography>
                    )}
                </Alert>
                <WalletSelectDialog
                    open={isOpen}
                    onClose={handleClose}
                    handleOpenInstall={handleOpenInstall}
                />
            </Stack>
        );
    },
};

// WalletSelectDialog with no wallets installed
export const WalletSelectNoWallets: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);

        // Mock no installed wallets
        React.useEffect(() => {
            (window as any).getInstalledWalletProviders = () => {
                return [];
            };
        }, []);

        const handleClose = (selectedKey: string | null) => {
            action("wallet-selected")(selectedKey);
            // Keep dialog open for demo purposes
        };

        const handleOpenInstall = () => {
            action("open-install-clicked")();
        };

        return (
            <Stack spacing={2}>
                <Alert severity="warning" sx={{ textAlign: "center" }}>
                    <Typography variant="body2">
                        <strong>WalletSelectDialog with No Installed Wallets</strong>
                    </Typography>
                    <Typography variant="caption" display="block">
                        Shows "No wallets installed" message and install button
                    </Typography>
                </Alert>
                <WalletSelectDialog
                    open={isOpen}
                    onClose={handleClose}
                    handleOpenInstall={handleOpenInstall}
                />
            </Stack>
        );
    },
};

// WalletInstallDialog standalone
export const WalletInstallStandalone: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);

        const handleClose = () => {
            action("install-dialog-closed")();
            // Keep dialog open for demo purposes
        };

        return (
            <Stack spacing={2}>
                <Alert severity="success" sx={{ textAlign: "center" }}>
                    <Typography variant="body2">
                        <strong>WalletInstallDialog</strong>
                    </Typography>
                    <Typography variant="caption" display="block">
                        Provides links to install popular Cardano wallets
                    </Typography>
                </Alert>
                <WalletInstallDialog
                    open={isOpen}
                    onClose={handleClose}
                />
            </Stack>
        );
    },
};

// Interactive flow demonstrating both dialogs
export const InteractiveFlow: Story = {
    render: () => {
        const [currentStep, setCurrentStep] = useState<"select" | "install" | "complete">("select");
        const [selectedWallet, setSelectedWallet] = useState<string | null>(null);

        // Mock no wallets initially, then some wallets after "installation"
        React.useEffect(() => {
            (window as any).getInstalledWalletProviders = () => {
                if (currentStep === "complete") {
                    return mockInstalledWallets.slice(0, 1).map((wallet, index) => [
                        `wallet_${index}`,
                        wallet,
                    ]);
                }
                return [];
            };
        }, [currentStep]);

        const handleSelectClose = (selectedKey: string | null) => {
            if (selectedKey) {
                setSelectedWallet(selectedKey);
                setCurrentStep("complete");
                action("wallet-selected")(selectedKey);
            }
        };

        const handleInstallClose = () => {
            // Simulate wallet installation
            setCurrentStep("complete");
            action("wallet-installed")();
        };

        const handleOpenInstall = () => {
            setCurrentStep("install");
            action("opened-install-from-select")();
        };

        const resetFlow = () => {
            setCurrentStep("select");
            setSelectedWallet(null);
            action("flow-reset")();
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
                {/* Flow indicator */}
                <div style={{
                    display: "flex",
                    justifyContent: "center",
                    gap: "16px",
                    padding: "16px",
                    backgroundColor: "#f8f9fa",
                    borderRadius: "8px",
                }}>
                    {["select", "install", "complete"].map((step, index) => (
                        <div
                            key={step}
                            style={{
                                padding: "8px 16px",
                                borderRadius: "20px",
                                backgroundColor: currentStep === step ? "#2196f3" : "#e0e0e0",
                                color: currentStep === step ? "white" : "#666",
                                fontWeight: currentStep === step ? "bold" : "normal",
                                fontSize: "14px",
                            }}
                        >
                            {index + 1}. {step.charAt(0).toUpperCase() + step.slice(1)}
                        </div>
                    ))}
                </div>

                {/* Step description */}
                <div style={{
                    padding: "16px",
                    backgroundColor: currentStep === "complete" ? "#e8f5e8" : "#fff3e0",
                    borderRadius: "8px",
                    textAlign: "center",
                }}>
                    {currentStep === "select" && (
                        <div>
                            <strong>Step 1: Select Wallet</strong>
                            <p style={{ margin: "8px 0 0 0", fontSize: "14px" }}>
                                No wallets are installed. Click "Install Wallet" to proceed.
                            </p>
                        </div>
                    )}
                    {currentStep === "install" && (
                        <div>
                            <strong>Step 2: Install Wallet</strong>
                            <p style={{ margin: "8px 0 0 0", fontSize: "14px" }}>
                                Choose a wallet to install. This will simulate the installation process.
                            </p>
                        </div>
                    )}
                    {currentStep === "complete" && (
                        <div>
                            <strong>Step 3: Complete</strong>
                            <p style={{ margin: "8px 0 0 0", fontSize: "14px" }}>
                                {selectedWallet 
                                    ? `Wallet selected: ${selectedWallet}` 
                                    : "Wallet installation simulated. You can now select a wallet."}
                            </p>
                        </div>
                    )}
                </div>

                {/* Reset button */}
                {currentStep === "complete" && (
                    <div style={{ textAlign: "center", marginBottom: "16px" }}>
                        <Button onClick={resetFlow} variant="secondary">
                            Reset Flow
                        </Button>
                    </div>
                )}

                {/* Dialog components */}
                <WalletSelectDialog
                    open={currentStep === "select" || currentStep === "complete"}
                    onClose={handleSelectClose}
                    handleOpenInstall={handleOpenInstall}
                />

                <WalletInstallDialog
                    open={currentStep === "install"}
                    onClose={handleInstallClose}
                />
            </div>
        );
    },
};

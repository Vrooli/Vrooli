import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { Box } from "@mui/material";
import { Button } from "../../buttons/Button.js";
import { InstallPWADialog } from "./InstallPWADialog.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

const meta: Meta<typeof InstallPWADialog> = {
    title: "Components/Dialogs/InstallPWADialog",
    component: InstallPWADialog,
    parameters: {
        docs: {
            description: {
                component: "A dialog that provides step-by-step instructions for installing the Vrooli PWA on different platforms (iOS Safari, Android Chrome, Desktop). Features tabbed interface with platform-specific guidance and visual icons.",
            },
        },
        layout: "fullscreen",
        backgrounds: { disable: true },
    },
    tags: ["autodocs"],
    argTypes: {
        open: {
            control: "boolean",
            description: "Whether the dialog is open",
        },
        onClose: {
            action: "onClose",
            description: "Callback when dialog is closed",
        },
        showBeforeInstallPrompt: {
            control: "boolean",
            description: "Simulate PWA install availability (shows different messaging)",
            table: {
                defaultValue: { summary: "false" },
            },
        },
    },
    args: {
        open: false,
        showBeforeInstallPrompt: false,
    },
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Interactive PWA installation dialog with platform-specific instructions.
 */
export const Showcase: Story = {
    render: (args) => {
        const [open, setOpen] = useState(args.open);

        const handleOpen = () => setOpen(true);
        const handleClose = () => {
            setOpen(false);
            args.onClose?.();
        };

        // Simulate beforeinstallprompt availability
        React.useEffect(() => {
            if (args.showBeforeInstallPrompt) {
                // Create a mock beforeinstallprompt event
                const mockEvent = new Event("beforeinstallprompt");
                Object.defineProperty(mockEvent, "prompt", {
                    value: () => Promise.resolve(),
                });
                window.dispatchEvent(mockEvent);
            }
        }, [args.showBeforeInstallPrompt]);

        return (
            <Box sx={{ display: "flex", flexDirection: "column", gap: 2, alignItems: "center" }}>
                <Box sx={{ 
                    p: 2, 
                    bgcolor: "info.light", 
                    color: "info.contrastText",
                    borderRadius: 2,
                    textAlign: "center",
                    maxWidth: 500,
                }}>
                    <h3 style={{ margin: "0 0 8px 0" }}>📱 PWA Installation Guide</h3>
                    <p style={{ margin: "0 0 12px 0", fontSize: "0.875rem" }}>
                        This dialog helps users install Vrooli as a Progressive Web App (PWA) 
                        with platform-specific instructions for iOS, Android, and Desktop.
                    </p>
                    <p style={{ margin: 0, fontSize: "0.75rem", opacity: 0.8 }}>
                        Use the controls to simulate different installation scenarios.
                    </p>
                </Box>

                <Button onClick={handleOpen} variant="primary" size="lg">
                    📱 Open Install Guide
                </Button>

                <InstallPWADialog 
                    open={open} 
                    onClose={handleClose}
                />
            </Box>
        );
    },
    args: {
        open: false,
        showBeforeInstallPrompt: false,
    },
};

/**
 * Multiple examples showing different PWA installation scenarios.
 */
export const Examples: Story = {
    render: () => {
        const [openDialog, setOpenDialog] = useState<string | null>(null);

        const examples = [
            {
                id: "basic",
                title: "📱 Basic Installation",
                description: "Standard PWA installation instructions",
                buttonText: "Open Basic Guide",
                props: {
                    showBeforeInstallPrompt: false,
                },
            },
            {
                id: "with-prompt",
                title: "⚡ With Install Prompt",
                description: "Simulates when browser shows native install banner",
                buttonText: "Open Enhanced Guide",
                props: {
                    showBeforeInstallPrompt: true,
                },
            },
        ];

        const handleOpen = (id: string) => {
            setOpenDialog(id);
        };

        const handleClose = () => {
            setOpenDialog(null);
        };

        return (
            <Box sx={{ display: "flex", flexDirection: "column", gap: 4 }}>
                <Box>
                    <h3 style={{ margin: "0 0 8px 0" }}>PWA Installation Dialog Examples</h3>
                    <p style={{ margin: 0, fontSize: "0.875rem", opacity: 0.7 }}>
                        Different scenarios for PWA installation guidance
                    </p>
                </Box>

                <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                    {examples.map((example) => (
                        <Box key={example.id} sx={{ 
                            p: 3, 
                            border: "1px solid", 
                            borderColor: "divider",
                            borderRadius: 2,
                            bgcolor: "background.paper",
                        }}>
                            <Box sx={{ mb: 2 }}>
                                <h4 style={{ margin: "0 0 4px 0" }}>{example.title}</h4>
                                <p style={{ margin: 0, fontSize: "0.875rem", opacity: 0.7 }}>
                                    {example.description}
                                </p>
                            </Box>
                            
                            <Button 
                                onClick={() => handleOpen(example.id)}
                                variant="outline"
                                size="sm"
                            >
                                {example.buttonText}
                            </Button>

                            <InstallPWADialog
                                open={openDialog === example.id}
                                onClose={handleClose}
                            />
                        </Box>
                    ))}
                </Box>
            </Box>
        );
    },
};

/**
 * Platform-specific installation instructions preview.
 */
export const PlatformGuides: Story = {
    render: () => {
        const [openDialog, setOpenDialog] = useState<string | null>(null);

        const platforms = [
            {
                id: "ios",
                title: "🍎 iOS Safari",
                description: "Installation guide for iPhone and iPad users",
                icon: "📱",
                details: [
                    "Open in Safari browser",
                    "Tap the Share button",
                    "Select 'Add to Home Screen'",
                    "Tap 'Add' to confirm",
                ],
            },
            {
                id: "android",
                title: "🤖 Android Chrome",
                description: "Installation guide for Android users",
                icon: "📲",
                details: [
                    "Open in Chrome browser",
                    "Tap the three dots menu",
                    "Select 'Add to Home screen'",
                    "Tap 'Add' to install",
                ],
            },
            {
                id: "desktop",
                title: "💻 Desktop",
                description: "Installation guide for desktop browsers",
                icon: "🖥️",
                details: [
                    "Look for install icon in address bar",
                    "Click the install button",
                    "Or use browser menu options",
                    "App opens in its own window",
                ],
            },
        ];

        const handleOpen = (id: string) => {
            setOpenDialog(id);
        };

        const handleClose = () => {
            setOpenDialog(null);
        };

        return (
            <Box sx={{ display: "flex", flexDirection: "column", gap: 4 }}>
                <Box>
                    <h3 style={{ margin: "0 0 8px 0" }}>Platform-Specific Installation Guides</h3>
                    <p style={{ margin: 0, fontSize: "0.875rem", opacity: 0.7 }}>
                        The dialog provides tailored instructions for each platform
                    </p>
                </Box>

                <Box sx={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))", gap: 3 }}>
                    {platforms.map((platform) => (
                        <Box key={platform.id} sx={{ 
                            p: 3, 
                            border: "1px solid", 
                            borderColor: "divider",
                            borderRadius: 2,
                            bgcolor: "background.paper",
                            display: "flex",
                            flexDirection: "column",
                        }}>
                            <Box sx={{ display: "flex", alignItems: "center", gap: 1, mb: 2 }}>
                                <span style={{ fontSize: "1.5rem" }}>{platform.icon}</span>
                                <h4 style={{ margin: 0 }}>{platform.title}</h4>
                            </Box>
                            
                            <p style={{ margin: "0 0 16px 0", fontSize: "0.875rem", opacity: 0.7 }}>
                                {platform.description}
                            </p>

                            <Box sx={{ mb: 3, flex: 1 }}>
                                <h5 style={{ margin: "0 0 8px 0", fontSize: "0.875rem" }}>Steps:</h5>
                                <ol style={{ margin: 0, paddingLeft: "20px" }}>
                                    {platform.details.map((step, index) => (
                                        <li key={index} style={{ fontSize: "0.8rem", marginBottom: "4px" }}>
                                            {step}
                                        </li>
                                    ))}
                                </ol>
                            </Box>
                            
                            <Button 
                                onClick={() => handleOpen(platform.id)}
                                variant="outline"
                                size="sm"
                                fullWidth
                            >
                                View Full Guide
                            </Button>
                        </Box>
                    ))}
                </Box>

                <InstallPWADialog
                    open={openDialog !== null}
                    onClose={handleClose}
                />
            </Box>
        );
    },
};

/**
 * Interactive example with live controls.
 */
export const Interactive: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const [showPrompt, setShowPrompt] = useState(false);

        const handleOpen = () => setIsOpen(true);
        const handleClose = () => setIsOpen(false);

        // Simulate PWA install availability based on control
        React.useEffect(() => {
            if (showPrompt) {
                const mockEvent = new Event("beforeinstallprompt");
                Object.defineProperty(mockEvent, "prompt", {
                    value: () => Promise.resolve(),
                });
                window.dispatchEvent(mockEvent);
            }
        }, [showPrompt]);

        return (
            <Box sx={{ display: "flex", flexDirection: "column", gap: 4, maxWidth: 600, mx: "auto" }}>
                {/* Controls */}
                <Box sx={{ p: 3, bgcolor: "action.hover", borderRadius: 2 }}>
                    <h4 style={{ margin: "0 0 16px 0" }}>Interactive Controls</h4>
                    
                    <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                        <label style={{ display: "flex", alignItems: "center", gap: 8 }}>
                            <input
                                type="checkbox"
                                checked={showPrompt}
                                onChange={(e) => setShowPrompt(e.target.checked)}
                            />
                            <span style={{ fontSize: "0.875rem" }}>
                                Simulate PWA Install Prompt Available
                            </span>
                        </label>

                        <Box sx={{ mt: 1 }}>
                            <Button 
                                onClick={handleOpen}
                                variant="primary"
                                disabled={isOpen}
                            >
                                {isOpen ? "Dialog Open" : "Open Installation Guide"}
                            </Button>
                        </Box>
                    </Box>
                </Box>

                {/* Info */}
                <Box sx={{ p: 2, bgcolor: "info.light", color: "info.contrastText", borderRadius: 2 }}>
                    <h5 style={{ margin: "0 0 8px 0" }}>💡 About PWA Installation</h5>
                    <ul style={{ margin: 0, paddingLeft: "20px", fontSize: "0.875rem" }}>
                        <li>Progressive Web Apps can be installed like native apps</li>
                        <li>Each platform has different installation methods</li>
                        <li>The dialog provides clear, step-by-step instructions</li>
                        <li>Features tabbed interface for iOS, Android, and Desktop</li>
                        <li>Includes visual icons and helpful tips</li>
                    </ul>
                </Box>

                {/* Dialog Status */}
                <Box sx={{ 
                    p: 2, 
                    border: "2px dashed", 
                    borderColor: isOpen ? "success.main" : "divider",
                    borderRadius: 2,
                    textAlign: "center",
                    bgcolor: isOpen ? "success.light" : "transparent",
                    color: isOpen ? "success.contrastText" : "inherit",
                }}>
                    <p style={{ margin: 0, fontSize: "0.875rem" }}>
                        {isOpen ? "📱 Installation guide is currently open" : "📱 Click button above to open installation guide"}
                    </p>
                </Box>

                <InstallPWADialog 
                    open={isOpen} 
                    onClose={handleClose}
                />
            </Box>
        );
    },
};

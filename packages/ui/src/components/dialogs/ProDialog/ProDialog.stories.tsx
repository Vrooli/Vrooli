import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { Box } from "@mui/material";
import { Button } from "../../buttons/Button.js";
import { ProDialog } from "./ProDialog.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

const meta: Meta<typeof ProDialog> = {
    title: "Components/Dialogs/ProDialog",
    component: ProDialog,
    parameters: {
        docs: {
            description: {
                component: "A promotional dialog encouraging users to upgrade to Pro features. Features gradient background, animated effects, and navigation to upgrade/credits pages. Includes decorative triangle elements and breadcrumb navigation.",
            },
        },
        layout: "fullscreen",
        backgrounds: { disable: true },
    },
    tags: ["autodocs"],
    argTypes: {
        isOpen: {
            control: "boolean",
            description: "Whether the dialog is open",
        },
        onClose: {
            action: "onClose",
            description: "Callback when dialog is closed",
        },
        userCredits: {
            control: { type: "number", min: 0, max: 10000, step: 100 },
            description: "Current user credits (for context in description)",
            table: {
                defaultValue: { summary: "0" },
            },
        },
        userHasPremium: {
            control: "boolean",
            description: "Whether user has premium subscription (affects messaging)",
            table: {
                defaultValue: { summary: "false" },
            },
        },
    },
    args: {
        isOpen: false,
        userCredits: 0,
        userHasPremium: false,
    },
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Interactive Pro upgrade dialog with animated effects.
 */
export const Showcase: Story = {
    render: (args) => {
        const [open, setOpen] = useState(args.isOpen);

        const handleOpen = () => setOpen(true);
        const handleClose = () => {
            setOpen(false);
            args.onClose?.();
        };

        return (
            <Box sx={{ display: "flex", flexDirection: "column", gap: 2, alignItems: "center" }}>
                <Box sx={{ 
                    p: 2, 
                    bgcolor: "primary.light", 
                    color: "primary.contrastText",
                    borderRadius: 2,
                    textAlign: "center",
                    maxWidth: 500,
                }}>
                    <h3 style={{ margin: "0 0 8px 0" }}>✨ Pro Features Dialog</h3>
                    <p style={{ margin: "0 0 12px 0", fontSize: "0.875rem" }}>
                        This dialog encourages users to upgrade to Pro features with an engaging 
                        visual design, animated effects, and clear calls-to-action.
                    </p>
                    <p style={{ margin: 0, fontSize: "0.75rem", opacity: 0.8 }}>
                        Features gradient background, pulsing animations, and breadcrumb navigation.
                    </p>
                </Box>

                <Button onClick={handleOpen} variant="primary" size="lg">
                    ✨ Open Pro Dialog
                </Button>

                <ProDialog 
                    isOpen={open} 
                    onClose={handleClose}
                />
            </Box>
        );
    },
    args: {
        isOpen: false,
        userCredits: 0,
        userHasPremium: false,
    },
};

/**
 * Examples showing different user states and scenarios.
 */
export const Examples: Story = {
    render: () => {
        const [openDialog, setOpenDialog] = useState<string | null>(null);

        const examples = [
            {
                id: "free-user",
                title: "💰 Free User",
                description: "New user with no credits or premium",
                buttonText: "View as Free User",
                props: {
                    userCredits: 0,
                    userHasPremium: false,
                },
                info: "Shows full promotional content encouraging upgrade",
            },
            {
                id: "some-credits",
                title: "💳 User with Credits",
                description: "User has some credits but no premium",
                buttonText: "View with Credits",
                props: {
                    userCredits: 500,
                    userHasPremium: false,
                },
                info: "Still shows upgrade path but acknowledges credits",
            },
            {
                id: "premium-user",
                title: "⭐ Premium User",
                description: "User with premium subscription",
                buttonText: "View as Premium",
                props: {
                    userCredits: 2000,
                    userHasPremium: true,
                },
                info: "May show different messaging for premium users",
            },
            {
                id: "low-credits",
                title: "⚠️ Low Credits Alert",
                description: "Premium user running low on credits",
                buttonText: "View Low Credits",
                props: {
                    userCredits: 50,
                    userHasPremium: true,
                },
                info: "Emphasizes credit purchase for existing premium users",
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
                    <h3 style={{ margin: "0 0 8px 0" }}>Pro Dialog User Scenarios</h3>
                    <p style={{ margin: 0, fontSize: "0.875rem", opacity: 0.7 }}>
                        Different messaging based on user subscription and credit status
                    </p>
                </Box>

                <Box sx={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))", gap: 3 }}>
                    {examples.map((example) => (
                        <Box key={example.id} sx={{ 
                            p: 3, 
                            border: "1px solid", 
                            borderColor: "divider",
                            borderRadius: 2,
                            bgcolor: "background.paper",
                            display: "flex",
                            flexDirection: "column",
                        }}>
                            <Box sx={{ mb: 2 }}>
                                <h4 style={{ margin: "0 0 4px 0" }}>{example.title}</h4>
                                <p style={{ margin: "0 0 8px 0", fontSize: "0.875rem", opacity: 0.7 }}>
                                    {example.description}
                                </p>
                            </Box>
                            
                            <Box sx={{ mb: 2, p: 1.5, bgcolor: "action.hover", borderRadius: 1 }}>
                                <Box sx={{ fontSize: "0.75rem", opacity: 0.8, mb: 1 }}>User State:</Box>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 0.5, fontSize: "0.8rem" }}>
                                    <div>Credits: <strong>{example.props.userCredits}</strong></div>
                                    <div>Premium: <strong>{example.props.userHasPremium ? "Yes" : "No"}</strong></div>
                                </Box>
                            </Box>

                            <Box sx={{ mb: 3, flex: 1 }}>
                                <p style={{ margin: 0, fontSize: "0.8rem", fontStyle: "italic" }}>
                                    {example.info}
                                </p>
                            </Box>
                            
                            <Button 
                                onClick={() => handleOpen(example.id)}
                                variant="outline"
                                size="sm"
                                fullWidth
                            >
                                {example.buttonText}
                            </Button>

                            <ProDialog
                                isOpen={openDialog === example.id}
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
 * Visual design showcase highlighting the dialog's aesthetic features.
 */
export const VisualFeatures: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);

        const handleOpen = () => setIsOpen(true);
        const handleClose = () => setIsOpen(false);

        const features = [
            {
                icon: "🎨",
                title: "Gradient Background",
                description: "Beautiful gradient from gray to blue-green",
            },
            {
                icon: "📐",
                title: "Geometric Elements",
                description: "Decorative triangles with custom gradients",
            },
            {
                icon: "✨",
                title: "Pulsing Animation",
                description: "Animated upgrade button with glow effect",
            },
            {
                icon: "🧭",
                title: "Breadcrumb Navigation",
                description: "Easy access to features, donate, FAQ, and terms",
            },
            {
                icon: "📱",
                title: "Responsive Design",
                description: "Optimized for mobile and desktop viewing",
            },
            {
                icon: "🎯",
                title: "Clear CTAs",
                description: "Prominent upgrade and buy credits buttons",
            },
        ];

        return (
            <Box sx={{ display: "flex", flexDirection: "column", gap: 4 }}>
                <Box sx={{ textAlign: "center" }}>
                    <h3 style={{ margin: "0 0 8px 0" }}>🎨 Visual Design Features</h3>
                    <p style={{ margin: "0 0 24px 0", fontSize: "0.875rem", opacity: 0.7 }}>
                        The Pro dialog combines engaging visuals with clear messaging
                    </p>
                    
                    <Button onClick={handleOpen} variant="primary" size="lg">
                        ✨ View Pro Dialog Design
                    </Button>
                </Box>

                <Box sx={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(250px, 1fr))", gap: 2 }}>
                    {features.map((feature, index) => (
                        <Box key={index} sx={{ 
                            p: 2, 
                            border: "1px solid", 
                            borderColor: "divider",
                            borderRadius: 2,
                            bgcolor: "background.paper",
                            textAlign: "center",
                        }}>
                            <Box sx={{ fontSize: "1.5rem", mb: 1 }}>{feature.icon}</Box>
                            <h5 style={{ margin: "0 0 4px 0", fontSize: "0.9rem" }}>{feature.title}</h5>
                            <p style={{ margin: 0, fontSize: "0.8rem", opacity: 0.7 }}>
                                {feature.description}
                            </p>
                        </Box>
                    ))}
                </Box>

                <ProDialog 
                    isOpen={isOpen} 
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
        const [userCredits, setUserCredits] = useState(0);
        const [userHasPremium, setUserHasPremium] = useState(false);

        const handleOpen = () => setIsOpen(true);
        const handleClose = () => setIsOpen(false);

        const presets = [
            { name: "New User", credits: 0, premium: false },
            { name: "Some Credits", credits: 500, premium: false },
            { name: "Premium User", credits: 2000, premium: true },
            { name: "Low Credits", credits: 50, premium: true },
        ];

        return (
            <Box sx={{ display: "flex", flexDirection: "column", gap: 4, maxWidth: 600, mx: "auto" }}>
                {/* Controls */}
                <Box sx={{ p: 3, bgcolor: "action.hover", borderRadius: 2 }}>
                    <h4 style={{ margin: "0 0 16px 0" }}>Interactive Controls</h4>
                    
                    <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                        <Box>
                            <label style={{ display: "block", marginBottom: 4, fontSize: "0.875rem" }}>
                                User Credits: {userCredits}
                            </label>
                            <input
                                type="range"
                                min="0"
                                max="5000"
                                step="50"
                                value={userCredits}
                                onChange={(e) => setUserCredits(Number(e.target.value))}
                                style={{ width: "100%" }}
                            />
                        </Box>

                        <label style={{ display: "flex", alignItems: "center", gap: 8 }}>
                            <input
                                type="checkbox"
                                checked={userHasPremium}
                                onChange={(e) => setUserHasPremium(e.target.checked)}
                            />
                            <span style={{ fontSize: "0.875rem" }}>
                                User has Premium subscription
                            </span>
                        </label>

                        <Box>
                            <p style={{ margin: "8px 0 4px 0", fontSize: "0.875rem" }}>Quick Presets:</p>
                            <Box sx={{ display: "flex", gap: 1, flexWrap: "wrap" }}>
                                {presets.map((preset, index) => (
                                    <button
                                        key={index}
                                        onClick={() => {
                                            setUserCredits(preset.credits);
                                            setUserHasPremium(preset.premium);
                                        }}
                                        style={{
                                            padding: "4px 8px",
                                            fontSize: "0.75rem",
                                            borderRadius: "4px",
                                            border: "1px solid #ccc",
                                            background: "white",
                                            cursor: "pointer",
                                        }}
                                    >
                                        {preset.name}
                                    </button>
                                ))}
                            </Box>
                        </Box>

                        <Box sx={{ mt: 1 }}>
                            <Button 
                                onClick={handleOpen}
                                variant="primary"
                                disabled={isOpen}
                                fullWidth
                            >
                                {isOpen ? "Dialog Open" : "Open Pro Dialog"}
                            </Button>
                        </Box>
                    </Box>
                </Box>

                {/* Current State Display */}
                <Box sx={{ 
                    p: 2, 
                    border: "2px solid", 
                    borderColor: userHasPremium ? "success.main" : "warning.main",
                    borderRadius: 2,
                    bgcolor: userHasPremium ? "success.light" : "warning.light",
                    color: userHasPremium ? "success.contrastText" : "warning.contrastText",
                }}>
                    <h5 style={{ margin: "0 0 8px 0" }}>
                        {userHasPremium ? "⭐ Premium User" : "💰 Free User"}
                    </h5>
                    <p style={{ margin: 0, fontSize: "0.875rem" }}>
                        <strong>Credits:</strong> {userCredits} | 
                        <strong> Status:</strong> {userHasPremium ? "Premium Subscriber" : "Free Tier"}
                    </p>
                </Box>

                {/* Info */}
                <Box sx={{ p: 2, bgcolor: "info.light", color: "info.contrastText", borderRadius: 2 }}>
                    <h5 style={{ margin: "0 0 8px 0" }}>💡 About Pro Features</h5>
                    <ul style={{ margin: 0, paddingLeft: "20px", fontSize: "0.875rem" }}>
                        <li>Pro users get monthly credits and priority in runs</li>
                        <li>Access to bots, routine execution, and advanced features</li>
                        <li>Dialog adjusts messaging based on user status</li>
                        <li>Beautiful gradient design with animated elements</li>
                        <li>Clear upgrade path with breadcrumb navigation</li>
                    </ul>
                </Box>

                <ProDialog 
                    isOpen={isOpen} 
                    onClose={handleClose}
                />
            </Box>
        );
    },
};

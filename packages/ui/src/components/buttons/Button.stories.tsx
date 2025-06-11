import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { IconCommon } from "../../icons/Icons.js";
import { OrbitalSpinner } from "../indicators/CircularProgress.js";
import { Button, ButtonFactory } from "./Button.js";
import type { ButtonVariant, ButtonSize } from "./Button.js";

const meta: Meta<typeof Button> = {
    title: "Components/Buttons/Button",
    component: Button,
    parameters: {
        layout: "centered",
    },
    argTypes: {
        variant: {
            control: "select",
            options: ["primary", "secondary", "outline", "ghost", "danger", "space"],
            description: "The visual style variant of the button",
            table: {
                type: { summary: "ButtonVariant" },
                defaultValue: { summary: "primary" },
            },
        },
        size: {
            control: "select",
            options: ["sm", "md", "lg", "icon"],
            description: "The size of the button",
            table: {
                type: { summary: "ButtonSize" },
                defaultValue: { summary: "md" },
            },
        },
        isLoading: {
            control: "boolean",
            description: "Whether the button is in a loading state",
            table: {
                type: { summary: "boolean" },
                defaultValue: { summary: "false" },
            },
        },
        loadingIndicator: {
            control: "select",
            options: ["circular", "orbital"],
            description: "The type of loading indicator to display",
            table: {
                type: { summary: '"circular" | "orbital"' },
                defaultValue: { summary: "circular" },
            },
        },
        disabled: {
            control: "boolean",
            description: "Whether the button is disabled",
            table: {
                type: { summary: "boolean" },
                defaultValue: { summary: "false" },
            },
        },
        fullWidth: {
            control: "boolean",
            description: "Whether the button should take full width of its container",
            table: {
                type: { summary: "boolean" },
                defaultValue: { summary: "false" },
            },
        },
        startIcon: {
            control: "select",
            options: [null, "Save", "Add", "Edit", "Delete", "Star", "Rocket", "Team", "Bot", "Routine"],
            mapping: {
                null: null,
                Save: <IconCommon name="Save" />,
                Add: <IconCommon name="Add" />,
                Edit: <IconCommon name="Edit" />,
                Delete: <IconCommon name="Delete" />,
                Star: <IconCommon name="Star" />,
                Rocket: <IconCommon name="Rocket" />,
                Team: <IconCommon name="Team" />,
                Bot: <IconCommon name="Bot" />,
                Routine: <IconCommon name="Routine" />,
            },
            description: "Icon to display at the start of the button",
            table: {
                type: { summary: "ReactNode" },
                defaultValue: { summary: "undefined" },
            },
        },
        endIcon: {
            control: "select",
            options: [null, "ArrowForward", "ChevronRight", "ExternalLink", "Download", "Upload"],
            mapping: {
                null: null,
                ArrowForward: <IconCommon name="ArrowForward" />,
                ChevronRight: <IconCommon name="ChevronRight" />,
                ExternalLink: <IconCommon name="ExternalLink" />,
                Download: <IconCommon name="Download" />,
                Upload: <IconCommon name="Upload" />,
            },
            description: "Icon to display at the end of the button",
            table: {
                type: { summary: "ReactNode" },
                defaultValue: { summary: "undefined" },
            },
        },
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default story with controls
export const Default: Story = {
    args: {
        children: "Button",
        variant: "primary",
        size: "md",
        isLoading: false,
        loadingIndicator: "circular",
        disabled: false,
        fullWidth: false,
    },
    decorators: [
        (Story, context) => {
            const backgroundColor = (context.globals as any)?.backgrounds?.value || "#ffffff";
            return (
                <Box sx={{ 
                    minWidth: 300, 
                    minHeight: 200, 
                    display: "flex", 
                    alignItems: "center", 
                    justifyContent: "center",
                    backgroundColor,
                    padding: 4,
                }}>
                    <Story />
                </Box>
            );
        },
    ],
};

// All variants showcase
export const AllVariants: Story = {
    render: () => (
        <Box sx={{ display: "flex", flexDirection: "column", gap: 4, p: 4, minWidth: 800 }}>
            <Typography variant="h4">Button Variants</Typography>
            
            <Box sx={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 2 }}>
                {(["primary", "secondary", "outline", "ghost", "danger", "space"] as ButtonVariant[]).map(variant => (
                    ["sm", "md", "lg", "icon"].map(size => (
                        <Button 
                            key={`${variant}-${size}`}
                            variant={variant} 
                            size={size as ButtonSize}
                        >
                            {size === "icon" ? <IconCommon name="Star" /> : `${variant} ${size}`}
                        </Button>
                    ))
                ))}
            </Box>
        </Box>
    ),
};

// Interactive loading demo
export const InteractiveLoading: Story = {
    render: () => {
        const [loadingStates, setLoadingStates] = useState<Record<string, boolean>>({});

        const handleClick = (key: string) => {
            setLoadingStates(prev => ({ ...prev, [key]: true }));
            setTimeout(() => {
                setLoadingStates(prev => ({ ...prev, [key]: false }));
            }, 3000);
        };

        return (
            <Box sx={{ p: 4, display: "flex", flexDirection: "column", gap: 3, maxWidth: 600 }}>
                <Typography variant="h5">Click buttons to see loading states</Typography>
                
                <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                    {(["primary", "secondary", "outline", "ghost", "danger", "space"] as ButtonVariant[]).map(variant => (
                        <Button
                            key={variant}
                            variant={variant}
                            isLoading={loadingStates[variant] || false}
                            loadingIndicator={variant === "space" ? "orbital" : "circular"}
                            onClick={() => handleClick(variant)}
                            startIcon={!loadingStates[variant] && <IconCommon name="Save" />}
                        >
                            {loadingStates[variant] ? "Processing..." : `${variant} Button`}
                        </Button>
                    ))}
                </Box>
            </Box>
        );
    },
};

// Space variant showcase
export const SpaceVariant: Story = {
    render: () => {
        const [loading, setLoading] = useState<Record<string, boolean>>({});

        const handleClick = (key: string) => {
            setLoading(prev => ({ ...prev, [key]: true }));
            setTimeout(() => setLoading(prev => ({ ...prev, [key]: false })), 3000);
        };

        return (
            <Box sx={{ 
                p: 4, 
                bgcolor: "#000",
                borderRadius: 2,
                minHeight: 400,
                background: 'radial-gradient(ellipse at center, #001122 0%, #000 100%)',
            }}>
                <Typography variant="h4" sx={{ color: "#fff", textAlign: "center", mb: 4 }}>
                    Space-Inspired Button
                </Typography>
                
                <Box sx={{ display: "flex", justifyContent: "center", gap: 2, flexWrap: "wrap" }}>
                    <Button
                        variant="space"
                        size="lg"
                        loadingIndicator="orbital"
                        isLoading={loading.hero || false}
                        onClick={() => handleClick("hero")}
                        startIcon={!loading.hero && <IconCommon name="Rocket" />}
                    >
                        {loading.hero ? "Launching..." : "Launch Into Space"}
                    </Button>
                    
                    <Button variant="space" size="sm">Small Space</Button>
                    <Button variant="space" size="md">Medium Space</Button>
                    <Button variant="space" size="icon"><IconCommon name="Star" /></Button>
                </Box>

                <Typography variant="body2" sx={{ color: "#666", textAlign: "center", mt: 4 }}>
                    Features clean gradient background with shimmer hover effect and click ripples
                </Typography>
            </Box>
        );
    },
};

// Accessibility demo
export const Accessibility: Story = {
    render: () => (
        <Box sx={{ p: 4, display: "flex", flexDirection: "column", gap: 3 }}>
            <Typography variant="h5">Accessibility Features</Typography>
            
            <Box sx={{ display: "flex", gap: 2, alignItems: "center" }}>
                <Button aria-label="Save document">
                    <IconCommon name="Save" />
                </Button>
                <Typography variant="body2">Icon-only button with aria-label</Typography>
            </Box>
            
            <Box sx={{ display: "flex", gap: 2, alignItems: "center" }}>
                <Button isLoading aria-busy="true">
                    Loading...
                </Button>
                <Typography variant="body2">Loading state with aria-busy</Typography>
            </Box>
            
            <Box sx={{ display: "flex", gap: 2, alignItems: "center" }}>
                <Button disabled aria-disabled="true">
                    Disabled
                </Button>
                <Typography variant="body2">Disabled state with aria-disabled</Typography>
            </Box>
        </Box>
    ),
};

// Button factory demo
export const ButtonFactoryShowcase: Story = {
    render: () => (
        <Box sx={{ p: 4, display: "flex", flexDirection: "column", gap: 3 }}>
            <Typography variant="h5">Button Factory Pattern</Typography>
            <Typography variant="body2" color="text.secondary">
                Pre-configured button components for common use cases
            </Typography>
            
            <Box sx={{ display: "flex", gap: 2, flexWrap: "wrap" }}>
                <ButtonFactory.Primary>Primary Action</ButtonFactory.Primary>
                <ButtonFactory.Secondary>Secondary Action</ButtonFactory.Secondary>
                <ButtonFactory.Outline>Outline Style</ButtonFactory.Outline>
                <ButtonFactory.Ghost>Ghost Style</ButtonFactory.Ghost>
                <ButtonFactory.Danger>Danger Action</ButtonFactory.Danger>
                <ButtonFactory.Space>Space Theme</ButtonFactory.Space>
            </Box>
        </Box>
    ),
};

// Storybook global configuration for background color
export const globalTypes = {
    backgrounds: {
        defaultValue: { name: "light", value: "#ffffff" },
        toolbar: {
            title: "Background",
            icon: "paintbrush",
            items: [
                { name: "light", value: "#ffffff" },
                { name: "dark", value: "#1a1a1a" },
                { name: "vrooli", value: "#001122" },
            ],
            dynamicTitle: true,
        },
    },
};
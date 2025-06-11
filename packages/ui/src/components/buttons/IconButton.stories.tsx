import type { Meta, StoryObj } from "@storybook/react";
import React from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { IconCommon } from "../../icons/Icons.js";
import { IconButton } from "./IconButton.js";
import type { IconButtonVariant } from "./IconButton.js";

const meta: Meta<typeof IconButton> = {
    title: "Components/Buttons/IconButton",
    component: IconButton,
    parameters: {
        layout: "centered",
    },
    argTypes: {
        variant: {
            control: "select",
            options: ["solid", "transparent", "space"],
            description: "The visual style variant of the icon button",
            table: {
                type: { summary: "IconButtonVariant" },
                defaultValue: { summary: "transparent" },
            },
        },
        size: {
            control: "select",
            options: ["sm", "md", "lg", 24, 32, 48, 64, 80],
            description: "Size of the button (predefined or custom in pixels)",
            table: {
                type: { summary: '"sm" | "md" | "lg" | number' },
                defaultValue: { summary: "md" },
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
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default story with controls
export const Default: Story = {
    args: {
        variant: "solid",
        size: "md",
        disabled: false,
        children: <IconCommon name="Star" />,
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
            <Typography variant="h4">Icon Button Variants</Typography>
            
            {(["solid", "transparent", "space"] as IconButtonVariant[]).map(variant => (
                <Box key={variant} sx={{ 
                    p: 3, 
                    borderRadius: 2,
                    backgroundColor: variant === "space" ? "#001122" : "#f5f5f5",
                }}>
                    <Typography 
                        variant="h6" 
                        sx={{ 
                            mb: 2,
                            color: variant === "space" ? "#fff" : "#000",
                            textTransform: "capitalize",
                        }}
                    >
                        {variant} Variant
                    </Typography>
                    <Box sx={{ display: "flex", gap: 2, alignItems: "center" }}>
                        <IconButton variant={variant} size="sm">
                            <IconCommon name="Save" />
                        </IconButton>
                        <IconButton variant={variant} size="md">
                            <IconCommon name="Edit" />
                        </IconButton>
                        <IconButton variant={variant} size="lg">
                            <IconCommon name="Delete" />
                        </IconButton>
                        <IconButton variant={variant} size={80}>
                            <IconCommon name="Star" />
                        </IconButton>
                        <IconButton variant={variant} size="md" disabled>
                            <IconCommon name="Add" />
                        </IconButton>
                    </Box>
                </Box>
            ))}
        </Box>
    ),
};

// Icon showcase
export const IconShowcase: Story = {
    render: () => {
        const icons = [
            "Save", "Edit", "Delete", "Add", "Remove", 
            "Star", "StarOutline", "Team", "Bot", "Routine",
            "Search", "Settings", "Close", "Check", "Info"
        ];

        return (
            <Box sx={{ display: "flex", flexDirection: "column", gap: 3, p: 4 }}>
                <Typography variant="h5">Icon Button Gallery</Typography>
                
                <Box sx={{ display: "grid", gridTemplateColumns: "repeat(5, 1fr)", gap: 2 }}>
                    {icons.map(icon => (
                        <Box
                            key={icon}
                            sx={{
                                display: "flex",
                                flexDirection: "column",
                                alignItems: "center",
                                gap: 1,
                                p: 2,
                                borderRadius: 1,
                                backgroundColor: "#f0f0f0",
                            }}
                        >
                            <IconButton variant="solid" size="md">
                                <IconCommon name={icon} />
                            </IconButton>
                            <Typography variant="caption">{icon}</Typography>
                        </Box>
                    ))}
                </Box>
            </Box>
        );
    },
};

// Physical button showcase
export const PhysicalButton: Story = {
    render: () => (
        <Box sx={{ 
            p: 4, 
            bgcolor: "#2a2a2a",
            borderRadius: 2,
            minHeight: 400,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            justifyContent: "center",
            gap: 4,
        }}>
            <Typography variant="h4" sx={{ color: "#fff", textAlign: "center" }}>
                3D Physical Icon Buttons
            </Typography>
            <Typography variant="body1" sx={{ color: "#aaa", textAlign: "center", maxWidth: 500 }}>
                Solid variant features a physical appearance with hover lift effect and press-down animation
            </Typography>
            
            <Box sx={{ display: "flex", gap: 3, alignItems: "center" }}>
                <IconButton variant="solid" size="lg">
                    <IconCommon name="Rocket" />
                </IconButton>
                <IconButton variant="solid" size="lg">
                    <IconCommon name="Star" />
                </IconButton>
                <IconButton variant="solid" size="lg">
                    <IconCommon name="Save" />
                </IconButton>
                <IconButton variant="solid" size="lg" disabled>
                    <IconCommon name="Delete" />
                </IconButton>
            </Box>

            <Typography variant="body2" sx={{ color: "#666", textAlign: "center" }}>
                Hover and click to see 3D effects
            </Typography>
        </Box>
    ),
};

// Space variant showcase
export const SpaceVariant: Story = {
    render: () => (
        <Box sx={{ 
            p: 4, 
            bgcolor: "#000",
            borderRadius: 2,
            minHeight: 400,
            background: 'radial-gradient(ellipse at center, #001122 0%, #000 100%)',
        }}>
            <Typography variant="h4" sx={{ color: "#fff", textAlign: "center", mb: 4 }}>
                Space-Themed Icon Buttons
            </Typography>
            
            <Box sx={{ display: "flex", justifyContent: "center", gap: 3, flexWrap: "wrap" }}>
                <IconButton variant="space" size="sm">
                    <IconCommon name="Star" />
                </IconButton>
                <IconButton variant="space" size="md">
                    <IconCommon name="Rocket" />
                </IconButton>
                <IconButton variant="space" size="lg">
                    <IconCommon name="Bot" />
                </IconButton>
                <IconButton variant="space" size={80}>
                    <IconCommon name="Team" />
                </IconButton>
            </Box>

            <Typography variant="body2" sx={{ color: "#666", textAlign: "center", mt: 4 }}>
                Features gradient background with shimmer hover effect and click ripples
            </Typography>
        </Box>
    ),
};

// Interactive states demo
export const InteractiveStates: Story = {
    render: () => {
        const [clickCounts, setClickCounts] = React.useState<Record<string, number>>({});

        const handleClick = (key: string) => {
            setClickCounts(prev => ({
                ...prev,
                [key]: (prev[key] || 0) + 1
            }));
        };

        return (
            <Box sx={{ p: 4, display: "flex", flexDirection: "column", gap: 3 }}>
                <Typography variant="h5">Interactive Icon Buttons</Typography>
                <Typography variant="body2" color="text.secondary">
                    Click the buttons to see click counts
                </Typography>
                
                <Box sx={{ display: "flex", gap: 2, alignItems: "center" }}>
                    {(["solid", "transparent", "space"] as IconButtonVariant[]).map(variant => (
                        <Box key={variant} sx={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 1 }}>
                            <IconButton
                                variant={variant}
                                size="md"
                                onClick={() => handleClick(variant)}
                            >
                                <IconCommon name="Add" />
                            </IconButton>
                            <Typography variant="caption">
                                {variant}: {clickCounts[variant] || 0}
                            </Typography>
                        </Box>
                    ))}
                </Box>
            </Box>
        );
    },
};
import type { Meta, StoryObj } from "@storybook/react";
import Box from "@mui/material/Box";
import { useState } from "react";
import { signedInNoPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { MenuTitle } from "./MenuTitle.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

const meta: Meta<typeof MenuTitle> = {
    title: "Components/Dialogs/MenuTitle",
    component: MenuTitle,
    parameters: {
        layout: "fullscreen",
        backgrounds: { disable: true },
        docs: {
            story: {
                inline: false,
                iframeHeight: 400,
            },
        },
        session: signedInNoPremiumWithCreditsSession,
    },
    tags: ["autodocs"],
    argTypes: {
        title: {
            control: { type: "text" },
            description: "Title text to display",
        },
        ariaLabel: {
            control: { type: "text" },
            description: "ARIA label for accessibility",
        },
        onClose: {
            action: "menu-closed",
            description: "Callback when close button is clicked",
        },
        // Title props
        variant: {
            control: { type: "select" },
            options: ["header", "subheader", "body", "caption"],
            description: "Title variant",
        },
        // Style props
        sxs: {
            control: { type: "object" },
            description: "Custom styles",
        },
    },
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Showcase with controls
export const Showcase: Story = {
    render: (args) => {
        const [isVisible, setIsVisible] = useState(true);

        const handleClose = () => {
            setIsVisible(false);
            args.onClose?.();
            // Reset visibility after a delay for demo purposes
            setTimeout(() => setIsVisible(true), 1000);
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px", width: "100%" }}>
                <Box
                    sx={{
                        width: "400px",
                        backgroundColor: "var(--background-paper)",
                        borderRadius: "8px",
                        overflow: "hidden",
                        border: "1px solid var(--divider)",
                    }}
                >
                    {isVisible && (
                        <MenuTitle
                            {...args}
                            onClose={handleClose}
                        />
                    )}
                    <Box sx={{ p: 2, textAlign: "center", color: "var(--text-secondary)" }}>
                        <p>Menu content would go here</p>
                    </Box>
                </Box>
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Title State: {isVisible ? "Visible" : "Hidden (will reappear)"}</p>
                    <p>Click the X button to close</p>
                </div>
            </div>
        );
    },
    args: {
        title: "Sample Menu Title",
        ariaLabel: "sample-menu-title",
        variant: "subheader",
    },
};

// Short title
export const ShortTitle: Story = {
    render: () => {
        const handleClose = () => console.log("Menu closed");

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Box
                    sx={{
                        width: "300px",
                        backgroundColor: "var(--background-paper)",
                        borderRadius: "8px",
                        overflow: "hidden",
                        border: "1px solid var(--divider)",
                    }}
                >
                    <MenuTitle
                        title="Actions"
                        ariaLabel="actions-menu"
                        onClose={handleClose}
                        variant="subheader"
                    />
                    <Box sx={{ p: 2, textAlign: "center", color: "var(--text-secondary)" }}>
                        <p>Simple short title</p>
                    </Box>
                </Box>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Menu title with a short, simple title.",
            },
        },
    },
};

// Long title
export const LongTitle: Story = {
    render: () => {
        const handleClose = () => console.log("Menu closed");

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Box
                    sx={{
                        width: "350px",
                        backgroundColor: "var(--background-paper)",
                        borderRadius: "8px",
                        overflow: "hidden",
                        border: "1px solid var(--divider)",
                    }}
                >
                    <MenuTitle
                        title="This is a Very Long Menu Title That Might Wrap"
                        ariaLabel="long-title-menu"
                        onClose={handleClose}
                        variant="subheader"
                    />
                    <Box sx={{ p: 2, textAlign: "center", color: "var(--text-secondary)" }}>
                        <p>Long title handling</p>
                    </Box>
                </Box>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Menu title with a long title to test text wrapping and layout.",
            },
        },
    },
};

// Different title variants
export const TitleVariants: Story = {
    render: () => {
        const handleClose = () => console.log("Menu closed");

        const variants = ["header", "subheader", "body", "caption"] as const;

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                {variants.map((variant) => (
                    <Box
                        key={variant}
                        sx={{
                            width: "300px",
                            backgroundColor: "var(--background-paper)",
                            borderRadius: "8px",
                            overflow: "hidden",
                            border: "1px solid var(--divider)",
                        }}
                    >
                        <MenuTitle
                            title={`${variant.charAt(0).toUpperCase() + variant.slice(1)} Title`}
                            ariaLabel={`${variant}-menu`}
                            onClose={handleClose}
                            variant={variant}
                        />
                    </Box>
                ))}
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Different title variants (header, subheader, body, caption).",
            },
        },
    },
};

// Custom styled
export const CustomStyled: Story = {
    render: () => {
        const handleClose = () => console.log("Menu closed");

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Box
                    sx={{
                        width: "350px",
                        backgroundColor: "var(--background-paper)",
                        borderRadius: "8px",
                        overflow: "hidden",
                        border: "1px solid var(--divider)",
                    }}
                >
                    <MenuTitle
                        title="Custom Styled Title"
                        ariaLabel="custom-styled-menu"
                        onClose={handleClose}
                        variant="subheader"
                        sxs={{
                            root: {
                                backgroundColor: "#2196f3",
                                color: "white",
                                "& .MuiIconButton-root": {
                                    color: "white",
                                },
                            },
                        }}
                    />
                    <Box sx={{ p: 2, textAlign: "center", color: "var(--text-secondary)" }}>
                        <p>Custom background and text color</p>
                    </Box>
                </Box>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Menu title with custom styling applied through the sxs prop.",
            },
        },
    },
};

// In narrow container
export const NarrowContainer: Story = {
    render: () => {
        const handleClose = () => console.log("Menu closed");

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Box
                    sx={{
                        width: "200px",
                        backgroundColor: "var(--background-paper)",
                        borderRadius: "8px",
                        overflow: "hidden",
                        border: "1px solid var(--divider)",
                    }}
                >
                    <MenuTitle
                        title="Narrow Container Title"
                        ariaLabel="narrow-menu"
                        onClose={handleClose}
                        variant="subheader"
                    />
                    <Box sx={{ p: 2, textAlign: "center", color: "var(--text-secondary)" }}>
                        <p style={{ fontSize: "12px" }}>Narrow layout</p>
                    </Box>
                </Box>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Menu title in a narrow container to test responsive layout.",
            },
        },
    },
};

// Wide container
export const WideContainer: Story = {
    render: () => {
        const handleClose = () => console.log("Menu closed");

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Box
                    sx={{
                        width: "600px",
                        backgroundColor: "var(--background-paper)",
                        borderRadius: "8px",
                        overflow: "hidden",
                        border: "1px solid var(--divider)",
                    }}
                >
                    <MenuTitle
                        title="Wide Container Menu Title"
                        ariaLabel="wide-menu"
                        onClose={handleClose}
                        variant="subheader"
                    />
                    <Box sx={{ p: 2, textAlign: "center", color: "var(--text-secondary)" }}>
                        <p>Wide layout with plenty of space</p>
                    </Box>
                </Box>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Menu title in a wide container to test layout with lots of space.",
            },
        },
    },
};

// Multiple titles (design comparison)
export const MultipleTitles: Story = {
    render: () => {
        const handleClose = () => console.log("Menu closed");

        const titles = [
            "Quick Actions",
            "User Settings and Preferences",
            "Data Export Options",
            "Help & Support",
        ];

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "15px" }}>
                {titles.map((title, index) => (
                    <Box
                        key={index}
                        sx={{
                            width: "350px",
                            backgroundColor: "var(--background-paper)",
                            borderRadius: "8px",
                            overflow: "hidden",
                            border: "1px solid var(--divider)",
                        }}
                    >
                        <MenuTitle
                            title={title}
                            ariaLabel={`menu-${index}`}
                            onClose={handleClose}
                            variant="subheader"
                        />
                    </Box>
                ))}
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Multiple menu titles of varying lengths for design comparison.",
            },
        },
    },
};

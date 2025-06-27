import type { Meta, StoryObj } from "@storybook/react";
import React, { useState, useRef } from "react";
import { Box, Typography } from "@mui/material";
import { Button } from "../../buttons/Button.js";
import { PopoverWithArrow } from "./PopoverWithArrow.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

const meta: Meta<typeof PopoverWithArrow> = {
    title: "Components/Dialogs/PopoverWithArrow",
    component: PopoverWithArrow,
    parameters: {
        docs: {
            description: {
                component: "A popover component with an arrow pointing to its anchor element. Features smart positioning, backdrop blur, and keyboard navigation. Automatically adjusts placement based on available space and supports custom content styling.",
            },
        },
        layout: "fullscreen",
        backgrounds: { disable: true },
    },
    tags: ["autodocs"],
    argTypes: {
        placement: {
            control: { type: "select" },
            options: ["auto", "top", "bottom", "left", "right"],
            description: "Preferred placement of the popover relative to anchor",
            table: {
                defaultValue: { summary: "auto" },
            },
        },
        anchorEl: {
            description: "The element to anchor the popover to",
            table: {
                type: { summary: "Element | null" },
            },
        },
        children: {
            control: "text",
            description: "Content to display in the popover",
            mapping: {
                "Simple text": "This is a simple popover with some text content.",
                "Rich content": (
                    <Box>
                        <Typography variant="h6" gutterBottom>Rich Content</Typography>
                        <Typography variant="body2" paragraph>
                            This popover contains multiple elements including headings, paragraphs, and other components.
                        </Typography>
                        <Box sx={{ mt: 1, p: 1, bgcolor: "action.hover", borderRadius: 1 }}>
                            <Typography variant="caption">
                                Even complex layouts work well!
                            </Typography>
                        </Box>
                    </Box>
                ),
            },
        },
        handleClose: {
            action: "handleClose",
            description: "Callback when popover should close",
        },
    },
    args: {
        placement: "auto",
        children: "Simple text",
    },
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Interactive popover with customizable placement and content.
 */
export const Showcase: Story = {
    render: (args) => {
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);
        const buttonRef = useRef<HTMLButtonElement>(null);

        const handleOpen = (event: React.MouseEvent) => {
            setAnchorEl(event.currentTarget);
        };

        const handleClose = () => {
            setAnchorEl(null);
            args.handleClose?.();
        };

        return (
            <Box sx={{ display: "flex", flexDirection: "column", gap: 4, alignItems: "center", pt: 8 }}>
                <Box sx={{ 
                    p: 2, 
                    bgcolor: "primary.light", 
                    color: "primary.contrastText",
                    borderRadius: 2,
                    textAlign: "center",
                    maxWidth: 500,
                }}>
                    <h3 style={{ margin: "0 0 8px 0" }}>🎯 Popover with Arrow</h3>
                    <p style={{ margin: "0 0 12px 0", fontSize: "0.875rem" }}>
                        Smart positioning popover that automatically adjusts placement based on available space. 
                        Features an arrow pointing to the anchor element.
                    </p>
                    <p style={{ margin: 0, fontSize: "0.75rem", opacity: 0.8 }}>
                        Click the button below to see it in action!
                    </p>
                </Box>

                <Button 
                    ref={buttonRef}
                    onClick={handleOpen} 
                    variant="primary" 
                    size="lg"
                >
                    🎯 Click to Open Popover
                </Button>

                <PopoverWithArrow
                    anchorEl={anchorEl}
                    placement={args.placement}
                    handleClose={handleClose}
                >
                    {args.children}
                </PopoverWithArrow>
            </Box>
        );
    },
    args: {
        placement: "auto",
        children: "Simple text",
    },
};

/**
 * Examples showing different placement options.
 */
export const Placements: Story = {
    render: () => {
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);
        const [placement, setPlacement] = useState<"auto" | "top" | "bottom" | "left" | "right">("auto");

        const placements: Array<{ key: "auto" | "top" | "bottom" | "left" | "right", label: string, description: string }> = [
            { key: "auto", label: "🧠 Auto", description: "Smart positioning based on available space" },
            { key: "top", label: "⬆️ Top", description: "Always position above the anchor" },
            { key: "bottom", label: "⬇️ Bottom", description: "Always position below the anchor" },
            { key: "left", label: "⬅️ Left", description: "Always position to the left of anchor" },
            { key: "right", label: "➡️ Right", description: "Always position to the right of anchor" },
        ];

        const handleOpen = (event: React.MouseEvent, newPlacement: typeof placement) => {
            setPlacement(newPlacement);
            setAnchorEl(event.currentTarget);
        };

        const handleClose = () => {
            setAnchorEl(null);
        };

        return (
            <Box sx={{ display: "flex", flexDirection: "column", gap: 4 }}>
                <Box>
                    <h3 style={{ margin: "0 0 8px 0" }}>Placement Options</h3>
                    <p style={{ margin: 0, fontSize: "0.875rem", opacity: 0.7 }}>
                        Click buttons to see how different placements work
                    </p>
                </Box>

                <Box sx={{ 
                    display: "grid", 
                    gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))", 
                    gap: 2,
                    mb: 4,
                }}>
                    {placements.map((p) => (
                        <Box key={p.key} sx={{ 
                            p: 2, 
                            border: "1px solid", 
                            borderColor: "divider",
                            borderRadius: 2,
                            bgcolor: "background.paper",
                            textAlign: "center",
                        }}>
                            <h5 style={{ margin: "0 0 8px 0" }}>{p.label}</h5>
                            <p style={{ margin: "0 0 12px 0", fontSize: "0.8rem", opacity: 0.7 }}>
                                {p.description}
                            </p>
                            <Button 
                                onClick={(e) => handleOpen(e, p.key)}
                                variant="outline"
                                size="sm"
                                fullWidth
                            >
                                Try {p.key}
                            </Button>
                        </Box>
                    ))}
                </Box>

                {/* Central positioning area */}
                <Box sx={{ 
                    height: "400px",
                    border: "2px dashed",
                    borderColor: "divider",
                    borderRadius: 2,
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    position: "relative",
                    bgcolor: "action.hover",
                }}>
                    <Box sx={{ textAlign: "center" }}>
                        <Typography variant="h6" gutterBottom>
                            Positioning Test Area
                        </Typography>
                        <Typography variant="body2" sx={{ mb: 2, opacity: 0.7 }}>
                            Current placement: <strong>{placement}</strong>
                        </Typography>
                        <Button 
                            onClick={(e) => handleOpen(e, placement)}
                            variant="primary"
                        >
                            🎯 Open Popover ({placement})
                        </Button>
                    </Box>
                </Box>

                <PopoverWithArrow
                    anchorEl={anchorEl}
                    placement={placement}
                    handleClose={handleClose}
                >
                    <Box>
                        <Typography variant="h6" gutterBottom>
                            Placement: {placement}
                        </Typography>
                        <Typography variant="body2">
                            This popover is positioned using the "{placement}" placement option.
                            {placement === "auto" && " It automatically chooses the best position based on available space."}
                        </Typography>
                    </Box>
                </PopoverWithArrow>
            </Box>
        );
    },
};

/**
 * Different content types and styling examples.
 */
export const ContentVariations: Story = {
    render: () => {
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);
        const [contentType, setContentType] = useState<string>("simple");

        const contentTypes = [
            {
                key: "simple",
                label: "📝 Simple Text",
                content: "This is a simple popover with plain text content.",
            },
            {
                key: "rich",
                label: "🎨 Rich Content",
                content: (
                    <Box>
                        <Typography variant="h6" gutterBottom>
                            Rich Content Example
                        </Typography>
                        <Typography variant="body2" paragraph>
                            This popover contains multiple elements including headings, paragraphs, and styled sections.
                        </Typography>
                        <Box sx={{ mt: 1, p: 1.5, bgcolor: "primary.light", color: "primary.contrastText", borderRadius: 1 }}>
                            <Typography variant="caption">
                                ✨ Even complex layouts work well with proper styling!
                            </Typography>
                        </Box>
                    </Box>
                ),
            },
            {
                key: "list",
                label: "📋 List Content",
                content: (
                    <Box>
                        <Typography variant="subtitle2" gutterBottom>
                            Available Actions:
                        </Typography>
                        <Box component="ul" sx={{ m: 0, pl: 2 }}>
                            <li>Edit item</li>
                            <li>Delete item</li>
                            <li>Share item</li>
                            <li>Export data</li>
                        </Box>
                    </Box>
                ),
            },
            {
                key: "interactive",
                label: "🎮 Interactive",
                content: (
                    <Box>
                        <Typography variant="subtitle2" gutterBottom>
                            Interactive Content
                        </Typography>
                        <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                            <Button variant="outline" size="sm" fullWidth>
                                Primary Action
                            </Button>
                            <Button variant="ghost" size="sm" fullWidth>
                                Secondary Action
                            </Button>
                        </Box>
                    </Box>
                ),
            },
            {
                key: "long",
                label: "📄 Long Content",
                content: (
                    <Box sx={{ maxWidth: 300 }}>
                        <Typography variant="h6" gutterBottom>
                            Scrollable Content
                        </Typography>
                        {Array.from({ length: 8 }, (_, i) => (
                            <Typography key={i} variant="body2" paragraph>
                                Paragraph {i + 1}: Lorem ipsum dolor sit amet, consectetur adipiscing elit. 
                                Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
                            </Typography>
                        ))}
                    </Box>
                ),
            },
        ];

        const handleOpen = (event: React.MouseEvent, type: string) => {
            setContentType(type);
            setAnchorEl(event.currentTarget);
        };

        const handleClose = () => {
            setAnchorEl(null);
        };

        return (
            <Box sx={{ display: "flex", flexDirection: "column", gap: 4 }}>
                <Box>
                    <h3 style={{ margin: "0 0 8px 0" }}>Content Variations</h3>
                    <p style={{ margin: 0, fontSize: "0.875rem", opacity: 0.7 }}>
                        Different types of content that work well in popovers
                    </p>
                </Box>

                <Box sx={{ 
                    display: "grid", 
                    gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))", 
                    gap: 2,
                }}>
                    {contentTypes.map((type) => (
                        <Box key={type.key} sx={{ 
                            p: 2, 
                            border: "1px solid", 
                            borderColor: "divider",
                            borderRadius: 2,
                            bgcolor: "background.paper",
                            textAlign: "center",
                        }}>
                            <h5 style={{ margin: "0 0 12px 0" }}>{type.label}</h5>
                            <Button 
                                onClick={(e) => handleOpen(e, type.key)}
                                variant="outline"
                                size="sm"
                                fullWidth
                            >
                                Show Example
                            </Button>
                        </Box>
                    ))}
                </Box>

                <PopoverWithArrow
                    anchorEl={anchorEl}
                    placement="auto"
                    handleClose={handleClose}
                >
                    {contentTypes.find(t => t.key === contentType)?.content}
                </PopoverWithArrow>
            </Box>
        );
    },
};

/**
 * Interactive positioning test with moveable anchor points.
 */
export const PositioningTest: Story = {
    render: () => {
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);
        const [placement, setPlacement] = useState<"auto" | "top" | "bottom" | "left" | "right">("auto");

        const handleOpen = (event: React.MouseEvent) => {
            setAnchorEl(event.currentTarget);
        };

        const handleClose = () => {
            setAnchorEl(null);
        };

        return (
            <Box 
                sx={{ 
                    position: "relative",
                    height: "600px",
                    border: "2px dashed",
                    borderColor: "divider",
                    borderRadius: 2,
                    bgcolor: "action.hover",
                    overflow: "hidden",
                }}
            >
                {/* Controls */}
                <Box sx={{ 
                    position: "absolute",
                    top: 16,
                    left: 16,
                    zIndex: 10,
                    p: 2,
                    bgcolor: "background.paper",
                    borderRadius: 2,
                    boxShadow: 2,
                }}>
                    <Typography variant="caption" sx={{ display: "block", mb: 1 }}>
                        Placement:
                    </Typography>
                    <Box sx={{ display: "flex", gap: 0.5, flexWrap: "wrap" }}>
                        {(["auto", "top", "bottom", "left", "right"] as const).map((p) => (
                            <button
                                key={p}
                                onClick={() => setPlacement(p)}
                                style={{
                                    padding: "2px 6px",
                                    fontSize: "0.7rem",
                                    borderRadius: "3px",
                                    border: "1px solid #ccc",
                                    background: placement === p ? "#007bff" : "white",
                                    color: placement === p ? "white" : "black",
                                    cursor: "pointer",
                                }}
                            >
                                {p}
                            </button>
                        ))}
                    </Box>
                </Box>

                {/* Corner buttons */}
                <Button 
                    onClick={handleOpen}
                    variant="primary"
                    size="sm"
                    sx={{ position: "absolute", top: 16, right: 16 }}
                >
                    Top Right
                </Button>
                
                <Button 
                    onClick={handleOpen}
                    variant="secondary"
                    size="sm"
                    sx={{ position: "absolute", bottom: 16, left: 16 }}
                >
                    Bottom Left
                </Button>
                
                <Button 
                    onClick={handleOpen}
                    variant="danger"
                    size="sm"
                    sx={{ position: "absolute", bottom: 16, right: 16 }}
                >
                    Bottom Right
                </Button>

                {/* Edge buttons */}
                <Button 
                    onClick={handleOpen}
                    variant="outline"
                    size="sm"
                    sx={{ position: "absolute", top: "50%", left: 16, transform: "translateY(-50%)" }}
                >
                    Left Edge
                </Button>
                
                <Button 
                    onClick={handleOpen}
                    variant="outline"
                    size="sm"
                    sx={{ position: "absolute", top: "50%", right: 16, transform: "translateY(-50%)" }}
                >
                    Right Edge
                </Button>
                
                <Button 
                    onClick={handleOpen}
                    variant="outline"
                    size="sm"
                    sx={{ position: "absolute", top: 16, left: "50%", transform: "translateX(-50%)" }}
                >
                    Top Edge
                </Button>
                
                <Button 
                    onClick={handleOpen}
                    variant="outline"
                    size="sm"
                    sx={{ position: "absolute", bottom: 16, left: "50%", transform: "translateX(-50%)" }}
                >
                    Bottom Edge
                </Button>

                {/* Center button */}
                <Box sx={{ 
                    position: "absolute",
                    top: "50%",
                    left: "50%",
                    transform: "translate(-50%, -50%)",
                    textAlign: "center",
                }}>
                    <Typography variant="h6" gutterBottom>
                        🎯 Positioning Test
                    </Typography>
                    <Typography variant="body2" sx={{ mb: 2, opacity: 0.7 }}>
                        Click any button to test popover positioning
                    </Typography>
                    <Button 
                        onClick={handleOpen}
                        variant="primary"
                        size="lg"
                    >
                        Center Target
                    </Button>
                </Box>

                <PopoverWithArrow
                    anchorEl={anchorEl}
                    placement={placement}
                    handleClose={handleClose}
                >
                    <Box>
                        <Typography variant="h6" gutterBottom>
                            Smart Positioning! 🎯
                        </Typography>
                        <Typography variant="body2" paragraph>
                            This popover automatically adjusts its position based on available space around the anchor element.
                        </Typography>
                        <Typography variant="body2">
                            <strong>Current placement:</strong> {placement}
                        </Typography>
                        <Typography variant="caption" sx={{ display: "block", mt: 1, opacity: 0.7 }}>
                            Try clicking buttons near the edges to see how it adapts!
                        </Typography>
                    </Box>
                </PopoverWithArrow>
            </Box>
        );
    },
};

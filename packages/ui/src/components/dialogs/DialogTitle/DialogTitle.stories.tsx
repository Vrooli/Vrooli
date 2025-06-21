import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { Box, Typography, FormControl, FormLabel, RadioGroup, FormControlLabel, Radio, TextField, Paper } from "@mui/material";
import { IconCommon } from "../../../icons/Icons.js";
import { Switch } from "../../inputs/Switch/Switch.js";
import { DialogTitle } from "./DialogTitle.js";

const meta: Meta<typeof DialogTitle> = {
    title: "Components/Dialogs/DialogTitle",
    component: DialogTitle,
    parameters: {
        layout: "fullscreen",
        docs: {
            description: {
                component: "A styled dialog title component with enhanced performance, reduced padding, line-clamped titles, and skeleton loading state. Features improved layout where action buttons stay in the same row, animations, responsive design, and support for custom children content for maximum flexibility.",
            },
        },
    },
    tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive DialogTitle Showcase with all features
export const Showcase: Story = {
    render: () => {
        const [titleText, setTitleText] = useState("Enhanced Dialog Title");
        const [helpText, setHelpText] = useState("This title showcases all the enhanced features and improvements");
        const [showStartComponent, setShowStartComponent] = useState(true);
        const [startComponentType, setStartComponentType] = useState<"Settings" | "User" | "Info" | "Warning">("Settings");
        const [showBelowContent, setShowBelowContent] = useState(false);
        const [belowContentType, setBelowContentType] = useState<"info" | "warning" | "success">("info");
        const [isLoading, setIsLoading] = useState(false);
        const [shadow, setShadow] = useState(true);
        const [animate, setAnimate] = useState(true);
        const [showCloseButton, setShowCloseButton] = useState(true);
        const [isClosed, setIsClosed] = useState(false);
        const [useCustomChildren, setUseCustomChildren] = useState(false);

        const handleClose = () => {
            setIsClosed(true);
        };

        const handleReset = () => {
            setIsClosed(false);
        };

        const getBelowContent = () => {
            if (!showBelowContent) return undefined;
            
            const contentMap = {
                info: {
                    bgcolor: "info.light",
                    color: "info.contrastText",
                    icon: "Info",
                    text: "This is informational content below the title"
                },
                warning: {
                    bgcolor: "warning.light", 
                    color: "warning.contrastText",
                    icon: "Warning",
                    text: "Warning: This action may have important consequences"
                },
                success: {
                    bgcolor: "success.light",
                    color: "success.contrastText", 
                    icon: "CheckCircle",
                    text: "Success: Operation completed successfully"
                }
            };

            const config = contentMap[belowContentType];

            return (
                <Box sx={{ 
                    p: 2, 
                    bgcolor: config.bgcolor,
                    color: config.color,
                    fontSize: "0.875rem",
                    display: "flex",
                    alignItems: "center",
                    gap: 1,
                }}>
                    <IconCommon name={config.icon as any} size={16} />
                    {config.text}
                </Box>
            );
        };

        return (
            <Box sx={{ 
                p: 2, 
                height: "100vh", 
                overflow: "auto",
                bgcolor: "background.default", 
            }}>
                <Box sx={{ 
                    display: "flex", 
                    gap: 2, 
                    flexDirection: "column",
                    maxWidth: 1400, 
                    mx: "auto", 
                }}>
                    {/* Controls Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        height: "fit-content",
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>DialogTitle Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)", lg: "repeat(4, 1fr)" },
                            gap: 3, 
                        }}>
                            {/* Title Text Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Title Text</FormLabel>
                                <TextField
                                    value={titleText}
                                    onChange={(e) => setTitleText(e.target.value)}
                                    size="small"
                                    sx={{ width: "100%" }}
                                    placeholder="Enter title text"
                                />
                            </FormControl>

                            {/* Help Text Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Help Text</FormLabel>
                                <TextField
                                    value={helpText}
                                    onChange={(e) => setHelpText(e.target.value)}
                                    size="small"
                                    sx={{ width: "100%" }}
                                    placeholder="Enter help text"
                                />
                            </FormControl>

                            {/* Start Component Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Start Component</FormLabel>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                                    <Switch
                                        checked={showStartComponent}
                                        onChange={(checked) => setShowStartComponent(checked)}
                                        size="sm"
                                        label="Show Icon"
                                        labelPosition="right"
                                    />
                                    {showStartComponent && (
                                        <RadioGroup
                                            value={startComponentType}
                                            onChange={(e) => setStartComponentType(e.target.value as any)}
                                            sx={{ gap: 0.5 }}
                                        >
                                            <FormControlLabel value="Settings" control={<Radio size="small" />} label="Settings" sx={{ m: 0 }} />
                                            <FormControlLabel value="User" control={<Radio size="small" />} label="User" sx={{ m: 0 }} />
                                            <FormControlLabel value="Info" control={<Radio size="small" />} label="Info" sx={{ m: 0 }} />
                                            <FormControlLabel value="Warning" control={<Radio size="small" />} label="Warning" sx={{ m: 0 }} />
                                        </RadioGroup>
                                    )}
                                </Box>
                            </FormControl>

                            {/* Below Content Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Below Content</FormLabel>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                                    <Switch
                                        checked={showBelowContent}
                                        onChange={(checked) => setShowBelowContent(checked)}
                                        size="sm"
                                        label="Show Content"
                                        labelPosition="right"
                                    />
                                    {showBelowContent && (
                                        <RadioGroup
                                            value={belowContentType}
                                            onChange={(e) => setBelowContentType(e.target.value as any)}
                                            sx={{ gap: 0.5 }}
                                        >
                                            <FormControlLabel value="info" control={<Radio size="small" />} label="Info" sx={{ m: 0 }} />
                                            <FormControlLabel value="warning" control={<Radio size="small" />} label="Warning" sx={{ m: 0 }} />
                                            <FormControlLabel value="success" control={<Radio size="small" />} label="Success" sx={{ m: 0 }} />
                                        </RadioGroup>
                                    )}
                                </Box>
                            </FormControl>

                            {/* Loading State Control */}
                            <FormControl component="fieldset" size="small">
                                <Switch
                                    checked={isLoading}
                                    onChange={(checked) => setIsLoading(checked)}
                                    size="sm"
                                    label="Loading State"
                                    labelPosition="right"
                                />
                            </FormControl>

                            {/* Visual Enhancement Controls */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Visual Effects</FormLabel>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                                    <Switch
                                        checked={shadow}
                                        onChange={(checked) => setShadow(checked)}
                                        size="sm"
                                        label="Shadow"
                                        labelPosition="right"
                                    />
                                    <Switch
                                        checked={animate}
                                        onChange={(checked) => setAnimate(checked)}
                                        size="sm"
                                        label="Animations"
                                        labelPosition="right"
                                    />
                                </Box>
                            </FormControl>

                            {/* Close Button Control */}
                            <FormControl component="fieldset" size="small">
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                                    <Switch
                                        checked={showCloseButton}
                                        onChange={(checked) => setShowCloseButton(checked)}
                                        size="sm"
                                        label="Show Close Button"
                                        labelPosition="right"
                                    />
                                    {isClosed && (
                                        <button 
                                            onClick={handleReset}
                                            style={{ 
                                                padding: "6px 12px", 
                                                borderRadius: "4px", 
                                                border: "1px solid #ccc",
                                                background: "#007bff",
                                                color: "white",
                                                cursor: "pointer",
                                                fontSize: "0.875rem",
                                            }}
                                        >
                                            Reset
                                        </button>
                                    )}
                                </Box>
                            </FormControl>

                            {/* Custom Children Control */}
                            <FormControl component="fieldset" size="small">
                                <Switch
                                    checked={useCustomChildren}
                                    onChange={(checked) => setUseCustomChildren(checked)}
                                    size="sm"
                                    label="Use Custom Children"
                                    labelPosition="right"
                                />
                            </FormControl>
                        </Box>
                    </Box>

                    {/* DialogTitle Preview */}
                    <Box sx={{ 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        overflow: "hidden",
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ p: 3, pb: 2 }}>Preview</Typography>
                        
                        {!isClosed ? (
                            <Paper sx={{ overflow: "hidden", borderRadius: 0, boxShadow: 0 }}>
                                {useCustomChildren ? (
                                    <DialogTitle
                                        id="dialog-title-showcase"
                                        onClose={showCloseButton ? handleClose : undefined}
                                        isLoading={isLoading}
                                        shadow={shadow}
                                        animate={animate}
                                        below={getBelowContent()}
                                    >
                                        <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
                                            <IconCommon name="Success" fill="currentColor" size={24} />
                                            <Box>
                                                <Typography variant="h5" component="h2" sx={{ fontWeight: 600, mb: 0.5 }}>
                                                    {titleText}
                                                </Typography>
                                                <Typography variant="body2" sx={{ opacity: 0.7 }}>
                                                    Custom layout with subtitle
                                                </Typography>
                                            </Box>
                                        </Box>
                                    </DialogTitle>
                                ) : (
                                    <DialogTitle
                                        id="dialog-title-showcase"
                                        title={titleText}
                                        help={helpText || undefined}
                                        startComponent={showStartComponent ? <IconCommon name={startComponentType} fill="currentColor" /> : undefined}
                                        below={getBelowContent()}
                                        onClose={showCloseButton ? handleClose : undefined}
                                        isLoading={isLoading}
                                        shadow={shadow}
                                        animate={animate}
                                    />
                                )}
                            </Paper>
                        ) : (
                            <Box sx={{ 
                                p: 4, 
                                textAlign: "center", 
                                bgcolor: "action.hover", 
                                border: "2px dashed",
                                borderColor: "divider",
                                mx: 3,
                                mb: 3,
                                borderRadius: 2,
                            }}>
                                <Typography variant="body2" sx={{ opacity: 0.7 }}>
                                    Dialog was closed. Click "Reset" above to show it again.
                                </Typography>
                            </Box>
                        )}
                    </Box>

                    {/* Feature Showcase */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Feature Examples</Typography>
                        
                        <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                            {/* Basic Example */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 1 }}>Basic</Typography>
                                <Paper sx={{ overflow: "hidden", borderRadius: 2 }}>
                                    <DialogTitle
                                        id="basic-example"
                                        title="Simple Dialog Title"
                                        onClose={() => {}}
                                    />
                                </Paper>
                            </Box>

                            {/* Loading Example */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 1 }}>Loading State</Typography>
                                <Paper sx={{ overflow: "hidden", borderRadius: 2 }}>
                                    <DialogTitle
                                        id="loading-example"
                                        title="Loading Dialog"
                                        startComponent={<IconCommon name="Settings" fill="currentColor" />}
                                        below={<Box sx={{ p: 2, bgcolor: "action.hover" }}>Additional content</Box>}
                                        isLoading={true}
                                        onClose={() => {}}
                                    />
                                </Paper>
                            </Box>

                            {/* Enhanced Example */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 1 }}>Enhanced with Animations</Typography>
                                <Paper sx={{ overflow: "hidden", borderRadius: 2 }}>
                                    <DialogTitle
                                        id="enhanced-example"
                                        title="Enhanced Dialog Title"
                                        help="With smooth animations and improved layout"
                                        startComponent={<IconCommon name="Settings" fill="currentColor" />}
                                        animate={true}
                                        shadow={true}
                                        onClose={() => {}}
                                    />
                                </Paper>
                            </Box>

                            {/* Full Featured Example */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 1 }}>Full Featured</Typography>
                                <Paper sx={{ overflow: "hidden", borderRadius: 2 }}>
                                    <DialogTitle
                                        id="full-example"
                                        title="Advanced Configuration"
                                        help="Complete example with all features enabled"
                                        startComponent={<IconCommon name="Warning" fill="currentColor" />}
                                        below={
                                            <Box sx={{ 
                                                p: 2, 
                                                bgcolor: "warning.light", 
                                                color: "warning.contrastText",
                                                fontSize: "0.875rem",
                                                display: "flex",
                                                alignItems: "center",
                                                gap: 1,
                                            }}>
                                                <IconCommon name="Warning" size={16} />
                                                Changes may affect system performance
                                            </Box>
                                        }
                                        animate={true}
                                        shadow={true}
                                        onClose={() => {}}
                                    />
                                </Paper>
                            </Box>

                            {/* Custom Children Example */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 1 }}>Custom Children Content</Typography>
                                <Paper sx={{ overflow: "hidden", borderRadius: 2 }}>
                                    <DialogTitle
                                        id="children-example"
                                        onClose={() => {}}
                                        shadow={true}
                                    >
                                        <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
                                            <IconCommon name="Success" fill="currentColor" size={24} />
                                            <Box>
                                                <Typography variant="h5" component="h2" sx={{ fontWeight: 600, mb: 0.5 }}>
                                                    Custom Title Layout
                                                </Typography>
                                                <Typography variant="body2" sx={{ opacity: 0.7 }}>
                                                    With subtitle and custom styling
                                                </Typography>
                                            </Box>
                                        </Box>
                                    </DialogTitle>
                                </Paper>
                            </Box>

                            {/* Complex Custom Children Example */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 1 }}>Complex Custom Layout</Typography>
                                <Paper sx={{ overflow: "hidden", borderRadius: 2 }}>
                                    <DialogTitle
                                        id="complex-children-example"
                                        onClose={() => {}}
                                        animate={true}
                                    >
                                        <Box sx={{ textAlign: "center", width: "100%" }}>
                                            <Typography variant="h5" sx={{ fontWeight: "bold", mb: 1 }}>
                                                Power Up Your AI Assistant 🚀
                                            </Typography>
                                            <Typography variant="body2" sx={{ opacity: 0.8 }}>
                                                Get credits to run AI-powered tasks and automations
                                            </Typography>
                                        </Box>
                                    </DialogTitle>
                                </Paper>
                            </Box>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};
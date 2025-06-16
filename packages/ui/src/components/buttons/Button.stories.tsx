import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import FormControl from "@mui/material/FormControl";
import FormLabel from "@mui/material/FormLabel";
import RadioGroup from "@mui/material/RadioGroup";
import FormControlLabel from "@mui/material/FormControlLabel";
import Radio from "@mui/material/Radio";
import TextField from "@mui/material/TextField";
import { IconCommon } from "../../icons/Icons.js";
import { Button } from "./Button.js";
import type { ButtonVariant, ButtonSize, ButtonBorderRadius } from "./Button.js";
import { getCustomButtonStyle } from "./buttonStyles.js";
import { Switch } from "../inputs/Switch/Switch.js";

const meta: Meta<typeof Button> = {
    title: "Components/Buttons/Button",
    component: Button,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive Button Playground with all variants
export const ButtonShowcase: Story = {
    render: () => {
        const [size, setSize] = useState<ButtonSize>("md");
        const [borderRadius, setBorderRadius] = useState<ButtonBorderRadius>("minimal");
        const [loadingState, setLoadingState] = useState<"none" | "circular" | "orbital">("none");
        const [iconPosition, setIconPosition] = useState<"none" | "start" | "end">("none");
        const [disabled, setDisabled] = useState(false);
        const [customColor, setCustomColor] = useState("#9333EA");

        const variants: ButtonVariant[] = ["primary", "secondary", "outline", "ghost", "danger", "space", "custom", "neon"];
        
        // Determine loading props
        const isLoading = loadingState !== "none";
        const loadingIndicator = loadingState === "orbital" ? "orbital" : "circular";
        
        // Determine icon props - let icons inherit text color naturally
        const icon = <IconCommon name="Save" />;
        const startIcon = iconPosition === "start" ? icon : undefined;
        const endIcon = iconPosition === "end" ? icon : undefined;
        

        return (
            <Box sx={{ 
                p: 2, 
                height: "100vh", 
                overflow: "auto",
                bgcolor: "background.default" 
            }}>
                <Box sx={{ 
                    display: "flex", 
                    gap: 2, 
                    flexDirection: "column",
                    maxWidth: 1400, 
                    mx: "auto" 
                }}>
                    {/* Controls Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        height: "fit-content",
                        width: "100%"
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Button Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)", lg: "repeat(4, 1fr)" },
                            gap: 3 
                        }}>
                            {/* Size Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Size</FormLabel>
                                <RadioGroup
                                    value={size}
                                    onChange={(e) => setSize(e.target.value as ButtonSize)}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="sm" control={<Radio size="small" />} label="Small" sx={{ m: 0 }} />
                                    <FormControlLabel value="md" control={<Radio size="small" />} label="Medium" sx={{ m: 0 }} />
                                    <FormControlLabel value="lg" control={<Radio size="small" />} label="Large" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            {/* Border Radius Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Border Radius</FormLabel>
                                <RadioGroup
                                    value={borderRadius}
                                    onChange={(e) => setBorderRadius(e.target.value as ButtonBorderRadius)}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="none" control={<Radio size="small" />} label="None" sx={{ m: 0 }} />
                                    <FormControlLabel value="minimal" control={<Radio size="small" />} label="Minimal (Default)" sx={{ m: 0 }} />
                                    <FormControlLabel value="pill" control={<Radio size="small" />} label="Pill" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            {/* Loading State Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Loading State</FormLabel>
                                <RadioGroup
                                    value={loadingState}
                                    onChange={(e) => setLoadingState(e.target.value as "none" | "circular" | "orbital")}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="none" control={<Radio size="small" />} label="Not Loading" sx={{ m: 0 }} />
                                    <FormControlLabel value="circular" control={<Radio size="small" />} label="Circular" sx={{ m: 0 }} />
                                    <FormControlLabel value="orbital" control={<Radio size="small" />} label="Orbital" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            {/* Icon Position Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Icon Position</FormLabel>
                                <RadioGroup
                                    value={iconPosition}
                                    onChange={(e) => setIconPosition(e.target.value as "none" | "start" | "end")}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="none" control={<Radio size="small" />} label="No Icon" sx={{ m: 0 }} />
                                    <FormControlLabel value="start" control={<Radio size="small" />} label="Start" sx={{ m: 0 }} />
                                    <FormControlLabel value="end" control={<Radio size="small" />} label="End" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            {/* Disabled Control */}
                            <FormControl component="fieldset" size="small">
                                <Switch
                                    checked={disabled}
                                    onChange={(checked) => setDisabled(checked)}
                                    size="sm"
                                    label="Disabled"
                                    labelPosition="right"
                                />
                            </FormControl>
                            
                            {/* Custom Color Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Custom Color</FormLabel>
                                <TextField
                                    type="color"
                                    value={customColor}
                                    onChange={(e) => setCustomColor(e.target.value)}
                                    size="small"
                                    sx={{ width: '100%' }}
                                />
                            </FormControl>
                        </Box>
                    </Box>

                    {/* Buttons Display */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%"
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>All Button Variants</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { 
                                xs: "1fr", 
                                sm: "repeat(2, 1fr)", 
                                md: "repeat(3, 1fr)",
                                lg: "repeat(4, 1fr)"
                            }, 
                            gap: 3 
                        }}>
                            {variants.map(variant => (
                                <Box 
                                    key={variant} 
                                    sx={{ 
                                        textAlign: "center",
                                        p: 2,
                                        borderRadius: 2,
                                        backgroundColor: (variant === "space" || variant === "neon") ? "#001122" : "transparent"
                                    }}
                                >
                                    <Typography 
                                        variant="subtitle2" 
                                        sx={{ 
                                            mb: 1, 
                                            textTransform: "capitalize",
                                            color: (variant === "space" || variant === "neon") ? "#fff" : "inherit"
                                        }}
                                    >
                                        {variant}
                                    </Typography>
                                    <Button
                                        variant={variant}
                                        size={size}
                                        borderRadius={borderRadius}
                                        isLoading={isLoading}
                                        loadingIndicator={loadingIndicator}
                                        startIcon={startIcon}
                                        endIcon={endIcon}
                                        disabled={disabled}
                                        style={{
                                            ...(variant === "custom" ? getCustomButtonStyle(customColor) : {})
                                        }}
                                        className={variant === "custom" ? "hover:tw-opacity-90" : undefined}
                                    >
                                        Button Text
                                    </Button>
                                </Box>
                            ))}
                        </Box>
                    </Box>

                    {/* Border Radius Showcase */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%"
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Border Radius Options</Typography>
                        
                        <Box sx={{ display: "flex", flexWrap: "wrap", gap: 4, justifyContent: "center" }}>
                            {/* None Example */}
                            <Box sx={{ textAlign: "center" }}>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>None</Typography>
                                <Button variant="primary" borderRadius="none">
                                    Sharp Corners
                                </Button>
                            </Box>

                            {/* Minimal Example */}
                            <Box sx={{ textAlign: "center" }}>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>Minimal (Default)</Typography>
                                <Button variant="primary" borderRadius="minimal">
                                    Subtle Rounds
                                </Button>
                            </Box>

                            {/* Pill Example */}
                            <Box sx={{ textAlign: "center" }}>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>Pill</Typography>
                                <Button variant="primary" borderRadius="pill">
                                    Fully Rounded
                                </Button>
                            </Box>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};


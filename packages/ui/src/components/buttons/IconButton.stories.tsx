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
import { IconButton } from "./IconButton.js";
import type { IconButtonVariant, IconButtonSize } from "./iconButtonStyles.js";
import { getCustomIconButtonStyle, getContrastTextColor } from "./iconButtonStyles.js";
import { Switch } from "../inputs/Switch/Switch.js";

const meta: Meta<typeof IconButton> = {
    title: "Components/Buttons/IconButton",
    component: IconButton,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive Icon Button Playground with all variants
export const IconButtonShowcase: Story = {
    render: () => {
        const [variant, setVariant] = useState<IconButtonVariant>("solid");
        const [sizeType, setSizeType] = useState<"preset" | "custom">("preset");
        const [presetSize, setPresetSize] = useState<"sm" | "md" | "lg">("md");
        const [customSize, setCustomSize] = useState(48);
        const [iconName, setIconName] = useState("Save");
        const [disabled, setDisabled] = useState(false);
        const [customColor, setCustomColor] = useState("#9333EA");
        const [borderRadius, setBorderRadius] = useState("50");

        const variants: IconButtonVariant[] = ["solid", "transparent", "space", "custom", "neon"];
        const iconOptions = [
            "Save", "Edit", "Delete", "Add", "Remove", 
            "Settings", "Search", "Close", "Info", "Home",
            "Download", "Upload", "Copy", "Refresh", "Play",
            "Pause", "Filter", "Menu", "User", "Team",
        ];
        
        // Determine final size based on size type
        const finalSize: IconButtonSize = sizeType === "preset" ? presetSize : customSize;

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
                        <Typography variant="h5" sx={{ mb: 3 }}>Icon Button Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)", lg: "repeat(5, 1fr)" },
                            gap: 3, 
                        }}>
                            {/* Variant Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Variant</FormLabel>
                                <RadioGroup
                                    value={variant}
                                    onChange={(e) => setVariant(e.target.value as IconButtonVariant)}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="solid" control={<Radio size="small" />} label="Solid (3D)" sx={{ m: 0 }} />
                                    <FormControlLabel value="transparent" control={<Radio size="small" />} label="Transparent" sx={{ m: 0 }} />
                                    <FormControlLabel value="space" control={<Radio size="small" />} label="Space" sx={{ m: 0 }} />
                                    <FormControlLabel value="neon" control={<Radio size="small" />} label="Neon" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            {/* Size Type Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Size Type</FormLabel>
                                <RadioGroup
                                    value={sizeType}
                                    onChange={(e) => setSizeType(e.target.value as "preset" | "custom")}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="preset" control={<Radio size="small" />} label="Preset" sx={{ m: 0 }} />
                                    <FormControlLabel value="custom" control={<Radio size="small" />} label="Custom" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            {/* Size Control */}
                            {sizeType === "preset" ? (
                                <FormControl component="fieldset" size="small">
                                    <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Preset Size</FormLabel>
                                    <RadioGroup
                                        value={presetSize}
                                        onChange={(e) => setPresetSize(e.target.value as "sm" | "md" | "lg")}
                                        sx={{ gap: 0.5 }}
                                    >
                                        <FormControlLabel value="sm" control={<Radio size="small" />} label="Small (32px)" sx={{ m: 0 }} />
                                        <FormControlLabel value="md" control={<Radio size="small" />} label="Medium (48px)" sx={{ m: 0 }} />
                                        <FormControlLabel value="lg" control={<Radio size="small" />} label="Large (64px)" sx={{ m: 0 }} />
                                    </RadioGroup>
                                </FormControl>
                            ) : (
                                <FormControl component="fieldset" size="small">
                                    <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Custom Size (px)</FormLabel>
                                    <TextField
                                        type="number"
                                        value={customSize}
                                        onChange={(e) => setCustomSize(Number(e.target.value))}
                                        size="small"
                                        inputProps={{ min: 16, max: 128, step: 4 }}
                                        sx={{ width: "100%" }}
                                    />
                                </FormControl>
                            )}

                            {/* Icon Selection */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Icon</FormLabel>
                                <TextField
                                    select
                                    SelectProps={{ native: true }}
                                    value={iconName}
                                    onChange={(e) => setIconName(e.target.value)}
                                    size="small"
                                    sx={{ width: "100%" }}
                                >
                                    {iconOptions.map(icon => (
                                        <option key={icon} value={icon}>{icon}</option>
                                    ))}
                                </TextField>
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
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>
                            
                            {/* Border Radius Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Border Radius (%)</FormLabel>
                                <TextField
                                    type="number"
                                    value={borderRadius}
                                    onChange={(e) => setBorderRadius(e.target.value)}
                                    size="small"
                                    inputProps={{ min: 0, max: 50, step: 5 }}
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>
                        </Box>
                    </Box>

                    {/* Icon Buttons Display */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>All Icon Button Variants</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { 
                                xs: "repeat(1, 1fr)", 
                                sm: "repeat(2, 1fr)",
                                md: "repeat(3, 1fr)",
                                lg: "repeat(5, 1fr)",
                            }, 
                            gap: 3, 
                        }}>
                            {variants.map(v => (
                                <Box 
                                    key={v} 
                                    sx={{ 
                                        p: 3,
                                        borderRadius: 2,
                                        backgroundColor: (v === "space" || v === "neon") ? "#001122" : "transparent",
                                        textAlign: "center",
                                    }}
                                >
                                    <Typography 
                                        variant="subtitle2" 
                                        sx={{ 
                                            mb: 2, 
                                            textTransform: "capitalize",
                                            color: (v === "space" || v === "neon") ? "#fff" : "inherit",
                                        }}
                                    >
                                        {v === "solid" ? "Solid (3D)" : v === "custom" ? "Custom" : v === "neon" ? "Neon" : v}
                                    </Typography>
                                    <IconButton
                                        variant={v}
                                        size={finalSize}
                                        disabled={disabled}
                                        aria-label={`${iconName} button`}
                                        style={{
                                            ...(v === "custom" ? getCustomIconButtonStyle(customColor) : {}),
                                        }}
                                        className={v === "custom" ? "hover:tw-opacity-90" : undefined}
                                    >
                                        <IconCommon name={iconName} />
                                    </IconButton>
                                    <Typography variant="caption" display="block" sx={{ mt: 1, color: (v === "space" || v === "neon") ? "#aaa" : "text.secondary" }}>
                                        {sizeType === "custom" ? `${customSize}px × ${customSize}px` : `Size: ${presetSize} (${finalSize})`}
                                    </Typography>
                                </Box>
                            ))}
                        </Box>
                    </Box>

                    {/* Common Use Cases */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Common Use Cases</Typography>
                        
                        <Box sx={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
                            {/* Toolbar Example */}
                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>Toolbar Actions</Typography>
                                <Box sx={{ display: "flex", gap: 1, p: 2, bgcolor: "grey.100", borderRadius: 1 }}>
                                    <IconButton variant="transparent" size="sm" aria-label="Save">
                                        <IconCommon name="Save" />
                                    </IconButton>
                                    <IconButton variant="transparent" size="sm" aria-label="Edit">
                                        <IconCommon name="Edit" />
                                    </IconButton>
                                    <IconButton variant="transparent" size="sm" aria-label="Delete">
                                        <IconCommon name="Delete" />
                                    </IconButton>
                                    <Box sx={{ mx: 1, borderLeft: 1, borderColor: "divider" }} />
                                    <IconButton variant="transparent" size="sm" aria-label="Settings">
                                        <IconCommon name="Settings" />
                                    </IconButton>
                                </Box>
                            </Box>

                            {/* Floating Action Example */}
                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>Floating Action</Typography>
                                <Box sx={{ p: 2, bgcolor: "grey.100", borderRadius: 1 }}>
                                    <IconButton variant="solid" size="lg" aria-label="Add new item">
                                        <IconCommon name="Add" />
                                    </IconButton>
                                </Box>
                            </Box>

                            {/* Navigation Example */}
                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>Navigation</Typography>
                                <Box sx={{ display: "flex", gap: 2, p: 2, bgcolor: "grey.100", borderRadius: 1 }}>
                                    <IconButton variant="transparent" size="md" aria-label="Previous">
                                        <IconCommon name="ChevronLeft" />
                                    </IconButton>
                                    <IconButton variant="transparent" size="md" aria-label="Next">
                                        <IconCommon name="ChevronRight" />
                                    </IconButton>
                                </Box>
                            </Box>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};

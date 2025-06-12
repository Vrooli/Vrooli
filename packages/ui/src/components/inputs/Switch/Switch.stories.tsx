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
import { Switch } from "./Switch.js";
import type { SwitchVariant, SwitchSize, LabelPosition } from "./Switch.js";

const meta: Meta<typeof Switch> = {
    title: "Components/Inputs/Switch",
    component: Switch,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive Switch Playground with all variants
export const SwitchShowcase: Story = {
    render: () => {
        const [size, setSize] = useState<SwitchSize>("md");
        const [labelPosition, setLabelPosition] = useState<LabelPosition>("right");
        const [disabled, setDisabled] = useState(false);
        const [customColor, setCustomColor] = useState("#9333EA");
        const [labelText, setLabelText] = useState("Toggle feature");

        const variants: SwitchVariant[] = ["default", "success", "warning", "danger", "space", "neon", "theme", "custom"];
        const sizes: SwitchSize[] = ["sm", "md", "lg"];
        const labelPositions: LabelPosition[] = ["left", "right", "none"];

        // State for individual switch controls - all switches default to ON
        const [switchStates, setSwitchStates] = useState<Record<string, boolean>>({
            // Live preview states
            'preview-default': true,
            'preview-success': true,
            'preview-warning': true,
            'preview-danger': true,
            'preview-space': true,
            'preview-neon': true,
            'preview-theme': true,
            'preview-custom': true,
            // All variants section
            default: true,
            success: true,
            warning: true,
            danger: true,
            space: true,
            neon: true,
            theme: true,
            custom: true,
            // Size comparison
            'size-sm': true,
            'size-md': true,
            'size-lg': true,
            // Label positioning
            'label-left': true,
            'label-right': true,
            'label-none': true,
            // Common use cases
            notifications: true,
            darkMode: true,
            autoSave: true,
            betaFeatures: true,
            experimentalUI: true,
            debugMode: true,
            compact1: true,
            compact2: true,
            compact3: true,
            spaceMode: true,
            neonEffects: true,
        });
        
        const handleSwitchChange = (key: string, checked: boolean) => {
            setSwitchStates(prev => ({ ...prev, [key]: checked }));
        };

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
                        <Typography variant="h5" sx={{ mb: 3 }}>Switch Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)", lg: "repeat(5, 1fr)" },
                            gap: 3 
                        }}>

                            {/* Size Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Size</FormLabel>
                                <RadioGroup
                                    value={size}
                                    onChange={(e) => setSize(e.target.value as SwitchSize)}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="sm" control={<Radio size="small" />} label="Small" sx={{ m: 0 }} />
                                    <FormControlLabel value="md" control={<Radio size="small" />} label="Medium" sx={{ m: 0 }} />
                                    <FormControlLabel value="lg" control={<Radio size="small" />} label="Large" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            {/* Label Position Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Label Position</FormLabel>
                                <RadioGroup
                                    value={labelPosition}
                                    onChange={(e) => setLabelPosition(e.target.value as LabelPosition)}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="left" control={<Radio size="small" />} label="Left" sx={{ m: 0 }} />
                                    <FormControlLabel value="right" control={<Radio size="small" />} label="Right" sx={{ m: 0 }} />
                                    <FormControlLabel value="none" control={<Radio size="small" />} label="None" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            {/* Label Text Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Label Text</FormLabel>
                                <TextField
                                    value={labelText}
                                    onChange={(e) => setLabelText(e.target.value)}
                                    size="small"
                                    placeholder="Enter label text..."
                                    sx={{ width: '100%' }}
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

                            {/* Disabled Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>State</FormLabel>
                                <Switch
                                    checked={disabled}
                                    onChange={(checked) => setDisabled(checked)}
                                    size="sm"
                                    variant="default"
                                    label="Disabled"
                                />
                            </FormControl>
                        </Box>
                    </Box>

                    {/* Live Preview - All Variants */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%"
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Live Preview - All Variants</Typography>
                        
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
                            {variants.map(v => (
                                <Box 
                                    key={`preview-${v}`} 
                                    sx={{ 
                                        p: 3,
                                        borderRadius: 2,
                                        backgroundColor: (v === "space" || v === "neon") ? "#001122" : "grey.50",
                                        textAlign: "center"
                                    }}
                                >
                                    <Typography 
                                        variant="subtitle2" 
                                        sx={{ 
                                            mb: 2, 
                                            textTransform: "capitalize",
                                            color: (v === "space" || v === "neon") ? "#fff" : "inherit"
                                        }}
                                    >
                                        {v}
                                    </Typography>
                                    <Switch
                                        variant={v}
                                        size={size}
                                        labelPosition={labelPosition}
                                        label={labelPosition !== "none" ? labelText : undefined}
                                        color={v === "custom" ? customColor : undefined}
                                        disabled={disabled}
                                        checked={switchStates[`preview-${v}`] !== false}
                                        onChange={(checked) => handleSwitchChange(`preview-${v}`, checked)}
                                    />
                                </Box>
                            ))}
                        </Box>
                    </Box>

                    {/* All Variants Display */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%"
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>All Switch Variants</Typography>
                        
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
                            {variants.map(v => (
                                <Box 
                                    key={v} 
                                    sx={{ 
                                        p: 3,
                                        borderRadius: 2,
                                        backgroundColor: (v === "space" || v === "neon") ? "#001122" : "grey.50",
                                        textAlign: "center"
                                    }}
                                >
                                    <Typography 
                                        variant="subtitle2" 
                                        sx={{ 
                                            mb: 2, 
                                            textTransform: "capitalize",
                                            color: (v === "space" || v === "neon") ? "#fff" : "inherit"
                                        }}
                                    >
                                        {v}
                                    </Typography>
                                    <Switch
                                        variant={v}
                                        size={size}
                                        color={customColor}
                                        label={`${v} switch`}
                                        labelPosition="right"
                                        checked={switchStates[v] || false}
                                        onChange={(checked) => handleSwitchChange(v, checked)}
                                    />
                                </Box>
                            ))}
                        </Box>
                    </Box>

                    {/* Size Comparison */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%"
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Size Comparison</Typography>
                        
                        <Box sx={{ 
                            display: "flex", 
                            flexDirection: "column",
                            gap: 3,
                            alignItems: "center"
                        }}>
                            {sizes.map(s => (
                                <Box key={s} sx={{ textAlign: "center" }}>
                                    <Typography variant="subtitle2" sx={{ mb: 1 }}>
                                        Size: {s} ({s === "sm" ? "32x18px" : s === "md" ? "44x24px" : "56x30px"})
                                    </Typography>
                                    <Switch
                                        variant="default"
                                        size={s}
                                        label={`${s.toUpperCase()} size switch`}
                                        checked={switchStates[`size-${s}`] || false}
                                        onChange={(checked) => handleSwitchChange(`size-${s}`, checked)}
                                    />
                                </Box>
                            ))}
                        </Box>
                    </Box>

                    {/* Label Position Examples */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%"
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Label Positioning</Typography>
                        
                        <Box sx={{ 
                            display: "flex", 
                            flexDirection: "column",
                            gap: 3,
                            alignItems: "center"
                        }}>
                            {labelPositions.map(pos => (
                                <Box key={pos} sx={{ textAlign: "center" }}>
                                    <Typography variant="subtitle2" sx={{ mb: 1 }}>
                                        Label: {pos}
                                    </Typography>
                                    <Switch
                                        variant="success"
                                        size="md"
                                        labelPosition={pos}
                                        label={pos !== "none" ? `Label on ${pos}` : undefined}
                                        checked={switchStates[`label-${pos}`] || false}
                                        onChange={(checked) => handleSwitchChange(`label-${pos}`, checked)}
                                    />
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
                        width: "100%"
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Common Use Cases</Typography>
                        
                        <Box sx={{ display: "flex", flexDirection: "column", gap: 4 }}>
                            {/* Settings Panel */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Settings Panel</Typography>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 2, p: 2, bgcolor: "grey.50", borderRadius: 1 }}>
                                    <Switch
                                        variant="default"
                                        size="md"
                                        label="Enable notifications"
                                        checked={switchStates.notifications || false}
                                        onChange={(checked) => handleSwitchChange("notifications", checked)}
                                    />
                                    <Switch
                                        variant="default"
                                        size="md"
                                        label="Dark mode"
                                        checked={switchStates.darkMode || false}
                                        onChange={(checked) => handleSwitchChange("darkMode", checked)}
                                    />
                                    <Switch
                                        variant="default"
                                        size="md"
                                        label="Auto-save"
                                        checked={switchStates.autoSave || false}
                                        onChange={(checked) => handleSwitchChange("autoSave", checked)}
                                    />
                                </Box>
                            </Box>

                            {/* Feature Flags */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Feature Flags</Typography>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 2, p: 2, bgcolor: "grey.50", borderRadius: 1 }}>
                                    <Switch
                                        variant="success"
                                        size="sm"
                                        label="Beta features"
                                        checked={switchStates.betaFeatures || false}
                                        onChange={(checked) => handleSwitchChange("betaFeatures", checked)}
                                    />
                                    <Switch
                                        variant="warning"
                                        size="sm"
                                        label="Experimental UI"
                                        checked={switchStates.experimentalUI || false}
                                        onChange={(checked) => handleSwitchChange("experimentalUI", checked)}
                                    />
                                    <Switch
                                        variant="danger"
                                        size="sm"
                                        label="Debug mode"
                                        checked={switchStates.debugMode || false}
                                        onChange={(checked) => handleSwitchChange("debugMode", checked)}
                                    />
                                </Box>
                            </Box>

                            {/* Compact List */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Compact List</Typography>
                                <Box sx={{ display: "flex", flexWrap: "wrap", gap: 3, p: 2, bgcolor: "grey.50", borderRadius: 1 }}>
                                    <Switch
                                        variant="default"
                                        size="sm"
                                        labelPosition="none"
                                        aria-label="Toggle option 1"
                                        checked={switchStates.compact1 || false}
                                        onChange={(checked) => handleSwitchChange("compact1", checked)}
                                    />
                                    <Switch
                                        variant="default"
                                        size="sm"
                                        labelPosition="none"
                                        aria-label="Toggle option 2"
                                        checked={switchStates.compact2 || false}
                                        onChange={(checked) => handleSwitchChange("compact2", checked)}
                                    />
                                    <Switch
                                        variant="default"
                                        size="sm"
                                        labelPosition="none"
                                        aria-label="Toggle option 3"
                                        checked={switchStates.compact3 || false}
                                        onChange={(checked) => handleSwitchChange("compact3", checked)}
                                    />
                                </Box>
                            </Box>

                            {/* Themed Switches */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Themed Examples</Typography>
                                <Box sx={{ 
                                    display: "flex", 
                                    flexDirection: "column", 
                                    gap: 2, 
                                    p: 3, 
                                    bgcolor: "#001122", 
                                    borderRadius: 1 
                                }}>
                                    <Switch
                                        variant="space"
                                        size="lg"
                                        label="Activate space mode"
                                        checked={switchStates.spaceMode || false}
                                        onChange={(checked) => handleSwitchChange("spaceMode", checked)}
                                    />
                                    <Switch
                                        variant="neon"
                                        size="md"
                                        label="Enable neon effects"
                                        checked={switchStates.neonEffects || false}
                                        onChange={(checked) => handleSwitchChange("neonEffects", checked)}
                                    />
                                </Box>
                            </Box>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};
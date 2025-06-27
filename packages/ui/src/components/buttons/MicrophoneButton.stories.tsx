import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import FormControl from "@mui/material/FormControl";
import FormLabel from "@mui/material/FormLabel";
import RadioGroup from "@mui/material/RadioGroup";
import FormControlLabel from "@mui/material/FormControlLabel";
import Radio from "@mui/material/Radio";
import Switch from "@mui/material/Switch";
import TextField from "@mui/material/TextField";
import { MicrophoneButton } from "./MicrophoneButton.js";
import type { IconButtonVariant } from "./IconButton.js";

const meta: Meta<typeof MicrophoneButton> = {
    title: "Components/Buttons/MicrophoneButton",
    component: MicrophoneButton,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive Microphone Button Playground
export const MicrophoneButtonShowcase: Story = {
    render: () => {
        const [variant, setVariant] = useState<IconButtonVariant>("solid");
        const [size, setSize] = useState(48);
        const [disabled, setDisabled] = useState(false);
        const [showWhenUnavailable, setShowWhenUnavailable] = useState(true);
        const [customFill, setCustomFill] = useState("");
        const [useCustomFill, setUseCustomFill] = useState(false);

        const variants: IconButtonVariant[] = ["solid", "transparent", "space", "neon"];

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
                    maxWidth: 1200, 
                    mx: "auto", 
                }}>
                    {/* Controls Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Microphone Button Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)" },
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

                            {/* Size Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Size (px)</FormLabel>
                                <TextField
                                    type="number"
                                    value={size}
                                    onChange={(e) => setSize(Number(e.target.value))}
                                    size="small"
                                    inputProps={{ min: 24, max: 100, step: 4 }}
                                    sx={{ width: "100%" }}
                                />
                                <Typography variant="caption" sx={{ mt: 0.5, color: "text.secondary" }}>
                                    {size}px × {size}px
                                </Typography>
                            </FormControl>

                            {/* State Controls */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>State</FormLabel>
                                <FormControlLabel
                                    control={
                                        <Switch
                                            checked={disabled}
                                            onChange={(e) => setDisabled(e.target.checked)}
                                            size="small"
                                        />
                                    }
                                    label="Disabled"
                                    sx={{ m: 0 }}
                                />
                                <FormControlLabel
                                    control={
                                        <Switch
                                            checked={showWhenUnavailable}
                                            onChange={(e) => setShowWhenUnavailable(e.target.checked)}
                                            size="small"
                                        />
                                    }
                                    label="Show when unavailable"
                                    sx={{ m: 0, mt: 1 }}
                                />
                            </FormControl>

                            {/* Custom Fill Color */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Custom Fill Color</FormLabel>
                                <FormControlLabel
                                    control={
                                        <Switch
                                            checked={useCustomFill}
                                            onChange={(e) => setUseCustomFill(e.target.checked)}
                                            size="small"
                                        />
                                    }
                                    label="Use custom fill"
                                    sx={{ m: 0, mb: 1 }}
                                />
                                {useCustomFill && (
                                    <TextField
                                        type="color"
                                        value={customFill || "#9333EA"}
                                        onChange={(e) => setCustomFill(e.target.value)}
                                        size="small"
                                        sx={{ width: "100%" }}
                                    />
                                )}
                            </FormControl>
                        </Box>
                    </Box>

                    {/* Preview Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Preview</Typography>
                        
                        <Box sx={{ 
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                            minHeight: 200,
                            bgcolor: (variant === "space" || variant === "neon") ? "#001122" : "background.default",
                            borderRadius: 2,
                            p: 4,
                        }}>
                            <MicrophoneButton
                                variant={variant}
                                width={size}
                                height={size}
                                disabled={disabled}
                                showWhenUnavailable={showWhenUnavailable}
                                fill={useCustomFill ? customFill : undefined}
                                onTranscriptChange={(transcript) => {
                                    console.log("Transcript received:", transcript);
                                }}
                            />
                        </Box>

                        <Typography variant="body2" sx={{ mt: 2, textAlign: "center", color: "text.secondary" }}>
                            Click the microphone to start listening. The button will pulse with a green glow when active.
                        </Typography>
                    </Box>

                    {/* All Variants Display */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>All Variants</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { 
                                xs: "repeat(2, 1fr)", 
                                sm: "repeat(4, 1fr)",
                            }, 
                            gap: 3, 
                        }}>
                            {variants.map(v => (
                                <Box 
                                    key={v} 
                                    sx={{ 
                                        p: 3,
                                        borderRadius: 2,
                                        backgroundColor: (v === "space" || v === "neon") ? "#001122" : "grey.100",
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
                                        {v === "solid" ? "Solid (3D)" : v === "neon" ? "Neon" : v}
                                    </Typography>
                                    <MicrophoneButton
                                        variant={v}
                                        width={48}
                                        height={48}
                                        showWhenUnavailable={true}
                                        onTranscriptChange={(transcript) => {
                                            console.log(`${v} variant transcript:`, transcript);
                                        }}
                                    />
                                </Box>
                            ))}
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};

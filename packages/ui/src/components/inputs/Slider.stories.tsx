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
import { Slider } from "./Slider.js";
import type { SliderVariant, SliderSize } from "./Slider.js";

const meta: Meta<typeof Slider> = {
    title: "Components/Inputs/Slider",
    component: Slider,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive Slider Playground with all variants
export const SliderShowcase: Story = {
    render: () => {
        const [size, setSize] = useState<SliderSize>("md");
        const [disabled, setDisabled] = useState(false);
        const [showValue, setShowValue] = useState(true);
        const [showMarks, setShowMarks] = useState(false);
        const [customColor, setCustomColor] = useState("#9333EA");
        const [min, setMin] = useState(0);
        const [max, setMax] = useState(100);
        const [step, setStep] = useState(1);
        const [throttleMs, setThrottleMs] = useState(100);

        const variants: SliderVariant[] = ["default", "primary", "secondary", "success", "warning", "danger", "space", "neon", "custom"];

        // State for individual slider values - all sliders default to 50
        const [sliderValues, setSliderValues] = useState<Record<string, number>>({
            // Live preview values
            'preview-default': 50,
            'preview-primary': 50,
            'preview-secondary': 50,
            'preview-success': 50,
            'preview-warning': 50,
            'preview-danger': 50,
            'preview-space': 50,
            'preview-neon': 50,
            'preview-custom': 50,
            // All variants section
            default: 50,
            primary: 50,
            secondary: 50,
            success: 50,
            warning: 50,
            danger: 50,
            space: 50,
            neon: 50,
            custom: 50,
            // Size comparison
            'size-sm': 50,
            'size-md': 50,
            'size-lg': 50,
            // Common use cases
            volume: 25,
            brightness: 75,
            temperature: 22,
            progress: 60,
            opacity: 80,
            zoom: 100,
            // Range examples
            rating: 5,
            decimal: 0.5,
            negative: 0,
            // Controlled example
            controlled: 30,
        });

        const handleSliderChange = (key: string, value: number) => {
            setSliderValues(prev => ({ ...prev, [key]: value }));
        };

        // Generate marks based on current range
        const generateMarks = () => {
            const marks = [];
            const range = max - min;
            const markCount = Math.min(5, Math.floor(range / step) + 1);
            for (let i = 0; i < markCount; i++) {
                const rawValue = min + (range * i) / (markCount - 1);
                const value = Math.round(rawValue / step) * step;
                
                // Format value to avoid floating point display issues
                const formattedValue = step && step < 1 
                    ? Number(value.toFixed(String(step).split('.')[1]?.length || 1))
                    : Math.round(value);
                
                marks.push({ value: formattedValue });
            }
            return marks;
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
                        <Typography variant="h5" sx={{ mb: 3 }}>Slider Controls</Typography>
                        
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
                                    onChange={(e) => setSize(e.target.value as SliderSize)}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="sm" control={<Radio size="small" />} label="Small" sx={{ m: 0 }} />
                                    <FormControlLabel value="md" control={<Radio size="small" />} label="Medium" sx={{ m: 0 }} />
                                    <FormControlLabel value="lg" control={<Radio size="small" />} label="Large" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            {/* Range Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Range</FormLabel>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                                    <TextField
                                        label="Min"
                                        type="number"
                                        value={min}
                                        onChange={(e) => setMin(Number(e.target.value))}
                                        size="small"
                                    />
                                    <TextField
                                        label="Max"
                                        type="number"
                                        value={max}
                                        onChange={(e) => setMax(Number(e.target.value))}
                                        size="small"
                                    />
                                </Box>
                            </FormControl>

                            {/* Step Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Step</FormLabel>
                                <TextField
                                    type="number"
                                    value={step}
                                    onChange={(e) => setStep(Number(e.target.value))}
                                    size="small"
                                    inputProps={{ min: 0.1, step: 0.1 }}
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
                                />
                            </FormControl>

                            {/* Throttle Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Throttle (ms)</FormLabel>
                                <TextField
                                    type="number"
                                    value={throttleMs}
                                    onChange={(e) => setThrottleMs(Number(e.target.value))}
                                    size="small"
                                    inputProps={{ min: 0, max: 1000, step: 50 }}
                                    helperText="0 = no throttle"
                                />
                            </FormControl>

                            {/* Options Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Options</FormLabel>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 0.5 }}>
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
                                                checked={showValue}
                                                onChange={(e) => setShowValue(e.target.checked)}
                                                size="small"
                                            />
                                        }
                                        label="Show Value"
                                        sx={{ m: 0 }}
                                    />
                                    <FormControlLabel
                                        control={
                                            <Switch
                                                checked={showMarks}
                                                onChange={(e) => setShowMarks(e.target.checked)}
                                                size="small"
                                            />
                                        }
                                        label="Show Marks"
                                        sx={{ m: 0 }}
                                    />
                                </Box>
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
                                md: "repeat(3, 1fr)"
                            }, 
                            gap: 4 
                        }}>
                            {variants.map(v => (
                                <Box 
                                    key={`preview-${v}`} 
                                    sx={{ 
                                        p: 3,
                                        borderRadius: 2,
                                        backgroundColor: (v === "space" || v === "neon") ? "#001122" : "transparent"
                                    }}
                                >
                                    <Typography 
                                        variant="subtitle2" 
                                        sx={{ 
                                            mb: 2, 
                                            textTransform: "capitalize",
                                            color: (v === "space" || v === "neon") ? "#fff" : "inherit",
                                            textAlign: "center"
                                        }}
                                    >
                                        {v}
                                    </Typography>
                                    <Slider
                                        variant={v}
                                        size={size}
                                        min={min}
                                        max={max}
                                        step={step}
                                        disabled={disabled}
                                        showValue={showValue}
                                        marks={showMarks ? generateMarks() : undefined}
                                        color={v === "custom" ? customColor : undefined}
                                        value={sliderValues[`preview-${v}`]}
                                        onChange={(value) => handleSliderChange(`preview-${v}`, value)}
                                        throttleMs={throttleMs}
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
                            gap: 4,
                            alignItems: "center"
                        }}>
                            {["sm", "md", "lg"].map(s => (
                                <Box key={s} sx={{ width: "100%", maxWidth: 400 }}>
                                    <Typography variant="subtitle2" sx={{ mb: 1, textAlign: "center" }}>
                                        Size: {s.toUpperCase()}
                                    </Typography>
                                    <Slider
                                        variant="primary"
                                        size={s as SliderSize}
                                        label={`${s.toUpperCase()} size slider`}
                                        showValue
                                        value={sliderValues[`size-${s}`]}
                                        onChange={(value) => handleSliderChange(`size-${s}`, value)}
                                        throttleMs={throttleMs}
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
                            {/* Media Controls */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Media Controls</Typography>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 3, p: 2, borderRadius: 1 }}>
                                    <Slider
                                        variant="primary"
                                        label="Volume"
                                        min={0}
                                        max={100}
                                        showValue
                                        marks={[
                                            { value: 0, label: "Mute" },
                                            { value: 50, label: "50%" },
                                            { value: 100, label: "Max" }
                                        ]}
                                        value={sliderValues.volume}
                                        onChange={(value) => handleSliderChange("volume", value)}
                                        throttleMs={throttleMs}
                                    />
                                    <Slider
                                        variant="secondary"
                                        label="Brightness"
                                        min={0}
                                        max={100}
                                        showValue
                                        value={sliderValues.brightness}
                                        onChange={(value) => handleSliderChange("brightness", value)}
                                        throttleMs={throttleMs}
                                    />
                                </Box>
                            </Box>

                            {/* Settings Panel */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Settings Panel</Typography>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 3, p: 2, borderRadius: 1 }}>
                                    <Slider
                                        variant="success"
                                        label="Temperature (°C)"
                                        min={15}
                                        max={30}
                                        step={0.5}
                                        showValue
                                        value={sliderValues.temperature}
                                        onChange={(value) => handleSliderChange("temperature", value)}
                                        throttleMs={throttleMs}
                                    />
                                    <Slider
                                        variant="warning"
                                        label="Opacity (%)"
                                        min={0}
                                        max={100}
                                        step={5}
                                        showValue
                                        value={sliderValues.opacity}
                                        onChange={(value) => handleSliderChange("opacity", value)}
                                        throttleMs={throttleMs}
                                    />
                                </Box>
                            </Box>

                            {/* Special Themed Examples */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Themed Examples</Typography>
                                <Box sx={{ 
                                    display: "flex", 
                                    flexDirection: "column", 
                                    gap: 3, 
                                    p: 3, 
                                    bgcolor: "#001122", 
                                    borderRadius: 1 
                                }}>
                                    <Slider
                                        variant="space"
                                        label="Warp Drive Power"
                                        min={0}
                                        max={100}
                                        showValue
                                        value={sliderValues.progress}
                                        onChange={(value) => handleSliderChange("progress", value)}
                                        throttleMs={throttleMs}
                                    />
                                    <Slider
                                        variant="neon"
                                        label="Neon Intensity"
                                        min={0}
                                        max={100}
                                        showValue
                                        value={sliderValues.zoom}
                                        onChange={(value) => handleSliderChange("zoom", value)}
                                        throttleMs={throttleMs}
                                    />
                                </Box>
                            </Box>

                            {/* Different Ranges */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Different Ranges & Steps</Typography>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 3, p: 2, borderRadius: 1 }}>
                                    <Slider
                                        variant="default"
                                        label="Rating (1-10)"
                                        min={1}
                                        max={10}
                                        step={1}
                                        showValue
                                        marks={Array.from({ length: 10 }, (_, i) => ({ value: i + 1 }))}
                                        value={sliderValues.rating}
                                        onChange={(value) => handleSliderChange("rating", value)}
                                        throttleMs={throttleMs}
                                    />
                                    <Slider
                                        variant="secondary"
                                        label="Decimal Precision (0.0-1.0)"
                                        min={0}
                                        max={1}
                                        step={0.1}
                                        showValue
                                        value={sliderValues.decimal}
                                        onChange={(value) => handleSliderChange("decimal", value)}
                                        throttleMs={throttleMs}
                                    />
                                    <Slider
                                        variant="danger"
                                        label="Temperature Range (-50°C to 50°C)"
                                        min={-50}
                                        max={50}
                                        step={5}
                                        showValue
                                        marks={[
                                            { value: -50, label: "-50°C" },
                                            { value: 0, label: "0°C" },
                                            { value: 50, label: "50°C" }
                                        ]}
                                        value={sliderValues.negative}
                                        onChange={(value) => handleSliderChange("negative", value)}
                                        throttleMs={throttleMs}
                                    />
                                </Box>
                            </Box>

                            {/* Accessibility Demo */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Accessibility Features</Typography>
                                <Box sx={{ p: 2, borderRadius: 1 }}>
                                    <Box sx={{ mb: 2, p: 2, bgcolor: "info.light", borderRadius: 1 }}>
                                        <Typography variant="body2" sx={{ fontWeight: "bold", mb: 1 }}>
                                            Keyboard Navigation:
                                        </Typography>
                                        <Typography variant="body2">
                                            • Arrow keys: Increase/decrease by step<br/>
                                            • Home/End: Go to min/max values<br/>
                                            • Page Up/Down: Large increments
                                        </Typography>
                                    </Box>
                                    <Slider
                                        variant="primary"
                                        label="Try keyboard navigation on this slider"
                                        min={0}
                                        max={100}
                                        step={5}
                                        showValue
                                        marks={[
                                            { value: 0, label: "0%" },
                                            { value: 50, label: "50%" },
                                            { value: 100, label: "100%" }
                                        ]}
                                        value={sliderValues.controlled}
                                        onChange={(value) => handleSliderChange("controlled", value)}
                                        throttleMs={throttleMs}
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
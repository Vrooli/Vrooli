import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import FormControl from "@mui/material/FormControl";
import FormLabel from "@mui/material/FormLabel";
import TextField from "@mui/material/TextField";
import { Radio, type RadioVariant, type RadioSize } from "./Radio.js";
import { FormControlLabel } from "./FormControlLabel.js";
import { FormGroup } from "./FormGroup.js";
import { Switch } from "./Switch/Switch.js";

const meta: Meta<typeof Radio> = {
    title: "Components/Inputs/Radio",
    component: Radio,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive Radio Playground with all variants
export const RadioShowcase: Story = {
    render: () => {
        const [size, setSize] = useState<RadioSize>("md");
        const [disabled, setDisabled] = useState(false);
        const [selectedValue, setSelectedValue] = useState("react");
        const [selectedColor, setSelectedColor] = useState<{ [key: string]: string }>({
            primary: "option1",
            secondary: "option1", 
            danger: "option1",
            success: "option1",
            warning: "option1",
            info: "option1",
            custom: "option1",
        });
        const [customColor, setCustomColor] = useState("#9333EA");

        const variants: RadioVariant[] = ["primary", "secondary", "danger", "success", "warning", "info", "custom"];
        const sizes: RadioSize[] = ["sm", "md", "lg"];

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
                        height: "fit-content",
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Radio Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)" },
                            gap: 3, 
                        }}>
                            {/* Size Control */}
                            <FormControl component="fieldset">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Size</FormLabel>
                                <div>
                                    {sizes.map(s => (
                                        <FormControlLabel
                                            key={s}
                                            control={
                                                <Radio 
                                                    size="sm"
                                                    checked={size === s}
                                                    onChange={() => setSize(s)}
                                                    name="size-control"
                                                />
                                            }
                                            label={s.toUpperCase()}
                                        />
                                    ))}
                                </div>
                            </FormControl>

                            {/* Disabled Control */}
                            <FormControl component="fieldset">
                                <Switch
                                    checked={disabled}
                                    onChange={(checked) => setDisabled(checked)}
                                    size="sm"
                                    label="Disabled"
                                    labelPosition="right"
                                />
                            </FormControl>

                            {/* Custom Color Control */}
                            <FormControl component="fieldset">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Custom Color</FormLabel>
                                <TextField
                                    type="color"
                                    value={customColor}
                                    onChange={(e) => setCustomColor(e.target.value)}
                                    size="small"
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>
                        </Box>
                    </Box>

                    {/* Radio Variants Display */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Radio Button Variants</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { 
                                xs: "1fr", 
                                sm: "repeat(2, 1fr)", 
                                md: "repeat(3, 1fr)",
                            }, 
                            gap: 3, 
                        }}>
                            {variants.map(variant => (
                                <Box key={variant}>
                                    <Typography 
                                        variant="subtitle2" 
                                        sx={{ 
                                            mb: 1, 
                                            textTransform: "capitalize",
                                        }}
                                    >
                                        {variant}
                                    </Typography>
                                    <FormGroup>
                                        {["option1", "option2", "option3"].map((value) => (
                                            <FormControlLabel
                                                key={value}
                                                control={
                                                    <Radio
                                                        color={variant}
                                                        customColor={variant === "custom" ? customColor : undefined}
                                                        size={size}
                                                        disabled={disabled}
                                                        checked={selectedColor[variant] === value}
                                                        onChange={(e) => {
                                                            if (e.target.checked) {
                                                                setSelectedColor(prev => ({
                                                                    ...prev,
                                                                    [variant]: value,
                                                                }));
                                                            }
                                                        }}
                                                        name={`radio-${variant}`}
                                                    />
                                                }
                                                label={`Option ${value.slice(-1)}`}
                                            />
                                        ))}
                                    </FormGroup>
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
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Size Comparison</Typography>
                        
                        <Box sx={{ display: "flex", flexWrap: "wrap", gap: 4, alignItems: "center" }}>
                            {sizes.map(s => (
                                <Box key={s} sx={{ textAlign: "center" }}>
                                    <Typography variant="subtitle2" sx={{ mb: 1 }}>{s.toUpperCase()}</Typography>
                                    <Radio
                                        color="primary"
                                        size={s}
                                        checked={true}
                                        onChange={() => {}}
                                    />
                                </Box>
                            ))}
                        </Box>
                    </Box>

                    {/* Radio Group Example */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Radio Group Example</Typography>
                        
                        <FormControl component="fieldset">
                            <FormLabel component="legend">Select your favorite framework</FormLabel>
                            <FormGroup>
                                <FormControlLabel 
                                    value="react" 
                                    control={
                                        <Radio 
                                            color="primary" 
                                            size={size} 
                                            disabled={disabled}
                                            checked={selectedValue === "react"}
                                            onChange={() => setSelectedValue("react")}
                                            name="framework-group"
                                        />
                                    } 
                                    label="React" 
                                />
                                <FormControlLabel 
                                    value="vue" 
                                    control={
                                        <Radio 
                                            color="primary" 
                                            size={size} 
                                            disabled={disabled}
                                            checked={selectedValue === "vue"}
                                            onChange={() => setSelectedValue("vue")}
                                            name="framework-group"
                                        />
                                    } 
                                    label="Vue" 
                                />
                                <FormControlLabel 
                                    value="angular" 
                                    control={
                                        <Radio 
                                            color="primary" 
                                            size={size} 
                                            disabled={disabled}
                                            checked={selectedValue === "angular"}
                                            onChange={() => setSelectedValue("angular")}
                                            name="framework-group"
                                        />
                                    } 
                                    label="Angular" 
                                />
                                <FormControlLabel 
                                    value="svelte" 
                                    control={
                                        <Radio 
                                            color="primary" 
                                            size={size} 
                                            disabled={disabled}
                                            checked={selectedValue === "svelte"}
                                            onChange={() => setSelectedValue("svelte")}
                                            name="framework-group"
                                        />
                                    } 
                                    label="Svelte" 
                                />
                            </FormGroup>
                        </FormControl>
                        
                        <Typography variant="body2" sx={{ mt: 2 }}>
                            Selected: <strong>{selectedValue}</strong>
                        </Typography>
                    </Box>
                </Box>
            </Box>
        );
    },
};

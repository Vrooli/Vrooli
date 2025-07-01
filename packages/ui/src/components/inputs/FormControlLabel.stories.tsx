import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import FormControl from "@mui/material/FormControl";
import FormLabel from "@mui/material/FormLabel";
import { FormControlLabel, type FormControlLabelPlacement } from "./FormControlLabel.js";
import { FormGroup } from "./FormGroup.js";
import { Radio } from "./Radio.js";
import { Checkbox } from "./Checkbox.js";
import { Switch } from "./Switch/Switch.js";

const meta: Meta<typeof FormControlLabel> = {
    title: "Components/Inputs/FormControlLabel",
    component: FormControlLabel,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive FormControlLabel Playground
export const FormControlLabelShowcase: Story = {
    render: () => {
        const [labelPlacement, setLabelPlacement] = useState<FormControlLabelPlacement>("end");
        const [disabled, setDisabled] = useState(false);
        const [required, setRequired] = useState(false);
        const [controlType, setControlType] = useState<"radio" | "checkbox" | "switch">("checkbox");
        
        // States for different control types section
        const [radioValue, setRadioValue] = useState(false);
        const [checkboxValue, setCheckboxValue] = useState(false);
        const [switchValue, setSwitchValue] = useState(false);
        
        // States for form example
        const [emailNotifications, setEmailNotifications] = useState(true);
        const [smsNotifications, setSmsNotifications] = useState(false);
        const [darkMode, setDarkMode] = useState(false);
        const [autoSave, setAutoSave] = useState(true);
        
        // State for long label example
        const [agreeTerms, setAgreeTerms] = useState(false);
        
        // States for placement examples
        const [placementStates, setPlacementStates] = useState<{ [key: string]: boolean }>({
            end: true,
            start: false,
            top: true,
            bottom: false,
        });

        const placements: FormControlLabelPlacement[] = ["end", "start", "top", "bottom"];

        const getControl = (type: string, placement: string) => {
            switch (type) {
                case "radio":
                    return (
                        <Radio 
                            color="primary" 
                            checked={placementStates[placement]}
                            onChange={(e) => setPlacementStates(prev => ({ ...prev, [placement]: e.target.checked }))}
                        />
                    );
                case "checkbox":
                    return (
                        <Checkbox 
                            color="primary" 
                            checked={placementStates[placement]}
                            onChange={(e) => setPlacementStates(prev => ({ ...prev, [placement]: e.target.checked }))}
                        />
                    );
                case "switch":
                    return (
                        <Switch 
                            variant="default" 
                            checked={placementStates[placement]}
                            onChange={(checked) => setPlacementStates(prev => ({ ...prev, [placement]: checked }))}
                        />
                    );
                default:
                    return (
                        <Checkbox 
                            color="primary" 
                            checked={placementStates[placement]}
                            onChange={(e) => setPlacementStates(prev => ({ ...prev, [placement]: e.target.checked }))}
                        />
                    );
            }
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormControlLabel Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(4, 1fr)" },
                            gap: 3, 
                        }}>
                            {/* Label Placement Control */}
                            <FormControl component="fieldset">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Label Placement</FormLabel>
                                <div>
                                    {placements.map(placement => (
                                        <FormControlLabel
                                            key={placement}
                                            control={
                                                <Radio 
                                                    size="sm" 
                                                    checked={labelPlacement === placement}
                                                    onChange={() => setLabelPlacement(placement)}
                                                    name="label-placement"
                                                />
                                            }
                                            label={placement.charAt(0).toUpperCase() + placement.slice(1)}
                                        />
                                    ))}
                                </div>
                            </FormControl>

                            {/* Control Type */}
                            <FormControl component="fieldset">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Control Type</FormLabel>
                                <div>
                                    <FormControlLabel 
                                        control={
                                            <Radio 
                                                size="sm" 
                                                checked={controlType === "radio"}
                                                onChange={() => setControlType("radio")}
                                                name="control-type"
                                            />
                                        } 
                                        label="Radio" 
                                    />
                                    <FormControlLabel 
                                        control={
                                            <Radio 
                                                size="sm" 
                                                checked={controlType === "checkbox"}
                                                onChange={() => setControlType("checkbox")}
                                                name="control-type"
                                            />
                                        } 
                                        label="Checkbox" 
                                    />
                                    <FormControlLabel 
                                        control={
                                            <Radio 
                                                size="sm" 
                                                checked={controlType === "switch"}
                                                onChange={() => setControlType("switch")}
                                                name="control-type"
                                            />
                                        } 
                                        label="Switch" 
                                    />
                                </div>
                            </FormControl>

                            {/* State Controls */}
                            <FormControl component="fieldset">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>States</FormLabel>
                                <FormGroup>
                                    <Switch
                                        checked={disabled}
                                        onChange={(checked) => setDisabled(checked)}
                                        size="sm"
                                        label="Disabled"
                                        labelPosition="right"
                                    />
                                    <Switch
                                        checked={required}
                                        onChange={(checked) => setRequired(checked)}
                                        size="sm"
                                        label="Required"
                                        labelPosition="right"
                                    />
                                </FormGroup>
                            </FormControl>
                        </Box>
                    </Box>

                    {/* Placement Examples */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Label Placement Examples</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { 
                                xs: "1fr", 
                                sm: "repeat(2, 1fr)",
                            }, 
                            gap: 4, 
                        }}>
                            {placements.map(placement => (
                                <Box 
                                    key={placement} 
                                    sx={{ 
                                        p: 2, 
                                        border: 1, 
                                        borderColor: "divider",
                                        borderRadius: 1,
                                        display: "flex",
                                        alignItems: "center",
                                        justifyContent: "center",
                                        minHeight: 100,
                                    }}
                                >
                                    <Box sx={{ textAlign: "center" }}>
                                        <Typography 
                                            variant="subtitle2" 
                                            sx={{ 
                                                mb: 2, 
                                                textTransform: "capitalize",
                                            }}
                                        >
                                            {placement}
                                        </Typography>
                                        <FormControlLabel
                                            control={getControl(controlType, placement)}
                                            label="Option Label"
                                            labelPlacement={placement}
                                            disabled={disabled}
                                            required={required}
                                        />
                                    </Box>
                                </Box>
                            ))}
                        </Box>
                    </Box>

                    {/* Different Control Types */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Different Control Types</Typography>
                        
                        <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                            <FormControlLabel
                                control={
                                    <Radio 
                                        color="primary" 
                                        checked={radioValue}
                                        onChange={(e) => setRadioValue(e.target.checked)}
                                    />
                                }
                                label="Radio Button Option"
                                labelPlacement={labelPlacement}
                                disabled={disabled}
                                required={required}
                            />
                            <FormControlLabel
                                control={
                                    <Checkbox 
                                        color="primary" 
                                        checked={checkboxValue}
                                        onChange={(e) => setCheckboxValue(e.target.checked)}
                                    />
                                }
                                label="Checkbox Option"
                                labelPlacement={labelPlacement}
                                disabled={disabled}
                                required={required}
                            />
                            <FormControlLabel
                                control={
                                    <Switch 
                                        variant="default" 
                                        checked={switchValue}
                                        onChange={(checked) => setSwitchValue(checked)}
                                    />
                                }
                                label="Switch Option"
                                labelPlacement={labelPlacement}
                                disabled={disabled}
                                required={required}
                            />
                        </Box>
                    </Box>

                    {/* Long Label Example */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Long Label Example</Typography>
                        
                        <Box sx={{ maxWidth: 600 }}>
                            <FormControlLabel
                                control={
                                    <Checkbox 
                                        color="primary" 
                                        checked={agreeTerms}
                                        onChange={(e) => setAgreeTerms(e.target.checked)}
                                    />
                                }
                                label="I agree to the terms and conditions and privacy policy. This is a very long label to demonstrate how the component handles text wrapping and alignment."
                                labelPlacement={labelPlacement}
                                disabled={disabled}
                                required={required}
                            />
                        </Box>
                    </Box>

                    {/* Form Example */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Form Example</Typography>
                        
                        <FormControl component="fieldset">
                            <FormLabel component="legend">User Preferences</FormLabel>
                            <FormGroup>
                                <FormControlLabel
                                    control={
                                        <Checkbox 
                                            color="primary" 
                                            checked={emailNotifications}
                                            onChange={(e) => setEmailNotifications(e.target.checked)}
                                        />
                                    }
                                    label="Email notifications"
                                    labelPlacement={labelPlacement}
                                    disabled={disabled}
                                />
                                <FormControlLabel
                                    control={
                                        <Checkbox 
                                            color="primary" 
                                            checked={smsNotifications}
                                            onChange={(e) => setSmsNotifications(e.target.checked)}
                                        />
                                    }
                                    label="SMS notifications"
                                    labelPlacement={labelPlacement}
                                    disabled={disabled}
                                />
                                <FormControlLabel
                                    control={
                                        <Switch 
                                            variant="default" 
                                            checked={darkMode}
                                            onChange={(checked) => setDarkMode(checked)}
                                        />
                                    }
                                    label="Dark mode"
                                    labelPlacement={labelPlacement}
                                    disabled={disabled}
                                />
                                <FormControlLabel
                                    control={
                                        <Switch 
                                            variant="default" 
                                            checked={autoSave}
                                            onChange={(checked) => setAutoSave(checked)}
                                        />
                                    }
                                    label="Auto-save"
                                    labelPlacement={labelPlacement}
                                    disabled={disabled}
                                />
                            </FormGroup>
                        </FormControl>
                    </Box>
                </Box>
            </Box>
        );
    },
};

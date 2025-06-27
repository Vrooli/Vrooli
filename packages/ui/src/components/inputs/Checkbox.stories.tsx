import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import FormControl from "@mui/material/FormControl";
import FormLabel from "@mui/material/FormLabel";
import TextField from "@mui/material/TextField";
import { Checkbox, type CheckboxVariant, type CheckboxSize } from "./Checkbox.js";
import { FormControlLabel } from "./FormControlLabel.js";
import { FormGroup } from "./FormGroup.js";
import { Switch } from "./Switch/Switch.js";
import { Radio } from "./Radio.js";

const meta: Meta<typeof Checkbox> = {
    title: "Components/Inputs/Checkbox",
    component: Checkbox,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive Checkbox Playground with all variants
export const CheckboxShowcase: Story = {
    render: () => {
        const [size, setSize] = useState<CheckboxSize>("md");
        const [disabled, setDisabled] = useState(false);
        const [showIndeterminate, setShowIndeterminate] = useState(false);
        const [checkedItems, setCheckedItems] = useState<{ [key: string]: boolean }>({
            item1: true,
            item2: false,
            item3: true,
        });
        const [checkboxStates, setCheckboxStates] = useState<{ [key: string]: { checked: boolean, unchecked: boolean, indeterminate: boolean } }>({
            primary: { checked: true, unchecked: false, indeterminate: false },
            secondary: { checked: true, unchecked: false, indeterminate: false },
            danger: { checked: true, unchecked: false, indeterminate: false },
            success: { checked: true, unchecked: false, indeterminate: false },
            warning: { checked: true, unchecked: false, indeterminate: false },
            info: { checked: true, unchecked: false, indeterminate: false },
            custom: { checked: true, unchecked: false, indeterminate: false },
        });
        const [customColor, setCustomColor] = useState("#9333EA");

        const variants: CheckboxVariant[] = ["primary", "secondary", "danger", "success", "warning", "info", "custom"];
        const sizes: CheckboxSize[] = ["sm", "md", "lg"];

        const handleParentChange = (checked: boolean) => {
            setCheckedItems({
                item1: checked,
                item2: checked,
                item3: checked,
            });
        };

        const allChecked = Object.values(checkedItems).every(v => v);
        const someChecked = Object.values(checkedItems).some(v => v);
        const parentIndeterminate = someChecked && !allChecked;

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
                        <Typography variant="h5" sx={{ mb: 3 }}>Checkbox Controls</Typography>
                        
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
                                        checked={showIndeterminate}
                                        onChange={(checked) => setShowIndeterminate(checked)}
                                        size="sm"
                                        label="Show Indeterminate"
                                        labelPosition="right"
                                    />
                                </FormGroup>
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

                    {/* Checkbox Variants Display */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Checkbox Variants</Typography>
                        
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
                                        <FormControlLabel
                                            control={
                                                <Checkbox
                                                    color={variant}
                                                    customColor={variant === "custom" ? customColor : undefined}
                                                    size={size}
                                                    disabled={disabled}
                                                    checked={checkboxStates[variant].checked}
                                                    onChange={(e) => {
                                                        setCheckboxStates(prev => ({
                                                            ...prev,
                                                            [variant]: {
                                                                ...prev[variant],
                                                                checked: e.target.checked,
                                                            },
                                                        }));
                                                    }}
                                                />
                                            }
                                            label="Checked"
                                        />
                                        <FormControlLabel
                                            control={
                                                <Checkbox
                                                    color={variant}
                                                    customColor={variant === "custom" ? customColor : undefined}
                                                    size={size}
                                                    disabled={disabled}
                                                    checked={checkboxStates[variant].unchecked}
                                                    onChange={(e) => {
                                                        setCheckboxStates(prev => ({
                                                            ...prev,
                                                            [variant]: {
                                                                ...prev[variant],
                                                                unchecked: e.target.checked,
                                                            },
                                                        }));
                                                    }}
                                                />
                                            }
                                            label="Unchecked"
                                        />
                                        {showIndeterminate && (
                                            <FormControlLabel
                                                control={
                                                    <Checkbox
                                                        color={variant}
                                                        customColor={variant === "custom" ? customColor : undefined}
                                                        size={size}
                                                        disabled={disabled}
                                                        indeterminate={true}
                                                    />
                                                }
                                                label="Indeterminate"
                                            />
                                        )}
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
                                    <Checkbox
                                        color="primary"
                                        size={s}
                                        checked={true}
                                        onChange={() => {}}
                                    />
                                </Box>
                            ))}
                        </Box>
                    </Box>

                    {/* Indeterminate Example */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Indeterminate State Example</Typography>
                        <Typography variant="body2" sx={{ mb: 2 }}>
                            Parent checkbox shows indeterminate state when some but not all children are selected
                        </Typography>
                        
                        <FormControl component="fieldset">
                            <FormControlLabel
                                control={
                                    <Checkbox
                                        color="primary"
                                        size={size}
                                        disabled={disabled}
                                        checked={allChecked}
                                        indeterminate={parentIndeterminate}
                                        onChange={(e) => handleParentChange(e.target.checked)}
                                    />
                                }
                                label="Select All"
                            />
                            <Box sx={{ display: "flex", flexDirection: "column", ml: 3 }}>
                                <FormControlLabel
                                    control={
                                        <Checkbox
                                            color="primary"
                                            size={size}
                                            disabled={disabled}
                                            checked={checkedItems.item1}
                                            onChange={(e) => setCheckedItems(prev => ({
                                                ...prev,
                                                item1: e.target.checked,
                                            }))}
                                        />
                                    }
                                    label="Item 1"
                                />
                                <FormControlLabel
                                    control={
                                        <Checkbox
                                            color="primary"
                                            size={size}
                                            disabled={disabled}
                                            checked={checkedItems.item2}
                                            onChange={(e) => setCheckedItems(prev => ({
                                                ...prev,
                                                item2: e.target.checked,
                                            }))}
                                        />
                                    }
                                    label="Item 2"
                                />
                                <FormControlLabel
                                    control={
                                        <Checkbox
                                            color="primary"
                                            size={size}
                                            disabled={disabled}
                                            checked={checkedItems.item3}
                                            onChange={(e) => setCheckedItems(prev => ({
                                                ...prev,
                                                item3: e.target.checked,
                                            }))}
                                        />
                                    }
                                    label="Item 3"
                                />
                            </Box>
                        </FormControl>
                    </Box>

                    {/* Checkbox List Example */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Task List Example</Typography>
                        
                        <FormControl component="fieldset">
                            <FormLabel component="legend">Daily Tasks</FormLabel>
                            <FormGroup>
                                {[
                                    "Complete project documentation",
                                    "Review pull requests",
                                    "Attend team standup",
                                    "Update task board",
                                    "Write unit tests",
                                ].map((task, index) => (
                                    <FormControlLabel
                                        key={index}
                                        control={
                                            <Checkbox
                                                color="success"
                                                size={size}
                                                disabled={disabled}
                                            />
                                        }
                                        label={task}
                                    />
                                ))}
                            </FormGroup>
                        </FormControl>
                    </Box>
                </Box>
            </Box>
        );
    },
};

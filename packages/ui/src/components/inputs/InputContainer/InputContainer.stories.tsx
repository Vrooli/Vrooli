import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextField from "@mui/material/TextField";
import FormLabel from "@mui/material/FormLabel";
import MenuItem from "@mui/material/MenuItem";
import { IconCommon } from "../../../icons/Icons.js";
import { Switch } from "../Switch/Switch.js";
import { InputContainer } from "./InputContainer.js";

const meta: Meta<typeof InputContainer> = {
    title: "Components/Inputs/InputContainer",
    component: InputContainer,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
    render: () => {
        const [variant, setVariant] = useState<"outline" | "filled" | "underline">("filled");
        const [size, setSize] = useState<"sm" | "md" | "lg">("md");
        const [error, setError] = useState(false);
        const [disabled, setDisabled] = useState(false);
        const [fullWidth, setFullWidth] = useState(false);
        const [focused, setFocused] = useState(false);
        const [showLabel, setShowLabel] = useState(true);
        const [showHelperText, setShowHelperText] = useState(true);
        const [showStartAdornment, setShowStartAdornment] = useState(false);
        const [showEndAdornment, setShowEndAdornment] = useState(false);
        const [labelText, setLabelText] = useState("Input Label");
        const [helperText, setHelperText] = useState("Helper text goes here");

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
                        <Typography variant="h5" sx={{ mb: 3 }}>InputContainer Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)" },
                            gap: 3, 
                        }}>
                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Variant</FormLabel>
                                <TextField
                                    select
                                    value={variant}
                                    onChange={(e) => setVariant(e.target.value as any)}
                                    size="small"
                                    fullWidth
                                >
                                    <MenuItem value="outline">Outline</MenuItem>
                                    <MenuItem value="filled">Filled</MenuItem>
                                    <MenuItem value="underline">Underline</MenuItem>
                                </TextField>
                            </Box>

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Size</FormLabel>
                                <TextField
                                    select
                                    value={size}
                                    onChange={(e) => setSize(e.target.value as any)}
                                    size="small"
                                    fullWidth
                                >
                                    <MenuItem value="sm">Small</MenuItem>
                                    <MenuItem value="md">Medium</MenuItem>
                                    <MenuItem value="lg">Large</MenuItem>
                                </TextField>
                            </Box>

                            <Switch
                                checked={error}
                                onChange={(checked) => setError(checked)}
                                size="sm"
                                label="Error State"
                                labelPosition="right"
                            />

                            <Switch
                                checked={disabled}
                                onChange={(checked) => setDisabled(checked)}
                                size="sm"
                                label="Disabled"
                                labelPosition="right"
                            />

                            <Switch
                                checked={fullWidth}
                                onChange={(checked) => setFullWidth(checked)}
                                size="sm"
                                label="Full Width"
                                labelPosition="right"
                            />

                            <Switch
                                checked={focused}
                                onChange={(checked) => setFocused(checked)}
                                size="sm"
                                label="Focused State"
                                labelPosition="right"
                            />

                            <Switch
                                checked={showLabel}
                                onChange={(checked) => setShowLabel(checked)}
                                size="sm"
                                label="Show Label"
                                labelPosition="right"
                            />

                            <Switch
                                checked={showHelperText}
                                onChange={(checked) => setShowHelperText(checked)}
                                size="sm"
                                label="Show Helper Text"
                                labelPosition="right"
                            />

                            <Switch
                                checked={showStartAdornment}
                                onChange={(checked) => setShowStartAdornment(checked)}
                                size="sm"
                                label="Start Adornment"
                                labelPosition="right"
                            />

                            <Switch
                                checked={showEndAdornment}
                                onChange={(checked) => setShowEndAdornment(checked)}
                                size="sm"
                                label="End Adornment"
                                labelPosition="right"
                            />

                            <Switch
                                checked={true}
                                onChange={() => {}}
                                size="sm"
                                label="Theme Colors (Auto)"
                                labelPosition="right"
                                disabled
                            />

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Label Text</FormLabel>
                                <TextField
                                    value={labelText}
                                    onChange={(e) => setLabelText(e.target.value)}
                                    size="small"
                                    fullWidth
                                />
                            </Box>

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Helper Text</FormLabel>
                                <TextField
                                    value={helperText}
                                    onChange={(e) => setHelperText(e.target.value)}
                                    size="small"
                                    fullWidth
                                />
                            </Box>
                        </Box>
                    </Box>

                    {/* Examples Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>InputContainer Examples</Typography>
                        
                        <Box sx={{ display: "flex", flexDirection: "column", gap: 4 }}>
                            {/* Text Input Example */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Text Input</Typography>
                                <InputContainer
                                    variant={variant}
                                    size={size}
                                    error={error}
                                    disabled={disabled}
                                    fullWidth={fullWidth}
                                    focused={focused}
                                    label={showLabel ? labelText : undefined}
                                    isRequired
                                    helperText={showHelperText ? helperText : undefined}
                                    startAdornment={showStartAdornment ? <IconCommon name="Search" /> : undefined}
                                    endAdornment={showEndAdornment ? <IconCommon name="Close" /> : undefined}
                                    htmlFor="text-input"
                                >
                                    <input
                                        id="text-input"
                                        type="text"
                                        placeholder="Enter text..."
                                        disabled={disabled}
                                        className="tw-w-full tw-bg-transparent tw-border-0 tw-outline-none focus:tw-outline-none focus:tw-ring-0 tw-text-text-primary placeholder:tw-text-text-secondary"
                                    />
                                </InputContainer>
                            </Box>

                            {/* Select Example */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Select</Typography>
                                <InputContainer
                                    variant={variant}
                                    size={size}
                                    error={error}
                                    disabled={disabled}
                                    fullWidth={fullWidth}
                                    focused={focused}
                                    label={showLabel ? "Select an option" : undefined}
                                    helperText={showHelperText ? "Choose from the dropdown" : undefined}
                                    endAdornment={<IconCommon name="ChevronDown" />}
                                    onClick={() => console.log("Container clicked")}
                                >
                                    <div className="tw-flex-1 tw-flex tw-items-center tw-text-text-primary">
                                        Option 1
                                    </div>
                                </InputContainer>
                            </Box>

                            {/* Custom Content Example */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Custom Content</Typography>
                                <InputContainer
                                    variant={variant}
                                    size={size}
                                    error={error}
                                    disabled={disabled}
                                    fullWidth={fullWidth}
                                    focused={focused}
                                    label={showLabel ? "Custom input" : undefined}
                                    helperText={showHelperText ? "This can contain any content" : undefined}
                                >
                                    <div className="tw-flex tw-items-center tw-gap-2 tw-w-full">
                                        <span className="tw-text-text-secondary">$</span>
                                        <input
                                            type="number"
                                            placeholder="0.00"
                                            disabled={disabled}
                                            className="tw-flex-1 tw-bg-transparent tw-border-0 tw-outline-none focus:tw-outline-none focus:tw-ring-0 tw-text-text-primary placeholder:tw-text-text-secondary"
                                        />
                                        <span className="tw-text-text-secondary">USD</span>
                                    </div>
                                </InputContainer>
                            </Box>
                        </Box>
                    </Box>

                    {/* All Variants */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>All Variants</Typography>
                        
                        <Box sx={{ display: "grid", gridTemplateColumns: { xs: "1fr", md: "repeat(3, 1fr)" }, gap: 3 }}>
                            {(["filled", "outline", "underline"] as const).map((v) => (
                                <Box key={v}>
                                    <Typography variant="subtitle2" sx={{ mb: 1, textTransform: "capitalize" }}>{v}</Typography>
                                    <InputContainer
                                        variant={v}
                                        label="Input label"
                                        helperText="Helper text"
                                        startAdornment={<IconCommon name="User" />}
                                    >
                                        <input
                                            type="text"
                                            placeholder="Placeholder..."
                                            className="tw-w-full tw-bg-transparent tw-border-0 tw-outline-none focus:tw-outline-none focus:tw-ring-0 tw-text-text-primary placeholder:tw-text-text-secondary"
                                        />
                                    </InputContainer>
                                </Box>
                            ))}
                        </Box>
                    </Box>

                    {/* All Sizes */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>All Sizes</Typography>
                        
                        <Box sx={{ display: "grid", gridTemplateColumns: { xs: "1fr", md: "repeat(3, 1fr)" }, gap: 3 }}>
                            {(["sm", "md", "lg"] as const).map((s) => (
                                <Box key={s}>
                                    <Typography variant="subtitle2" sx={{ mb: 1 }}>Size: {s}</Typography>
                                    <InputContainer
                                        size={s}
                                        label="Input label"
                                        helperText="Helper text"
                                    >
                                        <input
                                            type="text"
                                            placeholder="Placeholder..."
                                            className="tw-w-full tw-bg-transparent tw-border-0 tw-outline-none focus:tw-outline-none focus:tw-ring-0 tw-text-text-primary placeholder:tw-text-text-secondary"
                                        />
                                    </InputContainer>
                                </Box>
                            ))}
                        </Box>
                    </Box>

                    {/* All States */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>All States</Typography>
                        
                        <Box sx={{ display: "grid", gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)" }, gap: 3 }}>
                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>Normal State</Typography>
                                <InputContainer
                                    label="Normal input"
                                    helperText="This is normal helper text"
                                    startAdornment={<IconCommon name="User" />}
                                >
                                    <input
                                        type="text"
                                        placeholder="Enter text..."
                                        className="tw-w-full tw-bg-transparent tw-border-0 tw-outline-none focus:tw-outline-none focus:tw-ring-0 tw-text-text-primary placeholder:tw-text-text-secondary"
                                    />
                                </InputContainer>
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>Error State</Typography>
                                <InputContainer
                                    label="Error input"
                                    helperText="This is an error message"
                                    error={true}
                                    startAdornment={<IconCommon name="User" />}
                                >
                                    <input
                                        type="text"
                                        placeholder="Enter text..."
                                        className="tw-w-full tw-bg-transparent tw-border-0 tw-outline-none focus:tw-outline-none focus:tw-ring-0 tw-text-text-primary placeholder:tw-text-text-secondary"
                                    />
                                </InputContainer>
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>Disabled State</Typography>
                                <InputContainer
                                    label="Disabled input"
                                    helperText="This input is disabled"
                                    disabled={true}
                                    startAdornment={<IconCommon name="User" />}
                                >
                                    <input
                                        type="text"
                                        placeholder="Enter text..."
                                        disabled
                                        className="tw-w-full tw-bg-transparent tw-border-0 tw-outline-none focus:tw-outline-none focus:tw-ring-0 tw-text-text-primary placeholder:tw-text-text-secondary"
                                    />
                                </InputContainer>
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>Required Field</Typography>
                                <InputContainer
                                    label="Required input"
                                    helperText="This field is required"
                                    isRequired={true}
                                    startAdornment={<IconCommon name="User" />}
                                >
                                    <input
                                        type="text"
                                        placeholder="Enter text..."
                                        className="tw-w-full tw-bg-transparent tw-border-0 tw-outline-none focus:tw-outline-none focus:tw-ring-0 tw-text-text-primary placeholder:tw-text-text-secondary"
                                    />
                                </InputContainer>
                            </Box>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};

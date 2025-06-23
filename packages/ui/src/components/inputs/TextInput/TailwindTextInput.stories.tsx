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
import { TailwindTextInputBase } from "./TailwindTextInput.js";
import type { TextInputVariant, TextInputSize } from "./types.js";
import { Switch } from "../Switch/Switch.js";

// Simple adornment components for the showcase
const SearchIcon = () => (
    <svg className="tw-w-5 tw-h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
    </svg>
);

const UserIcon = () => (
    <svg className="tw-w-5 tw-h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
    </svg>
);

const EyeIcon = () => (
    <svg className="tw-w-5 tw-h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        <path strokeLinecap="round" strokeLinejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
    </svg>
);

const SendIcon = () => (
    <svg className="tw-w-5 tw-h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
    </svg>
);

const meta: Meta<typeof TailwindTextInputBase> = {
    title: "Components/Inputs/TailwindTextInput",
    component: TailwindTextInputBase,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive TextInput Playground with all variants
export const TextInputShowcase: Story = {
    render: () => {
        const [variant, setVariant] = useState<TextInputVariant>("filled");
        const [size, setSize] = useState<TextInputSize>("md");
        const [error, setError] = useState(false);
        const [disabled, setDisabled] = useState(false);
        const [fullWidth, setFullWidth] = useState(false);
        const [isRequired, setIsRequired] = useState(false);
        const [multiline, setMultiline] = useState(false);
        const [labelText, setLabelText] = useState("Sample Label");
        const [placeholder, setPlaceholder] = useState("Enter text here...");
        const [helperText, setHelperText] = useState("This is helper text");
        const [showHelperText, setShowHelperText] = useState(false);
        const [showLabel, setShowLabel] = useState(true);
        const [showStartAdornment, setShowStartAdornment] = useState(false);
        const [showEndAdornment, setShowEndAdornment] = useState(false);
        const [startAdornmentType, setStartAdornmentType] = useState<"icon" | "text">("icon");
        const [endAdornmentType, setEndAdornmentType] = useState<"icon" | "text">("icon");

        const variants: TextInputVariant[] = ["outline", "filled", "underline"];
        const sizes: TextInputSize[] = ["sm", "md", "lg"];

        // State for individual inputs
        const [inputValues, setInputValues] = useState<Record<string, string>>({
            preview: "Sample input value",
            outline: "",
            filled: "",
            underline: "",
            "size-sm": "",
            "size-md": "",
            "size-lg": "",
            multilineExample: "This is a sample\nmultiline text\nwith multiple lines...",
            form1: "",
            form2: "",
            form3: "",
            form4: "",
        });

        const handleInputChange = (key: string, value: string) => {
            setInputValues(prev => ({ ...prev, [key]: value }));
        };

        // Helper functions for adornments
        const getStartAdornment = () => {
            if (!showStartAdornment) return undefined;
            if (startAdornmentType === "icon") {
                return <SearchIcon />;
            }
            return <span className="tw-text-sm tw-font-medium">$</span>;
        };

        const getEndAdornment = () => {
            if (!showEndAdornment) return undefined;
            if (endAdornmentType === "icon") {
                return (
                    <button 
                        type="button"
                        className="tw-flex tw-items-center tw-justify-center hover:tw-text-primary-main tw-transition-colors tw-cursor-pointer tw-p-0 tw-m-0 tw-border-0 tw-bg-transparent tw-text-current"
                    >
                        <EyeIcon />
                    </button>
                );
            }
            return <span className="tw-text-sm tw-font-medium">.com</span>;
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
                        <Typography variant="h5" sx={{ mb: 3 }}>TextInput Controls</Typography>
                        
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
                                    onChange={(e) => setVariant(e.target.value as TextInputVariant)}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="outline" control={<Radio size="small" />} label="Outline" sx={{ m: 0 }} />
                                    <FormControlLabel value="filled" control={<Radio size="small" />} label="Filled" sx={{ m: 0 }} />
                                    <FormControlLabel value="underline" control={<Radio size="small" />} label="Underline" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            {/* Size Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Size</FormLabel>
                                <RadioGroup
                                    value={size}
                                    onChange={(e) => setSize(e.target.value as TextInputSize)}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="sm" control={<Radio size="small" />} label="Small" sx={{ m: 0 }} />
                                    <FormControlLabel value="md" control={<Radio size="small" />} label="Medium" sx={{ m: 0 }} />
                                    <FormControlLabel value="lg" control={<Radio size="small" />} label="Large" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            {/* States */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>States</FormLabel>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                                    <Switch
                                        checked={error}
                                        onChange={(checked) => setError(checked)}
                                        size="sm"
                                        variant="danger"
                                        label="Error"
                                        labelPosition="right"
                                    />
                                    <Switch
                                        checked={disabled}
                                        onChange={(checked) => setDisabled(checked)}
                                        size="sm"
                                        variant="default"
                                        label="Disabled"
                                        labelPosition="right"
                                    />
                                    <Switch
                                        checked={isRequired}
                                        onChange={(checked) => setIsRequired(checked)}
                                        size="sm"
                                        variant="warning"
                                        label="Required"
                                        labelPosition="right"
                                    />
                                </Box>
                            </FormControl>

                            {/* Display Options */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Display Options</FormLabel>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                                    <Switch
                                        checked={fullWidth}
                                        onChange={(checked) => setFullWidth(checked)}
                                        size="sm"
                                        variant="default"
                                        label="Full Width"
                                        labelPosition="right"
                                    />
                                    <Switch
                                        checked={multiline}
                                        onChange={(checked) => setMultiline(checked)}
                                        size="sm"
                                        variant="default"
                                        label="Multiline"
                                        labelPosition="right"
                                    />
                                    <Switch
                                        checked={showLabel}
                                        onChange={(checked) => setShowLabel(checked)}
                                        size="sm"
                                        variant="default"
                                        label="Show Label"
                                        labelPosition="right"
                                    />
                                </Box>
                            </FormControl>

                            {/* Label Text Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Label Text</FormLabel>
                                <TextField
                                    value={labelText}
                                    onChange={(e) => setLabelText(e.target.value)}
                                    size="small"
                                    placeholder="Enter label text..."
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>

                            {/* Placeholder Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Placeholder</FormLabel>
                                <TextField
                                    value={placeholder}
                                    onChange={(e) => setPlaceholder(e.target.value)}
                                    size="small"
                                    placeholder="Enter placeholder..."
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>

                            {/* Helper Text Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Helper Text</FormLabel>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                                    <Switch
                                        checked={showHelperText}
                                        onChange={(checked) => setShowHelperText(checked)}
                                        size="sm"
                                        variant="default"
                                        label="Show Helper"
                                        labelPosition="right"
                                    />
                                    <TextField
                                        value={helperText}
                                        onChange={(e) => setHelperText(e.target.value)}
                                        size="small"
                                        placeholder="Enter helper text..."
                                        sx={{ width: "100%" }}
                                        disabled={!showHelperText}
                                    />
                                </Box>
                            </FormControl>

                            {/* Start Adornment Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Start Adornment</FormLabel>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                                    <Switch
                                        checked={showStartAdornment}
                                        onChange={(checked) => setShowStartAdornment(checked)}
                                        size="sm"
                                        variant="default"
                                        label="Show Start"
                                        labelPosition="right"
                                    />
                                    <RadioGroup
                                        value={startAdornmentType}
                                        onChange={(e) => setStartAdornmentType(e.target.value as "icon" | "text")}
                                        sx={{ gap: 0.5 }}
                                    >
                                        <FormControlLabel value="icon" control={<Radio size="small" />} label="Icon" sx={{ m: 0 }} disabled={!showStartAdornment} />
                                        <FormControlLabel value="text" control={<Radio size="small" />} label="Text ($)" sx={{ m: 0 }} disabled={!showStartAdornment} />
                                    </RadioGroup>
                                </Box>
                            </FormControl>

                            {/* End Adornment Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>End Adornment</FormLabel>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                                    <Switch
                                        checked={showEndAdornment}
                                        onChange={(checked) => setShowEndAdornment(checked)}
                                        size="sm"
                                        variant="default"
                                        label="Show End"
                                        labelPosition="right"
                                    />
                                    <RadioGroup
                                        value={endAdornmentType}
                                        onChange={(e) => setEndAdornmentType(e.target.value as "icon" | "text")}
                                        sx={{ gap: 0.5 }}
                                    >
                                        <FormControlLabel value="icon" control={<Radio size="small" />} label="Button" sx={{ m: 0 }} disabled={!showEndAdornment} />
                                        <FormControlLabel value="text" control={<Radio size="small" />} label="Text (.com)" sx={{ m: 0 }} disabled={!showEndAdornment} />
                                    </RadioGroup>
                                </Box>
                            </FormControl>
                        </Box>
                    </Box>

                    {/* Live Preview */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Live Preview</Typography>
                        
                        <Box sx={{ maxWidth: fullWidth ? "100%" : 400, mx: "auto" }}>
                            <TailwindTextInputBase
                                variant={variant}
                                size={size}
                                error={error}
                                disabled={disabled}
                                fullWidth={fullWidth}
                                isRequired={isRequired}
                                multiline={multiline}
                                label={showLabel ? labelText : undefined}
                                placeholder={placeholder}
                                helperText={showHelperText ? helperText : undefined}
                                value={inputValues.preview}
                                onChange={(e) => handleInputChange("preview", e.target.value)}
                                rows={multiline ? 4 : undefined}
                                startAdornment={getStartAdornment()}
                                endAdornment={getEndAdornment()}
                            />
                        </Box>
                    </Box>

                    {/* All Variants Display */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>All Variants</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { 
                                xs: "1fr", 
                                sm: "repeat(2, 1fr)",
                                md: "repeat(3, 1fr)",
                            }, 
                            gap: 3, 
                        }}>
                            {variants.map(v => (
                                <Box key={v} sx={{ textAlign: "center" }}>
                                    <Typography 
                                        variant="subtitle2" 
                                        sx={{ mb: 2, textTransform: "capitalize" }}
                                    >
                                        {v}
                                    </Typography>
                                    <TailwindTextInputBase
                                        variant={v}
                                        size="md"
                                        label={`${v} input`}
                                        placeholder={`${v} placeholder...`}
                                        value={inputValues[v]}
                                        onChange={(e) => handleInputChange(v, e.target.value)}
                                        fullWidth
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
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Size Comparison</Typography>
                        
                        <Box sx={{ 
                            display: "flex", 
                            flexDirection: "column",
                            gap: 3,
                            alignItems: "center",
                        }}>
                            {sizes.map(s => (
                                <Box key={s} sx={{ textAlign: "center", width: "100%", maxWidth: 400 }}>
                                    <Typography variant="subtitle2" sx={{ mb: 1 }}>
                                        Size: {s.toUpperCase()}
                                    </Typography>
                                    <TailwindTextInputBase
                                        variant="outline"
                                        size={s}
                                        label={`${s.toUpperCase()} size input`}
                                        placeholder={`${s} size placeholder...`}
                                        value={inputValues[`size-${s}`]}
                                        onChange={(e) => handleInputChange(`size-${s}`, e.target.value)}
                                        fullWidth
                                    />
                                </Box>
                            ))}
                        </Box>
                    </Box>

                    {/* Multiline Examples */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Multiline Examples</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)" }, 
                            gap: 3, 
                        }}>
                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 2 }}>
                                    Standard Textarea
                                </Typography>
                                <TailwindTextInputBase
                                    variant="outline"
                                    multiline
                                    label="Message"
                                    placeholder="Enter your message..."
                                    value={inputValues.multilineExample}
                                    onChange={(e) => handleInputChange("multilineExample", e.target.value)}
                                    rows={4}
                                    fullWidth
                                />
                            </Box>
                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 2 }}>
                                    Error State Textarea
                                </Typography>
                                <TailwindTextInputBase
                                    variant="filled"
                                    multiline
                                    error
                                    label="Feedback"
                                    placeholder="Please provide feedback..."
                                    helperText="This field is required"
                                    rows={4}
                                    fullWidth
                                />
                            </Box>
                        </Box>
                    </Box>

                    {/* Adornment Examples */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Adornment Examples</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)" }, 
                            gap: 3, 
                        }}>
                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 2 }}>
                                    Start Adornments
                                </Typography>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                                    <TailwindTextInputBase
                                        variant="filled"
                                        label="Search"
                                        placeholder="Search for anything..."
                                        startAdornment={<SearchIcon />}
                                        fullWidth
                                    />
                                    <TailwindTextInputBase
                                        variant="filled"
                                        label="User Account"
                                        placeholder="Enter username..."
                                        startAdornment={<UserIcon />}
                                        fullWidth
                                    />
                                    <TailwindTextInputBase
                                        variant="filled"
                                        label="Price"
                                        placeholder="0.00"
                                        startAdornment={<span className="tw-text-sm tw-font-medium">$</span>}
                                        fullWidth
                                    />
                                </Box>
                            </Box>
                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 2 }}>
                                    End Adornments
                                </Typography>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                                    <TailwindTextInputBase
                                        variant="filled"
                                        label="Password"
                                        type="password"
                                        placeholder="Enter password..."
                                        endAdornment={
                                            <button 
                                                type="button"
                                                className="tw-flex tw-items-center tw-justify-center hover:tw-text-primary-main tw-transition-colors tw-cursor-pointer tw-p-0 tw-m-0 tw-border-0 tw-bg-transparent tw-text-current"
                                            >
                                                <EyeIcon />
                                            </button>
                                        }
                                        fullWidth
                                    />
                                    <TailwindTextInputBase
                                        variant="filled"
                                        label="Message"
                                        placeholder="Type your message..."
                                        endAdornment={
                                            <button 
                                                type="button"
                                                className="tw-flex tw-items-center tw-justify-center hover:tw-text-primary-main tw-transition-colors tw-cursor-pointer tw-p-0 tw-m-0 tw-border-0 tw-bg-transparent tw-text-current"
                                            >
                                                <SendIcon />
                                            </button>
                                        }
                                        fullWidth
                                    />
                                    <TailwindTextInputBase
                                        variant="filled"
                                        label="Website"
                                        placeholder="mysite"
                                        endAdornment={<span className="tw-text-sm tw-font-medium">.com</span>}
                                        fullWidth
                                    />
                                </Box>
                            </Box>
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
                        
                        <Box sx={{ display: "flex", flexDirection: "column", gap: 4 }}>
                            {/* Contact Form */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Contact Form</Typography>
                                <Box sx={{ display: "grid", gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)" }, gap: 2, p: 2, bgcolor: "grey.50", borderRadius: 1 }}>
                                    <TailwindTextInputBase
                                        variant="outline"
                                        size="md"
                                        label="First Name"
                                        placeholder="Enter first name"
                                        isRequired
                                        value={inputValues.form1}
                                        onChange={(e) => handleInputChange("form1", e.target.value)}
                                    />
                                    <TailwindTextInputBase
                                        variant="outline"
                                        size="md"
                                        label="Last Name"
                                        placeholder="Enter last name"
                                        isRequired
                                        value={inputValues.form2}
                                        onChange={(e) => handleInputChange("form2", e.target.value)}
                                    />
                                    <TailwindTextInputBase
                                        variant="outline"
                                        size="md"
                                        type="email"
                                        label="Email"
                                        placeholder="your@email.com"
                                        isRequired
                                        value={inputValues.form3}
                                        onChange={(e) => handleInputChange("form3", e.target.value)}
                                        helperText="We'll never share your email"
                                    />
                                    <TailwindTextInputBase
                                        variant="outline"
                                        size="md"
                                        type="tel"
                                        label="Phone"
                                        placeholder="(555) 123-4567"
                                        value={inputValues.form4}
                                        onChange={(e) => handleInputChange("form4", e.target.value)}
                                    />
                                </Box>
                            </Box>

                            {/* Different States */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Different States</Typography>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 2, p: 2, bgcolor: "grey.50", borderRadius: 1 }}>
                                    <TailwindTextInputBase
                                        variant="outline"
                                        size="md"
                                        label="Normal Input"
                                        placeholder="Type here..."
                                        fullWidth
                                    />
                                    <TailwindTextInputBase
                                        variant="outline"
                                        size="md"
                                        label="Error Input"
                                        placeholder="Invalid value"
                                        error
                                        helperText="Please enter a valid value"
                                        fullWidth
                                    />
                                    <TailwindTextInputBase
                                        variant="outline"
                                        size="md"
                                        label="Disabled Input"
                                        placeholder="Cannot edit"
                                        disabled
                                        value="Disabled value"
                                        fullWidth
                                    />
                                    <TailwindTextInputBase
                                        variant="outline"
                                        size="md"
                                        label="Required Input"
                                        placeholder="This field is required"
                                        isRequired
                                        helperText="This field must be filled out"
                                        fullWidth
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
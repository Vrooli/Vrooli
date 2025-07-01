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
import Slider from "@mui/material/Slider";
import { Formik, Form } from "formik";
import { IntegerInput } from "./IntegerInput.js";
import { Switch } from "../Switch/Switch.js";

const meta: Meta<typeof IntegerInput> = {
    title: "Components/Inputs/IntegerInput",
    component: IntegerInput,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive IntegerInput Playground
export const IntegerInputShowcase: Story = {
    render: () => {
        const [allowDecimal, setAllowDecimal] = useState(false);
        const [autoFocus, setAutoFocus] = useState(false);
        const [fullWidth, setFullWidth] = useState(false);
        const [disabled, setDisabled] = useState(false);
        const [label, setLabel] = useState("Number");
        const [min, setMin] = useState(0);
        const [max, setMax] = useState(100);
        const [step, setStep] = useState(1);
        const [offset, setOffset] = useState(0);
        const [initial, setInitial] = useState(0);
        const [zeroText, setZeroText] = useState("");
        const [tooltip, setTooltip] = useState("");

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
                        <Typography variant="h5" sx={{ mb: 3 }}>IntegerInput Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)", lg: "repeat(4, 1fr)" },
                            gap: 3, 
                        }}>
                            {/* Label Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Label</FormLabel>
                                <TextField
                                    value={label}
                                    onChange={(e) => setLabel(e.target.value)}
                                    size="small"
                                    placeholder="Enter label text..."
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>

                            {/* Min Value Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Min Value</FormLabel>
                                <TextField
                                    type="number"
                                    value={min}
                                    onChange={(e) => setMin(Number(e.target.value))}
                                    size="small"
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>

                            {/* Max Value Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Max Value</FormLabel>
                                <TextField
                                    type="number"
                                    value={max}
                                    onChange={(e) => setMax(Number(e.target.value))}
                                    size="small"
                                    sx={{ width: "100%" }}
                                />
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
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>

                            {/* Offset Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Offset</FormLabel>
                                <TextField
                                    type="number"
                                    value={offset}
                                    onChange={(e) => setOffset(Number(e.target.value))}
                                    size="small"
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>

                            {/* Initial Value Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Initial Value</FormLabel>
                                <TextField
                                    type="number"
                                    value={initial}
                                    onChange={(e) => setInitial(Number(e.target.value))}
                                    size="small"
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>

                            {/* Zero Text Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Zero Text</FormLabel>
                                <TextField
                                    value={zeroText}
                                    onChange={(e) => setZeroText(e.target.value)}
                                    size="small"
                                    placeholder="e.g., 'None', 'Unlimited'"
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>

                            {/* Tooltip Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Tooltip</FormLabel>
                                <TextField
                                    value={tooltip}
                                    onChange={(e) => setTooltip(e.target.value)}
                                    size="small"
                                    placeholder="Helpful tooltip text"
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>
                        </Box>

                        {/* Boolean Controls */}
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(4, 1fr)" },
                            gap: 3, 
                            mt: 3,
                        }}>
                            <Switch
                                checked={allowDecimal}
                                onChange={(checked) => setAllowDecimal(checked)}
                                size="sm"
                                label="Allow Decimal"
                                labelPosition="right"
                            />

                            <Switch
                                checked={autoFocus}
                                onChange={(checked) => setAutoFocus(checked)}
                                size="sm"
                                label="Auto Focus"
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
                                checked={disabled}
                                onChange={(checked) => setDisabled(checked)}
                                size="sm"
                                label="Disabled"
                                labelPosition="right"
                            />
                        </Box>
                    </Box>

                    {/* Live Demo */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Live Demo</Typography>
                        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                            Interactive number input with configurable constraints and formatting.
                            {offset !== 0 && ` The displayed value includes an offset of ${offset}.`}
                        </Typography>
                        
                        <Formik
                            initialValues={{
                                number: initial,
                            }}
                            onSubmit={(values) => {
                                console.log("Form submitted:", values);
                            }}
                        >
                            <Form>
                                <Box sx={{ maxWidth: 400 }}>
                                    <IntegerInput
                                        name="number"
                                        label={label}
                                        min={min}
                                        max={max}
                                        step={step}
                                        offset={offset}
                                        initial={initial}
                                        zeroText={zeroText || undefined}
                                        tooltip={tooltip}
                                        allowDecimal={allowDecimal}
                                        autoFocus={autoFocus}
                                        fullWidth={fullWidth}
                                        disabled={disabled}
                                    />
                                    <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: "block" }}>
                                        Range: {min} to {max} (step: {step})
                                        {zeroText && `, Zero displays as: "${zeroText}"`}
                                    </Typography>
                                </Box>
                            </Form>
                        </Formik>
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
                        
                        <Formik
                            initialValues={{
                                quantity: 1,
                                percentage: 50,
                                age: 25,
                                price: 19.99,
                                rating: 4,
                                unlimited: 0,
                            }}
                            onSubmit={(values) => {
                                console.log("Use cases submitted:", values);
                            }}
                        >
                            <Form>
                                <Box sx={{ 
                                    display: "grid", 
                                    gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)", lg: "repeat(3, 1fr)" },
                                    gap: 3, 
                                }}>
                                    {/* Quantity */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Quantity Input
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Basic integer input with min/max
                                        </Typography>
                                        <IntegerInput
                                            name="quantity"
                                            label="Quantity"
                                            min={1}
                                            max={999}
                                            initial={1}
                                            tooltip="Number of items to order"
                                            fullWidth
                                        />
                                    </Box>

                                    {/* Percentage */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Percentage
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            0-100 range with step control
                                        </Typography>
                                        <IntegerInput
                                            name="percentage"
                                            label="Percentage"
                                            min={0}
                                            max={100}
                                            step={5}
                                            initial={50}
                                            tooltip="Percentage value (0-100%)"
                                            fullWidth
                                        />
                                    </Box>

                                    {/* Age */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Age Input
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Human age with realistic bounds
                                        </Typography>
                                        <IntegerInput
                                            name="age"
                                            label="Age"
                                            min={0}
                                            max={120}
                                            initial={25}
                                            tooltip="Age in years"
                                            fullWidth
                                        />
                                    </Box>

                                    {/* Price with decimals */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Price (Decimals)
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Decimal input for currency
                                        </Typography>
                                        <IntegerInput
                                            name="price"
                                            label="Price ($)"
                                            min={0}
                                            max={9999.99}
                                            step={0.01}
                                            initial={19.99}
                                            allowDecimal={true}
                                            tooltip="Price in dollars"
                                            fullWidth
                                        />
                                    </Box>

                                    {/* Rating */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Star Rating
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            1-5 star rating system
                                        </Typography>
                                        <IntegerInput
                                            name="rating"
                                            label="Rating"
                                            min={1}
                                            max={5}
                                            initial={4}
                                            tooltip="Rate from 1 to 5 stars"
                                            fullWidth
                                        />
                                    </Box>

                                    {/* Unlimited with zero text */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Limit with "Unlimited"
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Zero displays as "Unlimited"
                                        </Typography>
                                        <IntegerInput
                                            name="unlimited"
                                            label="Download Limit"
                                            min={0}
                                            max={1000}
                                            step={10}
                                            initial={0}
                                            zeroText="Unlimited"
                                            tooltip="0 = unlimited downloads"
                                            fullWidth
                                        />
                                    </Box>
                                </Box>
                            </Form>
                        </Formik>
                    </Box>

                    {/* Offset Examples */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Offset Examples</Typography>
                        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                            The offset prop adjusts the displayed value while keeping the form value unchanged.
                        </Typography>
                        
                        <Formik
                            initialValues={{
                                arrayIndex: 0,
                                currentYear: new Date().getFullYear() - 2024,
                                temperature: 0,
                            }}
                            onSubmit={(values) => {
                                console.log("Offset examples submitted:", values);
                            }}
                        >
                            <Form>
                                <Box sx={{ 
                                    display: "grid", 
                                    gridTemplateColumns: { xs: "1fr", md: "repeat(3, 1fr)" },
                                    gap: 3, 
                                }}>
                                    {/* Array Index (0-based to 1-based) */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Array Index
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            0-based index shown as 1-based
                                        </Typography>
                                        <IntegerInput
                                            name="arrayIndex"
                                            label="Position"
                                            min={0}
                                            max={9}
                                            offset={1}
                                            initial={0}
                                            tooltip="Internal value is 0-based, display is 1-based"
                                            fullWidth
                                        />
                                        <Typography variant="caption" color="text.secondary">
                                            Form value: 0-9, Display: 1-10
                                        </Typography>
                                    </Box>

                                    {/* Year offset */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Year Selector
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Relative years shown as absolute
                                        </Typography>
                                        <IntegerInput
                                            name="currentYear"
                                            label="Year"
                                            min={-5}
                                            max={5}
                                            offset={2024}
                                            initial={0}
                                            tooltip="Relative to 2024"
                                            fullWidth
                                        />
                                        <Typography variant="caption" color="text.secondary">
                                            Form value: -5 to 5, Display: 2019-2029
                                        </Typography>
                                    </Box>

                                    {/* Temperature offset */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Temperature
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Celsius stored, Fahrenheit displayed
                                        </Typography>
                                        <IntegerInput
                                            name="temperature"
                                            label="Temperature (°F)"
                                            min={-40}
                                            max={50}
                                            offset={32}
                                            initial={0}
                                            tooltip="Stored in Celsius, displayed in Fahrenheit"
                                            fullWidth
                                        />
                                        <Typography variant="caption" color="text.secondary">
                                            Form value: -40°C to 50°C, Display: -8°F to 122°F
                                        </Typography>
                                    </Box>
                                </Box>
                            </Form>
                        </Formik>
                    </Box>

                    {/* States Showcase */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Different States</Typography>
                        
                        <Formik
                            initialValues={{
                                normalState: 42,
                                disabledState: 100,
                                errorState: 150,
                            }}
                            validate={(values) => {
                                const errors: any = {};
                                if (values.errorState > 100) {
                                    errors.errorState = "Value must be 100 or less";
                                }
                                return errors;
                            }}
                            onSubmit={(values) => {
                                console.log("States demo submitted:", values);
                            }}
                        >
                            <Form>
                                <Box sx={{ 
                                    display: "grid", 
                                    gridTemplateColumns: { xs: "1fr", md: "repeat(3, 1fr)" },
                                    gap: 3, 
                                }}>
                                    {/* Normal State */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Normal State
                                        </Typography>
                                        <IntegerInput
                                            name="normalState"
                                            label="Normal Number"
                                            min={0}
                                            max={100}
                                            fullWidth
                                        />
                                    </Box>

                                    {/* Disabled State */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Disabled State
                                        </Typography>
                                        <IntegerInput
                                            name="disabledState"
                                            label="Disabled Number"
                                            min={0}
                                            max={100}
                                            disabled
                                            fullWidth
                                        />
                                    </Box>

                                    {/* Error State */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Error State
                                        </Typography>
                                        <IntegerInput
                                            name="errorState"
                                            label="Number with Error"
                                            min={0}
                                            max={100}
                                            fullWidth
                                        />
                                    </Box>
                                </Box>
                            </Form>
                        </Formik>
                    </Box>

                    {/* Features */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Features</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)" },
                            gap: 2, 
                        }}>
                            <Box>
                                <Typography variant="subtitle2" sx={{ fontWeight: "bold", mb: 1 }}>
                                    🔢 Range Validation
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Automatically enforces minimum and maximum values with visual feedback.
                                </Typography>
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ fontWeight: "bold", mb: 1 }}>
                                    📐 Decimal Support
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Toggle between integer-only and decimal number input modes.
                                </Typography>
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ fontWeight: "bold", mb: 1 }}>
                                    ⚡ Step Control
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Configure increment/decrement step size for keyboard and scroll input.
                                </Typography>
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ fontWeight: "bold", mb: 1 }}>
                                    🏷️ Custom Zero Text
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Display custom text when value is zero (e.g., "Unlimited", "None").
                                </Typography>
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ fontWeight: "bold", mb: 1 }}>
                                    🔄 Value Offset
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Display values with an offset while keeping form values unchanged.
                                </Typography>
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ fontWeight: "bold", mb: 1 }}>
                                    🎨 Visual Feedback
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Color-coded labels indicate when values are at boundaries or out of range.
                                </Typography>
                            </Box>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};

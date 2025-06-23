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
import { Formik, Form } from "formik";
import { PasswordTextInput } from "./PasswordTextInput.js";
import { Switch } from "../Switch/Switch.js";

const meta: Meta<typeof PasswordTextInput> = {
    title: "Components/Inputs/PasswordTextInput",
    component: PasswordTextInput,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive PasswordTextInput Playground
export const PasswordTextInputShowcase: Story = {
    render: () => {
        const [autoComplete, setAutoComplete] = useState<string>("current-password");
        const [autoFocus, setAutoFocus] = useState(false);
        const [fullWidth, setFullWidth] = useState(true);
        const [disabled, setDisabled] = useState(false);
        const [label, setLabel] = useState("Password");

        const autoCompleteOptions = [
            "current-password",
            "new-password", 
            "off"
        ];

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
                        <Typography variant="h5" sx={{ mb: 3 }}>PasswordTextInput Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)", lg: "repeat(5, 1fr)" },
                            gap: 3, 
                        }}>
                            {/* AutoComplete Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>AutoComplete</FormLabel>
                                <RadioGroup
                                    value={autoComplete}
                                    onChange={(e) => setAutoComplete(e.target.value)}
                                    sx={{ gap: 0.5 }}
                                >
                                    {autoCompleteOptions.map(option => (
                                        <FormControlLabel 
                                            key={option}
                                            value={option} 
                                            control={<Radio size="small" />} 
                                            label={option} 
                                            sx={{ m: 0 }} 
                                        />
                                    ))}
                                </RadioGroup>
                            </FormControl>

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

                            {/* AutoFocus Control */}
                            <FormControl component="fieldset" size="small">
                                <Switch
                                    checked={autoFocus}
                                    onChange={(checked) => setAutoFocus(checked)}
                                    size="sm"
                                    label="Auto Focus"
                                    labelPosition="right"
                                />
                            </FormControl>

                            {/* Full Width Control */}
                            <FormControl component="fieldset" size="small">
                                <Switch
                                    checked={fullWidth}
                                    onChange={(checked) => setFullWidth(checked)}
                                    size="sm"
                                    label="Full Width"
                                    labelPosition="right"
                                />
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
                        </Box>
                    </Box>

                    {/* Password Input Examples */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Password Input Examples</Typography>
                        
                        <Formik
                            initialValues={{
                                currentPassword: "",
                                newPassword: "",
                                confirmPassword: "",
                            }}
                            onSubmit={(values) => {
                                console.log("Form submitted:", values);
                            }}
                        >
                            <Form>
                                <Box sx={{ 
                                    display: "flex", 
                                    flexDirection: "column",
                                    gap: 3, 
                                }}>
                                    {/* Current Password */}
                                    <Box>
                                        <Typography variant="h6" sx={{ mb: 2 }}>Current Password (with strength indicator disabled)</Typography>
                                        <PasswordTextInput
                                            name="currentPassword"
                                            label={label || "Current Password"}
                                            autoComplete="current-password"
                                            autoFocus={autoFocus}
                                            fullWidth={fullWidth}
                                            disabled={disabled}
                                        />
                                    </Box>

                                    {/* New Password */}
                                    <Box>
                                        <Typography variant="h6" sx={{ mb: 2 }}>New Password (with strength indicator)</Typography>
                                        <PasswordTextInput
                                            name="newPassword"
                                            label={label || "New Password"}
                                            autoComplete="new-password"
                                            autoFocus={false}
                                            fullWidth={fullWidth}
                                            disabled={disabled}
                                        />
                                    </Box>

                                    {/* Confirm Password */}
                                    <Box>
                                        <Typography variant="h6" sx={{ mb: 2 }}>Confirm Password</Typography>
                                        <PasswordTextInput
                                            name="confirmPassword"
                                            label={label || "Confirm Password"}
                                            autoComplete="new-password"
                                            autoFocus={false}
                                            fullWidth={fullWidth}
                                            disabled={disabled}
                                        />
                                    </Box>
                                </Box>
                            </Form>
                        </Formik>
                    </Box>

                    {/* AutoComplete Comparison */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>AutoComplete Behavior Comparison</Typography>
                        
                        <Formik
                            initialValues={{
                                currentPasswordDemo: "",
                                newPasswordDemo: "",
                                offPasswordDemo: "",
                            }}
                            onSubmit={(values) => {
                                console.log("AutoComplete demo submitted:", values);
                            }}
                        >
                            <Form>
                                <Box sx={{ 
                                    display: "grid", 
                                    gridTemplateColumns: { xs: "1fr", md: "repeat(3, 1fr)" },
                                    gap: 3, 
                                }}>
                                    {/* Current Password */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            current-password
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            For existing password fields. No strength indicator.
                                        </Typography>
                                        <PasswordTextInput
                                            name="currentPasswordDemo"
                                            label="Current Password"
                                            autoComplete="current-password"
                                            fullWidth
                                        />
                                    </Box>

                                    {/* New Password */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            new-password
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            For new passwords. Shows strength indicator.
                                        </Typography>
                                        <PasswordTextInput
                                            name="newPasswordDemo"
                                            label="New Password"
                                            autoComplete="new-password"
                                            fullWidth
                                        />
                                    </Box>

                                    {/* Off */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            off
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Disables browser autocomplete. No strength indicator.
                                        </Typography>
                                        <PasswordTextInput
                                            name="offPasswordDemo"
                                            label="Password (No AutoComplete)"
                                            autoComplete="off"
                                            fullWidth
                                        />
                                    </Box>
                                </Box>
                            </Form>
                        </Formik>
                    </Box>

                    {/* Password Strength Demo */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Password Strength Indicator Demo</Typography>
                        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                            Try entering different passwords to see the strength indicator in action. 
                            The strength meter only appears for inputs with autoComplete="new-password".
                        </Typography>
                        
                        <Formik
                            initialValues={{
                                strengthDemo: "",
                            }}
                            onSubmit={(values) => {
                                console.log("Strength demo submitted:", values);
                            }}
                        >
                            <Form>
                                <Box sx={{ maxWidth: 400 }}>
                                    <PasswordTextInput
                                        name="strengthDemo"
                                        label="Try different passwords"
                                        autoComplete="new-password"
                                        fullWidth
                                    />
                                    <Box sx={{ mt: 2 }}>
                                        <Typography variant="caption" color="text.secondary">
                                            Suggestions to test:<br/>
                                            • "123" (Weak)<br/>
                                            • "password123" (Moderate)<br/>
                                            • "MySecure123!" (Strong)<br/>
                                            • "MyVerySecureP@ssw0rd2024!" (Very Strong)
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
                                normalState: "",
                                disabledState: "",
                                errorState: "invalid",
                            }}
                            validate={(values) => {
                                const errors: any = {};
                                if (values.errorState && values.errorState.length < 8) {
                                    errors.errorState = "Password must be at least 8 characters long";
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
                                        <PasswordTextInput
                                            name="normalState"
                                            label="Normal Password"
                                            autoComplete="new-password"
                                            fullWidth
                                        />
                                    </Box>

                                    {/* Disabled State */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Disabled State
                                        </Typography>
                                        <PasswordTextInput
                                            name="disabledState"
                                            label="Disabled Password"
                                            autoComplete="current-password"
                                            disabled
                                            fullWidth
                                        />
                                    </Box>

                                    {/* Error State */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Error State
                                        </Typography>
                                        <PasswordTextInput
                                            name="errorState"
                                            label="Password with Error"
                                            autoComplete="new-password"
                                            fullWidth
                                        />
                                    </Box>
                                </Box>
                            </Form>
                        </Formik>
                    </Box>
                </Box>
            </Box>
        );
    },
};
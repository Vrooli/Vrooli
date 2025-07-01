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
import { PhoneNumberInput } from "./PhoneNumberInput.js";
import { Switch } from "../Switch/Switch.js";

const meta: Meta<typeof PhoneNumberInput> = {
    title: "Components/Inputs/PhoneNumberInput",
    component: PhoneNumberInput,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive PhoneNumberInput Playground
export const PhoneNumberInputShowcase: Story = {
    render: () => {
        const [autoComplete, setAutoComplete] = useState<string>("tel");
        const [autoFocus, setAutoFocus] = useState(false);
        const [fullWidth, setFullWidth] = useState(true);
        const [disabled, setDisabled] = useState(false);
        const [label, setLabel] = useState("Phone Number");

        const autoCompleteOptions = [
            "tel",
            "tel-national",
            "tel-country-code",
            "tel-local",
            "off",
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
                        <Typography variant="h5" sx={{ mb: 3 }}>PhoneNumberInput Controls</Typography>
                        
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
                            Click the country code prefix to change countries. The phone number will be automatically formatted as you type.
                        </Typography>
                        
                        <Formik
                            initialValues={{
                                phoneNumber: "",
                            }}
                            onSubmit={(values) => {
                                console.log("Form submitted:", values);
                            }}
                        >
                            <Form>
                                <Box sx={{ maxWidth: 400 }}>
                                    <PhoneNumberInput
                                        name="phoneNumber"
                                        label={label}
                                        autoComplete={autoComplete}
                                        autoFocus={autoFocus}
                                        fullWidth={fullWidth}
                                        disabled={disabled}
                                    />
                                </Box>
                            </Form>
                        </Formik>
                    </Box>

                    {/* AutoComplete Types Demo */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>AutoComplete Types</Typography>
                        
                        <Formik
                            initialValues={{
                                telDemo: "",
                                telNationalDemo: "",
                                telCountryCodeDemo: "",
                                telLocalDemo: "",
                                offDemo: "",
                            }}
                            onSubmit={(values) => {
                                console.log("AutoComplete demo submitted:", values);
                            }}
                        >
                            <Form>
                                <Box sx={{ 
                                    display: "grid", 
                                    gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)", lg: "repeat(3, 1fr)" },
                                    gap: 3, 
                                }}>
                                    {/* tel */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            tel (default)
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            General telephone autocomplete
                                        </Typography>
                                        <PhoneNumberInput
                                            name="telDemo"
                                            label="Phone (tel)"
                                            autoComplete="tel"
                                            fullWidth
                                        />
                                    </Box>

                                    {/* tel-national */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            tel-national
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            National format telephone
                                        </Typography>
                                        <PhoneNumberInput
                                            name="telNationalDemo"
                                            label="Phone (national)"
                                            autoComplete="tel-national"
                                            fullWidth
                                        />
                                    </Box>

                                    {/* tel-country-code */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            tel-country-code
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Country code autocomplete
                                        </Typography>
                                        <PhoneNumberInput
                                            name="telCountryCodeDemo"
                                            label="Phone (country)"
                                            autoComplete="tel-country-code"
                                            fullWidth
                                        />
                                    </Box>

                                    {/* tel-local */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            tel-local
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Local telephone number
                                        </Typography>
                                        <PhoneNumberInput
                                            name="telLocalDemo"
                                            label="Phone (local)"
                                            autoComplete="tel-local"
                                            fullWidth
                                        />
                                    </Box>

                                    {/* off */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            off
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                            Disables autocomplete
                                        </Typography>
                                        <PhoneNumberInput
                                            name="offDemo"
                                            label="Phone (no autocomplete)"
                                            autoComplete="off"
                                            fullWidth
                                        />
                                    </Box>
                                </Box>
                            </Form>
                        </Formik>
                    </Box>

                    {/* Example Phone Numbers */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Pre-filled Examples</Typography>
                        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                            These inputs show how the component handles different phone number formats and countries.
                        </Typography>
                        
                        <Formik
                            initialValues={{
                                usPhone: "+1 (555) 123-4567",
                                ukPhone: "+44 20 7946 0958",
                                frPhone: "+33 1 23 45 67 89",
                                jpPhone: "+81 3-1234-5678",
                                auPhone: "+61 2 1234 5678",
                                dePhone: "+49 30 12345678",
                            }}
                            onSubmit={(values) => {
                                console.log("Examples submitted:", values);
                            }}
                        >
                            <Form>
                                <Box sx={{ 
                                    display: "grid", 
                                    gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)", lg: "repeat(3, 1fr)" },
                                    gap: 3, 
                                }}>
                                    <Box>
                                        <Typography variant="subtitle2" sx={{ mb: 1 }}>US Phone</Typography>
                                        <PhoneNumberInput
                                            name="usPhone"
                                            label="US Number"
                                            fullWidth
                                        />
                                    </Box>

                                    <Box>
                                        <Typography variant="subtitle2" sx={{ mb: 1 }}>UK Phone</Typography>
                                        <PhoneNumberInput
                                            name="ukPhone"
                                            label="UK Number"
                                            fullWidth
                                        />
                                    </Box>

                                    <Box>
                                        <Typography variant="subtitle2" sx={{ mb: 1 }}>French Phone</Typography>
                                        <PhoneNumberInput
                                            name="frPhone"
                                            label="FR Number"
                                            fullWidth
                                        />
                                    </Box>

                                    <Box>
                                        <Typography variant="subtitle2" sx={{ mb: 1 }}>Japanese Phone</Typography>
                                        <PhoneNumberInput
                                            name="jpPhone"
                                            label="JP Number"
                                            fullWidth
                                        />
                                    </Box>

                                    <Box>
                                        <Typography variant="subtitle2" sx={{ mb: 1 }}>Australian Phone</Typography>
                                        <PhoneNumberInput
                                            name="auPhone"
                                            label="AU Number"
                                            fullWidth
                                        />
                                    </Box>

                                    <Box>
                                        <Typography variant="subtitle2" sx={{ mb: 1 }}>German Phone</Typography>
                                        <PhoneNumberInput
                                            name="dePhone"
                                            label="DE Number"
                                            fullWidth
                                        />
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
                                disabledState: "+1 (555) 123-4567",
                                errorState: "invalid",
                            }}
                            validate={(values) => {
                                const errors: any = {};
                                if (values.errorState && !values.errorState.startsWith("+")) {
                                    errors.errorState = "Phone number must include country code";
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
                                        <PhoneNumberInput
                                            name="normalState"
                                            label="Normal Phone"
                                            fullWidth
                                        />
                                    </Box>

                                    {/* Disabled State */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Disabled State
                                        </Typography>
                                        <PhoneNumberInput
                                            name="disabledState"
                                            label="Disabled Phone"
                                            disabled
                                            fullWidth
                                        />
                                    </Box>

                                    {/* Error State */}
                                    <Box>
                                        <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: "bold" }}>
                                            Error State
                                        </Typography>
                                        <PhoneNumberInput
                                            name="errorState"
                                            label="Phone with Error"
                                            fullWidth
                                        />
                                    </Box>
                                </Box>
                            </Form>
                        </Formik>
                    </Box>

                    {/* Feature Highlights */}
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
                                    🌍 Country Selection
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Click the country code prefix to open a searchable country selector with calling codes.
                                </Typography>
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ fontWeight: "bold", mb: 1 }}>
                                    📱 Auto-formatting
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Phone numbers are automatically formatted according to the selected country's format.
                                </Typography>
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ fontWeight: "bold", mb: 1 }}>
                                    ✅ Validation
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Real-time validation ensures phone numbers are valid for the selected country.
                                </Typography>
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ fontWeight: "bold", mb: 1 }}>
                                    🔍 Search Countries
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Search countries by name or calling code in the country selection popup.
                                </Typography>
                            </Box>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};

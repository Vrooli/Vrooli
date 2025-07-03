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
import { DateInput } from "./DateInput.js";
import { Switch } from "../Switch/Switch.js";

const meta: Meta<typeof DateInput> = {
    title: "Components/Inputs/DateInput",
    component: DateInput,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive DateInput Playground
export const DateInputShowcase: Story = {
    render: () => {
        const [inputType, setInputType] = useState<"date" | "datetime-local">("datetime-local");
        const [isRequired, setIsRequired] = useState(false);
        const [label, setLabel] = useState("Select Date & Time");

        const typeOptions: Array<"date" | "datetime-local"> = ["date", "datetime-local"];

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
                        <Typography variant="h5" sx={{ mb: 3 }}>DateInput Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(4, 1fr)" },
                            gap: 3, 
                        }}>
                            {/* Type Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Input Type</FormLabel>
                                <RadioGroup
                                    value={inputType}
                                    onChange={(e) => setInputType(e.target.value as "date" | "datetime-local")}
                                    sx={{ gap: 0.5 }}
                                >
                                    {typeOptions.map(option => (
                                        <FormControlLabel 
                                            key={option}
                                            value={option} 
                                            control={<Radio size="small" />} 
                                            label={option === "date" ? "Date Only" : "Date & Time"} 
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

                            {/* Required Control */}
                            <FormControl component="fieldset" size="small">
                                <Switch
                                    checked={isRequired}
                                    onChange={(checked) => setIsRequired(checked)}
                                    size="sm"
                                    label="Required"
                                    labelPosition="right"
                                />
                            </FormControl>
                        </Box>
                    </Box>

                    {/* DateInput Examples */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>DateInput Examples</Typography>
                        
                        <Formik
                            initialValues={{
                                currentDateTime: new Date().toISOString(),
                                eventDate: "",
                                deadline: "",
                                birthday: "",
                                appointment: "",
                            }}
                            onSubmit={(values) => {
                                console.log("Form submitted:", values);
                            }}
                        >
                            {({ values }) => (
                                <Form>
                                    <Box sx={{ 
                                        display: "grid", 
                                        gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)" },
                                        gap: 3, 
                                    }}>
                                        {/* Live Preview */}
                                        <Box>
                                            <Typography variant="h6" sx={{ mb: 2 }}>Live Preview</Typography>
                                            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                                Type: <strong>{inputType === "date" ? "Date Only" : "Date & Time"}</strong><br/>
                                                Required: <strong>{isRequired ? "Yes" : "No"}</strong>
                                            </Typography>
                                            <DateInput
                                                name="currentDateTime"
                                                label={label}
                                                type={inputType}
                                                isRequired={isRequired}
                                            />
                                            <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: "block" }}>
                                                Current value: {values.currentDateTime || "Empty"}
                                            </Typography>
                                        </Box>

                                        {/* Date Only Example */}
                                        <Box>
                                            <Typography variant="h6" sx={{ mb: 2 }}>Date Only Input</Typography>
                                            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                                Perfect for birthdays, deadlines, or events that don't require specific times.
                                            </Typography>
                                            <DateInput
                                                name="eventDate"
                                                label="Event Date"
                                                type="date"
                                            />
                                            <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: "block" }}>
                                                Value: {values.eventDate || "No date selected"}
                                            </Typography>
                                        </Box>

                                        {/* DateTime Local Example */}
                                        <Box>
                                            <Typography variant="h6" sx={{ mb: 2 }}>Date & Time Input</Typography>
                                            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                                Ideal for appointments, meetings, or events with specific timing.
                                            </Typography>
                                            <DateInput
                                                name="appointment"
                                                label="Appointment Date & Time"
                                                type="datetime-local"
                                                isRequired
                                            />
                                            <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: "block" }}>
                                                Value: {values.appointment || "No appointment scheduled"}
                                            </Typography>
                                        </Box>

                                        {/* Required Example */}
                                        <Box>
                                            <Typography variant="h6" sx={{ mb: 2 }}>Required Field</Typography>
                                            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                                This field is marked as required and must be filled.
                                            </Typography>
                                            <DateInput
                                                name="deadline"
                                                label="Project Deadline *"
                                                type="datetime-local"
                                                isRequired
                                            />
                                            <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: "block" }}>
                                                Value: {values.deadline || "Deadline not set"}
                                            </Typography>
                                        </Box>
                                    </Box>
                                </Form>
                            )}
                        </Formik>
                    </Box>

                    {/* Comparison Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Date vs DateTime Comparison</Typography>
                        
                        <Formik
                            initialValues={{
                                dateOnly: "",
                                dateTime: "",
                            }}
                            onSubmit={(values) => {
                                console.log("Comparison form submitted:", values);
                            }}
                        >
                            {({ values }) => (
                                <Form>
                                    <Box sx={{ 
                                        display: "grid", 
                                        gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)" },
                                        gap: 3, 
                                    }}>
                                        {/* Date Only */}
                                        <Box sx={{ 
                                            p: 2, 
                                            borderRadius: 1, 
                                            bgcolor: "grey.50",
                                            border: "1px solid",
                                            borderColor: "grey.200",
                                        }}>
                                            <Typography variant="h6" sx={{ mb: 1, color: "primary.main" }}>
                                                type="date"
                                            </Typography>
                                            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                                • Shows date picker only<br/>
                                                • Format: YYYY-MM-DD<br/>
                                                • Good for: birthdays, deadlines, events
                                            </Typography>
                                            <DateInput
                                                name="dateOnly"
                                                label="Choose a Date"
                                                type="date"
                                            />
                                            <Typography variant="caption" sx={{ mt: 1, display: "block", fontFamily: "monospace" }}>
                                                Value: "{values.dateOnly || ""}"
                                            </Typography>
                                        </Box>

                                        {/* DateTime Local */}
                                        <Box sx={{ 
                                            p: 2, 
                                            borderRadius: 1, 
                                            bgcolor: "grey.50",
                                            border: "1px solid",
                                            borderColor: "grey.200",
                                        }}>
                                            <Typography variant="h6" sx={{ mb: 1, color: "secondary.main" }}>
                                                type="datetime-local"
                                            </Typography>
                                            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                                • Shows date and time picker<br/>
                                                • Format: YYYY-MM-DDTHH:mm:ss<br/>
                                                • Good for: appointments, meetings, schedules
                                            </Typography>
                                            <DateInput
                                                name="dateTime"
                                                label="Choose Date & Time"
                                                type="datetime-local"
                                            />
                                            <Typography variant="caption" sx={{ mt: 1, display: "block", fontFamily: "monospace" }}>
                                                Value: "{values.dateTime || ""}"
                                            </Typography>
                                        </Box>
                                    </Box>
                                </Form>
                            )}
                        </Formik>
                    </Box>

                    {/* Use Cases Section */}
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
                                birthday: "",
                                meeting: "",
                                deadline: "",
                                event: "",
                                reminder: "",
                                vacation: "",
                            }}
                            onSubmit={(values) => {
                                console.log("Use cases form submitted:", values);
                            }}
                        >
                            <Form>
                                <Box sx={{ 
                                    display: "grid", 
                                    gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", lg: "repeat(3, 1fr)" },
                                    gap: 3, 
                                }}>
                                    {/* Personal Information */}
                                    <Box>
                                        <Typography variant="h6" sx={{ mb: 2 }}>Personal Information</Typography>
                                        <DateInput
                                            name="birthday"
                                            label="Date of Birth"
                                            type="date"
                                            isRequired
                                        />
                                    </Box>

                                    {/* Business Meetings */}
                                    <Box>
                                        <Typography variant="h6" sx={{ mb: 2 }}>Business Meetings</Typography>
                                        <DateInput
                                            name="meeting"
                                            label="Meeting Time"
                                            type="datetime-local"
                                            isRequired
                                        />
                                    </Box>

                                    {/* Project Management */}
                                    <Box>
                                        <Typography variant="h6" sx={{ mb: 2 }}>Project Management</Typography>
                                        <DateInput
                                            name="deadline"
                                            label="Project Deadline"
                                            type="date"
                                            isRequired
                                        />
                                    </Box>

                                    {/* Event Planning */}
                                    <Box>
                                        <Typography variant="h6" sx={{ mb: 2 }}>Event Planning</Typography>
                                        <DateInput
                                            name="event"
                                            label="Event Start Time"
                                            type="datetime-local"
                                        />
                                    </Box>

                                    {/* Reminders */}
                                    <Box>
                                        <Typography variant="h6" sx={{ mb: 2 }}>Reminders</Typography>
                                        <DateInput
                                            name="reminder"
                                            label="Remind Me At"
                                            type="datetime-local"
                                        />
                                    </Box>

                                    {/* Travel Planning */}
                                    <Box>
                                        <Typography variant="h6" sx={{ mb: 2 }}>Travel Planning</Typography>
                                        <DateInput
                                            name="vacation"
                                            label="Vacation Start Date"
                                            type="date"
                                        />
                                    </Box>
                                </Box>
                            </Form>
                        </Formik>
                    </Box>

                    {/* Features Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Key Features</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)" },
                            gap: 3, 
                        }}>
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Built-in Features</Typography>
                                <Box component="ul" sx={{ pl: 3, m: 0 }}>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>Clear Button:</strong> Easy one-click clearing when a date is selected
                                    </Typography>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>Native Picker:</strong> Uses browser's native date/time picker for best UX
                                    </Typography>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>Dark Mode Support:</strong> Automatically adapts picker styling to theme
                                    </Typography>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>Formik Integration:</strong> Seamless integration with form validation
                                    </Typography>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>Required Field Support:</strong> Visual indicators for required fields
                                    </Typography>
                                </Box>
                            </Box>

                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Browser Compatibility</Typography>
                                <Box component="ul" sx={{ pl: 3, m: 0 }}>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>Modern Browsers:</strong> Full support in Chrome, Firefox, Safari, Edge
                                    </Typography>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>Mobile Friendly:</strong> Optimized experience on mobile devices
                                    </Typography>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>Accessibility:</strong> Screen reader compatible with proper ARIA labels
                                    </Typography>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>Keyboard Navigation:</strong> Full keyboard support for accessibility
                                    </Typography>
                                </Box>
                            </Box>
                        </Box>

                        <Box sx={{ mt: 3 }}>
                            <Typography variant="h6" sx={{ mb: 2 }}>Try These Interactions:</Typography>
                            <Box component="ul" sx={{ pl: 3, m: 0 }}>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    Click on any date input to open the native date picker
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    Select a date and notice the clear button (X) appears on the right
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    Use the clear button to reset the input to empty
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    Compare how "date" vs "datetime-local" types behave differently
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    Test the responsive design by resizing your browser window
                                </Typography>
                            </Box>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};

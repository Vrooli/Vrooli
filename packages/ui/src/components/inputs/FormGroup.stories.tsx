import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import FormControl from "@mui/material/FormControl";
import FormLabel from "@mui/material/FormLabel";
import { FormGroup } from "./FormGroup.js";
import { FormControlLabel } from "./FormControlLabel.js";
import { Radio } from "./Radio.js";
import { Checkbox } from "./Checkbox.js";
import { Switch } from "./Switch/Switch.js";

const meta: Meta<typeof FormGroup> = {
    title: "Components/Inputs/FormGroup",
    component: FormGroup,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive FormGroup Playground
export const FormGroupShowcase: Story = {
    render: () => {
        const [row, setRow] = useState(false);
        const [notifications, setNotifications] = useState({
            email: true,
            push: false,
            sms: true,
            desktop: false,
        });
        const [features, setFeatures] = useState({
            darkMode: true,
            autoSave: false,
            spellCheck: true,
            analytics: false,
        });

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
                    maxWidth: 1200, 
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormGroup Controls</Typography>
                        
                        <Switch
                            checked={row}
                            onChange={(checked) => setRow(checked)}
                            size="sm"
                            label="Row Layout"
                            labelPosition="right"
                        />
                    </Box>

                    {/* Basic FormGroup Example */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%"
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Basic FormGroup</Typography>
                        <Typography variant="body2" sx={{ mb: 2 }}>
                            FormGroup wraps multiple form controls and manages their layout
                        </Typography>
                        
                        <FormControl component="fieldset">
                            <FormLabel component="legend">Select your interests</FormLabel>
                            <FormGroup row={row}>
                                <FormControlLabel control={<Checkbox />} label="Technology" />
                                <FormControlLabel control={<Checkbox />} label="Design" />
                                <FormControlLabel control={<Checkbox />} label="Business" />
                                <FormControlLabel control={<Checkbox />} label="Marketing" />
                                <FormControlLabel control={<Checkbox />} label="Science" />
                            </FormGroup>
                        </FormControl>
                    </Box>

                    {/* Mixed Controls Example */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%"
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Mixed Controls</Typography>
                        <Typography variant="body2" sx={{ mb: 2 }}>
                            FormGroup can contain different types of form controls
                        </Typography>
                        
                        <FormControl component="fieldset">
                            <FormLabel component="legend">Application Settings</FormLabel>
                            <FormGroup row={row}>
                                <FormControlLabel 
                                    control={<Switch variant="default" />} 
                                    label="Enable notifications" 
                                />
                                <FormControlLabel 
                                    control={<Checkbox color="primary" />} 
                                    label="Auto-update" 
                                />
                                <FormControlLabel 
                                    control={<Switch variant="success" />} 
                                    label="Developer mode" 
                                />
                                <FormControlLabel 
                                    control={<Checkbox color="secondary" />} 
                                    label="Show tips" 
                                />
                            </FormGroup>
                        </FormControl>
                    </Box>

                    {/* Notification Preferences Example */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%"
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Notification Preferences</Typography>
                        
                        <FormControl component="fieldset">
                            <FormLabel component="legend">Choose how you want to be notified</FormLabel>
                            <FormGroup row={row}>
                                <FormControlLabel
                                    control={
                                        <Checkbox 
                                            color="primary"
                                            checked={notifications.email}
                                            onChange={(e) => setNotifications(prev => ({
                                                ...prev,
                                                email: e.target.checked
                                            }))}
                                        />
                                    }
                                    label="Email"
                                />
                                <FormControlLabel
                                    control={
                                        <Checkbox 
                                            color="primary"
                                            checked={notifications.push}
                                            onChange={(e) => setNotifications(prev => ({
                                                ...prev,
                                                push: e.target.checked
                                            }))}
                                        />
                                    }
                                    label="Push Notifications"
                                />
                                <FormControlLabel
                                    control={
                                        <Checkbox 
                                            color="primary"
                                            checked={notifications.sms}
                                            onChange={(e) => setNotifications(prev => ({
                                                ...prev,
                                                sms: e.target.checked
                                            }))}
                                        />
                                    }
                                    label="SMS"
                                />
                                <FormControlLabel
                                    control={
                                        <Checkbox 
                                            color="primary"
                                            checked={notifications.desktop}
                                            onChange={(e) => setNotifications(prev => ({
                                                ...prev,
                                                desktop: e.target.checked
                                            }))}
                                        />
                                    }
                                    label="Desktop"
                                />
                            </FormGroup>
                        </FormControl>
                        
                        <Typography variant="body2" sx={{ mt: 2 }}>
                            Selected: {Object.entries(notifications)
                                .filter(([_, checked]) => checked)
                                .map(([key]) => key)
                                .join(", ") || "None"}
                        </Typography>
                    </Box>

                    {/* Nested FormGroups Example */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%"
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Nested FormGroups</Typography>
                        
                        <FormControl component="fieldset">
                            <FormLabel component="legend">Advanced Settings</FormLabel>
                            <FormGroup>
                                <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>Appearance</Typography>
                                <FormGroup row={!row} sx={{ ml: 2, mb: 2 }}>
                                    <FormControlLabel
                                        control={
                                            <Switch 
                                                variant="default"
                                                checked={features.darkMode}
                                                onChange={(checked) => setFeatures(prev => ({
                                                    ...prev,
                                                    darkMode: checked
                                                }))}
                                            />
                                        }
                                        label="Dark Mode"
                                    />
                                    <FormControlLabel
                                        control={<Checkbox color="primary" />}
                                        label="Compact View"
                                    />
                                    <FormControlLabel
                                        control={<Checkbox color="primary" />}
                                        label="Show Animations"
                                    />
                                </FormGroup>

                                <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>Editor</Typography>
                                <FormGroup row={!row} sx={{ ml: 2, mb: 2 }}>
                                    <FormControlLabel
                                        control={
                                            <Switch 
                                                variant="default"
                                                checked={features.autoSave}
                                                onChange={(checked) => setFeatures(prev => ({
                                                    ...prev,
                                                    autoSave: checked
                                                }))}
                                            />
                                        }
                                        label="Auto Save"
                                    />
                                    <FormControlLabel
                                        control={
                                            <Switch 
                                                variant="default"
                                                checked={features.spellCheck}
                                                onChange={(checked) => setFeatures(prev => ({
                                                    ...prev,
                                                    spellCheck: checked
                                                }))}
                                            />
                                        }
                                        label="Spell Check"
                                    />
                                    <FormControlLabel
                                        control={<Checkbox color="primary" />}
                                        label="Line Numbers"
                                    />
                                </FormGroup>

                                <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>Privacy</Typography>
                                <FormGroup row={!row} sx={{ ml: 2 }}>
                                    <FormControlLabel
                                        control={
                                            <Switch 
                                                variant="danger"
                                                checked={features.analytics}
                                                onChange={(checked) => setFeatures(prev => ({
                                                    ...prev,
                                                    analytics: checked
                                                }))}
                                            />
                                        }
                                        label="Analytics"
                                    />
                                    <FormControlLabel
                                        control={<Checkbox color="danger" />}
                                        label="Share Usage Data"
                                    />
                                </FormGroup>
                            </FormGroup>
                        </FormControl>
                    </Box>

                    {/* Responsive Layout Example */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%"
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Responsive Layout</Typography>
                        <Typography variant="body2" sx={{ mb: 2 }}>
                            This FormGroup automatically switches between row and column layout based on screen size
                        </Typography>
                        
                        <FormControl component="fieldset">
                            <FormLabel component="legend">Select Features</FormLabel>
                            <FormGroup sx={{ 
                                flexDirection: { xs: 'column', sm: row ? 'row' : 'column' },
                                gap: { xs: 1, sm: row ? 3 : 1 }
                            }}>
                                <FormControlLabel control={<Checkbox color="success" />} label="Feature A" />
                                <FormControlLabel control={<Checkbox color="success" />} label="Feature B" />
                                <FormControlLabel control={<Checkbox color="success" />} label="Feature C" />
                                <FormControlLabel control={<Checkbox color="success" />} label="Feature D" />
                            </FormGroup>
                        </FormControl>
                    </Box>
                </Box>
            </Box>
        );
    },
};
import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextField from "@mui/material/TextField";
import FormLabel from "@mui/material/FormLabel";
import { InputType, type SliderFormInput } from "@vrooli/shared";
import { Formik } from "formik";
import { Switch } from "../Switch/Switch.js";
import { FormInputSlider } from "./FormInputSlider.js";

const meta: Meta<typeof FormInputSlider> = {
    title: "Components/Inputs/Form/FormInputSlider",
    component: FormInputSlider,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
    render: () => {
        const [isEditing, setIsEditing] = useState(false);
        const [disabled, setDisabled] = useState(false);
        const [labelText, setLabelText] = useState("Slider Input");
        const [helperText, setHelperText] = useState("Drag to select a value");
        const [minValue, setMinValue] = useState("0");
        const [maxValue, setMaxValue] = useState("100");
        const [stepValue, setStepValue] = useState("1");

        const [fieldData, setFieldData] = useState<SliderFormInput>({
            id: "slider-1",
            fieldName: "sliderField",
            type: InputType.Slider,
            label: "Slider Input",
            helperText: "Drag to select a value",
            isRequired: false,
            yup: "",
            min: 0,
            max: 100,
            step: 1,
            defaultValue: 50,
        });

        const handleConfigUpdate = (newFieldData: SliderFormInput) => {
            setFieldData(newFieldData);
        };

        const handleDelete = () => {
            console.log("FormInputSlider deleted");
        };

        const handleLabelChange = (newLabel: string) => {
            setLabelText(newLabel);
            setFieldData(prev => ({ ...prev, label: newLabel }));
        };

        const handleHelperTextChange = (newHelperText: string) => {
            setHelperText(newHelperText);
            setFieldData(prev => ({ ...prev, helperText: newHelperText }));
        };

        const handleMinChange = (newMin: string) => {
            setMinValue(newMin);
            const minNum = parseFloat(newMin);
            if (!isNaN(minNum)) {
                setFieldData(prev => ({ ...prev, min: minNum }));
            }
        };

        const handleMaxChange = (newMax: string) => {
            setMaxValue(newMax);
            const maxNum = parseFloat(newMax);
            if (!isNaN(maxNum)) {
                setFieldData(prev => ({ ...prev, max: maxNum }));
            }
        };

        const handleStepChange = (newStep: string) => {
            setStepValue(newStep);
            const stepNum = parseFloat(newStep);
            if (!isNaN(stepNum)) {
                setFieldData(prev => ({ ...prev, step: stepNum }));
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
                    maxWidth: 1000, 
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormInputSlider Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)" },
                            gap: 3, 
                        }}>
                            <Switch
                                checked={isEditing}
                                onChange={(checked) => setIsEditing(checked)}
                                size="sm"
                                label="Editing Mode"
                                labelPosition="right"
                            />

                            <Switch
                                checked={disabled}
                                onChange={(checked) => setDisabled(checked)}
                                size="sm"
                                label="Disabled"
                                labelPosition="right"
                            />

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Label Text</FormLabel>
                                <TextField
                                    value={labelText}
                                    onChange={(e) => handleLabelChange(e.target.value)}
                                    size="small"
                                    fullWidth
                                />
                            </Box>

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Helper Text</FormLabel>
                                <TextField
                                    value={helperText}
                                    onChange={(e) => handleHelperTextChange(e.target.value)}
                                    size="small"
                                    fullWidth
                                    multiline
                                    rows={2}
                                />
                            </Box>

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Min Value</FormLabel>
                                <TextField
                                    value={minValue}
                                    onChange={(e) => handleMinChange(e.target.value)}
                                    size="small"
                                    type="number"
                                    fullWidth
                                />
                            </Box>

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Max Value</FormLabel>
                                <TextField
                                    value={maxValue}
                                    onChange={(e) => handleMaxChange(e.target.value)}
                                    size="small"
                                    type="number"
                                    fullWidth
                                />
                            </Box>

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Step Value</FormLabel>
                                <TextField
                                    value={stepValue}
                                    onChange={(e) => handleStepChange(e.target.value)}
                                    size="small"
                                    type="number"
                                    fullWidth
                                />
                            </Box>
                        </Box>
                    </Box>

                    {/* Component Display */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>FormInputSlider Component</Typography>
                        
                        <Box sx={{ p: 2, border: "1px dashed #ccc", borderRadius: 1 }}>
                            <Formik
                                initialValues={{ sliderField: 50 }}
                                onSubmit={() => {}}
                            >
                                <FormInputSlider
                                    disabled={disabled}
                                    fieldData={fieldData}
                                    index={0}
                                    isEditing={isEditing}
                                    onConfigUpdate={handleConfigUpdate}
                                    onDelete={handleDelete}
                                />
                            </Formik>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};
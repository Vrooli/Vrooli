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
import { InputType, type FormInputType } from "@vrooli/shared";
import { Formik } from "formik";
import { Switch } from "../Switch/Switch.js";
import { FormInput } from "./FormInput.js";

const meta: Meta<typeof FormInput> = {
    title: "Components/Inputs/Form/FormInput",
    component: FormInput,
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
        const [inputType, setInputType] = useState<InputType>(InputType.Text);
        const [labelText, setLabelText] = useState("Sample Input");
        const [helperText, setHelperText] = useState("This is helper text");

        const [fieldData, setFieldData] = useState<FormInputType>({
            id: "input-1",
            fieldName: "sampleInput",
            type: inputType,
            label: "Sample Input",
            helperText: "This is helper text",
            isRequired: false,
            yup: {
                type: "string",
                required: false,
                checks: [],
            },
            props: {
                defaultValue: "",
                placeholder: "Enter text here",
                isMarkdown: false,
                maxChars: 1000,
                maxRows: 2,
                minRows: 1,
            },
        } as TextFormInput);

        const handleConfigUpdate = (newFieldData: FormInputType) => {
            setFieldData(newFieldData);
        };

        const handleDelete = () => {
            console.log("FormInput deleted");
        };

        const handleTypeChange = (newType: InputType) => {
            setInputType(newType);
            
            // Create appropriate props based on the input type
            let newProps: any = {};
            switch (newType) {
                case InputType.Text:
                    newProps = {
                        defaultValue: "",
                        placeholder: "Enter text here",
                        isMarkdown: false,
                        maxChars: 1000,
                        maxRows: 2,
                        minRows: 1,
                    };
                    break;
                case InputType.IntegerInput:
                    newProps = {
                        defaultValue: 0,
                        min: 0,
                        max: 100,
                        step: 1,
                    };
                    break;
                case InputType.Checkbox:
                    newProps = {
                        defaultValue: [],
                        options: [
                            { label: "Option 1", value: "option1" },
                            { label: "Option 2", value: "option2" },
                        ],
                        row: false,
                    };
                    break;
                case InputType.Switch:
                    newProps = {
                        defaultValue: false,
                        label: "Toggle me",
                        size: "medium",
                        color: "primary",
                    };
                    break;
                case InputType.Radio:
                    newProps = {
                        defaultValue: null,
                        options: [
                            { label: "Option A", value: "optionA" },
                            { label: "Option B", value: "optionB" },
                        ],
                        row: false,
                    };
                    break;
                case InputType.Selector:
                    newProps = {
                        defaultValue: null,
                        options: [
                            { label: "Select 1", value: "select1" },
                            { label: "Select 2", value: "select2" },
                        ],
                        getOptionLabel: (option: any) => option?.label || "",
                        getOptionValue: (option: any) => option?.value || "",
                        getOptionDescription: (option: any) => option?.description || null,
                        fullWidth: true,
                    };
                    break;
                case InputType.Slider:
                    newProps = {
                        defaultValue: 50,
                        min: 0,
                        max: 100,
                        step: 1,
                        valueLabelDisplay: "auto",
                    };
                    break;
                case InputType.LanguageInput:
                    newProps = {
                        defaultValue: [],
                    };
                    break;
            }
            
            setFieldData(prev => ({ 
                ...prev, 
                type: newType,
                props: newProps,
            }));
        };

        const handleLabelChange = (newLabel: string) => {
            setLabelText(newLabel);
            setFieldData(prev => ({ ...prev, label: newLabel }));
        };

        const handleHelperTextChange = (newHelperText: string) => {
            setHelperText(newHelperText);
            setFieldData(prev => ({ ...prev, helperText: newHelperText }));
        };

        const inputTypes = [
            InputType.Text,
            InputType.IntegerInput,
            InputType.Checkbox,
            InputType.Switch,
            InputType.Radio,
            InputType.Selector,
            InputType.Slider,
            InputType.LanguageInput,
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormInput Controls</Typography>
                        
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

                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Input Type</FormLabel>
                                <RadioGroup
                                    value={inputType}
                                    onChange={(e) => handleTypeChange(e.target.value as InputType)}
                                    sx={{ gap: 0.5, maxHeight: 200, overflow: "auto" }}
                                >
                                    {inputTypes.map(type => (
                                        <FormControlLabel 
                                            key={type}
                                            value={type} 
                                            control={<Radio size="small" />} 
                                            label={type} 
                                            sx={{ m: 0 }} 
                                        />
                                    ))}
                                </RadioGroup>
                            </FormControl>

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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormInput Component</Typography>
                        
                        <Box sx={{ p: 2, border: "1px dashed #ccc", borderRadius: 1 }}>
                            <Formik
                                initialValues={{ sampleInput: "" }}
                                onSubmit={() => {}}
                            >
                                <FormInput
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
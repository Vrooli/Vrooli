import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextField from "@mui/material/TextField";
import FormLabel from "@mui/material/FormLabel";
import FormControl from "@mui/material/FormControl";
import RadioGroup from "@mui/material/RadioGroup";
import FormControlLabel from "@mui/material/FormControlLabel";
import Radio from "@mui/material/Radio";
import { InputType, type TextFormInput } from "@vrooli/shared";
import { Formik } from "formik";
import { Switch } from "../Switch/Switch.js";
import { FormInputText } from "./FormInputText.js";

const meta: Meta<typeof FormInputText> = {
    title: "Components/Inputs/Form/FormInputText",
    component: FormInputText,
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
        const [labelText, setLabelText] = useState("Text Input Field");
        const [helperText, setHelperText] = useState("Enter some text");
        const [placeholder, setPlaceholder] = useState("Type here...");
        const [maxLength, setMaxLength] = useState("100");
        const [isMultiline, setIsMultiline] = useState(false);
        const [isMarkdown, setIsMarkdown] = useState(false);

        const [fieldData, setFieldData] = useState<TextFormInput>({
            id: "text-1",
            fieldName: "textField",
            type: InputType.Text,
            label: "Text Input Field",
            helperText: "Enter some text",
            isRequired: false,
            yup: "",
            defaultValue: "",
            placeholder: "Type here...",
            maxLength: 100,
            isMarkdown: false,
            isMultiline: false,
        });

        const handleConfigUpdate = (newFieldData: TextFormInput) => {
            setFieldData(newFieldData);
        };

        const handleDelete = () => {
            console.log("FormInputText deleted");
        };

        const handleLabelChange = (newLabel: string) => {
            setLabelText(newLabel);
            setFieldData(prev => ({ ...prev, label: newLabel }));
        };

        const handleHelperTextChange = (newHelperText: string) => {
            setHelperText(newHelperText);
            setFieldData(prev => ({ ...prev, helperText: newHelperText }));
        };

        const handlePlaceholderChange = (newPlaceholder: string) => {
            setPlaceholder(newPlaceholder);
            setFieldData(prev => ({ ...prev, placeholder: newPlaceholder }));
        };

        const handleMaxLengthChange = (newMaxLength: string) => {
            setMaxLength(newMaxLength);
            const maxLengthNum = parseInt(newMaxLength, 10);
            if (!isNaN(maxLengthNum)) {
                setFieldData(prev => ({ ...prev, maxLength: maxLengthNum }));
            }
        };

        const handleMultilineChange = (checked: boolean) => {
            setIsMultiline(checked);
            setFieldData(prev => ({ ...prev, isMultiline: checked }));
        };

        const handleMarkdownChange = (checked: boolean) => {
            setIsMarkdown(checked);
            setFieldData(prev => ({ ...prev, isMarkdown: checked }));
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormInputText Controls</Typography>
                        
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

                            <Switch
                                checked={isMultiline}
                                onChange={handleMultilineChange}
                                size="sm"
                                label="Multiline"
                                labelPosition="right"
                            />

                            <Switch
                                checked={isMarkdown}
                                onChange={handleMarkdownChange}
                                size="sm"
                                label="Markdown Support"
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
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Placeholder</FormLabel>
                                <TextField
                                    value={placeholder}
                                    onChange={(e) => handlePlaceholderChange(e.target.value)}
                                    size="small"
                                    fullWidth
                                />
                            </Box>

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Max Length</FormLabel>
                                <TextField
                                    value={maxLength}
                                    onChange={(e) => handleMaxLengthChange(e.target.value)}
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormInputText Component</Typography>
                        
                        <Box sx={{ p: 2, border: "1px dashed #ccc", borderRadius: 1 }}>
                            <Formik
                                initialValues={{ textField: "" }}
                                onSubmit={() => {}}
                            >
                                <FormInputText
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

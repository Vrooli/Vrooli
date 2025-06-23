import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextField from "@mui/material/TextField";
import FormLabel from "@mui/material/FormLabel";
import { InputType, type TagSelectorFormInput } from "@vrooli/shared";
import { Formik } from "formik";
import { Switch } from "../Switch/Switch.js";
import { FormInputTagSelector } from "./FormInputTagSelector.js";

const meta: Meta<typeof FormInputTagSelector> = {
    title: "Components/Inputs/Form/FormInputTagSelector",
    component: FormInputTagSelector,
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
        const [labelText, setLabelText] = useState("Tag Selector");
        const [helperText, setHelperText] = useState("Select or create tags");

        const [fieldData, setFieldData] = useState<TagSelectorFormInput>({
            id: "tagselector-1",
            fieldName: "tagSelectorField",
            type: InputType.TagSelector,
            label: "Tag Selector",
            helperText: "Select or create tags",
            isRequired: false,
            yup: "",
            defaultValue: [],
        });

        const handleConfigUpdate = (newFieldData: TagSelectorFormInput) => {
            setFieldData(newFieldData);
        };

        const handleDelete = () => {
            console.log("FormInputTagSelector deleted");
        };

        const handleLabelChange = (newLabel: string) => {
            setLabelText(newLabel);
            setFieldData(prev => ({ ...prev, label: newLabel }));
        };

        const handleHelperTextChange = (newHelperText: string) => {
            setHelperText(newHelperText);
            setFieldData(prev => ({ ...prev, helperText: newHelperText }));
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormInputTagSelector Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)" },
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormInputTagSelector Component</Typography>
                        
                        <Box sx={{ p: 2, border: "1px dashed #ccc", borderRadius: 1 }}>
                            <Formik
                                initialValues={{ tagSelectorField: [] }}
                                onSubmit={() => {}}
                            >
                                <FormInputTagSelector
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
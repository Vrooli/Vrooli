import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextField from "@mui/material/TextField";
import FormLabel from "@mui/material/FormLabel";
import { InputType, type LinkUrlFormInput } from "@vrooli/shared";
import { Formik } from "formik";
import { Switch } from "../Switch/Switch.js";
import { FormInputLinkUrl } from "./FormInputLinkUrl.js";

const meta: Meta<typeof FormInputLinkUrl> = {
    title: "Components/Inputs/Form/FormInputLinkUrl",
    component: FormInputLinkUrl,
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
        const [labelText, setLabelText] = useState("URL Input");
        const [helperText, setHelperText] = useState("Enter a valid URL");

        const [fieldData, setFieldData] = useState<LinkUrlFormInput>({
            id: "linkurl-1",
            fieldName: "linkUrlField",
            type: InputType.LinkUrl,
            label: "URL Input",
            helperText: "Enter a valid URL",
            isRequired: false,
            yup: "",
            defaultValue: "https://example.com",
        });

        const handleConfigUpdate = (newFieldData: LinkUrlFormInput) => {
            setFieldData(newFieldData);
        };

        const handleDelete = () => {
            console.log("FormInputLinkUrl deleted");
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormInputLinkUrl Controls</Typography>
                        
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormInputLinkUrl Component</Typography>
                        
                        <Box sx={{ p: 2, border: "1px dashed #ccc", borderRadius: 1 }}>
                            <Formik
                                initialValues={{ linkUrlField: "https://example.com" }}
                                onSubmit={() => {}}
                            >
                                <FormInputLinkUrl
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
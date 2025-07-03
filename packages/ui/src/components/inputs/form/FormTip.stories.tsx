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
import { FormStructureType, type FormTipType } from "@vrooli/shared";
import { Switch } from "../Switch/Switch.js";
import { FormTip } from "./FormTip.js";

const meta: Meta<typeof FormTip> = {
    title: "Components/Inputs/Form/FormTip",
    component: FormTip,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
    render: () => {
        const [isEditing, setIsEditing] = useState(false);
        const [text, setText] = useState("This is a helpful tip for users!");
        const [variant, setVariant] = useState<"info" | "warning" | "error" | "success">("info");

        const [element, setElement] = useState<FormTipType>({
            id: "tip-1",
            fieldName: "tip",
            type: FormStructureType.Tip,
            text: "This is a helpful tip for users!",
            variant: "info",
        });

        const handleUpdate = (data: Partial<FormTipType>) => {
            setElement(prev => ({ ...prev, ...data }));
        };

        const handleDelete = () => {
            console.log("FormTip deleted");
        };

        const handleTextChange = (newText: string) => {
            setText(newText);
            handleUpdate({ text: newText });
        };

        const handleVariantChange = (newVariant: "info" | "warning" | "error" | "success") => {
            setVariant(newVariant);
            handleUpdate({ variant: newVariant });
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormTip Controls</Typography>
                        
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

                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Tip Variant</FormLabel>
                                <RadioGroup
                                    value={variant}
                                    onChange={(e) => handleVariantChange(e.target.value as "info" | "warning" | "error" | "success")}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="info" control={<Radio size="small" />} label="Info" sx={{ m: 0 }} />
                                    <FormControlLabel value="warning" control={<Radio size="small" />} label="Warning" sx={{ m: 0 }} />
                                    <FormControlLabel value="error" control={<Radio size="small" />} label="Error" sx={{ m: 0 }} />
                                    <FormControlLabel value="success" control={<Radio size="small" />} label="Success" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Tip Text</FormLabel>
                                <TextField
                                    value={text}
                                    onChange={(e) => handleTextChange(e.target.value)}
                                    size="small"
                                    fullWidth
                                    multiline
                                    rows={3}
                                    placeholder="Enter helpful tip text"
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormTip Component</Typography>
                        
                        <Box sx={{ p: 2, border: "1px dashed #ccc", borderRadius: 1 }}>
                            <FormTip
                                element={element}
                                isEditing={isEditing}
                                onUpdate={handleUpdate}
                                onDelete={handleDelete}
                            />
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};

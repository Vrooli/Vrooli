import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextField from "@mui/material/TextField";
import FormLabel from "@mui/material/FormLabel";
import { FormStructureType, type FormQrCodeType } from "@vrooli/shared";
import { Switch } from "../Switch/Switch.js";
import { FormQrCode } from "./FormQrCode.js";

const meta: Meta<typeof FormQrCode> = {
    title: "Components/Inputs/Form/FormQrCode",
    component: FormQrCode,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
    render: () => {
        const [isEditing, setIsEditing] = useState(false);
        const [data, setData] = useState("https://example.com");
        const [description, setDescription] = useState("Sample QR Code");

        const [element, setElement] = useState<FormQrCodeType>({
            id: "qr-1",
            fieldName: "qrCode",
            type: FormStructureType.QrCode,
            data: "https://example.com",
            description: "Sample QR Code",
        });

        const handleUpdate = (data: Partial<FormQrCodeType>) => {
            setElement(prev => ({ ...prev, ...data }));
        };

        const handleDelete = () => {
            console.log("FormQrCode deleted");
        };

        const handleDataChange = (newData: string) => {
            setData(newData);
            handleUpdate({ data: newData });
        };

        const handleDescriptionChange = (newDescription: string) => {
            setDescription(newDescription);
            handleUpdate({ description: newDescription });
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormQrCode Controls</Typography>
                        
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

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>QR Code Data</FormLabel>
                                <TextField
                                    value={data}
                                    onChange={(e) => handleDataChange(e.target.value)}
                                    size="small"
                                    fullWidth
                                    placeholder="URL, text, or data to encode"
                                />
                            </Box>

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Description</FormLabel>
                                <TextField
                                    value={description}
                                    onChange={(e) => handleDescriptionChange(e.target.value)}
                                    size="small"
                                    fullWidth
                                    multiline
                                    rows={2}
                                    placeholder="Optional description"
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormQrCode Component</Typography>
                        
                        <Box sx={{ p: 2, border: "1px dashed #ccc", borderRadius: 1 }}>
                            <FormQrCode
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

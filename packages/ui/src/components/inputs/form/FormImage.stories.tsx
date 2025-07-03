import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextField from "@mui/material/TextField";
import FormLabel from "@mui/material/FormLabel";
import { FormStructureType, type FormImageType } from "@vrooli/shared";
import { Switch } from "../Switch/Switch.js";
import { FormImage } from "./FormImage.js";

const meta: Meta<typeof FormImage> = {
    title: "Components/Inputs/Form/FormImage",
    component: FormImage,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
    render: () => {
        const [isEditing, setIsEditing] = useState(false);
        const [src, setSrc] = useState("https://via.placeholder.com/400x300");
        const [alt, setAlt] = useState("Sample image");
        const [description, setDescription] = useState("A sample image for demonstration");

        const [element, setElement] = useState<FormImageType>({
            id: "image-1",
            fieldName: "image",
            type: FormStructureType.Image,
            src: "https://via.placeholder.com/400x300",
            alt: "Sample image",
            description: "A sample image for demonstration",
        });

        const handleUpdate = (data: Partial<FormImageType>) => {
            setElement(prev => ({ ...prev, ...data }));
        };

        const handleDelete = () => {
            console.log("FormImage deleted");
        };

        const handleSrcChange = (newSrc: string) => {
            setSrc(newSrc);
            handleUpdate({ src: newSrc });
        };

        const handleAltChange = (newAlt: string) => {
            setAlt(newAlt);
            handleUpdate({ alt: newAlt });
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormImage Controls</Typography>
                        
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
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Image Source URL</FormLabel>
                                <TextField
                                    value={src}
                                    onChange={(e) => handleSrcChange(e.target.value)}
                                    size="small"
                                    fullWidth
                                    placeholder="https://example.com/image.jpg"
                                />
                            </Box>

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Alt Text</FormLabel>
                                <TextField
                                    value={alt}
                                    onChange={(e) => handleAltChange(e.target.value)}
                                    size="small"
                                    fullWidth
                                    placeholder="Image description"
                                />
                            </Box>

                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Description</FormLabel>
                                <TextField
                                    value={description}
                                    onChange={(e) => handleDescriptionChange(e.target.value)}
                                    size="small"
                                    multiline
                                    rows={2}
                                    fullWidth
                                    placeholder="Optional image description"
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormImage Component</Typography>
                        
                        <Box sx={{ p: 2, border: "1px dashed #ccc", borderRadius: 1 }}>
                            <FormImage
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

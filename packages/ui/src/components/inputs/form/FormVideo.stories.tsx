import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import TextField from "@mui/material/TextField";
import FormLabel from "@mui/material/FormLabel";
import { FormStructureType, type FormVideoType } from "@vrooli/shared";
import { Switch } from "../Switch/Switch.js";
import { FormVideo } from "./FormVideo.js";

const meta: Meta<typeof FormVideo> = {
    title: "Components/Inputs/Form/FormVideo",
    component: FormVideo,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
    render: () => {
        const [isEditing, setIsEditing] = useState(false);
        const [src, setSrc] = useState("https://www.youtube.com/watch?v=dQw4w9WgXcQ");
        const [description, setDescription] = useState("Sample video demonstration");

        const [element, setElement] = useState<FormVideoType>({
            id: "video-1",
            fieldName: "video",
            type: FormStructureType.Video,
            src: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
            description: "Sample video demonstration",
        });

        const handleUpdate = (data: Partial<FormVideoType>) => {
            setElement(prev => ({ ...prev, ...data }));
        };

        const handleDelete = () => {
            console.log("FormVideo deleted");
        };

        const handleSrcChange = (newSrc: string) => {
            setSrc(newSrc);
            handleUpdate({ src: newSrc });
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormVideo Controls</Typography>
                        
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
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Video Source URL</FormLabel>
                                <TextField
                                    value={src}
                                    onChange={(e) => handleSrcChange(e.target.value)}
                                    size="small"
                                    fullWidth
                                    placeholder="YouTube, Vimeo, or direct video URL"
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
                                    placeholder="Optional video description"
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormVideo Component</Typography>
                        
                        <Box sx={{ p: 2, border: "1px dashed #ccc", borderRadius: 1 }}>
                            <FormVideo
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
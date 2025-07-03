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
import { FormStructureType, type FormHeaderType, type HeaderTag } from "@vrooli/shared";
import { Switch } from "../Switch/Switch.js";
import { FormHeader } from "./FormHeader.js";

const meta: Meta<typeof FormHeader> = {
    title: "Components/Inputs/Form/FormHeader",
    component: FormHeader,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
    render: () => {
        const [isEditing, setIsEditing] = useState(false);
        const [headerTag, setHeaderTag] = useState<HeaderTag>("h1");
        const [content, setContent] = useState("Sample Header Text");
        const [variant, setVariant] = useState<"default" | "primary" | "secondary">("default");

        const [element, setElement] = useState<FormHeaderType>({
            id: "header-1",
            fieldName: "header",
            type: FormStructureType.Header,
            tag: "h1",
            variant: "default",
            text: "Sample Header Text",
        });

        const handleUpdate = (data: Partial<FormHeaderType>) => {
            setElement(prev => ({ ...prev, ...data }));
        };

        const handleDelete = () => {
            console.log("FormHeader deleted");
        };

        const handleTagChange = (newTag: HeaderTag) => {
            setHeaderTag(newTag);
            handleUpdate({ tag: newTag });
        };

        const handleContentChange = (newContent: string) => {
            setContent(newContent);
            handleUpdate({ text: newContent });
        };

        const handleVariantChange = (newVariant: "default" | "primary" | "secondary") => {
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormHeader Controls</Typography>
                        
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

                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Header Tag</FormLabel>
                                <RadioGroup
                                    value={headerTag}
                                    onChange={(e) => handleTagChange(e.target.value as HeaderTag)}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="h1" control={<Radio size="small" />} label="H1 (Title)" sx={{ m: 0 }} />
                                    <FormControlLabel value="h2" control={<Radio size="small" />} label="H2 (Subtitle)" sx={{ m: 0 }} />
                                    <FormControlLabel value="h3" control={<Radio size="small" />} label="H3 (Header)" sx={{ m: 0 }} />
                                    <FormControlLabel value="h4" control={<Radio size="small" />} label="H4 (Subheader)" sx={{ m: 0 }} />
                                    <FormControlLabel value="body1" control={<Radio size="small" />} label="Paragraph" sx={{ m: 0 }} />
                                    <FormControlLabel value="body2" control={<Radio size="small" />} label="Caption" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Color Variant</FormLabel>
                                <RadioGroup
                                    value={variant}
                                    onChange={(e) => handleVariantChange(e.target.value as "default" | "primary" | "secondary")}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="default" control={<Radio size="small" />} label="Default" sx={{ m: 0 }} />
                                    <FormControlLabel value="primary" control={<Radio size="small" />} label="Primary" sx={{ m: 0 }} />
                                    <FormControlLabel value="secondary" control={<Radio size="small" />} label="Secondary" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Content</FormLabel>
                                <TextField
                                    value={content}
                                    onChange={(e) => handleContentChange(e.target.value)}
                                    size="small"
                                    multiline
                                    rows={2}
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>
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
                        <Typography variant="h5" sx={{ mb: 3 }}>FormHeader Component</Typography>
                        
                        <Box sx={{ p: 2, border: "1px dashed #ccc", borderRadius: 1 }}>
                            <FormHeader
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

import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import FormControl from "@mui/material/FormControl";
import FormLabel from "@mui/material/FormLabel";
import TextField from "@mui/material/TextField";
import Chip from "@mui/material/Chip";
import Stack from "@mui/material/Stack";
import { Dropzone } from "./Dropzone.js";
import { Switch } from "../Switch/Switch.js";

const meta: Meta<typeof Dropzone> = {
    title: "Components/Inputs/Dropzone",
    component: Dropzone,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive Dropzone Playground
export const DropzoneShowcase: Story = {
    render: () => {
        const [disabled, setDisabled] = useState(false);
        const [showThumbs, setShowThumbs] = useState(true);
        const [maxFiles, setMaxFiles] = useState(5);
        const [dropzoneText, setDropzoneText] = useState("Drag 'n' drop files here or click");
        const [uploadText, setUploadText] = useState("Upload file(s)");
        const [cancelText, setCancelText] = useState("Cancel upload");
        const [acceptedFileTypes, setAcceptedFileTypes] = useState(["image/*", ".heic", ".heif"]);
        
        // Track uploaded files across all examples
        const [uploadedFiles, setUploadedFiles] = useState<any[]>([]);

        const handleUpload = (files: any[]) => {
            console.log("Files uploaded:", files);
            setUploadedFiles(prev => [...prev, ...files.map(file => ({
                name: file.name,
                size: file.size,
                type: file.type,
                uploadedAt: new Date().toLocaleString(),
            }))]);
        };

        const clearUploadedFiles = () => {
            setUploadedFiles([]);
        };

        const removeFile = (index: number) => {
            setUploadedFiles(prev => prev.filter((_, i) => i !== index));
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
                    maxWidth: 1400, 
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
                        <Typography variant="h5" sx={{ mb: 3 }}>Dropzone Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)", lg: "repeat(4, 1fr)" },
                            gap: 3, 
                        }}>
                            {/* Disabled Control */}
                            <FormControl component="fieldset" size="small">
                                <Switch
                                    checked={disabled}
                                    onChange={(checked) => setDisabled(checked)}
                                    size="sm"
                                    label="Disabled"
                                    labelPosition="right"
                                />
                            </FormControl>

                            {/* Show Thumbnails Control */}
                            <FormControl component="fieldset" size="small">
                                <Switch
                                    checked={showThumbs}
                                    onChange={(checked) => setShowThumbs(checked)}
                                    size="sm"
                                    label="Show Thumbnails"
                                    labelPosition="right"
                                />
                            </FormControl>

                            {/* Max Files Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Max Files</FormLabel>
                                <TextField
                                    type="number"
                                    value={maxFiles}
                                    onChange={(e) => setMaxFiles(Math.max(1, parseInt(e.target.value) || 1))}
                                    size="small"
                                    inputProps={{ min: 1, max: 100 }}
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>

                            {/* Dropzone Text Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Dropzone Text</FormLabel>
                                <TextField
                                    value={dropzoneText}
                                    onChange={(e) => setDropzoneText(e.target.value)}
                                    size="small"
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>

                            {/* Upload Button Text */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Upload Button Text</FormLabel>
                                <TextField
                                    value={uploadText}
                                    onChange={(e) => setUploadText(e.target.value)}
                                    size="small"
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>

                            {/* Cancel Button Text */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Cancel Button Text</FormLabel>
                                <TextField
                                    value={cancelText}
                                    onChange={(e) => setCancelText(e.target.value)}
                                    size="small"
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>

                            {/* Accepted File Types */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Accepted File Types</FormLabel>
                                <TextField
                                    value={acceptedFileTypes.join(", ")}
                                    onChange={(e) => setAcceptedFileTypes(e.target.value.split(",").map(s => s.trim()).filter(Boolean))}
                                    size="small"
                                    placeholder="image/*, .pdf, .doc"
                                    sx={{ width: "100%" }}
                                />
                            </FormControl>
                        </Box>
                    </Box>

                    {/* Live Preview */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Live Preview</Typography>
                        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                            This dropzone uses your current settings. Try dragging files or clicking to select them.
                        </Typography>
                        
                        <Dropzone
                            acceptedFileTypes={acceptedFileTypes}
                            dropzoneText={dropzoneText}
                            uploadText={uploadText}
                            cancelText={cancelText}
                            maxFiles={maxFiles}
                            showThumbs={showThumbs}
                            disabled={disabled}
                            onUpload={handleUpload}
                        />
                    </Box>

                    {/* File Type Examples */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>File Type Examples</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)", lg: "repeat(3, 1fr)" },
                            gap: 3, 
                        }}>
                            {/* Images Only */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Images Only</Typography>
                                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                    Accepts: image/*, .heic, .heif
                                </Typography>
                                <Dropzone
                                    acceptedFileTypes={["image/*", ".heic", ".heif"]}
                                    dropzoneText="Drop images here"
                                    uploadText="Upload Images"
                                    maxFiles={3}
                                    onUpload={(files) => {
                                        console.log("Images uploaded:", files);
                                        handleUpload(files);
                                    }}
                                />
                            </Box>

                            {/* Documents Only */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Documents Only</Typography>
                                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                    Accepts: .pdf, .doc, .docx, .txt
                                </Typography>
                                <Dropzone
                                    acceptedFileTypes={[".pdf", ".doc", ".docx", ".txt"]}
                                    dropzoneText="Drop documents here"
                                    uploadText="Upload Documents"
                                    maxFiles={5}
                                    showThumbs={false}
                                    onUpload={(files) => {
                                        console.log("Documents uploaded:", files);
                                        handleUpload(files);
                                    }}
                                />
                            </Box>

                            {/* Any File Type */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Any File Type</Typography>
                                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                    Accepts: All file types
                                </Typography>
                                <Dropzone
                                    acceptedFileTypes={[]}
                                    dropzoneText="Drop any files here"
                                    uploadText="Upload Files"
                                    maxFiles={10}
                                    showThumbs={false}
                                    onUpload={(files) => {
                                        console.log("Files uploaded:", files);
                                        handleUpload(files);
                                    }}
                                />
                            </Box>
                        </Box>
                    </Box>

                    {/* Configuration Examples */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Configuration Examples</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", lg: "repeat(2, 1fr)" },
                            gap: 3, 
                        }}>
                            {/* Single File Upload */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Single File Upload</Typography>
                                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                    Limited to 1 file with custom text
                                </Typography>
                                <Dropzone
                                    acceptedFileTypes={["image/*"]}
                                    dropzoneText="Select your profile picture"
                                    uploadText="Set Profile Picture"
                                    cancelText="Remove Picture"
                                    maxFiles={1}
                                    onUpload={(files) => {
                                        console.log("Profile picture uploaded:", files);
                                        handleUpload(files);
                                    }}
                                />
                            </Box>

                            {/* Disabled State */}
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Disabled State</Typography>
                                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                    Shows how the component looks when disabled
                                </Typography>
                                <Dropzone
                                    acceptedFileTypes={["image/*"]}
                                    dropzoneText="Upload temporarily disabled"
                                    uploadText="Upload"
                                    disabled={true}
                                    onUpload={(files) => {
                                        console.log("This should not be called when disabled");
                                    }}
                                />
                            </Box>
                        </Box>
                    </Box>

                    {/* Uploaded Files Display */}
                    {uploadedFiles.length > 0 && (
                        <Box sx={{ 
                            p: 3, 
                            bgcolor: "background.paper", 
                            borderRadius: 2, 
                            boxShadow: 1,
                            width: "100%",
                        }}>
                            <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 3 }}>
                                <Typography variant="h5">Uploaded Files ({uploadedFiles.length})</Typography>
                                <Typography 
                                    variant="body2" 
                                    color="primary" 
                                    sx={{ cursor: "pointer", textDecoration: "underline" }}
                                    onClick={clearUploadedFiles}
                                >
                                    Clear All
                                </Typography>
                            </Box>
                            
                            <Stack spacing={1}>
                                {uploadedFiles.map((file, index) => (
                                    <Chip
                                        key={`${file.name}-${index}`}
                                        label={`${file.name} (${(file.size / 1024).toFixed(1)} KB) - ${file.uploadedAt}`}
                                        onDelete={() => removeFile(index)}
                                        color="primary"
                                        variant="outlined"
                                        sx={{ justifyContent: "flex-start" }}
                                    />
                                ))}
                            </Stack>
                        </Box>
                    )}

                    {/* Features Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Key Features</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)" },
                            gap: 3, 
                        }}>
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Drag & Drop</Typography>
                                <Box component="ul" sx={{ pl: 3, m: 0 }}>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>Drag Files:</strong> Drag files from your file explorer directly onto the dropzone
                                    </Typography>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>Click to Browse:</strong> Click anywhere in the dropzone to open file browser
                                    </Typography>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>Multiple Files:</strong> Select multiple files at once (up to configured limit)
                                    </Typography>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>Visual Feedback:</strong> Clear visual indicators during drag operations
                                    </Typography>
                                </Box>
                            </Box>

                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>File Management</Typography>
                                <Box component="ul" sx={{ pl: 3, m: 0 }}>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>File Type Filtering:</strong> Accept only specific file types (images, documents, etc.)
                                    </Typography>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>File Limit:</strong> Configure maximum number of files that can be selected
                                    </Typography>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>Preview Thumbnails:</strong> Show image previews for visual files
                                    </Typography>
                                    <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                        <strong>Upload Control:</strong> Separate upload and cancel actions for better UX
                                    </Typography>
                                </Box>
                            </Box>
                        </Box>

                        <Box sx={{ mt: 3 }}>
                            <Typography variant="h6" sx={{ mb: 2 }}>Try These Interactions:</Typography>
                            <Box component="ul" sx={{ pl: 3, m: 0 }}>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    Drag image files from your computer onto any of the dropzones
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    Click on a dropzone to open the file browser dialog
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    Try uploading files that don't match the accepted types to see error handling
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    Select more files than the maximum allowed to test the file limit
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    Use the "Cancel upload" button to clear selected files before uploading
                                </Typography>
                                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                                    Toggle the controls above to see how they affect the dropzone behavior
                                </Typography>
                            </Box>
                        </Box>
                    </Box>

                    {/* Common Use Cases */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        width: "100%",
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Common Use Cases</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)" },
                            gap: 3, 
                        }}>
                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Profile Pictures</Typography>
                                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                    Single image upload with preview for user avatars
                                </Typography>
                                <Box component="pre" sx={{ 
                                    bgcolor: "grey.100", 
                                    p: 2, 
                                    borderRadius: 1, 
                                    fontSize: "0.875rem",
                                    overflow: "auto"
                                }}>
{`<Dropzone
  acceptedFileTypes={["image/*"]}
  maxFiles={1}
  dropzoneText="Upload your profile picture"
  uploadText="Set Picture"
/>`}
                                </Box>
                            </Box>

                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Document Upload</Typography>
                                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                    Multiple document upload for forms and applications
                                </Typography>
                                <Box component="pre" sx={{ 
                                    bgcolor: "grey.100", 
                                    p: 2, 
                                    borderRadius: 1, 
                                    fontSize: "0.875rem",
                                    overflow: "auto"
                                }}>
{`<Dropzone
  acceptedFileTypes={[".pdf", ".doc", ".docx"]}
  maxFiles={5}
  showThumbs={false}
  dropzoneText="Upload supporting documents"
/>`}
                                </Box>
                            </Box>

                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>Media Gallery</Typography>
                                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                    Multiple image upload with thumbnail previews
                                </Typography>
                                <Box component="pre" sx={{ 
                                    bgcolor: "grey.100", 
                                    p: 2, 
                                    borderRadius: 1, 
                                    fontSize: "0.875rem",
                                    overflow: "auto"
                                }}>
{`<Dropzone
  acceptedFileTypes={["image/*"]}
  maxFiles={20}
  showThumbs={true}
  dropzoneText="Upload gallery images"
/>`}
                                </Box>
                            </Box>

                            <Box>
                                <Typography variant="h6" sx={{ mb: 2 }}>File Backup</Typography>
                                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                    General file upload for backup or data transfer
                                </Typography>
                                <Box component="pre" sx={{ 
                                    bgcolor: "grey.100", 
                                    p: 2, 
                                    borderRadius: 1, 
                                    fontSize: "0.875rem",
                                    overflow: "auto"
                                }}>
{`<Dropzone
  acceptedFileTypes={[]}
  maxFiles={100}
  showThumbs={false}
  dropzoneText="Drop any files to backup"
/>`}
                                </Box>
                            </Box>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};
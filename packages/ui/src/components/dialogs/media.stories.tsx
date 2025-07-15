import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { action } from "@storybook/addon-actions";
import {
    Box,
    Card,
    CardContent,
    Typography,
    TextField,
    Alert,
    Stack,
    Grid,
} from "@mui/material";
import { Button } from "../buttons/Button.js";
import { ImagePopup, VideoPopup, VideoThumbnail } from "./media.js";
import { centeredDecorator } from "../../__test/helpers/storybookDecorators.tsx";

const meta: Meta = {
    title: "Components/Dialogs/Media",
    parameters: {
        layout: "fullscreen",
        backgrounds: { disable: true },
        docs: {
            story: {
                inline: false,
                iframeHeight: 600,
            },
        },
    },
    tags: ["autodocs"],
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Sample media URLs for stories
const sampleImageUrl = "https://picsum.photos/800/600";
const sampleVideoUrl = "https://www.youtube.com/embed/dQw4w9WgXcQ";
const sampleYouTubeUrl = "https://www.youtube.com/watch?v=dQw4w9WgXcQ";

// Showcase story with controls for all media components
export const Showcase: Story = {
    render: () => {
        const [imagePopupOpen, setImagePopupOpen] = useState(false);
        const [videoPopupOpen, setVideoPopupOpen] = useState(false);
        const [currentImageUrl, setCurrentImageUrl] = useState(sampleImageUrl);
        const [currentVideoUrl, setCurrentVideoUrl] = useState(sampleVideoUrl);
        const [imageAlt, setImageAlt] = useState("Sample image");
        const [customZIndex, setCustomZIndex] = useState(1300);

        const handleOpenImagePopup = () => {
            setImagePopupOpen(true);
            action("image-popup-opened")(currentImageUrl);
        };

        const handleCloseImagePopup = () => {
            setImagePopupOpen(false);
            action("image-popup-closed")();
        };

        const handleOpenVideoPopup = () => {
            setVideoPopupOpen(true);
            action("video-popup-opened")(currentVideoUrl);
        };

        const handleCloseVideoPopup = () => {
            setVideoPopupOpen(false);
            action("video-popup-closed")();
        };

        const handleThumbnailClick = () => {
            setVideoPopupOpen(true);
            action("video-thumbnail-clicked")(sampleYouTubeUrl);
        };

        return (
            <Stack spacing={3} sx={{ maxWidth: 800 }}>
                {/* Control Panel */}
                <Card>
                    <CardContent>
                        <Typography variant="h6" gutterBottom>
                            Media Dialog Controls
                        </Typography>
                        
                        <Grid container spacing={2}>
                            <Grid item xs={12} md={6}>
                                <Typography variant="h6" color="text.secondary" gutterBottom>
                                    Image Popup
                                </Typography>
                                
                                <Stack spacing={2}>
                                    <TextField
                                        label="Image URL"
                                        type="url"
                                        value={currentImageUrl}
                                        onChange={(e) => setCurrentImageUrl(e.target.value)}
                                        fullWidth
                                    />

                                    <TextField
                                        label="Alt Text"
                                        value={imageAlt}
                                        onChange={(e) => setImageAlt(e.target.value)}
                                        fullWidth
                                    />
                                </Stack>
                            </Grid>

                            <Grid item xs={12} md={6}>
                                <Typography variant="h6" color="text.secondary" gutterBottom>
                                    Video Popup
                                </Typography>
                                
                                <Stack spacing={2}>
                                    <TextField
                                        label="Video Embed URL"
                                        type="url"
                                        value={currentVideoUrl}
                                        onChange={(e) => setCurrentVideoUrl(e.target.value)}
                                        fullWidth
                                    />

                                    <TextField
                                        label="Z-Index"
                                        type="number"
                                        value={customZIndex}
                                        onChange={(e) => setCustomZIndex(parseInt(e.target.value) || 1300)}
                                        fullWidth
                                    />
                                </Stack>
                            </Grid>
                        </Grid>
                    </CardContent>
                </Card>

                {/* Action Buttons */}
                <Box sx={{ display: "flex", gap: 2, justifyContent: "center", flexWrap: "wrap" }}>
                    <Button onClick={handleOpenImagePopup} variant="primary" size="lg">
                        Open Image Popup
                    </Button>
                    <Button onClick={handleOpenVideoPopup} variant="primary" size="lg">
                        Open Video Popup
                    </Button>
                </Box>

                {/* Video Thumbnail Demo */}
                <div style={{
                    padding: "24px",
                    backgroundColor: "#f8f9fa",
                    borderRadius: "8px",
                    border: "1px solid #dee2e6",
                }}>
                    <h4 style={{ marginTop: 0, marginBottom: "16px", textAlign: "center" }}>
                        Video Thumbnail Component
                    </h4>
                    <div style={{ maxWidth: "400px", margin: "0 auto" }}>
                        <VideoThumbnail
                            src={sampleYouTubeUrl}
                            onClick={handleThumbnailClick}
                        />
                        <p style={{ 
                            textAlign: "center", 
                            marginTop: "8px", 
                            fontSize: "14px", 
                            color: "#666", 
                        }}>
                            Click the thumbnail to open video popup
                        </p>
                    </div>
                </div>

                {/* Info about media components */}
                <Alert severity="success">
                    <Typography variant="body2">
                        <strong>Media Dialog Features:</strong>
                    </Typography>
                    <Box component="ul" sx={{ mt: 1, mb: 0, pl: 2 }}>
                        <li><strong>ImagePopup:</strong> Full-screen image viewer with dark overlay</li>
                        <li><strong>VideoPopup:</strong> Embedded video player in modal</li>
                        <li><strong>VideoThumbnail:</strong> YouTube thumbnail with play button overlay</li>
                        <li>Click outside or close button to dismiss</li>
                        <li>Customizable z-index for layering</li>
                        <li>Responsive design for different screen sizes</li>
                    </Box>
                </Alert>

                {/* Current state */}
                <Alert severity="info">
                    <Typography variant="body2">
                        <strong>Current State:</strong>
                    </Typography>
                    <Box component="ul" sx={{ mt: 1, mb: 0, pl: 2 }}>
                        <li>Image Popup: {imagePopupOpen ? "Open" : "Closed"}</li>
                        <li>Video Popup: {videoPopupOpen ? "Open" : "Closed"}</li>
                        <li>Image URL: {currentImageUrl}</li>
                        <li>Video URL: {currentVideoUrl}</li>
                        <li>Z-Index: {customZIndex}</li>
                    </Box>
                </Alert>

                {/* Media Components */}
                <ImagePopup
                    open={imagePopupOpen}
                    onClose={handleCloseImagePopup}
                    src={currentImageUrl}
                    alt={imageAlt}
                    zIndex={customZIndex}
                />

                <VideoPopup
                    open={videoPopupOpen}
                    onClose={handleCloseVideoPopup}
                    src={currentVideoUrl}
                    zIndex={customZIndex}
                />
            </Stack>
        );
    },
};

// ImagePopup standalone
export const ImagePopupDemo: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const [currentImage, setCurrentImage] = useState(sampleImageUrl);

        const imageOptions = [
            { url: "https://picsum.photos/800/600", name: "Landscape" },
            { url: "https://picsum.photos/600/800", name: "Portrait" },
            { url: "https://picsum.photos/1200/400", name: "Wide Banner" },
            { url: "https://picsum.photos/400/400", name: "Square" },
        ];

        const handleOpen = (imageUrl: string) => {
            setCurrentImage(imageUrl);
            setIsOpen(true);
            action("image-opened")(imageUrl);
        };

        const handleClose = () => {
            setIsOpen(false);
            action("image-closed")();
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", gap: "24px" }}>
                <div style={{ 
                    padding: "16px", 
                    backgroundColor: "#e8f5e8", 
                    borderRadius: "4px", 
                    textAlign: "center",
                }}>
                    <strong>ImagePopup Demo</strong>
                    <div style={{ marginTop: "4px", fontSize: "14px", color: "#2e7d32" }}>
                        Click any image below to view it in full screen
                    </div>
                </div>

                <div style={{
                    display: "grid",
                    gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))",
                    gap: "16px",
                }}>
                    {imageOptions.map((option, index) => (
                        <div key={index} style={{ textAlign: "center" }}>
                            <img
                                src={option.url}
                                alt={option.name}
                                style={{
                                    width: "100%",
                                    height: "150px",
                                    objectFit: "cover",
                                    borderRadius: "8px",
                                    cursor: "pointer",
                                    border: "2px solid #e0e0e0",
                                    transition: "border-color 0.2s",
                                }}
                                onClick={() => handleOpen(option.url)}
                                onMouseEnter={(e) => {
                                    e.currentTarget.style.borderColor = "#2196f3";
                                }}
                                onMouseLeave={(e) => {
                                    e.currentTarget.style.borderColor = "#e0e0e0";
                                }}
                            />
                            <p style={{ marginTop: "8px", fontSize: "14px", color: "#666" }}>
                                {option.name}
                            </p>
                        </div>
                    ))}
                </div>

                <ImagePopup
                    open={isOpen}
                    onClose={handleClose}
                    src={currentImage}
                    alt="Demo image"
                />
            </div>
        );
    },
};

// VideoPopup standalone
export const VideoPopupDemo: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const [currentVideo, setCurrentVideo] = useState(sampleVideoUrl);

        const videoOptions = [
            { url: "https://www.youtube.com/embed/dQw4w9WgXcQ", name: "Rick Roll" },
            { url: "https://www.youtube.com/embed/9bZkp7q19f0", name: "Gangnam Style" },
            { url: "https://www.youtube.com/embed/kJQP7kiw5Fk", name: "Despacito" },
        ];

        const handleOpen = (videoUrl: string) => {
            setCurrentVideo(videoUrl);
            setIsOpen(true);
            action("video-opened")(videoUrl);
        };

        const handleClose = () => {
            setIsOpen(false);
            action("video-closed")();
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", gap: "24px" }}>
                <div style={{ 
                    padding: "16px", 
                    backgroundColor: "#fff3e0", 
                    borderRadius: "4px", 
                    textAlign: "center",
                }}>
                    <strong>VideoPopup Demo</strong>
                    <div style={{ marginTop: "4px", fontSize: "14px", color: "#e65100" }}>
                        Click any button below to open video in popup
                    </div>
                </div>

                <div style={{
                    display: "flex",
                    flexDirection: "column",
                    gap: "12px",
                    alignItems: "center",
                }}>
                    {videoOptions.map((option, index) => (
                        <Button
                            key={index}
                            onClick={() => handleOpen(option.url)}
                            variant="primary"
                            size="lg"
                            style={{ minWidth: "200px" }}
                        >
                            Play: {option.name}
                        </Button>
                    ))}
                </div>

                <div style={{
                    padding: "16px",
                    backgroundColor: "#f0f0f0",
                    borderRadius: "8px",
                    fontSize: "14px",
                    textAlign: "center",
                }}>
                    <strong>Note:</strong> Videos are embedded from YouTube. Make sure you have internet access to view them.
                </div>

                <VideoPopup
                    open={isOpen}
                    onClose={handleClose}
                    src={currentVideo}
                />
            </div>
        );
    },
};

// VideoThumbnail standalone
export const VideoThumbnailDemo: Story = {
    render: () => {
        const [videoPopupOpen, setVideoPopupOpen] = useState(false);
        const [currentVideoUrl, setCurrentVideoUrl] = useState("");

        const videoUrls = [
            "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
            "https://www.youtube.com/watch?v=9bZkp7q19f0",
            "https://www.youtube.com/watch?v=kJQP7kiw5Fk",
        ];

        const handleThumbnailClick = (url: string) => {
            // Convert YouTube watch URL to embed URL
            const videoId = url.match(/(?:youtu\.be\/|youtube\.com\/(?:embed\/|v\/|watch\?v=|watch\?.+&v=))([^&?]+)/)?.[1];
            const embedUrl = videoId ? `https://www.youtube.com/embed/${videoId}` : url;
            
            setCurrentVideoUrl(embedUrl);
            setVideoPopupOpen(true);
            action("thumbnail-clicked")(url);
        };

        const handleCloseVideoPopup = () => {
            setVideoPopupOpen(false);
            action("video-popup-closed")();
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", gap: "24px" }}>
                <div style={{ 
                    padding: "16px", 
                    backgroundColor: "#f3e5f5", 
                    borderRadius: "4px", 
                    textAlign: "center",
                }}>
                    <strong>VideoThumbnail Demo</strong>
                    <div style={{ marginTop: "4px", fontSize: "14px", color: "#7b1fa2" }}>
                        Hover over thumbnails to see play button animation
                    </div>
                </div>

                <div style={{
                    display: "grid",
                    gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))",
                    gap: "24px",
                }}>
                    {videoUrls.map((url, index) => (
                        <div key={index} style={{ 
                            padding: "16px",
                            backgroundColor: "white",
                            borderRadius: "8px",
                            border: "1px solid #e0e0e0",
                        }}>
                            <h4 style={{ marginTop: 0, marginBottom: "12px", textAlign: "center" }}>
                                Video {index + 1}
                            </h4>
                            <VideoThumbnail
                                src={url}
                                onClick={() => handleThumbnailClick(url)}
                            />
                        </div>
                    ))}
                </div>

                <div style={{
                    padding: "16px",
                    backgroundColor: "#e1f5fe",
                    borderRadius: "8px",
                    fontSize: "14px",
                }}>
                    <strong>VideoThumbnail Features:</strong>
                    <ul style={{ marginTop: "8px", marginBottom: 0 }}>
                        <li>Automatically extracts thumbnail from YouTube URL</li>
                        <li>Hover effect on play button</li>
                        <li>16:9 aspect ratio maintained</li>
                        <li>Responsive design</li>
                        <li>Accessibility support with proper ARIA labels</li>
                    </ul>
                </div>

                <VideoPopup
                    open={videoPopupOpen}
                    onClose={handleCloseVideoPopup}
                    src={currentVideoUrl}
                />
            </div>
        );
    },
};

// All media components together
export const MediaGallery: Story = {
    render: () => {
        const [imagePopupOpen, setImagePopupOpen] = useState(false);
        const [videoPopupOpen, setVideoPopupOpen] = useState(false);
        const [currentImageUrl, setCurrentImageUrl] = useState("");
        const [currentVideoUrl, setCurrentVideoUrl] = useState("");

        const galleryItems = [
            { type: "image", url: "https://picsum.photos/800/600", title: "Landscape Photo" },
            { type: "video", url: "https://www.youtube.com/watch?v=dQw4w9WgXcQ", title: "Music Video" },
            { type: "image", url: "https://picsum.photos/600/800", title: "Portrait Photo" },
            { type: "video", url: "https://www.youtube.com/watch?v=9bZkp7q19f0", title: "Dance Video" },
            { type: "image", url: "https://picsum.photos/1200/400", title: "Banner Image" },
            { type: "video", url: "https://www.youtube.com/watch?v=kJQP7kiw5Fk", title: "Music Hit" },
        ];

        const handleImageClick = (url: string) => {
            setCurrentImageUrl(url);
            setImagePopupOpen(true);
            action("gallery-image-clicked")(url);
        };

        const handleVideoClick = (url: string) => {
            const videoId = url.match(/(?:youtu\.be\/|youtube\.com\/(?:embed\/|v\/|watch\?v=|watch\?.+&v=))([^&?]+)/)?.[1];
            const embedUrl = videoId ? `https://www.youtube.com/embed/${videoId}` : url;
            setCurrentVideoUrl(embedUrl);
            setVideoPopupOpen(true);
            action("gallery-video-clicked")(url);
        };

        const handleCloseImage = () => {
            setImagePopupOpen(false);
            action("gallery-image-closed")();
        };

        const handleCloseVideo = () => {
            setVideoPopupOpen(false);
            action("gallery-video-closed")();
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", gap: "24px" }}>
                <div style={{ 
                    padding: "16px", 
                    backgroundColor: "#fce4ec", 
                    borderRadius: "4px", 
                    textAlign: "center",
                }}>
                    <strong>Media Gallery</strong>
                    <div style={{ marginTop: "4px", fontSize: "14px", color: "#c2185b" }}>
                        Mixed gallery of images and videos
                    </div>
                </div>

                <div style={{
                    display: "grid",
                    gridTemplateColumns: "repeat(auto-fit, minmax(250px, 1fr))",
                    gap: "16px",
                }}>
                    {galleryItems.map((item, index) => (
                        <div key={index} style={{
                            backgroundColor: "white",
                            borderRadius: "8px",
                            border: "1px solid #e0e0e0",
                            overflow: "hidden",
                        }}>
                            {item.type === "image" ? (
                                <img
                                    src={item.url}
                                    alt={item.title}
                                    style={{
                                        width: "100%",
                                        height: "200px",
                                        objectFit: "cover",
                                        cursor: "pointer",
                                    }}
                                    onClick={() => handleImageClick(item.url)}
                                />
                            ) : (
                                <VideoThumbnail
                                    src={item.url}
                                    onClick={() => handleVideoClick(item.url)}
                                />
                            )}
                            <div style={{ padding: "12px" }}>
                                <h4 style={{ margin: 0, fontSize: "16px", color: "#333" }}>
                                    {item.title}
                                </h4>
                                <p style={{ 
                                    margin: "4px 0 0 0", 
                                    fontSize: "14px", 
                                    color: "#666",
                                    textTransform: "capitalize",
                                }}>
                                    {item.type}
                                </p>
                            </div>
                        </div>
                    ))}
                </div>

                <ImagePopup
                    open={imagePopupOpen}
                    onClose={handleCloseImage}
                    src={currentImageUrl}
                    alt="Gallery image"
                />

                <VideoPopup
                    open={videoPopupOpen}
                    onClose={handleCloseVideo}
                    src={currentVideoUrl}
                />
            </div>
        );
    },
};

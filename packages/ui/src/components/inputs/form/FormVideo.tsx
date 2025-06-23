import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import FormControlLabel from "@mui/material/FormControlLabel";
import Switch from "@mui/material/Switch";
import { useTheme } from "@mui/material/styles";
import type { Palette } from "@mui/material/styles";
import { useCallback, useState, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { LinkInput } from "../LinkInput/LinkInput.js";
import { FormSettingsButtonRow, FormSettingsSection, propButtonStyle } from "./styles.js";
import { type FormVideoProps } from "./types.js";

export function FormVideo({
    element,
    isEditing,
    onUpdate,
    onDelete,
}: FormVideoProps) {
    const { palette } = useTheme() as { palette: Palette };
    const { t } = useTranslation();
    const [showSettings, setShowSettings] = useState(false);
    const [autoplay, setAutoplay] = useState(false);
    const [showControls, setShowControls] = useState(true);

    // Extract video ID from YouTube URL
    const videoId = useMemo(() => {
        if (!element.url) return null;
        const match = element.url.match(/(?:youtube\.com\/watch\?v=|youtu\.be\/)([^&\n?#]+)/);
        return match ? match[1] : null;
    }, [element.url]);

    const handleUrlChange = useCallback((newUrl: string) => {
        onUpdate({ url: newUrl });
    }, [onUpdate]);

    const handleLabelChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        onUpdate({ label: event.target.value });
    }, [onUpdate]);

    const handleDescriptionChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        onUpdate({ description: event.target.value });
    }, [onUpdate]);

    const handleAutoplayChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        setAutoplay(event.target.checked);
    }, []);

    const handleControlsChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        setShowControls(event.target.checked);
    }, []);

    // Render the video embed if URL exists and is valid
    const videoDisplay = videoId ? (
        <Box 
            sx={{ 
                position: "relative",
                paddingBottom: "56.25%", // 16:9 aspect ratio
                height: 0,
                overflow: "hidden",
                marginTop: 2,
                marginBottom: isEditing ? 1 : 2,
            }}
            data-testid="form-video-container"
            role="region"
            aria-label={element.label || t("VideoContent")}
        >
            <iframe
                src={`https://www.youtube.com/embed/${videoId}?autoplay=${autoplay ? 1 : 0}&controls=${showControls ? 1 : 0}`}
                title={element.label || t("VideoContent")}
                allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                allowFullScreen
                style={{
                    position: "absolute",
                    top: 0,
                    left: 0,
                    width: "100%",
                    height: "100%",
                    border: "none",
                    borderRadius: "8px",
                }}
                data-testid="form-video-iframe"
                data-autoplay={autoplay}
                data-controls={showControls}
            />
        </Box>
    ) : (
        <Box 
            sx={{
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                minHeight: 200,
                backgroundColor: palette.background.paper,
                border: `2px dashed ${palette.divider}`,
                borderRadius: 1,
                marginTop: 2,
                marginBottom: isEditing ? 1 : 2,
                color: palette.text.disabled,
                fontSize: "48px",
            }}
            data-testid="form-video-placeholder"
            role="img"
            aria-label={t("NoVideoAdded")}
        >
            🎬
        </Box>
    );

    // In non-editing mode, only show the video if URL exists
    if (!isEditing) {
        if (!element.url) return null;
        return (
            <div 
                data-testid="form-video"
                data-editing="false"
            >
                {videoDisplay}
            </div>
        );
    }

    // In editing mode, show video with controls
    return (
        <div 
            data-testid="form-video"
            data-editing="true"
        >
            {videoDisplay}
            
            <FormSettingsButtonRow>
                <Button
                    variant="text"
                    sx={propButtonStyle}
                    onClick={() => setShowSettings(!showSettings)}
                    data-testid="form-video-settings-button"
                    aria-label={t("VideoSettings")}
                    aria-expanded={showSettings}
                >
                    {t("VideoSettings")}
                </Button>
                <Button
                    variant="text"
                    sx={{ ...propButtonStyle, color: palette.error.main }}
                    onClick={onDelete}
                    data-testid="form-video-delete-button"
                    aria-label={t("DeleteVideo")}
                >
                    {t("Delete")}
                </Button>
            </FormSettingsButtonRow>

            {showSettings && (
                <FormSettingsSection data-testid="form-video-settings">
                    <LinkInput
                        fullWidth
                        label={t("VideoUrl")}
                        value={element.url || ""}
                        onChange={handleUrlChange}
                        placeholder={t("EnterYouTubeUrl")}
                        helperText={t("OnlyYouTubeSupported")}
                        data-testid="form-video-url-input"
                        aria-label={t("VideoUrl")}
                    />

                    <TextField
                        fullWidth
                        label={t("VideoTitle")}
                        value={element.label || ""}
                        onChange={handleLabelChange}
                        placeholder={t("EnterVideoTitle")}
                        data-testid="form-video-label-input"
                        aria-label={t("VideoTitle")}
                    />

                    <TextField
                        fullWidth
                        label={t("VideoDescription")}
                        value={element.description || ""}
                        onChange={handleDescriptionChange}
                        placeholder={t("EnterVideoDescription")}
                        multiline
                        rows={2}
                        data-testid="form-video-description-input"
                        aria-label={t("VideoDescription")}
                    />

                    <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={autoplay}
                                    onChange={handleAutoplayChange}
                                    data-testid="form-video-autoplay-switch"
                                />
                            }
                            label={t("AutoplayVideo")}
                        />
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={showControls}
                                    onChange={handleControlsChange}
                                    data-testid="form-video-controls-switch"
                                />
                            }
                            label={t("ShowVideoControls")}
                        />
                    </Box>
                </FormSettingsSection>
            )}
        </div>
    );
}

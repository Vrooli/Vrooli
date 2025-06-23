import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import TextField from "@mui/material/TextField";
import { Tooltip } from "../../Tooltip/Tooltip.js";
import { useTheme } from "@mui/material/styles";
import type { Palette } from "@mui/material/styles";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { Dropzone } from "../Dropzone/Dropzone.js";
import { LinkInput } from "../LinkInput/LinkInput.js";
import { FormSettingsButtonRow, FormSettingsSection, propButtonStyle } from "./styles.js";
import { type FormImageProps } from "./types.js";

export function FormImage({
    element,
    isEditing,
    onUpdate,
    onDelete,
}: FormImageProps) {
    const { palette } = useTheme() as { palette: Palette };
    const { t } = useTranslation();
    const [showSettings, setShowSettings] = useState(false);
    const [inputMode, setInputMode] = useState<"url" | "upload">("url");

    const handleUrlChange = useCallback((newUrl: string) => {
        onUpdate({ url: newUrl });
    }, [onUpdate]);

    const handleFileUpload = useCallback((files: File[]) => {
        if (files.length > 0) {
            // For now, we'll create a local URL for the uploaded file
            // In a real implementation, this would upload to a server
            const url = URL.createObjectURL(files[0]);
            onUpdate({ url });
        }
    }, [onUpdate]);

    const handleAltTextChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        onUpdate({ label: event.target.value });
    }, [onUpdate]);

    // Render the image if URL exists
    const imageDisplay = element.url ? (
        <Box 
            data-testid="form-image-display"
            sx={{ 
                position: "relative", 
                textAlign: "center",
                marginTop: 2,
                marginBottom: isEditing ? 1 : 2,
            }}
        >
            <img 
                src={element.url} 
                alt={element.label || t("Image")}
                data-testid="form-image-img"
                style={{
                    maxWidth: "100%",
                    height: "auto",
                    borderRadius: "8px",
                    display: "block",
                    margin: "0 auto",
                }}
            />
        </Box>
    ) : (
        <Box 
            data-testid="form-image-placeholder"
            sx={{
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                minHeight: 120,
                backgroundColor: palette.background.paper,
                border: `2px dashed ${palette.divider}`,
                borderRadius: 1,
                marginTop: 2,
                marginBottom: isEditing ? 1 : 2,
                color: palette.text.disabled,
                fontSize: "48px",
            }}
        >
            🖼️
        </Box>
    );

    // In non-editing mode, only show the image if URL exists
    if (!isEditing) {
        return element.url ? imageDisplay : null;
    }

    // In editing mode, show image with controls
    return (
        <Box sx={{ position: "relative" }} data-testid="form-image-container">
            {imageDisplay}
            
            <FormSettingsButtonRow>
                <Button
                    variant="text"
                    sx={propButtonStyle}
                    onClick={() => setShowSettings(!showSettings)}
                    data-testid="form-image-settings-button"
                    aria-label="Toggle image settings"
                    aria-expanded={showSettings}
                >
                    {t("ImageSettings")}
                </Button>
                <Button
                    variant="text"
                    sx={{ ...propButtonStyle, color: palette.error.main }}
                    onClick={onDelete}
                    data-testid="form-image-delete-button"
                    aria-label="Delete image"
                >
                    {t("Delete")}
                </Button>
            </FormSettingsButtonRow>

            {showSettings && (
                <FormSettingsSection data-testid="form-image-settings">
                    <Box sx={{ display: "flex", gap: 1, marginBottom: 2 }}>
                        <Tooltip title={t("UseImageUrl")}>
                            <Button
                                variant={inputMode === "url" ? "contained" : "outlined"}
                                size="small"
                                onClick={() => setInputMode("url")}
                                data-testid="form-image-url-mode-button"
                                aria-label="Use image URL"
                                aria-pressed={inputMode === "url"}
                            >
                                {t("UrlInput")}
                            </Button>
                        </Tooltip>
                        <Tooltip title={t("UploadImage")}>
                            <Button
                                variant={inputMode === "upload" ? "contained" : "outlined"}
                                size="small"
                                onClick={() => setInputMode("upload")}
                                data-testid="form-image-upload-mode-button"
                                aria-label="Upload image"
                                aria-pressed={inputMode === "upload"}
                            >
                                {t("Upload")}
                            </Button>
                        </Tooltip>
                    </Box>

                    {inputMode === "url" ? (
                        <LinkInput
                            fullWidth
                            label={t("ImageUrl")}
                            value={element.url || ""}
                            onChange={handleUrlChange}
                            placeholder={t("EnterImageUrl")}
                            data-testid="form-image-url-input"
                        />
                    ) : (
                        <Dropzone
                            accept={{
                                "image/*": [".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg"],
                            }}
                            maxFiles={1}
                            onDrop={handleFileUpload}
                            dropzoneText={t("DragDropImageHere")}
                            data-testid="form-image-dropzone"
                        />
                    )}

                    <TextField
                        fullWidth
                        label={t("AltText")}
                        value={element.label || ""}
                        onChange={handleAltTextChange}
                        placeholder={t("DescribeImageForAccessibility")}
                        helperText={t("AltTextHelp")}
                        data-testid="form-image-alt-text"
                        inputProps={{
                            "aria-label": "Alt text for image",
                        }}
                    />
                </FormSettingsSection>
            )}
        </Box>
    );
}

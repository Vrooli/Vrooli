import { Box, Button, TextField, Typography } from "@mui/material";
import QRCode from "react-qr-code";
import { useState, useCallback } from "react";
import { propButtonStyle } from "./styles.js";
import { type FormQrCodeProps } from "./types.js";

export function FormQrCode({
    element,
    isEditing,
    onUpdate,
    onDelete,
}: FormQrCodeProps) {
    const [isEditingQr, setIsEditingQr] = useState(false);
    const [tempData, setTempData] = useState(element.data || "");
    const [tempDescription, setTempDescription] = useState(element.description || "");

    const handleSave = useCallback(() => {
        onUpdate({
            data: tempData,
            description: tempDescription,
        });
        setIsEditingQr(false);
    }, [tempData, tempDescription, onUpdate]);

    const handleCancel = useCallback(() => {
        setTempData(element.data || "");
        setTempDescription(element.description || "");
        setIsEditingQr(false);
    }, [element.data, element.description]);

    const handleStartEdit = useCallback(() => {
        setIsEditingQr(true);
    }, []);

    // In non-editing mode, just display the QR code
    if (!isEditing) {
        return (
            <Box
                data-testid="qr-code-display"
                sx={{
                    display: "flex",
                    flexDirection: "column",
                    alignItems: "center",
                    gap: 2,
                    py: 2,
                }}
            >
                {element.data && (
                    <Box
                        data-testid="qr-code-image"
                        sx={{
                            p: 2,
                            bgcolor: "white",
                            borderRadius: 1,
                        }}
                    >
                        <QRCode
                            value={element.data}
                            size={200}
                            level="M"
                        />
                    </Box>
                )}
                {element.description && (
                    <Typography
                        data-testid="qr-code-description"
                        variant="body2"
                        sx={{ textAlign: "center" }}
                    >
                        {element.description}
                    </Typography>
                )}
            </Box>
        );
    }

    // In editing mode but not actively editing this QR code
    if (!isEditingQr) {
        return (
            <Box
                data-testid="qr-code-edit-container"
                sx={{
                    display: "flex",
                    flexDirection: "column",
                    alignItems: "center",
                    gap: 2,
                    py: 2,
                    position: "relative",
                }}
            >
                <Box sx={{ position: "absolute", top: 8, right: 8, display: "flex", gap: 1 }}>
                    <Button
                        data-testid="qr-code-edit-button"
                        onClick={handleStartEdit}
                        size="small"
                        variant="outlined"
                        aria-label="Edit QR code"
                        sx={propButtonStyle}
                    >
                        Edit
                    </Button>
                    <Button
                        data-testid="qr-code-delete-button"
                        onClick={onDelete}
                        size="small"
                        variant="outlined"
                        color="error"
                        aria-label="Delete QR code"
                        sx={propButtonStyle}
                    >
                        Delete
                    </Button>
                </Box>
                {element.data && (
                    <Box
                        data-testid="qr-code-image"
                        sx={{
                            p: 2,
                            bgcolor: "white",
                            borderRadius: 1,
                        }}
                    >
                        <QRCode
                            value={element.data}
                            size={200}
                            level="M"
                        />
                    </Box>
                )}
                {element.description && (
                    <Typography
                        data-testid="qr-code-description"
                        variant="body2"
                        sx={{ textAlign: "center" }}
                    >
                        {element.description}
                    </Typography>
                )}
            </Box>
        );
    }

    // Actively editing the QR code
    return (
        <Box
            data-testid="qr-code-editing"
            sx={{
                display: "flex",
                flexDirection: "column",
                gap: 2,
                py: 2,
            }}
        >
            <TextField
                data-testid="qr-code-data-input"
                label="QR Code Data"
                value={tempData}
                onChange={(e) => setTempData(e.target.value)}
                fullWidth
                multiline
                rows={2}
                placeholder="URL, text, or data to encode"
                inputProps={{ "aria-label": "QR code data" }}
            />
            <TextField
                data-testid="qr-code-description-input"
                label="Description (optional)"
                value={tempDescription}
                onChange={(e) => setTempDescription(e.target.value)}
                fullWidth
                placeholder="Description for the QR code"
                inputProps={{ "aria-label": "QR code description" }}
            />
            {tempData && (
                <Box
                    data-testid="qr-code-preview"
                    sx={{
                        display: "flex",
                        flexDirection: "column",
                        alignItems: "center",
                        gap: 1,
                    }}
                >
                    <Typography variant="caption">Preview:</Typography>
                    <Box
                        sx={{
                            p: 2,
                            bgcolor: "white",
                            borderRadius: 1,
                        }}
                    >
                        <QRCode
                            value={tempData}
                            size={150}
                            level="M"
                        />
                    </Box>
                </Box>
            )}
            <Box sx={{ display: "flex", gap: 1, justifyContent: "flex-end" }}>
                <Button
                    data-testid="qr-code-cancel-button"
                    variant="outlined"
                    onClick={handleCancel}
                    sx={propButtonStyle}
                >
                    Cancel
                </Button>
                <Button
                    data-testid="qr-code-save-button"
                    variant="contained"
                    onClick={handleSave}
                    sx={propButtonStyle}
                >
                    Save
                </Button>
            </Box>
        </Box>
    );
}

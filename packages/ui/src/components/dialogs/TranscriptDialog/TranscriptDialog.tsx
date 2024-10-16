import { Box, Dialog, DialogContent, DialogTitle, Palette, Typography, useTheme } from "@mui/material";
import { useZIndex } from "hooks/useZIndex";
import { useTranslation } from "react-i18next";
import { TranscriptDialogProps } from "../types";

const waveBarSx = (palette: Palette) => ({
    width: "8px",
    backgroundColor: palette.primary.main,
    animation: "waveAnimation 1s infinite ease-in-out",
    borderRadius: "5px",
    "&:nth-child(2)": { animationDelay: "0.1s" },
    "&:nth-child(3)": { animationDelay: "0.2s" },
    "&:nth-child(4)": { animationDelay: "0.3s" },
    "&:nth-child(5)": { animationDelay: "0.4s" },
});

/**
 * TranscriptDialog - A dialog for displaying the current transcript and listening animation
 */
export const TranscriptDialog = ({
    handleClose,
    isListening,
    showHint,
    transcript,
}: TranscriptDialogProps) => {
    const zIndex = useZIndex(isListening, false, 1000);
    const { t } = useTranslation();
    const { palette } = useTheme();

    return (
        <Dialog
            onClose={handleClose}
            open={isListening}
            sx={{
                zIndex: zIndex + 999,
                "& > .MuiDialog-container": {
                    "& > .MuiPaper-root": {
                        zIndex: zIndex + 999,
                    },
                },
            }}>
            <DialogTitle>{t("Listening")}</DialogTitle>
            <DialogContent>
                {/* Waveform animation */}
                <Box sx={{
                    display: "flex",
                    justifyContent: "center",
                    alignItems: "center",
                    marginBottom: "16px",
                    height: "50px",
                    gap: "8px",
                    "@keyframes waveAnimation": {
                        "0%, 40%, 100%": { height: "20px" },
                        "20%": { height: "50px" },
                    },
                }}>
                    <Box sx={waveBarSx(palette)}></Box>
                    <Box sx={waveBarSx(palette)}></Box>
                    <Box sx={waveBarSx(palette)}></Box>
                    <Box sx={waveBarSx(palette)}></Box>
                    <Box sx={waveBarSx(palette)}></Box>
                </Box>
                {/* Centered transcript */}
                <Typography align="center" variant="h6">
                    {transcript.length > 0 ? transcript : showHint ? t("SpeakClearly") : ""}
                </Typography>
            </DialogContent>
        </Dialog>
    );
};

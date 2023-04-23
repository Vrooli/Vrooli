import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, Dialog, DialogContent, DialogTitle, Typography, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
const waveBarSx = (palette) => ({
    width: "8px",
    backgroundColor: palette.primary.main,
    animation: "waveAnimation 1s infinite ease-in-out",
    borderRadius: "5px",
    "&:nth-child(2)": { animationDelay: "0.1s" },
    "&:nth-child(3)": { animationDelay: "0.2s" },
    "&:nth-child(4)": { animationDelay: "0.3s" },
    "&:nth-child(5)": { animationDelay: "0.4s" },
});
export const TranscriptDialog = ({ handleClose, isListening, transcript, }) => {
    const { t } = useTranslation();
    const { palette } = useTheme();
    return (_jsxs(Dialog, { onClose: handleClose, open: isListening, children: [_jsx(DialogTitle, { children: t("Listening") }), _jsxs(DialogContent, { children: [_jsxs(Box, { sx: {
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
                        }, children: [_jsx(Box, { sx: waveBarSx(palette) }), _jsx(Box, { sx: waveBarSx(palette) }), _jsx(Box, { sx: waveBarSx(palette) }), _jsx(Box, { sx: waveBarSx(palette) }), _jsx(Box, { sx: waveBarSx(palette) })] }), _jsx(Typography, { align: "center", variant: "h6", children: transcript })] })] }));
};
//# sourceMappingURL=TranscriptDialog.js.map
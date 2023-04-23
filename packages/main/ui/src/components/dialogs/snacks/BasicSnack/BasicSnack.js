import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { CloseIcon, ErrorIcon, InfoIcon, SuccessIcon, WarningIcon } from "@local/icons";
import { Box, Button, IconButton, Typography, useTheme } from "@mui/material";
import { useEffect, useMemo, useRef, useState } from "react";
export var SnackSeverity;
(function (SnackSeverity) {
    SnackSeverity["Error"] = "Error";
    SnackSeverity["Info"] = "Info";
    SnackSeverity["Success"] = "Success";
    SnackSeverity["Warning"] = "Warning";
})(SnackSeverity || (SnackSeverity = {}));
const severityStyle = (severity, palette) => {
    let backgroundColor = palette.primary.light;
    let color = palette.primary.contrastText;
    switch (severity) {
        case "Error":
            backgroundColor = palette.error.dark;
            color = palette.error.contrastText;
            break;
        case "Info":
            backgroundColor = palette.info.main;
            color = palette.info.contrastText;
            break;
        case "Success":
            backgroundColor = palette.success.main;
            color = palette.success.contrastText;
            break;
        default:
            backgroundColor = palette.warning.main;
            color = palette.warning.contrastText;
            break;
    }
    return { backgroundColor, color };
};
export const BasicSnack = ({ buttonClicked, buttonText, data, duration, handleClose, id, message, severity, }) => {
    const { palette } = useTheme();
    const [open, setOpen] = useState(true);
    const timeoutRef = useRef(null);
    useEffect(() => {
        if (duration === "persist")
            return;
        timeoutRef.current = setTimeout(() => {
            setOpen(false);
            timeoutRef.current = setTimeout(() => {
                handleClose();
            }, 400);
        }, duration ?? 5000);
        return () => {
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current);
            }
        };
    }, [duration, handleClose]);
    useEffect(() => {
        if (process.env.NODE_ENV === "development" && data) {
            if (severity === "Error")
                console.error("Snack data", data);
            else
                console.info("Snack data", data);
        }
    }, [data, severity]);
    const Icon = useMemo(() => {
        switch (severity) {
            case "Error":
                return ErrorIcon;
            case "Info":
                return InfoIcon;
            case "Success":
                return SuccessIcon;
            default:
                return WarningIcon;
        }
    }, [severity]);
    return (_jsxs(Box, { sx: {
            display: "flex",
            pointerEvents: "auto",
            justifyContent: "space-between",
            alignItems: "center",
            maxWidth: { xs: "100%", sm: "600px" },
            transform: open ? "translateX(0)" : "translateX(-150%)",
            transition: "transform 0.4s ease-in-out",
            padding: "8px 16px",
            borderRadius: 2,
            boxShadow: 8,
            ...severityStyle(severity, palette),
        }, children: [_jsx(Icon, { fill: "white" }), _jsx(Typography, { variant: "body1", sx: { color: "white", marginLeft: "4px" }, children: message }), buttonText && buttonClicked && (_jsx(Button, { variant: "text", sx: { color: "black", marginLeft: "16px", padding: "4px", border: "1px solid black", borderRadius: "8px" }, onClick: buttonClicked, children: buttonText })), _jsx(IconButton, { onClick: handleClose, children: _jsx(CloseIcon, {}) })] }));
};
//# sourceMappingURL=BasicSnack.js.map
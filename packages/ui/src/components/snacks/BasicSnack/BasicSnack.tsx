import { Box, Button, IconButton, Palette, Typography, useTheme } from "@mui/material";
import { CloseIcon, ErrorIcon, InfoIcon, SuccessIcon, WarningIcon } from "icons";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { SvgComponent } from "types";
import { BasicSnackProps } from "../types";

export enum SnackSeverity {
    Error = "Error",
    Info = "Info",
    Success = "Success",
    Warning = "Warning",
}

const iconColor = (severity: SnackSeverity | `${SnackSeverity}` | undefined, palette: Palette) => {
    switch (severity) {
        case "Error":
            return palette.error.dark;
        case "Info":
            return palette.info.main;
        case "Success":
            return palette.success.main;
        case "Warning":
            return palette.warning.main;
        default:
            return palette.primary.light;
    }
};
/**
 * Basic snack item in the snack stack. 
 * Look changes based on severity. 
 * Supports a button with a callback.
 */
export const BasicSnack = ({
    autoHideDuration,
    buttonClicked,
    buttonText,
    data,
    handleClose,
    id,
    message,
    severity,
}: BasicSnackProps) => {
    const { palette } = useTheme();

    const [open, setOpen] = useState<boolean>(true);

    // Timout to close the snack, if not persistent
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);
    const startAutoHideTimeout = useCallback(() => {
        // Certain snack types require manual closing
        if (autoHideDuration === "persist") return;
        timeoutRef.current = setTimeout(() => {
            // First set to close
            setOpen(false);
            // Then start a second timeout to remove from the stack
            timeoutRef.current = setTimeout(() => {
                handleClose();
            }, 400);
        }, autoHideDuration ?? 5000);
    }, [autoHideDuration, handleClose]);
    // Start close timeout automatically
    useEffect(() => {
        startAutoHideTimeout();
        return () => {
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current);
            }
        };
    }, [autoHideDuration, handleClose, startAutoHideTimeout]);
    // Clear timeout when interacting with the snack
    const handleMouseEnter = () => {
        if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
        }
    };
    // Restart timeout when done interacting with the snack
    const handleMouseLeave = () => {
        startAutoHideTimeout();
    };

    useEffect(() => {
        // Log snack errors if in development
        if (import.meta.env.DEV && data) {
            if (severity === "Error") console.error("Snack data", data);
            else console.info("Snack data", data);
        }
    }, [data, severity]);

    const Icon = useMemo<SvgComponent>(() => {
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

    return (
        <Box
            onMouseEnter={handleMouseEnter}
            onMouseLeave={handleMouseLeave}
            sx={{
                display: "flex",
                pointerEvents: "auto",
                justifyContent: "space-between",
                alignItems: "center",
                maxWidth: { xs: "100%", sm: "600px" },
                // Scrolls out of view when closed
                transform: open ? "translateX(0)" : "translateX(-150%)",
                transition: "transform 0.4s ease-in-out",
                padding: 1,
                borderRadius: 2,
                boxShadow: 8,
                background: palette.background.paper,
                color: palette.background.textPrimary,
            }}>
            {/* Icon */}
            <Icon fill={iconColor(severity, palette)} />
            {/* Message */}
            <Box sx={{
                flex: 1, // take up available space
                marginLeft: "4px",
                maxHeight: "25vh",
                overflowY: "auto",
            }}>
                <Typography
                    variant="body1"
                    sx={{
                        marginLeft: "4px",
                        overflowWrap: "break-word",
                        wordWrap: "anywhere",
                    }}>
                    {message}
                </Typography>
            </Box>
            {/* Button */}
            {buttonText && buttonClicked && (
                <Button
                    variant="text"
                    sx={{
                        color: palette.secondary.main,
                        marginLeft: "16px",
                        padding: "4px",
                        border: `1px solid ${palette.secondary.main}`,
                        borderRadius: "8px",
                    }}
                    onClick={buttonClicked}
                >
                    {buttonText}
                </Button>
            )}
            {/* Close icon */}
            <IconButton onClick={handleClose}>
                <CloseIcon fill={palette.background.textPrimary} />
            </IconButton>
        </Box>
    );
};

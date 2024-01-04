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

    // States to track touch positions
    const [touchStart, setTouchStart] = useState<number | null>(null);
    const [touchPosition, setTouchPosition] = useState<number | null>(null);

    // Ref for the snack's container
    const snackRef = useRef<HTMLDivElement>(null);

    // Handle the start of a touch
    const handleTouchStart = useCallback((e: TouchEvent) => {
        setTouchStart(e.targetTouches[0].clientX);
    }, []);

    // Handle the touch movement
    const handleTouchMove = useCallback((e: TouchEvent) => {
        if (touchStart === null) return;
        const moveDistance = e.targetTouches[0].clientX - touchStart;
        if (Math.abs(moveDistance) > 10) {  // Threshold to determine a swipe
            e.preventDefault();   // Prevent scrolling other elements, like the page
        }
        setTouchPosition(e.targetTouches[0].clientX - touchStart);
    }, [touchStart]);

    // Handle the end of a touch
    const handleTouchEnd = useCallback(() => {
        // Define the minimum swipe distance
        const minSwipeDistance = 50;
        if (touchPosition && Math.abs(touchPosition) > minSwipeDistance) {
            // Close the snack if swiped far enough
            setOpen(false);
            setTimeout(() => handleClose(), 400);
        } else {
            // Reset if not swiped far enough
            setTouchPosition(null);
        }
        setTouchStart(null);
    }, [touchPosition, handleClose]);

    // Side effects to handle the touch events
    useEffect(() => {
        const snackElement = snackRef.current;
        if (snackElement) {
            snackElement.addEventListener('touchstart', handleTouchStart);
            snackElement.addEventListener('touchmove', handleTouchMove);
            snackElement.addEventListener('touchend', handleTouchEnd);
        }
        return () => {
            if (snackElement) {
                snackElement.removeEventListener('touchstart', handleTouchStart);
                snackElement.removeEventListener('touchmove', handleTouchMove);
                snackElement.removeEventListener('touchend', handleTouchEnd);
            }
        };
    }, [handleTouchStart, handleTouchMove, handleTouchEnd]);

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
        if (process.env.DEV && data) {
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
            ref={snackRef}
            onMouseEnter={handleMouseEnter}
            onMouseLeave={handleMouseLeave}
            sx={{
                display: "flex",
                pointerEvents: "auto",
                justifyContent: "space-between",
                alignItems: "center",
                maxWidth: { xs: "100%", sm: "600px" },
                transform: open
                    ? touchPosition !== null
                        ? `translateX(${touchPosition}px)` // Swipe-to-close functionality
                        : "translateX(0)" // Original position when open
                    : "translateX(-150%)", // Slide out of view when closed
                transition: touchPosition ? "none" : "transform 0.4s ease-in-out",
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

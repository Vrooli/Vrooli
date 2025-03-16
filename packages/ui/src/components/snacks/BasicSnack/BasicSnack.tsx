import { Box, BoxProps, Button, IconButton, Palette, Typography, styled, useTheme } from "@mui/material";
import { CloseIcon, CopyIcon, ErrorIcon, InfoIcon, SuccessIcon, WarningIcon } from "icons/common.js";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { SvgComponent } from "types";
import { SNACK_HIGHLIGHT, addHighlight, removeHighlights } from "utils/display/documentTools";
import { PubSub } from "utils/pubsub.js";
import { BasicSnackProps } from "../types.js";

export enum SnackSeverity {
    Error = "Error",
    Info = "Info",
    Success = "Success",
    Warning = "Warning",
}

const DEFAULT_AUTO_HIDE_DURATION_MS = 5000;
const SWIPE_THRESHOLD_PX = 10;

function iconColor(severity: SnackSeverity | `${SnackSeverity}` | undefined, palette: Palette) {
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
}

interface OuterBoxProps extends BoxProps {
    isOpen: boolean;
    touchPosition: number | null;
}
const OuterBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isOpen" && prop !== "touchPosition",
})<OuterBoxProps>(({ isOpen, theme, touchPosition }) => ({
    display: "flex",
    pointerEvents: "auto",
    justifyContent: "space-between",
    alignItems: "center",
    transform: isOpen
        ? touchPosition !== null
            ? `translateX(${touchPosition}px)` // Swipe-to-close functionality
            : "translateX(0)" // Original position when open
        : "translateX(-150%)", // Slide out of view when closed
    transition: touchPosition ? "none" : "transform 0.4s ease-in-out",
    padding: theme.spacing(1),
    borderRadius: theme.shape.borderRadius,
    boxShadow: theme.shadows[8],
    background: theme.palette.background.paper,
    color: theme.palette.background.textPrimary,
    maxWidth: "600px",
    [theme.breakpoints.down("sm")]: {
        maxWidth: `calc(100% - ${theme.spacing(1)})`,
    },
}));

const MessageBox = styled(Box)(() => ({
    flex: 1, // take up available space
    marginLeft: "4px",
    maxHeight: "25vh",
    overflowY: "auto",
}));

const MessageText = styled(Typography)(() => ({
    marginLeft: "4px",
    overflowWrap: "break-word",
    wordWrap: "break-word",
}));

const ActionButton = styled(Button)(({ theme }) => ({
    color: theme.palette.secondary.main,
    marginLeft: "16px",
    padding: "4px",
    border: `1px solid ${theme.palette.secondary.main}`,
    borderRadius: "8px",
}));

/**
 * Basic snack item in the snack stack. 
 * Look changes based on severity. 
 * Supports a button with a callback.
 */
export function BasicSnack({
    autoHideDuration,
    buttonClicked,
    buttonText,
    data,
    handleClose,
    id,
    message,
    severity,
}: BasicSnackProps) {
    const { palette } = useTheme();
    const [open, setOpen] = useState<boolean>(true);
    const [isHovered, setIsHovered] = useState<boolean>(false);

    // States to track touch positions
    const [touchStart, setTouchStart] = useState<number | null>(null);
    const [touchPosition, setTouchPosition] = useState<number | null>(null);

    // Ref for the snack's container
    const snackRef = useRef<HTMLDivElement>(null);

    // Handle the start of a touch
    const handleTouchStart = useCallback(function handleTouchStartCallback(e: TouchEvent) {
        setTouchStart(e.targetTouches[0].clientX);
    }, []);

    // Handle the touch movement
    const handleTouchMove = useCallback(function handleTouchMoveCallback(e: TouchEvent) {
        if (touchStart === null) return;
        const moveDistance = e.targetTouches[0].clientX - touchStart;
        if (Math.abs(moveDistance) > SWIPE_THRESHOLD_PX) {  // Threshold to determine a swipe
            e.preventDefault();   // Prevent scrolling other elements, like the page
        }
        setTouchPosition(e.targetTouches[0].clientX - touchStart);
    }, [touchStart]);

    // Handle the end of a touch
    const handleTouchEnd = useCallback(function handleTouchEndCallback() {
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
    useEffect(function touchEventListenersEffect() {
        const snackElement = snackRef.current;
        if (snackElement) {
            snackElement.addEventListener("touchstart", handleTouchStart);
            snackElement.addEventListener("touchmove", handleTouchMove);
            snackElement.addEventListener("touchend", handleTouchEnd);
        }
        return () => {
            if (snackElement) {
                snackElement.removeEventListener("touchstart", handleTouchStart);
                snackElement.removeEventListener("touchmove", handleTouchMove);
                snackElement.removeEventListener("touchend", handleTouchEnd);
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
        }, autoHideDuration ?? DEFAULT_AUTO_HIDE_DURATION_MS);
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

    const copyMessage = useCallback(function copyMessageCallback() {
        navigator.clipboard.writeText(message ?? "");
        PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
    }, [message]);

    // Clear timeout when interacting with the snack
    function handleMouseEnter() {
        if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
        }
        if (snackRef.current) {
            addHighlight(SNACK_HIGHLIGHT, snackRef.current);
            setIsHovered(true);
        }
    }

    // Restart timeout when done interacting with the snack
    function handleMouseLeave() {
        startAutoHideTimeout();
        if (snackRef.current) {
            removeHighlights(SNACK_HIGHLIGHT, snackRef.current);
            setIsHovered(false);
        }
    }

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
        <OuterBox
            isOpen={open}
            onMouseEnter={handleMouseEnter}
            onMouseLeave={handleMouseLeave}
            ref={snackRef}
            touchPosition={touchPosition}
        >
            {isHovered ? (
                <IconButton onClick={copyMessage}>
                    <CopyIcon fill={palette.secondary.main} />
                </IconButton>
            ) : (
                <Box width="40px" height="40px" display="flex" justifyContent="center" alignItems="center">
                    <Icon fill={iconColor(severity, palette)} width="24px" height="24px" />
                </Box>
            )}
            <MessageBox>
                <MessageText variant="body1">
                    {message}
                </MessageText>
            </MessageBox>
            {buttonText && buttonClicked && (
                <ActionButton
                    variant="text"
                    onClick={buttonClicked}
                >
                    {buttonText}
                </ActionButton>
            )}
            <IconButton onClick={handleClose}>
                <CloseIcon fill={palette.background.textPrimary} />
            </IconButton>
        </OuterBox>
    );
}

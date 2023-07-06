import { Box, ClickAwayListener, Popper, useTheme } from "@mui/material";
import { useCallback, useEffect, useRef, useState } from "react";
import { PopoverWithArrowProps } from "../types";

/**
 * Popover with arrow on the bottom
 */
export const PopoverWithArrow = ({
    anchorEl,
    children,
    disableScrollLock = false,
    handleClose,
    sxs,
    ...props
}: PopoverWithArrowProps) => {
    const { palette } = useTheme();
    const isOpen = Boolean(anchorEl);
    const [canTouch, setCanTouch] = useState(false);

    // Timeout to prevent pressing the popover when it was just opened. 
    // This is useful for mobile devices which can trigger multiple events on press.
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);
    useEffect(() => {
        const stopTimeout = () => {
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current);
                timeoutRef.current = null;
                setCanTouch(false);
            }
        };
        if (isOpen) {
            timeoutRef.current = setTimeout(() => {
                timeoutRef.current = null;
                setCanTouch(true);
            }, 250);
        }
        else { stopTimeout(); }
        return () => { stopTimeout(); };
    }, [isOpen]);

    const onClose = useCallback(() => {
        if (handleClose) {
            setCanTouch(false);
            handleClose();
        }
    }, [handleClose]);

    const handleEscape = useCallback((event) => {
        if (event.key === "Escape") {
            onClose();
        }
    }, [onClose]);
    useEffect(() => {
        if (isOpen) {
            document.addEventListener("keydown", handleEscape);
        } else {
            document.removeEventListener("keydown", handleEscape);
        }
        return () => {
            document.removeEventListener("keydown", handleEscape);
        };
    }, [isOpen, handleEscape]);

    return (
        <Popper
            {...props}
            open={isOpen}
            anchorEl={anchorEl}
            placement="top"
        >
            <ClickAwayListener onClickAway={onClose}>
                <Box sx={{ paddingBottom: "10px" }}>
                    <Box sx={{
                        ...(sxs?.content ?? {}),
                        overflow: "auto",
                        padding: 1,
                        minWidth: "50px",
                        minHeight: "25px",
                        // Disable touch while timeout is active
                        pointerEvents: canTouch ? "auto" : "none",
                        background: palette.background.paper,
                        boxShadow: 12,
                        borderRadius: 2,
                        ...(sxs?.paper ?? {}),
                    }}>
                        {children}
                    </Box>
                    {/* Triangle placed below popper */}
                    <Box sx={{
                        width: "0",
                        height: "0",
                        borderLeft: "10px solid transparent",
                        borderRight: "10px solid transparent",
                        borderTop: `10px solid ${palette.background.paper}`,
                        position: "absolute",
                        bottom: "-9px",
                        left: "50%",
                        transform: "translateX(-50%)",
                        marginBottom: "10px",
                    }} />
                </Box>
            </ClickAwayListener>
        </Popper>
    );
};

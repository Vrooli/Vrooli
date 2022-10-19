import { Box, Popover, useTheme } from "@mui/material";
import { useCallback, useEffect, useRef, useState } from "react";
import { PopoverWithArrowProps } from "../types";

/**
 * Popover with arrow on the bottom
 */
export const PopoverWithArrow = ({
    anchorEl,
    children,
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
        }
        if (isOpen) {
            timeoutRef.current = setTimeout(() => {
                timeoutRef.current = null;
                setCanTouch(true);
            }, 250);
        }
        else { stopTimeout(); }
        return () => { stopTimeout(); }
    }, [isOpen]);

    const onClose = useCallback(() => {
        setCanTouch(false);
        handleClose();
    }, [handleClose]);

    return (
        <Popover
            {...props}
            open={isOpen}
            anchorEl={anchorEl}
            onClose={onClose}
            anchorOrigin={{
                vertical: 'top',
                horizontal: 'center',
            }}
            transformOrigin={{
                vertical: 'bottom',
                horizontal: 'center',
            }}
            sx={{
                ...(sxs?.root ?? {}),
                '& .MuiPopover-paper': {
                    ...(sxs?.root?.['& .MuiPopover-paper'] ?? {}),
                    padding: 1,
                    overflow: 'unset',
                    minWidth: '50px',
                    minHeight: '25px',
                    background: palette.background.paper,
                    color: palette.background.textPrimary,
                    boxShadow: 12,
                }
            }}
        >
            <Box sx={{
                ...(sxs?.content ?? {}),
                overflow: 'auto',
                // Disable touch while timeout is active
                pointerEvents: canTouch ? 'auto' : 'none',
            }}>
                {children}
            </Box>
            {/* Triangle placed below popper */}
            <Box sx={{
                width: '0',
                height: '0',
                borderLeft: '10px solid transparent',
                borderRight: '10px solid transparent',
                borderTop: `10px solid ${palette.background.paper}`,
                position: 'absolute',
                bottom: '-10px',
                left: '50%',
                transform: 'translateX(-50%)',
            }} />
        </Popover>
    )
}
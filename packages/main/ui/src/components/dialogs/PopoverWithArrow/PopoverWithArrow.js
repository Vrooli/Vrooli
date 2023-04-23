import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, Popover, useTheme } from "@mui/material";
import { useCallback, useEffect, useRef, useState } from "react";
export const PopoverWithArrow = ({ anchorEl, children, handleClose, sxs, ...props }) => {
    const { palette } = useTheme();
    const isOpen = Boolean(anchorEl);
    const [canTouch, setCanTouch] = useState(false);
    const timeoutRef = useRef(null);
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
        else {
            stopTimeout();
        }
        return () => { stopTimeout(); };
    }, [isOpen]);
    const onClose = useCallback(() => {
        setCanTouch(false);
        handleClose();
    }, [handleClose]);
    return (_jsxs(Popover, { ...props, open: isOpen, anchorEl: anchorEl, onClose: onClose, anchorOrigin: {
            vertical: "top",
            horizontal: "center",
        }, transformOrigin: {
            vertical: "bottom",
            horizontal: "center",
        }, sx: {
            ...(sxs?.root ?? {}),
            "& .MuiPopover-paper": {
                ...(sxs?.root?.["& .MuiPopover-paper"] ?? {}),
                padding: 1,
                overflow: "unset",
                minWidth: "50px",
                minHeight: "25px",
                background: palette.background.paper,
                color: palette.background.textPrimary,
                boxShadow: 12,
            },
        }, children: [_jsx(Box, { sx: {
                    ...(sxs?.content ?? {}),
                    overflow: "auto",
                    pointerEvents: canTouch ? "auto" : "none",
                }, children: children }), _jsx(Box, { sx: {
                    width: "0",
                    height: "0",
                    borderLeft: "10px solid transparent",
                    borderRight: "10px solid transparent",
                    borderTop: `10px solid ${palette.background.paper}`,
                    position: "absolute",
                    bottom: "-10px",
                    left: "50%",
                    transform: "translateX(-50%)",
                } })] }));
};
//# sourceMappingURL=PopoverWithArrow.js.map
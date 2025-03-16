import { Box, ClickAwayListener, Palette, Popper, PopperPlacementType, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useHotkeys } from "../../../hooks/useHotkeys.js";
import { Z_INDEX } from "../../../utils/consts.js";
import { PopoverWithArrowProps } from "../types.js";

/** Size of arrow pointing to anchor */
const ARROW_SIZE = 10;
const TIMEOUT_TO_PREVENT_TOUCH_MULTIPLE_EVENTS_MS = 250;

type Direction = "top" | "bottom" | "left" | "right";

/** Style for each arrow direction */
const ArrowStyles: Record<Direction, (palette: Palette) => React.CSSProperties> = {
    top: (palette: Palette) => ({
        borderLeft: `${ARROW_SIZE}px solid transparent`,
        borderRight: `${ARROW_SIZE}px solid transparent`,
        borderTop: `${ARROW_SIZE}px solid ${palette.background.paper}`,
        bottom: "0px",
    }),
    bottom: (palette: Palette) => ({
        borderLeft: `${ARROW_SIZE}px solid transparent`,
        borderRight: `${ARROW_SIZE}px solid transparent`,
        borderBottom: `${ARROW_SIZE}px solid ${palette.background.paper}`,
        top: "0px",
    }),
    left: (palette: Palette) => ({
        borderTop: `${ARROW_SIZE}px solid transparent`,
        borderBottom: `${ARROW_SIZE}px solid transparent`,
        borderLeft: `${ARROW_SIZE}px solid ${palette.background.paper}`,
        right: "0px",
    }),
    right: (palette: Palette) => ({
        borderTop: `${ARROW_SIZE}px solid transparent`,
        borderBottom: `${ARROW_SIZE}px solid transparent`,
        borderRight: `${ARROW_SIZE}px solid ${palette.background.paper}`,
        left: "0px",
    }),
};

export function PopoverWithArrow({
    anchorEl,
    children,
    handleClose,
    placement = "auto",
    sxs,
    ...props
}: PopoverWithArrowProps) {
    const { palette } = useTheme();
    const isOpen = Boolean(anchorEl);
    const [canTouch, setCanTouch] = useState(false);
    const [actualPlacement, setActualPlacement] = useState<Direction | null>(null);

    const handlePopperState = useCallback((popperState: { placement: PopperPlacementType }) => {
        if (popperState) {
            const { placement } = popperState;
            // Limit to top, bottom, left, right. Avoid corners.
            const newPlacement = placement.split("-")[0] as Direction;
            setActualPlacement(newPlacement);
        }
    }, []);

    // Timeout to prevent pressing the popover when it was just opened. 
    // This is useful for mobile devices which can trigger multiple events on press.
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);
    useEffect(() => {
        function stopTimeout() {
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
            }, TIMEOUT_TO_PREVENT_TOUCH_MULTIPLE_EVENTS_MS);
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

    useHotkeys([{ keys: ["Escape"], callback: onClose }], isOpen);

    const arrowStyle = useMemo(function arrowStyleMemo() {
        if (!actualPlacement || !ArrowStyles[actualPlacement]) {
            return {};
        }
        return ArrowStyles[actualPlacement](palette);
    }, [actualPlacement, palette]);

    const popoverOptions = useMemo(function popoverOptionsMemo() {
        return {
            modifiers: [
                { name: "arrow", options: { element: "[data-popper-arrow]" } },
                {
                    name: "flip",
                    options: {
                        altBoundary: true,
                        fallbackPlacements: ["top", "right", "bottom", "left"],
                    },
                },
                {
                    name: "preventOverflow",
                    options: {
                        altAxis: true,
                        tether: true,
                        boundary: "viewport",
                    },
                },
                { name: "onUpdate", enabled: true, phase: "write", fn: ({ state }) => handlePopperState(state) },
            ],
        };
    }, [handlePopperState]);

    const popoverStyle = useMemo(function popoverStyleMemo() {
        return {
            zIndex: Z_INDEX.Popup,
            ...sxs?.root,
        } as const;
    }, [sxs?.root]);

    const innerBoxStyle = useMemo(function innerBoxStyleMemo() {
        return {
            ...(sxs?.paper ?? {}),
        } as const;
    }, [sxs?.paper]);

    const childrenWrapperStyle = useMemo(function childrenWrapperStyleMemo() {
        return {
            overflow: "auto",
            padding: 1,
            margin: `${ARROW_SIZE}px`,
            minWidth: "50px",
            minHeight: "25px",
            // Disable touch while timeout is active
            pointerEvents: canTouch ? "auto" : "none",
            background: palette.background.paper,
            color: palette.background.textPrimary,
            boxShadow: 12,
            borderRadius: 2,
            ...(sxs?.content ?? {}),
        } as const;
    }, [canTouch, palette.background.paper, palette.background.textPrimary, sxs?.content]);

    const triangleStyle = useMemo(function triangleStyleMemo() {
        return {
            width: "0",
            height: "0",
            ...arrowStyle,
            position: "absolute",
            transform: actualPlacement === "top" || actualPlacement === "bottom" ? "translateX(-50%)" : "translateY(-50%)",
        } as const;
    }, [actualPlacement, arrowStyle]);

    return (
        <Popper
            {...props}
            open={isOpen}
            anchorEl={anchorEl}
            placement={placement}
            popperOptions={popoverOptions}
            style={popoverStyle}
        >
            <ClickAwayListener onClickAway={onClose}>
                <Box sx={innerBoxStyle}>
                    <Box sx={childrenWrapperStyle}>
                        {children}
                    </Box>
                    <Box data-popper-arrow sx={triangleStyle} />
                </Box>
            </ClickAwayListener>
        </Popper>
    );
}

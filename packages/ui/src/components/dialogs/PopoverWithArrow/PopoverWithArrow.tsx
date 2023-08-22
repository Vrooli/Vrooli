import { Box, ClickAwayListener, Palette, Popper, PopperPlacementType, useTheme } from "@mui/material";
import { useHotkeys } from "hooks/useHotkeys";
import { useZIndex } from "hooks/useZIndex";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { PopoverWithArrowProps } from "../types";

type Direction = "top" | "bottom" | "left" | "right";

/** Style for each arrow direction */
const ArrowStyles: Record<Direction, (palette: Palette) => React.CSSProperties> = {
    top: (palette: Palette) => ({
        borderLeft: "10px solid transparent",
        borderRight: "10px solid transparent",
        borderTop: `10px solid ${palette.background.paper}`,
        bottom: "-10px",
    }),
    bottom: (palette: Palette) => ({
        borderLeft: "10px solid transparent",
        borderRight: "10px solid transparent",
        borderBottom: `10px solid ${palette.background.paper}`,
        top: "-10px",
    }),
    left: (palette: Palette) => ({
        borderTop: "10px solid transparent",
        borderBottom: "10px solid transparent",
        borderLeft: `10px solid ${palette.background.paper}`,
        right: "-10px",
    }),
    right: (palette: Palette) => ({
        borderTop: "10px solid transparent",
        borderBottom: "10px solid transparent",
        borderRight: `10px solid ${palette.background.paper}`,
        left: "-10px",
    }),
};

const getOffsetModifier = (placement: string) => {
    switch (placement) {
        case "top":
        case "bottom":
            return { offset: [0, 10] };  // Adjust the Y-offset
        case "left":
        case "right":
            return { offset: [10, 0] };  // Adjust the X-offset
        default:
            return { offset: [0, 0] };
    }
};

export const PopoverWithArrow = ({
    anchorEl,
    children,
    disableScrollLock = false,
    handleClose,
    placement = "top",
    sxs,
    ...props
}: PopoverWithArrowProps) => {
    const { palette } = useTheme();
    const isOpen = Boolean(anchorEl);
    const zIndex = useZIndex(isOpen);
    const [canTouch, setCanTouch] = useState(false);
    const [actualPlacement, setActualPlacement] = useState(placement);

    const handlePopperState = useCallback((popperState: { placement: PopperPlacementType }) => {
        if (popperState) {
            const { placement } = popperState;
            // Limit to top, bottom, left, right
            setActualPlacement(placement.split("-")[0] as "top" | "bottom" | "left" | "right");
        }
    }, []);

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

    useHotkeys([{ keys: ["Escape"], callback: onClose }], isOpen);

    const arrowStyle = useMemo(() => ArrowStyles[actualPlacement](palette), [actualPlacement, palette]);
    const offsetModifier = getOffsetModifier(actualPlacement);

    return (
        <Popper
            {...props}
            open={isOpen}
            anchorEl={anchorEl}
            placement={placement}
            popperOptions={{
                modifiers: [
                    { name: "flip", options: { altBoundary: true, fallbackPlacements: ["top", "right", "bottom", "left"] } },
                    { name: "preventOverflow", options: { altAxis: true, tether: false, padding: 10, boundary: "viewport" } },
                    { name: "onUpdate", enabled: true, phase: "write", fn: ({ state }) => handlePopperState(state) },
                    { name: "offset", options: offsetModifier },
                    { name: "arrow", options: { element: "[data-popper-arrow]" } },
                ],
            }}
            style={{
                zIndex: zIndex + 1000,
                ...sxs?.root,
            }}
        >
            <ClickAwayListener onClickAway={onClose}>
                <Box sx={{
                    paddingTop: actualPlacement === "bottom" ? "10px" : undefined,
                    paddingBottom: actualPlacement === "top" ? "10px" : undefined,
                    paddingLeft: actualPlacement === "right" ? "10px" : undefined,
                    paddingRight: actualPlacement === "left" ? "10px" : undefined,
                    ...(sxs?.paper ?? {}),
                }}>
                    <Box sx={{
                        overflow: "auto",
                        padding: 1,
                        minWidth: "50px",
                        minHeight: "25px",
                        // Disable touch while timeout is active
                        pointerEvents: canTouch ? "auto" : "none",
                        background: palette.background.paper,
                        boxShadow: 12,
                        borderRadius: 2,
                        ...(sxs?.content ?? {}),
                    }}>
                        {children}
                    </Box>
                    {/* Triangle placed accordingly to the popper */}
                    <Box data-popper-arrow sx={{
                        width: "0",
                        height: "0",
                        ...arrowStyle,
                        position: "absolute",
                        margin: "10px",
                        transform: actualPlacement === "top" || actualPlacement === "bottom" ? "translateX(-50%)" : "translateY(-50%)",
                    }} />
                </Box>
            </ClickAwayListener>
        </Popper>
    );
};

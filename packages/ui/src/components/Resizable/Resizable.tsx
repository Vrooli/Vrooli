import { Box, useTheme } from "@mui/material";
import React, { createContext, useContext, useEffect, useMemo, useRef, useState } from "react";
import { SxType } from "types";
import { getCookieDimensions, setCookieDimensions } from "utils/cookies";

type Position = "top" | "bottom" | "left" | "right";
type Dimensions = { width: number, height: number };

const DimensionContext = createContext<Dimensions | undefined>(undefined);

export const useDimensionContext = () => {
    const context = useContext(DimensionContext);
    if (!context) {
        throw new Error("useDimensionContext must be used within a DimensionProvider");
    }
    return context;
};

const size = "10px";
const styles: Record<Position, React.CSSProperties> = {
    top: {
        top: 0,
        left: 0,
        right: 0,
        height: size,
        cursor: "ns-resize",
    },
    bottom: {
        bottom: 0,
        left: 0,
        right: 0,
        height: size,
        cursor: "ns-resize",
    },
    left: {
        top: 0,
        left: 0,
        bottom: 0,
        width: size,
        cursor: "ew-resize",
    },
    right: {
        top: 0,
        right: 0,
        bottom: 0,
        width: size,
        cursor: "ew-resize",
    },
};

//TODO Need better min/max. Should be able to set as percentage of screen size in addition to px
const DEFAULT_SIZE = 150;
const areDimensionsValid = (
    dimensions: Dimensions | null | undefined,
    position: Position,
    max?: number,
    min?: number,
) => {
    if (!dimensions) return false;
    const limit = (["top", "bottom"].includes(position)) ? dimensions.height : dimensions.width;
    const upperBound = (["top", "bottom"].includes(position)) ? window.innerHeight : window.innerWidth;
    return Boolean(limit) && limit >= (min ?? DEFAULT_SIZE) && limit <= (max ?? upperBound);
};
const getDefaultDimensions = (position: Position, max?: number | undefined, min?: number | undefined): Dimensions => {
    const limitHeight = (["top", "bottom"].includes(position)) ?
        Math.min(min ?? DEFAULT_SIZE, max ?? window.innerHeight) :
        window.innerHeight;
    const limitWidth = (["left", "right"].includes(position)) ?
        Math.min(min ?? DEFAULT_SIZE, max ?? window.innerWidth) :
        window.innerWidth;
    return { width: limitWidth, height: limitHeight };
};
const adjustDimensions = (dimensions: Dimensions, position: Position, max?: number | undefined, min?: number | undefined): Dimensions => {
    if (["top", "bottom"].includes(position)) {
        if (dimensions.height > (max ?? window.innerHeight)) {
            dimensions.height = max ?? window.innerHeight;
        } else if (dimensions.height < (min ?? DEFAULT_SIZE)) {
            dimensions.height = min ?? DEFAULT_SIZE;
        }
    } else {
        if (dimensions.width > (max ?? window.innerWidth)) {
            dimensions.width = max ?? window.innerWidth;
        } else if (dimensions.width < (min ?? DEFAULT_SIZE)) {
            dimensions.width = min ?? DEFAULT_SIZE;
        }
    }
    return dimensions;
};

const parseDimension = (value?: string | number) => {
    if (typeof value === "number") {
        return value;
    }
    if (value?.endsWith("vh")) {
        const vh = parseFloat(value);
        return window.innerHeight * (vh / 100);
    }
    if (value?.endsWith("vw")) {
        const vw = parseFloat(value);
        return window.innerWidth * (vw / 100);
    }
    if (value?.endsWith("px")) {
        return parseFloat(value);
    }
    return undefined;
};

export const Resizable = ({
    children,
    id,
    max,
    min,
    position = "right",
    sx,
}: {
    children: React.ReactNode,
    id: string,
    max?: string | number,
    min?: string | number,
    position?: Position,
    sx?: SxType,
}) => {
    const { palette } = useTheme();
    const containerRef = useRef<HTMLDivElement | null>(null);

    const minPixels = useMemo(() => parseDimension(min), [min]);
    const maxPixels = useMemo(() => parseDimension(max), [max]);
    const isVerticalResize = ["top", "bottom"].includes(position);
    const storedDimensions = useMemo(() => id ? getCookieDimensions(id) : null, [id]);
    const [dimensions, setDimensions] = useState<Dimensions>(areDimensionsValid(storedDimensions, position, maxPixels, minPixels) ? (storedDimensions as Dimensions) : getDefaultDimensions(position, maxPixels, minPixels));

    // Set initial dimensions
    useEffect(() => {
        // Skip if dimensions already set
        const storedDimensions = id ? getCookieDimensions(id) : null;
        if (storedDimensions?.width && storedDimensions?.height) return;

        if (!containerRef.current) return;
        const rect = containerRef.current.getBoundingClientRect();

        const newDimensions = adjustDimensions({
            width: rect.width,
            height: rect.height,
        }, position, maxPixels, minPixels);
        setDimensions(newDimensions);
        setCookieDimensions(id, newDimensions);
    }, [id, maxPixels, minPixels, position]);

    const handleResize = (event: MouseEvent | TouchEvent) => {
        event.preventDefault();
        if (!containerRef.current) return;
        let isTouchEvent = false;
        let startX: number;
        let startY: number;
        if ("touches" in event) {
            isTouchEvent = true;
            startX = event.touches[0]?.clientX ?? 0;
            startY = event.touches[0]?.clientY ?? 0;
        } else {
            startX = event.clientX;
            startY = event.clientY;
        }
        const { width: startWidth, height: startHeight } = containerRef.current.getBoundingClientRect();
        let newWidth = startWidth;
        let newHeight = startHeight;

        const handleMove = (moveEvent: MouseEvent | TouchEvent) => {
            if (!containerRef.current) return;
            moveEvent.preventDefault();
            let currentX: number;
            let currentY: number;
            if ("touches" in moveEvent) {
                currentX = moveEvent.touches[0]?.clientX ?? 0;
                currentY = moveEvent.touches[0]?.clientY ?? 0;
            } else {
                currentX = moveEvent.clientX;
                currentY = moveEvent.clientY;
            }
            if (isVerticalResize) {
                const diffY = currentY - startY;
                newHeight = position === "bottom" ? startHeight + diffY : startHeight - diffY;
            } else {
                const diffX = currentX - startX;
                newWidth = position === "right" ? startWidth + diffX : startWidth - diffX;
            }
            const newDimensions = adjustDimensions({ width: newWidth, height: newHeight }, position, maxPixels, minPixels);
            if (isVerticalResize) containerRef.current.style.height = `${newDimensions.height}px`;
            else containerRef.current.style.width = `${newDimensions.width}px`;
            setDimensions(newDimensions);
            setCookieDimensions(id, newDimensions);
        };

        const handleEnd = () => {
            document.removeEventListener(isTouchEvent ? "touchmove" : "mousemove", handleMove as EventListener);
            document.removeEventListener(isTouchEvent ? "touchend" : "mouseup", handleEnd);
        };
        document.addEventListener(isTouchEvent ? "touchmove" : "mousemove", handleMove as EventListener);
        document.addEventListener(isTouchEvent ? "touchend" : "mouseup", handleEnd);
    };

    useEffect(() => {
        const handleResize = () => {
            if (!containerRef.current) return;
            const rect = containerRef.current.getBoundingClientRect();
            const newDimensions = adjustDimensions({
                width: rect.width,
                height: rect.height,
            }, position, maxPixels, minPixels);
            setDimensions(newDimensions);
            setCookieDimensions(id, newDimensions);
        };

        window.addEventListener("resize", handleResize);
        return () => {
            window.removeEventListener("resize", handleResize);
        };
    }, [id, maxPixels, minPixels, position]);

    return (
        <Box
            ref={containerRef}
            sx={{
                position: "relative",
                overflow: "hidden",
                resize: "none",
                ...sx,
                ...(["top", "bottom"].includes(position) && { height: dimensions.height ? `${dimensions.height}px` : (sx as Record<string, any>)?.height }),
                ...(["left", "right"].includes(position) && { width: dimensions.width ? `${dimensions.width}px` : (sx as Record<string, any>)?.width }),
            }}
        >
            <DimensionContext.Provider value={dimensions}>
                {children}
            </DimensionContext.Provider>
            <div
                style={{
                    ...styles[position],
                    position: "absolute",
                    background: palette.divider,
                    zIndex: 1000,
                    border: isVerticalResize
                        ? position === "bottom"
                            ? "1px solid rgba(0, 0, 0, 0.3)"
                            : "1px solid rgba(0, 0, 0, 0.3) 0 0"
                        : position === "right"
                            ? "0 1px solid rgba(0, 0, 0, 0.3)"
                            : "0 1px solid rgba(0, 0, 0, 0.3) 0",
                }}
                onMouseDown={(e: React.MouseEvent<HTMLDivElement>) => handleResize(e.nativeEvent)}
                onTouchStart={(e: React.TouchEvent<HTMLDivElement>) => handleResize(e.nativeEvent)}
            ></div>
        </Box>
    );
};
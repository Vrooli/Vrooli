import Box from "@mui/material/Box";
import { keyframes } from "@mui/material";
import type { SxProps, Theme } from "@mui/material";
import { styled } from "@mui/material";
import { useCallback, useEffect, useRef, useState } from "react";
import { useMenu } from "../hooks/useMenu.js";
import { ELEMENT_IDS } from "../utils/consts.js";
import { type MenuPayloads } from "../utils/pubsub.js";
import { type DiagonalWaveLoaderProps } from "./types.js";

const DEFAULT_LOADER_SIZE = 60;
const BOX_WIDTH_IN_CIRCLES = 3;
const CIRCLE_FADE_DELAY_S = 0.3;

const fade = keyframes`
    0%, 40% {
        opacity: 1;
    }
    20% {
        opacity: 0;
    }
`;

interface LoaderProps {
    size?: number;
    sx?: SxProps<Theme>;
}

const Loader = styled(Box, { shouldForwardProp: (prop) => prop !== "size" })<LoaderProps>(({ size }) => ({
    display: "inline-grid",
    gridTemplateColumns: "repeat(3, 1fr)",
    gap: 6,
    width: size || DEFAULT_LOADER_SIZE,
    height: size || DEFAULT_LOADER_SIZE,
    justifyContent: "center",
    alignItems: "center",
}));

interface CircleProps {
    color?: string;
    index: number;
}

const Circle = styled(Box, {
    shouldForwardProp: (prop) => prop !== "color" && prop !== "index",
})<CircleProps>(({ color, index, theme }) => ({
    borderRadius: "50%",
    width: "100%",
    height: "100%",
    backgroundColor: color || theme.palette.primary.main,
    animation: `${fade} 3s ease-in-out infinite`,
    animationDelay: `${(index % BOX_WIDTH_IN_CIRCLES) * CIRCLE_FADE_DELAY_S + Math.floor(index / BOX_WIDTH_IN_CIRCLES) * CIRCLE_FADE_DELAY_S}s`,
}));

export function DiagonalWaveLoader({
    color,
    size,
    sx,
    "data-testid": testId,
}: DiagonalWaveLoaderProps) {
    return (
        <Loader size={size} sx={sx} data-testid={testId || "diagonal-wave-loader"}>
            {Array.from({ length: BOX_WIDTH_IN_CIRCLES * BOX_WIDTH_IN_CIRCLES }, (_, i) => (
                <Circle
                    key={`circle-${i}`} // Using the index as key is generally not recommended, but it's fine here
                    color={color}
                    index={i}
                />
            ))}
        </Loader>
    );
}

const fullPageSpinnerOuterStyle = {
    position: "absolute",
    top: "50%",
    left: "50%",
    transform: "translate(-50%, -50%)",
    zIndex: 100000,
} as const;

export function FullPageSpinner() {
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);
    const [isOpen, setIsOpen] = useState(false);

    useEffect(function resetLoadingStateOnMountEffect() {
        if (timeoutRef.current) clearTimeout(timeoutRef.current);
    }, []);

    const onEvent = useCallback(function onEventCallback({ data }: MenuPayloads[typeof ELEMENT_IDS.FullPageSpinner]) {
        if (timeoutRef.current) clearTimeout(timeoutRef.current);
        const show = data?.show ?? true;
        const delay = data?.delay ?? 0;
        if (!delay) {
            setIsOpen(show);
        } else if (Number.isInteger(delay) && delay > 0) {
            timeoutRef.current = setTimeout(() => setIsOpen(show), delay);
        } else {
            setIsOpen(show);
        }
    }, []);
    const { isOpen: isActive } = useMenu({
        id: ELEMENT_IDS.FullPageSpinner,
        onEvent,
    });
    useEffect(function closeWhenInactiveEffect() {
        if (!isActive) setIsOpen(false);
    }, [isActive]);

    if (!isActive || !isOpen) return null;
    return (
        <Box id={ELEMENT_IDS.FullPageSpinner} sx={fullPageSpinnerOuterStyle}>
            <DiagonalWaveLoader size={100} />
        </Box>
    );
}

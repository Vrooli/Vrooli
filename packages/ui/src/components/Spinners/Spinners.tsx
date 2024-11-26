import { Box, keyframes, styled, SxProps, Theme } from "@mui/material";
import { DiagonalWaveLoaderProps } from "components/types";

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
}: DiagonalWaveLoaderProps) {
    return (
        <Loader size={size} sx={sx} data-testid="diagonal-wave-loader">
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
    return (
        <Box sx={fullPageSpinnerOuterStyle}>
            <DiagonalWaveLoader size={100} />
        </Box>
    );
}

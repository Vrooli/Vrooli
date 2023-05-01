import { Box, keyframes, styled } from "@mui/material";
import { DiagonalWaveLoaderProps } from "components/types";

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
    sx?: any;
}

const Loader = styled(Box, { shouldForwardProp: (prop) => prop !== "size" })<LoaderProps>(({ theme, size }) => ({
    display: "inline-grid",
    gridTemplateColumns: "repeat(3, 1fr)",
    gap: 6,
    width: size || 60,
    height: size || 60,
    justifyContent: "center",
    alignItems: "center",
}));

interface CircleProps {
    color?: string;
}

const Circle = styled(Box, { shouldForwardProp: (prop) => prop !== "color" })<CircleProps>(({ theme, color }) => ({
    borderRadius: "50%",
    width: "100%",
    height: "100%",
    backgroundColor: color || theme.palette.primary.main,
    animation: `${fade} 3s ease-in-out infinite`,
}));

export const DiagonalWaveLoader = ({
    color,
    size,
    sx,
}: DiagonalWaveLoaderProps) => {
    return (
        <Loader size={size} sx={sx}>
            {Array.from({ length: 9 }, (_, i) => (
                <Circle
                    key={i}
                    color={color}
                    style={{ animationDelay: `${(i % 3) * 0.3 + Math.floor(i / 3) * 0.3}s` }}
                />
            ))}
        </Loader>
    );
};

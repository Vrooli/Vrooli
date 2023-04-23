import { jsx as _jsx } from "react/jsx-runtime";
import { Box, keyframes, styled } from "@mui/material";
const fade = keyframes `
    0%, 40% {
        opacity: 1;
    }
    20% {
        opacity: 0;
    }
`;
const Loader = styled(Box, { shouldForwardProp: (prop) => prop !== "size" })(({ theme, size }) => ({
    display: "inline-grid",
    gridTemplateColumns: "repeat(3, 1fr)",
    gap: 6,
    width: size || 60,
    height: size || 60,
    justifyContent: "center",
    alignItems: "center",
}));
const Circle = styled(Box, { shouldForwardProp: (prop) => prop !== "color" })(({ theme, color }) => ({
    borderRadius: "50%",
    width: "100%",
    height: "100%",
    backgroundColor: color || theme.palette.primary.main,
    animation: `${fade} 3s ease-in-out infinite`,
}));
export const DiagonalWaveLoader = ({ color, size, sx, }) => {
    return (_jsx(Loader, { size: size, sx: sx, children: Array.from({ length: 9 }, (_, i) => (_jsx(Circle, { color: color, style: { animationDelay: `${(i % 3) * 0.3 + Math.floor(i / 3) * 0.3}s` } }, i))) }));
};
//# sourceMappingURL=DiagonalWaveLoader.js.map
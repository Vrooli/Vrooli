import { jsx as _jsx } from "react/jsx-runtime";
import { Box, keyframes } from "@mui/material";
import { useMemo } from "react";
const twinkle = keyframes `
    0% {
        opacity: 1;
    }
    50% {
        opacity: 0.5;
    }
    100% {
        opacity: 1;
    }
`;
export const TwinkleStars = ({ amount = 100, size = 4, color = "white", speed = 20, sx = {}, }) => {
    const stars = useMemo(() => [...Array(amount)].map((e, i) => {
        const starSize = Math.max(Math.random() * size, 1);
        const starSpeed = Math.max(Math.random() * speed, 1);
        return (_jsx(Box, { sx: {
                position: "absolute",
                top: `${Math.random() * 100}%`,
                left: `${Math.random() * 100}%`,
                width: `${starSize}px`,
                height: `${starSize}px`,
                background: color,
                borderRadius: "50%",
                animation: `${twinkle} ${starSpeed}s linear infinite`,
                zIndex: 1,
            } }, i));
    }), [amount, size, color, speed]);
    return (_jsx(Box, { sx: {
            position: "fixed",
            top: 0,
            left: 0,
            width: "100vw",
            height: "100vh",
            overflow: "hidden",
            zIndex: 2,
            ...sx,
        }, children: stars }));
};
//# sourceMappingURL=TwinkleStars.js.map
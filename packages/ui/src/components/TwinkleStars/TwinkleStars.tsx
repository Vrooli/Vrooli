import { Box, keyframes } from "@mui/material";
import { TwinklingStarsProps } from "components/types";
import { useMemo } from "react";

// Twinkle animation
const twinkle = keyframes`
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

/**
 * Displays twinkling stars in the background.
 */
export const TwinkleStars = ({
    amount = 100,
    size = 4,
    color = 'white',
    speed = 20,
    sx = {},
}: TwinklingStarsProps) => {

    const stars = useMemo(() => [...Array(amount)].map((e, i) => {
        const starSize = Math.max(Math.random() * size, 1);
        const starSpeed = Math.max(Math.random() * speed, 1);
        return (
            <Box
                key={i}
                sx={{
                    position: 'absolute',
                    top: `${Math.random() * 100}%`,
                    left: `${Math.random() * 100}%`,
                    width: `${starSize}px`,
                    height: `${starSize}px`,
                    background: color,
                    borderRadius: '50%',
                    animation: `${twinkle} ${starSpeed}s linear infinite`,
                    zIndex: 1,
                }}
            />
        )
    }), [amount, size, color, speed]);

    return (
        <Box
            sx={{
                position: 'fixed',
                top: 0,
                left: 0,
                width: '100vw',
                height: '100vh',
                overflow: 'hidden',
                zIndex: 2,
                ...sx,
            }}
        >
            {stars}
        </Box>
    );
};
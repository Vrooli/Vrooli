import { Box, styled } from "@mui/material";
import { useEffect, useRef } from "react";

const STAR_AMOUNT = 800; // Total stars
const STAR_SIZE = 2;

const SpaceBox = styled(Box)(() => ({
    position: "absolute",
    pointerEvents: "none",
    top: 0,
    left: 0,
    width: "100%",
    height: "100%",
    overflow: "hidden",
    zIndex: 4,
    background: "black",
}));

const canvasStyle = { width: "100%", height: "100%" } as const;

export function StarryBackground() {
    const canvasRef = useRef<HTMLCanvasElement>(null);

    useEffect(() => {
        const canvas = canvasRef.current;
        if (canvas) {
            const ctx = canvas.getContext("2d");
            if (ctx) {
                const { width, height } = canvas.getBoundingClientRect();

                canvas.width = width;
                canvas.height = height;

                // Draw random stars
                for (let i = 0; i < STAR_AMOUNT; i++) {
                    const x = Math.random() * width;
                    const y = Math.random() * height;
                    const size = Math.random() * STAR_SIZE;
                    const opacity = Math.random();
                    ctx.beginPath();
                    // eslint-disable-next-line no-magic-numbers
                    ctx.arc(x, y, size, 0, 2 * Math.PI);
                    ctx.fillStyle = `rgba(255, 255, 255, ${opacity})`;
                    ctx.fill();
                }
            }
        }
    }, []);

    return (
        <SpaceBox>
            <canvas ref={canvasRef} style={canvasStyle} />
        </SpaceBox>
    );
}

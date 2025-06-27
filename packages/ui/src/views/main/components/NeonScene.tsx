/* eslint-disable no-magic-numbers */
import Box, { type BoxProps } from "@mui/material/Box";
import { styled } from "@mui/material/styles";
import { useTheme } from "@mui/material/styles";
import { useEffect, useRef } from "react";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { ParticleCanvas } from "./ParticleCanvas.js";
import { RandomBlobs } from "./RandomBlobs.js";

const PAGE_LAYERS = {
    Neon: 0,
};

/** The number of particles. (a number lesser than 1000 is recommended under regular settings) */
const POINT_COUNT_MOBILE = 25;
const POINT_COUNT_DESKTOP = 50;
/** Number of neon blobs on mobile */
const NUM_NEON_BLOBS_MOBILE = 5;
/** Number of neon blobs on desktop */
const NUM_NEON_BLOBS_DESKTOP = 8;

interface NeonBoxProps extends BoxProps {
    isVisible: boolean;
}

const NeonBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isVisible",
})<NeonBoxProps>(({ isVisible }) => ({
    background: "black",
    display: isVisible ? "block" : "none",
    transition: "display 0.5s",
    position: "fixed",
    width: "100%",
    height: "100%",
    zIndex: PAGE_LAYERS.Neon,
}));

interface NeonSceneProps {
    isVisible: boolean;
}

/**
 * Custom background for hero section. 
 * Contains neon blobs and particles
 */
export function NeonScene({
    isVisible,
}: NeonSceneProps) {
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const canvasRef = useRef<HTMLCanvasElement>(null);

    useEffect(function handleCanvasInstance() {
        const canvasInstance = new ParticleCanvas(canvasRef.current, isMobile ? POINT_COUNT_MOBILE : POINT_COUNT_DESKTOP);
        return () => {
            canvasInstance?.destroy();
        };
    }, [isMobile]);

    return (
        <NeonBox isVisible={isVisible} >
            <canvas ref={canvasRef} />
            <RandomBlobs numberOfBlobs={isMobile ? NUM_NEON_BLOBS_MOBILE : NUM_NEON_BLOBS_DESKTOP} />
        </NeonBox>
    );
}

import { Box, keyframes } from "@mui/material";
import Blob1 from "assets/img/blob1.svg";
import Blob2 from "assets/img/blob2.svg";
import { useCallback } from "react";
import Particles from "react-tsparticles";
import { SlideContainer } from "styles";
import { loadFull } from "tsparticles";
import type { Container, Engine } from "tsparticles-engine";
import { SlideContainerNeonProps } from "../types";

const blackRadial = "radial-gradient(circle, rgb(6 6 46) 12%, rgb(1 1 36) 52%, rgb(3 3 20) 80%)";

// Animation for blob1
// Moves up and grows, then moves down to the right and shrinks.
// Then it moves to the left - while continuing to shrink- until it reaches the starting position.
const blob1Animation = keyframes`
    0% {
        transform: translateY(0) scale(0.5);
        filter: hue-rotate(0deg) blur(150px);
    }
    33% {
        transform: translateY(-160px) scale(0.9) rotate(-120deg);
        filter: hue-rotate(30deg) blur(150px);
    }
    66% {
        transform: translate(50px, 0px) scale(0.6) rotate(-200deg);
        filter: hue-rotate(60deg) blur(150px);
    }
    100% {
        transform: translate(0px, 0px) scale(0.5) rotate(0deg);
        filter: hue-rotate(0deg) blur(150px);
    }
`;

// Animation for blob2
// Moves to the right and changes hue, then moves back to the left and turns its original color.
const blob2Animation = keyframes`
    0% {
        transform: translateX(0) scale(1);
        filter: hue-rotate(0deg) blur(50px);
    }
    50% {
        transform: translateX(150px) scale(1.2);
        filter: hue-rotate(-50deg) blur(50px);
    }
    100% {
        transform: translateX(0) scale(1);
        filter: hue-rotate(0deg) blur(50px);
    }
`;

/**
 * Custom slide container for hero section. 
 * Background is black with neon blobs that move around and grow/shrink
 */
export const SlideContainerNeon = ({
    id,
    children,
    show,
    sx,
}: SlideContainerNeonProps) => {

    const particlesInit = useCallback(async (engine: Engine) => {
        await loadFull(engine);
    }, []);

    const particlesLoaded = useCallback((container: Container | undefined) => {
        console.log(container);
        return Promise.resolve();
    }, []);


    return (
        <SlideContainer
            id={id}
            key={id}
            sx={{
                // Set background and color
                background: blackRadial,
                backgroundAttachment: "fixed",
                color: "white",
                ...sx,
            }}
        >
            {/* Constellation */}
            <Box sx={{
                position: "fixed",
                pointerEvents: "none",
                opacity: show === false ? 0 : 0.5,
                transition: "opacity 1s ease-in-out",
                zIndex: 1,
            }}
            >
                <Particles
                    id="tsparticles"
                    init={particlesInit}
                    loaded={particlesLoaded}
                    options={{
                        fullScreen: { enable: true, zIndex: 0 },
                        fpsLimit: 60,
                        interactivity: {
                            events: {
                                onHover: {
                                    enable: true,
                                    mode: "repulse",
                                },
                                resize: true,
                            },
                            modes: {
                                bubble: {
                                    distance: 400,
                                    duration: 2,
                                    opacity: 0.8,
                                    size: 40,
                                },
                                push: {
                                    quantity: 4,
                                },
                                repulse: {
                                    distance: 50,
                                    duration: 5,
                                },
                            },
                        },
                        particles: {
                            color: {
                                value: "#ffffff",
                            },
                            links: {
                                color: "#ffffff",
                                distance: 150,
                                enable: true,
                                opacity: 0.5,
                                width: 1,
                            },
                            collisions: {
                                enable: true,
                            },
                            move: {
                                direction: "none",
                                enable: true,
                                outMode: "bounce",
                                random: false,
                                speed: 0.3,
                                straight: false,
                            },
                            number: {
                                density: {
                                    enable: true,
                                    area: 800,
                                },
                                value: 100,
                            },
                            opacity: {
                                value: 0.5,
                            },
                            shape: {
                                type: "circle",
                            },
                            size: {
                                random: true,
                                value: 5,
                            },
                        },
                        detectRetina: true,
                    }}
                />
            </Box>
            {/* Blob 1 */}
            <Box sx={{
                position: "fixed",
                pointerEvents: "none",
                bottom: -300,
                left: -175,
                width: "100%",
                height: "100%",
                zIndex: 2,
                opacity: show === false ? 0 : 0.5,
                transition: "opacity 1s ease-in-out",
            }}>
                <Box
                    component="img"
                    src={Blob1}
                    alt="Blob 1"
                    sx={{
                        width: "100%",
                        height: "100%",
                        animation: `${blob1Animation} 20s linear infinite`,
                    }}
                />
            </Box>
            {/* Blob 2 */}
            <Box sx={{
                position: "fixed",
                pointerEvents: "none",
                top: -154,
                right: -175,
                width: "100%",
                height: "100%",
                zIndex: 2,
                opacity: show === false ? 0 : 0.5,
                transition: "opacity 1s ease-in-out",
            }}>
                <Box
                    component="img"
                    src={Blob2}
                    alt="Blob 2"
                    sx={{
                        width: "100%",
                        height: "100%",
                        animation: `${blob2Animation} 20s linear infinite`,
                    }}
                />
            </Box>
            {children}
        </SlideContainer>
    );
};

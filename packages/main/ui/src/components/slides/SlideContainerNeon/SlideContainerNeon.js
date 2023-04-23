import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, keyframes } from "@mui/material";
import Particles from "react-tsparticles";
import { SlideContainer } from "../SlideContainer/SlideContainer";
const blackRadial = "radial-gradient(circle, rgb(6 6 46) 12%, rgb(1 1 36) 52%, rgb(3 3 20) 80%)";
const blob1Animation = keyframes `
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
const blob2Animation = keyframes `
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
export const SlideContainerNeon = ({ id, children, show, sx, }) => {
    return (_jsxs(SlideContainer, { id: id, sx: {
            background: blackRadial,
            backgroundAttachment: "fixed",
            color: "white",
            scrollSnapAlign: "start",
            ...sx,
        }, children: [_jsx(Box, { sx: {
                    position: "fixed",
                    pointerEvents: "none",
                    opacity: show === false ? 0 : 0.5,
                    transition: "opacity 1s ease-in-out",
                    zIndex: 1,
                }, children: _jsx(Particles, { id: "tsparticles", canvasClassName: "tsparticles-canvas", options: {
                        fullScreen: { enable: true, zIndex: 0 },
                        fpsLimit: 60,
                        interactivity: {
                            events: {
                                onClick: {
                                    enable: true,
                                    mode: "push",
                                },
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
                    } }) }), _jsx(Box, { sx: {
                    position: "fixed",
                    pointerEvents: "none",
                    bottom: -300,
                    left: -175,
                    width: "100%",
                    height: "100%",
                    zIndex: 2,
                    opacity: show === false ? 0 : 0.5,
                    transition: "opacity 1s ease-in-out",
                }, children: _jsx(Box, { component: "img", src: "assets/img/blob1.svg", alt: "Blob 1", sx: {
                        width: "100%",
                        height: "100%",
                        animation: `${blob1Animation} 20s linear infinite`,
                    } }) }), _jsx(Box, { sx: {
                    position: "fixed",
                    pointerEvents: "none",
                    top: -154,
                    right: -175,
                    width: "100%",
                    height: "100%",
                    zIndex: 2,
                    opacity: show === false ? 0 : 0.5,
                    transition: "opacity 1s ease-in-out",
                }, children: _jsx(Box, { component: "img", src: "assets/img/blob2.svg", alt: "Blob 2", sx: {
                        width: "100%",
                        height: "100%",
                        animation: `${blob2Animation} 20s linear infinite`,
                    } }) }), children] }, id));
};
//# sourceMappingURL=SlideContainerNeon.js.map
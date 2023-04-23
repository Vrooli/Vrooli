import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { LINKS, SOCIALS, WHITE_PAPER_URL } from "@local/consts";
import { ArticleIcon, DiscordIcon, GitHubIcon, PlayIcon, TwitterIcon } from "@local/icons";
import { Box, Grid, IconButton, Stack, Tooltip, Typography } from "@mui/material";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { PulseButton } from "../../../components/buttons/PulseButton/PulseButton";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { SlideContainer, SlideContainerNeon, SlideContent, SlidePage } from "../../../components/slides";
import { TwinkleStars } from "../../../components/TwinkleStars/TwinkleStars";
import { greenNeonText, iconButtonProps, slideImageContainer, slideText, slideTitle, textPop } from "../../../styles";
import { openLink, useLocation } from "../../../utils/route";
const GlossyContainer = ({ children, sx, ...props }) => {
    return (_jsx(Box, { sx: {
            boxShadow: "0px 0px 6px #040505",
            backgroundColor: "rgba(255,255,255,0.4)",
            backdropFilter: "blur(24px)",
            borderRadius: "0.5rem",
            padding: 1,
            height: "100%",
            maxWidth: "500px",
            margin: "auto",
            ...sx,
        }, ...props, children: children }));
};
const slide1Id = "open-source-economy";
const slide2Id = "three-steps";
const slide3Id = "freedom";
const slide4Id = "share";
const slide5Id = "automate";
const slide6Id = "sky-is-limit";
const slide7Id = "get-started";
const slideContentIds = [slide1Id, slide2Id, slide3Id, slide4Id, slide5Id, slide6Id, slide7Id];
export const LandingView = ({ display = "page", }) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const [earthTransform, setEarthTransform] = useState("translate(0%, 100%) scale(1)");
    const [isSkyVisible, setIsSkyVisible] = useState(false);
    const timeoutRef = useRef(null);
    const lasScrollPosRef = useRef(window.pageYOffset || document.documentElement.scrollTop);
    const currScrollPosRef = useRef(window.pageYOffset || document.documentElement.scrollTop);
    const scrollDirectionRef = useRef("down");
    useEffect(() => {
        const onScroll = () => {
            const scrollPos = window.pageYOffset || document.documentElement.scrollTop;
            if (scrollPos !== currScrollPosRef.current) {
                scrollDirectionRef.current = scrollPos > currScrollPosRef.current ? "down" : "up";
                lasScrollPosRef.current = currScrollPosRef.current;
                currScrollPosRef.current = scrollPos;
            }
            const inView = (element) => {
                if (!element)
                    return false;
                const rect = element.getBoundingClientRect();
                const windowHeight = (window.innerHeight || document.documentElement.clientHeight);
                return rect.top < windowHeight / 2;
            };
            const earthHorizonSlide = document.getElementById(slide6Id);
            const earthFullSlide = document.getElementById(slide7Id);
            if (inView(earthFullSlide)) {
                setEarthTransform("translate(25%, 25%) scale(0.8)");
                setIsSkyVisible(true);
            }
            else if (inView(earthHorizonSlide)) {
                setEarthTransform("translate(0%, 69%) scale(1)");
                setIsSkyVisible(true);
            }
            else {
                setEarthTransform("translate(0%, 100%) scale(1)");
                setIsSkyVisible(false);
            }
            if (timeoutRef.current)
                clearTimeout(timeoutRef.current);
            timeoutRef.current = setTimeout(() => {
                const slides = slideContentIds.map(id => document.getElementById(id));
                let minDistance = Infinity;
                let nearestSlide;
                for (const slide of slides) {
                    if (!slide)
                        continue;
                    const rect = slide.getBoundingClientRect();
                    const windowHeight = (window.innerHeight || document.documentElement.clientHeight);
                    const distance = Math.abs(rect.top - windowHeight / 2);
                    if (rect.top < windowHeight / 2 && rect.bottom > windowHeight / 2) {
                        minDistance = 0;
                        nearestSlide = slide;
                    }
                    else if ((scrollDirectionRef.current === "down" && rect.top > 0) ||
                        (scrollDirectionRef.current === "up" && rect.bottom < windowHeight)) {
                        if (distance < minDistance) {
                            minDistance = distance;
                            nearestSlide = slide;
                        }
                    }
                }
                nearestSlide?.scrollIntoView({ behavior: "smooth" });
            }, 350);
        };
        window.addEventListener("scroll", onScroll, { passive: true });
        return () => {
            window.removeEventListener("scroll", onScroll);
            if (timeoutRef.current)
                clearTimeout(timeoutRef.current);
        };
    }, []);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { } }), _jsxs(SlidePage, { id: "landing-slides", sx: { background: "radial-gradient(circle, rgb(6 6 46) 12%, rgb(1 1 36) 52%, rgb(3 3 20) 80%)" }, children: [_jsxs(SlideContainerNeon, { id: "neon-container", show: !isSkyVisible, sx: { zIndex: 3 }, children: [_jsxs(SlideContent, { id: slide1Id, children: [_jsx(Typography, { component: "h1", sx: {
                                            ...slideTitle,
                                            ...greenNeonText,
                                            fontFamily: "Neuropol",
                                            fontWeight: "bold",
                                        }, children: "An Open-Source Economy" }), _jsx(Typography, { component: "h2", variant: "h5", sx: { ...slideText, textAlign: "center" }, children: "We're building the tools to automate the future of work" }), _jsx(PulseButton, { variant: "outlined", color: "secondary", onClick: () => openLink(setLocation, LINKS.Start), startIcon: _jsx(PlayIcon, { fill: '#0fa' }), sx: {
                                            marginLeft: "auto !important",
                                            marginRight: "auto !important",
                                        }, children: "Start" }), _jsxs(Stack, { direction: "row", spacing: 2, display: "flex", justifyContent: "center", alignItems: "center", sx: { paddingTop: 8, zIndex: 3 }, children: [_jsx(Tooltip, { title: "Read the white Paper", placement: "bottom", children: _jsx(IconButton, { onClick: () => openLink(setLocation, WHITE_PAPER_URL), sx: iconButtonProps, children: _jsx(ArticleIcon, { fill: '#0fa' }) }) }), _jsx(Tooltip, { title: "Check out our code", placement: "bottom", children: _jsx(IconButton, { onClick: () => openLink(setLocation, SOCIALS.GitHub), sx: iconButtonProps, children: _jsx(GitHubIcon, { fill: '#0fa' }) }) }), _jsx(Tooltip, { title: "Follow us on Twitter", placement: "bottom", children: _jsx(IconButton, { onClick: () => openLink(setLocation, SOCIALS.Twitter), sx: iconButtonProps, children: _jsx(TwitterIcon, { fill: '#0fa' }) }) }), _jsx(Tooltip, { title: "Join us on Discord", placement: "bottom", children: _jsx(IconButton, { onClick: () => openLink(setLocation, SOCIALS.Discord), sx: iconButtonProps, children: _jsx(DiscordIcon, { fill: '#0fa' }) }) })] })] }), _jsxs(SlideContent, { id: slide2Id, children: [_jsx(Typography, { variant: 'h2', sx: { ...slideTitle }, children: "Three Easy Steps" }), _jsxs(Grid, { container: true, children: [_jsx(Grid, { item: true, xs: 12, md: 4, p: 2, children: _jsxs(GlossyContainer, { children: [_jsx(Typography, { variant: 'h5', mb: 2, children: _jsx("b", { children: "Connect" }) }), _jsxs("ul", { style: { textAlign: "left" }, children: [_jsx("li", { children: "Find or create a routine for anything you want to accomplish" }), _jsx("li", { children: "Fly solo or join an organization" })] })] }) }), _jsx(Grid, { item: true, xs: 12, md: 4, p: 2, children: _jsxs(GlossyContainer, { children: [_jsx(Typography, { variant: 'h5', mb: 2, children: _jsx("b", { children: "Collaborate" }) }), _jsxs("ul", { style: { textAlign: "left" }, children: [_jsx("li", { children: "Build, vote, and give feedback on routines" }), _jsx("li", { children: "Design the ultimate organization" })] })] }) }), _jsx(Grid, { item: true, xs: 12, md: 4, p: 2, children: _jsxs(GlossyContainer, { children: [_jsx(Typography, { variant: 'h5', mb: 2, children: _jsx("b", { children: "Automate" }) }), _jsxs("ul", { style: { textAlign: "left" }, children: [_jsx("li", { children: "Connect to APIs and smart contracts" }), _jsx("li", { children: "Complete complex tasks from a single site" })] })] }) })] }), _jsxs(Typography, { variant: "h3", pt: 2, sx: { ...slideText, textAlign: "center", lineHeight: 2 }, children: ["This combination creates a", _jsx(Box, { sx: {
                                                    display: "inline-block",
                                                    color: "#ffe768",
                                                    filter: "drop-shadow(0 0 1px #ffe768) drop-shadow(0 0 10px #ffe768) drop-shadow(0 0 41px #ffe768)",
                                                    paddingLeft: 2,
                                                    transform: "scale(1.05",
                                                }, children: _jsx("b", { children: "self-improving productivity machine" }) })] })] }), _jsxs(SlideContent, { id: slide3Id, children: [_jsx(Typography, { variant: 'h2', mb: 4, sx: { ...slideTitle }, children: "Take Back Your Freedom" }), _jsxs(Grid, { container: true, spacing: 2, children: [_jsx(Grid, { item: true, xs: 12, sm: 6, margin: "auto", children: _jsxs(Stack, { direction: "column", spacing: 2, children: [_jsx(Typography, { variant: "h5", sx: { ...slideText }, children: "Vrooli empowers you to unlock personal growth, streamline tasks, and leverage your skills effectively." }), _jsx(Typography, { variant: "h5", sx: { ...slideText }, children: "It guides you through organizing your life, acquiring new knowledge, and monetizing your talents." })] }) }), _jsx(Grid, { item: true, xs: 12, sm: 6, sx: { paddingLeft: "0 !important" }, children: _jsx(Box, { sx: { ...slideImageContainer }, children: _jsx(Box, { component: "img", alt: "Showcases the concept of taking back your freedom. Features a woman who looks empowered and in control, looking straight ahead, with triumph in her eyes", src: "assets/img/woman-triumph.jpg", sx: { borderRadius: "32px", objectFit: "cover" } }) }) })] })] }), _jsxs(SlideContent, { id: slide4Id, children: [_jsx(Typography, { variant: "h2", sx: { ...slideTitle }, children: "Sharing is Scaling" }), _jsxs(Grid, { container: true, spacing: 2, children: [_jsx(Grid, { item: true, xs: 12, sm: 6, sx: { paddingLeft: "0 !important" }, children: _jsx(Box, { sx: { ...slideImageContainer }, children: _jsx(Box, { component: "img", alt: "showcases the concept of 'Sharing is Scaling' through the use of robots collaborating to build something.", src: "assets/img/robots-collab.jpg", sx: { borderRadius: "32px", objectFit: "cover" } }) }) }), _jsx(Grid, { item: true, xs: 12, sm: 6, margin: "auto", children: _jsxs(Stack, { direction: "column", spacing: 2, children: [_jsx(Typography, { variant: "h5", sx: { ...slideText }, children: "Utilize existing routines as building blocks to rapidly construct complex processes." }), _jsx(Typography, { variant: "h5", sx: { ...slideText }, children: "Contribute to the community to receive rewards, recognition, and valuable feedback." })] }) })] })] }), _jsxs(SlideContent, { id: slide5Id, children: [_jsx(Typography, { variant: "h2", sx: { ...slideTitle }, children: "Automate With Minimal Effort" }), _jsxs(Grid, { container: true, spacing: 2, children: [_jsx(Grid, { item: true, xs: 12, sm: 6, margin: "auto", children: _jsx(Stack, { direction: "column", spacing: 2, children: _jsx(Typography, { variant: "h5", sx: { ...slideText }, children: "Harness the power of AI to generate routines, streamlining processes and maximizing productivity." }) }) }), _jsx(Grid, { item: true, xs: 12, sm: 6, sx: { paddingLeft: "0 !important" }, children: _jsx(Box, { sx: { ...slideImageContainer }, children: _jsx(Box, { component: "img", alt: "Visually represents the concept of 'Automate With Minimal Effort'. Features a modern living room environment with a clean and minimalistic design. Within this environment, there is a person relaxing on a couch using an AR headset to access automation technology, which is visually represented in the image in a way that conveys ease of use and minimal effort. The person's interactions with a device or interface that is visually distinct from the rest of the environment, such as a touchscreen or a voice-activated assistant.", src: "assets/img/relaxing-couch.jpg", sx: { borderRadius: "32px", objectFit: "cover" } }) }) })] })] })] }), _jsxs(SlideContainer, { id: 'sky-slide', sx: { color: "white", background: "black", zIndex: 4 }, children: [_jsx(TwinkleStars, { amount: 400, sx: {
                                    position: "absolute",
                                    pointerEvents: "none",
                                    top: 0,
                                    left: 0,
                                    width: "100%",
                                    height: "100%",
                                    zIndex: 4,
                                    background: "black",
                                    opacity: isSkyVisible ? 1 : 0,
                                    transition: "opacity 1s ease-in-out",
                                } }), _jsx(Box, { id: "earth", component: "img", src: "assets/img/Earth.svg", alt: "Earth illustration", sx: {
                                    width: "150%",
                                    position: "fixed",
                                    bottom: "0",
                                    left: "-25%",
                                    right: "-25%",
                                    margin: "auto",
                                    maxWidth: "1000px",
                                    maxHeight: "1000px",
                                    transform: earthTransform,
                                    transition: "transform 1.5s ease-in-out",
                                    zIndex: 5,
                                } }), _jsxs(SlideContent, { id: slide6Id, children: [_jsx(Typography, { variant: 'h2', mb: 4, sx: { ...slideTitle, zIndex: 6 }, children: "The Sky is the Limit" }), _jsx(Typography, { variant: "h5", sx: { ...slideText, zIndex: 6 }, children: "Our ultimate goal is to transition the world to a fully automated, post-capitalist economy. Here's how:" }), _jsxs("ul", { style: { textAlign: "left", zIndex: 6 }, children: [_jsx("li", { children: "Foster a decentralized, collaborative AI ecosystem" }), _jsx("li", { children: "Prioritize ethical and socially responsible AI development" }), _jsx("li", { children: "Democratize access to AI-driven automation for all" })] })] }), _jsxs(SlideContent, { id: slide7Id, children: [_jsx(Typography, { variant: "h2", mb: 4, sx: { ...slideTitle, ...textPop, zIndex: 6 }, children: "Ready to Change the World?" }), _jsx(PulseButton, { variant: "outlined", color: "secondary", onClick: () => openLink(setLocation, LINKS.Start), startIcon: _jsx(PlayIcon, { fill: '#0fa' }), sx: {
                                            marginLeft: "auto !important",
                                            marginRight: "auto !important",
                                            zIndex: 6,
                                        }, children: t("Start") })] })] })] })] }));
};
//# sourceMappingURL=LandingView.js.map
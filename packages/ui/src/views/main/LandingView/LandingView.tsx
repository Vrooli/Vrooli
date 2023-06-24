import { ArticleIcon, DiscordIcon, GitHubIcon, LINKS, openLink, PlayIcon, SOCIALS, TwitterIcon, useLocation, WHITE_PAPER_URL } from "@local/shared";
import { Box, BoxProps, Grid, IconButton, Stack, Tooltip, Typography } from "@mui/material";
import Earth from "assets/img/Earth.svg";
import { PulseButton } from "components/buttons/PulseButton/PulseButton";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SlideContainer, SlideContainerNeon, SlideContent, SlidePage } from "components/slides";
import { TwinkleStars } from "components/TwinkleStars/TwinkleStars";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { greenNeonText, iconButtonProps, slideImageContainer, slideText, slideTitle, textPop } from "styles";
import { LandingViewProps } from "../types";

interface GlossyContainerProps extends BoxProps {
    children: React.ReactNode;
    sx?: { [key: string]: any };
}

/**
 * A semi-transparent container with a glossy (blur) effect
 */
const GlossyContainer = ({
    children,
    sx,
    ...props
}: GlossyContainerProps) => {
    return (
        <Box
            sx={{
                boxShadow: "0px 0px 6px #040505",
                backgroundColor: "rgba(255,255,255,0.4)",
                backdropFilter: "blur(24px)",
                borderRadius: "0.5rem",
                padding: 1,
                height: "100%",
                maxWidth: "500px",
                margin: "auto",
                ...sx,
            }}
            {...props}
        >
            {children}
        </Box>
    );
};

// Used for scroll snapping and url hash
const slide1Id = "open-source-economy";
const slide2Id = "three-steps";
const slide3Id = "freedom";
const slide4Id = "share";
const slide5Id = "automate";
const slide6Id = "sky-is-limit";
const slide7Id = "get-started";
const slideContentIds = [slide1Id, slide2Id, slide3Id, slide4Id, slide5Id, slide6Id, slide7Id];

/**
 * View displayed for Home page when not logged in
 */
export const LandingView = ({
    display = "page",
    onClose,
}: LandingViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    // Track if earth/sky is in view, and hndle scroll snap on slides
    const [earthTransform, setEarthTransform] = useState<string>("translate(0%, 100%) scale(1)");
    const [isSkyVisible, setIsSkyVisible] = useState<boolean>(false);
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);
    const lasScrollPosRef = useRef<number>(window.pageYOffset || document.documentElement.scrollTop);
    const currScrollPosRef = useRef<number>(window.pageYOffset || document.documentElement.scrollTop);
    const scrollDirectionRef = useRef<"up" | "down">("down");
    useEffect(() => {
        const onScroll = () => {
            const scrollPos = window.pageYOffset || document.documentElement.scrollTop;
            if (scrollPos !== currScrollPosRef.current) {
                scrollDirectionRef.current = scrollPos > currScrollPosRef.current ? "down" : "up";
                lasScrollPosRef.current = currScrollPosRef.current;
                currScrollPosRef.current = scrollPos;
            }
            // Helper to check if an element is in view
            const inView = (element: HTMLElement | null) => {
                if (!element) return false;
                const rect = element.getBoundingClientRect();
                const windowHeight = (window.innerHeight || document.documentElement.clientHeight);
                return rect.top < windowHeight / 2;
            };
            // Use slides 6 and 7 to determine earth position and sky visibility
            const earthHorizonSlide = document.getElementById(slide6Id);
            const earthFullSlide = document.getElementById(slide7Id);
            if (inView(earthFullSlide)) {
                setEarthTransform("translate(25%, 25%) scale(0.8)");
                setIsSkyVisible(true);
            } else if (inView(earthHorizonSlide)) {
                setEarthTransform("translate(0%, 69%) scale(1)");
                setIsSkyVisible(true);
            } else {
                setEarthTransform("translate(0%, 100%) scale(1)");
                setIsSkyVisible(false);
            }
            // Create timeout to snap to nearest slide
            if (timeoutRef.current) clearTimeout(timeoutRef.current);
            timeoutRef.current = setTimeout(() => {
                const slides = slideContentIds.map(id => document.getElementById(id));
                let minDistance = Infinity;
                let nearestSlide;

                for (const slide of slides) {
                    if (!slide) continue;
                    // Calculate distance from slide to scroll position
                    const rect = slide.getBoundingClientRect();
                    const windowHeight = (window.innerHeight || document.documentElement.clientHeight);
                    const distance = Math.abs(rect.top - windowHeight / 2);
                    // If slide is at least halfway in view, set with min distance of 0
                    if (rect.top < windowHeight / 2 && rect.bottom > windowHeight / 2) {
                        minDistance = 0;
                        nearestSlide = slide;
                    }
                    // Otherwise, determine closest scroll using direction and distance
                    else if (
                        (scrollDirectionRef.current === "down" && rect.top > 0) ||
                        (scrollDirectionRef.current === "up" && rect.bottom < windowHeight)
                    ) {
                        if (distance < minDistance) {
                            minDistance = distance;
                            nearestSlide = slide;
                        }
                    }
                }
                nearestSlide?.scrollIntoView({ behavior: "smooth" });
            }, 350);
        };
        // Add scroll listener to body
        window.addEventListener("scroll", onScroll, { passive: true });
        return () => {
            window.removeEventListener("scroll", onScroll);
            if (timeoutRef.current) clearTimeout(timeoutRef.current);
        };
    }, []);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
            />
            <SlidePage id="landing-slides" sx={{ background: "radial-gradient(circle, rgb(6 6 46) 12%, rgb(1 1 36) 52%, rgb(3 3 20) 80%)" }}>
                <SlideContainerNeon id="neon-container" show={!isSkyVisible} sx={{ zIndex: 3 }}>
                    <SlideContent id={slide1Id}>
                        <Typography component="h1" sx={{
                            ...slideTitle,
                            ...greenNeonText,
                            fontWeight: "bold",
                        }}>
                            Revolutionize Your Workflow
                        </Typography>
                        <Typography component="h2" variant="h5" sx={{ ...slideText, textAlign: "center" }}>
                            Harness the power of AI to automate tasks, collaborate effortlessly, and start businesses with a few clicks.
                        </Typography>
                        <PulseButton
                            variant="outlined"
                            color="secondary"
                            onClick={() => openLink(setLocation, LINKS.Start)}
                            startIcon={<PlayIcon fill='#0fa' />}
                            sx={{
                                marginLeft: "auto !important",
                                marginRight: "auto !important",
                            }}
                        >Start Now</PulseButton>
                        {/* Icon buttons for White paper, GitHub, Twitter, and Discord */}
                        <Stack direction="row" spacing={2} display="flex" justifyContent="center" alignItems="center" sx={{ paddingTop: 8, zIndex: 3 }}>
                            <Tooltip title="Read the white Paper" placement="bottom">
                                <IconButton onClick={() => openLink(setLocation, WHITE_PAPER_URL)} sx={iconButtonProps}>
                                    <ArticleIcon fill='#0fa' />
                                </IconButton>
                            </Tooltip>
                            <Tooltip title="Check out our code" placement="bottom">
                                <IconButton onClick={() => openLink(setLocation, SOCIALS.GitHub)} sx={iconButtonProps}>
                                    <GitHubIcon fill='#0fa' />
                                </IconButton>
                            </Tooltip>
                            <Tooltip title="Follow us on Twitter" placement="bottom">
                                <IconButton onClick={() => openLink(setLocation, SOCIALS.Twitter)} sx={iconButtonProps}>
                                    <TwitterIcon fill='#0fa' />
                                </IconButton>
                            </Tooltip>
                            <Tooltip title="Join us on Discord" placement="bottom">
                                <IconButton onClick={() => openLink(setLocation, SOCIALS.Discord)} sx={iconButtonProps}>
                                    <DiscordIcon fill='#0fa' />
                                </IconButton>
                            </Tooltip>
                        </Stack>
                    </SlideContent>
                    <SlideContent id={slide2Id}>
                        <Stack
                            direction="column"
                            spacing={5}
                            p={4}
                            textAlign="center"
                            justifyContent="center"
                            alignItems="center"
                            sx={{
                                background: "#2c2d2fd1",
                                borderRadius: 4,
                                boxShadow: 2,
                                zIndex: 2,
                            }}
                        >
                            <Typography variant='h2' sx={{ ...slideTitle }}>AI-Driven Conversations</Typography>
                            <Grid container spacing={2}>
                                <Grid item xs={12} sm={6} margin="auto">
                                    <Typography component="h2" variant="h5" sx={{ ...slideText, textAlign: "center" }}>
                                        Create and interact with AI bots, capable of intelligent conversation and task execution.
                                    </Typography>
                                </Grid>
                                <Grid item xs={12} sm={6} sx={{ paddingLeft: "0 !important" }}>
                                    <Box sx={{ ...slideImageContainer }}>
                                        {/* <Box
                                            component="img"
                                            alt="Showcases the concept of taking back your freedom. Features a woman who looks empowered and in control, looking straight ahead, with triumph in her eyes"
                                            src={WomanTriumph}
                                            sx={{ borderRadius: "32px", objectFit: "cover" }}
                                        /> */}
                                    </Box>
                                </Grid>
                            </Grid>
                            {/* Illustration for the slide. A stylized graphic of a bot engaging in a chat conversation */}
                        </Stack>
                    </SlideContent>
                    <SlideContent id={slide3Id}>
                        <Typography variant='h2' sx={{ ...slideTitle }}>Collaborative Routines</Typography>
                        <Typography component="h2" variant="h5" sx={{ ...slideText, textAlign: "center" }}>
                            Design routines with users and bots, catering to various tasks like creative writing, form creation, and scheduled automation.
                        </Typography>
                        {/* GIF for the slide. Animated representation of a routine being built and executed */}
                    </SlideContent>
                    <SlideContent id={slide4Id}>
                        <Typography variant='h2' sx={{ ...slideTitle }}>Organizational Management</Typography>
                        <Typography component="h2" variant="h5" sx={{ ...slideText, textAlign: "center" }}>
                            Organize your business processes efficiently with routines and bots, or copy an existing business in a few clicks.
                        </Typography>
                        {/* Illustration for the slide. A graphic showing a bird-eye view of an organization managed by bots */}
                    </SlideContent>
                    <SlideContent id={slide5Id}>
                        <Typography variant='h2' sx={{ ...slideTitle }}>Automating the Economy</Typography>
                        <Typography component="h2" variant="h5" sx={{ ...slideText, textAlign: "center" }}>
                            By combining bots, routines, and organizations, we pave the way for an automated and transparent economy, accessible to everyone.
                        </Typography>
                        {/* GIF for the slide. Animated visualization of a global network showing the growth of automated systems */}
                    </SlideContent>
                </SlideContainerNeon>
                <SlideContainer id='sky-slide' sx={{ color: "white", background: "black", zIndex: 4 }}>
                    {/* Background stars */}
                    <TwinkleStars
                        amount={400}
                        sx={{
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
                        }}
                    />
                    {/* Earth at bottom of page. Changes position depending on the slide  */}
                    <Box
                        id="earth"
                        component="img"
                        src={Earth}
                        alt="Earth illustration"
                        sx={{
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
                        }}
                    />
                    <SlideContent id={slide6Id}>
                        <Typography variant='h2' mb={4} sx={{ ...slideTitle, zIndex: 6 }}>The Sky is the Limit</Typography>
                        <Typography variant="h5" sx={{ ...slideText, zIndex: 6 }}>
                            Our ultimate goal is to transition the world to a fully automated, post-capitalist economy. Here's how:
                        </Typography>
                        <ul style={{ textAlign: "left", zIndex: 6 }}>
                            <li>Foster a decentralized, collaborative AI ecosystem</li>
                            <li>Prioritize ethical and socially responsible AI development</li>
                            <li>Democratize access to AI-driven automation for all</li>
                        </ul>
                    </SlideContent>
                    <SlideContent id={slide7Id}>
                        <Typography variant="h2" mb={4} sx={{ ...slideTitle, ...textPop, zIndex: 6 }}>
                            Ready to Change the World?
                        </Typography>
                        <PulseButton
                            variant="outlined"
                            color="secondary"
                            onClick={() => openLink(setLocation, LINKS.Start)}
                            startIcon={<PlayIcon fill='#0fa' />}
                            sx={{
                                marginLeft: "auto !important",
                                marginRight: "auto !important",
                                zIndex: 6,
                            }}
                        >{t("Start")}</PulseButton>
                    </SlideContent>
                </SlideContainer>
            </SlidePage >
        </>
    );
};

import { LINKS, SOCIALS, WHITE_PAPER_URL } from "@local/shared";
import { Box, Grid, Stack, Tooltip, useTheme } from "@mui/material";
import AiDrivenConvo from "assets/img/AiDrivenConvo.png";
import CollaborativeRoutines from "assets/img/CollaborativeRoutines.png";
import Earth from "assets/img/Earth.svg";
import OrganizationalManagement from "assets/img/OrganizationalManagement.png";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SlideContainerNeon } from "components/slides";
import { TwinkleStars } from "components/TwinkleStars/TwinkleStars";
import { ArticleIcon, DiscordIcon, GitHubIcon, PlayIcon, TwitterIcon } from "icons";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { greenNeonText, PulseButton, SlideBox, SlideContainer, SlideContent, SlideIconButton, SlideImage, SlideImageContainer, SlidePage, SlideText, textPop } from "styles";
import { SvgComponent } from "types";
import { Forms } from "utils/consts";
import { toDisplay } from "utils/display/pageTools";
import { SlideTitle } from "../../../styles";
import { LandingViewProps } from "../types";

// Used for scroll snapping and url hash
const slide1Id = "revolutionize-workflow";
const slide2Id = "chats";
const slide3Id = "routines";
const slide4Id = "organizations";
const slide5Id = "sky-is-limit";
const slide6Id = "get-started";

const externalLinks: [string, string, SvgComponent][] = [
    ["Read the white paper", WHITE_PAPER_URL, ArticleIcon],
    ["Check out our code", SOCIALS.GitHub, GitHubIcon],
    ["Follow us on Twitter", SOCIALS.Twitter, TwitterIcon],
    ["Join us on Discord", SOCIALS.Discord, DiscordIcon],
];

/**
 * View displayed for Home page when not logged in
 */
export const LandingView = ({
    isOpen,
    onClose,
}: LandingViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const theme = useTheme();
    const display = toDisplay(isOpen);

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
            const earthHorizonSlide = document.getElementById(slide5Id);
            const earthFullSlide = document.getElementById(slide6Id);
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
            <SlidePage id="landing-slides" sx={{
                background: theme.palette.mode === "light" ? "radial-gradient(circle, rgb(6 6 46) 12%, rgb(1 1 36) 52%, rgb(3 3 20) 80%)" : "none",
            }}>
                <SlideContainerNeon id="neon-container" show={!isSkyVisible} sx={{ zIndex: 6 }}>
                    <SlideContent id={slide1Id} sx={{
                        minHeight: {
                            xs: "calc(100vh - 64px - 56px)",
                            md: "calc(100vh - 64px)",
                        },
                    }}>
                        <SlideTitle variant="h1" sx={{
                            ...greenNeonText,
                            fontWeight: "bold",
                            marginBottom: "0!important",
                            [theme.breakpoints.up("md")]: {
                                fontSize: "4.75rem",
                            },
                            [theme.breakpoints.up("sm")]: {
                                fontSize: "4rem",
                            },
                            [theme.breakpoints.up("xs")]: {
                                fontSize: "3.4rem",
                            },
                        }}>
                            Revolutionize Your Workflow
                        </SlideTitle>
                        <SlideText sx={{ paddingBottom: 4 }}>
                            Harness the power of AI to automate tasks, collaborate effortlessly, and start businesses with ease.
                        </SlideText>
                        <PulseButton
                            variant="outlined"
                            color="secondary"
                            onClick={() => openLink(setLocation, LINKS.Start, { form: Forms.SignUp })}
                            startIcon={<PlayIcon fill='#0fa' />}
                            sx={{
                                marginLeft: "auto !important",
                                marginRight: "auto !important",
                                marginBottom: 3,
                            }}
                        >Start Now</PulseButton>
                        {/* Icon buttons for White paper, GitHub, Twitter, and Discord */}
                        <Stack direction="row" spacing={2} display="flex" justifyContent="center" alignItems="center">
                            {externalLinks.map(([tooltip, link, Icon]) => (
                                <Tooltip key={tooltip} title={tooltip} placement="bottom">
                                    <SlideIconButton onClick={() => openLink(setLocation, link, { form: Forms.SignUp })}>
                                        <Icon fill='#0fa' />
                                    </SlideIconButton>
                                </Tooltip>
                            ))}
                        </Stack>
                    </SlideContent>
                    <SlideContent id={slide2Id}>
                        <SlideBox>
                            <SlideTitle variant='h2'>AI Coworkers</SlideTitle>
                            <Grid container spacing={2}>
                                <Grid item xs={12} sm={6} margin="auto">
                                    <SlideText>
                                        Create AI bots capable of intelligent conversation and task execution.
                                    </SlideText>
                                </Grid>
                                <Grid item xs={12} sm={6}>
                                    <SlideImageContainer>
                                        <SlideImage
                                            alt="A conversation between a user and a bot. The user asks the bot about starting a business, and the bot gives suggestions on how to get started."
                                            src={AiDrivenConvo}
                                        />
                                    </SlideImageContainer>
                                </Grid>
                            </Grid>
                        </SlideBox>
                    </SlideContent>
                    <SlideContent id={slide3Id}>
                        <SlideBox>
                            <SlideTitle variant='h2'>Do Anything</SlideTitle>
                            <Grid container spacing={2}>
                                <Grid item xs={12} sm={6} margin="auto">
                                    <SlideText>
                                        Collaboratively design routines for a wide variety of tasks, such as business management, content creation, and surveys.
                                    </SlideText>
                                </Grid>
                                <Grid item xs={12} sm={6}>
                                    <SlideImageContainer>
                                        <SlideImage
                                            alt="A graphical representation of the nodes and edges of a routine."
                                            src={CollaborativeRoutines}
                                        />
                                    </SlideImageContainer>
                                </Grid>
                            </Grid>
                        </SlideBox>
                    </SlideContent>
                    <SlideContent id={slide4Id}>
                        <SlideBox>
                            <SlideTitle variant='h2'>Manage Organizations</SlideTitle>
                            <Grid container spacing={2}>
                                <Grid item xs={12} sm={6} margin="auto">
                                    <SlideText>
                                        Organize your business processes efficiently with routines and bots, or copy an existing business in a few clicks.
                                    </SlideText>
                                </Grid>
                                <Grid item xs={12} sm={6}>
                                    <SlideImageContainer>
                                        <SlideImage
                                            alt="The page for an organization, showing the organization's name, bio, picture, and members."
                                            src={OrganizationalManagement}
                                        />
                                    </SlideImageContainer>
                                </Grid>
                            </Grid>
                        </SlideBox>
                    </SlideContent>
                </SlideContainerNeon>
                <SlideContainer id='sky-slide' sx={{ color: "white", background: "black", zIndex: 3 }}>
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
                    <SlideContent id={slide5Id}>
                        <SlideTitle variant='h2' mb={4} sx={{ zIndex: 6 }}>The Sky is the Limit</SlideTitle>
                        <SlideText sx={{ zIndex: 6 }}>
                            By combining bots, routines, and organizations, we're paving the way for an automated and transparent economy - accessible to all.
                        </SlideText>
                    </SlideContent>
                    <SlideContent id={slide6Id}>
                        <SlideTitle variant="h2" mb={4} sx={{ ...textPop, zIndex: 6 }}>
                            Ready to Change the World?
                        </SlideTitle>
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

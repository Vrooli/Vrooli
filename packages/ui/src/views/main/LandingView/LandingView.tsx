import { DOCS_URL, LINKS, SOCIALS } from "@local/shared";
import { Box, BoxProps, Button, ButtonProps, Grid, Stack, Tooltip, keyframes, styled, useTheme } from "@mui/material";
import AiDrivenConvo from "assets/img/AiDrivenConvo.png";
import CollaborativeRoutines from "assets/img/CollaborativeRoutines.webp";
import Earth from "assets/img/Earth.svg";
import OrganizationalManagement from "assets/img/OrganizationalManagement.webp";
import { PageContainer } from "components/Page/Page";
import { StarryBackground } from "components/StarryBackground/StarryBackground";
import { Footer } from "components/navigation/Footer/Footer";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SlideContainerNeon } from "components/slides/SlideContainerNeon/SlideContainerNeon";
import { useWindowSize } from "hooks/useWindowSize";
import { ArticleIcon, GitHubIcon, PlayIcon, XIcon } from "icons";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { ScrollBox, SlideBox, SlideContainer, SlideContent, SlideIconButton, SlideImage, SlideImageContainer, SlidePage, SlideText, greenNeonText, textPop } from "styles";
import { SvgComponent } from "types";
import { SlideTitle } from "../../../styles";
import { LandingViewProps } from "../types";

type EarthPosition = keyof typeof earthPositions;
type ScrollDirection = "up" | "down";
type ScrollInfo = {
    currScrollPos: number;
    lastScrollPos: number;
    scrollDirection: ScrollDirection;
}

// Used for scroll snapping and url hash
const slideIds = {
    slide1: "revolutionize-workflow",
    slide2: "chats",
    slide3: "routines",
    slide4: "teams",
    slide5: "sky-is-limit",
    slide6: "get-started",
} as const;

const externalLinks: [string, string, SvgComponent][] = [
    ["Read the docs", DOCS_URL, ArticleIcon],
    ["Check out our code", SOCIALS.GitHub, GitHubIcon],
    ["Follow us on X/Twitter", SOCIALS.X, XIcon],
];

const pulse = keyframes`
    0% {
        box-shadow: 0 0 0 0 rgba(0, 255, 170, 0.7);
    }
    70% {
        box-shadow: 0 0 0 10px rgba(0, 255, 170, 0);
    }
    100% {
        box-shadow: 0 0 0 0 rgba(0, 255, 170, 0);
    }
`;

const PulseButton = styled(Button)(({ theme }) => ({
    // Button border has neon green glow animation
    animation: `${pulse} 3s infinite ease`,
    background: "#00aa71",
    borderColor: "#0fa",
    borderWidth: "2px",
    borderRadius: theme.spacing(2),
    color: "white",
    fontWeight: 550,
    width: "fit-content",
    // On hover, brighten and grow
    "&:hover": {
        background: "#2f9875",
        borderColor: "#0fa",
        filter: "brightness(1.2)",
        transform: "scale(1.05)",
    },
    transition: "all 0.2s ease",
}));

const earthPositions = {
    full: "translate(25%, 25%) scale(0.8)",
    hidden: "translate(0%, 100%) scale(1)",
    horizon: "translate(0%, 69%) scale(1)",
} as const;
interface EarthBoxProps extends BoxProps {
    earthPosition: EarthPosition;
    scrollDirection: ScrollDirection;
}
const EarthBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "earthPosition" && prop !== "scrollDirection",
})<EarthBoxProps>(({ earthPosition, scrollDirection }) => ({
    width: "150%",
    position: "fixed",
    bottom: "0",
    left: "-25%",
    right: "-25%",
    margin: "auto",
    maxWidth: "1000px",
    maxHeight: "1000px",
    pointerEvents: "none",
    transform: earthPositions[earthPosition],
    // Transition time shortens when scrolling to the previous slide
    transition: (scrollDirection === "up" && earthPosition === "hidden")
        ? "transform 0.2s ease-in-out"
        : "transform 1.5s ease-in-out",
    zIndex: 5,
}));

const LandingSlidesPage = styled(SlidePage)(({ theme }) => ({
    background: theme.palette.mode === "light" ? "radial-gradient(circle, rgb(6 6 46) 12%, rgb(1 1 36) 52%, rgb(3 3 20) 80%)" : "none",
}));
const SlideContainerSky = styled(SlideContainer)(() => ({
    color: "white",
    background: "black",
    zIndex: 5,
}));
const SlideContent1 = styled(SlideContent)(({ theme }) => ({
    minHeight: "calc(100vh - 64px - 56px)",
    [theme.breakpoints.up("md")]: {
        minHeight: "calc(100vh - 64px)",
    },
}));
const Slide1Title = styled(SlideTitle)(({ theme }) => ({
    ...greenNeonText,
    fontWeight: "bold",
    marginBottom: "0!important",
    [theme.breakpoints.up("xs")]: {
        fontSize: "3.4rem",
    },
    [theme.breakpoints.up("sm")]: {
        fontSize: "4rem",
    },
    [theme.breakpoints.up("md")]: {
        fontSize: "4.75rem",
    },
}));
const Slide1Subtitle = styled(SlideText)(() => ({
    paddingBottom: 4,
}));
const Slide1SiteNavBox = styled(Box)(() => ({
    display: "flex",
    flexDirection: "column",
    justifyContent: "center",
    gap: 1,
    alignItems: "center",
    marginLeft: "auto !important",
    marginRight: "auto !important",
    marginBottom: 3,
}));
interface Slide1StartButtonProps extends ButtonProps {
    isMobile: boolean;
}
const Slide1StartButton = styled(PulseButton, {
    shouldForwardProp: (prop) => prop !== "isMobile",
})<Slide1StartButtonProps>(({ isMobile }) => ({
    fontSize: isMobile ? "1.3rem" : "1.8rem",
    zIndex: 2,
    textTransform: "none",
}));
interface Slide1ViewSiteButtonProps extends ButtonProps {
    isMobile: boolean;
}
const Slide1ViewSiteButton = styled(Button, {
    shouldForwardProp: (prop) => prop !== "isMobile",
})<Slide1ViewSiteButtonProps>(({ isMobile }) => ({
    fontSize: isMobile ? "1rem" : "1.4rem",
    zIndex: 2,
    color: "white",
    borderColor: "white",
    textDecoration: "underline",
    textTransform: "none",
}));
const Slide5Title = styled(SlideTitle)(() => ({
    zIndex: 6,
    marginBottom: 4,
}));
const Slide5Text = styled(SlideText)(() => ({
    zIndex: 6,
}));
const Slide6Title = styled(SlideTitle)(() => ({
    ...textPop,
    zIndex: 6,
    marginBottom: 4,
}));
interface Slide6StartButtonProps extends ButtonProps {
    isMobile: boolean;
}
const Slide6StartButton = styled(PulseButton, {
    shouldForwardProp: (prop) => prop !== "isMobile",
})<Slide6StartButtonProps>(({ isMobile }) => ({
    fontSize: isMobile ? "1rem" : "1.4rem",
    marginLeft: "auto !important",
    marginRight: "auto !important",
    zIndex: 6,
}));

/**
 * View displayed for Home page when not logged in
 */
export function LandingView({
    display,
    onClose,
}: LandingViewProps) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    function toSignUp() {
        openLink(setLocation, LINKS.Signup);
    }
    function toSearch() {
        openLink(setLocation, LINKS.Search);
    }

    // Track if earth/sky is in view, and handle scroll snap on slides
    const [earthPosition, setEarthPosition] = useState<EarthPosition>("hidden");
    const scrollBoxRef = useRef<HTMLDivElement>(null);
    const scrollInfoRef = useRef<ScrollInfo>({
        currScrollPos: window.pageYOffset || document.documentElement.scrollTop,
        lastScrollPos: window.pageYOffset || document.documentElement.scrollTop,
        scrollDirection: "down",
    });
    useEffect(function earthAnimationEffect() {
        const scrollBox = scrollBoxRef.current;
        if (!scrollBox) return;

        function onScroll() {
            const { currScrollPos } = scrollInfoRef.current;
            const scrollPos = scrollBox?.scrollTop || window.pageYOffset || document.documentElement.scrollTop;
            if (scrollPos !== currScrollPos) {
                scrollInfoRef.current = {
                    currScrollPos: scrollPos,
                    lastScrollPos: currScrollPos,
                    scrollDirection: scrollPos > currScrollPos ? "down" : "up",
                };
            }

            // Helper to check if an element is in view
            function inView(element: HTMLElement | null) {
                if (!element || !scrollBox) return false;
                const rect = element.getBoundingClientRect();
                const scrollBoxRect = scrollBox.getBoundingClientRect();
                const elementTop = rect.top - scrollBoxRect.top;
                const elementBottom = rect.bottom - scrollBoxRect.top;
                const scrollHeight = scrollBox.clientHeight;
                return (elementTop < scrollHeight / 2) && (elementBottom > 0);
            }

            // Use slides 5 and 6 to determine Earth position
            const earthHorizonSlide = document.getElementById(slideIds.slide5);
            const earthFullSlide = document.getElementById(slideIds.slide6);
            if (inView(earthFullSlide)) {
                setEarthPosition("full");
            } else if (inView(earthHorizonSlide)) {
                setEarthPosition("horizon");
            } else {
                setEarthPosition("hidden");
            }
        }

        scrollBox.addEventListener("scroll", onScroll, { passive: true });
        return () => {
            scrollBox.removeEventListener("scroll", onScroll);
        };
    }, []);

    return (
        <PageContainer size="fullSize">
            <ScrollBox ref={scrollBoxRef}>
                <TopBar
                    display={display}
                    onClose={onClose}
                    titleBehaviorMobile="ShowIn"
                />
                <LandingSlidesPage id="landing-slides">
                    <SlideContainerNeon id="neon-container">
                        <SlideContent1 id={slideIds.slide1}>
                            <Slide1Title variant="h1">
                                Let AI Take the Wheel
                            </Slide1Title>
                            <Slide1Subtitle>
                                Use AI teams to automate business tasks and enhance personal productivity
                            </Slide1Subtitle>
                            <Slide1SiteNavBox>
                                <Slide1StartButton
                                    variant="outlined"
                                    color="secondary"
                                    isMobile={isMobile}
                                    onClick={toSignUp}
                                    startIcon={<PlayIcon fill='white' />}
                                >Start for free</Slide1StartButton>
                                <Slide1ViewSiteButton
                                    variant="text"
                                    color="secondary"
                                    isMobile={isMobile}
                                    onClick={toSearch}
                                >View Site</Slide1ViewSiteButton>
                            </Slide1SiteNavBox>
                            <Stack direction="row" spacing={2} display="flex" justifyContent="center" alignItems="center">
                                {externalLinks.map(([tooltip, link, Icon]) => {
                                    function onClick() {
                                        openLink(setLocation, link);
                                    }

                                    return (
                                        <Tooltip key={tooltip} title={tooltip} placement="bottom">
                                            <SlideIconButton onClick={onClick}>
                                                <Icon fill='#0fa' />
                                            </SlideIconButton>
                                        </Tooltip>
                                    );
                                })}
                            </Stack>
                        </SlideContent1>
                        <SlideContent id={slideIds.slide2}>
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
                        <SlideContent id={slideIds.slide3}>
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
                        <SlideContent id={slideIds.slide4}>
                            <SlideBox>
                                <SlideTitle variant='h2'>Manage Teams</SlideTitle>
                                <Grid container spacing={2}>
                                    <Grid item xs={12} sm={6} margin="auto">
                                        <SlideText>
                                            Organize your business processes efficiently with routines and bots, or copy an existing business in a few clicks.
                                        </SlideText>
                                    </Grid>
                                    <Grid item xs={12} sm={6}>
                                        <SlideImageContainer>
                                            <SlideImage
                                                alt="The page for a team, showing the team's name, bio, picture, and members."
                                                src={OrganizationalManagement}
                                            />
                                        </SlideImageContainer>
                                    </Grid>
                                </Grid>
                            </SlideBox>
                        </SlideContent>
                    </SlideContainerNeon>
                    <SlideContainerSky id='sky-slide'>
                        <StarryBackground />
                        {/* Earth at bottom of page. Changes position depending on the slide  */}
                        <EarthBox
                            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                            // @ts-ignore
                            alt="Earth illustration"
                            component="img"
                            earthPosition={earthPosition}
                            id="earth"
                            scrollDirection={scrollInfoRef.current.scrollDirection}
                            src={Earth}
                        />
                        <SlideContent id={slideIds.slide5}>
                            <Slide5Title variant='h2'>The Sky is the Limit</Slide5Title>
                            <Slide5Text>
                                By combining bots, routines, and teams, we&apos;re paving the way for an automated and transparent economy - accessible to all.
                            </Slide5Text>
                        </SlideContent>
                        <SlideContent id={slideIds.slide6}>
                            <Slide6Title variant="h2">
                                Ready to Change the World?
                            </Slide6Title>
                            <Slide6StartButton
                                variant="outlined"
                                color="secondary"
                                isMobile={isMobile}
                                onClick={toSignUp}
                                startIcon={<PlayIcon fill='white' />}
                            >{t("Start")}</Slide6StartButton>
                        </SlideContent>
                    </SlideContainerSky>
                </LandingSlidesPage>
                <Footer />
            </ScrollBox>
        </PageContainer>
    );
}

import { DOCS_URL, LINKS, SOCIALS } from "@local/shared";
import { Box, BoxProps, Button, ButtonProps, Grid, Stack, Tooltip, Typography, keyframes, styled, useTheme } from "@mui/material";
import Blob1 from "assets/img/blob1.svg";
import Blob2 from "assets/img/blob2.svg";
import { ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { openLink, useLocation } from "route";
import AiDrivenConvo from "../../assets/img/AiDrivenConvo.png";
import CollaborativeRoutines from "../../assets/img/CollaborativeRoutines.webp";
import Earth from "../../assets/img/Earth.svg";
import { default as OrganizationalManagement, default as TourThumbnail } from "../../assets/img/OrganizationalManagement.webp";
import { PageContainer } from "../../components/Page/Page.js";
import { Footer } from "../../components/navigation/Footer/Footer.js";
import { TopBar } from "../../components/navigation/TopBar/TopBar.js";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { ArticleIcon, GitHubIcon, LogInIcon, PlayIcon, XIcon } from "../../icons/common.js";
import { ScrollBox, SlideIconButton, greenNeonText } from "../../styles.js";
import { SvgComponent } from "../../types.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { CHAT_SIDE_MENU_ID, PubSub, SIDE_MENU_ID } from "../../utils/pubsub.js";
import { LandingViewProps } from "./types.js";

const PAGE_LAYERS = {
    Neon: 0, // Switches with space based on scroll position
    VacuumOfSpace: 0,
    Stars: 2,
    Earth: 4,
    Content: 5,
};

/** Blur factor for odd-numbered blobs */
const BLUR_DIVISOR_ODD = 16;
/** Blur factor for even-numbered blobs */
const BLUR_DIVISOR_EVEN = 50;
const ODD_SIZE_BASE = 0.4;
const EVEN_SIZE_BASE = 0.6;
const RANDOM_SIZE_RANGE = 0.3;
const POSITION_OFFSET_MIN = 0.1;
const POSITION_OFFSET_RANGE = 0.8;
const MAX_HUE_ROTATE = 360;
const NUM_NEON_BLOBS_MOBILE = 3;
const NUM_NEON_BLOBS_DESKTOP = 5;

function calculateBlobPosition(percent: number, dimension: number, offset: number) {
    return (percent * dimension) - (offset / 2);
}

function calculateBlobSize(percent: number, { width, height }: Dimensions) {
    return percent * Math.min(width, height);
}

function calculateBlobBlur(widthPercent: number, heightPercent: number, { width, height }: Dimensions, id: number) {
    return Math.floor((widthPercent + heightPercent) * (width + height) / (id % 2 ? BLUR_DIVISOR_ODD : BLUR_DIVISOR_EVEN));
}

type BlobInfo = {
    id: number;
    imgSrc: string;
    top: number;
    left: number;
    width: number;
    height: number;
    hueRotate: number;
};
type Dimensions = {
    width: number;
    height: number;
};

interface BlobBoxProps extends BoxProps {
    blob: BlobInfo;
    viewport: Dimensions;
}
const BlobBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "blob" && prop !== "viewport",
})<BlobBoxProps>(({ blob, viewport }) => {
    const width = calculateBlobSize(blob.width, viewport);
    const height = calculateBlobSize(blob.height, viewport);
    const top = calculateBlobPosition(blob.top, viewport.height, height);
    const left = calculateBlobPosition(blob.left, viewport.width, width);

    return {
        position: "fixed",
        pointerEvents: "none",
        top: top + "px",
        left: left + "px",
        width: width + "px",
        height: height + "px",
        opacity: 0.5,
        filter: `hue-rotate(${blob.hueRotate}deg) blur(${calculateBlobBlur(blob.width, blob.height, viewport, blob.id)}px)`,
        zIndex: 0,
    } as const;
});
const BlobImage = styled(Box)(() => ({
    maxWidth: "100%",
    maxHeight: "100%",
}));

export function RandomBlobs({ numberOfBlobs }: { numberOfBlobs: number }) {
    const [viewport, setViewport] = useState<Dimensions>({ width: window.innerWidth, height: window.innerHeight });
    const [blobs, setBlobs] = useState<BlobInfo[]>([]);

    useEffect(function generateBlobsEffect() {
        function generateInitialBlobs() {
            const newBlobs: BlobInfo[] = [];
            for (let i = 0; i < numberOfBlobs; i++) {
                const isOdd = i % 2 !== 0;
                const sizeBase = isOdd ? ODD_SIZE_BASE : EVEN_SIZE_BASE;
                const width = Math.random() * RANDOM_SIZE_RANGE + sizeBase;
                const height = Math.random() * RANDOM_SIZE_RANGE + sizeBase;
                const top = Math.random() * POSITION_OFFSET_RANGE + POSITION_OFFSET_MIN;
                const left = Math.random() * POSITION_OFFSET_RANGE + POSITION_OFFSET_MIN;

                newBlobs.push({
                    id: i,
                    imgSrc: isOdd ? Blob1 : Blob2,
                    top,
                    left,
                    width,
                    height,
                    hueRotate: Math.random() * MAX_HUE_ROTATE,
                });
            }
            return newBlobs;
        }

        setBlobs(generateInitialBlobs());
    }, [numberOfBlobs]);

    useEffect(function resizeEffect() {
        function handleResize() {
            setViewport({ width: window.innerWidth, height: window.innerHeight });
        }

        window.addEventListener("resize", handleResize);
        return () => window.removeEventListener("resize", handleResize);
    }, []);

    return (
        <>
            {blobs.map(blob => (
                <BlobBox key={blob.id} blob={blob} viewport={viewport}>
                    <BlobImage
                        component="img"
                        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                        // @ts-ignore
                        src={blob.imgSrc}
                        alt={`Blob ${blob.id}`}
                    />
                </BlobBox>
            ))}
        </>
    );
}

interface NeonBoxProps extends BoxProps {
    isVisible: boolean;
}
const NeonBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isVisible",
})<NeonBoxProps>(({ isVisible }) => ({
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
function NeonScene({
    isVisible,
}: NeonSceneProps) {
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    return (
        <NeonBox isVisible={isVisible} >
            <RandomBlobs numberOfBlobs={isMobile ? NUM_NEON_BLOBS_MOBILE : NUM_NEON_BLOBS_DESKTOP} />
        </NeonBox>
    );
}


const STAR_AMOUNT = 800; // Total stars
const STAR_SIZE = 2;

interface SpaceBoxProps extends BoxProps {
    isVisible: boolean;
}
const SpaceBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isVisible",
})<SpaceBoxProps>(({ isVisible }) => ({
    opacity: isVisible ? 1 : 0,
    transition: "opacity 1s",
    position: "fixed",
    pointerEvents: "none",
    top: 0,
    left: 0,
    width: "100%",
    height: "100%",
    overflow: "hidden",
    zIndex: PAGE_LAYERS.VacuumOfSpace,
    background: "black",
}));
const starCanvasStyle = {
    width: "100%",
    height: "100%",
    pointerEvents: "none",
} as const;

type StarryBackgroundProps = {
    isVisible: boolean;
}

function StarryBackground({
    isVisible,
}: StarryBackgroundProps) {
    const canvasRef = useRef<HTMLCanvasElement>(null);

    useEffect(function drawStars() {
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
        <SpaceBox isVisible={isVisible}>
            <canvas ref={canvasRef} style={starCanvasStyle} />
        </SpaceBox>
    );
}

const VideoContainerOuter = styled(Box)(() => ({
    width: "--webkit-fill-available",
    height: "100%",
    cursor: "pointer",
    padding: 0,
    margin: 0,
    border: "none",
    borderRadius: 0,
    boxShadow: "none",
    background: "transparent",
    position: "relative",
    "& > img": {
        borderRadius: 0,
        objectFit: "cover",
        width: "100%",
        height: "100%",
    },
}));
const PlayIconBox = styled(Box)(({ theme }) => ({
    position: "absolute",
    top: "50%",
    left: "50%",
    transform: "translate(-50%, -50%)",
    display: "flex",
    padding: theme.spacing(1),
    background: "#bfbfbf88",
    borderRadius: "100%",
    zIndex: 10,
    "&:hover": {
        transform: "translate(-50%, -50%) scale(1.1)",
        background: "#bfbfbfcc",
    },
    transition: "all 0.2s ease",
}));

type VideoContainerProps = {
    children: ReactNode;
    onClick: () => unknown;
}
function VideoContainer({
    onClick,
    children,
}: VideoContainerProps) {
    return (
        <VideoContainerOuter onClick={onClick}>
            {children}
            <PlayIconBox>
                <PlayIcon fill="white" width={40} height={40} />
            </PlayIconBox>
        </VideoContainerOuter>
    );
}

type EarthPosition = keyof typeof earthPositions;
type ScrollDirection = "up" | "down";
type ScrollInfo = {
    currScrollPos: number;
    lastScrollPos: number;
    scrollDirection: ScrollDirection;
}

const externalLinks: [string, string, SvgComponent][] = [
    ["Read the docs", DOCS_URL, ArticleIcon],
    ["Check out our code", SOCIALS.GitHub, GitHubIcon],
    ["Follow us on X/Twitter", SOCIALS.X, XIcon],
];

// TODO create videos and update URLs
const videoUrls = {
    Tour: "https://www.youtube.com/embed/dQw4w9WgXcQ?autoplay=1",
    Convo: "https://www.youtube.com/embed/dQw4w9WgXcQ?autoplay=1",
    Routine: "https://www.youtube.com/embed/dQw4w9WgXcQ?autoplay=1",
    Team: "https://www.youtube.com/embed/dQw4w9WgXcQ?autoplay=1",
};

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
const pulseHover = {
    background: "#2f9875",
    borderColor: "#0fa",
    borderWidth: "2px",
    filter: "brightness(1.2)",
} as const;
const PulseButton = styled(Button)(({ theme }) => ({
    // Button border has neon green glow animation
    animation: `${pulse} 3s infinite ease`,
    background: "#017d53",
    borderColor: "#0fa",
    borderWidth: "2px",
    borderRadius: theme.spacing(2),
    color: "white",
    fontWeight: 550,
    width: "fit-content",
    "&:hover": pulseHover,
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
    zIndex: PAGE_LAYERS.Earth,
}));

const SlideContainer = styled(Box)(() => ({
    overflow: "hidden",
    position: "relative",
}));
const SlideContent = styled(Stack)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    justifyContent: "center",
    marginLeft: "auto",
    marginRight: "auto",
    padding: 0,
    textAlign: "center",
    zIndex: PAGE_LAYERS.Content,
    // eslint-disable-next-line no-magic-numbers
    gap: theme.spacing(8),
    [theme.breakpoints.down("md")]: {
        // eslint-disable-next-line no-magic-numbers
        gap: theme.spacing(4),
    },
}));
const SlideTitle = styled(Typography)(({ theme }) => ({
    textAlign: "start",
    wordBreak: "break-word",
    zIndex: 10,
    color: "white",
    [theme.breakpoints.up("xs")]: {
        fontSize: "2rem",
    },
    [theme.breakpoints.up("sm")]: {
        fontSize: "2.5rem",
    },
    [theme.breakpoints.up("md")]: {
        fontSize: "3rem",
    },
}));
const SlideText = styled("h3")(({ theme }) => ({
    zIndex: 10,
    textAlign: "start",
    textWrap: "balance",
    maxWidth: "700px",
    color: theme.palette.text.secondary,
    borderRadius: theme.spacing(1),
    [theme.breakpoints.up("md")]: {
        fontSize: "1.4rem",
    },
    [theme.breakpoints.down("md")]: {
        fontSize: "1.2rem",
    },
}));
const FullWidth = styled(Box)(() => ({
    display: "flex",
    flexDirection: "row",
    flexWrap: "wrap",
    width: "100%",
}));
const HalfWidth = styled(Box)(({ theme }) => ({
    display: "flex",
    flexBasis: "50%",
    flexDirection: "column",
    maxWidth: "50%",
    [theme.breakpoints.down("md")]: {
        flexBasis: "100%",
        maxWidth: "100%",
    },
}));
const LeftGridContent = styled(Grid)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    gap: 2,
    marginTop: 4,
    marginLeft: "auto",
    marginRight: 0,
    maxWidth: "min(700px, 45vw)",
    padding: theme.spacing(2),
    paddingTop: theme.spacing(4),
    [theme.breakpoints.down("md")]: {
        marginRight: "auto",
        maxWidth: "min(700px, 100%)",
    },
}));
const RightGridContent = styled(Grid)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    gap: 2,
    marginTop: 4,
    marginRight: "auto",
    marginLeft: 0,
    maxWidth: "min(700px, 45vw)",
    padding: theme.spacing(2),
    paddingTop: theme.spacing(4),
    [theme.breakpoints.down("md")]: {
        marginLeft: "auto",
        maxWidth: "min(700px, 100%)",
    },
}));

const Slide1Title = styled(SlideTitle)(() => ({
    ...greenNeonText,
    fontWeight: "bold",
    marginBottom: "0!important",
}));
const Slide1MainButtonsBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    alignItems: "center",
    gap: theme.spacing(2),
    [theme.breakpoints.down("md")]: {
        flexDirection: "column",
        gap: theme.spacing(1),
    },
}));
interface Slide1StartButtonProps extends ButtonProps {
    isMobile: boolean;
}
const Slide1StartButton = styled(PulseButton, {
    shouldForwardProp: (prop) => prop !== "isMobile",
})<Slide1StartButtonProps>(({ isMobile, theme }) => ({
    fontSize: isMobile ? "1.25rem" : "1.5rem",
    zIndex: 2,
    textTransform: "none",
    marginLeft: "auto",
    marginRight: "auto",
    width: "min(300px, 40vw)",
    marginTop: theme.spacing(4),
}));
interface Slide1QuickTourButtonProps extends ButtonProps {
    isMobile: boolean;
}
const Slide1QuickTourButton = styled(PulseButton, {
    shouldForwardProp: (prop) => prop !== "isMobile",
})<Slide1QuickTourButtonProps>(({ isMobile }) => ({
    animation: "none",
    background: "#00000033",
    fontSize: isMobile ? "1.3rem" : "1.8rem",
    zIndex: 2,
    textTransform: "none",
    "&:hover": {
        ...pulseHover,
        background: "#00000088",
    },
}));
interface Slide1ViewSiteButtonProps extends ButtonProps {
    isMobile: boolean;
}
const Slide1ViewSiteButton = styled(Button, {
    shouldForwardProp: (prop) => prop !== "isMobile",
})<Slide1ViewSiteButtonProps>(({ isMobile, theme }) => ({
    fontSize: isMobile ? "0.75rem" : "1rem",
    zIndex: 2,
    background: "#00000033",
    color: "white",
    borderColor: "white",
    borderRadius: theme.spacing(2),
    textDecoration: "underline",
    textTransform: "none",
    marginBottom: theme.spacing(8),
}));
const Slide5Title = styled(SlideTitle)(({ theme }) => ({
    zIndex: 6,
    marginTop: theme.spacing(24),
    marginBottom: theme.spacing(0),
    textAlign: "center",
}));
const Slide5Text = styled(SlideText)(({ theme }) => ({
    zIndex: 6,
    textAlign: "center",
    marginBottom: theme.spacing(16),
}));
const Slide6Title = Slide1Title;
const Slide6MainButtonsBox = styled(Slide1MainButtonsBox)(() => ({
    marginLeft: "auto",
    marginRight: "auto",
}));
const Slide6StartButton = styled(Slide1StartButton)(() => ({
    zIndex: 6,
}));
const Slide6QuickTourButton = styled(Slide1QuickTourButton)(() => ({
    zIndex: 6,
}));
const ExternalLinksBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    gap: theme.spacing(2),
    zIndex: 6,
    background: "#00000033",
    width: "fit-content",
    padding: theme.spacing(1),
    // eslint-disable-next-line no-magic-numbers
    borderRadius: theme.spacing(4),
    marginLeft: "auto",
    marginRight: "auto",
}));
const pageContainerStyle = { background: "black" } as const;

/**
 * View displayed for Home page when not logged in
 */
export function LandingView({
    display,
    onClose,
}: LandingViewProps) {
    const [, setLocation] = useLocation();
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    // Side menus are not supported in this page due to the way it's styled
    useMemo(function hideSideMenusMemo() {
        PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen: false });
        PubSub.get().publish("sideMenu", { id: CHAT_SIDE_MENU_ID, isOpen: false });
    }, []);

    function toSignUp() {
        openLink(setLocation, LINKS.Signup);
    }
    function toSearch() {
        openLink(setLocation, LINKS.Search);
    }
    function toPricing() {
        openLink(setLocation, LINKS.Pro);
    }

    function openVideo(src: string) {
        PubSub.get().publish("popupVideo", { src });
    }
    function toTourVideo() {
        openVideo(videoUrls.Tour);
    }
    function toConvoVideo() {
        openVideo(videoUrls.Convo);
    }
    function toRoutineVideo() {
        openVideo(videoUrls.Routine);
    }
    function toTeamVideo() {
        openVideo(videoUrls.Team);
    }

    // Track scroll information
    const scrollBoxRef = useRef<HTMLDivElement>(null);
    const scrollInfoRef = useRef<ScrollInfo>({
        currScrollPos: document.documentElement.scrollTop,
        lastScrollPos: document.documentElement.scrollTop,
        scrollDirection: "down",
    });
    // Track if earth/sky is in view
    const [earthPosition, setEarthPosition] = useState<EarthPosition>("hidden");
    useEffect(function earthAnimationEffect() {
        const scrollBox = scrollBoxRef.current;
        if (!scrollBox) return;

        function onScroll() {
            const { currScrollPos } = scrollInfoRef.current;
            const scrollPos = scrollBox?.scrollTop || document.documentElement.scrollTop;
            if (scrollPos !== currScrollPos) {
                scrollInfoRef.current = {
                    currScrollPos: scrollPos,
                    lastScrollPos: currScrollPos,
                    scrollDirection: scrollPos > currScrollPos ? "down" : "up",
                };
            }

            // Helper to check if an element is in view
            function isInView(element: HTMLElement | null) {
                if (!element || !scrollBox) return false;
                const rect = element.getBoundingClientRect();
                const scrollBoxRect = scrollBox.getBoundingClientRect();
                const elementTop = rect.top - scrollBoxRect.top;
                const elementBottom = rect.bottom - scrollBoxRect.top;
                const scrollHeight = scrollBox.clientHeight;
                return (elementTop < scrollHeight / 2) && (elementBottom > 0);
            }

            // Use slides 5 and 6 to determine Earth position
            const earthHorizonSlide = document.getElementById(ELEMENT_IDS.LandingViewSlideSkyIsLimit);
            const earthFullSlide = document.getElementById(ELEMENT_IDS.LandingViewSlideGetStarted);
            if (isInView(earthFullSlide)) {
                setEarthPosition("full");
            } else if (isInView(earthHorizonSlide)) {
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
        <PageContainer size="fullSize" sx={pageContainerStyle}>
            <NeonScene isVisible={earthPosition === "hidden"} />
            <StarryBackground isVisible={earthPosition !== "hidden"} />
            <ScrollBox ref={scrollBoxRef}>
                <TopBar
                    display={display}
                    onClose={onClose}
                    titleBehaviorMobile="ShowIn"
                />
                <SlideContainer id={ELEMENT_IDS.LandingViewSlideContainerNeon}>
                    <SlideContent id={ELEMENT_IDS.LandingViewSlideWorkflow}>
                        <FullWidth>
                            <HalfWidth>
                                <LeftGridContent item xs={12} md={6}>
                                    <Slide1Title variant="h1">
                                        Unlock Billionaire-Level Productivity
                                    </Slide1Title>
                                    <SlideText>
                                        Delegate like a billionaire — Let our AI agents handle the details so you can focus on what truly matters.
                                    </SlideText>
                                    <Slide1StartButton
                                        variant="outlined"
                                        color="secondary"
                                        isMobile={isMobile}
                                        onClick={toSignUp}
                                        startIcon={<LogInIcon fill='white' />}
                                    >Start for free</Slide1StartButton>
                                    <Slide1ViewSiteButton
                                        variant="text"
                                        color="secondary"
                                        isMobile={isMobile}
                                        onClick={toSearch}
                                    >View Site</Slide1ViewSiteButton>
                                </LeftGridContent>
                            </HalfWidth>
                            <HalfWidth>
                                <VideoContainer onClick={toTourVideo}>
                                    <img
                                        alt="Thumbnail for the video tour of Vrooli."
                                        src={TourThumbnail}
                                    />
                                </VideoContainer>
                            </HalfWidth>
                        </FullWidth>
                    </SlideContent>
                    <SlideContent id={ELEMENT_IDS.LandingViewSlideChats}>
                        <FullWidth>
                            <HalfWidth>
                                <VideoContainer onClick={toConvoVideo}>
                                    <img
                                        alt="A conversation between a user and a bot. The user asks the bot about starting a business, and the bot gives suggestions on how to get started."
                                        src={AiDrivenConvo}
                                    />
                                </VideoContainer>
                            </HalfWidth>
                            <HalfWidth>
                                <RightGridContent>
                                    <SlideTitle variant='h2'>AI Coworkers, Ready on Demand</SlideTitle>
                                    <SlideText>
                                        Our bots work around the clock to handle your repetitive tasks, so you can focus on what matters most.
                                    </SlideText>
                                    <SlideText>
                                        Whether it&apos;s managing projects, automating workflows, or answering common questions, our AI bots are here to make your life easier and your business run smoother.
                                    </SlideText>
                                </RightGridContent>
                            </HalfWidth>
                        </FullWidth>
                    </SlideContent>
                    <SlideContent id={ELEMENT_IDS.LandingViewSlideRoutines}>
                        <FullWidth>
                            <HalfWidth>
                                <LeftGridContent>
                                    <SlideTitle variant='h2'>Build Consistent, Automated Workflows</SlideTitle>
                                    <SlideText>
                                        Create reusable workflows that ensure your AI-driven business operates smoothly, every time.
                                    </SlideText>
                                    <SlideText>
                                        Manage your business, create content, and more, in a way that is repeatable and reliable.
                                    </SlideText>
                                    <SlideText>
                                        Alternatively, let our AI bots build routines for you based on your preferences and needs.
                                    </SlideText>
                                </LeftGridContent>
                            </HalfWidth>
                            <HalfWidth>
                                <VideoContainer onClick={toRoutineVideo}>
                                    <img
                                        alt="A graphical representation of the nodes and edges of a routine."
                                        src={CollaborativeRoutines}
                                    />
                                </VideoContainer>
                            </HalfWidth>
                        </FullWidth>
                    </SlideContent>
                    <SlideContent id={ELEMENT_IDS.LandingViewSlideTeams}>
                        <FullWidth>
                            <HalfWidth>
                                <VideoContainer onClick={toTeamVideo}>
                                    <img
                                        alt="The page for a team, showing the team's name, bio, picture, and members."
                                        src={OrganizationalManagement}
                                    />
                                </VideoContainer>
                            </HalfWidth>
                            <HalfWidth>
                                <RightGridContent>
                                    <SlideTitle variant='h2'>Manage Teams Like a Pro</SlideTitle>
                                    <SlideText>
                                        With AI-powered routines and bots, you can handle the workload of an entire team on your own—no humans required.
                                    </SlideText>
                                    <SlideText>
                                        Perfect for solo developers or small businesses, Vrooli lets you automate and manage everything from operations to communication, freeing up your time to focus on growth.
                                    </SlideText>
                                </RightGridContent>
                            </HalfWidth>
                        </FullWidth>
                    </SlideContent>
                </SlideContainer>
                <SlideContainer id={ELEMENT_IDS.LandingViewSlideContainerSky}>
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
                    <SlideContent id={ELEMENT_IDS.LandingViewSlideSkyIsLimit}>
                        <Box display="flex" flexDirection="column" alignItems="center" minHeight="100vh">
                            <Slide5Title variant='h2'>The Sky is the Limit</Slide5Title>
                            <Slide5Text>
                                By combining bots, routines, and teams, we&apos;re paving the way for an automated and transparent economy - accessible to all.
                            </Slide5Text>
                        </Box>
                    </SlideContent>
                    <SlideContent id={ELEMENT_IDS.LandingViewSlideGetStarted}>
                        <Box display="flex" flexDirection="column" alignItems="center" minHeight="100vh">
                            <Slide6Title variant="h2">
                                Ready to Change the World?
                            </Slide6Title>
                            <Slide6MainButtonsBox>
                                <Slide6StartButton
                                    variant="outlined"
                                    color="secondary"
                                    isMobile={isMobile}
                                    onClick={toSignUp}
                                    startIcon={<LogInIcon fill='white' />}
                                >I&apos;m ready</Slide6StartButton>
                                <Slide6QuickTourButton
                                    variant="outlined"
                                    color="secondary"
                                    isMobile={isMobile}
                                    onClick={toPricing}
                                >View pricing</Slide6QuickTourButton>
                            </Slide6MainButtonsBox>
                            <ExternalLinksBox>
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
                            </ExternalLinksBox>
                        </Box>
                    </SlideContent>
                </SlideContainer>
                <Footer />
            </ScrollBox>
        </PageContainer>
    );
}

/* eslint-disable no-magic-numbers */
import type { BoxProps } from "@mui/material";
import { keyframes, styled } from "@mui/material";
import Box from "@mui/material/Box";
import Grid from "@mui/material/Grid";
import Stack from "@mui/material/Stack";
import { Tooltip } from "../../components/Tooltip/Tooltip.js";
import Typography from "@mui/material/Typography";
import { LINKS, PaymentType, SOCIALS } from "@vrooli/shared";
import { lazy, Suspense, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import AiDrivenConvo from "../../assets/img/AiDrivenConvo.png";
import CollaborativeRoutines from "../../assets/img/CollaborativeRoutines.webp";
import Earth from "../../assets/img/Earth.svg";
import { default as OrganizationalManagement, default as TourThumbnail } from "../../assets/img/OrganizationalManagement.webp";
import { PageContainer } from "../../components/Page/Page.js";
import { BreadcrumbsBase } from "../../components/breadcrumbs/BreadcrumbsBase.js";
import { Button } from "../../components/buttons/Button.js";
import { IconButton } from "../../components/buttons/IconButton.js";
import { Footer } from "../../components/navigation/Footer.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SnackSeverity } from "../../components/snacks/BasicSnack/BasicSnack.js";
import { SessionContext } from "../../contexts/session.js";
import { useStripe } from "../../hooks/useStripe.js";
import { IconCommon, IconService } from "../../icons/Icons.js";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { ScrollBox } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { darkPalette } from "../../utils/display/theme.js";
import { PubSub } from "../../utils/pubsub.js";
import { BillingCycle, BillingCycleToggle, CreditDialog, PricingTiers, PricingTierType } from "../ProView/ProView.js";
import { InstallPWADialog } from "../../components/dialogs/InstallPWADialog/InstallPWADialog.js";
import { type LandingViewProps } from "./types.js";

// Lazy load the animation components
const NeonScene = lazy(() => import("./components/NeonScene.js").then(module => ({ default: module.NeonScene })));

// TODO create videos and update URLs
const videoUrls = {
    Tour: "https://www.youtube.com/embed/dQw4w9WgXcQ?autoplay=1",
    Convo: "https://www.youtube.com/embed/dQw4w9WgXcQ?autoplay=1",
    Routine: "https://www.youtube.com/embed/dQw4w9WgXcQ?autoplay=1",
    Team: "https://www.youtube.com/embed/dQw4w9WgXcQ?autoplay=1",
};

const PAGE_LAYERS = {
    Neon: 0, // Switches with space based on scroll position
    VacuumOfSpace: 0,
    Stars: 2,
    Earth: 4,
    Content: 5,
};

// Path data for the GitHub icon
const gitHubPathD = "M12 2C6.475 2 2 6.475 2 12a9.994 9.994 0 0 0 6.838 9.488c.5.087.687-.213.687-.476 0-.237-.013-1.024-.013-1.862-2.512.463-3.162-.612-3.362-1.175-.113-.288-.6-1.175-1.025-1.413-.35-.187-.85-.65-.013-.662.788-.013 1.35.725 1.538 1.025.9 1.512 2.338 1.087 2.912.825.088-.65.35-1.087.638-1.337-2.225-.25-4.55-1.113-4.55-4.938 0-1.088.387-1.987 1.025-2.688-.1-.25-.45-1.275.1-2.65 0 0 .837-.262 2.75 1.026a9.28 9.28 0 0 1 2.5-.338c.85 0 1.7.112 2.5.337 1.912-1.3 2.75-1.024 2.75-1.024.55 1.375.2 2.4.1 2.65.637.7 1.025 1.587 1.025 2.687 0 3.838-2.337 4.688-4.562 4.938.362.312.675.912.675 1.85 0 1.337-.013 2.412-.013 2.75 0 .262.188.574.688.474A10.016 10.016 0 0 0 22 12c0-5.525-4.475-10-10-10Z";

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
                    ctx.fillStyle = `rgba(100, 100, 100, ${opacity})`;
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

const VideoContainerOuter = styled(Box)(({ theme }) => ({
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
    [theme.breakpoints.down("md")]: {
        maxWidth: "min(500px, 75%)",
        margin: "auto",
        borderRadius: theme.spacing(2),
        overflow: "overlay",
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
        <VideoContainerOuter
            aria-label="Play video"
            component="button"
            onClick={onClick}
        >
            {children}
            <PlayIconBox>
                <IconCommon
                    decorative
                    fill="white"
                    name="Play"
                    width={40}
                    height={40}
                />
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

const moreTransparentButtonStyle = {
    background: "#017d5366",
    animation: "none",
} as const;

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
    textShadow: "0 3px 6px #000",
    textAlign: "center",
    wordBreak: "break-word",
    zIndex: 10,
    color: "white",
    marginBottom: theme.spacing(2),
    [theme.breakpoints.up("xs")]: {
        fontSize: "2rem",
    },
    [theme.breakpoints.up("sm")]: {
        fontSize: "2.5rem",
    },
    [theme.breakpoints.up("md")]: {
        fontSize: "3rem",
        textAlign: "start",
    },
}));
const SlideText = styled(Typography)(({ theme }) => ({
    zIndex: 10,
    textWrap: "balance",
    maxWidth: "600px",
    color: darkPalette.background.textSecondary,
    borderRadius: theme.spacing(1),
    marginBottom: theme.spacing(2),
    textShadow: "0 0 10px rgba(0, 0, 0, 0.5)",
    [theme.breakpoints.up("md")]: {
        fontSize: "1.4rem",
        textAlign: "start",
    },
    [theme.breakpoints.down("md")]: {
        fontSize: "1.2rem",
        textAlign: "center",
    },
}));
type FullWidthProps = {
    reverse?: boolean;
}
const FullWidth = styled(Box, {
    shouldForwardProp: (prop) => prop !== "reverse",
})<FullWidthProps>(({ theme, reverse }) => ({
    display: "flex",
    flexDirection: "row",
    flexWrap: "wrap",
    width: "100%",
    [theme.breakpoints.down("md")]: {
        flexDirection: reverse ? "column-reverse" : "column",
    },
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

const Slide1LeftGridContent = styled(LeftGridContent)(() => ({
    marginTop: "min(64px, 5vh)",
}));
const Slide1Title = styled(SlideTitle)(({ theme }) => ({
    fontWeight: "bold",
    marginBottom: theme.spacing(2),
    color: "#fff",
    filter: "drop-shadow(0 0 2px #0fa) drop-shadow(0 0 20px #0fa)",
    textShadow: "unset",
}));

const Slide1Buttons = styled(Box)(({ theme }) => ({
    width: "min(600px, 40vw)",
    display: "flex",
    flexDirection: "row",
    gap: theme.spacing(2),
    [theme.breakpoints.down("md")]: {
        width: "100%",
        justifyContent: "center",
    },
}));

const Slide1ActionBox = styled(Box)(({ theme }) => ({
    zIndex: 2,
    marginTop: theme.spacing(4),
    marginLeft: 0,
    marginRight: "auto",
    [theme.breakpoints.down("md")]: {
        marginLeft: "auto",
    },
}));
const Slide6Title = styled(SlideTitle)(({ theme }) => ({
    zIndex: 6,
    marginTop: theme.spacing(24),
    marginBottom: theme.spacing(2),
    textAlign: "center",
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

/**
 * View displayed for Home page when not logged in
 */
export function LandingView({
    display,
    onClose,
}: LandingViewProps) {
    const [, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const currentUser = useMemo(() => getCurrentUser(session), [session]);

    const {
        prices,
        redirectToCustomerPortal,
        startCheckout,
    } = useStripe();
    const [billingCycle, setBillingCycle] = useState<BillingCycle>(BillingCycle.Yearly);

    const [isCreditDialogOpen, setIsCreditDialogOpen] = useState(false);
    function closeCreditDialog() {
        setIsCreditDialogOpen(false);
    }

    const [isInstallPWADialogOpen, setIsInstallPWADialogOpen] = useState(false);
    function closeInstallPWADialog() {
        setIsInstallPWADialogOpen(false);
    }

    function onTierClick(tier: PricingTierType) {
        switch (tier) {
            case PricingTierType.Basic:
                openLink(setLocation, LINKS.Signup);
                break;
            case PricingTierType.Pro:
                if (currentUser.hasPremium) {
                    redirectToCustomerPortal();
                } else {
                    startCheckout(billingCycle === BillingCycle.Yearly ? PaymentType.PremiumYearly : PaymentType.PremiumMonthly);
                }
                break;
            case PricingTierType.Credits:
                if (session?.isLoggedIn) {
                    setIsCreditDialogOpen(true);
                } else {
                    PubSub.get().publish("snack", { message: "Please sign in first", severity: SnackSeverity.Warning });
                    const currentUrl = window.location.href;
                    openLink(setLocation, LINKS.Signup, { searchParams: { redirect: currentUrl } });
                }
                break;
        }
    }

    // Side menus are not supported in this page due to the way it's styled
    useMemo(function hideMenusMemo() {
        PubSub.get().publish("menu", { id: ELEMENT_IDS.LeftDrawer, isOpen: false });
        PubSub.get().publish("menu", { id: ELEMENT_IDS.RightDrawer, isOpen: false });
    }, []);

    const breadcrumbPaths = [
        {
            text: "View site",
            link: LINKS.Search,
        },
        {
            text: "Code",
            link: SOCIALS.GitHub,
        },
    ] as const;

    function toSignUp() {
        openLink(setLocation, LINKS.Signup);
    }
    function toApp() {
        setIsInstallPWADialogOpen(true);
    }
    function toGitHub() {
        openLink(setLocation, SOCIALS.GitHub);
    }
    function toX() {
        openLink(setLocation, SOCIALS.X);
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
        if (!scrollBox) {
            console.error("[LandingView.earthAnimationEffect] Scroll box not found");
            return;
        }

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

            // Use slide IDs to determine Earth position
            const earthHorizonSlides = [
                document.getElementById(ELEMENT_IDS.LandingViewSlidePricing),
            ];
            const earthFullSlides = [
                document.getElementById(ELEMENT_IDS.LandingViewSlideGetStarted),
            ];
            if (earthFullSlides.some(isInView)) {
                setEarthPosition("full");
            } else if (earthHorizonSlides.some(isInView)) {
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
        <>
            <CreditDialog
                open={isCreditDialogOpen}
                onClose={closeCreditDialog}
                startCheckout={startCheckout}
            />
            <InstallPWADialog
                open={isInstallPWADialogOpen}
                onClose={closeInstallPWADialog}
            />
            <PageContainer size="fullSize">
                <Suspense fallback={<Box sx={{ background: "black", position: "fixed", width: "100%", height: "100%", zIndex: 0 }} />}>
                    <NeonScene isVisible={earthPosition === "hidden"} />
                </Suspense>
                <StarryBackground isVisible={earthPosition !== "hidden"} />
                <ScrollBox ref={scrollBoxRef}>
                    <Navbar
                        color={darkPalette.background.textSecondary}
                    />
                    <SlideContainer id={ELEMENT_IDS.LandingViewSlideContainerNeon}>
                        <FullWidth id={ELEMENT_IDS.LandingViewSlideWorkflow}>
                            <HalfWidth>
                                <Slide1LeftGridContent>
                                    <Slide1Title variant="h1">
                                        Unlock Billionaire-Level Productivity
                                    </Slide1Title>
                                    <SlideText variant="h2">
                                        Delegate like a billionaire — Let our AI agents handle the details so you can focus on what truly matters.
                                    </SlideText>
                                    <Slide1ActionBox>
                                        <Slide1Buttons>
                                            <Button
                                                variant="neon"
                                                borderRadius="pill"
                                                onClick={toSignUp}
                                                size="lg"
                                            >Start for free</Button>
                                            <Button
                                                variant="outline"
                                                borderRadius="pill"
                                                onClick={toApp}
                                                size="lg"
                                                style={{
                                                    background: "rgba(156, 163, 175, 0.2)",
                                                    borderColor: "#9ca3af",
                                                    color: "#e5e7eb",
                                                    animation: "none",
                                                }}
                                                className="hover:tw-bg-gray-400 hover:tw-bg-opacity-30 hover:tw-border-gray-300"
                                            >Get the app</Button>
                                        </Slide1Buttons>
                                        <Slide1Buttons>
                                            <BreadcrumbsBase
                                                onClick={onClose}
                                                paths={breadcrumbPaths}
                                                separator={"•"}
                                            />
                                        </Slide1Buttons>
                                    </Slide1ActionBox>
                                </Slide1LeftGridContent>
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
                        <FullWidth id={ELEMENT_IDS.LandingViewSlideChats} reverse>
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
                                    <SlideTitle variant='h2'>Build Anything Without Writing Code</SlideTitle>
                                    <SlideText variant="h3">
                                        Just describe what you want. Your AI swarm handles the rest. Want a fitness app? They'll design interfaces, write code, create content, and prepare store listings. Need a data dashboard? They'll connect APIs, analyze patterns, and build visualizations.
                                    </SlideText>
                                    <SlideText variant="h3">
                                        No technical skills required. No team to manage. No months of development. Whether it's an app, a business, or a research project—if you can describe it, your swarm can build it.
                                    </SlideText>
                                </RightGridContent>
                            </HalfWidth>
                        </FullWidth>
                        <FullWidth id={ELEMENT_IDS.LandingViewSlideRoutines}>
                            <HalfWidth>
                                <LeftGridContent>
                                    <SlideTitle variant='h2'>Work That Gets Better While You Sleep</SlideTitle>
                                    <SlideText variant="h3">
                                        Set up a workflow once. Watch it improve itself. What took an hour last week takes minutes today. Your AI agents learn from every run, finding shortcuts you never imagined.
                                    </SlideText>
                                    <SlideText variant="h3">
                                        No more tweaking, optimizing, or managing. Your routines evolve like living things—getting faster, smarter, and more efficient with each use. Wake up to workflows that run better than when you left them.
                                    </SlideText>
                                    <SlideText variant="h3">
                                        Build complex processes from simple pieces. Like digital LEGO blocks that assemble themselves into whatever you need. The more you build, the more powerful each piece becomes.
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
                        <FullWidth id={ELEMENT_IDS.LandingViewSlideTeams} reverse>
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
                                    <SlideTitle variant='h2'>Your AI Company, Running at Light Speed</SlideTitle>
                                    <SlideText variant="h3">
                                        Think of it as a self-organizing company in your pocket. AI executives set strategy. AI managers break down goals. AI workers execute with precision. All coordinating in perfect harmony, 24/7, at the speed of thought.
                                    </SlideText>
                                    <SlideText variant="h3">
                                        When challenges arise, watch your swarm dynamically reorganize—forming specialist teams, hierarchies, and workflows on demand. Problems that took weeks now solve themselves while you sleep.
                                    </SlideText>
                                </RightGridContent>
                            </HalfWidth>
                        </FullWidth>
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
                        <SlideContent id={ELEMENT_IDS.LandingViewSlidePricing}>
                            <Box display="flex" flexDirection="column" pl={2} pr={2} pt={4} alignItems="center" minHeight="100vh" justifyContent="center" zIndex={PAGE_LAYERS.Content}>
                                <SlideTitle variant='h2'>Start Free. Scale Infinitely.</SlideTitle>
                                <SlideText>Because big ideas deserve big teams—regardless of your budget.</SlideText>
                                <Box mt={4} mb={2}>
                                    <BillingCycleToggle value={billingCycle} onChange={setBillingCycle} />
                                </Box>
                                <PricingTiers
                                    billingCycle={billingCycle}
                                    currentTheme="dark"
                                    onTierClick={onTierClick}
                                    prices={prices}
                                />
                                <Typography variant="body2" color={darkPalette.background.textSecondary} sx={{ mt: 2, textAlign: "center", textShadow: "0 0 10px rgba(0, 0, 0, 0.5)" }}>
                                    Subscriptions can be canceled at any time, hassle-free.
                                </Typography>
                            </Box>
                        </SlideContent>
                        <SlideContent id={ELEMENT_IDS.LandingViewSlideGetStarted}>
                            <Box display="flex" flexDirection="column" alignItems="center" minHeight="100vh">
                                <Slide6Title variant='h2'>Ready to Change Your World?</Slide6Title>
                                <SlideText>Boost your productivity and get more done with Vrooli.</SlideText>
                                <Button
                                    aria-label="I'm ready!"
                                    variant="neon"
                                    borderRadius="pill"
                                    size="lg"
                                    onClick={toSignUp}
                                    style={{
                                        zIndex: 6,
                                        width: "min(300px, 40vw)",
                                        marginTop: "16px",
                                        marginBottom: "16px",
                                    }}
                                    startIcon={<IconCommon
                                        decorative
                                        fill="white"
                                        name="Launch"
                                    />}
                                >I&apos;m ready!</Button>
                                <ExternalLinksBox>
                                    <Tooltip title="Check out our code" placement="bottom">
                                        <IconButton
                                            variant="transparent"
                                            size="md"
                                            aria-label="Check out our code"
                                            onClick={toGitHub}
                                            style={{
                                                background: "transparent",
                                                border: "1px solid #0fa",
                                            }}
                                            className="hover:tw-brightness-125 hover:tw-scale-110 tw-transition-all tw-duration-200"
                                        >
                                            <svg
                                                width="24"
                                                height="24"
                                                viewBox="0 0 24 24"
                                                fill="#0fa"
                                                xmlns="http://www.w3.org/2000/svg"
                                                role="img"
                                                aria-hidden="true"
                                            >
                                                <path d={gitHubPathD} />
                                            </svg>
                                        </IconButton>
                                    </Tooltip>
                                    <Tooltip title="Follow us on X/Twitter" placement="bottom">
                                        <IconButton
                                            variant="transparent"
                                            size="md"
                                            aria-label="Follow us on X/Twitter"
                                            onClick={toX}
                                            style={{
                                                background: "transparent",
                                                border: "1px solid #0fa",
                                            }}
                                            className="hover:tw-brightness-125 hover:tw-scale-110 tw-transition-all tw-duration-200"
                                        >
                                            <IconService
                                                decorative
                                                fill='#0fa'
                                                name="X"
                                            />
                                        </IconButton>
                                    </Tooltip>
                                </ExternalLinksBox>
                            </Box>
                        </SlideContent>
                    </SlideContainer>
                    <Footer />
                </ScrollBox>
            </PageContainer>
        </>
    );
}

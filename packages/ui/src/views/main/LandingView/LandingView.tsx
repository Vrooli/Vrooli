import { DOCS_URL, LINKS, SOCIALS } from "@local/shared";
import { Box, BoxProps, Button, ButtonProps, Grid, Stack, Tooltip, keyframes, styled, useTheme } from "@mui/material";
import AiDrivenConvo from "assets/img/AiDrivenConvo.png";
import CollaborativeRoutines from "assets/img/CollaborativeRoutines.webp";
import Earth from "assets/img/Earth.svg";
import OrganizationalManagement from "assets/img/OrganizationalManagement.webp";
import Blob1 from "assets/img/blob1.svg";
import Blob2 from "assets/img/blob2.svg";
import { PageContainer } from "components/Page/Page";
import { Footer } from "components/navigation/Footer/Footer";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useWindowSize } from "hooks/useWindowSize";
import { ArticleIcon, GitHubIcon, PlayIcon, XIcon } from "icons";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { ScrollBox, SlideBox, SlideContainer, SlideContent, SlideIconButton, SlideImage, SlideImageContainer, SlideText, greenNeonText, textPop } from "styles";
import { SvgComponent } from "types";
import { SlideTitle } from "../../../styles";
import { LandingViewProps } from "../types";

const PAGE_LAYERS = {
    Neon: 0, // Switches with space based on scroll position
    VacuumOfSpace: 0,
    Stars: 2,
    Earth: 4,
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

type LineInfo = {
    alpha: number;
    fromX: number;
    fromY: number;
    toX: number;
    toY: number;
    width: number;
}
type PointInfo = {
    size: number;
    x: number;
    y: number;
}

/** The number of particles. (a number lesser than 1000 is recommended under regular settings) */
const POINT_COUNT = 50;
/** Minimum point size */
const POINT_SIZE_MIN = 2;
/** Maximum point size */
const POINT_SIZE_MAX = 8;
/** Minimum line width */
const LINE_WIDTH_MIN = 1;
/** Maximum line width */
const LINE_WIDTH_MAX = 5;
const COLOR = "197, 198, 199";
/** The maximum moving speed of a particle in x or y coordinate can has in each frame. (in pixels) */
const POINT_MAX_SPEED = 0.05;
/** The maximum length of a line (i.e. the interaction radius of a particle). (in pixels) */
const INTERACTION_DISTANCE = 200;
/** Maximum allowed alpha value for lines between points */
const ALPHA_MIN = 0.5;
/** Minimum allowed alpha value for lines between points */
const ALPHA_MAX = 0.1;
/** Number of neon blobs on mobile */
const NUM_NEON_BLOBS_MOBILE = 5;
/** Number of neon blobs on desktop */
const NUM_NEON_BLOBS_DESKTOP = 8;
const RADIANS_IN_CIRCLE = Math.PI * 2;

const particleCanvasStyle = "position:fixed;top:0;left:0;pointer-events:none;";

function rand(min: number, max: number) {
    return (max - min) * Math.random() + min;
}

function getLineAttributes(length: number): { alpha: number, width: number } {
    // Ensure the length is within bounds
    if (length > INTERACTION_DISTANCE) length = INTERACTION_DISTANCE;
    if (length < 0) length = 0;

    const alpha = ALPHA_MAX + (ALPHA_MIN - ALPHA_MAX) * (1 - length / INTERACTION_DISTANCE);
    const width = LINE_WIDTH_MIN + (LINE_WIDTH_MAX - LINE_WIDTH_MIN) * (1 - length / INTERACTION_DISTANCE);

    const TENS = 10;
    return {
        alpha: Math.round(alpha * TENS) / TENS, // round to 1 decimal place
        width: Math.round(width * TENS) / TENS, // round to 1 decimal place
    };
}

class Point {
    x: number;
    y: number;
    vx: number;
    vy: number;
    size: number;

    constructor(w: number, h: number) {
        this.x = rand(0, w);
        this.y = rand(0, h);

        this.size = rand(POINT_SIZE_MIN, POINT_SIZE_MAX);

        const speed = POINT_MAX_SPEED * (this.size / POINT_SIZE_MAX);
        const angle = rand(0, Math.PI * 2);
        this.vx = Math.cos(angle) * speed;
        this.vy = Math.sin(angle) * speed;
    }

    evolve(): void {
        this.x += this.vx;
        this.y += this.vy;
    }

    getPointInteraction(p: { x: number, y: number }): LineInfo | null {
        const dx = p.x - this.x, dy = p.y - this.y;
        const d = Math.hypot(dx, dy);

        if (d > INTERACTION_DISTANCE || d < 1) return null;

        const { alpha, width } = getLineAttributes(d);
        return {
            alpha,
            fromX: this.x,
            fromY: this.y,
            toX: p.x,
            toY: p.y,
            width,
        };
    }
}

class DrawBuffer {
    ctx: CanvasRenderingContext2D;
    lineBuffer: LineInfo[];
    pointBuffer: PointInfo[];

    constructor(ctx: CanvasRenderingContext2D) {
        this.ctx = ctx;
        this.ctx.fillStyle = `rgb(${COLOR})`;
        this.ctx.lineCap = "round";

        this.lineBuffer = [];
        this.pointBuffer = [];
    }

    dumpLines() {
        const pathMap = new Map<string, Path2D>();

        for (const line of this.lineBuffer) {
            const { alpha, fromX, fromY, toX, toY, width } = line;
            const styleKey = `rgba(${COLOR},${alpha})_${width}`;

            if (!pathMap.has(styleKey)) {
                pathMap.set(styleKey, new Path2D());
            }

            const path = pathMap.get(styleKey)!;
            path.moveTo(fromX, fromY);
            path.lineTo(toX, toY);
        }

        for (const [styleKey, path] of pathMap.entries()) {
            const [strokeStyle, lineWidth] = styleKey.split("_");
            this.ctx.strokeStyle = strokeStyle;
            this.ctx.lineWidth = parseFloat(lineWidth);
            this.ctx.stroke(path);
        }

        this.lineBuffer.length = 0;
    }

    dumpPoints() {
        this.ctx.beginPath();

        for (let i = 0; i < this.pointBuffer.length; i++) {
            const { size, x, y } = this.pointBuffer[i];

            this.ctx.moveTo(x, y);
            this.ctx.arc(x, y, size, 0, RADIANS_IN_CIRCLE);
        }
        this.ctx.fill();

        this.pointBuffer.length = 0;
    }
}

class Simulator {
    ctx: CanvasRenderingContext2D;
    points: Point[];
    draw_buffer: DrawBuffer;
    w: number;
    h: number;

    constructor(ctx: CanvasRenderingContext2D, w: number, h: number) {
        this.ctx = ctx;
        this.w = w;
        this.h = h;
        this.draw_buffer = new DrawBuffer(this.ctx);
        this.points = [];

        this.initializePoints();
    }

    initializePoints(): void {
        for (let i = 0; i < POINT_COUNT; i++) this.genNewPoint();
    }

    genNewPoint(): void {
        const point = new Point(this.w, this.h);
        this.points.push(point);
    }

    evolvePoints() {
        for (const p of this.points) {
            this.draw_buffer.pointBuffer.push({ size: p.size, x: p.x, y: p.y });
            p.evolve();

            // Boundary check and reflection
            if (p.x <= 0 || p.x >= this.w) p.vx *= -1;
            if (p.y <= 0 || p.y >= this.h) p.vy *= -1;

            // Keep particles within bounds
            p.x = Math.max(0, Math.min(this.w, p.x));
            p.y = Math.max(0, Math.min(this.h, p.y));
        }
    }

    calculateInteractions() {
        const len = this.points.length;
        for (let i = 0; i < len - 1; i++) {
            const p1 = this.points[i];
            for (let j = i + 1; j < len; j++) {
                const p2 = this.points[j];
                const interaction = p1.getPointInteraction(p2);
                if (interaction) {
                    this.draw_buffer.lineBuffer.push(interaction);
                }
            }
        }
    }

    draw() {
        this.ctx.clearRect(0, 0, this.w, this.h);
        this.evolvePoints();
        this.calculateInteractions();
        this.draw_buffer.dumpLines();
        this.draw_buffer.dumpPoints();
    }
}

class ParticleCanvas {
    canvas: HTMLCanvasElement | null;
    ctx: CanvasRenderingContext2D | null;
    lastHeight: number;
    lastWidth: number;
    simulator: Simulator | undefined;
    render: {
        draw: boolean;
        need_initialize: boolean;
        delay_after: number;
        last_changed_time: number;
    };
    resizeListener: (() => unknown) | undefined;
    tid: number | undefined;

    constructor(canvasRef: HTMLCanvasElement | null) {
        this.canvas = canvasRef;
        this.initializeCanvas();
        this.ctx = this.canvas!.getContext("2d");

        this.simulator = undefined;

        this.lastWidth = window.innerWidth;
        this.lastHeight = window.innerHeight;

        this.registerListener();

        this.render = {
            draw: true,
            need_initialize: true,
            delay_after: 0.2,
            last_changed_time: 0,
        };

        this.pendingRender();
    }

    updateCanvasSize() {
        if (!this.canvas) return;
        this.canvas.width = window.innerWidth
            || document.documentElement.clientWidth
            || document.body.clientWidth;
        this.canvas.height = window.innerHeight
            || document.documentElement.clientHeight
            || document.body.clientHeight;
    }

    cancelTimerOrAnimationFrame(tid: NodeJS.Timeout | number | undefined) {
        if (typeof tid === "number") {
            cancelAnimationFrame(tid);
        } else if (tid) {
            clearTimeout(tid);
        }
    }

    resetRenderInfo() {
        this.render.last_changed_time = Date.now();
        this.render.draw = false;
        this.render.need_initialize = true;
    }

    registerListener() {
        const resizeThreshold = 50; // Threshold for resize to trigger a reset

        this.resizeListener = () => {
            // Get the current width and height
            const currentWidth = window.innerWidth;
            const currentHeight = window.innerHeight;

            // Check if the change in size is greater than the threshold
            if (Math.abs(currentWidth - this.lastWidth) > resizeThreshold ||
                Math.abs(currentHeight - this.lastHeight) > resizeThreshold) {

                if (this.tid) this.cancelTimerOrAnimationFrame(this.tid);
                this.updateCanvasSize();
                this.resetRenderInfo();
                this.render.draw = true;
                this.pendingRender();

                // Update the last width and height
                this.lastWidth = currentWidth;
                this.lastHeight = currentHeight;
            }
        };

        window.addEventListener("resize", this.resizeListener);
    }


    initializeCanvas() {
        if (!this.canvas) return;
        this.canvas.style.cssText = particleCanvasStyle;
        this.updateCanvasSize();
    }

    async pendingRender() {
        if (this.render.draw && this.canvas) {
            if (this.render.need_initialize) {

                this.simulator = new Simulator(
                    this.ctx!,
                    this.canvas.width,
                    this.canvas.height,
                );

                this.render.need_initialize = false;
            }
            this.requestFrame();
        } else if (Date.now() - this.render.last_changed_time > 500) {
            this.render.draw = true;
            this.pendingRender();
        } else { // wait until ready
            setTimeout(() => {
                this.pendingRender();
            }, this.render.delay_after * 1000);
        }
    }

    requestFrame() {
        if (!this.simulator) return;
        this.simulator.draw();
        this.tid = requestAnimationFrame(() => { this.pendingRender(); });
    }

    destroy() {
        if (this.tid) this.cancelTimerOrAnimationFrame(this.tid);
        if (this.resizeListener) window.removeEventListener("resize", this.resizeListener);
    }
}

interface NeonBoxProps extends BoxProps {
    isVisible: boolean;
}
const NeonBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isVisible",
})<NeonBoxProps>(({ isVisible }) => ({
    display: isVisible ? "block" : "none",
    transition: "display 0.5s",
    zIndex: PAGE_LAYERS.Neon,
}));

interface NeonBackgroundProps {
    isVisible: boolean;
}

/**
 * Custom background for hero section. 
 * Contains neon blobs and particles
 */
export function NeonBackground({
    isVisible,
}: NeonBackgroundProps) {
    const canvasRef = useRef<HTMLCanvasElement>(null);
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    useEffect(function handleCanvasInstance() {
        const canvasInstance = new ParticleCanvas(canvasRef.current);
        return () => {
            canvasInstance?.destroy();
        };
    }, []);

    return (
        <NeonBox isVisible={isVisible}>
            <canvas ref={canvasRef} />
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

export function StarryBackground({
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

type EarthPosition = keyof typeof earthPositions;
type ScrollDirection = "up" | "down";
type ScrollInfo = {
    currScrollPos: number;
    lastScrollPos: number;
    scrollDirection: ScrollDirection;
}

// Used for scroll snapping and url hash
const SLIDE_IDS = {
    slide1: "revolutionize-workflow",
    slide2: "chats",
    slide3: "routines",
    slide4: "teams",
    slide5: "sky-is-limit",
    slide6: "get-started",
} as const;
const SLIDE_CONTAINER_IDS = {
    neon: "neon-container",
    sky: "sky-container",
};

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
    zIndex: PAGE_LAYERS.Earth,
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
            const earthHorizonSlide = document.getElementById(SLIDE_IDS.slide5);
            const earthFullSlide = document.getElementById(SLIDE_IDS.slide6);
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
            <NeonBackground isVisible={earthPosition === "hidden"} />
            <StarryBackground isVisible={earthPosition !== "hidden"} />
            <ScrollBox ref={scrollBoxRef}>
                <TopBar
                    display={display}
                    onClose={onClose}
                    titleBehaviorMobile="ShowIn"
                />
                <SlideContainer id={SLIDE_CONTAINER_IDS.neon}>
                    <SlideContent1 id={SLIDE_IDS.slide1}>
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
                    <SlideContent id={SLIDE_IDS.slide2}>
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
                    <SlideContent id={SLIDE_IDS.slide3}>
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
                    <SlideContent id={SLIDE_IDS.slide4}>
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
                </SlideContainer>
                <SlideContainer id={SLIDE_CONTAINER_IDS.sky}>
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
                    <SlideContent id={SLIDE_IDS.slide5}>
                        <Slide5Title variant='h2'>The Sky is the Limit</Slide5Title>
                        <Slide5Text>
                            By combining bots, routines, and teams, we&apos;re paving the way for an automated and transparent economy - accessible to all.
                        </Slide5Text>
                    </SlideContent>
                    <SlideContent id={SLIDE_IDS.slide6}>
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
                </SlideContainer>
                <Footer />
            </ScrollBox>
        </PageContainer>
    );
}

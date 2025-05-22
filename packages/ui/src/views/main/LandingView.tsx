/* eslint-disable no-magic-numbers */
import { LINKS, PaymentType, SOCIALS } from "@local/shared";
import { Box, type BoxProps, Button, Grid, Stack, Tooltip, Typography, keyframes, styled, useTheme } from "@mui/material";
import { type ReactNode, useContext, useEffect, useMemo, useRef, useState } from "react";
import AiDrivenConvo from "../../assets/img/AiDrivenConvo.png";
import CollaborativeRoutines from "../../assets/img/CollaborativeRoutines.webp";
import Earth from "../../assets/img/Earth.svg";
import { default as OrganizationalManagement, default as TourThumbnail } from "../../assets/img/OrganizationalManagement.webp";
import Blob1 from "../../assets/img/blob1.svg";
import Blob2 from "../../assets/img/blob2.svg";
import { PageContainer } from "../../components/Page/Page.js";
import { BreadcrumbsBase } from "../../components/breadcrumbs/BreadcrumbsBase.js";
import { Footer } from "../../components/navigation/Footer.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SnackSeverity } from "../../components/snacks/BasicSnack/BasicSnack.js";
import { SessionContext } from "../../contexts/session.js";
import { useStripe } from "../../hooks/useStripe.js";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { IconCommon, IconService } from "../../icons/Icons.js";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { ScrollBox, SlideIconButton } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { darkPalette } from "../../utils/display/theme.js";
import { PubSub } from "../../utils/pubsub.js";
import { BillingCycle, BillingCycleToggle, CreditDialog, PricingTierType, PricingTiers } from "../ProView/ProView.js";
import { type LandingViewProps } from "./types.js";

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

// Blob properties
/** Blur factor for odd-numbered blobs */
const BLUR_DIVISOR_ODD = 16;
/** Blur factor for even-numbered blobs */
const BLUR_DIVISOR_EVEN = 50;
const ODD_SIZE_BASE = 0.4;
const EVEN_SIZE_BASE = 0.6;
const RANDOM_SIZE_RANGE = 0.3;
const POSITION_OFFSET_MIN = 0.1;
const POSITION_OFFSET_RANGE = 0.8;
const COOL_HUE_MIN = 120; // Green
const COOL_HUE_MAX = 300; // Purple

// Particle properties
/** The number of particles. (a number lesser than 1000 is recommended under regular settings) */
const POINT_COUNT_MOBILE = 25;
const POINT_COUNT_DESKTOP = 50;
/** Minimum point size */
const POINT_SIZE_MIN = 2;
/** Maximum point size */
const POINT_SIZE_MAX = 8;
/** Minimum line width */
const LINE_WIDTH_MIN = 1;
/** Maximum line width */
const LINE_WIDTH_MAX = 5;
const COLOR = "100, 100, 100";
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
                    hueRotate: COOL_HUE_MIN + Math.random() * (COOL_HUE_MAX - COOL_HUE_MIN),
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

            const path = pathMap.get(styleKey);
            if (!path) continue;
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
    numPoints: number;
    points: Point[];
    draw_buffer: DrawBuffer;
    w: number;
    h: number;

    constructor(ctx: CanvasRenderingContext2D, w: number, h: number, numPoints: number) {
        this.ctx = ctx;
        this.w = w;
        this.h = h;
        this.numPoints = numPoints;
        this.draw_buffer = new DrawBuffer(this.ctx);
        this.points = [];

        this.initializePoints();
    }

    initializePoints(): void {
        for (let i = 0; i < this.numPoints; i++) this.genNewPoint();
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
    numPoints: number;
    simulator: Simulator | undefined;
    render: {
        draw: boolean;
        need_initialize: boolean;
        delay_after: number;
        last_changed_time: number;
    };
    resizeListener: (() => unknown) | undefined;
    tid: number | undefined;

    constructor(canvasRef: HTMLCanvasElement | null, numPoints: number) {
        this.canvas = canvasRef;
        this.numPoints = numPoints;

        this.initializeCanvas();
        this.ctx = this.canvas?.getContext("2d") ?? null;

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
                    this.numPoints,
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
function NeonScene({
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
const Slide1StartButton = styled(PulseButton)(({ theme }) => ({
    fontSize: "1.5rem",
    textTransform: "none",
    width: "fit-content",
    [theme.breakpoints.down("md")]: {
        fontSize: "1.25rem",
    },
}));
const Slide1Button = styled(Slide1StartButton)(() => ({
    animation: "none",
    background: "#017d5366",
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
const Slide6StartButton = styled(Slide1StartButton)(({ theme }) => ({
    zIndex: 6,
    width: "min(300px, 40vw)",
    marginTop: theme.spacing(2),
    marginBottom: theme.spacing(2),
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
        openLink(setLocation, LINKS.Signup); //TODO: Change to popup that shows app download options and instructions for installing the app from the website
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
            <PageContainer size="fullSize">
                <NeonScene isVisible={earthPosition === "hidden"} />
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
                                            <Slide1StartButton
                                                variant="outlined"
                                                color="secondary"
                                                onClick={toSignUp}
                                            >Start for free</Slide1StartButton>
                                            <Slide1Button
                                                variant="outlined"
                                                color="secondary"
                                                onClick={toApp}
                                            >Get the app</Slide1Button>
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
                                    <SlideTitle variant='h2'>AI Coworkers, Ready on Demand</SlideTitle>
                                    <SlideText variant="h3">
                                        Our bots work around the clock to handle your repetitive tasks, so you can focus on what matters most.
                                    </SlideText>
                                    <SlideText variant="h3">
                                        Whether it&apos;s managing projects, automating workflows, or answering common questions, our AI bots are here to make your life easier and your business run smoother.
                                    </SlideText>
                                </RightGridContent>
                            </HalfWidth>
                        </FullWidth>
                        <FullWidth id={ELEMENT_IDS.LandingViewSlideRoutines}>
                            <HalfWidth>
                                <LeftGridContent>
                                    <SlideTitle variant='h2'>Build Consistent, Automated Workflows</SlideTitle>
                                    <SlideText variant="h3">
                                        Create reusable workflows that ensure your AI-driven business operates smoothly, every time.
                                    </SlideText>
                                    <SlideText variant="h3">
                                        Manage your business, create content, and more, in a way that is repeatable and reliable.
                                    </SlideText>
                                    <SlideText variant="h3">
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
                                    <SlideTitle variant='h2'>Manage Teams Like a Pro</SlideTitle>
                                    <SlideText variant="h3">
                                        With AI-powered routines and bots, you can handle the workload of an entire team on your own—no humans required.
                                    </SlideText>
                                    <SlideText variant="h3">
                                        Perfect for solo developers or small businesses, Vrooli lets you automate and manage everything from operations to communication, freeing up your time to focus on growth.
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
                                <SlideTitle variant='h2'>Supercharge Your Workflow</SlideTitle>
                                <SlideText>Select the perfect plan for your needs.</SlideText>
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
                                <Slide6StartButton
                                    aria-label="I'm ready!"
                                    variant="outlined"
                                    color="secondary"
                                    onClick={toSignUp}
                                    startIcon={<IconCommon
                                        decorative
                                        fill="white"
                                        name="Launch"
                                    />}
                                >I&apos;m ready!</Slide6StartButton>
                                <ExternalLinksBox>
                                    <Tooltip title="Check out our code" placement="bottom">
                                        <SlideIconButton
                                            aria-label="Check out our code"
                                            onClick={toGitHub}
                                        >
                                            <IconService
                                                decorative
                                                fill='#0fa'
                                                name="GitHub"
                                            />
                                        </SlideIconButton>
                                    </Tooltip>
                                    <Tooltip title="Follow us on X/Twitter" placement="bottom">
                                        <SlideIconButton
                                            aria-label="Follow us on X/Twitter"
                                            onClick={toX}
                                        >
                                            <IconService
                                                decorative
                                                fill='#0fa'
                                                name="X"
                                            />
                                        </SlideIconButton>
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

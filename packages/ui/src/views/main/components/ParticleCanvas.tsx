/* eslint-disable no-magic-numbers */
// Particle animation system extracted from LandingView for code splitting

const RADIANS_IN_CIRCLE = Math.PI * 2;

// Particle properties
/** The maximum moving speed of a particle in x or y coordinate can has in each frame. (in pixels) */
const POINT_MAX_SPEED = 0.05;
/** The maximum length of a line (i.e. the interaction radius of a particle). (in pixels) */
const INTERACTION_DISTANCE = 200;
/** Maximum allowed alpha value for lines between points */
const ALPHA_MIN = 0.5;
/** Minimum allowed alpha value for lines between points */
const ALPHA_MAX = 0.1;
/** Minimum point size */
const POINT_SIZE_MIN = 2;
/** Maximum point size */
const POINT_SIZE_MAX = 8;
/** Minimum line width */
const LINE_WIDTH_MIN = 1;
/** Maximum line width */
const LINE_WIDTH_MAX = 5;
const COLOR = "100, 100, 100";

const particleCanvasStyle = "position:fixed;top:0;left:0;pointer-events:none;";

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

export class ParticleCanvas {
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

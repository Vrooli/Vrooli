import { useTheme } from "@mui/material";
import { RandomBlobs } from "components/RandomBlobs/RandomBlobs";
import { useWindowSize } from "hooks/useWindowSize";
import { useEffect, useRef } from "react";
import { SlideContainer } from "styles";
import { SlideContainerNeonProps } from "../types";

const blackRadial = "radial-gradient(circle, rgb(6 6 46) 12%, rgb(1 1 36) 52%, rgb(3 3 20) 80%)";
const slideStyle = {
    background: blackRadial,
    backgroundAttachment: "fixed",
    color: "white",
    zIndex: 5,
} as const;

type Config = {
    X_CHUNK: number;
    Y_CHUNK: number;
};

type Line = {
    buffer: number[][][];
};

type Arc = {
    buffer: number[][];
};

type InterWithPoint = {
    type: "l" | "a";
    a?: number;
    pos_info: number[];
}

/** The number of particles. (a number lesser than 1000 is recommended under regular settings) */
const POINT_COUNT = 100;
/** The interaction radius of a particle. (in pixels) */
const POINT_DIST = 150;
/** Minimum point size */
const POINT_SIZE_MIN = 2;
/** Maximum point size */
const POINT_SIZE_MAX = 5;
/** The line-width multiplier for the line between two particles. If set to 0, the line will not be rendered. (any number between 0.5 ~ 1.5 is recommended) */
const LINE_WIDTH_MULTIPLIER = 1;
const COLOR = "255, 255, 255";
const OPACITY = "0.5";
/** The maximum moving speed of a particle in x or y coordinate can has in each frame. (in pixels) */
const POINT_MAX_SPEED = 0.04;
/** The deceleration rate of a particle after it succeeds it's maximum moving speed. */
const POINT_SLOW_DOWN_RATE = 0.8;
/** The ratio of the width or height of the chunk divided by the interaction radius of particle. (a number greater than 1 means lossless computing. 0.8 is recommended) */
const CHUNK_SIZE_CONSTANT = 0.8;
/** The number of particle that a chunk can contain. */
const CHUNK_CAPACITY = 15;
/** Multiplier for adding/removing decimals */
const DECIMAL_MULTIPLIER = 100;

const canvasStyle = `position:fixed;top:0;left:0;opacity:${OPACITY}`;

function rand(min: number, max: number) { return (max - min) * Math.random() + min; }

class Point {
    x: number;
    y: number;
    vx: number;
    vy: number;
    size: number;
    pointer_inter: boolean;

    constructor(w: number, h: number) {
        this.x = rand(0, w);
        this.y = rand(0, h);
        this.vx = rand(-1, 1);
        this.vy = rand(-1, 1);

        this.size = rand(POINT_SIZE_MIN, POINT_SIZE_MAX);
        this.pointer_inter = false;
    }

    evolve(): void {
        this.x += this.vx;
        this.y += this.vy;

        if (Math.abs(this.vx) > POINT_MAX_SPEED)
            this.vx *= POINT_SLOW_DOWN_RATE;
        if (Math.abs(this.vy) > POINT_MAX_SPEED)
            this.vy *= POINT_SLOW_DOWN_RATE;
    }

    getAlpha(dist: number): number {
        return Math.round((Math.max(Math.min(1.2 - dist / POINT_DIST, 1), 0.2) * 10)) / 10;
    }

    getPointInteraction(p: { x: number, y: number }): InterWithPoint | null {
        const dx = p.x - this.x, dy = p.y - this.y;
        const d = Math.hypot(dx, dy);

        if (d > POINT_DIST || d < 1) return null;

        return {
            type: "l",
            a: this.getAlpha(d),
            pos_info: [this.x, this.y, p.x, p.y],
        };
    }
}

class Chunk {
    x: number;
    y: number;
    w: number;
    h: number;
    points: Point[];
    traversed: boolean;

    constructor(x: number, y: number, w: number, h: number) {
        this.x = x;
        this.y = y;
        this.w = w;
        this.h = h;
        this.points = [];
        this.traversed = false;
    }
}

class Grid {
    c: Config;
    w: number;
    h: number;
    chunk_w: number;
    chunk_h: number;
    chunks: Chunk[][];

    constructor(w: number, h: number, config: Config) {
        this.c = config;

        this.w = w;
        this.h = h;

        this.chunk_w = this.w / this.c.X_CHUNK;
        this.chunk_h = this.h / this.c.Y_CHUNK;

        this.chunks = [];
        this.genChunks();
        this.initializePoints();
    }

    genChunks(): void {
        this.chunks = new Array(this.c.X_CHUNK).fill(null)
            .map(() => new Array(this.c.Y_CHUNK).fill(null));

        for (let i = 0; i < this.c.X_CHUNK; i++) {
            for (let j = 0; j < this.c.Y_CHUNK; j++) {
                this.chunks[i][j] = new Chunk(
                    this.chunk_w * i, this.chunk_h * j,
                    this.chunk_w, this.chunk_h,
                );
            }
        }
    }

    initializePoints(): void {
        for (let i = 0; i < POINT_COUNT; i++) this.genNewPoint();
    }

    genNewPoint(): void {
        const point = new Point(this.w, this.h);

        const chunk_x_num = Math.floor(point.x / this.chunk_w);
        const chunk_y_num = Math.floor(point.y / this.chunk_h);

        const t_chunk = this.chunks[chunk_x_num][chunk_y_num];

        // force push
        t_chunk.points.push(point);
    }
}

class DrawBuffer {
    ctx: CanvasRenderingContext2D;
    line: Line;
    arc: Arc;
    rad: number;

    constructor(ctx: CanvasRenderingContext2D) {
        this.ctx = ctx;
        this.ctx.fillStyle = `rgb(${COLOR})`;
        this.ctx.lineCap = "round";

        this.line = {
            buffer: new Array(11).fill(null).map(() => []), // 11 because getAlpha() returns 0.2 ~ 1.2
        };

        this.arc = {
            buffer: [],
        };

        this.rad = Math.PI * 2;
    }

    push(d_info: InterWithPoint) {
        if (!d_info) return;

        if (d_info.type === "l" && LINE_WIDTH_MULTIPLIER > 0) {
            this.line.buffer[10 * d_info.a!].push(d_info.pos_info);
        } else if (d_info.type === "a") {
            this.arc.buffer.push(d_info.pos_info);
        }
    }

    dumpLine() {
        for (let i = 0; i < 11; i++) this.dumpLineSlot(i);
    }

    dumpLineSlot(slot_num: number) {
        const info = this.line.buffer[slot_num];

        this.ctx.beginPath();

        const alpha = slot_num / 10;
        this.ctx.lineWidth = alpha * LINE_WIDTH_MULTIPLIER;
        this.ctx.strokeStyle = `rgba(${COLOR},${alpha})`;

        for (let i = 0, len = info.length; i < len; i++) {
            const pos_info = info[i];

            this.ctx.moveTo(pos_info[0], pos_info[1]);
            this.ctx.lineTo(pos_info[2], pos_info[3]);
        }
        this.ctx.stroke();

        this.line.buffer[slot_num].length = 0;
    }

    dumpArc() {
        this.ctx.beginPath();
        for (let i = 0; i < this.arc.buffer.length; i++) {
            const pos_info = this.arc.buffer[i];
            this.ctx.moveTo(pos_info[0], pos_info[1]);
            this.ctx.arc(pos_info[0], pos_info[1], pos_info[2], 0, this.rad);
        }
        this.ctx.fill();

        this.arc.buffer.length = 0;
    }
}

class Simulator {
    c: Config;
    ctx: CanvasRenderingContext2D;
    grid: Grid;
    draw_buffer: DrawBuffer;
    chunkBoundaries: Int16Array;

    constructor(ctx, grid: Grid, config: Config) {
        this.c = config;
        this.ctx = ctx;

        this.grid = grid;

        this.draw_buffer = new DrawBuffer(this.ctx);

        this.chunkBoundaries = new Int16Array(this.c.X_CHUNK * this.c.Y_CHUNK * 2);
        this.initializeChunkBoundaries();
    }

    initializeChunkBoundaries() {
        for (let ci = 0; ci < this.c.X_CHUNK; ci++) {
            for (let cj = 0; cj < this.c.Y_CHUNK; cj++) {
                const chunk = this.grid.chunks[ci][cj];
                const index = (ci * this.c.Y_CHUNK + cj) * 2;
                this.chunkBoundaries[index] = (chunk.x + chunk.w) * DECIMAL_MULTIPLIER; //Remove decimal places
                this.chunkBoundaries[index + 1] = (chunk.y + chunk.h) * DECIMAL_MULTIPLIER; //Remove decimal places
            }
        }
    }

    async traverse() {
        for (let ci = 0; ci < this.c.X_CHUNK + 4; ci++) {
            // Calculate interactions for the current chunk
            if (ci < this.c.X_CHUNK) await this.calVerticalInteraction(ci);
            // Evolve chunks that are two steps behind
            if (ci - 2 >= 0 && ci - 2 < this.c.X_CHUNK) await this.evolveVerticalChunks(ci - 2);
            // Update chunks that are four steps behind
            if (ci - 4 >= 0 && ci - 4 < this.c.X_CHUNK) await this.updateVerticalChunks(ci - 4);
        }
        // Dump buffers
        this.draw_buffer.dumpLine();
        this.draw_buffer.dumpArc();
    }

    async calVerticalInteraction(ci: number) {
        for (let cj = 0; cj < this.c.Y_CHUNK; cj++) {
            const chunk = this.grid.chunks[ci][cj];

            this.calLocalInteraction(chunk);

            // temp var
            const right_x = chunk.x + chunk.w;
            const right_y = chunk.y + chunk.h;

            // calculate interaction in surrounding chunk
            for (let pi = 0; pi < chunk.points.length; pi++) {
                const loc_p = chunk.points[pi];
                // Calculate if point interaction range exceeds current chunk
                // The reason for not calculating left_dx is that
                // the traverse direction is left to right
                const right_dx = (loc_p.x + POINT_DIST >= right_x) ? 1 : 0;
                const left_dy = (loc_p.y - POINT_DIST < chunk.y) ? -1 : 0;
                const right_dy = (loc_p.y + POINT_DIST >= right_y) ? 1 : 0;
                for (let i = 0; i <= right_dx; i++) {
                    for (let j = left_dy; j <= right_dy; j++) {
                        if (i === 0 && j === 0) continue; // do not compute local chunk
                        if (
                            ci + i >= this.c.X_CHUNK
                            || cj + j < 0 || cj + j >= this.c.Y_CHUNK
                        ) continue; // out of range

                        const tar_chunk = this.grid.chunks[ci + i][cj + j];
                        if (tar_chunk.traversed) continue;

                        this.calSurroundingInteraction(tar_chunk, loc_p);
                    }
                }
            }
            chunk.traversed = true;
        }
    }

    calLocalInteraction(chunk: Chunk) {
        for (let i = 0; i < chunk.points.length - 1; i++) {
            const p = chunk.points[i];
            for (let j = i + 1; j < chunk.points.length; j++) {
                const tar_p = chunk.points[j];
                const interaction = p.getPointInteraction(tar_p);
                if (interaction) this.draw_buffer.push(interaction);
            }
        }
    }

    calSurroundingInteraction(tar_chunk: Chunk, local_p: Point) {
        for (let i = 0; i < tar_chunk.points.length; i++) {
            const interaction = local_p.getPointInteraction(tar_chunk.points[i]);
            if (interaction) this.draw_buffer.push(interaction);
        }
    }

    async evolveVerticalChunks(ci: number) {
        for (let cj = 0; cj < this.c.Y_CHUNK; cj++) {
            const chunk = this.grid.chunks[ci][cj];
            for (let pi = 0; pi < chunk.points.length; pi++) {
                const p = chunk.points[pi];
                this.draw_buffer.push({
                    type: "a",
                    pos_info: [p.x, p.y, p.size],
                });
                p.evolve();
            }
        }
    }

    // update what chunk is point currently in after evolving
    async updateVerticalChunks(ci: number) {
        const MAX_X = this.c.X_CHUNK - 1;
        const MAX_Y = this.c.Y_CHUNK - 1;

        for (let cj = 0; cj <= MAX_Y; cj++) {
            const chunk = this.grid.chunks[ci][cj];
            const boundariesIndex = (ci * this.c.Y_CHUNK + cj) * 2;
            const right_x = this.chunkBoundaries[boundariesIndex] / DECIMAL_MULTIPLIER; // Add decimal places back
            const right_y = this.chunkBoundaries[boundariesIndex + 1] / DECIMAL_MULTIPLIER; // Add decimal places back

            chunk.traversed = false; // reset status for next frame

            let keepCount = 0; // Number of points we're keeping in this chunk

            for (const cur_p of chunk.points) {
                if (cur_p.x >= chunk.x && cur_p.x < right_x && cur_p.y >= chunk.y && cur_p.y < right_y) {
                    chunk.points[keepCount++] = cur_p;
                    continue;
                }

                const chunk_dx = cur_p.x < chunk.x ? -1 : (cur_p.x >= right_x ? 1 : 0);
                const chunk_dy = cur_p.y < chunk.y ? -1 : (cur_p.y >= right_y ? 1 : 0);
                const new_chunk_x = ci + chunk_dx;
                const new_chunk_y = cj + chunk_dy;

                // boundary check
                if (new_chunk_x < 0 || new_chunk_x > MAX_X || new_chunk_y < 0 || new_chunk_y > MAX_Y) {
                    if (new_chunk_x < 0 || new_chunk_x > MAX_X) {
                        cur_p.x = Math.max(0, Math.min(this.grid.w - 1, cur_p.x));
                        cur_p.vx *= -1;
                    }
                    if (new_chunk_y < 0 || new_chunk_y > MAX_Y) {
                        cur_p.y = Math.max(0, Math.min(this.grid.h - 1, cur_p.y));
                        cur_p.vy *= -1;
                    }
                    chunk.points[keepCount++] = cur_p;
                } else {
                    const new_chunk = this.grid.chunks[new_chunk_x][new_chunk_y];
                    if (new_chunk.points.length < CHUNK_CAPACITY) {
                        new_chunk.points.push(cur_p);
                    } else {
                        this.grid.genNewPoint();
                    }
                }
            }
            chunk.points.length = keepCount; // Adjust the length of the points array to remove points moved out
        }
    }

    draw() {
        this.ctx.clearRect(0, 0, this.grid.w, this.grid.h);
        this.traverse();
    }
}

class ParticleCanvas {
    c: Config;
    canvas: HTMLCanvasElement | null;
    ctx: CanvasRenderingContext2D | null;
    grid: Grid | undefined;
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
    tid: NodeJS.Timeout | number | undefined;
    constructor(canvasRef: HTMLCanvasElement | null) {
        this.c = {
            X_CHUNK: 10,
            Y_CHUNK: 10,
        };

        this.canvas = canvasRef;
        this.initializeCanvas();
        this.ctx = this.canvas!.getContext("2d");

        this.grid = undefined; // chunk manager
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
        this.canvas.style.cssText = canvasStyle;
        this.updateCanvasSize();
    }

    optimizeChunkSize() {
        if (!this.canvas) return;
        const opti_size = Math.floor(POINT_DIST * CHUNK_SIZE_CONSTANT);
        console.log("Optimized chunk size:", opti_size);
        const calOpti = (dimension: number) => {
            const diff = (num_of_chunks: number) => Math.abs(dimension / num_of_chunks - opti_size);
            const test_num = dimension / opti_size;
            if (diff(Math.floor(test_num)) < diff(Math.ceil(test_num))) return Math.floor(test_num);
            else return Math.ceil(test_num);
        };
        this.c.X_CHUNK = calOpti(this.canvas.width);
        this.c.Y_CHUNK = calOpti(this.canvas.height);
        console.log("X_CHUNK", this.c.X_CHUNK);
        console.log("Y_CHUNK", this.c.Y_CHUNK);
    }

    async pendingRender() {
        if (this.render.draw && this.canvas) {
            if (this.render.need_initialize) {
                this.optimizeChunkSize();

                this.grid = new Grid(this.canvas.width, this.canvas.height, this.c);
                this.simulator = new Simulator(this.ctx, this.grid, this.c);

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

/**
 * Custom slide container for hero section. 
 * Background is black with neon blobs that move around and grow/shrink
 */
export function SlideContainerNeon({
    id,
    children,
}: SlideContainerNeonProps) {
    const canvasRef = useRef<HTMLCanvasElement>(null);
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    useEffect(() => {
        const canvasInstance = new ParticleCanvas(canvasRef.current);
        return () => {
            canvasInstance?.destroy();
        };
    }, []);

    return (
        <SlideContainer
            id={id}
            key={id}
            sx={slideStyle}
        >
            <canvas ref={canvasRef} />
            <RandomBlobs numberOfBlobs={isMobile ? 5 : 8} />
            {children}
        </SlideContainer>
    );
}

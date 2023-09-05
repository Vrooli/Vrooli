import { Box, keyframes } from "@mui/material";
import Blob1 from "assets/img/blob1.svg";
import Blob2 from "assets/img/blob2.svg";
import { useEffect } from "react";
import { SlideContainer } from "styles";
import { SlideContainerNeonProps } from "../types";

const blackRadial = "radial-gradient(circle, rgb(6 6 46) 12%, rgb(1 1 36) 52%, rgb(3 3 20) 80%)";

// Animation for blob1
// Moves up and grows, then moves down to the right and shrinks.
// Then it moves to the left - while continuing to shrink- until it reaches the starting position.
const blob1Animation = keyframes`
    0% {
        transform: translateY(0) scale(0.5);
        filter: hue-rotate(0deg) blur(150px);
    }
    33% {
        transform: translateY(-160px) scale(0.9) rotate(-120deg);
        filter: hue-rotate(30deg) blur(150px);
    }
    66% {
        transform: translate(50px, 0px) scale(0.6) rotate(-200deg);
        filter: hue-rotate(60deg) blur(150px);
    }
    100% {
        transform: translate(0px, 0px) scale(0.5) rotate(0deg);
        filter: hue-rotate(0deg) blur(150px);
    }
`;

// Animation for blob2
// Moves to the right and changes hue, then moves back to the left and turns its original color.
const blob2Animation = keyframes`
    0% {
        transform: translateX(0) scale(1);
        filter: hue-rotate(0deg) blur(50px);
    }
    50% {
        transform: translateX(150px) scale(1.2);
        filter: hue-rotate(-50deg) blur(50px);
    }
    100% {
        transform: translateX(0) scale(1);
        filter: hue-rotate(0deg) blur(50px);
    }
`;

type Config = {
    /** The maximum moving speed of a particle in x or y coordinate can has in each frame. (in pixels) */
    max_point_speed: number;
    /** The deceleration rate of a particle after it succeeds it's maximum moving speed. */
    point_slow_down_rate: number;
    /** The line-width multiplier for the line between two particles. If set to 0, the line will not be rendered. (any number between 0.5 ~ 1.5 is recommended) */
    line_width_multiplier: number;
    /** Number of frames to render per second. Will use browser default if not set. */
    render_rate?: number;
    X_CHUNK: number;
    Y_CHUNK: number;
};

type PointConfig = {
    max_speed: Config["max_point_speed"];
    slow_down: Config["point_slow_down_rate"];
}

type Line = {
    buffer: number[][][];
    max_cap: number;
    cur_cap: number;
};

type Arc = {
    buffer: number[][];
    max_cap: number;
    cur_cap: number;
};

/** The number of particles. (a number lesser than 1000 is recommended under regular settings) */
const POINT_COUNT = 100;
/** The interaction radius of a particle. (in pixels) */
const POINT_DIST = 150;
/** Minimum point size */
const POINT_SIZE_MIN = 2;
/** Maximum point size */
const POINT_SIZE_MAX = 5;
const COLOR = "255, 255, 255";
const OPACITY = "0.5";
const ZINDEX = 5;
/** The ratio of the width or height of the chunk divided by the interaction radius of particle. (a number greater than 1 means lossless computing. 0.8 is recommended) */
const CHUNK_SIZE_CONSTANT = 0.8;
/** The number of particle that a chunk can contain. */
const CHUNK_CAPACITY = 15;

const canvasStyle = `position:fixed;top:0;left:0;z-index:${ZINDEX};opacity:${OPACITY}`;


const rand = (min: number, max: number) => (max - min) * Math.random() + min;

class Point {
    c: PointConfig;
    x: number;
    y: number;
    vx: number;
    vy: number;
    size: number;
    pointer_inter: boolean;

    constructor(w: number, h: number, config: PointConfig) {
        this.c = config;

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

        if (Math.abs(this.vx) > this.c.max_speed)
            this.vx *= this.c.slow_down;
        if (Math.abs(this.vy) > this.c.max_speed)
            this.vy *= this.c.slow_down;
    }

    calAlpha(dist: number): number {
        return Number(Math.max(Math.min(1.2 - dist / POINT_DIST, 1), 0.2).toFixed(1));
    }

    calPointerForce(ratio: number): number {
        if (1 < ratio) {
            return Math.pow(ratio - 1, 2) * this.c.max_speed;
        } else if (0.5 < ratio) {
            return -Math.pow(1 - ratio, 2) * this.c.max_speed;
        }
        return 0;
    }

    calInterWithPoint(p: { x: number, y: number }): { type: string, a: number, pos_info: number[] } | void {
        const dx = p.x - this.x, dy = p.y - this.y;
        const d = Math.hypot(dx, dy);

        if (d > POINT_DIST || d < 1) return;

        return {
            type: "l",
            a: this.calAlpha(d),
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
    pc: PointConfig;
    w: number;
    h: number;
    chunk_w: number;
    chunk_h: number;
    chunks: Chunk[][];

    constructor(w: number, h: number, config: Config) {
        this.c = config;

        // point config
        this.pc = {
            max_speed: config.max_point_speed,
            slow_down: config.point_slow_down_rate,
        };

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
        const point = new Point(this.w, this.h, this.pc);

        const chunk_x_num = Math.floor(point.x / this.chunk_w);
        const chunk_y_num = Math.floor(point.y / this.chunk_h);

        const t_chunk = this.chunks[chunk_x_num][chunk_y_num];

        // force push
        t_chunk.points.push(point);
    }
}

class DrawBuffer {
    line_width_multiplier: number;
    ctx: CanvasRenderingContext2D;
    line: Line;
    arc: Arc;
    rad: number;

    constructor(ctx: CanvasRenderingContext2D, config: Config) {
        this.line_width_multiplier = config.line_width_multiplier;

        this.ctx = ctx;
        this.ctx.fillStyle = `rgb(${COLOR})`;
        this.ctx.lineCap = "round";

        this.line = {
            buffer: new Array(9).fill(null).map(() => []),
            max_cap: 1000,
            cur_cap: 0,
        };

        this.arc = {
            buffer: [],
            max_cap: 1000,
            cur_cap: 0,
        };

        this.rad = Math.PI * 2;
    }

    push(d_info: DInfo) {
        if (!d_info) return;

        if (d_info.type === "l" && this.line_width_multiplier > 0) {
            this.line.buffer[10 * d_info.a! - 2].push(d_info.pos_info);
            if (++this.line.cur_cap > this.line.max_cap) this.dumpLine();
        } else if (d_info.type === "a") {
            this.arc.buffer.push(d_info.pos_info);
            if (++this.arc.cur_cap > this.arc.max_cap) this.dumpArc();
        }
    }

    dumpLine() {
        for (let i = 0; i < 9; i++) this.dumpLineSlot(i);
        this.line.cur_cap = 0;
    }

    dumpLineSlot(slot_num: number) {
        const info = this.line.buffer[slot_num];

        this.ctx.beginPath();

        const alpha = (slot_num + 2) / 10;
        this.ctx.lineWidth = alpha * this.line_width_multiplier;
        this.ctx.strokeStyle = `rgba(${COLOR},${alpha})`;

        for (let i = 0, len = info.length; i < len; i++) {
            const pos_info = info[i];

            this.ctx.moveTo(pos_info[0], pos_info[1]);
            this.ctx.lineTo(pos_info[2], pos_info[3]);
        }
        this.ctx.stroke();

        this.line.buffer[slot_num] = [];
    }

    dumpArc() {
        this.ctx.beginPath();
        for (let i = 0; i < this.arc.buffer.length; i++) {
            const pos_info = this.arc.buffer[i];
            this.ctx.moveTo(pos_info[0], pos_info[1]);
            this.ctx.arc(pos_info[0], pos_info[1], pos_info[2], 0, this.rad);
        }
        this.ctx.fill();

        this.arc.buffer = [];
        this.arc.cur_cap = 0;
    }
}

class Simulator {
    c: Config;
    ctx: CanvasRenderingContext2D;
    grid: Grid;
    pointer: Point;
    draw_buffer: DrawBuffer;

    constructor(ctx, grid, pointer, config) {
        this.c = config;
        this.ctx = ctx;

        this.grid = grid;
        this.pointer = pointer;

        this.draw_buffer = new DrawBuffer(this.ctx, config);
    }

    async traverse() {
        for (let ci = 0; ; ci++) {
            const tasks = [];

            if (ci >= 0 && ci < this.c.X_CHUNK) tasks.push(this.calVerticalInteraction(ci));
            if (ci - 2 >= 0 && ci - 2 < this.c.X_CHUNK) tasks.push(this.evolveVerticalChunks(ci - 2));
            if (ci - 4 >= 0 && ci - 4 < this.c.X_CHUNK) tasks.push(this.updateVerticalChunks(ci - 4));
            if (tasks.length === 0) {
                this.draw_buffer.dumpLine();
                this.draw_buffer.dumpArc();
                break;
            }

            await Promise.all(tasks);
        }
    }

    async calVerticalInteraction(ci) {
        for (let cj = 0; cj < this.c.Y_CHUNK; cj++) {
            const chunk = this.grid.chunks[ci][cj];

            this.calLocalInteraction(chunk);

            // temp var
            const right_x = chunk.x + chunk.w;
            const right_y = chunk.y + chunk.h;

            // calculate interaction in surrounding chunk
            chunk.points.forEach(loc_p => {
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
            });
            chunk.traversed = true;
        }
    }

    calLocalInteraction(chunk) {
        for (let i = 0; i < chunk.points.length - 1; i++) {
            const p = chunk.points[i];

            for (let j = i + 1; j < chunk.points.length; j++) {
                const tar_p = chunk.points[j];

                this.draw_buffer.push(
                    p.calInterWithPoint(tar_p),
                );
                // deprecated calculation
                // tar_p.calInterWithPoint(p, false);
            }
        }
    }

    calSurroundingInteraction(tar_chunk, local_p) {
        tar_chunk.points.forEach(tar_p => {
            this.draw_buffer.push(
                local_p.calInterWithPoint(tar_p),
            );
            // deprecated calculation, since simulation in legacy versions
            // needs calculating the gravity of each particle
            // 
            // tar_p.calInterWithPoint(local_p, false);
        });
    }

    async evolveVerticalChunks(ci) {
        for (let cj = 0; cj < this.c.Y_CHUNK; cj++) {
            const chunk = this.grid.chunks[ci][cj];
            chunk.points.forEach(p => {
                this.draw_buffer.push({
                    type: "a",
                    pos_info: [p.x, p.y, p.size],
                });
                p.evolve();
            });
        }
    }

    // update what chunk is point currently in after evolving
    async updateVerticalChunks(ci) {
        for (let cj = 0; cj < this.c.Y_CHUNK; cj++) {
            const chunk = this.grid.chunks[ci][cj];
            chunk.traversed = false; // reset status for next frame

            // for temp
            const right_x = chunk.x + chunk.w;
            const right_y = chunk.y + chunk.h;

            const rmv_list: number[] = [];
            for (let i = 0; i < chunk.points.length; i++) {
                const cur_p = chunk.points[i];

                let chunk_dx = 0, chunk_dy = 0;
                if (cur_p.x < chunk.x) chunk_dx = -1;
                else if (cur_p.x >= right_x) chunk_dx = 1;

                if (cur_p.y < chunk.y) chunk_dy = -1;
                else if (cur_p.y >= right_y) chunk_dy = 1;

                if (chunk_dx === 0 && chunk_dy === 0) continue;

                const new_chunk_x = ci + chunk_dx;
                const new_chunk_y = cj + chunk_dy;

                // boundary check
                if (new_chunk_x < 0) {
                    cur_p.x *= -1;
                    cur_p.vx *= -1;
                } else if (new_chunk_x >= this.c.X_CHUNK) {
                    cur_p.x = 2 * this.grid.w - cur_p.x;
                    cur_p.vx *= -1;
                } else if (new_chunk_y < 0) {
                    cur_p.y *= -1;
                    cur_p.vy *= -1;
                } else if (new_chunk_y >= this.c.Y_CHUNK) {
                    cur_p.y = 2 * this.grid.h - cur_p.y;
                    cur_p.vy *= -1;
                } else { // move to new chunk, or to random location if full
                    rmv_list.push(i);

                    const new_chunk = this.grid.chunks[new_chunk_x][new_chunk_y];
                    if (new_chunk.points.length < CHUNK_CAPACITY)
                        new_chunk.points.push(cur_p);
                    else this.grid.genNewPoint();
                }
            }

            // remove in O(N) time
            let cur_rmv_ind = 0;
            chunk.points = chunk.points.filter((v, ind) => {
                if (cur_rmv_ind !== rmv_list.length && ind === rmv_list[cur_rmv_ind]) {
                    cur_rmv_ind++;
                    return false;
                } else {
                    return true;
                }
            });
        }
    }

    draw() {
        this.ctx.clearRect(0, 0, this.grid.w, this.grid.h);
        this.traverse();
    }
}

class CanvasNice {
    c: Config;
    canvas: HTMLCanvasElement | undefined;
    ctx: CanvasRenderingContext2D | null;
    grid: any; // Use actual type if known
    simulator: any; // Use actual type if known
    pointer: {
        x: number | null;
        y: number | null;
    };
    render: {
        draw: boolean;
        need_initialize: boolean;
        delay_after: number;
        last_changed_time: number;
    };
    tid: NodeJS.Timeout | number | undefined;
    frameCount: number | undefined; //TODO temporary
    totalTime: number | undefined; //TODO temporary
    constructor(config: Omit<Config, "X_CHUNK" | "Y_CHUNK">) {
        this.c = {
            ...config,
            X_CHUNK: 10,
            Y_CHUNK: 10,
        };

        this.canvas = undefined;
        this.initializeCanvas();
        this.ctx = this.canvas!.getContext("2d");

        this.grid = undefined; // chunk manager
        this.simulator = undefined;

        this.pointer = {
            x: null,
            y: null,
        };

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
        window.onresize = () => {
            if (this.tid) this.cancelTimerOrAnimationFrame(this.tid);

            this.updateCanvasSize();
            this.resetRenderInfo();
        };
    }

    initializeCanvas() {
        this.canvas = document.createElement("canvas");
        this.canvas.style.cssText = canvasStyle;
        this.updateCanvasSize();

        document.body.appendChild(this.canvas);
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
                this.simulator = new Simulator(
                    this.ctx, this.grid, this.pointer, this.c,
                );

                this.render.need_initialize = false;
                // console.log(`[c-nice.js] Canvas Size: ${this.canvas.width}*${this.canvas.height}`);
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

    //TODO performance measurement is temporary
    requestFrame() {
        const start = performance.now();

        this.simulator.draw();

        const end = performance.now();
        const timeTaken = end - start;

        this.frameCount = this.frameCount ? this.frameCount + 1 : 1;
        this.totalTime = this.totalTime ? this.totalTime + timeTaken : timeTaken;

        if (this.frameCount >= 10) {
            const averageTime = this.totalTime / this.frameCount;
            console.log(`Average time for 10 frames: ${averageTime.toFixed(2)}ms`);

            // Reset counters for the next batch of 10 frames
            // this.frameCount = 0;
            // this.totalTime = 0;
        }

        if (this.c.render_rate) {
            this.tid = setTimeout(
                () => { this.pendingRender(); },
                1000 / this.c.render_rate,
            );
        } else {
            this.tid = requestAnimationFrame(
                () => { this.pendingRender(); },
            );
        }
    }


    destroy() {
        if (this.tid) this.cancelTimerOrAnimationFrame(this.tid);
        if (this.canvas) document.body.removeChild(this.canvas);
    }
}

/**
 * Custom slide container for hero section. 
 * Background is black with neon blobs that move around and grow/shrink
 */
export const SlideContainerNeon = ({
    id,
    children,
    show,
    sx,
}: SlideContainerNeonProps) => {

    useEffect(() => {
        // Initialize CanvasNice
        const canvasInstance = new CanvasNice({
            point_slow_down_rate: 0.8,
            line_width_multiplier: 1,
            max_point_speed: 0.5,
            render_rate: 2,//30,
        });
        console.log("canvasInstance", canvasInstance);

        return () => {
            // Destroy CanvasNice instance
            canvasInstance?.destroy();
        };
    }, []);

    return (
        <SlideContainer
            id={id}
            key={id}
            sx={{
                // Set background and color
                background: blackRadial,
                backgroundAttachment: "fixed",
                color: "white",
                ...sx,
            }}
        >
            {/* Blob 1 */}
            <Box sx={{
                position: "fixed",
                pointerEvents: "none",
                bottom: -300,
                left: -175,
                width: "100%",
                height: "100%",
                zIndex: 2,
                opacity: show === false ? 0 : 0.5,
                transition: "opacity 1s ease-in-out",
            }}>
                <Box
                    component="img"
                    src={Blob1}
                    alt="Blob 1"
                    sx={{
                        width: "100%",
                        height: "100%",
                        animation: `${blob1Animation} 20s linear infinite`,
                    }}
                />
            </Box>
            {/* Blob 2 */}
            <Box sx={{
                position: "fixed",
                pointerEvents: "none",
                top: -154,
                right: -175,
                width: "100%",
                height: "100%",
                zIndex: 2,
                opacity: show === false ? 0 : 0.5,
                transition: "opacity 1s ease-in-out",
            }}>
                <Box
                    component="img"
                    src={Blob2}
                    alt="Blob 2"
                    sx={{
                        width: "100%",
                        height: "100%",
                        animation: `${blob2Animation} 20s linear infinite`,
                    }}
                />
            </Box>
            {children}
        </SlideContainer>
    );
};

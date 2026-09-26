export type ProbeTier = "full" | "reduced" | "still";

/** The capability ladder: one build, tiered at runtime. Selection never blocks first paint. */
export function probeTier(forced: string | null): ProbeTier {
  if (forced === "still" || forced === "reduced" || forced === "full") return forced;
  if (typeof window === "undefined") return "still";
  if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return "still";
  try {
    const probe = document.createElement("canvas");
    const gl = (probe.getContext("webgl2") ?? probe.getContext("webgl")) as WebGLRenderingContext | null;
    if (!gl) return "reduced";
    // A software or virtualised renderer reports WebGL but cannot hold a full
    // ambient scene: it would block the main thread every frame. Keep the still.
    const info = gl.getExtension("WEBGL_debug_renderer_info");
    const renderer = info ? String(gl.getParameter(info.UNMASKED_RENDERER_WEBGL) ?? "") : "";
    if (SOFTWARE_RENDERER.test(renderer)) return "still";
    const cores = navigator.hardwareConcurrency;
    const memory = (navigator as Navigator & { deviceMemory?: number }).deviceMemory ?? 8;
    const area = window.innerWidth * window.innerHeight;
    // Few cores, little memory or a very large panel: keep the lower element
    // counts and the 30 fps cap rather than risk a main-thread stall.
    if (cores < 4 || memory <= 2 || area > 4_000_000) return "reduced";
    return "full";
  } catch {
    return "reduced";
  }
}

/** Renderer names that mean the GPU is emulated in software. */
const SOFTWARE_RENDERER = /swiftshader|llvmpipe|softpipe|software|mesa|virtual|basic render/i;

/** Samples the drawn frame; a mounted scene that draws nothing is a failure, not a pass. */
export function frameIsBlank(ctx: CanvasRenderingContext2D, width: number, height: number): boolean {
  if (width < 2 || height < 2) return true;
  const { data } = ctx.getImageData(0, 0, width, height);
  let painted = 0;
  const stride = Math.max(4, Math.floor(data.length / 4 / 4000) * 4);
  for (let i = 3; i < data.length; i += stride) {
    if ((data[i] ?? 0) > 8) painted += 1;
  }
  return painted < 12;
}

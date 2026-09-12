export type ProbeTier = "full" | "reduced" | "still";

/** The capability ladder: one build, tiered at runtime. Selection never blocks first paint. */
export function probeTier(forced: string | null): ProbeTier {
  if (forced === "still" || forced === "reduced" || forced === "full") return forced;
  if (typeof window === "undefined") return "still";
  if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return "still";
  try {
    const probe = document.createElement("canvas");
    const gl = probe.getContext("webgl2") ?? probe.getContext("webgl");
    if (!gl) return "reduced";
    const cores = navigator.hardwareConcurrency;
    return cores >= 4 ? "full" : "reduced";
  } catch {
    return "reduced";
  }
}

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

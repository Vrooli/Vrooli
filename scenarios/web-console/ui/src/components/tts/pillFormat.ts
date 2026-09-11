/** m:ss, or "--:--" while the length is unknown. */
export function formatClock(seconds: number | null): string {
  if (seconds === null || !Number.isFinite(seconds)) return "--:--";
  const total = Math.round(seconds);
  return `${String(Math.floor(total / 60))}:${String(total % 60).padStart(2, "0")}`;
}

const RATE_STEPS = [1, 1.25, 1.5, 1.75, 2, 0.75];

/** The speed after `rate` in the chip's cycle; an off-cycle rate returns to 1×. */
export function nextRate(rate: number): number {
  return RATE_STEPS[(RATE_STEPS.indexOf(rate) + 1) % RATE_STEPS.length] ?? 1;
}

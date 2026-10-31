export interface CycleBeat {
  dwellSeconds?: number;
}

export interface BeatPosition {
  index: number;
  progress: number;
  startSeconds: number;
}

/** The factor the requested cycle applies to authored times (60 seconds is the reference). */
export function cycleScale(cycleSeconds: number): number {
  return Math.max(5, cycleSeconds) / 60;
}

/**
 * Each beat lasts its reading time, never less than its authored dwell, scaled to
 * the requested cycle (60 seconds is the reference).
 */
export function buildBeatDurations(beats: CycleBeat[], cycleSeconds: number, readingSeconds: number[] = []): number[] {
  if (!beats.length) return [];
  const scale = cycleScale(cycleSeconds);
  return beats.map((beat, index) => Math.max(1, beat.dwellSeconds ?? (60 / beats.length), readingSeconds[index] ?? 0) * scale);
}

/** The same beat at the same point through it, under new durations, so a beat that grows never moves the room to another beat. */
export function remapProgress(progress: number, from: number[], to: number[]): number {
  if (!from.length || from.length !== to.length) return progress;
  const total = to.reduce((sum, duration) => sum + duration, 0);
  if (!total) return progress;
  const { index, progress: within } = beatPositionAtProgress(progress, from);
  const start = to.slice(0, index).reduce((sum, duration) => sum + duration, 0);
  return (start + within * (to[index] ?? 0)) / total;
}

export function beatPositionAtProgress(progress: number, durations: number[]): BeatPosition {
  if (!durations.length) return { index: 0, progress: 0, startSeconds: 0 };
  const bounded = Math.max(0, Math.min(0.999999, progress));
  const total = durations.reduce((sum, duration) => sum + duration, 0);
  const elapsed = bounded * total;
  let startSeconds = 0;
  for (let index = 0; index < durations.length; index += 1) {
    const duration = durations[index] ?? 0;
    if (elapsed < startSeconds + duration || index === durations.length - 1) {
      return { index, startSeconds, progress: duration ? Math.max(0, Math.min(1, (elapsed - startSeconds) / duration)) : 0 };
    }
    startSeconds += duration;
  }
  return { index: durations.length - 1, startSeconds, progress: 1 };
}

export function progressAtBeat(index: number, durations: number[]): number {
  if (!durations.length) return 0;
  const safeIndex = Math.max(0, Math.min(durations.length - 1, index));
  const total = durations.reduce((sum, duration) => sum + duration, 0);
  const start = durations.slice(0, safeIndex).reduce((sum, duration) => sum + duration, 0);
  return total ? start / total : 0;
}

/** Whether moving from one cycle progress to the next would leave the current beat (or the room, at the end). */
export function crossesBeat(from: number, to: number, durations: number[]): boolean {
  if (to >= 1) return true;
  if (!durations.length) return false;
  return beatPositionAtProgress(to, durations).index !== beatPositionAtProgress(from, durations).index;
}

/**
 * The progress that draws the current beat's segment full without moving the
 * cycle into the next beat, so a held beat waits at the segment's end.
 */
export function beatEndProgress(index: number, durations: number[]): number {
  if (!durations.length) return 0;
  const next = index >= durations.length - 1 ? 1 : progressAtBeat(index + 1, durations);
  return Math.max(0, Math.min(0.999999, next - 1e-6));
}

export interface CycleTickInput {
  /** Monotonic clock reading in milliseconds. */
  now: number;
  /** When the current cycle interval started. */
  startedAt: number;
  /** The whole room's dwell in milliseconds. */
  dwellMs: number;
  /** The progress the rail currently shows. */
  progress: number;
  durations: number[];
  /** Whether a strip page or list pass is still holding the beat. */
  holding: boolean;
  /** When the current hold began, or null. */
  heldSince: number | null;
  maxHoldMs: number;
}

export interface CycleTick {
  progress: number;
  held: boolean;
  heldSince: number | null;
  /** The room has run out and the controller should advance to the next room. */
  navigate: boolean;
}

/**
 * One frame of the cycle clock. Progress follows the wall clock every frame, so
 * the rail never reads as frozen or stutters; a held beat waits at its
 * segment's end until the strip has been read, bounded by `maxHoldMs`.
 */
export function tickCycle({ now, startedAt, dwellMs, progress, durations, holding, heldSince, maxHoldMs }: CycleTickInput): CycleTick {
  if (dwellMs <= 0) return { progress, held: false, heldSince: null, navigate: false };
  const raw = (now - startedAt) / dwellMs;
  // A room with no beats is one unsegmented interval; the rail still shows time.
  if (!durations.length) return raw >= 1 ? { progress: 1, held: false, heldSince: null, navigate: true } : { progress: Math.max(0, Math.min(1, raw)), held: false, heldSince: null, navigate: false };
  const displayIndex = beatPositionAtProgress(progress, durations).index;
  const crossing = raw >= 1 || beatPositionAtProgress(raw, durations).index !== displayIndex;
  if (crossing && holding) {
    const since = heldSince ?? now;
    if (now - since < maxHoldMs) {
      return { progress: beatEndProgress(displayIndex, durations), held: true, heldSince: since, navigate: false };
    }
  }
  if (raw >= 1) return { progress: 1, held: false, heldSince: null, navigate: true };
  return { progress: Math.max(0, Math.min(1, raw)), held: false, heldSince: null, navigate: false };
}

export function parseBeat(value: string | null, count: number): number {
  if (count <= 0) return 0;
  const requested = Number.parseInt(value ?? "0", 10);
  return Number.isFinite(requested) ? Math.max(0, Math.min(count - 1, requested)) : 0;
}

/** Room identity changes start a fresh beat while retaining kiosk configuration. */
export function roomNavigationSuffix(search: string): string {
  const next = new URLSearchParams(search);
  next.delete("beat");
  const query = next.toString();
  return query ? `?${query}` : "";
}

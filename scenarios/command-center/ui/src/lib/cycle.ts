export interface CycleBeat {
  dwellSeconds?: number;
}

export interface BeatPosition {
  index: number;
  progress: number;
  startSeconds: number;
}

/** Build the authored beat durations after scaling them to the requested cycle. */
export function buildBeatDurations(beats: CycleBeat[], cycleSeconds: number): number[] {
  if (!beats.length) return [];
  const scale = Math.max(5, cycleSeconds) / 60;
  return beats.map((beat) => Math.max(1, beat.dwellSeconds ?? (60 / beats.length)) * scale);
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

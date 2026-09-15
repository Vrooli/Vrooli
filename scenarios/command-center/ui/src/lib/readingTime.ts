import type { BeatLayout, BoardRoom, Reading } from "./api";
import { pickHero } from "./hero";
import { readingsForBeat } from "./beat";
import { ladderOf, nextRung, rungsAfterNext } from "./ladder";

/**
 * Seconds a viewer needs per thing a beat shows, at the 60-second reference
 * cycle (UI-ARCHITECTURE §"Beats, layouts and list readings"). A line is one
 * row read across: a panel row, a blocker, a fact, a lane.
 */
export const READING_SECONDS = { headline: 7, line: 1.5, token: 0.5, tile: 0.4, stripCap: 4, beatCap: 45 } as const;

const PANEL_MAX_ROWS = 24;

function ladderGaps(hero: Reading): number {
  const ladder = ladderOf(hero)?.ladder;
  return ladder ? (ladder.unscheduled.length ? 1 : 0) + (ladder.unavailable?.length ?? 0) : 0;
}

/** What the hero shows beyond its headline, in seconds. */
function heroSeconds(hero: Reading, layout: BeatLayout): number {
  const { line, token } = READING_SECONDS;
  if (hero.kind === "panel") return Math.min(PANEL_MAX_ROWS, (hero.rows?.length ? hero.rows : hero.sample?.rows ?? []).length) * line;
  if (hero.kind !== "ladder") return 0;
  const ladder = ladderOf(hero)?.ladder;
  if (!ladder) return 0;
  const gaps = ladderGaps(hero) * line;
  if (layout === "wide") return ladder.reach.length * line + new Set(ladder.reach.map((unlock) => unlock.kind)).size * token + gaps;
  const rung = nextRung(ladder);
  if (!rung) return gaps;
  const opens = line + (rung.ramps.length + rung.streams.length + rung.audiences.length) * token;
  const blockers = Math.max(1, rung.blockers.length) * line;
  const facts = 2 * line;
  const then = rungsAfterNext(ladder).length ? line : 0;
  return opens + blockers + facts + then + gaps;
}

/** One beat's reading time: its headline, what the hero lists, and a capped glance across the strip. */
export function beatReadingSeconds(hero: Reading | null, layout: BeatLayout, supportingCount: number): number {
  if (!hero) return 0;
  const { headline, tile, stripCap, beatCap } = READING_SECONDS;
  return Math.min(beatCap, headline + heroSeconds(hero, layout) + Math.min(stripCap, supportingCount * tile));
}

/** Every beat's reading time, choosing hero and strip exactly as the room page does. A ladder beat has no strip. */
export function roomReadingSeconds(beats: NonNullable<BoardRoom["beats"]>, readings: Reading[]): number[] {
  return beats.map((beat) => {
    const beatReadings = readingsForBeat(beat, readings);
    const hero = pickHero(beatReadings, beat.hero);
    const supporting = hero?.kind === "ladder" ? 0 : beatReadings.length - (hero ? 1 : 0);
    return beatReadingSeconds(hero, beat.layout === "wide" ? "wide" : "standard", supporting);
  });
}

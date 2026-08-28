import { describe, expect, it } from "vitest";
import {
  MORPH_SCORE_THRESHOLD,
  geometryFromNodes,
  morphCompatibility,
  type SvgNode,
} from "./IconGeometry";

/**
 * A calibration record, not a unit test.
 *
 * `MORPH_SCORE_THRESHOLD` decides whether `morph="auto"` upgrades a crossfade to
 * a path morph, and a threshold invented at a desk is a guess. This file scores
 * real icon pairs — verbatim lucide node data and the registry glyphs — so the
 * constant is chosen against evidence and so a later change to the scoring
 * function has to confront every pair at once instead of one convenient
 * example. The printed table is the artifact; the assertions only pin the
 * verdicts that matter.
 */

const path = (d: string): SvgNode => ({ tag: "path", attrs: { d } });

const ICONS: Record<string, SvgNode[]> = {
  // Registry glyphs.
  menu: [path("M4 7h16"), path("M4 12h16"), path("M4 17h16")],
  close: [path("M6 6l12 12"), path("M18 6L6 18")],
  plus: [path("M12 5v14"), path("M5 12h14")],
  check: [path("M5 12l4 4L19 6")],
  chevronDown: [path("M6 9l6 6 6-6")],
  chevronRight: [path("M9 6l6 6-6 6")],
  search: [path("M11 4a7 7 0 1 0 0 14a7 7 0 1 0 0-14"), path("M16 16l4 4")],
  arrowStart: [path("M19 12H5"), path("M11 6l-6 6 6 6")],
  arrowEnd: [path("M5 12h14"), path("M13 6l6 6-6 6")],

  // Verbatim lucide, including the pair driving the web-console view toggle.
  MessageSquareText: [
    path("M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"),
    path("M13 8H7"),
    path("M17 12H7"),
  ],
  SquareTerminal: [
    path("m7 11 2-2-2-2"),
    path("M11 13h4"),
    { tag: "rect", attrs: { width: "18", height: "18", x: "3", y: "3", rx: "2", ry: "2" } },
  ],
  Play: [{ tag: "polygon", attrs: { points: "6 3 20 12 6 21 6 3" } }],
  Pause: [
    { tag: "rect", attrs: { x: "14", y: "4", width: "4", height: "16", rx: "1" } },
    { tag: "rect", attrs: { x: "6", y: "4", width: "4", height: "16", rx: "1" } },
  ],
  Sun: [
    { tag: "circle", attrs: { cx: "12", cy: "12", r: "4" } },
    path("M12 2v2"), path("M12 20v2"), path("m4.93 4.93 1.41 1.41"),
    path("m17.66 17.66 1.41 1.41"), path("M2 12h2"), path("M20 12h2"),
    path("m6.34 17.66-1.41 1.41"), path("m19.07 4.93-1.41 1.41"),
  ],
  Moon: [path("M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z")],
  Volume2: [
    { tag: "polygon", attrs: { points: "11 5 6 9 2 9 2 15 6 15 11 19 11 5" } },
    path("M15.54 8.46a5 5 0 0 1 0 7.07"),
    path("M19.07 4.93a10 10 0 0 1 0 14.14"),
  ],
  VolumeX: [
    { tag: "polygon", attrs: { points: "11 5 6 9 2 9 2 15 6 15 11 19 11 5" } },
    path("m22 9-6 6"), path("m16 9 6 6"),
  ],
  Copy: [
    { tag: "rect", attrs: { width: "14", height: "14", x: "8", y: "8", rx: "2", ry: "2" } },
    path("M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"),
  ],
  Eye: [
    path("M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"),
    { tag: "circle", attrs: { cx: "12", cy: "12", r: "3" } },
  ],
  EyeOff: [
    path("M9.88 9.88a3 3 0 1 0 4.24 4.24"),
    path("M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68"),
    path("M6.61 6.61A13.526 13.526 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61"),
    path("m2 2 20 20"),
  ],
};

/**
 * Every verdict below was settled by rendering the transition and looking at it.
 *
 * The first run of this survey disagreed with the author's predictions on three
 * of eleven pairs. Rather than tune `MORPH_SCORE_THRESHOLD` until it reproduced
 * those predictions — which would fit the function to an opinion instead of to
 * the geometry — each pair was rendered as a nine-frame filmstrip and reviewed.
 * The score was right on all three:
 *
 *   - `arrowStart → arrowEnd` (0.534, rejected). Predicted a clean mirror. The
 *     midpoint frames are a collapsed bowtie: linear interpolation drags the
 *     arrowhead through the shaft. Correctly rejected.
 *   - `Play → Pause` (0.650, accepted). Predicted a mess. The triangle narrows
 *     and splits into two bars, which is the transition media players ship.
 *   - `MessageSquareText → SquareTerminal` (0.862, accepted). Predicted a mess,
 *     on the reasoning that the two icons are unrelated. They are not: both are
 *     a rounded container plus two short interior marks, so the bubble's tail
 *     retracts into the frame and the text lines become the prompt. It is the
 *     best-looking morph in the table.
 *
 * `why` records what the frames actually show, not what was expected.
 */
const PAIRS: Array<{ from: string; to: string; expect: "morph" | "crossfade"; why: string }> = [
  { from: "menu", to: "close", expect: "morph", why: "the canonical hamburger fold" },
  { from: "plus", to: "check", expect: "morph", why: "short travel, same corner of the canvas" },
  { from: "chevronDown", to: "chevronRight", expect: "morph", why: "one stroke, pure rotation" },
  { from: "arrowStart", to: "arrowEnd", expect: "crossfade", why: "a mirror flip collapses through a degenerate arrowhead at the midpoint" },
  { from: "Volume2", to: "VolumeX", expect: "morph", why: "shared speaker body, differing tail" },
  { from: "Eye", to: "EyeOff", expect: "crossfade", why: "a slash appears and the lids restructure" },
  { from: "Play", to: "Pause", expect: "morph", why: "the triangle splits cleanly; the second bar grows from the split" },
  { from: "Sun", to: "Moon", expect: "crossfade", why: "nine strokes against one" },
  { from: "MessageSquareText", to: "SquareTerminal", expect: "morph", why: "both are a rounded container plus two interior marks; only the bubble tail differs" },
  { from: "search", to: "close", expect: "crossfade", why: "a circle has nowhere to go in a cross" },
  { from: "Copy", to: "check", expect: "crossfade", why: "unrelated construction" },
];

describe("morph compatibility survey", () => {
  const geometry = Object.fromEntries(
    Object.entries(ICONS).map(([name, nodes]) => [name, geometryFromNodes(nodes)]),
  );

  it("prints the calibration table", () => {
    const rows = PAIRS.map(({ from, to, expect: verdict, why }) => {
      const result = morphCompatibility(geometry[from]!, geometry[to]!);
      return {
        pair: `${from} → ${to}`,
        score: result.score.toFixed(3),
        travel: result.travel.toFixed(3),
        subpaths: `${result.fromSubpaths}→${result.toSubpaths}`,
        verdict: result.score > MORPH_SCORE_THRESHOLD ? "morph" : "crossfade",
        wanted: verdict,
        why,
      };
    });
    // eslint-disable-next-line no-console -- the table is this file's purpose.
    console.table(rows);
    expect(rows).toHaveLength(PAIRS.length);
  });

  for (const { from, to, expect: wanted, why } of PAIRS) {
    it(`classifies ${from} → ${to} as ${wanted} (${why})`, () => {
      const result = morphCompatibility(geometry[from]!, geometry[to]!);
      const verdict = result.score > MORPH_SCORE_THRESHOLD ? "morph" : "crossfade";
      expect(verdict, `score ${result.score.toFixed(3)}, travel ${result.travel.toFixed(3)}`)
        .toBe(wanted);
    });
  }

  /**
   * The threshold is only meaningful if the two classes actually separate. They
   * do, and by a clear margin: the highest-scoring crossfade is 0.534 and the
   * lowest-scoring morph is 0.588, so 0.55 sits inside a genuine gap rather
   * than slicing through a cluster. A scoring change that closes this gap has
   * made the signal worse even if every individual verdict still passes.
   */
  it("separates the two classes with the threshold inside the gap", () => {
    const scored = PAIRS.map(({ from, to, expect: wanted }) => ({
      wanted,
      score: morphCompatibility(geometry[from]!, geometry[to]!).score,
    }));
    const highestCrossfade = Math.max(
      ...scored.filter((row) => row.wanted === "crossfade").map((row) => row.score),
    );
    const lowestMorph = Math.min(
      ...scored.filter((row) => row.wanted === "morph").map((row) => row.score),
    );
    expect(highestCrossfade).toBeLessThan(lowestMorph);
    expect(MORPH_SCORE_THRESHOLD).toBeGreaterThan(highestCrossfade);
    expect(MORPH_SCORE_THRESHOLD).toBeLessThan(lowestMorph);
  });

  it("keeps every icon self-identical at the top of the range", () => {
    for (const [name, value] of Object.entries(geometry)) {
      expect(morphCompatibility(value, value).score, name).toBeCloseTo(1, 6);
    }
  });

  it("recovers the search circle that the previous parser dropped", () => {
    // Two subpaths, and the first spans the full 14-unit glyph rather than
    // collapsing to the 4-unit handle.
    const search = geometry.search!;
    expect(search.subpaths).toHaveLength(2);
    const xs = search.subpaths[0]!.points.map((p) => p.x);
    expect(Math.max(...xs) - Math.min(...xs)).toBeGreaterThan(13);
  });
});

/**
 * @libraryId react-component-library:RollingNumber
 * @displayName RollingNumber
 * @description A number drawn in an ink. The glyph box is identical in every ink so a metric going live never moves a pixel, and only the digits that changed roll.
 * @version 0.1.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:RollingNumber */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { useEffect, useRef, useState } from "react";
import { formatFigure } from "./format";
import { ProvenanceInkStyles, type Ink } from "@vrooli/react-component-library/ProvenanceInk/0";

export interface RollingNumberProps {
  value: number | null;
  /** Registry format vocabulary: integer, compact, currency, currency.compact, percent, percent.signed, minutes, duration.days. */
  format?: string;
  unit?: string;
  ink: Ink;
  /** wall is a hero read from across a room; display is a supporting reading. */
  scale: "wall" | "display";
  /** Shown instead of a number when there is none to show. */
  placeholder?: string;
  className?: string;
}

interface Glyph {
  char: string;
  changed: boolean;
}

const styles = `
  [data-rcl-figure] { display: inline-flex; align-items: baseline; font-family: var(--font-sans, system-ui, sans-serif); font-weight: 700; font-variant-numeric: tabular-nums lining-nums; letter-spacing: -0.03em; line-height: 0.92; white-space: nowrap; color: var(--color-foreground, #e8ecf3); }
  [data-rcl-figure="wall"] { font-size: var(--text-wall, clamp(5rem, 16vw, 20rem)); }
  [data-rcl-figure="display"] { font-size: var(--text-display, clamp(1.9rem, 3.4vw, 3.6rem)); letter-spacing: -0.02em; }
  [data-rcl-figure-affix] { font-size: 0.42em; font-weight: 600; letter-spacing: 0; margin: 0 0.08em; align-self: baseline; opacity: 0.85; }
  [data-rcl-figure-digits] { display: inline-flex; }
  [data-rcl-digit] { display: inline-block; }
  [data-rcl-digit="rolled"] { animation: rcl-figure-roll var(--dur-normal, 380ms) cubic-bezier(.2,.7,.2,1); }
  @keyframes rcl-figure-roll { from { transform: translateY(-0.55em); opacity: 0; } to { transform: none; opacity: 1; } }
  @media (prefers-reduced-motion: reduce) { [data-rcl-digit="rolled"] { animation: none; } }
`;

/**
 * A number drawn in an ink. The glyph box is identical in every ink so a
 * metric going live never moves a pixel, and only the digits that changed
 * roll; a whole-number crossfade reads as a flicker at distance.
 */
export const RollingNumber = withClassName(function RollingNumber({
  value,
  format,
  unit,
  ink,
  scale,
  placeholder = "—",
  className,
}: RollingNumberProps) {
  const formatted =
    value === null
      ? { prefix: "", text: placeholder, suffix: "" }
      : formatFigure(value, format, unit);
  const previous = useRef<string>(formatted.text);
  const [glyphs, setGlyphs] = useState<Glyph[]>(() =>
    Array.from(formatted.text).map((char) => ({ char, changed: false })),
  );
  const [rollKey, setRollKey] = useState(0);

  useEffect(() => {
    const before = previous.current;
    const after = formatted.text;
    if (before === after) return;
    previous.current = after;
    const padBefore = before.padStart(after.length, " ");
    setGlyphs(
      Array.from(after).map((char, i) => ({
        char,
        changed: padBefore[padBefore.length - after.length + i] !== char,
      })),
    );
    setRollKey((key) => key + 1);
  }, [formatted.text]);

  return (
    <>
      <StyleSheet name="rolling-number-1" css={styles} />
      <ProvenanceInkStyles />
      <span
        data-rcl-figure={scale}
        data-figure
        data-ink={ink}
        data-value={value ?? undefined}
        className={className}
      >
        {formatted.prefix ? <span data-rcl-figure-affix>{formatted.prefix}</span> : null}
        <span
          data-rcl-figure-digits
          aria-label={`${formatted.prefix}${formatted.text}${formatted.suffix}`}
        >
          {glyphs.map((glyph, i) => (
            <span
              key={`${rollKey}-${i}`}
              data-rcl-digit={glyph.changed ? "rolled" : "held"}
              aria-hidden="true"
            >
              {glyph.char}
            </span>
          ))}
        </span>
        {formatted.suffix ? <span data-rcl-figure-affix>{formatted.suffix}</span> : null}
      </span>
    </>
  );
});

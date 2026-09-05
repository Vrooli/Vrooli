/**
 * @libraryId react-component-library:RollingNumber
 * @displayName RollingNumber
 * @description A number drawn in an ink. The glyph box is identical in every ink so a metric going live never moves a pixel, and only the digits that changed roll.
 * @version 0.1.5
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:RollingNumber */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { useLocale, useStrings } from "@vrooli/react-component-library/useLocale/1";
import { useEffect, useRef, useState } from "react";
import { ProvenanceInkStyles, type Ink } from "@vrooli/react-component-library/ProvenanceInk/0";

interface FormattedFigure {
  /** Characters of the number itself, for digit-level animation. */
  text: string;
  prefix: string;
  suffix: string;
}

const compact = (value: number, locale: string): string =>
  new Intl.NumberFormat(locale, {
    notation: "compact",
    compactDisplay: "short",
    maximumFractionDigits: 1,
  }).format(value);

const trim = (s: string): string => s.replace(/\.0+$/, "").replace(/(\.\d*?)0+$/, "$1");
const group = (value: number, maxFraction: number, locale: string): string =>
  new Intl.NumberFormat(locale, { maximumFractionDigits: maxFraction }).format(value);

/**
 * Formats a reading's number for display. The format vocabulary is the
 * registry's: integer, compact, currency, currency.compact, percent,
 * percent.signed, minutes, duration.days.
 */
function formatFigure(
  value: number,
  format: string | undefined,
  unit: string | undefined,
  locale: string,
  strings: (key: string, fallback: string) => string,
): FormattedFigure {
  switch (format) {
    case "currency":
      return { prefix: "$", text: group(value, 0, locale), suffix: "" };
    case "currency.compact":
      return { prefix: "$", text: compact(value, locale), suffix: "" };
    case "percent":
      return {
        prefix: "",
        text: group(value * 100, value * 100 < 10 ? 1 : 0, locale),
        suffix: "%",
      };
    case "percent.signed":
      return {
        prefix: value >= 0 ? "+" : "−",
        text: group(Math.abs(value) * 100, 1, locale),
        suffix: "%",
      };
    case "minutes": {
      const abs = Math.abs(value);
      const divisor = abs >= 1_440 ? 1_440 : abs >= 60 ? 60 : 1;
      const unitKey = divisor === 1_440 ? "day" : divisor === 60 ? "hour" : "minute";
      const fallback = divisor === 1_440 ? "d" : divisor === 60 ? "h" : "min";
      return {
        prefix: "",
        text: group(value / divisor, value / divisor < 10 ? 1 : 0, locale),
        suffix: ` ${strings(`data-display.rolling-number.unit.${unitKey}`, fallback)}`,
      };
    }
    case "duration.days":
      return {
        prefix: "",
        text: group(value, 1, locale),
        suffix: ` ${strings("data-display.rolling-number.unit.day", "d")}`,
      };
    case "compact":
      return { prefix: "", text: compact(value, locale), suffix: "" };
    default:
      return {
        prefix: "",
        text: group(value, Number.isInteger(value) ? 0 : 1, locale),
        suffix: unit && unit !== "count" ? ` ${unit}` : "",
      };
  }
}

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
  [data-rcl-figure] { display: inline-flex; align-items: baseline; font-family: var(--font-sans, Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif); font-weight: 700; font-variant-numeric: tabular-nums lining-nums; letter-spacing: -0.03em; line-height: 0.92; white-space: nowrap; color: var(--color-foreground, #0f172a); }
  [data-rcl-figure="wall"] { font-size: var(--text-wall, clamp(5rem, 16vw, 20rem)); }
  [data-rcl-figure="display"] { font-size: var(--text-display, 700 var(--text-display-size) / var(--text-display-line) var(--font-sans)); letter-spacing: -0.02em; }
  [data-rcl-figure-affix] { font-size: 0.42em; font-weight: 600; letter-spacing: 0; margin: 0 0.08em; align-self: baseline; opacity: 0.85; }
  [data-rcl-figure-digits] { display: inline-flex; }
  [data-rcl-digit] { display: inline-block; }
  [data-rcl-digit="rolled"] { animation: rcl-figure-roll var(--dur-normal, var(--dur-moderate)) cubic-bezier(.2,.7,.2,1); }
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
  placeholder,
  className,
}: RollingNumberProps) {
  const strings = useStrings();
  const locale = useLocale();
  const empty = placeholder ?? strings("data-display.rolling-number.placeholder", "—");
  const formatted =
    value === null
      ? { prefix: "", text: empty, suffix: "" }
      : formatFigure(value, format, unit, locale, strings);
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

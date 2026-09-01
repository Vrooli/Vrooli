import { useEffect, useRef, useState } from "react";
import { formatFigure } from "../lib/format";
import type { Ink } from "../lib/provenance";

interface FigureProps {
  value: number | null;
  format?: string;
  unit?: string;
  ink: Ink;
  /** wall is the hero; display is a supporting reading. */
  scale: "wall" | "display";
  /** Shown instead of a number when there is none to show. */
  placeholder?: string;
}

interface Glyph { char: string; changed: boolean }

/**
 * A number drawn in an ink. The ink is a material (solid, dimmed, hollow,
 * dotted), never a colour, and the glyph box is identical in every ink so a
 * metric going live never moves a pixel. Only the digits that changed roll.
 */
export function Figure({ value, format, unit, ink, scale, placeholder = "—" }: FigureProps) {
  const formatted = value === null ? { prefix: "", text: placeholder, suffix: "" } : formatFigure(value, format, unit);
  const previous = useRef<string>(formatted.text);
  const [glyphs, setGlyphs] = useState<Glyph[]>(() => Array.from(formatted.text).map((char) => ({ char, changed: false })));
  const [rollKey, setRollKey] = useState(0);

  useEffect(() => {
    const before = previous.current;
    const after = formatted.text;
    if (before === after) return;
    previous.current = after;
    const padBefore = before.padStart(after.length, " ");
    setGlyphs(Array.from(after).map((char, i) => ({ char, changed: padBefore[padBefore.length - after.length + i] !== char })));
    setRollKey((key) => key + 1);
  }, [formatted.text]);

  return (
    <span className={`cc-figure cc-figure-${scale}`} data-figure data-ink={ink} data-value={value ?? undefined}>
      {formatted.prefix ? <span className="cc-figure-affix">{formatted.prefix}</span> : null}
      <span className="cc-figure-digits" aria-label={`${formatted.prefix}${formatted.text}${formatted.suffix}`}>
        {glyphs.map((glyph, i) => (
          <span key={`${rollKey}-${i}`} className={glyph.changed ? "cc-digit cc-digit-roll" : "cc-digit"} aria-hidden="true">
            {glyph.char}
          </span>
        ))}
      </span>
      {formatted.suffix ? <span className="cc-figure-affix">{formatted.suffix}</span> : null}
    </span>
  );
}

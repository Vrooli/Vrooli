import type { CSSProperties } from "react";
import { cn } from "../../lib/classnames";

/**
 * Tailwind classes for a TTS-related range input ("scrub" or "volume" slider).
 *
 * Centralizes the summarized-vs-original visual contract so the audio bar,
 * audio settings volume slider, and the message-jump drawer all stay in sync.
 * When TTS is rendering a summarized version of an event, sliders use amber
 * accents; otherwise they use the standard accent color.
 */
export function getAccentClasses(isSummarized: boolean): string {
  return isSummarized
    ? "[&::-webkit-slider-thumb]:bg-amber-400 accent-amber-400"
    : "accent-wc-accent";
}

/**
 * Full className for a TTS scrub bar (1px-tall track).
 * Combines layout, cursor, accent, and disabled treatments.
 */
export function getScrubClasses({
  isSummarized,
  enabled,
  extra,
}: {
  isSummarized: boolean;
  enabled: boolean;
  extra?: string;
}): string {
  return cn(
    "h-1 min-w-0",
    enabled ? "cursor-pointer" : "cursor-not-allowed opacity-50",
    getAccentClasses(isSummarized),
    extra,
  );
}

/**
 * The summarized-vs-original accent, for a library Slider rather than a bare
 * range input. `accent-color` only reaches a native control; a token-bound one
 * is retinted by pointing its primary token at the summarized hue.
 */
export function getSliderAccentStyle(isSummarized: boolean): CSSProperties | undefined {
  // Tailwind amber-400, matching the `accent-amber-400` used above.
  return isSummarized ? ({ "--color-primary": "#fbbf24" } as CSSProperties) : undefined;
}

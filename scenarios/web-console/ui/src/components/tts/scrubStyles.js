import { cn } from "../../lib/classnames";
/**
 * Tailwind classes for a TTS-related range input ("scrub" or "volume" slider).
 *
 * Centralizes the summarized-vs-original visual contract so the audio bar,
 * audio settings volume slider, and the message-jump drawer all stay in sync.
 * When TTS is rendering a summarized version of an event, sliders use amber
 * accents; otherwise they use the standard accent color.
 */
export function getAccentClasses(isSummarized) {
    return isSummarized
        ? "[&::-webkit-slider-thumb]:bg-amber-400 accent-amber-400"
        : "accent-wc-accent";
}
/**
 * Full className for a TTS scrub bar (1px-tall track).
 * Combines layout, cursor, accent, and disabled treatments.
 */
export function getScrubClasses({ isSummarized, enabled, extra, }) {
    return cn("h-1 min-w-0", enabled ? "cursor-pointer" : "cursor-not-allowed opacity-50", getAccentClasses(isSummarized), extra);
}

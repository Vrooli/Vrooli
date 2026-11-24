import type { Reading } from "../lib/api";
import { qualify, resolveReading } from "@vrooli/react-component-library/ProvenanceInk/0.1.3";
import { ReadingTrustLine } from "./ReadingTrustLine";

interface TileQualifierProps {
  qualifier?: { text: string; tone: string };
  reading: Reading;
  /** The room mixes origins, so a non-local reading marks its own. */
  showOrigin?: boolean;
}

/** Short mark for a non-local origin; the full name rides in the accessible label. */
const originMark =(env: string): string => (env === "production" ? "PROD" : env.slice(0, 6).toUpperCase());

/**
 * The one line under a supporting figure. A live reading's qualifier is its
 * hairline and the room's source strip, so its text stays in the accessibility
 * tree but is not drawn; every other state draws its reason on one line.
 */
export function TileQualifier({ qualifier: authoredQualifier, reading, showOrigin = false }: TileQualifierProps) {
  const qualifier = authoredQualifier ?? qualify(reading, resolveReading(reading));
  const live = qualifier.tone === "live" && !(reading.coverage === "NOW" && reading.trust !== "VALID");
  const origin = showOrigin && reading.origin_env && reading.origin_env !== "local" ? reading.origin_env : null;
  const text = <ReadingTrustLine reading={reading} qualifier={qualifier} hideLive />;
  if (live && !origin) return text;
  return (
    <span className="cc-tile-qualifier" title={qualifier.text}>
      {origin ? <span className="cc-origin-mark" data-origin role="img" aria-label={reading.origin_display || origin}>{originMark(origin)}</span> : null}
      {text}
    </span>
  );
}

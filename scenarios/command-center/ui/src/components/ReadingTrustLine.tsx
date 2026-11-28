import type { ReactNode } from "react";
import type { Reading } from "../lib/api";
import { formatAge, qualify, resolveReading, type Resolution } from "@vrooli/react-component-library/ProvenanceInk/0.1.3";
import "./ReadingTrustLine.css";

interface Props {
  reading: Reading;
  resolution?: Resolution;
  qualifier?: { text: string; tone: string };
  prefix?: ReactNode;
  hideLive?: boolean;
}

/** One visible trust line for every figure; current-state copy comes from the shared provenance policy. */
export function ReadingTrustLine({ reading, resolution = resolveReading(reading), qualifier, prefix, hideLive = false }: Props) {
  const degraded = reading.coverage === "NOW" && reading.trust !== "VALID";
  const canonicalQualifier = qualify(reading, resolution);
  const degradedQualifier = degraded && qualifier && qualifier.text !== canonicalQualifier.text
    ? { ...canonicalQualifier, text: `${qualifier.text} · ${canonicalQualifier.text}` }
    : canonicalQualifier;
  const resolvedQualifier = degraded ? degradedQualifier : qualifier ?? canonicalQualifier;
  const retainedAge = reading.trust === "UNAVAILABLE" && reading.value !== null && reading.observedAt
    ? ` · last good ${formatAge(reading.observedAt)}`
    : "";
  const fullText = `${resolvedQualifier.text}${retainedAge}`;
  const compactUnavailable = degraded && reading.trust === "UNAVAILABLE" && Boolean(reading.trustReason || retainedAge);
  const compactUntrusted = degraded && reading.trust === "UNTRUSTED" && !reading.observedAt;
  const text = compactUnavailable ? `source not answering${retainedAge}` : compactUntrusted ? "no source time" : fullText;
  const hidden = hideLive && !degraded && resolvedQualifier.tone === "live";
  const stateLabel = reading.trust === "VALID" ? "CURRENT" : reading.trust;
  const accessibleLabel = `${reading.label}: ${stateLabel}. ${fullText}`;

  return (
    <span className="cc-reading-trust-line" data-reading-trust={reading.trust} role="group" aria-label={accessibleLabel}>
      {degraded ? <strong className="cc-reading-trust-line__state" data-trust-state={reading.trust}>{reading.trust}</strong> : null}
      <span className={`cc-qualifier${hidden ? " cc-visually-hidden" : ""}`} data-qualifier data-tone={resolvedQualifier.tone} title={fullText}>
        {prefix}{text}
      </span>
    </span>
  );
}

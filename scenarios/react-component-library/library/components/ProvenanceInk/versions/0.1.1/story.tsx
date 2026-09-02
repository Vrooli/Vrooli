import {
  InkMark,
  InkSwatch,
  INK_LABELS,
  ProvenanceInkStyles,
  qualify,
  resolveReading,
  type ProvenanceReading,
} from "./ProvenanceInk";

const inks = ["solid", "dimmed", "hollow", "dotted"] as const;

const readings: Record<string, ProvenanceReading> = {
  live: {
    coverage: "NOW",
    trust: "VALID",
    value: 1284,
    observedAt: new Date(Date.now() - 9_000).toISOString(),
    sample: null,
    source: { team: "director-swarm", binding: "scenario:swarm-manager" },
  },
  cached: {
    coverage: "NOW",
    trust: "CACHED",
    trustReason: "connection refused",
    value: 1284,
    observedAt: new Date(Date.now() - 240_000).toISOString(),
    sample: null,
    source: { binding: "scenario:swarm-manager" },
  },
  inReach: {
    coverage: "IN-REACH",
    trust: "UNAVAILABLE",
    value: null,
    observedAt: null,
    whatIsNeeded: "a revenue surface on the monetization instrument",
    sample: { value: 12400, series: [8100, 12400], basis: "hand-authored" },
    source: { team: "monetization" },
  },
  missing: {
    coverage: "MISSING",
    trust: "UNAVAILABLE",
    value: null,
    observedAt: null,
    owner: "marketing-crew",
    gapOpenDays: 14,
    sample: { value: 5400, series: [3200, 5400], basis: "hand-authored" },
    source: { team: "marketing-crew" },
  },
};

const ground = {
  background: "var(--color-background, #05070e)",
  color: "var(--color-foreground, #e8ecf3)",
  padding: "var(--space-lg, 32px)",
};

/** The four inks as a legend: the same glyph, four materials. */
export function Legend() {
  return (
    <div style={{ ...ground, display: "flex", flexWrap: "wrap", gap: "var(--space-md, 24px)" }}>
      <ProvenanceInkStyles />
      {inks.map((ink) => (
        <span
          key={ink}
          style={{
            display: "inline-flex",
            alignItems: "center",
            font: "var(--text-caption, 600 0.75rem/1.3 system-ui)",
          }}
        >
          <InkSwatch ink={ink} />
          {INK_LABELS[ink]}
        </span>
      ))}
    </div>
  );
}

/** Every chip the resolver can produce, with the qualifier it pairs with. */
export function Chips() {
  return (
    <dl
      style={{
        ...ground,
        display: "grid",
        gridTemplateColumns: "auto 1fr",
        gap: "var(--space-xs, 12px) var(--space-md, 24px)",
        font: "var(--text-caption, 600 0.75rem/1.3 system-ui)",
      }}
    >
      {Object.entries(readings).map(([key, reading]) => {
        const resolution = resolveReading(reading);
        const qualifier = qualify(reading, resolution);
        return [
          <dt key={`${key}-dt`}>
            {resolution.ink === "none" ? null : <InkMark ink={resolution.ink} />}
          </dt>,
          <dd key={`${key}-dd`} style={{ margin: 0 }} data-tone={qualifier.tone}>
            {qualifier.text}
          </dd>,
        ];
      })}
    </dl>
  );
}

/** A silent sensor: the frame with the stated reason, never a zero. */
export function Unavailable() {
  const reading: ProvenanceReading = {
    coverage: "NOW",
    trust: "UNAVAILABLE",
    trustReason: "deadline exceeded",
    value: null,
    observedAt: null,
    sample: null,
    source: { binding: "scenario:vrooli-core" },
  };
  const resolution = resolveReading(reading);
  return (
    <p style={{ ...ground, font: "var(--text-body, 400 1rem/1.5 system-ui)" }}>
      <InkMark ink="unavailable" /> {qualify(reading, resolution).text}
    </p>
  );
}

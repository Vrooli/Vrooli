import type { CSSProperties } from "react";
import { Text, type TextStyle } from "./Text";

const samples: Record<TextStyle, string> = {
  display: "Make the important thing unmistakable.",
  title: "A considered hierarchy makes every decision easier.",
  heading: "Typography that carries the product’s point of view.",
  body: "Good interface writing gives people enough context to move with confidence, then gets out of the way.",
  label: "Workspace identity",
  caption: "Updated a moment ago · visible to your team",
  code: "const result = await workspace.publish();",
  overline: "TYPE SCALE / 08 STYLES",
};

const showcaseStyle: CSSProperties = {
  display: "grid",
  gap: "var(--space-md)",
  inlineSize: "100%",
  maxInlineSize: "100%",
  boxSizing: "border-box",
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background:
    "linear-gradient(145deg, var(--color-surface-raised), color-mix(in srgb, var(--color-primary) 5%, var(--color-surface-raised)))",
  boxShadow: "var(--elev-raised)",
};

export function TypographyShowcase({
  args,
}: {
  args?: { textStyle?: TextStyle };
}) {
  const style = args?.textStyle ?? "body";
  return (
    <section
      aria-label="Typography specimen"
      data-rcl-typography-showcase
      style={showcaseStyle}
    >
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <Text textStyle="overline" tone="accent">
          Token typography
        </Text>
        <Text as="h1" textStyle="title">
          Typography as a system
        </Text>
        <Text tone="muted" balance>
          Eight bundled styles keep hierarchy, rhythm, and tone coherent across
          every surface.
        </Text>
      </div>
      <div
        style={{
          display: "grid",
          gap: "var(--space-sm)",
          paddingBlock: "var(--space-md)",
          borderBlock: "var(--border-hairline) solid var(--color-border)",
        }}
      >
        <Text textStyle={style} balance>
          {samples[style]}
        </Text>
        <Text textStyle="caption" tone="muted">
          Active style: {style}
        </Text>
      </div>
      <div
        style={{ display: "flex", flexWrap: "wrap", gap: "var(--space-xs)" }}
      >
        <Text textStyle="label">Consistent</Text>
        <Text textStyle="caption" tone="muted">
          Responsive
        </Text>
        <Text textStyle="code" numeric>
          01 / 08
        </Text>
      </div>
    </section>
  );
}

const toneSamples = [
  ["default", "Foreground text for the main reading path."],
  ["muted", "Muted text for supporting context."],
  ["accent", "Accent text for a deliberate emphasis."],
  ["danger", "Danger text for a recoverable warning."],
] as const;

export function TextAnatomy() {
  return (
    <section
      aria-label="Text anatomy"
      data-rcl-text-anatomy
      style={showcaseStyle}
    >
      <Text as="h1" textStyle="title">
        A clear typographic hierarchy
      </Text>
      <Text tone="muted">
        One component carries the reading rhythm across a surface.
      </Text>
      <Text textStyle="caption" tone="accent">
        Default anatomy
      </Text>
    </section>
  );
}

export function TextScaleMatrix() {
  return (
    <section
      aria-label="Text scale matrix"
      data-rcl-text-scale
      style={showcaseStyle}
    >
      <Text as="h2" textStyle="heading">
        Eight bundled styles
      </Text>
      <div style={{ display: "grid", gap: "var(--space-sm)" }}>
        {(Object.keys(samples) as TextStyle[]).map((style) => (
          <Text key={style} textStyle={style}>
            {samples[style]}
          </Text>
        ))}
      </div>
    </section>
  );
}

export function TextToneMatrix() {
  return (
    <section
      aria-label="Text tone matrix"
      data-rcl-text-tone
      style={showcaseStyle}
    >
      <Text as="h2" textStyle="heading">
        Four semantic tones
      </Text>
      <div style={{ display: "grid", gap: "var(--space-sm)" }}>
        {toneSamples.map(([tone, copy]) => (
          <Text key={tone} tone={tone}>
            {copy}
          </Text>
        ))}
      </div>
    </section>
  );
}

export function TextBoundaries() {
  return (
    <section
      aria-label="Text boundary states"
      data-rcl-text-boundaries
      style={showcaseStyle}
    >
      <Text as="h2" textStyle="heading">
        Boundary conditions
      </Text>
      <Text aria-label="Empty text" textStyle="body" />
      <Text truncate>
        Truncated content remains legible when a single line must fit inside a
        constrained surface.
      </Text>
      <div style={{ maxInlineSize: "12rem" }}>
        <Text>Overflow content demonstrates the wrapping boundary.</Text>
      </div>
      <Text numeric tone="muted">
        1,024.00
      </Text>
    </section>
  );
}

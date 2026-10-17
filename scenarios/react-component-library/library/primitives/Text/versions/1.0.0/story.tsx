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

export function TypographyShowcase({ args }: { args?: { style?: TextStyle } }) {
  const style = args?.style ?? "body";
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
        <Text as="p" data-testid="rcl-text-sample" textStyle={style} balance>
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

export function TextAnatomy() {
  return <TypographyShowcase args={{ style: "body" }} />;
}

export function TextScaleMatrix() {
  return <TypographyShowcase args={{ style: "display" }} />;
}

export function TextToneMatrix() {
  return <TypographyShowcase args={{ style: "label" }} />;
}

export function TextBoundaries() {
  return <TypographyShowcase args={{ style: "caption" }} />;
}

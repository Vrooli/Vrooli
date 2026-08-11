import type { CSSProperties } from "react";
import { Heading } from "./Heading";
import { Text, type TextStyle } from "../../../Text/versions/1.0.0/Text";

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

export function HeadingShowcase({
  args,
}: {
  args?: { style?: TextStyle; level?: 1 | 2 | 3 | 4 | 5 | 6 };
}) {
  const style = args?.style ?? "heading";
  const level = args?.level ?? 2;
  return (
    <section
      aria-label="Heading specimen"
      data-rcl-heading-showcase
      style={showcaseStyle}
    >
      <Text textStyle="overline" tone="accent">
        Semantic hierarchy
      </Text>
      <Heading level={level} style={style} balance>
        Meaning and visual scale stay independent.
      </Heading>
      <Text tone="muted" balance>
        Change the document level without losing the typography that makes the
        page feel intentional.
      </Text>
      <Text textStyle="caption" tone="muted">
        h{level} · {style} style
      </Text>
    </section>
  );
}

import type { ComponentType, ReactNode } from "react";

type ShowcaseProps = {
  subject: ComponentType<Record<string, unknown>>;
  args: Record<string, unknown>;
  config?: { title?: string; detail?: string };
  children?: ReactNode;
};

/**
 * Generic Preview-only context for self-contained assets. The subject is
 * injected by the host; this module has no component-specific imports.
 */
export function Showcase({
  subject: Subject,
  args,
  config,
  children,
}: ShowcaseProps) {
  return (
    <section
      data-preview-harness="showcase"
      data-preview-sheet="shared-harness"
      style={{
        display: "grid",
        gap: "var(--space-lg, 24px)",
        width: "min(100%, 42rem)",
        boxSizing: "border-box",
        padding: "var(--space-xl, 40px)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "var(--color-surface-raised)",
        boxShadow: "var(--elev-raised)",
      }}
    >
      <header style={{ display: "grid", gap: "var(--space-2xs, 8px)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".08em",
            textTransform: "uppercase",
          }}
        >
          Component specimen
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          {config?.title || "Default presentation"}
        </strong>
        {config?.detail ? (
          <span style={{ color: "var(--color-muted-foreground)" }}>
            {config.detail}
          </span>
        ) : null}
      </header>
      <div data-preview-harness-subject="true">
        <Subject {...args} />
      </div>
      {children}
    </section>
  );
}

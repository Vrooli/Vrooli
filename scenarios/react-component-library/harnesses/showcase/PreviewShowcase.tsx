import type { ComponentType, ReactNode } from "react";

export type PreviewShowcaseConfig = {
  title?: string;
  detail?: string;
  status?: string;
  density?: "comfortable" | "compact";
};

export type PreviewShowcaseProps = {
  subject: ComponentType<Record<string, unknown>>;
  args?: Record<string, unknown>;
  config?: PreviewShowcaseConfig;
  family?: string;
  label?: string;
  description?: string;
  children?: ReactNode;
};

/** Stable preview-only harness surface. It is intentionally outside the
 * versioned component catalog and is not a production-library export. */
export function PreviewShowcase({
  subject: Subject,
  args = {},
  config,
  family = "showcase",
  label = "Component specimen",
  description,
  children,
}: PreviewShowcaseProps) {
  const compact = config?.density === "compact";
  return (
    <section
      aria-label={label}
      data-preview-harness={family}
      data-preview-harness-density={config?.density ?? "comfortable"}
      style={{
        display: "grid",
        gap: compact ? "var(--space-md, 16px)" : "var(--space-lg, 24px)",
        width: "min(100%, 48rem)",
        maxWidth: "100%",
        marginInline: "auto",
        boxSizing: "border-box",
        padding: compact ? "var(--space-lg, 24px)" : "var(--space-xl, 40px)",
        border: "var(--border-hairline, 1px) solid var(--color-border)",
        borderRadius: "var(--radius-panel, 12px)",
        background: "var(--color-surface-raised)",
        color: "var(--color-foreground)",
        boxShadow: compact
          ? "none"
          : "var(--elev-raised, 0 12px 32px rgb(15 23 42 / 12%))",
      }}
    >
      <header
        data-preview-harness-header
        style={{ display: "grid", gap: "var(--space-2xs, 8px)" }}
      >
        <span
          data-preview-harness-family
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline, 700 0.75rem/1.2 sans-serif)",
            letterSpacing: ".08em",
            textTransform: "uppercase",
          }}
        >
          {label}
        </span>
        <strong
          data-preview-harness-title
          style={{ font: "var(--text-title, 700 1.5rem/1.2 sans-serif)" }}
        >
          {config?.title || label}
        </strong>
        {(config?.detail || description) && (
          <span
            data-preview-harness-description
            style={{ color: "var(--color-muted-foreground)" }}
          >
            {config?.detail || description}
          </span>
        )}
      </header>
      <div
        data-preview-harness-subject
        style={{
          display: "grid",
          justifyItems: compact ? "start" : "center",
          minWidth: 0,
        }}
      >
        <Subject {...args} />
      </div>
      {config?.status && (
        <output data-preview-harness-status role="status">
          {config.status}
        </output>
      )}
      {children && <footer data-preview-harness-actions>{children}</footer>}
    </section>
  );
}

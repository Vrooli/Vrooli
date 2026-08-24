import type { ComponentType, ReactNode } from "react";

export type PreviewHarnessProps = {
  subject: ComponentType<Record<string, unknown>>;
  args?: Record<string, unknown>;
  children?: ReactNode;
  label?: string;
  description?: string;
};

/** Generic fixture-backed data context; fixture values are supplied by the Preview runtime. */
export function DataState({
  subject: Subject,
  args = {},
  children,
  label = "Data state",
  description = "A deterministic collection or selection state.",
}: PreviewHarnessProps) {
  return (
    <section
      aria-label={label}
      data-preview-harness="data-state"
      data-preview-sheet="shared-harness"
    >
      <header>
        <p data-preview-harness-label>{label}</p>
        <p data-preview-harness-description>{description}</p>
      </header>
      <div data-preview-harness-subject>
        <Subject {...args} />
      </div>
      {children}
    </section>
  );
}

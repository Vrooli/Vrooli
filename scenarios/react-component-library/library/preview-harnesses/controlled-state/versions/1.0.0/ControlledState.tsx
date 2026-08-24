import type { ComponentType, ReactNode } from "react";

export type PreviewHarnessProps = {
  subject: ComponentType<Record<string, unknown>>;
  args?: Record<string, unknown>;
  children?: ReactNode;
  label?: string;
  description?: string;
};

/** Generic controlled-state story shell. The subject remains the only component under test. */
export function ControlledState({
  subject: Subject,
  args = {},
  children,
  label = "Controlled state",
  description = "A focused state with its inputs visible for inspection.",
}: PreviewHarnessProps) {
  return (
    <section
      aria-label={label}
      data-preview-harness="controlled-state"
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

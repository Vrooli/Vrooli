import type { ComponentType, ReactNode } from "react";

export type PreviewHarnessProps = {
  subject: ComponentType<Record<string, unknown>>;
  args?: Record<string, unknown>;
  children?: ReactNode;
  label?: string;
  description?: string;
};

/** Generic interaction shell for stories that demonstrate a before/after transition. */
export function StateTransition({
  subject: Subject,
  args = {},
  children,
  label = "State transition",
  description = "Interact with the subject to observe its meaningful state change.",
}: PreviewHarnessProps) {
  return (
    <section
      aria-label={label}
      data-preview-harness="state-transition"
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

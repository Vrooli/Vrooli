import type { ComponentType, ReactNode } from "react";

export type PreviewHarnessProps = {
  subject: ComponentType<Record<string, unknown>>;
  args?: Record<string, unknown>;
  children?: ReactNode;
  label?: string;
  description?: string;
};

/** Generic overlay context; focus, escape, and dismissal remain subject behavior. */
export function OverlayInteraction({
  subject: Subject,
  args = {},
  children,
  label = "Overlay interaction",
  description = "Open, inspect, and dismiss the subject in its overlay context.",
}: PreviewHarnessProps) {
  return (
    <section
      aria-label={label}
      data-preview-harness="overlay-interaction"
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

import type { ComponentType, ReactNode } from "react";

export type PreviewHarnessProps = {
  subject: ComponentType<Record<string, unknown>>;
  args?: Record<string, unknown>;
  children?: ReactNode;
  label?: string;
  description?: string;
};

/** Generic responsive context; the host viewport is the source of breakpoint truth. */
export function ResponsiveMode({
  subject: Subject,
  args = {},
  children,
  label = "Responsive mode",
  description = "A responsive subject rendered at the current Preview viewport.",
}: PreviewHarnessProps) {
  return (
    <section
      aria-label={label}
      data-preview-harness="responsive-mode"
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

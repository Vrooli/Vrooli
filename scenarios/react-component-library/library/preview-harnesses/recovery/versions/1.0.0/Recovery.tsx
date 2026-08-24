import type { ComponentType, ReactNode } from "react";

export type PreviewHarnessProps = {
  subject: ComponentType<Record<string, unknown>>;
  args?: Record<string, unknown>;
  children?: ReactNode;
  label?: string;
  description?: string;
};

/** Generic recovery context; retry and validation behavior belongs to the subject story. */
export function Recovery({
  subject: Subject,
  args = {},
  children,
  label = "Recovery state",
  description = "A recoverable failure with the subject's recovery action visible.",
}: PreviewHarnessProps) {
  return (
    <section
      aria-label={label}
      data-preview-harness="recovery"
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

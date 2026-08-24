import type { ComponentType, ReactNode } from "react";

export type PreviewHarnessProps = {
  subject: ComponentType<Record<string, unknown>>;
  args?: Record<string, unknown>;
  children?: ReactNode;
  label?: string;
  description?: string;
};

/** Generic loading/error/success context shell; async behavior stays in the injected story subject. */
export function AsyncState({
  subject: Subject,
  args = {},
  children,
  label = "Async state",
  description = "A representative asynchronous state with its status context visible.",
}: PreviewHarnessProps) {
  return (
    <section
      aria-label={label}
      data-preview-harness="async-state"
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

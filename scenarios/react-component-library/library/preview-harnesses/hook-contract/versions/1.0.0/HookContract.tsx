import type { ComponentType, ReactNode } from "react";

export type PreviewHarnessProps = {
  subject: ComponentType<Record<string, unknown>>;
  args?: Record<string, unknown>;
  children?: ReactNode;
  label?: string;
  description?: string;
};

/** Hook/adapter context; the injected subject is an observable hook adapter, not a production UI. */
export function HookContract({
  subject: Subject,
  args = {},
  children,
  label = "Hook contract",
  description = "Observable hook output and actions for the declared runtime contract.",
}: PreviewHarnessProps) {
  return (
    <section
      aria-label={label}
      data-preview-harness="hook-contract"
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

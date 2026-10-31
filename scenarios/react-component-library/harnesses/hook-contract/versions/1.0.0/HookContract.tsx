import type { ComponentType, ReactNode } from "react";

import { PreviewShowcase } from "../../../showcase/PreviewShowcase";

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
    <PreviewShowcase
      subject={Subject}
      args={args}
      family="hook-contract"
      label={label}
      description={description}
    >
      {children}
    </PreviewShowcase>
  );
}

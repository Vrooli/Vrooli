import type { ComponentType, ReactNode } from "react";

import { PreviewShowcase } from "../../../showcase/PreviewShowcase";

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
    <PreviewShowcase
      subject={Subject}
      args={args}
      family="state-transition"
      label={label}
      description={description}
    >
      {children}
    </PreviewShowcase>
  );
}

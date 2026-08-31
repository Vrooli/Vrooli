import type { ComponentType, ReactNode } from "react";

import { PreviewShowcase } from "../../../showcase/PreviewShowcase";

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
    <PreviewShowcase
      subject={Subject}
      args={args}
      family="overlay-interaction"
      label={label}
      description={description}
    >
      {children}
    </PreviewShowcase>
  );
}

import type { ComponentType, ReactNode } from "react";

import { PreviewShowcase } from "../../../showcase/versions/1.0.0/PreviewShowcase";

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
    <PreviewShowcase
      subject={Subject}
      args={args}
      family="recovery"
      label={label}
      description={description}
    >
      {children}
    </PreviewShowcase>
  );
}

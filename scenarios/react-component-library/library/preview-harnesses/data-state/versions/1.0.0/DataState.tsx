import type { ComponentType, ReactNode } from "react";

import { PreviewShowcase } from "../../../showcase/versions/1.0.0/PreviewShowcase";

export type PreviewHarnessProps = {
  subject: ComponentType<Record<string, unknown>>;
  args?: Record<string, unknown>;
  children?: ReactNode;
  label?: string;
  description?: string;
};

/** Generic fixture-backed data context; fixture values are supplied by the Preview runtime. */
export function DataState({
  subject: Subject,
  args = {},
  children,
  label = "Data state",
  description = "A deterministic collection or selection state.",
}: PreviewHarnessProps) {
  return (
    <PreviewShowcase
      subject={Subject}
      args={args}
      family="data-state"
      label={label}
      description={description}
    >
      {children}
    </PreviewShowcase>
  );
}

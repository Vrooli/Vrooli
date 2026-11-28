import type { ComponentType, ReactNode } from "react";

import { PreviewShowcase } from "../../../showcase/PreviewShowcase";

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
    <PreviewShowcase
      subject={Subject}
      args={args}
      family="async-state"
      label={label}
      description={description}
    >
      {children}
    </PreviewShowcase>
  );
}

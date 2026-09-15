import type { ComponentType, ReactNode } from "react";

import { PreviewShowcase } from "./PreviewShowcase";

type ShowcaseProps = {
  subject: ComponentType<Record<string, unknown>>;
  args: Record<string, unknown>;
  config?: { title?: string; detail?: string };
  children?: ReactNode;
};

/**
 * Generic Preview-only context for self-contained assets. The subject is
 * injected by the host; this module has no component-specific imports.
 */
export function Showcase({
  subject: Subject,
  args,
  config,
  children,
}: ShowcaseProps) {
  return (
    <PreviewShowcase
      subject={Subject}
      args={args}
      config={config}
      family="showcase"
      label="Component specimen"
    >
      {children}
    </PreviewShowcase>
  );
}

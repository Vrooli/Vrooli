import type { ComponentType, ReactNode } from "react";
import { AppShell } from "@vrooli/react-component-library/AppShell/2";

type NavigationContextProps = {
  subject: ComponentType<Record<string, unknown>>;
  args?: Record<string, unknown>;
  config?: { title?: string; detail?: string; region?: "navigation" | "content" };
  children?: ReactNode;
};

/** Preview-only composition. The injected subject remains the specimen;
 * AppShell owns the workspace geometry, landmarks and responsive behavior. */
export function NavigationContext({ subject: Subject, args = {}, config, children }: NavigationContextProps) {
  const specimen = children === undefined ? <Subject {...args} /> : <Subject {...args}>{children}</Subject>;
  const inContent = config?.region === "content";
  return (
    <div data-preview-harness="navigation-context">
    <AppShell
      brand="Preview workspace"
      items={[]}
      density="sidebar"
      mobileNav="tabs"
      mainMode="scroll"
      testId="preview-navigation-context"
      utility={inContent ? undefined : specimen}
    >
      <h1>{config?.title ?? "Workspace content"}</h1>
      <p>{config?.detail ?? "The navigation specimen is rendered beside the content it organizes."}</p>
      {inContent ? specimen : null}
    </AppShell>
    </div>
  );
}

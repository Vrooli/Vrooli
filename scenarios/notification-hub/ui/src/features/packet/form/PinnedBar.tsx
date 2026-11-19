import type { ReactNode } from "react";

/**
 * A bar pinned to the bottom of the decision, holding its main action. It must
 * be a direct child of the tall content it pins within (a sticky bar cannot
 * leave its parent), and `.packet-dock` (styles.css) keeps it flush against the
 * viewport edge while clearing the home indicator.
 */
export function PinnedBar({ children }: { children: ReactNode }) {
  return <footer className="packet-dock z-10 mt-auto border-t border-app-border bg-app-surface/95 pt-2 backdrop-blur lg:mt-0 lg:rounded-b-panel">{children}</footer>;
}

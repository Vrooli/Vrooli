/**
 * @libraryId react-component-library:TopBar
 * @displayName TopBar
 * @version 1.0.7
 * @tags ["navigation","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:TopBar */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";
const panel = {
  border: "1px solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface)",
  color: "var(--color-foreground)",
  padding: "var(--space-md)",
  boxShadow: "var(--elev-raised)",
};
const muted = { color: "var(--color-muted-foreground)" };
export const TopBar = withClassName(function TopBar({ children }: { children?: ReactNode }) {
  const strings = useStrings();
  return (
    <header
      data-testid="navigation.top-bar"
      data-top-bar
      style={{
        display: "flex",
        alignItems: "center",
        gap: 16,
        minHeight: 64,
        ...panel,
        paddingInline: 24,
      }}
    >
      {children ?? (
        <>
          <strong style={{ fontSize: 18 }}>
            {strings("navigation.top-bar.application", "Application")}
          </strong>
          <span style={{ marginInlineStart: "auto", ...muted }}>
            {strings("navigation.top-bar.workspace", "Workspace")}
          </span>
        </>
      )}
    </header>
  );
});

/**
 * @libraryId react-component-library:TopBar
 * @displayName TopBar
 * @description A stable application header region that adapts its navigation affordances by viewport.
 * @version 1.0.5
 * @tags ["navigation","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:TopBar */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { ReactNode } from "react";
const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export const TopBar = withClassName(function TopBar({
  children,
}: {
  children?: ReactNode;
}) {
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

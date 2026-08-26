/**
 * @libraryId react-component-library:TopBar
 * @displayName TopBar
 * @description A stable application header region that adapts its navigation affordances by viewport.
 * @version 1.0.4
 * @tags ["navigation","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:TopBar */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

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
export const TopBar = withClassName(function TopBar({ children }: { children?: ReactNode }) {
  return (
    <header
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
            {translate("navigation.top-bar.text.1", "Application")}
          </strong>
          <span style={{ marginInlineStart: "auto", ...muted }}>
            {translate("navigation.top-bar.text.2", "Workspace")}
          </span>
        </>
      )}
    </header>
  );
});

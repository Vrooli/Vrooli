/** @vrooliComponentSource react-component-library:AppNavigation */
import type { ReactNode } from "react";
const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};
export function AppNavigation({
  mode = "desktop",
  children,
}: {
  mode?: "mobile" | "tablet" | "desktop" | "wide";
  children?: ReactNode;
}) {
  const mobile = mode === "mobile";
  const content = children ?? (
    <a
      href="/"
      aria-current="page"
      style={{
        minHeight: 44,
        display: "flex",
        alignItems: "center",
        borderRadius: 8,
        background: "var(--color-primary, #2563eb)",
        color: "var(--color-primary-foreground, #fff)",
        paddingInline: 12,
        textDecoration: "none",
        fontWeight: 700,
      }}
    >
      Home
    </a>
  );
  return (
    <div
      data-responsive-transformation="sidebar-to-drawer modal-to-bottom-sheet header-to-bottom-navigation"
      data-viewport-mode={mode}
      data-presentation={
        mobile ? "bottom-navigation" : mode === "tablet" ? "drawer" : "sidebar"
      }
      style={{ ...panel, minHeight: mobile ? 64 : 260, padding: 16 }}
    >
      <nav
        aria-label="Application navigation"
        data-testid={
          mobile
            ? "app-navigation-bottom-navigation"
            : mode === "tablet"
              ? "app-navigation-drawer"
              : "app-navigation-sidebar"
        }
        style={{
          display: mobile ? "flex" : "grid",
          gap: 8,
          justifyContent: mobile ? "space-around" : undefined,
        }}
      >
        {content}
      </nav>
    </div>
  );
}

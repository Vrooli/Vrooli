/** @vrooliComponentSource materialized.resizablesidebar */
import type { ReactNode } from "react";
const panel = { border: "1px solid var(--color-border, #cbd5e1)", borderRadius: "var(--radius-panel, .75rem)", background: "var(--color-surface, #fff)", color: "var(--color-foreground, #0f172a)", padding: "var(--space-md, 24px)", boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))" };
export function ResizableSidebar({ children }: { children?: ReactNode }) { return <aside data-resizable-sidebar style={{ ...panel, minInlineSize: 260, maxInlineSize: 480 }}>{children}</aside>; }

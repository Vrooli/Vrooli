/** @vrooliComponentSource materialized.topbar */
import type { ReactNode } from "react";
const panel = { border: "1px solid var(--color-border, #cbd5e1)", borderRadius: "var(--radius-panel, .75rem)", background: "var(--color-surface, #fff)", color: "var(--color-foreground, #0f172a)", padding: "var(--space-md, 24px)", boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))" };
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export function TopBar({ children }: { children?: ReactNode }) { return <header data-top-bar style={{ display: "flex", alignItems: "center", gap: 16, minHeight: 64, ...panel, paddingInline: 24 }}>{children ?? <><strong style={{ fontSize: 18 }}>Application</strong><span style={{ marginInlineStart: "auto", ...muted }}>Workspace</span></>}</header>; }

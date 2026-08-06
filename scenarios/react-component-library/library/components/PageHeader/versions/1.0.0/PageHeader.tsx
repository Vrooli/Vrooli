/** @vrooliComponentSource materialized.pageheader */
import type { ReactNode } from "react";
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export function PageHeader({ title = "Page", description, actions }: { title?: string; description?: string; actions?: ReactNode }) { return <header style={{ display: "flex", alignItems: "end", justifyContent: "space-between", gap: 24, flexWrap: "wrap", paddingBottom: 16 }}><div><h1 style={{ margin: 0, fontSize: 24, letterSpacing: "-.025em" }}>{title}</h1>{description && <p style={{ ...muted, margin: "8px 0 0" }}>{description}</p>}</div>{actions && <div style={{ display: "flex", gap: 8 }}>{actions}</div>}</header>; }

/** @vrooliComponentSource materialized.inspectorpanel */
import type { ReactNode } from "react";
const panel = { border: "1px solid var(--color-border, #cbd5e1)", borderRadius: "var(--radius-panel, .75rem)", background: "var(--color-surface, #fff)", color: "var(--color-foreground, #0f172a)", padding: "var(--space-md, 24px)", boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))" };
export function InspectorPanel({ open = false, onClose, children }: { open?: boolean; onClose?: () => void; children?: ReactNode }) { if (!open) return null; return <section role="dialog" aria-label="Inspector" style={{ ...panel, display: "grid", gap: 16 }}>{children ?? "Inspector details"}<button type="button" onClick={onClose}>Close</button></section>; }

/** @vrooliComponentSource materialized.collapsibleregion */
import type { ReactNode } from "react";
export function CollapsibleRegion({ open = true, children }: { open?: boolean; children?: ReactNode }) { return <div data-collapsible-region data-open={open} aria-hidden={!open || undefined} style={{ overflow: "hidden", opacity: open ? 1 : 0, transition: "opacity var(--dur-moderate, 280ms) ease" }}>{open ? children : null}</div>; }

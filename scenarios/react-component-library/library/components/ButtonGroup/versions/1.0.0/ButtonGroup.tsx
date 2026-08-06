/** @vrooliComponentSource materialized.buttongroup */
import type { ReactNode } from "react";
import type { HTMLAttributes } from "react";
export function ButtonGroup({ children, label = "Actions", ...props }: HTMLAttributes<HTMLDivElement> & { label?: string; children?: ReactNode }) { return <div role="group" aria-label={label} style={{ display: "inline-flex", flexWrap: "wrap", gap: 8 }} {...props}>{children}</div>; }

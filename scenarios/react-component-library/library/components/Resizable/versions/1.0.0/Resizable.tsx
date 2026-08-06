/** @vrooliComponentSource materialized.resizable */
import type { ReactNode } from "react";
export function Resizable({ children, orientation = "horizontal" }: { children?: ReactNode; orientation?: "horizontal" | "vertical" }) { return <div data-orientation={orientation} style={{ display: "flex", flexDirection: orientation === "horizontal" ? "row" : "column", minInlineSize: 240, minBlockSize: 160, gap: 12 }}>{children}</div>; }

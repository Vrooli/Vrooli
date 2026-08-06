/** @vrooliComponentSource materialized.splitview */
import type { ReactNode } from "react";
export function SplitView({ primary, secondary }: { primary?: ReactNode; secondary?: ReactNode }) { return <div data-split-view style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(min(100%, 280px), 1fr))", gap: 24 }}><section>{primary}</section><section>{secondary}</section></div>; }

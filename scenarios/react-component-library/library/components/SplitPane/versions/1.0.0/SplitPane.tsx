/** @vrooliComponentSource react-component-library:SplitPane */
import type { ReactNode } from "react";
export function SplitPane({
  primary,
  secondary,
}: {
  primary?: ReactNode;
  secondary?: ReactNode;
}) {
  return (
    <div
      data-split-pane
      style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fit, minmax(min(100%, 280px), 1fr))",
        gap: 16,
      }}
    >
      <section>{primary}</section>
      <section>{secondary}</section>
    </div>
  );
}

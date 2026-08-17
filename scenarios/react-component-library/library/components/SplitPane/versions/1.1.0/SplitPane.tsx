/**
 * @vrooliComponentSource react-component-library:SplitPane
 * @libraryId react-component-library:SplitPane
 * @version 1.1.0
 * @status released
 * @deps {"react":"^18"}
 */
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

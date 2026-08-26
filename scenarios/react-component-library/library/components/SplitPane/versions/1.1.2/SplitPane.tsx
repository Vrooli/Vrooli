/**
 * @libraryId react-component-library:SplitPane
 * @displayName SplitPane
 * @description A two-region workspace layout with stable pane geometry and a shared responsive composition contract.
 * @version 1.1.2
 * @tags ["layout","interaction","token-bound"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { ReactNode } from "react";

export const SplitPane = withClassName(function SplitPane({
  primary,
  secondary,
}: {
  primary?: ReactNode;
  secondary?: ReactNode;
}) {
  return (
    <div data-testid="manipulation.split-pane"
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
});

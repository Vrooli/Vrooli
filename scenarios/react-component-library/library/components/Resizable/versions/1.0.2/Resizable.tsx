/**
 * @libraryId react-component-library:Resizable
 * @displayName Resizable
 * @description A bounded resizable region that preserves usable minimum geometry while the user adjusts a layout.
 * @version 1.0.2
 * @tags ["layout","interaction","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Resizable */
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

import type { ReactNode } from "react";
export const Resizable = withClassName(function Resizable({
  children,
  orientation = "horizontal",
}: {
  children?: ReactNode;
  orientation?: "horizontal" | "vertical";
}) {
  return (
    <div
      data-orientation={orientation}
      style={{
        display: "flex",
        flexDirection: orientation === "horizontal" ? "row" : "column",
        minInlineSize: 240,
        minBlockSize: 160,
        gap: 12,
      }}
    >
      {children}
    </div>
  );
});

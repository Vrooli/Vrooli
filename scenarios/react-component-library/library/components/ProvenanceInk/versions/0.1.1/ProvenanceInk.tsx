/**
 * @libraryId react-component-library:ProvenanceInk
 * @displayName ProvenanceInk
 * @description Reusable command-display primitive
 * @version 0.1.1
 * @tags ["ambient-display","command-center"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import * as React from "react";
export interface ProvenanceInkProps {
  coverage?: string;
  basis?: string;
  children?: React.ReactNode;
}
export default function ProvenanceInk({ coverage = "NOW", basis, children }: ProvenanceInkProps) {
  return (
    <span
      data-coverage={coverage}
      title={basis}
      className={coverage === "NOW" ? "rcl-ink-solid" : "rcl-ink-hollow"}
    >
      {children ?? coverage}
    </span>
  );
}

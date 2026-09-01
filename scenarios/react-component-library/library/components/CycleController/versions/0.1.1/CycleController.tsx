/**
 * @libraryId react-component-library:CycleController
 * @displayName CycleController
 * @description Reusable command-display primitive
 * @version 0.1.1
 * @tags ["ambient-display","command-center"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import * as React from "react";
export interface CycleControllerProps {
  index: number;
  count: number;
  onChange: (index: number) => void;
}
export default function CycleController({ index, count, onChange }: CycleControllerProps) {
  return (
    <nav aria-label="Cycle controller">
      <button onClick={() => onChange((index - 1 + count) % count)} aria-label="Previous">
        ‹
      </button>
      <span>
        {index + 1} / {count}
      </span>
      <button onClick={() => onChange((index + 1) % count)} aria-label="Next">
        ›
      </button>
    </nav>
  );
}

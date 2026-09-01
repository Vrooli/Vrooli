/**
 * @libraryId react-component-library:AmbientCanvas
 * @displayName AmbientCanvas
 * @description Reusable command-display primitive
 * @version 0.1.1
 * @tags ["ambient-display","command-center"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import * as React from "react";
export default function AmbientCanvas({ children }: { children?: React.ReactNode }) {
  return (
    <div className="rcl-ambient-canvas" role="img" aria-label="Ambient display">
      {children}
    </div>
  );
}

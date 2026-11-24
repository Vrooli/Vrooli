/**
 * @libraryId react-component-library:UnknownValue
 * @displayName Unknown Value
 * @description Inline fallback for a value that is absent or cannot be derived
 * @version 0.1.0
 * @tags ["data-display","uncertainty"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ReactNode } from "react";
import "./UnknownValue.css";

export interface UnknownValueProps {
  label: ReactNode;
  reason?: string;
}

/** Inline fallback when a value is absent or cannot be derived. */
export function UnknownValue({ label, reason }: UnknownValueProps) {
  return (
    <span data-rcl-unknown-value="" title={reason}>
      {label}
      {reason ? <span data-rcl-unknown-value-reason="">{` · ${reason}`}</span> : null}
    </span>
  );
}

export default UnknownValue;

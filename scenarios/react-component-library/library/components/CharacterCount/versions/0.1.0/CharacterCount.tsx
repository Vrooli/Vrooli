/**
 * @libraryId react-component-library:CharacterCount
 * @displayName Character Count
 * @description Live character-count feedback and optional limit message for editable text controls
 * @version 0.1.0
 * @tags ["forms","feedback"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ReactNode } from "react";
import "./CharacterCount.css";

export interface CharacterCountProps {
  id?: string;
  count: ReactNode;
  overLimit?: boolean;
  message?: ReactNode;
}

/** Live character-count feedback associated with an editable text control. */
export function CharacterCount({ id, count, overLimit = false, message }: CharacterCountProps) {
  return (
    <output
      id={id}
      data-rcl-character-count=""
      data-exceeded={overLimit || undefined}
      aria-live="polite"
    >
      {count}
      {message ? <span data-rcl-character-count-message="">{message}</span> : null}
    </output>
  );
}

export default CharacterCount;

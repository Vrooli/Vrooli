import type { ReactNode } from "react";
export interface InlineCodeProps {
  children: ReactNode;
  onClick?: (text: string) => void;
}
/**
 * @libraryId react-component-library:inline-code
 * @displayName Inline Code
 * @description Inline code token renderer with copy affordance.
 * @version 0.1.1
 * @tags ["markdown","code"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */

export function InlineCode({ children, onClick }: InlineCodeProps) {
  const text =
    typeof children === "string" || typeof children === "number"
      ? String(children)
      : "";
  return (
    <button
      type="button"
      onClick={() => onClick?.(text)}
      className="rounded bg-[var(--color-surface-muted)] [padding-inline:var(--space-3xs)] [padding-block:var(--space-3xs)] font-mono text-[var(--color-accent)] disabled:cursor-text"
      disabled={!onClick}
    >
      {children}
    </button>
  );
}

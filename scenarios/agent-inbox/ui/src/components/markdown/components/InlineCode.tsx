import type { ReactNode } from "react";

interface InlineCodeProps {
  children: ReactNode;
}

/**
 * Styled inline code component matching the dark theme.
 */
export function InlineCode({ children }: InlineCodeProps) {
  return (
    <code className="bg-slate-700 text-slate-200 px-1.5 py-0.5 rounded text-sm font-mono">
      {children}
    </code>
  );
}

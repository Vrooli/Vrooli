import type { ReactNode } from "react";

interface InlineCodeProps {
  children: ReactNode;
}

export function InlineCode({ children }: InlineCodeProps) {
  return (
    <code className="bg-muted/60 text-foreground px-1.5 py-0.5 rounded text-sm font-mono">
      {children}
    </code>
  );
}

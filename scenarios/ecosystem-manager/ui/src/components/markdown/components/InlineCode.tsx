import type { ReactNode } from 'react';

export function InlineCode({ children }: { children?: ReactNode }) {
  return (
    <code className="rounded bg-black/30 px-1.5 py-0.5 font-mono text-[0.9em] text-slate-100">
      {children}
    </code>
  );
}

import type { ReactNode } from "react";

/**
 * The save and discard actions of a draft. It stays at the bottom edge of the
 * scrolling page so the thumb never has to look for it, and it exists only
 * while there is something to save.
 */
export function SaveBar({ children }: { children: ReactNode }) {
  return (
    <div className="sticky bottom-0 z-10 -mx-4 mt-2 flex flex-col gap-2 border-t border-app-border bg-app-background px-4 py-3 sm:flex-row-reverse sm:justify-start">
      {children}
    </div>
  );
}

import type { HTMLAttributes } from "react";
import { cn } from "../../lib/utils";

/**
 * Padded content for one CollectionList row.
 *
 * The list's CardShell owns the row chrome (border, selection rail, cursor,
 * context menu) and the open interaction, so row content must not draw a
 * second card or wrap itself in its own button.
 */
export function CollectionRow({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("min-w-0 cursor-pointer p-2.5 transition-colors hover:bg-slate-800/40", className)}
      {...props}
    />
  );
}

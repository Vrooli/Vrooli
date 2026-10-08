// DOC: docs/internal/COHERENCE-NOTES.md
import { cn } from "../lib/utils";
import type { ReactNode } from "react";

interface PanelProps {
  children: ReactNode;
  className?: string;
  /** Use "overlay" for darker backgrounds (e.g. event detail), default is "elevated". */
  variant?: "elevated" | "overlay";
}

export function Panel({ children, className, variant = "elevated" }: PanelProps) {
  return (
    <div
      className={cn(
        "rounded-xl border border-[var(--border-default)]",
        variant === "elevated" ? "bg-[var(--surface-elevated)]" : "bg-[var(--surface-overlay)]",
        "p-6",
        className,
      )}
    >
      {children}
    </div>
  );
}

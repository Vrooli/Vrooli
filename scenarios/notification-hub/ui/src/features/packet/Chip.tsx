import type { ReactNode } from "react";

const TONES = {
  neutral: "border-app-border bg-app-surface-muted text-app-foreground",
  attention: "border-app-warning/60 bg-app-warning/15 text-app-foreground",
  info: "border-app-primary/60 bg-app-primary/10 text-app-foreground",
  danger: "border-app-danger/60 bg-app-danger/10 text-app-foreground",
} as const;

/** A short status or category label; always text, never color alone. */
export function Chip({ tone = "neutral", icon, children, className = "" }: { tone?: keyof typeof TONES; icon?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <span className={`inline-flex max-w-full items-center gap-1 rounded-pill border px-2 py-0.5 text-xs font-medium leading-5 ${TONES[tone]} ${className}`}>
      {icon}
      <span className="min-w-0 break-words">{children}</span>
    </span>
  );
}

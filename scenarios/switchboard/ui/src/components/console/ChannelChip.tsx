import type { HTMLAttributes } from "react";

interface ChannelChipProps extends HTMLAttributes<HTMLSpanElement> {
  id: string;
  name?: string;
  accent?: string;
  /** Compact renders the accent dot with the name in small caps. */
  size?: "sm" | "md";
  testId?: string;
}

/**
 * Identifies which channel a thing arrived on without reading the message.
 * The accent colour is read from the channel descriptor, never chosen here,
 * so a new channel lands with its own colour by arriving.
 */
export function ChannelChip({ id, name, accent, size = "sm", testId, className, ...rest }: ChannelChipProps) {
  const label = name?.trim() || id;
  return (
    <span
      role="img"
      aria-label={label}
      data-testid={testId}
      data-channel={id}
      className={[
        "inline-flex max-w-full items-center gap-1.5 rounded-pill border border-app-border bg-app-surface-muted font-medium text-app-foreground",
        size === "sm" ? "px-2 py-0.5 text-xs" : "px-2.5 py-1 text-sm",
        className ?? "",
      ].join(" ")}
      {...rest}
    >
      <span aria-hidden="true" className="h-2 w-2 shrink-0 rounded-full" style={{ background: accent ?? "var(--color-accent)" }} />
      <span className="truncate">{label}</span>
    </span>
  );
}

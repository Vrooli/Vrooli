import { useId } from "react";

/**
 * The product mark: a crescent cut from the moonlight, with the one cyan
 * signal as a small satellite. Purely decorative; the brand text names it.
 */
export function MoonMark({ className = "h-[22px] w-[22px]" }: { className?: string }) {
  const maskId = `hub-crescent-${useId().replace(/[^a-zA-Z0-9_-]/g, "")}`;
  return (
    <svg className={`shrink-0 text-app-foreground ${className}`} viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <defs>
        <mask id={maskId}>
          <rect width="24" height="24" fill="#000" />
          <circle cx="11.3" cy="12" r="9.4" fill="#fff" />
          <circle cx="17" cy="11" r="7.5" fill="#000" />
        </mask>
      </defs>
      <rect width="24" height="24" fill="currentColor" mask={`url(#${maskId})`} />
      <circle cx="18.3" cy="17.6" r="1.75" fill="var(--color-signal)" />
    </svg>
  );
}

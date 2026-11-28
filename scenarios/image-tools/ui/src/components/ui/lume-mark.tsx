import { useId } from "react";

import { cn } from "../../lib/utils";

export interface LumeMarkProps {
  /** Square edge length in px. */
  size?: number;
  className?: string;
  /**
   * Accessible label. When omitted the mark is decorative (`aria-hidden`) —
   * the common case, since it sits beside the "Lume" wordmark.
   */
  title?: string;
}

/**
 * The Lume product mark — a soft luminous gold dot (an aperture catching light)
 * on the deep-slate shell. Inline SVG so it scales crisply and the gradient
 * tracks no theme (the mark is a fixed identity object, like a logo). Gradient
 * ids are `useId`-scoped so multiple marks on one page stay valid.
 */
export function LumeMark({ size = 24, className, title }: LumeMarkProps) {
  const gid = useId();
  const glow = `lume-glow-${gid}`;
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 512 512"
      className={cn("shrink-0", className)}
      role={title ? "img" : undefined}
      aria-hidden={title ? undefined : true}
    >
      {title ? <title>{title}</title> : null}
      <defs>
        <radialGradient id={glow} cx="50%" cy="44%" r="58%">
          <stop offset="0%" stopColor="#fbeec2" />
          <stop offset="34%" stopColor="#e9cf86" />
          <stop offset="68%" stopColor="#cba74f" />
          <stop offset="100%" stopColor="#9c7415" />
        </radialGradient>
      </defs>
      <rect width="512" height="512" rx="112" fill="#0b1020" />
      <circle cx="256" cy="248" r="132" fill={`url(#${glow})`} />
      <circle cx="256" cy="248" r="132" fill="none" stroke="#fbeec2" strokeOpacity="0.4" strokeWidth="5" />
      <circle cx="222" cy="214" r="30" fill="#fff6dd" fillOpacity="0.55" />
    </svg>
  );
}

import type { Appearance } from "../../api/console";
import { hueFor, initials } from "../../lib/identity";

export type AgentMarkSize = "xs" | "sm" | "md" | "lg" | "xl";

const SIZE_PX: Record<AgentMarkSize, number> = { xs: 20, sm: 28, md: 36, lg: 48, xl: 72 };

interface AgentMarkProps {
  name: string;
  appearance?: Appearance | null;
  size?: AgentMarkSize;
  /** Adds a live/paused ring. */
  live?: boolean;
  className?: string;
  testId?: string;
}

/**
 * The agent's visual identity, rendered from the `appearance` colour triple
 * that a prompt-manager descriptor already carries (body, head, accent). The
 * same agent therefore looks identical everywhere in the ecosystem. Agents
 * without a triple fall back to a deterministic hue from their id so two
 * unstyled agents never look the same.
 */
export function AgentMark({ name, appearance, size = "md", live, className, testId }: AgentMarkProps) {
  const px = SIZE_PX[size];
  const hue = hueFor(name);
  const body = appearance?.body || `hsl(${hue} 55% 42%)`;
  const head = appearance?.head || `hsl(${hue} 70% 62%)`;
  const accent = appearance?.accent || `hsl(${hue} 80% 85%)`;
  const radius = Math.max(4, Math.round(px * 0.22));
  return (
    <span
      role="img"
      aria-label={name}
      data-testid={testId}
      data-live={live === undefined ? undefined : String(live)}
      className={["relative inline-flex shrink-0 select-none", className ?? ""].join(" ")}
      style={{ width: px, height: px }}
    >
      <svg viewBox="0 0 40 40" width={px} height={px} aria-hidden="true" focusable="false">
        <rect x="0" y="0" width="40" height="40" rx={(radius / px) * 40} fill={body} />
        <circle cx="20" cy="15.5" r="7.5" fill={head} />
        <path d="M8 34c1.6-7 6.2-10.5 12-10.5S30.4 27 32 34" fill={accent} opacity="0.95" />
        <rect x="0" y="0" width="40" height="40" rx={(radius / px) * 40} fill="none" stroke="rgba(0,0,0,0.12)" />
      </svg>
      <span className="sr-only">{initials(name)}</span>
      {live !== undefined ? (
        <span
          aria-hidden="true"
          className={[
            "absolute -bottom-0.5 -right-0.5 rounded-full border-2 border-app-surface",
            live ? "bg-app-success" : "bg-app-muted-foreground",
          ].join(" ")}
          style={{ width: Math.max(8, px * 0.3), height: Math.max(8, px * 0.3) }}
        />
      ) : null}
    </span>
  );
}

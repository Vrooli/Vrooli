import { type ReactNode } from "react";

import { useTranslation } from "../i18n";
import { TONE_CLASSES, type StatusDescriptor } from "../lib/planStatus";

/**
 * StatusBadge renders a domain enum (plan/phase status, staleness, verdict) as a
 * token-toned chip. It pairs the tone color with the translated label *and* a
 * leading dot so meaning never relies on color alone (WCAG 1.4.1). Pass the
 * descriptor from `lib/planStatus` so every surface renders the same enum
 * identically.
 */
export interface StatusBadgeProps {
  descriptor: StatusDescriptor;
  /** Optional leading icon (e.g. a lucide glyph) for extra non-color signal. */
  icon?: ReactNode;
  "data-testid"?: string;
  className?: string;
}

export function StatusBadge({ descriptor, icon, className, ...rest }: StatusBadgeProps) {
  const { t } = useTranslation();
  const tone = TONE_CLASSES[descriptor.tone];
  return (
    <span
      data-testid={rest["data-testid"]}
      className={[
        "inline-flex items-center gap-1.5 rounded-pill px-2.5 py-0.5 text-xs font-medium",
        tone.badge,
        className ?? "",
      ].join(" ")}
    >
      {icon ? (
        <span aria-hidden="true" className="inline-flex">
          {icon}
        </span>
      ) : (
        <span aria-hidden="true" className={["h-1.5 w-1.5 rounded-full", tone.dot].join(" ")} />
      )}
      {t(descriptor.labelKey)}
    </span>
  );
}

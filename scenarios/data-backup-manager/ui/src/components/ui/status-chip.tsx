import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { cn } from "../../lib/utils";
import { TONE_TEXT_CLASS, type Tone } from "../../lib/status";
import type { StringKey } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * Compact status chip. A tone sets the text + dot color (the single source of
 * status color is `lib/status`), and the label always renders text so status
 * never relies on color alone (DESIGN.md / WCAG). Pass an i18n key for the
 * label — never inline copy.
 */
export function StatusChip({
  tone,
  labelKey,
  className,
  "data-testid": testId,
}: {
  tone: Tone;
  labelKey: StringKey;
  className?: string;
  "data-testid"?: string;
}) {
  const { t } = useTranslation();
  const label = t(labelKey);

  // StatusBadge/1.2.5 covers every shared status tone except DBM's `primary`
  // role. Keep only that still-unpublished token variant local.
  if (tone !== "primary") {
    return (
      <StatusBadge tone={tone} className={className} data-testid={testId}>
        {label}
      </StatusBadge>
    );
  }

  return (
    <span
      data-testid={testId}
      className={cn(
        "inline-flex items-center gap-1.5 whitespace-nowrap rounded-pill border border-app-border bg-app-surface-muted px-2 py-0.5 text-xs font-medium",
        TONE_TEXT_CLASS[tone],
        className,
      )}
    >
      <span aria-hidden="true" className="h-1.5 w-1.5 rounded-full bg-current" />
      {label}
    </span>
  );
}

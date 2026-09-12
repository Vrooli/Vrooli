import { ChevronDown } from "lucide-react";
import type { ReactNode } from "react";

import { clsx } from "clsx";

import { usePersistedDisclosure } from "../../hooks/usePersistedDisclosure";

export interface DetailSectionProps {
  title: string;
  /** Entity/section icon. The first "Overview" section carries the entity icon. */
  icon?: ReactNode;
  /** Optional trailing header slot (e.g. a count badge or link). */
  headerAction?: ReactNode;
  /** Drop the header's bottom border. Used by the Overview section. */
  hideDivider?: boolean;
  /**
   * Opt-in persisted collapse. When set, the section becomes a disclosure whose
   * open/closed state is remembered under this key across reloads.
   */
  storageKey?: string;
  defaultOpen?: boolean;
  testId?: string;
  children: ReactNode;
}

/**
 * A titled panel for detail-page bodies. Passing `storageKey` turns it into a
 * persisted disclosure (toggle testid = `${testId}-toggle`); otherwise it is a
 * static section. Built to the fleet detail body convention.
 */
export function DetailSection({
  title,
  icon,
  headerAction,
  hideDivider = false,
  storageKey,
  defaultOpen = true,
  testId,
  children,
}: DetailSectionProps) {
  const collapsible = storageKey !== undefined;
  const [open, toggle] = usePersistedDisclosure(storageKey ?? "", defaultOpen);
  const expanded = collapsible ? open : true;

  const headingContent = (
    <span className="flex min-w-0 items-center gap-2">
      {icon && <span className="shrink-0 text-app-muted-foreground">{icon}</span>}
      <span className="min-w-0 truncate text-base font-semibold text-app-foreground">{title}</span>
    </span>
  );

  return (
    <section
      data-testid={testId}
      className="rounded-panel border border-app-border bg-app-surface text-app-foreground"
    >
      <div
        className={clsx(
          "flex min-w-0 items-center justify-between gap-3 px-4 py-3",
          !hideDivider && "border-b border-app-border",
        )}
      >
        {collapsible ? (
          <button
            type="button"
            data-testid={testId ? `${testId}-toggle` : undefined}
            aria-expanded={expanded}
            onClick={toggle}
            className="flex min-h-11 min-w-0 flex-1 items-center gap-2 rounded-control text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
          >
            <ChevronDown
              aria-hidden
              className={clsx("h-4 w-4 shrink-0 transition-transform", !expanded && "-rotate-90")}
            />
            {headingContent}
          </button>
        ) : (
          <h3 className="flex min-h-11 min-w-0 flex-1 items-center">{headingContent}</h3>
        )}
        {headerAction && <div className="shrink-0">{headerAction}</div>}
      </div>
      {expanded && <div className="min-w-0 px-4 py-4">{children}</div>}
    </section>
  );
}

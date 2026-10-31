/**
 * DetailSection
 *
 * Lightweight section wrapper for entity detail pages. Replaces the
 * Card-inside-tab pattern with flat sections separated by thin dividers,
 * eliminating compounding padding on mobile.
 *
 * Passing `storageKey` makes the section collapsible: the heading becomes a
 * disclosure toggle whose open state persists to localStorage (same
 * `usePersistedDisclosure` mechanism as `CollapsibleSection`).
 */

import { type ReactNode } from "react";
import { ChevronDown, ChevronRight, type LucideIcon } from "lucide-react";
import { cn } from "../../lib/utils";
import { usePersistedDisclosure } from "../ui/collapsible-section";

export interface DetailSectionProps {
  /** Section heading text. */
  title: string;
  /** Optional icon rendered before the title. */
  icon?: LucideIcon;
  /** Optional action slot (e.g., edit button) rendered at the right of the heading. */
  action?: ReactNode;
  /** Section content. */
  children: ReactNode;
  /** Hide the top divider (for the first section in a group). */
  hideDivider?: boolean;
  /** When set, the section is collapsible and persists open state under this key. */
  storageKey?: string;
  /** Initial open state for collapsible sections (default true). */
  defaultOpen?: boolean;
  className?: string;
  "data-testid"?: string;
}

function SectionShell({
  hideDivider,
  className,
  testId,
  header,
  children,
}: {
  hideDivider?: boolean;
  className?: string;
  testId?: string;
  header: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className={cn(!hideDivider && "mt-4 border-t border-slate-800 pt-4", hideDivider && "pt-1", className)} data-testid={testId}>
      <div className="flex items-center gap-2 pb-3">{header}</div>
      {children}
    </section>
  );
}

function StaticHeading({ icon: Icon, title }: { icon?: LucideIcon; title: string }) {
  return (
    <>
      {Icon && <Icon className="h-4 w-4 text-slate-400" />}
      <h2 className="text-base font-semibold text-slate-100">{title}</h2>
    </>
  );
}

function CollapsibleDetailSection({
  title,
  icon: Icon,
  action,
  children,
  hideDivider,
  storageKey,
  defaultOpen = true,
  className,
  "data-testid": testId,
}: DetailSectionProps & { storageKey: string }) {
  const [open, toggle] = usePersistedDisclosure(storageKey, defaultOpen);

  return (
    <SectionShell
      hideDivider={hideDivider}
      className={className}
      testId={testId}
      header={
        <>
          <button
            type="button"
            onClick={toggle}
            aria-expanded={open}
            className="flex min-w-0 items-center gap-2 rounded text-left transition-colors hover:text-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50"
            data-testid={testId ? `${testId}-toggle` : undefined}
          >
            {open ? (
              <ChevronDown className="h-4 w-4 shrink-0 text-slate-500" aria-hidden />
            ) : (
              <ChevronRight className="h-4 w-4 shrink-0 text-slate-500" aria-hidden />
            )}
            <StaticHeading icon={Icon} title={title} />
          </button>
          {action && <div className="ml-auto">{action}</div>}
        </>
      }
    >
      {open && <div className="pb-2">{children}</div>}
    </SectionShell>
  );
}

export function DetailSection(props: DetailSectionProps) {
  if (props.storageKey) {
    return <CollapsibleDetailSection {...props} storageKey={props.storageKey} />;
  }

  const { title, icon: Icon, action, children, hideDivider, className, "data-testid": testId } = props;
  return (
    <SectionShell
      hideDivider={hideDivider}
      className={className}
      testId={testId}
      header={
        <>
          <StaticHeading icon={Icon} title={title} />
          {action && <div className="ml-auto">{action}</div>}
        </>
      }
    >
      <div className="pb-2">{children}</div>
    </SectionShell>
  );
}

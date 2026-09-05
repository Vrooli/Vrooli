/**
 * CollapsibleSection — the standard disclosure pattern for inline
 * expand/collapse areas (sidebar Filters & Sort, Plan Now-column lanes,
 * Recently Viewed, …).
 *
 * Open/closed state persists to localStorage per `storageKey`, so each
 * section remembers its own state independently across reloads.
 */

import { useCallback, useState, type ReactNode } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import { cn } from "../../lib/utils";

const STORAGE_PREFIX = "swarm-manager.section.";

function readPersisted(storageKey: string, defaultOpen: boolean): boolean {
  if (typeof window === "undefined") return defaultOpen;
  try {
    const raw = window.localStorage.getItem(STORAGE_PREFIX + storageKey);
    return raw === null ? defaultOpen : raw === "1";
  } catch {
    return defaultOpen;
  }
}

/**
 * Disclosure state persisted per storageKey. Returns [open, toggle].
 */
export function usePersistedDisclosure(storageKey: string, defaultOpen: boolean): [boolean, () => void] {
  const [open, setOpen] = useState(() => readPersisted(storageKey, defaultOpen));

  const toggle = useCallback(() => {
    setOpen((prev) => {
      const next = !prev;
      try {
        window.localStorage.setItem(STORAGE_PREFIX + storageKey, next ? "1" : "0");
      } catch {
        // Storage unavailable — state still toggles for this session.
      }
      return next;
    });
  }, [storageKey]);

  return [open, toggle];
}

interface CollapsibleSectionProps {
  /** Unique per UI area — becomes the localStorage key suffix. */
  storageKey: string;
  defaultOpen?: boolean;
  /** Toggle-button content, rendered after the chevron. */
  label: ReactNode;
  /** Optional content on the right side of the header row (not part of the toggle). */
  headerRight?: ReactNode;
  children: ReactNode;
  className?: string;
  headerClassName?: string;
  toggleClassName?: string;
  contentClassName?: string;
  toggleTestId?: string;
  contentTestId?: string;
}

export function CollapsibleSection({
  storageKey,
  defaultOpen = false,
  label,
  headerRight,
  children,
  className,
  headerClassName,
  toggleClassName,
  contentClassName,
  toggleTestId,
  contentTestId,
}: CollapsibleSectionProps) {
  const [open, toggle] = usePersistedDisclosure(storageKey, defaultOpen);

  return (
    <div className={className}>
      <div className={cn("flex items-center justify-between", headerClassName)}>
        <button
          type="button"
          onClick={toggle}
          aria-expanded={open}
          className={cn(
            "flex items-center gap-1 text-[11px] font-medium text-slate-400 transition-colors hover:text-slate-200",
            toggleClassName,
          )}
          data-testid={toggleTestId}
        >
          {open ? <ChevronDown className="h-3 w-3" aria-hidden /> : <ChevronRight className="h-3 w-3" aria-hidden />}
          {label}
        </button>
        {headerRight}
      </div>
      {open && (
        <div className={contentClassName} data-testid={contentTestId}>
          {children}
        </div>
      )}
    </div>
  );
}

import { Link } from "react-router-dom";

import { cn } from "../lib/utils";

export interface SectionTab {
  id: string;
  to: string;
  label: string;
  current: boolean;
  testId?: string;
}

/**
 * A segmented switcher between sibling pages. The segments are links, so each
 * has its own address, back button and long-press preview.
 */
export function SectionTabs({ label, tabs }: { label: string; tabs: readonly SectionTab[] }) {
  return (
    <nav aria-label={label}>
      <ul className="inline-flex gap-1 rounded-control border border-app-border bg-app-surface-muted p-1">
        {tabs.map((tab) => (
          <li key={tab.id}>
            <Link
              to={tab.to}
              data-testid={tab.testId}
              aria-current={tab.current ? "page" : undefined}
              className={cn(
                "flex min-h-11 min-w-24 items-center justify-center rounded-control px-4 text-sm font-medium transition-colors duration-quick focus-visible:outline focus-visible:outline-2",
                tab.current ? "bg-app-surface text-app-foreground shadow-raised" : "text-app-muted-foreground hover:text-app-foreground",
              )}
            >
              {tab.label}
            </Link>
          </li>
        ))}
      </ul>
    </nav>
  );
}

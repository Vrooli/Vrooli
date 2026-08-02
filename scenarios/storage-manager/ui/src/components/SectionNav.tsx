import type { ReactNode } from "react";

export interface SectionNavItem {
  id: string;
  label: string;
  icon: ReactNode;
}

interface SectionNavProps {
  activeId: string;
  items: SectionNavItem[];
  onSelect: (id: string) => void;
}

export function SectionNav({ activeId, items, onSelect }: SectionNavProps) {
  return (
    <nav aria-label="Storage console sections" className="sticky top-0 z-10 -mx-4 hidden overflow-x-auto border-b border-app-border bg-app-background/95 px-4 py-2 backdrop-blur md:-mx-6 md:block md:px-6">
      <div className="flex min-w-max gap-1" role="tablist">
        {items.map((item) => (
          <a
            key={item.id}
            role="tab"
            aria-selected={activeId === item.id}
            href={`#storage-${item.id}`}
            className={`inline-flex min-h-11 items-center gap-2 rounded-control px-3 py-2 text-sm font-medium ${activeId === item.id ? "bg-app-primary text-app-primary-foreground" : "text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"}`}
            onClick={() => onSelect(item.id)}
          >
            <span aria-hidden="true">{item.icon}</span>
            {item.label}
          </a>
        ))}
      </div>
    </nav>
  );
}

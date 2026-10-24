import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

interface SectionLabelProps {
  children: ReactNode;
  icon?: LucideIcon;
}

export function SectionLabel({ children, icon: Icon }: SectionLabelProps) {
  return (
    <h3 className="mb-2 inline-flex items-center gap-1.5 text-xs font-medium uppercase tracking-wider text-slate-500">
      {Icon ? <Icon aria-hidden="true" className="h-3.5 w-3.5" /> : null}
      {children}
    </h3>
  );
}

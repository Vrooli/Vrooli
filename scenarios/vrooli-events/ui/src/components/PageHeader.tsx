// DOC: docs/internal/COHERENCE-NOTES.md
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

interface PageHeaderProps {
  icon: LucideIcon;
  title: string;
  /** Optional right-aligned actions (buttons, etc.) */
  actions?: ReactNode;
}

export function PageHeader({ icon: Icon, title, actions }: PageHeaderProps) {
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-3">
        <Icon className="h-5 w-5 text-[var(--text-accent)]" />
        <h2 className="text-lg font-semibold">{title}</h2>
      </div>
      {actions}
    </div>
  );
}

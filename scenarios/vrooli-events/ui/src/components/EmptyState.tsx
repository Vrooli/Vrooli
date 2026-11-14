// DOC: docs/internal/COHERENCE-NOTES.md
// DOC: docs/internal/EXPERIENCE-AUDIT.md
import type { LucideIcon } from "lucide-react";
import { Link } from "react-router-dom";

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: {
    label: string;
    to: string;
  };
}

export function EmptyState({ icon: Icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center" data-testid="empty-state">
      <Icon className="mb-4 h-12 w-12 text-[var(--text-muted)]" />
      <h3 className="mb-2 text-lg font-medium text-[var(--text-primary)]">{title}</h3>
      <p className="max-w-md text-sm text-[var(--text-muted)]">{description}</p>
      {action && (
        <Link
          to={action.to}
          className="mt-4 text-sm text-[var(--color-accent)] hover:underline"
        >
          {action.label}
        </Link>
      )}
    </div>
  );
}

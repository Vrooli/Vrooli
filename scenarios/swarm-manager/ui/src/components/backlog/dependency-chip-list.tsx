/**
 * Dependency Chip List
 *
 * Renders a labeled list of clickable, status-colored chips for parent
 * (upstream) or children (downstream) dependencies. Each chip navigates
 * to the dependency's backlog details page.
 */

import { memo } from "react";
import { Link } from "react-router-dom";
import type { LucideIcon } from "lucide-react";
import type { ResolvedDependency } from "../../lib/backlog-queue-utils";
import { BACKLOG_STATUS_CHIP_COLORS, formatBacklogStatus } from "../../types/constants";

interface DependencyChipListProps {
  label: string;
  items: ResolvedDependency[];
  icon: LucideIcon;
}

export const DependencyChipList = memo(function DependencyChipList({
  label,
  items,
  icon: Icon,
}: DependencyChipListProps) {
  if (items.length === 0) return null;

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
        <Icon className="h-3.5 w-3.5" />
        {label}
      </div>
      <div className="flex flex-wrap gap-1.5">
        {items.map((dep) => (
          <Link
            key={`${dep.kind}/${dep.name}`}
            to={`/backlog/${dep.kind}/${dep.name}`}
            title={formatBacklogStatus(dep.status)}
            className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium transition-colors hover:brightness-125 ${BACKLOG_STATUS_CHIP_COLORS[dep.status]}`}
          >
            {dep.title}
          </Link>
        ))}
      </div>
    </div>
  );
});

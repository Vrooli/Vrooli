/**
 * EmptyState — Cheerful empty state for the Command Post.
 *
 * Shown when no actionable items exist. Offers quick navigation
 * to Topology or Operations lenses.
 */

import { CheckCircle } from "lucide-react";
import { Button } from "../ui/button";

interface EmptyStateProps {
  onSwitchLens: (lens: string) => void;
}

export function EmptyState({ onSwitchLens }: EmptyStateProps) {
  return (
    <div
      className="flex flex-col items-center justify-center gap-4 py-20 text-center"
      data-testid="command-post-empty-state"
    >
      <CheckCircle className="h-12 w-12 text-emerald-400" />
      <div>
        <h3 className="text-lg font-medium text-slate-200">All clear!</h3>
        <p className="mt-1 text-sm text-slate-400">Nothing needs you right now.</p>
      </div>
      <div className="flex gap-2">
        <Button variant="outline" size="sm" onClick={() => onSwitchLens("topology")}>
          Topology
        </Button>
        <Button variant="outline" size="sm" onClick={() => onSwitchLens("operations")}>
          Operations
        </Button>
      </div>
    </div>
  );
}

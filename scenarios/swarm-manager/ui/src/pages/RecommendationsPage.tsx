/**
 * Recommendations Page - Displays system-generated improvement suggestions
 *
 * PURPOSE:
 * Shows actionable recommendations from the recommendation engine. Users can
 * approve, reject, or configure how recommendations are generated.
 *
 * CURRENT STATUS: Placeholder UI
 * This page currently shows only an empty state directing users to Settings.
 * The actual recommendation listing and management will be implemented when
 * the backend recommendation engine (OT-P1-001, OT-P1-002) is complete.
 *
 * FUTURE BEHAVIOR (when implemented):
 * - List pending recommendations sorted by priority
 * - Show recommendation type (test, feature, refactor, docs)
 * - Allow approve/reject actions per recommendation
 * - Filter by scenario, type, or priority
 * - Display recommendation rationale and source data
 *
 * DEPENDENCIES (not yet connected):
 * - recommendation engine backend (P1 feature)
 * - Settings page recommendation mode configuration
 *
 * Experience Architecture (Phase 29):
 * - Empty state provides direct navigation to Settings
 * - Reduces cognitive load by surfacing the action path immediately
 *
 * Related PRD targets: OT-P1-001, OT-P1-002
 */

import { Link } from "react-router-dom";
import { Filter, Zap, Settings, ArrowRight } from "lucide-react";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { selectors } from "../consts/selectors";

export function RecommendationsPage() {
  return (
    <div className="space-y-6" data-testid={selectors.recommendations.page}>
      {/* Header actions */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <h2 className="text-xl font-semibold">Recommendations</h2>
        <Button
          variant="outline"
          size="sm"
          disabled
          title="Filter functionality coming soon"
          data-testid={selectors.recommendations.filter}
        >
          <Filter className="mr-2 h-4 w-4" />
          Filter
        </Button>
      </div>

      {/* Empty state - with direct navigation to reduce friction (Phase 29) */}
      <Card padding="lg" centered data-testid={selectors.recommendations.empty}>
        <Zap className="mx-auto h-12 w-12 text-slate-600" />
        <h3 className="mt-4 text-lg font-medium text-slate-300">No recommendations</h3>
        <p className="mt-2 text-sm text-slate-400">
          Enable the recommendation engine in Settings to start receiving suggestions
        </p>
        <Link to="/settings" data-testid={selectors.recommendations.settingsLink}>
          <Button variant="outline" className="mt-4 group">
            <Settings className="mr-2 h-4 w-4" />
            Configure Settings
            <ArrowRight className="ml-2 h-4 w-4 opacity-0 transition group-hover:opacity-100" />
          </Button>
        </Link>
      </Card>
    </div>
  );
}

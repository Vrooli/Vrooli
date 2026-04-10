import { useState } from "react";
import {
  CheckCircle2,
  ChevronDown,
  ChevronLeft,
  ChevronUp,
  Loader2,
  MoreHorizontal,
  Settings2,
  Trash2,
  XCircle,
} from "lucide-react";
import { Button } from "../ui/button";
import { TagList } from "../ui/tag-list";
import { DetailSection } from "../detail/DetailSection";
import { ReviewClassificationBadge } from "./ReviewClassificationBadge";
import { ScenarioCliHints } from "./ScenarioCliHints";
import { capitalize } from "../../lib";
import { SCENARIO_STATUS_COLORS, type ScenarioStatus } from "../../types";
import type { LucideIcon } from "lucide-react";

export interface ScenarioMobileViewProps {
  scenario: {
    displayName: string;
    description: string;
    status: ScenarioStatus;
    priority: number;
    tags: string[];
    completenessScore?: number;
    lastReviewClassification?: string;
    lastReviewAt?: string;
  };
  name: string;
  StatusIcon: LucideIcon;
  localGreenfield: boolean | null;
  onClose: () => void;
  onShowActionsSheet: () => void;
  actionError: string | null;
  // Settings
  onGreenfieldToggle: () => void;
  updatePending: boolean;
  updateError: boolean;
  // Delete
  onDeleteClick: () => void;
  deletePending: boolean;
  deleteError: boolean;
}

export function ScenarioMobileView({
  scenario,
  name,
  StatusIcon,
  localGreenfield,
  onClose,
  onShowActionsSheet,
  actionError,
  onGreenfieldToggle,
  updatePending,
  updateError,
  onDeleteClick,
  deletePending,
  deleteError,
}: ScenarioMobileViewProps) {
  const [mobileDangerExpanded, setMobileDangerExpanded] = useState(false);

  return (
    <div className="flex min-h-[100dvh] flex-col lg:hidden">
      <div className="sticky top-0 z-30 flex items-center gap-2 border-b border-slate-800 bg-slate-950/95 px-3 py-2 backdrop-blur">
        <Button
          variant="outline"
          size="sm"
          className="h-9 w-9 rounded-md border-transparent bg-transparent p-0 hover:bg-slate-800/70"
          onClick={onClose}
          aria-label="Close scenario details"
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-semibold text-slate-100">{scenario.displayName}</p>
          <p className="truncate text-xs text-slate-400">
            {capitalize(scenario.status)} · P{scenario.priority}
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          className="h-9 rounded-lg border-slate-700/80 bg-slate-900/45 px-3 text-xs font-medium text-slate-100 hover:bg-slate-800/70"
          onClick={onShowActionsSheet}
          aria-label="Open scenario actions"
        >
          Actions
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="h-9 w-9 rounded-md border-transparent bg-transparent p-0 hover:bg-slate-800/70"
          onClick={() => setMobileDangerExpanded((prev) => !prev)}
          aria-label={mobileDangerExpanded ? "Hide danger section" : "Show danger section"}
        >
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      </div>

      <div className="flex-1 space-y-0 overflow-y-auto pb-6">
        {actionError && (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
            {actionError}
          </div>
        )}

        <DetailSection title="Overview" hideDivider>
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <StatusIcon className={`h-4 w-4 ${SCENARIO_STATUS_COLORS[scenario.status]}`} />
              <span className="text-xs uppercase tracking-wider text-slate-500">
                {capitalize(scenario.status)}
              </span>
              <span className="rounded-full bg-slate-700 px-2.5 py-0.5 text-xs text-slate-300">P{scenario.priority}</span>
              {localGreenfield && (
                <span className="rounded-full bg-cyan-500/20 px-2 py-0.5 text-[11px] text-cyan-300">
                  Greenfield
                </span>
              )}
              {scenario.lastReviewClassification ? (
                <ReviewClassificationBadge
                  classification={scenario.lastReviewClassification}
                  reviewedAt={scenario.lastReviewAt}
                  showTimestamp
                />
              ) : (
                <span className="rounded-full bg-slate-500/20 px-2 py-0.5 text-[11px] text-slate-400">
                  Not reviewed
                </span>
              )}
            </div>
            <p className="text-sm leading-relaxed text-slate-300">
              {scenario.description || "No description provided"}
            </p>
            {scenario.completenessScore !== undefined && (
              <div className="flex items-center gap-2">
                <div className="h-2 flex-1 overflow-hidden rounded-full bg-slate-700">
                  <div
                    className="h-full bg-gradient-to-r from-cyan-500 to-purple-500"
                    style={{ width: `${scenario.completenessScore}%` }}
                  />
                </div>
                <span className="text-xs text-slate-400">{scenario.completenessScore}%</span>
              </div>
            )}
            {scenario.tags.length > 0 && <TagList tags={scenario.tags} maxTags={10} />}
          </div>
        </DetailSection>

        <DetailSection title="Scenario Settings" icon={Settings2}>
          <div className="space-y-3">
            {updatePending && (
              <Loader2 className="h-4 w-4 animate-spin text-cyan-400" />
            )}

            <div className="rounded-lg bg-slate-700/30 p-3">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-slate-200">Greenfield Mode</span>
                  {localGreenfield ? (
                    <CheckCircle2 className="h-4 w-4 text-cyan-400" />
                  ) : (
                    <XCircle className="h-4 w-4 text-slate-500" />
                  )}
                </div>
                <p className="text-xs text-slate-400">
                  Treat this scenario as a new project without existing code base
                </p>
              </div>
              <Button
                variant={localGreenfield ? "default" : "outline"}
                size="sm"
                className="mt-3 w-full"
                onClick={onGreenfieldToggle}
                disabled={updatePending}
              >
                {localGreenfield ? "Enabled" : "Disabled"}
              </Button>
            </div>

            {updateError && (
              <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                Failed to update settings. Please try again.
              </div>
            )}
          </div>
        </DetailSection>

        <ScenarioCliHints name={name} variant="mobile" />

        <section className="mt-4 border-t border-slate-800 pt-4">
          <button
            type="button"
            className="flex w-full items-center justify-between pt-3 pb-2 text-left"
            onClick={() => setMobileDangerExpanded((prev) => !prev)}
          >
            <span className="flex items-center gap-2 text-base font-semibold text-red-300">
              <Trash2 className="h-4 w-4" />
              Danger Zone
            </span>
            {mobileDangerExpanded ? (
              <ChevronUp className="h-4 w-4 text-red-300" />
            ) : (
              <ChevronDown className="h-4 w-4 text-red-300" />
            )}
          </button>
          {mobileDangerExpanded && (
            <div className="space-y-3 pb-3">
              <p className="text-sm text-slate-400">
                Permanently remove this scenario from the catalog. This action cannot be undone.
              </p>
              <Button
                variant="destructive"
                size="sm"
                className="w-full"
                onClick={onDeleteClick}
                disabled={deletePending}
              >
                {deletePending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Deleting...
                  </>
                ) : (
                  <>
                    <Trash2 className="mr-2 h-4 w-4" />
                    Delete Scenario
                  </>
                )}
              </Button>
              {deleteError && (
                <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                  Failed to delete scenario. Please try again.
                </div>
              )}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

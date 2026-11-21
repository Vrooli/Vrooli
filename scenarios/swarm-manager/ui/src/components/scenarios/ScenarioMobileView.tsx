import { useState, type ReactNode } from "react";
import {
  CheckCircle2,
  ClipboardList,
  ChevronDown,
  ChevronLeft,
  ChevronUp,
  Loader2,
  MoreHorizontal,
  ShieldCheck,
  Settings2,
  Trash2,
  XCircle,
} from "lucide-react";
import { Button } from "../ui/button";
import { DetailSection } from "../detail/DetailSection";
import { ScenarioCliHints } from "./ScenarioCliHints";
import { ScenarioOverviewSection } from "./ScenarioOverviewSection";
import type { ScenarioHealthSnapshot, ScenarioStatus } from "../../types";
import { capitalize } from "../../lib";
import type { LucideIcon } from "lucide-react";
import { CompactTabBar, type CompactTabItem } from "../ui/compact-tab-bar";

export type ScenarioDetailTab = "overview" | "work" | "quality" | "manage";

const MOBILE_TABS: CompactTabItem<ScenarioDetailTab>[] = [
  { value: "overview", label: "Overview", icon: CheckCircle2 },
  { value: "work", label: "Work", icon: ClipboardList },
  { value: "quality", label: "Quality", icon: ShieldCheck },
  { value: "manage", label: "Settings", icon: Settings2 },
];

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
    health?: ScenarioHealthSnapshot;
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
  workContent?: ReactNode;
  qualityContent?: ReactNode;
  activeTab: ScenarioDetailTab;
  onTabChange: (tab: ScenarioDetailTab) => void;
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
  workContent,
  qualityContent,
  activeTab,
  onTabChange,
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

      <div className="flex-1 space-y-0 overflow-y-auto px-4 pb-8 sm:px-6">
        {actionError && (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
            {actionError}
          </div>
        )}

        <div className="sticky top-0 z-20 -mx-4 border-b border-slate-800 bg-slate-950/95 backdrop-blur sm:-mx-6">
          <CompactTabBar
            items={MOBILE_TABS}
            activeValue={activeTab}
            onValueChange={onTabChange}
            aria-label="Scenario detail sections"
            className="w-full flex-nowrap justify-start gap-1 overflow-x-auto rounded-none bg-transparent px-4 sm:px-6"
            tabTestIdPrefix="scenario-detail-tab"
          />
        </div>

        <div className={activeTab === "overview" ? undefined : "hidden"} aria-hidden={activeTab !== "overview"}>
          <ScenarioOverviewSection
            scenario={scenario}
            StatusIcon={StatusIcon}
            localGreenfield={localGreenfield}
            onOpenWork={() => onTabChange("work")}
            onOpenQuality={() => onTabChange("quality")}
            evidenceState={scenario.health?.evidenceState}
            testIds={false}
          />
        </div>

        <div className={activeTab === "work" ? undefined : "hidden"} aria-hidden={activeTab !== "work"}>{workContent}</div>

        <div className={activeTab === "quality" ? undefined : "hidden"} aria-hidden={activeTab !== "quality"}>{qualityContent}</div>

        <div className={activeTab === "manage" ? undefined : "hidden"} aria-hidden={activeTab !== "manage"}>
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
    </div>
  );
}

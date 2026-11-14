import { ArrowRight, ClipboardList, Package, ShieldCheck } from "lucide-react";
import { TagList } from "../ui/tag-list";
import { DetailSection } from "../detail/DetailSection";
import { ReviewClassificationBadge } from "./ReviewClassificationBadge";
import { capitalize } from "../../lib";
import { selectors } from "../../consts/selectors";
import { SCENARIO_STATUS_COLORS, type ScenarioStatus } from "../../types";
import type { LucideIcon } from "lucide-react";

export interface ScenarioOverviewSectionProps {
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
  StatusIcon: LucideIcon;
  localGreenfield: boolean | null;
  /** Desktop action buttons rendered on the right side. */
  actionButtons?: React.ReactNode;
  /** Action error message, if any. */
  actionError?: string | null;
  /** Move directly to the work context from the orientation surface. */
  onOpenWork?: () => void;
  /** Move directly to quality evidence from the orientation surface. */
  onOpenQuality?: () => void;
  /** Test evidence freshness, used to make the quality destination legible. */
  evidenceState?: string;
  /** The desktop surface owns stable detail selectors; mobile shares the visual component. */
  testIds?: boolean;
}

/** Desktop overview section showing metadata, status, tags, and action buttons. */
export function ScenarioOverviewSection({
  scenario,
  StatusIcon,
  localGreenfield,
  actionButtons,
  actionError,
  onOpenWork,
  onOpenQuality,
  evidenceState,
  testIds = true,
}: ScenarioOverviewSectionProps) {
  const qualityLabel = evidenceState === "fresh" ? "Evidence is current" : "Evidence needs review";

  return (
    <DetailSection title="Scenario at a glance" icon={Package} hideDivider data-testid={testIds ? selectors.scenarioDetails.header : undefined}>
      <div className="overflow-hidden rounded-2xl border border-slate-800/80 bg-slate-900/35">
        <h1 className="sr-only" data-testid={testIds ? selectors.scenarioDetails.title : undefined}>{scenario.displayName}</h1>
        <div className="grid gap-5 p-4 lg:p-5 xl:grid-cols-[minmax(0,1fr)_auto]">
          <div className="min-w-0 space-y-4">
            <div className="flex flex-wrap items-center gap-2">
            <StatusIcon
              className={`h-4 w-4 ${SCENARIO_STATUS_COLORS[scenario.status]}`}
              data-testid={testIds ? selectors.scenarioDetails.status : undefined}
            />
            <span className="text-sm uppercase tracking-wider text-slate-500">
              {capitalize(scenario.status)}
            </span>
            <span
              className="rounded-full bg-slate-800 px-2.5 py-1 text-xs font-medium text-slate-300"
              data-testid={testIds ? selectors.scenarioDetails.priority : undefined}
            >
              P{scenario.priority}
            </span>
            {localGreenfield && (
              <span className="rounded-full bg-cyan-500/20 px-2 py-0.5 text-xs text-cyan-400">
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
              <span className="rounded-full bg-slate-500/20 px-2 py-0.5 text-xs text-slate-400">
                Not reviewed
              </span>
            )}
            </div>
            <div className="rounded-xl border border-slate-800/70 bg-slate-950/35 p-3.5">
              <p className="text-[11px] font-medium uppercase tracking-[0.16em] text-slate-500">Purpose</p>
              <p
                className="mt-1.5 max-w-3xl text-sm leading-6 text-slate-200"
                data-testid={testIds ? selectors.scenarioDetails.description : undefined}
              >
                {scenario.description || "No description provided"}
              </p>
              {scenario.tags.length > 0 && (
                <TagList
                  tags={scenario.tags}
                  maxTags={10}
                  className="mt-3"
                  data-testid={testIds ? selectors.scenarioDetails.tags : undefined}
                />
              )}
            </div>
          </div>

          <div className="flex min-w-0 flex-col gap-3 xl:w-64">
            {scenario.completenessScore !== undefined && (
              <div className="rounded-xl border border-slate-800 bg-slate-950/50 p-3">
                <div className="mb-1.5 flex items-center justify-between gap-6 text-[11px] uppercase tracking-wide text-slate-500">
                  <span>Delivery readiness</span>
                  <span className="text-slate-300">{scenario.completenessScore}%</span>
                </div>
                <div className="h-1.5 overflow-hidden rounded-full bg-slate-800">
                  <div
                    className="h-full bg-gradient-to-r from-cyan-500 to-purple-500"
                    style={{ width: `${scenario.completenessScore}%` }}
                  />
                </div>
              </div>
            )}
            {actionButtons}
            {actionError && (
              <div
                className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-1 text-xs text-red-400"
                data-testid={selectors.scenarioDetails.actionError}
              >
                {actionError}
              </div>
            )}
          </div>
        </div>

        <div className="grid border-t border-slate-800/80 sm:grid-cols-2">
          <button
            type="button"
            onClick={onOpenWork}
            disabled={!onOpenWork}
            className="group flex min-w-0 items-start gap-3 px-4 py-4 text-left transition-colors hover:bg-slate-800/35 disabled:cursor-default disabled:hover:bg-transparent lg:px-5"
            data-testid="scenario-overview-work-link"
          >
            <span className="rounded-lg bg-cyan-500/10 p-2 text-cyan-300"><ClipboardList className="h-4 w-4" /></span>
            <span className="min-w-0 flex-1">
              <span className="block text-sm font-medium text-slate-100">Planned work</span>
              <span className="mt-0.5 block text-xs leading-5 text-slate-400">See goals, backlog items, and the work still outside a goal.</span>
            </span>
            <ArrowRight className="mt-1 h-4 w-4 shrink-0 text-slate-600 transition-transform group-hover:translate-x-0.5 group-hover:text-cyan-300" />
          </button>
          <button
            type="button"
            onClick={onOpenQuality}
            disabled={!onOpenQuality}
            className="group flex min-w-0 items-start gap-3 border-t border-slate-800/80 px-4 py-4 text-left transition-colors hover:bg-slate-800/35 disabled:cursor-default disabled:hover:bg-transparent sm:border-l sm:border-t-0 lg:px-5"
            data-testid="scenario-overview-quality-link"
          >
            <span className="rounded-lg bg-violet-500/10 p-2 text-violet-300"><ShieldCheck className="h-4 w-4" /></span>
            <span className="min-w-0 flex-1">
              <span className="block text-sm font-medium text-slate-100">Quality evidence</span>
              <span className="mt-0.5 block text-xs leading-5 text-slate-400">{qualityLabel}. Review Test Genie findings and create governed remediation.</span>
            </span>
            <ArrowRight className="mt-1 h-4 w-4 shrink-0 text-slate-600 transition-transform group-hover:translate-x-0.5 group-hover:text-violet-300" />
          </button>
        </div>
      </div>
    </DetailSection>
  );
}

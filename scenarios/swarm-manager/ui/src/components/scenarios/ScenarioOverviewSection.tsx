import { Package } from "lucide-react";
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
}

/** Desktop overview section showing metadata, status, tags, and action buttons. */
export function ScenarioOverviewSection({
  scenario,
  StatusIcon,
  localGreenfield,
  actionButtons,
  actionError,
}: ScenarioOverviewSectionProps) {
  return (
    <DetailSection title="Overview" hideDivider data-testid={selectors.scenarioDetails.header}>
      <div className="flex items-start justify-between">
        <div className="space-y-2">
          <div className="flex items-center gap-3">
            <StatusIcon
              className={`h-4 w-4 ${SCENARIO_STATUS_COLORS[scenario.status]}`}
              data-testid={selectors.scenarioDetails.status}
            />
            <span className="text-sm uppercase tracking-wider text-slate-500">
              {capitalize(scenario.status)}
            </span>
            <span
              className="rounded-full bg-slate-700 px-3 py-1 text-sm text-slate-300"
              data-testid={selectors.scenarioDetails.priority}
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
          <h1
            className="text-2xl font-bold text-slate-100"
            data-testid={selectors.scenarioDetails.title}
          >
            {scenario.displayName}
          </h1>
          <p
            className="text-slate-400"
            data-testid={selectors.scenarioDetails.description}
          >
            {scenario.description || "No description provided"}
          </p>
          <TagList
            tags={scenario.tags}
            maxTags={10}
            className="mt-4"
            data-testid={selectors.scenarioDetails.tags}
          />
        </div>

        <div className="flex flex-col items-end gap-2">
          <Package className="h-12 w-12 text-slate-600" />
          {scenario.completenessScore !== undefined && (
            <div className="flex items-center gap-2">
              <div className="h-2 w-24 overflow-hidden rounded-full bg-slate-700">
                <div
                  className="h-full bg-gradient-to-r from-cyan-500 to-purple-500"
                  style={{ width: `${scenario.completenessScore}%` }}
                />
              </div>
              <span className="text-sm text-slate-400">{scenario.completenessScore}%</span>
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
    </DetailSection>
  );
}

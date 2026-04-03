/**
 * ScenarioCard - Single scenario row with status, metadata, and action buttons.
 *
 * Extracted from ScenariosPage.tsx to reduce component size and improve testability.
 */

import { type MouseEvent } from "react";
import { ArrowRight, Circle, Loader2, Play, RefreshCw, Square } from "lucide-react";
import { Button } from "../components/ui/button";
import { ResponsiveListItem } from "../components/ui/responsive-list";
import { TagList } from "../components/ui/tag-list";
import { SCENARIO_STATUS_ICONS, SCENARIO_STATUS_COLORS, type ScenarioStatus } from "../types";
import { displayLimitsConfig } from "../config";
import { selectors } from "../consts/selectors";

export type ScenarioAction = "start" | "stop" | "restart";

export interface ScenarioCardProps {
  name: string;
  displayName: string;
  description: string;
  status: ScenarioStatus;
  priority: number;
  isGreenfield: boolean;
  tags: string[];
  completenessScore?: number;
  /** Whether any mutation is currently pending (disables all action buttons). */
  isAnyActionPending: boolean;
  /** Whether a specific action is pending for this scenario. */
  isActionPending: (action: ScenarioAction) => boolean;
  onAction: (event: MouseEvent<HTMLButtonElement>, action: ScenarioAction) => void;
  onNavigate: () => void;
}

export function ScenarioCard({
  name,
  displayName,
  description,
  status,
  priority,
  isGreenfield,
  tags,
  completenessScore,
  isAnyActionPending,
  isActionPending,
  onAction,
  onNavigate,
}: ScenarioCardProps) {
  const StatusIcon = SCENARIO_STATUS_ICONS[status] || Circle;

  return (
    <ResponsiveListItem
      className="group cursor-pointer"
      interactive
      data-testid={selectors.scenarios.cardByName({ name })}
      role="link"
      tabIndex={0}
      onClick={onNavigate}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onNavigate();
        }
      }}
    >
      <div className="flex items-start gap-4">
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <StatusIcon
              className={`h-4 w-4 ${SCENARIO_STATUS_COLORS[status]}`}
            />
            <h3 className="font-medium text-slate-100">{displayName}</h3>
            {isGreenfield && (
              <span className="rounded-full bg-cyan-500/20 px-2 py-0.5 text-xs text-cyan-400">
                Greenfield
              </span>
            )}
          </div>
          <p className="mt-1 text-sm text-slate-400">{description}</p>
          <TagList
            tags={tags}
            maxTags={displayLimitsConfig.scenarioCardMaxTags}
            className="mt-2"
          />
        </div>
        <div className="flex flex-col items-end gap-2">
          <span className="rounded-full bg-slate-700 px-2 py-0.5 text-xs text-slate-300">
            P{priority}
          </span>
          {completenessScore !== undefined && (
            <div className="flex items-center gap-1">
              <div className="h-1.5 w-16 overflow-hidden rounded-full bg-slate-700">
                <div
                  className="h-full bg-gradient-to-r from-cyan-500 to-purple-500"
                  style={{ width: `${completenessScore}%` }}
                />
              </div>
              <span className="text-xs text-slate-400">{completenessScore}%</span>
            </div>
          )}
          <div className="flex flex-wrap items-center justify-end gap-1">
            <Button
              variant="outline"
              size="sm"
              className="h-7 px-2 text-xs"
              onClick={(event) => onAction(event, "start")}
              disabled={isAnyActionPending || status === "running"}
              data-testid={selectors.scenarios.actionStart({ name })}
            >
              {isActionPending("start") ? (
                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
              ) : (
                <Play className="mr-1 h-3 w-3" />
              )}
              Start
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="h-7 px-2 text-xs"
              onClick={(event) => onAction(event, "stop")}
              disabled={isAnyActionPending || status === "stopped"}
              data-testid={selectors.scenarios.actionStop({ name })}
            >
              {isActionPending("stop") ? (
                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
              ) : (
                <Square className="mr-1 h-3 w-3" />
              )}
              Stop
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="h-7 px-2 text-xs"
              onClick={(event) => onAction(event, "restart")}
              disabled={isAnyActionPending}
              data-testid={selectors.scenarios.actionRestart({ name })}
            >
              {isActionPending("restart") ? (
                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
              ) : (
                <RefreshCw className="mr-1 h-3 w-3" />
              )}
              Restart
            </Button>
          </div>
          <ArrowRight className="h-4 w-4 text-slate-500 opacity-0 transition group-hover:opacity-100" />
        </div>
      </div>
    </ResponsiveListItem>
  );
}

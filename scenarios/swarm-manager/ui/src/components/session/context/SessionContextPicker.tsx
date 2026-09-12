import { useEffect, useMemo, useState } from "react";
import { Search } from "lucide-react";
import { BottomSheet } from "../../ui/bottom-sheet";
import { Button } from "../../ui/button";
import { Input } from "../../ui/input";
import { CompactTabBar } from "../../ui/compact-tab-bar";
import { cn } from "../../../lib/utils";
import { selectors } from "../../../consts/selectors";
import {
  useAgentActivitiesStore,
  useAgentSessionStore,
  useBacklogStore,
  useCaptureStore,
  useExecutionStore,
  useScenariosStore,
} from "../../../stores";
import type { AgentSessionContextType, AgentSessionKind, AgentSession, BacklogItem, Capture, ExecutionRecord, Scenario } from "../../../types";
import { ContextChipTray } from "../../composer/ContextChipTray";
import { BacklogCard } from "../../backlog/backlog-card";
import { GoalProgressSummary } from "../../goals/GoalProgressCard";
import { CollectionRow } from "../../ui/collection-row";
import { ExecutionSummaryCard } from "../../execution/execution-summary-card";
import { ScenarioSummaryCard } from "../../scenario/scenario-summary-card";
import { SessionSummaryCard } from "../session-summary-card";
import { CaptureCard } from "../../capture/capture-card";
import { CollectionList } from "@vrooli/react-component-library/CollectionList/1";
import { allowedContextTypesForKind, CONTEXT_TYPE_CAPS, CONTEXT_TYPE_LABELS, totalContextCapForKind } from "./session-context-config";
import { buildContextOptionsByType } from "./session-context-options";
import { backlogItemIsStale, executionIsFailedOrStale, STARTER_FILTER_TARGET_TYPE, type StarterContextFilterKey } from "./starter-context-filters";
import {
  activityOption,
  backlogOption,
  captureOption,
  contextKey,
  executionOption,
  operationsBriefingOption,
  scenarioOption,
  sessionOption,
  startupBriefOption,
  type SessionContextOption,
} from "./session-context-refs";
import { goalsService } from "../../../services/goals-service";
import type { GoalWithScope } from "../../../types/goal";

interface SessionContextPickerProps {
  isOpen: boolean;
  onClose: () => void;
  sessionKind: AgentSessionKind;
  selected: SessionContextOption[];
  onApply: (items: SessionContextOption[]) => void;
  currentSessionId?: string;
  initialType?: AgentSessionContextType | null;
  /**
   * Narrows one type's list to the actionable subset when opened from a starter
   * card (e.g. "failed or stale" executions), so the picker matches that card's
   * count badge. Null/absent → show the full list (e.g. the composer's +context).
   */
  initialFilterKey?: StarterContextFilterKey | null;
}

export function SessionContextPicker({
  isOpen,
  ...props
}: SessionContextPickerProps) {
  if (!isOpen) return null;
  return <SessionContextPickerContent isOpen={isOpen} {...props} />;
}

function SessionContextPickerContent({
  isOpen,
  onClose,
  sessionKind,
  selected,
  onApply,
  currentSessionId,
  initialType,
  initialFilterKey,
}: SessionContextPickerProps) {
  const allowedTypes = useMemo(() => allowedContextTypesForKind(sessionKind), [sessionKind]);
	const [activeType, setActiveType] = useState<AgentSessionContextType>(allowedTypes[0] ?? "goal");
  const [query, setQuery] = useState("");
  const [draft, setDraft] = useState<SessionContextOption[]>(selected);

  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const backlogItems = useBacklogStore((s) => s.items);
	const [goals, setGoals] = useState<GoalWithScope[]>([]);
  const fetchCaptures = useCaptureStore((s) => s.fetchCaptures);
  const captures = useCaptureStore((s) => s.captures);
  const fetchExecutions = useExecutionStore((s) => s.fetchExecutions);
  const executions = useExecutionStore((s) => s.items);
  const refreshActivities = useAgentActivitiesStore((s) => s.refreshActivities);
  const activities = useAgentActivitiesStore((s) => s.activities);
  const fetchScenarios = useScenariosStore((s) => s.fetchScenarios);
  const scenarios = useScenariosStore((s) => s.scenarios);
  const fetchSessions = useAgentSessionStore((s) => s.fetchSessions);
  const sessions = useAgentSessionStore((s) => s.sessions);

  useEffect(() => {
    if (!isOpen) return;
    setDraft(selected);
    setActiveType((current) => {
      if (initialType && allowedTypes.includes(initialType)) return initialType;
		return allowedTypes.includes(current) ? current : allowedTypes[0] ?? "goal";
    });
    setQuery("");
    void fetchBacklog();
		void goalsService.list().then(setGoals).catch(() => setGoals([]));
    void fetchCaptures();
    void fetchExecutions();
    void refreshActivities(false);
    void fetchScenarios();
    void fetchSessions({ limit: 100 });
	}, [allowedTypes, fetchBacklog, fetchCaptures, fetchExecutions, fetchScenarios, fetchSessions, initialType, isOpen, refreshActivities, selected]);

  // Narrowing is applied to the source records, not the built options, so the
  // tab list and its count come from one filtered set — the option shape has
  // no staleness of its own to filter on.
  const filterStaleBacklog = initialFilterKey === "backlog_item_stale";
  const visibleBacklogItems = useMemo(
    () => (filterStaleBacklog ? backlogItems.filter(backlogItemIsStale) : backlogItems),
    [backlogItems, filterStaleBacklog],
  );

  const optionsByType = useMemo<Record<AgentSessionContextType, SessionContextOption[]>>(() => buildContextOptionsByType({
    backlogItems: visibleBacklogItems,
		goals,
    captures,
    executions,
    activities,
    scenarios,
    sessions,
    sessionKind,
    currentSessionId,
	}), [activities, visibleBacklogItems, captures, currentSessionId, executions, goals, scenarios, sessionKind, sessions]);

  // Phase-3 narrowing: when opened from a starter card carrying a filter key,
  // the targeted type's list (and its tab count) shrink to the actionable subset,
  // mirroring that card's badge. Other tabs are unaffected.
  const filterExecutions = initialFilterKey === "execution_failed_or_stale";
  const visibleExecutions = useMemo(
    () => (filterExecutions ? executions.filter((execution) => executionIsFailedOrStale(execution)) : executions),
    [executions, filterExecutions],
  );
  const tabCountFor = (type: AgentSessionContextType): number => {
    if (filterExecutions && type === STARTER_FILTER_TARGET_TYPE.execution_failed_or_stale) {
      return visibleExecutions.length;
    }
    return optionsByType[type]?.length ?? 0;
  };

  const selectedKeys = useMemo(() => new Set(draft.map((item) => contextKey(item.type, item.ref))), [draft]);

  const totalCap = totalContextCapForKind(sessionKind);
  const activeTypeCount = draft.filter((item) => item.type === activeType).length;
  const activeTypeCap = CONTEXT_TYPE_CAPS[activeType];
  const capMessage = draft.length >= totalCap
    ? `This session kind allows ${totalCap} context items per message.`
    : activeTypeCount >= activeTypeCap
      ? `${CONTEXT_TYPE_LABELS[activeType]} allows ${activeTypeCap} selections.`
      : "";

  const remove = (type: AgentSessionContextType, ref: string) => {
    setDraft((items) => items.filter((item) => !(item.type === type && item.ref === ref)));
  };

  // Cap state for the active tab. All not-yet-selected items in the active
  // list share this disabled state (selection policy lives here, not in cards).
  const capReached = draft.length >= totalCap || activeTypeCount >= activeTypeCap;

  const matchesNeedle = (option: SessionContextOption): boolean => {
    const needle = query.trim().toLowerCase();
    if (!needle) return true;
    return `${option.title} ${option.subtitle ?? ""} ${option.ref}`.toLowerCase().includes(needle);
  };

  type PickerRow = {
    option: SessionContextOption;
    entity?: BacklogItem | GoalWithScope | Capture | ExecutionRecord | AgentSession | Scenario;
  };

  const pickerRows = useMemo<PickerRow[]>(() => {
    switch (activeType) {
      case "backlog_item":
        return visibleBacklogItems
          .map((entity) => ({ entity, option: backlogOption(entity) }))
          .filter(({ option }) => matchesNeedle(option))
          .slice(0, 80)
          .map(({ entity, option }) => ({ entity, option }));
      case "goal":
        return (optionsByType[activeType] ?? [])
          .filter(matchesNeedle)
          .slice(0, 80)
          .map((option) => ({ option, entity: goals.find((goal) => goal.goal.name === option.ref) }));
      case "execution":
        return visibleExecutions
          .map((entity) => ({ entity, option: executionOption(entity) }))
          .filter(({ option }) => matchesNeedle(option))
          .slice(0, 80)
          .map(({ entity, option }) => ({ entity, option }));
      case "session":
        return sessions
          .filter((session) => session.id !== currentSessionId)
          .map((entity) => ({ entity, option: sessionOption(entity) }))
          .filter(({ option }) => matchesNeedle(option))
          .slice(0, 80)
          .map(({ entity, option }) => ({ entity, option }));
      case "scenario":
        return scenarios
          .map((entity) => ({ entity, option: scenarioOption(entity) }))
          .filter(({ option }) => matchesNeedle(option))
          .slice(0, 80)
          .map(({ entity, option }) => ({ entity, option }));
      case "capture":
        return captures.map((entity) => ({ entity, option: captureOption(entity) })).filter(({ option }) => matchesNeedle(option)).slice(0, 80);
      case "agent_activity":
        return activities.map(activityOption).filter(matchesNeedle).slice(0, 80).map((option) => ({ option }));
      case "operations_briefing":
        return [operationsBriefingOption()].filter(matchesNeedle).map((option) => ({ option }));
      case "startup_brief":
        return [startupBriefOption(sessionKind)].filter(matchesNeedle).map((option) => ({ option }));
      default:
        return [];
    }
  }, [activities, activeType, captures, currentSessionId, executions, goals, matchesNeedle, optionsByType, scenarios, sessionKind, sessions, visibleBacklogItems, visibleExecutions]);

  const selectedKeysForList = useMemo(() => [...selectedKeys], [selectedKeys]);
  const selection = useMemo(() => ({
    mode: "multi" as const,
    retain: "keep" as const,
    selected: selectedKeysForList,
    selectable: (row: PickerRow) => {
      const selected = selectedKeys.has(contextKey(row.option.type, row.option.ref));
      return !selected && capReached ? (capMessage || "Selection limit reached") : false;
    },
    onChange: (keys: string[]) => {
      const visibleKeys = new Set(keys);
      setDraft((current) => {
        const next = new Map(current.map((item) => [contextKey(item.type, item.ref), item]));
        pickerRows.forEach(({ option }) => {
          const key = contextKey(option.type, option.ref);
          if (visibleKeys.has(key)) next.set(key, option);
          else next.delete(key);
        });
        return [...next.values()];
      });
    },
  }), [capMessage, capReached, pickerRows, selectedKeys, selectedKeysForList]);

  const renderPickerRow = (row: PickerRow) => <CollectionRow>{pickerRowContent(row)}</CollectionRow>;

  const pickerRowContent = (row: PickerRow) => {
    if (row.option.type === "backlog_item" && row.entity) {
      return <BacklogCard item={row.entity as BacklogItem} />;
    }
    if (row.option.type === "goal" && row.entity) {
      const goal = row.entity as GoalWithScope;
      return <GoalProgressSummary title={goal.goal.title || goal.goal.name} subtitle={row.option.subtitle} priority={goal.goal.priority} completed={goal.scope.completedCount} total={goal.scope.total} targets={goal.scope.targets.length} ready={goal.scope.ready.length} blocked={goal.scope.blockedCount} />;
    }
    if (row.option.type === "capture" && row.entity) {
      return <CaptureCard capture={row.entity as Capture} />;
    }
    if (row.option.type === "execution" && row.entity) {
      return <ExecutionSummaryCard item={row.entity as ExecutionRecord} />;
    }
    if (row.option.type === "session" && row.entity) {
      return <SessionSummaryCard session={row.entity as AgentSession} />;
    }
    if (row.option.type === "scenario" && row.entity) {
      return <ScenarioSummaryCard scenario={row.entity as Scenario} />;
    }
    return (
      <>
        <span className="block truncate text-sm font-medium leading-5">{row.option.title}</span>
        <span className="block truncate text-xs leading-5 text-slate-400">{row.option.subtitle || row.option.ref}</span>
      </>
    );
  };

  return (
    <BottomSheet
      isOpen={isOpen}
      onClose={onClose}
      title="Attach context"
      description="Select existing work to include with this message."
      className="!max-w-3xl border-slate-700/80 bg-slate-900"
      contentClassName="px-0 py-0"
      footer={
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p className={cn("text-xs", capMessage ? "text-amber-300" : "text-slate-400")}>
            {capMessage || `${draft.length}/${totalCap} context items selected.`}
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="ghost" size="sm" onClick={onClose}>Cancel</Button>
            <Button
              size="sm"
              onClick={() => {
                onApply(draft);
                onClose();
              }}
              data-testid={selectors.agentSessions.contextAttachButton}
            >
              Attach
            </Button>
          </div>
        </div>
      }
      data-testid={selectors.agentSessions.contextPicker}
    >
      <div className="flex min-h-0 flex-col">
        <div className="space-y-2.5 border-b border-white/10 px-3 py-2.5 sm:px-4">
          <ContextChipTray
            items={draft}
            onRemove={remove}
            className="max-h-16"
            testId={selectors.agentSessions.contextSelectedTray}
          />

          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-500" />
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Search context..."
              className="h-9 border-slate-700 bg-slate-950/70 pl-9 text-slate-100 placeholder:text-slate-500"
              data-testid={selectors.agentSessions.contextSearch}
            />
          </div>
        </div>

        <CompactTabBar
          items={allowedTypes.map((type) => ({
            value: type,
            label: CONTEXT_TYPE_LABELS[type],
            count: tabCountFor(type),
          }))}
          activeValue={activeType}
          onValueChange={setActiveType}
          aria-label="Context types"
          className="border-b border-white/10 px-1"
          tabTestIdPrefix="session-context-tab"
        />

        <div className="max-h-[56vh] overflow-y-auto px-2.5 py-2.5 sm:max-h-[50vh] sm:px-3" data-testid={selectors.agentSessions.contextEntityList}>
          {pickerRows.length > 0 ? (
            <CollectionList
              items={pickerRows}
              getKey={(row) => contextKey(row.option.type, row.option.ref)}
              label={`${CONTEXT_TYPE_LABELS[activeType]} context`}
              selection={selection}
              bulkBar="none"
              renderItem={renderPickerRow}
            />
          ) : (
            <div className="rounded-md border border-dashed border-slate-700 bg-slate-950/40 px-3 py-10 text-center text-sm text-slate-500">
              No matching context.
            </div>
          )}
        </div>
      </div>
    </BottomSheet>
  );
}

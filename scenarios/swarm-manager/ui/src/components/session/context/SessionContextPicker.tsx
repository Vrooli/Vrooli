import { useEffect, useMemo, useState } from "react";
import { Search } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { BottomSheet } from "../../ui/bottom-sheet";
import { Button } from "../../ui/button";
import { Input } from "../../ui/input";
import { CompactTabBar } from "../../ui/compact-tab-bar";
import { cn } from "../../../lib/utils";
import { selectors } from "../../../consts/selectors";
import { initiativeModeService } from "../../../services";
import {
  useAgentActivitiesStore,
  useAgentSessionStore,
  useBacklogStore,
  useCaptureStore,
  useExecutionStore,
  useInitiativeStore,
  useScenariosStore,
} from "../../../stores";
import type { AgentSessionContextType, AgentSessionKind } from "../../../types";
import { ContextChipTray } from "../../composer/ContextChipTray";
import { BacklogCard } from "../../backlog/backlog-card";
import { InitiativeSummaryCard } from "../../initiative/initiative-summary-card";
import { ExecutionSummaryCard } from "../../execution/execution-summary-card";
import { ScenarioSummaryCard } from "../../scenario/scenario-summary-card";
import { OperatingModeCard } from "../../initiative/operating-mode/operating-mode-card";
import { SessionSummaryCard } from "../session-summary-card";
import { PickModeRow } from "./selectable-card";
import type { CardSelection } from "./selectable";
import { allowedContextTypesForKind, CONTEXT_TYPE_CAPS, CONTEXT_TYPE_LABELS, totalContextCapForKind } from "./session-context-config";
import { buildContextOptionsByType } from "./session-context-options";
import { executionIsFailedOrStale, STARTER_FILTER_TARGET_TYPE, type StarterContextFilterKey } from "./starter-context-filters";
import {
  activityOption,
  backlogOption,
  captureOption,
  contextKey,
  executionOption,
  initiativeOption,
  operatingModeOption,
  operationsBriefingOption,
  scenarioOption,
  sessionOption,
  startupBriefOption,
  type SessionContextOption,
} from "./session-context-refs";

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
  const [activeType, setActiveType] = useState<AgentSessionContextType>(allowedTypes[0] ?? "initiative");
  const [query, setQuery] = useState("");
  const [draft, setDraft] = useState<SessionContextOption[]>(selected);

  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const backlogItems = useBacklogStore((s) => s.items);
  const fetchInitiatives = useInitiativeStore((s) => s.fetchInitiatives);
  const initiatives = useInitiativeStore((s) => s.items);
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
  const modesQuery = useQuery({
    queryKey: ["operating-modes", "catalog"],
    queryFn: () => initiativeModeService.catalog(),
    enabled: isOpen && allowedTypes.includes("operating_mode"),
  });

  useEffect(() => {
    if (!isOpen) return;
    setDraft(selected);
    setActiveType((current) => {
      if (initialType && allowedTypes.includes(initialType)) return initialType;
      return allowedTypes.includes(current) ? current : allowedTypes[0] ?? "initiative";
    });
    setQuery("");
    void fetchBacklog();
    void fetchInitiatives();
    void fetchCaptures();
    void fetchExecutions();
    void refreshActivities(false);
    void fetchScenarios();
    void fetchSessions({ limit: 100 });
  }, [allowedTypes, fetchBacklog, fetchCaptures, fetchExecutions, fetchInitiatives, fetchScenarios, fetchSessions, initialType, isOpen, refreshActivities, selected]);

  const optionsByType = useMemo<Record<AgentSessionContextType, SessionContextOption[]>>(() => buildContextOptionsByType({
    backlogItems,
    initiatives,
    captures,
    executions,
    activities,
    scenarios,
    modes: modesQuery.data?.modes ?? [],
    sessions,
    sessionKind,
    currentSessionId,
  }), [activities, backlogItems, captures, currentSessionId, executions, initiatives, modesQuery.data?.modes, scenarios, sessionKind, sessions]);

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

  const toggle = (option: SessionContextOption) => {
    const key = contextKey(option.type, option.ref);
    if (selectedKeys.has(key)) {
      setDraft((items) => items.filter((item) => contextKey(item.type, item.ref) !== key));
      return;
    }
    if (draft.length >= totalCap || draft.filter((item) => item.type === option.type).length >= CONTEXT_TYPE_CAPS[option.type]) {
      return;
    }
    setDraft((items) => [...items, option]);
  };

  const remove = (type: AgentSessionContextType, ref: string) => {
    setDraft((items) => items.filter((item) => !(item.type === type && item.ref === ref)));
  };

  // Cap state for the active tab. All not-yet-selected items in the active
  // list share this disabled state (selection policy lives here, not in cards).
  const capReached = draft.length >= totalCap || activeTypeCount >= activeTypeCap;

  const selectionFor = (option: SessionContextOption): CardSelection => {
    const selected = selectedKeys.has(contextKey(option.type, option.ref));
    const disabled = !selected && capReached;
    return {
      selectionMode: true,
      selected,
      disabled,
      disabledReason: disabled ? capMessage : undefined,
      onToggleSelect: () => toggle(option),
    };
  };

  const matchesNeedle = (option: SessionContextOption): boolean => {
    const needle = query.trim().toLowerCase();
    if (!needle) return true;
    return `${option.title} ${option.subtitle ?? ""} ${option.ref}`.toLowerCase().includes(needle);
  };

  // Singleton / cardless / deferred (capture, agent_activity) types keep the
  // flat title+subtitle row via the shared PickModeRow.
  const fallbackRow = (option: SessionContextOption) => (
    <PickModeRow key={contextKey(option.type, option.ref)} selection={selectionFor(option)}>
      <span className="block truncate text-sm font-medium leading-5">{option.title}</span>
      <span className="block truncate text-xs leading-5 text-slate-400">{option.subtitle || option.ref}</span>
    </PickModeRow>
  );

  const renderPickNodes = () => {
    switch (activeType) {
      case "backlog_item":
        return backlogItems
          .map((entity) => ({ entity, option: backlogOption(entity) }))
          .filter(({ option }) => matchesNeedle(option))
          .slice(0, 80)
          .map(({ entity, option }) => (
            <BacklogCard key={contextKey(option.type, option.ref)} item={entity} selection={selectionFor(option)} />
          ));
      case "initiative":
        return initiatives
          .map((entity) => ({ entity, option: initiativeOption(entity) }))
          .filter(({ option }) => matchesNeedle(option))
          .slice(0, 80)
          .map(({ entity, option }) => (
            <InitiativeSummaryCard key={contextKey(option.type, option.ref)} item={entity} selection={selectionFor(option)} />
          ));
      case "execution":
        return visibleExecutions
          .map((entity) => ({ entity, option: executionOption(entity) }))
          .filter(({ option }) => matchesNeedle(option))
          .slice(0, 80)
          .map(({ entity, option }) => (
            <ExecutionSummaryCard key={contextKey(option.type, option.ref)} item={entity} selection={selectionFor(option)} />
          ));
      case "session":
        return sessions
          .filter((session) => session.id !== currentSessionId)
          .map((entity) => ({ entity, option: sessionOption(entity) }))
          .filter(({ option }) => matchesNeedle(option))
          .slice(0, 80)
          .map(({ entity, option }) => (
            <SessionSummaryCard key={contextKey(option.type, option.ref)} session={entity} selection={selectionFor(option)} />
          ));
      case "scenario":
        return scenarios
          .map((entity) => ({ entity, option: scenarioOption(entity) }))
          .filter(({ option }) => matchesNeedle(option))
          .slice(0, 80)
          .map(({ entity, option }) => (
            <ScenarioSummaryCard key={contextKey(option.type, option.ref)} scenario={entity} selection={selectionFor(option)} />
          ));
      case "operating_mode":
        return (modesQuery.data?.modes ?? [])
          .map((entity) => ({ entity, option: operatingModeOption(entity) }))
          .filter(({ option }) => option.ref && matchesNeedle(option))
          .slice(0, 80)
          .map(({ entity, option }) => (
            <OperatingModeCard key={contextKey(option.type, option.ref)} mode={entity} selection={selectionFor(option)} />
          ));
      case "capture":
        return captures.map(captureOption).filter(matchesNeedle).slice(0, 80).map(fallbackRow);
      case "agent_activity":
        return activities.map(activityOption).filter(matchesNeedle).slice(0, 80).map(fallbackRow);
      case "operations_briefing":
        return [operationsBriefingOption()].filter(matchesNeedle).map(fallbackRow);
      case "startup_brief":
        return [startupBriefOption(sessionKind)].filter(matchesNeedle).map(fallbackRow);
      default:
        return [];
    }
  };

  const pickNodes = renderPickNodes();

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
          {pickNodes.length > 0 ? (
            <div className="space-y-1.5">{pickNodes}</div>
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

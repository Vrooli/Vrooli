import { useEffect, useMemo, useState } from "react";
import { Check, Search } from "lucide-react";
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
import { allowedContextTypesForKind, CONTEXT_TYPE_CAPS, CONTEXT_TYPE_LABELS, totalContextCapForKind } from "./session-context-config";
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

  const optionsByType = useMemo<Record<AgentSessionContextType, SessionContextOption[]>>(() => ({
    backlog_item: backlogItems.map(backlogOption),
    initiative: initiatives.map(initiativeOption),
    capture: captures.map(captureOption),
    execution: executions.map(executionOption),
    agent_activity: activities.map(activityOption),
    scenario: scenarios.map(scenarioOption),
    operating_mode: (modesQuery.data?.modes ?? []).map(operatingModeOption).filter((mode) => mode.ref),
    session: sessions.filter((session) => session.id !== currentSessionId).map(sessionOption),
    operations_briefing: [operationsBriefingOption()],
    startup_brief: [startupBriefOption(sessionKind)],
  }), [activities, backlogItems, captures, currentSessionId, executions, initiatives, modesQuery.data?.modes, scenarios, sessionKind, sessions]);

  const selectedKeys = useMemo(() => new Set(draft.map((item) => contextKey(item.type, item.ref))), [draft]);
  const filteredOptions = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const options = optionsByType[activeType] ?? [];
    if (!needle) return options.slice(0, 80);
    return options
      .filter((option) => `${option.title} ${option.subtitle ?? ""} ${option.ref}`.toLowerCase().includes(needle))
      .slice(0, 80);
  }, [activeType, optionsByType, query]);

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

  const activeTypeItems = filteredOptions;

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
        <div className="space-y-3 border-b border-white/10 px-4 py-3">
          <ContextChipTray
            items={draft}
            onRemove={remove}
            className="max-h-24"
            testId={selectors.agentSessions.contextSelectedTray}
          />

          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-500" />
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Search context..."
              className="h-10 border-slate-700 bg-slate-950/70 pl-9 text-slate-100 placeholder:text-slate-500"
              data-testid={selectors.agentSessions.contextSearch}
            />
          </div>
        </div>

        <CompactTabBar
          items={allowedTypes.map((type) => ({
            value: type,
            label: CONTEXT_TYPE_LABELS[type],
            count: optionsByType[type]?.length ?? 0,
          }))}
          activeValue={activeType}
          onValueChange={setActiveType}
          aria-label="Context types"
          className="border-b border-white/10 px-1"
          tabTestIdPrefix="session-context-tab"
        />

        <div className="max-h-[52vh] overflow-y-auto px-3 py-3 sm:max-h-[48vh]" data-testid={selectors.agentSessions.contextEntityList}>
          {activeTypeItems.length > 0 ? (
            <div className="space-y-1.5">
              {activeTypeItems.map((option) => {
                const checked = selectedKeys.has(contextKey(option.type, option.ref));
                return (
                  <button
                    key={contextKey(option.type, option.ref)}
                    type="button"
                    onClick={() => toggle(option)}
                    className={cn(
                      "flex w-full items-start gap-3 rounded-md border px-3 py-2.5 text-left transition-colors",
                      checked
                        ? "border-cyan-400/50 bg-cyan-400/10 text-cyan-50"
                        : "border-slate-800 bg-slate-950/45 text-slate-200 hover:border-slate-700 hover:bg-slate-800/55",
                    )}
                    data-testid={selectors.agentSessions.contextRow}
                  >
                    <span
                      className={cn(
                        "mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded border transition-colors",
                        checked ? "border-cyan-300 bg-cyan-300 text-slate-950" : "border-slate-600 bg-slate-900",
                      )}
                    >
                      {checked && <Check className="h-3.5 w-3.5" />}
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-medium leading-5">{option.title}</span>
                      <span className="block truncate text-xs leading-5 text-slate-400">{option.subtitle || option.ref}</span>
                    </span>
                  </button>
                );
              })}
            </div>
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

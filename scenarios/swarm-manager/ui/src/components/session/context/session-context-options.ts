import type {
  AgentActivity,
  AgentSession,
  AgentSessionContextRef,
  AgentSessionContextType,
  AgentSessionKind,
  BacklogItem,
  Capture,
  ExecutionRecord,
  InitiativeWithRollup,
  Scenario,
} from "../../../types";
import type { OperatingModeCatalogEntry } from "../../../types/operating-mode";
import {
  activityOption,
  backlogOption,
  captureOption,
  executionOption,
  initiativeOption,
  operatingModeOption,
  operationsBriefingOption,
  scenarioOption,
  sessionOption,
  startupBriefOption,
  type SessionContextOption,
} from "./session-context-refs";

export function optionsToRefs(options: SessionContextOption[]): AgentSessionContextRef[] {
  return options.map(({ type, ref }) => ({ type, ref }));
}

/**
 * Raw store data the picker and the starter-card count badges both convert into
 * selectable options. Bundled so both call sites build from identical inputs.
 */
export interface ContextOptionInputs {
  backlogItems: BacklogItem[];
  initiatives: InitiativeWithRollup[];
  captures: Capture[];
  executions: ExecutionRecord[];
  activities: AgentActivity[];
  scenarios: Scenario[];
  modes: OperatingModeCatalogEntry[];
  sessions: AgentSession[];
  sessionKind: AgentSessionKind;
  currentSessionId?: string;
}

/**
 * Single source of truth for "what context the picker can show", keyed by type.
 *
 * Both {@link SessionContextPicker} (the dialog) and `useStarterContextCounts`
 * (the starter-card count badges) build their lists from this one function, so a
 * card's badge count equals the picker's selectable set by construction — there
 * is no second code path that could drift. Pure: same inputs → same output.
 */
export function buildContextOptionsByType(
  inputs: ContextOptionInputs,
): Record<AgentSessionContextType, SessionContextOption[]> {
  const {
    backlogItems,
    initiatives,
    captures,
    executions,
    activities,
    scenarios,
    modes,
    sessions,
    sessionKind,
    currentSessionId,
  } = inputs;
  return {
    backlog_item: backlogItems.map(backlogOption),
    initiative: initiatives.map(initiativeOption),
    capture: captures.map(captureOption),
    execution: executions.map(executionOption),
    agent_activity: activities.map(activityOption),
    scenario: scenarios.map(scenarioOption),
    operating_mode: modes.map(operatingModeOption).filter((mode) => mode.ref),
    session: sessions.filter((session) => session.id !== currentSessionId).map(sessionOption),
    operations_briefing: [operationsBriefingOption()],
    startup_brief: [startupBriefOption(sessionKind)],
    // Plan cycles/ETA are attached from the plan board, not browsed in the
    // generic context picker, so they contribute no pickable options here.
    plan_dependency_cycles: [],
    plan_eta: [],
  };
}

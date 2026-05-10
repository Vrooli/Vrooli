import type { MatrixRow } from "./matrix";
import type { Trace } from "./traces";

export interface WorkflowSpec {
  readonly id: string;
  readonly domain: string;
  readonly description: string;
  readonly states: readonly string[];
  readonly events: readonly string[];
  readonly initialState: string;
  readonly terminalStates: readonly string[];
  readonly transitions: readonly SpecTransition[];
  readonly invariants: readonly string[];
  readonly traces: readonly SpecTrace[];
  readonly formalModel?: {
    readonly status: string;
    readonly tool?: string;
    readonly model?: string;
    readonly generatedArtifacts?: string;
    readonly driftCheck?: string;
  };
}

export interface SpecTransition {
  readonly from: string;
  readonly event: string;
  readonly to: string;
  readonly wantError?: boolean;
}

export interface SpecTrace {
  readonly name: string;
  readonly initial: string;
  readonly steps: readonly SpecTraceStep[];
}

export interface SpecTraceStep {
  readonly event: string;
  readonly want: string;
  readonly wantError?: boolean;
}

const pairKey = (state: PropertyKey, event: PropertyKey) => `${String(state)}\u0000${String(event)}`;

export const validateWorkflowSpecConformance = <State extends PropertyKey, Event extends PropertyKey>(
  spec: WorkflowSpec,
  states: readonly State[],
  events: readonly Event[],
  rows: readonly MatrixRow<State, Event>[],
  traces: readonly Trace<State, Event>[],
): string[] => {
  const errors: string[] = [];

  if (spec.id.trim() === "") {
    errors.push("spec id is required");
  }
  if (spec.domain.trim() === "") {
    errors.push("spec domain is required");
  }
  if (spec.initialState.trim() === "") {
    errors.push("spec initialState is required");
  }
  if (spec.transitions.length === 0) {
    errors.push("spec transitions must not be empty");
  }

  const specStates = new Set(spec.states);
  states.forEach((state) => {
    if (!specStates.has(String(state))) {
      errors.push(`spec missing production state ${String(state)}`);
    }
  });
  spec.states.forEach((state) => {
    if (!states.map(String).includes(state)) {
      errors.push(`spec state ${state} is not a production state`);
    }
  });
  if (!specStates.has(spec.initialState)) {
    errors.push(`spec initialState ${spec.initialState} is not a known state`);
  }
  spec.terminalStates.forEach((state) => {
    if (!specStates.has(state)) {
      errors.push(`spec terminal state ${state} is not a known state`);
    }
  });

  const specEvents = new Set(spec.events);
  events.forEach((event) => {
    if (!specEvents.has(String(event))) {
      errors.push(`spec missing production event ${String(event)}`);
    }
  });
  spec.events.forEach((event) => {
    if (!events.map(String).includes(event)) {
      errors.push(`spec event ${event} is not a production event`);
    }
  });

  const matrixByPair = new Map<string, SpecTransition>();
  rows.forEach((row) => {
    matrixByPair.set(pairKey(row.from, row.event), {
      from: String(row.from),
      event: String(row.event),
      to: String(row.to),
      wantError: row.wantError,
    });
  });

  const specByPair = new Map<string, SpecTransition>();
  spec.transitions.forEach((transition) => {
    const key = pairKey(transition.from, transition.event);
    if (specByPair.has(key)) {
      errors.push(`spec duplicate transition ${transition.from}/${transition.event}`);
      return;
    }
    specByPair.set(key, transition);
    const row = matrixByPair.get(key);
    if (!row) {
      errors.push(`spec transition ${transition.from}/${transition.event} missing from matrix`);
      return;
    }
    const specWantError = transition.wantError ?? false;
    const rowWantError = row.wantError ?? false;
    if (transition.to !== row.to || specWantError !== rowWantError) {
      errors.push(
        `spec transition ${transition.from}/${transition.event} mismatch: spec to=${transition.to} wantError=${specWantError} matrix to=${row.to} wantError=${rowWantError}`,
      );
    }
  });

  matrixByPair.forEach((row, key) => {
    if (!specByPair.has(key)) {
      errors.push(`matrix transition ${row.from}/${row.event} missing from spec`);
    }
  });

  const traceByName = new Map<string, Trace<State, Event>>();
  traces.forEach((trace, index) => {
    traceByName.set(trace.name ?? `trace ${index}`, trace);
  });
  spec.traces.forEach((specTrace) => {
    const trace = traceByName.get(specTrace.name);
    if (!trace) {
      errors.push(`spec trace ${specTrace.name} missing from tests`);
      return;
    }
    if (specTrace.initial !== String(trace.initial)) {
      errors.push(`spec trace ${specTrace.name} initial=${specTrace.initial} test initial=${String(trace.initial)}`);
    }
    if (specTrace.steps.length !== trace.steps.length) {
      errors.push(`spec trace ${specTrace.name} step count=${specTrace.steps.length} test step count=${trace.steps.length}`);
      return;
    }
    specTrace.steps.forEach((specStep, index) => {
      const testStep = trace.steps[index];
      if (!testStep) {
        errors.push(`spec trace ${specTrace.name} step ${index} missing from test trace`);
        return;
      }
      if (
        specStep.event !== String(testStep.event) ||
        specStep.want !== String(testStep.want) ||
        (specStep.wantError ?? false) !== (testStep.wantError ?? false)
      ) {
        errors.push(`spec trace ${specTrace.name} step ${index} differs from test trace`);
      }
    });
  });

  return errors;
};

export const assertWorkflowSpecConformance = <State extends PropertyKey, Event extends PropertyKey>(
  spec: WorkflowSpec,
  states: readonly State[],
  events: readonly Event[],
  rows: readonly MatrixRow<State, Event>[],
  traces: readonly Trace<State, Event>[],
): void => {
  const errors = validateWorkflowSpecConformance(spec, states, events, rows, traces);
  if (errors.length > 0) {
    throw new Error(`workflow spec mismatch:\n${errors.map((err) => `  - ${err}`).join("\n")}`);
  }
};

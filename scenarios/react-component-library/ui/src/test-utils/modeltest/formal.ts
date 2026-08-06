import type { MatrixRow, WorkflowTransition } from "./matrix";
import { validateTransitionMatrix } from "./matrix";
import type { Trace } from "./traces";
import { validateTraces } from "./traces";

const formalArtifactSchemaVersion = 6;
const sha256Pattern = /^[a-f0-9]{64}$/;

export interface FormalArtifact {
  readonly schemaVersion: number;
  readonly flowId: string;
  readonly source: {
    readonly contractPath: string;
    readonly contractSha256: string;
    readonly generatorPath: string;
    readonly generatorSha256: string;
    readonly generatorVersion: number;
    readonly modelPath: string;
    readonly modelSha256: string;
    readonly quintVersion: string;
    readonly verificationBackend: string;
  };
  readonly commands: Record<string, readonly string[]>;
  readonly states: readonly string[];
  readonly events: readonly string[];
  readonly transitions: readonly FormalArtifactTransition[];
  readonly namedTraces: readonly FormalArtifactTrace[];
  readonly generatedTraces: readonly FormalArtifactTrace[];
  readonly invariants: readonly string[];
  readonly generatedChecks: readonly string[];
  readonly coverage: {
    readonly transitionMatrixComplete: boolean;
    readonly terminalTransitionsChecked: boolean;
    readonly namedTraces: FormalArtifactTraceCoverage;
    readonly generatedTraces: FormalArtifactTraceCoverage;
  };
  readonly checks: {
    readonly typechecked: boolean;
    readonly tested: boolean;
    readonly verified: boolean;
    readonly generatedFromContract: boolean;
    readonly generatedFromModel: boolean;
  };
}

export interface FormalArtifactTraceCoverage {
  readonly allStatesCovered: boolean;
  readonly allEventsCovered: boolean;
  readonly coveredStates: readonly string[];
  readonly coveredEvents: readonly string[];
  readonly coveredPairs?: readonly string[];
  readonly allPairsCovered?: boolean;
}

export interface FormalArtifactTransition {
  readonly from: string;
  readonly event: string;
  readonly to: string;
  readonly wantError: boolean;
}

export interface FormalArtifactTrace {
  readonly name: string;
  readonly initial: string;
  readonly steps: readonly FormalArtifactTraceStep[];
}

export interface FormalArtifactTraceStep {
  readonly event: string;
  readonly want: string;
  readonly wantError: boolean;
}

export interface FormalReplayAdapter<
  State extends PropertyKey,
  Event extends PropertyKey,
  RuntimeState,
  RuntimeEvent,
> {
  readonly states: readonly State[];
  readonly events: readonly Event[];
  readonly stateFor: Record<State, () => RuntimeState>;
  readonly eventFor: Record<Event, () => RuntimeEvent>;
  readonly statusOf: (state: RuntimeState) => State;
  readonly transition: (state: RuntimeState, event: RuntimeEvent) => RuntimeState;
}

export const transitionFromReplayAdapter =
  <State extends PropertyKey, Event extends PropertyKey, RuntimeState, RuntimeEvent>(
    adapter: FormalReplayAdapter<State, Event, RuntimeState, RuntimeEvent>,
  ): WorkflowTransition<State, Event> =>
  (state, event) =>
    adapter.statusOf(adapter.transition(adapter.stateFor[state](), adapter.eventFor[event]()));

export interface FormalArtifactFreshExpectation {
  readonly contractPath: string;
  readonly contractSha256?: string;
  readonly modelPath: string;
  readonly modelSha256?: string;
  readonly generatorPath?: string;
  readonly generatorSha256?: string;
  readonly invariants?: readonly string[];
  readonly generatedChecks?: readonly string[];
}

export const validateFormalArtifactFresh = (
  artifact: FormalArtifact,
  expected: FormalArtifactFreshExpectation,
): string[] => {
  const errors: string[] = [];
  if (artifact.schemaVersion !== formalArtifactSchemaVersion) {
    errors.push(
      `formal artifact schemaVersion=${artifact.schemaVersion}, want ${formalArtifactSchemaVersion}`,
    );
  }
  if (artifact.flowId.trim() === "") {
    errors.push("formal artifact flowId is required");
  }
  if (artifact.source.contractPath !== expected.contractPath) {
    errors.push(
      `formal artifact contractPath=${artifact.source.contractPath}, want ${expected.contractPath}`,
    );
  }
  requireSha256(errors, "contractSha256", artifact.source.contractSha256);
  if (expected.contractSha256 && artifact.source.contractSha256 !== expected.contractSha256) {
    errors.push(
      `formal artifact contractSha256=${artifact.source.contractSha256}, want ${expected.contractSha256}`,
    );
  }
  if (artifact.source.modelPath !== expected.modelPath) {
    errors.push(
      `formal artifact modelPath=${artifact.source.modelPath}, want ${expected.modelPath}`,
    );
  }
  if (expected.generatorPath && artifact.source.generatorPath !== expected.generatorPath) {
    errors.push(
      `formal artifact generatorPath=${artifact.source.generatorPath}, want ${expected.generatorPath}`,
    );
  }
  requireSha256(errors, "generatorSha256", artifact.source.generatorSha256);
  if (expected.generatorSha256 && artifact.source.generatorSha256 !== expected.generatorSha256) {
    errors.push(
      `formal artifact generatorSha256=${artifact.source.generatorSha256}, want ${expected.generatorSha256}`,
    );
  }
  if (!Number.isInteger(artifact.source.generatorVersion) || artifact.source.generatorVersion < 1) {
    errors.push("formal artifact generatorVersion is required");
  }
  if (artifact.source.verificationBackend.trim() === "") {
    errors.push("formal artifact verificationBackend is required");
  }
  if (artifact.source.quintVersion.trim() === "") {
    errors.push("formal artifact quintVersion is required");
  }
  requireSha256(errors, "modelSha256", artifact.source.modelSha256);
  if (expected.modelSha256 && artifact.source.modelSha256 !== expected.modelSha256) {
    errors.push(
      `formal artifact modelSha256=${artifact.source.modelSha256}, want ${expected.modelSha256}`,
    );
  }
  if (!artifact.checks.typechecked) {
    errors.push("formal artifact was not typechecked");
  }
  if (!artifact.checks.tested) {
    errors.push("formal artifact was not tested");
  }
  if (!artifact.checks.verified) {
    errors.push("formal artifact was not verified");
  }
  if (!artifact.checks.generatedFromContract) {
    errors.push("formal artifact was not generated from contract");
  }
  if (!artifact.checks.generatedFromModel) {
    errors.push("formal artifact was not generated from model");
  }
  for (const invariant of expected.invariants ?? []) {
    if (!artifact.invariants.includes(invariant)) {
      errors.push(`formal artifact missing invariant ${invariant}`);
    }
  }
  for (const check of expected.generatedChecks ?? []) {
    if (!artifact.generatedChecks.includes(check)) {
      errors.push(`formal artifact missing generated check ${check}`);
    }
  }
  if (!artifact.coverage.transitionMatrixComplete) {
    errors.push("formal artifact transition matrix is incomplete");
  }
  if (!artifact.coverage.terminalTransitionsChecked) {
    errors.push("formal artifact does not check terminal transitions");
  }
  if (!artifact.coverage.namedTraces.allStatesCovered) {
    errors.push("formal artifact named traces do not cover all states");
  }
  if (!artifact.coverage.namedTraces.allEventsCovered) {
    errors.push("formal artifact named traces do not cover all events");
  }
  if (artifact.coverage.generatedTraces.coveredStates.length === 0) {
    errors.push("formal artifact generated traces do not report covered states");
  }
  if (artifact.coverage.generatedTraces.coveredEvents.length === 0) {
    errors.push("formal artifact generated traces do not report covered events");
  }
  if (!artifact.coverage.generatedTraces.coveredPairs) {
    errors.push("formal artifact generated traces do not report covered pairs");
  }
  if (artifact.transitions.length === 0) {
    errors.push("formal artifact transitions must not be empty");
  }
  if (artifact.namedTraces.length === 0) {
    errors.push("formal artifact namedTraces must not be empty");
  }
  if (artifact.generatedTraces.length === 0) {
    errors.push("formal artifact generatedTraces must not be empty");
  }
  return errors;
};

const requireSha256 = (errors: string[], field: string, value: string): void => {
  if (!sha256Pattern.test(value)) {
    errors.push(`formal artifact ${field} is required`);
  }
};

export const assertFormalArtifactFresh = (
  artifact: FormalArtifact,
  expected: FormalArtifactFreshExpectation,
): void => {
  const errors = validateFormalArtifactFresh(artifact, expected);
  if (errors.length > 0) {
    throw new Error(`formal artifact is stale or incomplete:\n${formatErrors(errors)}`);
  }
};

export const validateFormalTransitionsReplay = <
  State extends PropertyKey,
  Event extends PropertyKey,
>(
  artifact: FormalArtifact,
  states: readonly State[],
  events: readonly Event[],
  transition: WorkflowTransition<State, Event>,
): string[] => {
  const { rows, errors } = formalRows(artifact, states, events);
  if (errors.length > 0) {
    return errors;
  }
  return validateTransitionMatrix(states, events, rows, transition);
};

export const assertFormalTransitionsReplay = <State extends PropertyKey, Event extends PropertyKey>(
  artifact: FormalArtifact,
  states: readonly State[],
  events: readonly Event[],
  transition: WorkflowTransition<State, Event>,
): void => {
  const errors = validateFormalTransitionsReplay(artifact, states, events, transition);
  if (errors.length > 0) {
    throw new Error(`formal transition replay mismatch:\n${formatErrors(errors)}`);
  }
};

export const validateFormalTracesReplay = <State extends PropertyKey, Event extends PropertyKey>(
  artifact: FormalArtifact,
  states: readonly State[],
  events: readonly Event[],
  transition: WorkflowTransition<State, Event>,
): string[] => {
  const { traces, errors } = formalTraces(
    [...artifact.namedTraces, ...artifact.generatedTraces],
    states,
    events,
  );
  if (errors.length > 0) {
    return errors;
  }
  return validateTraces(traces, transition);
};

export const assertFormalTracesReplay = <State extends PropertyKey, Event extends PropertyKey>(
  artifact: FormalArtifact,
  states: readonly State[],
  events: readonly Event[],
  transition: WorkflowTransition<State, Event>,
): void => {
  const errors = validateFormalTracesReplay(artifact, states, events, transition);
  if (errors.length > 0) {
    throw new Error(`formal trace replay mismatch:\n${formatErrors(errors)}`);
  }
};

const formalRows = <State extends PropertyKey, Event extends PropertyKey>(
  artifact: FormalArtifact,
  states: readonly State[],
  events: readonly Event[],
): { rows: MatrixRow<State, Event>[]; errors: string[] } => {
  const stateByName = valuesByString(states);
  const eventByName = valuesByString(events);
  const rows: MatrixRow<State, Event>[] = [];
  const errors: string[] = [];

  artifact.transitions.forEach((transition, index) => {
    const from = stateByName.get(transition.from);
    const to = stateByName.get(transition.to);
    const event = eventByName.get(transition.event);
    if (from === undefined) {
      errors.push(`formal transition ${index} unknown from state ${transition.from}`);
    }
    if (to === undefined) {
      errors.push(`formal transition ${index} unknown to state ${transition.to}`);
    }
    if (event === undefined) {
      errors.push(`formal transition ${index} unknown event ${transition.event}`);
    }
    if (from === undefined || to === undefined || event === undefined) {
      return;
    }
    rows.push({
      name: `formal transition ${transition.from}/${transition.event}`,
      from,
      event,
      to,
      wantError: transition.wantError,
    });
  });

  return { rows, errors };
};

const formalTraces = <State extends PropertyKey, Event extends PropertyKey>(
  artifactTraces: readonly FormalArtifactTrace[],
  states: readonly State[],
  events: readonly Event[],
): { traces: Trace<State, Event>[]; errors: string[] } => {
  const stateByName = valuesByString(states);
  const eventByName = valuesByString(events);
  const traces: Trace<State, Event>[] = [];
  const errors: string[] = [];

  artifactTraces.forEach((trace, traceIndex) => {
    const initial = stateByName.get(trace.initial);
    if (initial === undefined) {
      errors.push(`formal trace ${trace.name} unknown initial state ${trace.initial}`);
      return;
    }
    const steps = trace.steps.flatMap((step, stepIndex) => {
      const event = eventByName.get(step.event);
      const want = stateByName.get(step.want);
      if (event === undefined) {
        errors.push(`formal trace ${trace.name} step ${stepIndex} unknown event ${step.event}`);
      }
      if (want === undefined) {
        errors.push(`formal trace ${trace.name} step ${stepIndex} unknown want state ${step.want}`);
      }
      if (event === undefined || want === undefined) {
        return [];
      }
      return [{ event, want, wantError: step.wantError }];
    });
    traces.push({
      name: trace.name || `formal trace ${traceIndex}`,
      initial,
      steps,
    });
  });

  return { traces, errors };
};

const valuesByString = <Value extends PropertyKey>(values: readonly Value[]): Map<string, Value> =>
  new Map(values.map((value) => [String(value), value]));

const formatErrors = (errors: readonly string[]): string =>
  errors.map((error) => `  - ${error}`).join("\n");

import type { MatrixRow, WorkflowTransition } from "./matrix";
import { validateTransitionMatrix } from "./matrix";
import type { Trace } from "./traces";
import { validateTraces } from "./traces";

export interface FormalArtifact {
  readonly schemaVersion: number;
  readonly flowId: string;
  readonly source: {
    readonly modelPath: string;
    readonly modelSha256: string;
    readonly quintVersion: string;
  };
  readonly commands: Record<string, readonly string[]>;
  readonly states: readonly string[];
  readonly events: readonly string[];
  readonly transitions: readonly FormalArtifactTransition[];
  readonly traces: readonly FormalArtifactTrace[];
  readonly checks: {
    readonly typechecked: boolean;
    readonly tested: boolean;
    readonly verified: boolean;
    readonly generatedFromModel: boolean;
  };
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

export const validateFormalArtifactFresh = (
  artifact: FormalArtifact,
  expected: { readonly modelPath: string; readonly modelSha256?: string },
): string[] => {
  const errors: string[] = [];
  if (artifact.schemaVersion !== 1) {
    errors.push(`formal artifact schemaVersion=${artifact.schemaVersion}, want 1`);
  }
  if (artifact.flowId.trim() === "") {
    errors.push("formal artifact flowId is required");
  }
  if (artifact.source.modelPath !== expected.modelPath) {
    errors.push(`formal artifact modelPath=${artifact.source.modelPath}, want ${expected.modelPath}`);
  }
  if (artifact.source.quintVersion.trim() === "") {
    errors.push("formal artifact quintVersion is required");
  }
  if (!/^[a-f0-9]{64}$/.test(artifact.source.modelSha256)) {
    errors.push("formal artifact modelSha256 is required");
  }
  if (expected.modelSha256 && artifact.source.modelSha256 !== expected.modelSha256) {
    errors.push(`formal artifact modelSha256=${artifact.source.modelSha256}, want ${expected.modelSha256}`);
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
  if (!artifact.checks.generatedFromModel) {
    errors.push("formal artifact was not generated from model");
  }
  if (artifact.transitions.length === 0) {
    errors.push("formal artifact transitions must not be empty");
  }
  if (artifact.traces.length === 0) {
    errors.push("formal artifact traces must not be empty");
  }
  return errors;
};

export const assertFormalArtifactFresh = (
  artifact: FormalArtifact,
  expected: { readonly modelPath: string; readonly modelSha256?: string },
): void => {
  const errors = validateFormalArtifactFresh(artifact, expected);
  if (errors.length > 0) {
    throw new Error(`formal artifact is stale or incomplete:\n${formatErrors(errors)}`);
  }
};

export const validateFormalTransitionsReplay = <State extends PropertyKey, Event extends PropertyKey>(
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
  const { traces, errors } = formalTraces(artifact, states, events);
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
  artifact: FormalArtifact,
  states: readonly State[],
  events: readonly Event[],
): { traces: Trace<State, Event>[]; errors: string[] } => {
  const stateByName = valuesByString(states);
  const eventByName = valuesByString(events);
  const traces: Trace<State, Event>[] = [];
  const errors: string[] = [];

  artifact.traces.forEach((trace, traceIndex) => {
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

const formatErrors = (errors: readonly string[]): string => errors.map((error) => `  - ${error}`).join("\n");

import type { WorkflowTransition } from "./matrix";

export interface TraceStep<State extends PropertyKey, Event extends PropertyKey> {
  readonly name?: string;
  readonly event: Event;
  readonly want: State;
  readonly wantError?: boolean;
}

export interface Trace<State extends PropertyKey, Event extends PropertyKey> {
  readonly name?: string;
  readonly initial: State;
  readonly steps: readonly TraceStep<State, Event>[];
}

export const validateTraces = <State extends PropertyKey, Event extends PropertyKey>(
  traces: readonly Trace<State, Event>[],
  transition: WorkflowTransition<State, Event>,
): string[] => {
  const errors: string[] = [];

  traces.forEach((trace, traceIndex) => {
    const traceName = trace.name ?? `trace ${traceIndex}`;
    let state = trace.initial;

    trace.steps.forEach((step, stepIndex) => {
      const stepName = step.name ?? `step ${stepIndex}`;
      try {
        const got = transition(state, step.event);
        if (step.wantError) {
          errors.push(`${traceName}/${stepName}: expected error, got success`);
        }
        if (got !== step.want) {
          errors.push(`${traceName}/${stepName}: got state ${String(got)}, want ${String(step.want)}`);
        }
        state = got;
      } catch (error) {
        if (!step.wantError) {
          errors.push(`${traceName}/${stepName}: unexpected error: ${error instanceof Error ? error.message : String(error)}`);
        }
        if (state !== step.want) {
          errors.push(`${traceName}/${stepName}: got state ${String(state)}, want ${String(step.want)}`);
        }
      }
    });
  });

  return errors;
};

export const replayTraces = <State extends PropertyKey, Event extends PropertyKey>(
  traces: readonly Trace<State, Event>[],
  transition: WorkflowTransition<State, Event>,
): void => {
  const errors = validateTraces(traces, transition);
  if (errors.length > 0) {
    throw new Error(`trace replay mismatch:\n${errors.map((err) => `  - ${err}`).join("\n")}`);
  }
};

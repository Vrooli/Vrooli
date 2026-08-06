export type WorkflowTransition<State extends PropertyKey, Event extends PropertyKey> = (
  state: State,
  event: Event,
) => State;

export interface MatrixRow<State extends PropertyKey, Event extends PropertyKey> {
  readonly name?: string;
  readonly from: State;
  readonly event: Event;
  readonly to: State;
  readonly wantError?: boolean;
}

const pairKey = (state: PropertyKey, event: PropertyKey) =>
  `${String(state)}\u0000${String(event)}`;

export const validateTransitionMatrix = <State extends PropertyKey, Event extends PropertyKey>(
  states: readonly State[],
  events: readonly Event[],
  rows: readonly MatrixRow<State, Event>[],
  transition: WorkflowTransition<State, Event>,
): string[] => {
  const errors: string[] = [];
  if (states.length === 0) {
    errors.push("states must not be empty");
  }
  if (events.length === 0) {
    errors.push("events must not be empty");
  }

  const knownStates = new Set<State>();
  states.forEach((state) => {
    if (knownStates.has(state)) {
      errors.push(`duplicate state ${String(state)}`);
      return;
    }
    knownStates.add(state);
  });

  const knownEvents = new Set<Event>();
  events.forEach((event) => {
    if (knownEvents.has(event)) {
      errors.push(`duplicate event ${String(event)}`);
      return;
    }
    knownEvents.add(event);
  });

  const seen = new Map<string, string>();
  rows.forEach((row, index) => {
    const label = row.name ?? `row ${index}`;
    if (!knownStates.has(row.from)) {
      errors.push(`${label}: unknown from state ${String(row.from)}`);
    }
    if (!knownStates.has(row.to)) {
      errors.push(`${label}: unknown to state ${String(row.to)}`);
    }
    if (!knownEvents.has(row.event)) {
      errors.push(`${label}: unknown event ${String(row.event)}`);
    }

    const key = pairKey(row.from, row.event);
    const first = seen.get(key);
    if (first) {
      errors.push(
        `${label}: duplicate pair ${String(row.from)}/${String(row.event)} already covered by ${first}`,
      );
      return;
    }
    seen.set(key, label);
  });

  states.forEach((state) => {
    events.forEach((event) => {
      if (!seen.has(pairKey(state, event))) {
        errors.push(`missing pair ${String(state)}/${String(event)}`);
      }
    });
  });

  if (errors.length > 0) {
    return errors;
  }

  rows.forEach((row, index) => {
    const label = row.name ?? `row ${index}`;
    try {
      const got = transition(row.from, row.event);
      if (row.wantError) {
        errors.push(`${label}: expected error, got success`);
      }
      if (got !== row.to) {
        errors.push(`${label}: got state ${String(got)}, want ${String(row.to)}`);
      }
    } catch (error) {
      if (!row.wantError) {
        errors.push(
          `${label}: unexpected error: ${error instanceof Error ? error.message : String(error)}`,
        );
      }
      if (row.from !== row.to) {
        errors.push(
          `${label}: error rows must keep state unchanged; got ${String(row.from)}, want ${String(row.to)}`,
        );
      }
    }
  });

  return errors;
};

export const assertTransitionMatrix = <State extends PropertyKey, Event extends PropertyKey>(
  states: readonly State[],
  events: readonly Event[],
  rows: readonly MatrixRow<State, Event>[],
  transition: WorkflowTransition<State, Event>,
): void => {
  const errors = validateTransitionMatrix(states, events, rows, transition);
  if (errors.length > 0) {
    throw new Error(`transition matrix mismatch:\n${errors.map((err) => `  - ${err}`).join("\n")}`);
  }
};

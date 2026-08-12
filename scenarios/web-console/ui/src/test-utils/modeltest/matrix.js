const pairKey = (state, event) => `${String(state)}\u0000${String(event)}`;
export const validateTransitionMatrix = (states, events, rows, transition) => {
    const errors = [];
    if (states.length === 0) {
        errors.push("states must not be empty");
    }
    if (events.length === 0) {
        errors.push("events must not be empty");
    }
    const knownStates = new Set();
    states.forEach((state) => {
        if (knownStates.has(state)) {
            errors.push(`duplicate state ${String(state)}`);
            return;
        }
        knownStates.add(state);
    });
    const knownEvents = new Set();
    events.forEach((event) => {
        if (knownEvents.has(event)) {
            errors.push(`duplicate event ${String(event)}`);
            return;
        }
        knownEvents.add(event);
    });
    const seen = new Map();
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
            errors.push(`${label}: duplicate pair ${String(row.from)}/${String(row.event)} already covered by ${first}`);
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
        }
        catch (error) {
            if (!row.wantError) {
                errors.push(`${label}: unexpected error: ${error instanceof Error ? error.message : String(error)}`);
            }
            if (row.from !== row.to) {
                errors.push(`${label}: error rows must keep state unchanged; got ${String(row.from)}, want ${String(row.to)}`);
            }
        }
    });
    return errors;
};
export const assertTransitionMatrix = (states, events, rows, transition) => {
    const errors = validateTransitionMatrix(states, events, rows, transition);
    if (errors.length > 0) {
        throw new Error(`transition matrix mismatch:\n${errors.map((err) => `  - ${err}`).join("\n")}`);
    }
};

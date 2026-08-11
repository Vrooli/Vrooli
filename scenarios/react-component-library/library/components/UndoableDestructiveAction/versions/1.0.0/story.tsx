import { UndoableDestructiveAction } from "./UndoableDestructiveAction";

const noop = () => undefined;

export function Default() {
  return (
    <UndoableDestructiveAction
      itemLabel="Project brief"
      onDelete={noop}
      onUndo={noop}
    />
  );
}

export function Submitting() {
  return (
    <UndoableDestructiveAction
      itemLabel="Project brief"
      defaultState="submitting"
      onDelete={noop}
      onUndo={noop}
    />
  );
}

export function Success() {
  return (
    <UndoableDestructiveAction
      itemLabel="Project brief"
      defaultState="success"
      onDelete={noop}
      onUndo={noop}
    />
  );
}

export function RequestError() {
  return (
    <UndoableDestructiveAction
      itemLabel="Project brief"
      defaultState="error"
      errorMessage="The workspace service is temporarily unavailable. Try again without losing your place."
      onDelete={noop}
      onUndo={noop}
    />
  );
}

export function Interactive() {
  return (
    <UndoableDestructiveAction
      itemLabel="Project brief"
      onDelete={noop}
      onUndo={noop}
    />
  );
}

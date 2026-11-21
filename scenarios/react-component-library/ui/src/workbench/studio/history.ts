/**
 * A named command stack over immutable snapshots.
 *
 * The studio mutates three kinds of thing (a frame's spec, the design theme,
 * and the frame list) but all of them resolve to one value: the design
 * document. Storing before/after snapshots keeps undo exact, lets the history
 * panel jump to any point, and avoids a second mutation model beside the
 * existing TemplateNode spec helpers.
 */

export interface CommandEntry<T> {
  id: string;
  /** Human-readable action name shown in the history panel. */
  label: string;
  /** Consecutive commands with the same key collapse into one entry. */
  mergeKey?: string;
  at: number;
  before: T;
  after: T;
}

export interface CommandStack<T> {
  entries: CommandEntry<T>[];
  /** Number of entries currently applied; always within 0..entries.length. */
  index: number;
}

export interface PushInput<T> {
  label: string;
  before: T;
  after: T;
  mergeKey?: string;
}

/** Two keystrokes further apart than this start a new undo step. */
export const MERGE_WINDOW_MS = 1_000;

let sequence = 0;

export function emptyStack<T>(): CommandStack<T> {
  return { entries: [], index: 0 };
}

export function pushCommand<T>(
  stack: CommandStack<T>,
  input: PushInput<T>,
  now = Date.now(),
): CommandStack<T> {
  // A new command after undo discards the redo tail, exactly like an editor.
  const base = stack.index < stack.entries.length ? stack.entries.slice(0, stack.index) : stack.entries;
  const last = base[base.length - 1];

  if (
    last &&
    input.mergeKey &&
    last.mergeKey === input.mergeKey &&
    last.label === input.label &&
    now - last.at <= MERGE_WINDOW_MS
  ) {
    return {
      entries: [...base.slice(0, -1), { ...last, after: input.after, at: now }],
      index: base.length,
    };
  }

  sequence += 1;
  return {
    entries: [
      ...base,
      {
        id: `cmd-${now.toString(36)}-${sequence.toString(36)}`,
        label: input.label,
        mergeKey: input.mergeKey,
        at: now,
        before: input.before,
        after: input.after,
      },
    ],
    index: base.length + 1,
  };
}

export function canUndo<T>(stack: CommandStack<T>): boolean {
  return stack.index > 0;
}

export function canRedo<T>(stack: CommandStack<T>): boolean {
  return stack.index < stack.entries.length;
}

/** The state a stack position represents: 0 is before the first command. */
export function stateAt<T>(stack: CommandStack<T>, position: number): T | null {
  const clamped = Math.max(0, Math.min(stack.entries.length, position));
  const entry = clamped === 0 ? stack.entries[0] : stack.entries[clamped - 1];
  if (!entry) return null;
  return clamped === 0 ? entry.before : entry.after;
}

export function undoCommand<T>(
  stack: CommandStack<T>,
): { stack: CommandStack<T>; state: T } | null {
  if (!canUndo(stack)) return null;
  const position = stack.index - 1;
  const state = stateAt(stack, position);
  if (state === null) return null;
  return { stack: { ...stack, index: position }, state };
}

export function redoCommand<T>(
  stack: CommandStack<T>,
): { stack: CommandStack<T>; state: T } | null {
  if (!canRedo(stack)) return null;
  const state = stateAt(stack, stack.index + 1);
  if (state === null) return null;
  return { stack: { ...stack, index: stack.index + 1 }, state };
}

export function jumpCommand<T>(
  stack: CommandStack<T>,
  position: number,
): { stack: CommandStack<T>; state: T } | null {
  const clamped = Math.max(0, Math.min(stack.entries.length, Math.round(position)));
  if (clamped === stack.index) return null;
  const state = stateAt(stack, clamped);
  if (state === null) return null;
  return { stack: { ...stack, index: clamped }, state };
}

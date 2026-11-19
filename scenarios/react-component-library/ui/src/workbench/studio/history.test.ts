import { describe, expect, it } from "vitest";

import {
  canRedo,
  canUndo,
  emptyStack,
  jumpCommand,
  pushCommand,
  redoCommand,
  stateAt,
  undoCommand,
  type CommandStack,
} from "./history";

interface Doc {
  value: number;
}

const entry = (before: number, after: number, label = "Edit value", mergeKey?: string) => ({
  label,
  mergeKey,
  before: { value: before },
  after: { value: after },
});

describe("command stack", () => {
  it("undoes and redoes named entries", () => {
    let stack: CommandStack<Doc> = emptyStack();
    stack = pushCommand(stack, entry(0, 1), 1_000);
    stack = pushCommand(stack, entry(1, 2, "Edit other"), 2_000);
    expect(stack.entries.map((item) => item.label)).toEqual(["Edit value", "Edit other"]);

    const undone = undoCommand(stack);
    expect(undone?.state).toEqual({ value: 1 });
    expect(undone?.stack.index).toBe(1);

    const redone = redoCommand(undone?.stack ?? stack);
    expect(redone?.state).toEqual({ value: 2 });
    expect(redone?.stack.index).toBe(2);
    expect(canRedo(redone?.stack ?? stack)).toBe(false);
  });

  it("stops at the first snapshot", () => {
    let stack: CommandStack<Doc> = emptyStack();
    stack = pushCommand(stack, entry(0, 1), 1_000);
    const undone = undoCommand(stack);
    expect(undone?.state).toEqual({ value: 0 });
    expect(undone ? canUndo(undone.stack) : true).toBe(false);
    expect(undoCommand(undone?.stack ?? stack)).toBeNull();
  });

  it("merges consecutive commands that share a merge key inside the window", () => {
    let stack: CommandStack<Doc> = emptyStack();
    stack = pushCommand(stack, entry(0, 1, "Edit text", "text:node"), 1_000);
    stack = pushCommand(stack, entry(1, 2, "Edit text", "text:node"), 1_400);
    expect(stack.entries).toHaveLength(1);
    expect(stack.entries[0]?.after).toEqual({ value: 2 });
    expect(stack.entries[0]?.before).toEqual({ value: 0 });

    stack = pushCommand(stack, entry(2, 3, "Edit text", "text:node"), 4_000);
    expect(stack.entries).toHaveLength(2);
  });

  it("keeps separate steps for different keys or labels", () => {
    let stack: CommandStack<Doc> = emptyStack();
    stack = pushCommand(stack, entry(0, 1, "Edit text", "text:node"), 1_000);
    stack = pushCommand(stack, entry(1, 2, "Edit text", "text:other"), 1_100);
    stack = pushCommand(stack, entry(2, 3, "Move layer", "text:other"), 1_200);
    expect(stack.entries).toHaveLength(3);
  });

  it("discards the redo tail when a new command follows an undo", () => {
    let stack: CommandStack<Doc> = emptyStack();
    stack = pushCommand(stack, entry(0, 1), 1_000);
    stack = pushCommand(stack, entry(1, 2, "Second"), 2_000);
    const undone = undoCommand(stack);
    const branched = pushCommand(undone?.stack ?? stack, entry(1, 9, "Branch"), 3_000);
    expect(branched.entries).toHaveLength(2);
    expect(branched.entries.map((item) => item.label)).toEqual(["Edit value", "Branch"]);
    expect(canRedo(branched)).toBe(false);
    expect(stateAt(branched, 2)).toEqual({ value: 9 });
  });

  it("jumps to any point in history", () => {
    let stack: CommandStack<Doc> = emptyStack();
    stack = pushCommand(stack, entry(0, 1), 1_000);
    stack = pushCommand(stack, entry(1, 2, "Second"), 2_000);
    stack = pushCommand(stack, entry(2, 3, "Third"), 3_000);

    const start = jumpCommand(stack, 0);
    expect(start?.state).toEqual({ value: 0 });
    expect(start?.stack.index).toBe(0);

    const middle = jumpCommand(stack, 2);
    expect(middle?.state).toEqual({ value: 2 });
    expect(middle?.stack.index).toBe(2);
    expect(canRedo(middle?.stack ?? stack)).toBe(true);
    expect(jumpCommand(stack, 3)).toBeNull();
  });

  it("undoes fifty mixed operations back to the original document", () => {
    let stack: CommandStack<Doc> = emptyStack();
    let current = 0;
    for (let step = 0; step < 50; step += 1) {
      const next = current + 1;
      stack = pushCommand(
        stack,
        { label: `Step ${step}`, before: { value: current }, after: { value: next } },
        step * 1_000,
      );
      current = next;
    }
    expect(stack.entries).toHaveLength(50);
    for (let step = 0; step < 50; step += 1) {
      const undone = undoCommand(stack);
      if (!undone) throw new Error("undo exhausted early");
      stack = undone.stack;
    }
    expect(stateAt(stack, stack.index)).toEqual({ value: 0 });
    expect(canUndo(stack)).toBe(false);
    for (let step = 0; step < 50; step += 1) {
      const redone = redoCommand(stack);
      if (!redone) throw new Error("redo exhausted early");
      stack = redone.stack;
    }
    expect(stateAt(stack, stack.index)).toEqual({ value: 50 });
  });
});

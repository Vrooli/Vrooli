import { describe, expect, it } from "vitest";

import { historyShortcut } from "./shortcuts";

const base = { key: "z", meta: false, ctrl: false, shift: false, editable: false };

describe("studio history shortcuts", () => {
  it("undoes with Ctrl/Cmd+Z and redoes with Shift", () => {
    expect(historyShortcut({ ...base, ctrl: true })).toBe("undo");
    expect(historyShortcut({ ...base, meta: true })).toBe("undo");
    expect(historyShortcut({ ...base, meta: true, shift: true })).toBe("redo");
  });

  it("accepts Ctrl+Y as redo but not Cmd+Y", () => {
    expect(historyShortcut({ ...base, key: "y", ctrl: true })).toBe("redo");
    expect(historyShortcut({ ...base, key: "y", meta: true })).toBeNull();
  });

  it("leaves text fields and unrelated keys alone", () => {
    expect(historyShortcut({ ...base, ctrl: true, editable: true })).toBeNull();
    expect(historyShortcut({ ...base, ctrl: true, key: "a" })).toBeNull();
    expect(historyShortcut({ ...base, key: "z" })).toBeNull();
  });
});

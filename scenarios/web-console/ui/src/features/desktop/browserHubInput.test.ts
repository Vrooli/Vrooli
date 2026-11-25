import { describe, expect, it, vi } from "vitest";
import { createBASHubInputHandler, sendBASHubInput, toBASHubRecordingInput } from "./browserHubInput";

describe("BAS Hub input adapter", () => {
  it("normalizes text to the driver's atomic keyboard-text event", () => {
    expect(toBASHubRecordingInput("bas-session", { type: "text", text: "hello" })).toEqual({
      type: "recording_input",
      session_id: "bas-session",
      input: { type: "keyboard", text: "hello" },
    });
    expect(() => toBASHubRecordingInput("bas-session", { type: "text", text: "é".repeat(2049) })).toThrow(/4 KiB/);
  });

  it("sends keyboard presses once with modifiers and drops key-up events", () => {
    expect(toBASHubRecordingInput("bas-session", { type: "key", phase: "down", key: "c", modifiers: ["Control"] })).toEqual({
      type: "recording_input",
      session_id: "bas-session",
      input: { type: "keyboard", key: "c", modifiers: ["Control"] },
    });
    expect(toBASHubRecordingInput("bas-session", { type: "key", phase: "up", key: "c", modifiers: [] })).toBeUndefined();
  });

  it("preserves pointer modifiers and maps wheel deltas to Hub field names", () => {
    expect(toBASHubRecordingInput("bas-session", { type: "pointer", phase: "down", button: "left", x: 12, y: 34, modifiers: ["Shift"] })).toEqual({
      type: "recording_input",
      session_id: "bas-session",
      input: { type: "pointer", action: "down", button: "left", x: 12, y: 34, modifiers: ["Shift"] },
    });
    expect(toBASHubRecordingInput("bas-session", { type: "wheel", x: 12, y: 34, deltaX: 1, deltaY: -2 })).toEqual({
      type: "recording_input",
      session_id: "bas-session",
      input: { type: "wheel", x: 12, y: 34, delta_x: 1, delta_y: -2 },
    });
  });

  it("fails closed for an absent owner session identifier", () => {
    expect(toBASHubRecordingInput(" ", { type: "text", text: "hello" })).toBeUndefined();
  });

  it("sends only mapped input through an already-open owner-admitted socket", () => {
    const send = vi.fn();
    expect(sendBASHubInput({ readyState: 1, send }, "bas-session", { type: "text", text: "hello" })).toBe(true);
    expect(send).toHaveBeenCalledWith(JSON.stringify({
      type: "recording_input",
      session_id: "bas-session",
      input: { type: "keyboard", text: "hello" },
    }));
    expect(sendBASHubInput({ readyState: 0, send }, "bas-session", { type: "text", text: "hello" })).toBe(false);
    expect(sendBASHubInput({ readyState: 1, send }, "bas-session", { type: "key", phase: "up", key: "c", modifiers: [] })).toBe(false);
    expect(send).toHaveBeenCalledOnce();
  });

  it("adapts BrowserPane's ordered callback without putting receipts in Hub messages", async () => {
    const send = vi.fn();
    let readyState = 1;
    const handler = createBASHubInputHandler({ get readyState() { return readyState; }, send }, "bas-session");
    await handler({ type: "key", phase: "down", key: "c", modifiers: ["Control"] }, 3, { receipt: "opaque-ui-gate", epoch: 2n });
    expect(send).toHaveBeenCalledWith(JSON.stringify({
      type: "recording_input",
      session_id: "bas-session",
      input: { type: "keyboard", key: "c", modifiers: ["Control"] },
    }));
    expect(send.mock.calls[0]?.[0]).not.toContain("opaque-ui-gate");
    readyState = 3;
    await expect(handler({ type: "text", text: "x" }, 4, { receipt: "opaque-ui-gate", epoch: 2n })).rejects.toThrow(/owner-admitted BAS browser stream is not open/);
  });
});

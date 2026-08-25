import { describe, expect, it, vi } from "vitest";
import {
  createInputGate,
  type GateResult,
  type GateTransport,
  type InputIntent,
  terminalIsInMouseTrackingMode,
} from "../components/terminal/inputGate";

type FakeMouseMode = "none" | "x10" | "vt200" | "drag" | "any";

function fakeTerminal(mouseTrackingMode: FakeMouseMode = "none") {
  // Match the subset of Terminal.modes the gate reads.
  return { modes: { mouseTrackingMode } } as unknown as import("@xterm/xterm").Terminal;
}

function makeTransport(initial: { sent: boolean; seq?: number; reason?: "not-ready" | "ws-closed" | "paused" } = { sent: false, reason: "not-ready" }) {
  const sent: string[] = [];
  const queued: string[] = [];
  const transport: GateTransport = {
    send(data) {
      sent.push(data);
      return initial.sent
        ? { sent: true, seq: initial.seq ?? 1 }
        : { sent: false, reason: initial.reason };
    },
    enqueue(data) {
      queued.push(data);
    },
  };
  return { transport, sent, queued };
}

const everyIntent: Exclude<InputIntent, "control">[] = [
  "typing",
  "bulk_text",
  "named_key",
];

describe("TerminalInputGate", () => {
  it("rejects empty input regardless of source", () => {
    const { transport } = makeTransport({ sent: true, seq: 1 });
    const gate = createInputGate({ transport, getTerminal: () => fakeTerminal() });
    for (const intent of everyIntent) {
      const res = gate.submit("", intent);
      expect(res).toEqual({ status: "rejected", reason: "empty" });
    }
  });

  it("sends when transport accepts and session is ready", () => {
    const { transport, sent } = makeTransport({ sent: true, seq: 42 });
    const gate = createInputGate({ transport, getTerminal: () => fakeTerminal() });
    expect(gate.submit("hello", "typing")).toEqual({ status: "sent", seq: 42 });
    expect(sent).toEqual(["hello"]);
  });

  it("queues when transport reports not-ready, preserving reason", () => {
    const { transport, queued } = makeTransport({ sent: false, reason: "not-ready" });
    const gate = createInputGate({ transport, getTerminal: () => fakeTerminal() });
    const res = gate.submit("hello", "bulk_text");
    expect(res).toEqual({ status: "queued", reason: "not-ready" });
    expect(queued).toEqual(["hello"]);
  });

  it("queues when transport reports ws-closed", () => {
    const { transport } = makeTransport({ sent: false, reason: "ws-closed" });
    const gate = createInputGate({ transport, getTerminal: () => fakeTerminal() });
    expect(gate.submit("x", "typing")).toEqual({ status: "queued", reason: "ws-closed" });
  });

  it("queues bulk text when xterm is in mouse-tracking mode", () => {
    const { transport, sent, queued } = makeTransport({ sent: true, seq: 7 });
    const gate = createInputGate({
      transport,
      getTerminal: () => fakeTerminal("buttonEvent" as unknown as FakeMouseMode),
    });
    const res = gate.submit("pasted text", "bulk_text");
    expect(res).toEqual({ status: "queued", reason: "paused" });
    expect(sent).toEqual([]);
    expect(queued).toEqual(["pasted text"]);
  });

  it("sends typing and named keys even when xterm is in mouse-tracking mode", () => {
    const { transport, sent } = makeTransport({ sent: true, seq: 7 });
    const gate = createInputGate({
      transport,
      getTerminal: () => fakeTerminal("buttonEvent" as unknown as FakeMouseMode),
    });
    for (const intent of everyIntent) {
      if (intent === "bulk_text") continue;
      const res = gate.submit("k", intent);
      expect(res.status).toBe("sent");
    }
    expect(sent).toEqual(["k", "k"]);
  });

  it("queues every source when isPaused returns true", () => {
    const { transport, queued } = makeTransport({ sent: true, seq: 1 });
    const gate = createInputGate({
      transport,
      getTerminal: () => fakeTerminal(),
      isPaused: () => true,
    });
    for (const intent of everyIntent) {
      const res = gate.submit("p", intent);
      expect(res).toEqual({ status: "queued", reason: "paused" });
    }
    expect(queued.length).toBe(everyIntent.length);
  });

  it("rejects after dispose", () => {
    const { transport } = makeTransport({ sent: true, seq: 1 });
    const gate = createInputGate({ transport, getTerminal: () => fakeTerminal() });
    gate.dispose();
    for (const intent of everyIntent) {
      const res = gate.submit("x", intent);
      expect(res).toEqual({ status: "rejected", reason: "disposed" });
    }
  });

  it("canAcceptPaste reflects current terminal mode", () => {
    let mode: FakeMouseMode = "none";
    const { transport } = makeTransport({ sent: true, seq: 1 });
    const gate = createInputGate({
      transport,
      getTerminal: () => fakeTerminal(mode),
    });
    expect(gate.canAcceptPaste()).toBe(true);
    mode = "any";
    expect(gate.canAcceptPaste()).toBe(false);
  });

  it("handles a null terminal by allowing paste", () => {
    const { transport, sent } = makeTransport({ sent: true, seq: 1 });
    const gate = createInputGate({ transport, getTerminal: () => null });
    const res = gate.submit("paste", "bulk_text");
    expect(res.status).toBe("sent");
    expect(sent).toEqual(["paste"]);
  });

  it("terminalIsInMouseTrackingMode returns false for null or missing modes", () => {
    expect(terminalIsInMouseTrackingMode(null)).toBe(false);
    const t = {} as unknown as import("@xterm/xterm").Terminal;
    expect(terminalIsInMouseTrackingMode(t)).toBe(false);
  });

  it("does not call transport.send when paused", () => {
    const { transport, sent } = makeTransport({ sent: true, seq: 1 });
    const sendSpy = vi.spyOn(transport, "send");
    const gate = createInputGate({
      transport,
      getTerminal: () => fakeTerminal(),
      isPaused: () => true,
    });
    gate.submit("x", "typing");
    expect(sendSpy).not.toHaveBeenCalled();
    expect(sent).toEqual([]);
  });

  it("returns sent with the seq supplied by transport", () => {
    const { transport } = makeTransport({ sent: true, seq: 999 });
    const gate = createInputGate({ transport, getTerminal: () => fakeTerminal() });
    const res = gate.submit("x", "typing") as Extract<GateResult, { status: "sent" }>;
    expect(res.status).toBe("sent");
    expect(res.seq).toBe(999);
  });
});

import { describe, expect, it, vi } from "vitest";
import {
  createInputGate,
  type GateResult,
  type GateTransport,
  type InputIntent,
} from "../components/terminal/inputGate";

function makeTransport(initial: { sent: boolean; offset?: number; reason?: "not-ready" | "ws-closed" | "paused" } = { sent: false, reason: "not-ready" }) {
  const sent: string[] = [];
  const queued: string[] = [];
  const transport: GateTransport = {
    send(data) {
      sent.push(data);
      return initial.sent
        ? { sent: true, offset: initial.offset ?? 1 }
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
    const { transport } = makeTransport({ sent: true, offset: 1 });
    const gate = createInputGate({ transport });
    for (const intent of everyIntent) {
      const res = gate.submit("", intent);
      expect(res).toEqual({ status: "rejected", reason: "empty" });
    }
  });

  it("sends when transport accepts and session is ready", () => {
    const { transport, sent } = makeTransport({ sent: true, offset: 42 });
    const gate = createInputGate({ transport });
    expect(gate.submit("hello", "typing")).toEqual({ status: "sent", offset: 42 });
    expect(sent).toEqual(["hello"]);
  });

  it("queues when transport reports not-ready, preserving reason", () => {
    const { transport, queued } = makeTransport({ sent: false, reason: "not-ready" });
    const gate = createInputGate({ transport });
    const res = gate.submit("hello", "bulk_text");
    expect(res).toEqual({ status: "queued", reason: "not-ready" });
    expect(queued).toEqual(["hello"]);
  });

  it("queues when transport reports ws-closed", () => {
    const { transport } = makeTransport({ sent: false, reason: "ws-closed" });
    const gate = createInputGate({ transport });
    expect(gate.submit("x", "typing")).toEqual({ status: "queued", reason: "ws-closed" });
  });

  it("sends bulk text through the same reliable input gate", () => {
    const { transport, sent, queued } = makeTransport({ sent: true, offset: 7 });
    const gate = createInputGate({ transport });
    const res = gate.submit("pasted text", "bulk_text");
    expect(res).toEqual({ status: "sent", offset: 7 });
    expect(sent).toEqual(["pasted text"]);
    expect(queued).toEqual([]);
  });

  it("sends typing and named keys through the same reliable input gate", () => {
    const { transport, sent } = makeTransport({ sent: true, offset: 7 });
    const gate = createInputGate({ transport });
    for (const intent of ["typing", "named_key"] as const) {
      const res = gate.submit("k", intent);
      expect(res.status).toBe("sent");
    }
    expect(sent).toEqual(["k", "k"]);
  });

  it("queues every source when isPaused returns true", () => {
    const { transport, queued } = makeTransport({ sent: true, offset: 1 });
    const gate = createInputGate({
      transport,
      isPaused: () => true,
    });
    for (const intent of everyIntent) {
      const res = gate.submit("p", intent);
      expect(res).toEqual({ status: "queued", reason: "paused" });
    }
    expect(queued.length).toBe(everyIntent.length);
  });

  it("rejects after dispose", () => {
    const { transport } = makeTransport({ sent: true, offset: 1 });
    const gate = createInputGate({ transport });
    gate.dispose();
    for (const intent of everyIntent) {
      const res = gate.submit("x", intent);
      expect(res).toEqual({ status: "rejected", reason: "disposed" });
    }
  });

  it("handles bulk text as reliable input", () => {
    const { transport, sent } = makeTransport({ sent: true, offset: 1 });
    const gate = createInputGate({ transport });
    const res = gate.submit("paste", "bulk_text");
    expect(res.status).toBe("sent");
    expect(sent).toEqual(["paste"]);
  });

  it("does not call transport.send when paused", () => {
    const { transport, sent } = makeTransport({ sent: true, offset: 1 });
    const sendSpy = vi.spyOn(transport, "send");
    const gate = createInputGate({
      transport,
      isPaused: () => true,
    });
    gate.submit("x", "typing");
    expect(sendSpy).not.toHaveBeenCalled();
    expect(sent).toEqual([]);
  });

  it("returns sent with the offset supplied by transport", () => {
    const { transport } = makeTransport({ sent: true, offset: 999 });
    const gate = createInputGate({ transport });
    const res = gate.submit("x", "typing") as Extract<GateResult, { status: "sent" }>;
    expect(res.status).toBe("sent");
    expect(res.offset).toBe(999);
  });
});

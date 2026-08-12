import { describe, expect, it, vi } from "vitest";
import { createInputGate, terminalIsInMouseTrackingMode, } from "../components/terminal/inputGate";
function fakeTerminal(mouseTrackingMode = "none") {
    // Match the subset of Terminal.modes the gate reads.
    return { modes: { mouseTrackingMode } };
}
function makeTransport(initial = { sent: false, reason: "not-ready" }) {
    const sent = [];
    const queued = [];
    const transport = {
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
const everySource = [
    "xterm",
    "toolbar-key",
    "toolbar-submit",
    "paste",
    "voice",
    "upload",
];
describe("TerminalInputGate", () => {
    it("rejects empty input regardless of source", () => {
        const { transport } = makeTransport({ sent: true, seq: 1 });
        const gate = createInputGate({ transport, getTerminal: () => fakeTerminal() });
        for (const src of everySource) {
            const res = gate.submit("", src);
            expect(res).toEqual({ status: "rejected", reason: "empty" });
        }
    });
    it("sends when transport accepts and session is ready", () => {
        const { transport, sent } = makeTransport({ sent: true, seq: 42 });
        const gate = createInputGate({ transport, getTerminal: () => fakeTerminal() });
        expect(gate.submit("hello", "xterm")).toEqual({ status: "sent", seq: 42 });
        expect(sent).toEqual(["hello"]);
    });
    it("queues when transport reports not-ready, preserving reason", () => {
        const { transport, queued } = makeTransport({ sent: false, reason: "not-ready" });
        const gate = createInputGate({ transport, getTerminal: () => fakeTerminal() });
        const res = gate.submit("hello", "toolbar-submit");
        expect(res).toEqual({ status: "queued", reason: "not-ready" });
        expect(queued).toEqual(["hello"]);
    });
    it("queues when transport reports ws-closed", () => {
        const { transport } = makeTransport({ sent: false, reason: "ws-closed" });
        const gate = createInputGate({ transport, getTerminal: () => fakeTerminal() });
        expect(gate.submit("x", "xterm")).toEqual({ status: "queued", reason: "ws-closed" });
    });
    it("queues paste payloads when xterm is in mouse-tracking mode", () => {
        const { transport, sent, queued } = makeTransport({ sent: true, seq: 7 });
        const gate = createInputGate({
            transport,
            getTerminal: () => fakeTerminal("buttonEvent"),
        });
        const res = gate.submit("pasted text", "paste");
        expect(res).toEqual({ status: "queued", reason: "paused" });
        expect(sent).toEqual([]);
        expect(queued).toEqual(["pasted text"]);
    });
    it("sends non-paste payloads even when xterm is in mouse-tracking mode", () => {
        const { transport, sent } = makeTransport({ sent: true, seq: 7 });
        const gate = createInputGate({
            transport,
            getTerminal: () => fakeTerminal("buttonEvent"),
        });
        for (const src of everySource) {
            if (src === "paste")
                continue;
            const res = gate.submit("k", src);
            expect(res.status).toBe("sent");
        }
        expect(sent).toEqual(["k", "k", "k", "k", "k"]);
    });
    it("queues every source when isPaused returns true", () => {
        const { transport, queued } = makeTransport({ sent: true, seq: 1 });
        const gate = createInputGate({
            transport,
            getTerminal: () => fakeTerminal(),
            isPaused: () => true,
        });
        for (const src of everySource) {
            const res = gate.submit("p", src);
            expect(res).toEqual({ status: "queued", reason: "paused" });
        }
        expect(queued.length).toBe(everySource.length);
    });
    it("rejects after dispose", () => {
        const { transport } = makeTransport({ sent: true, seq: 1 });
        const gate = createInputGate({ transport, getTerminal: () => fakeTerminal() });
        gate.dispose();
        for (const src of everySource) {
            const res = gate.submit("x", src);
            expect(res).toEqual({ status: "rejected", reason: "disposed" });
        }
    });
    it("canAcceptPaste reflects current terminal mode", () => {
        let mode = "none";
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
        const res = gate.submit("paste", "paste");
        expect(res.status).toBe("sent");
        expect(sent).toEqual(["paste"]);
    });
    it("terminalIsInMouseTrackingMode returns false for null or missing modes", () => {
        expect(terminalIsInMouseTrackingMode(null)).toBe(false);
        const t = {};
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
        gate.submit("x", "xterm");
        expect(sendSpy).not.toHaveBeenCalled();
        expect(sent).toEqual([]);
    });
    it("returns sent with the seq supplied by transport", () => {
        const { transport } = makeTransport({ sent: true, seq: 999 });
        const gate = createInputGate({ transport, getTerminal: () => fakeTerminal() });
        const res = gate.submit("x", "xterm");
        expect(res.status).toBe("sent");
        expect(res.seq).toBe(999);
    });
});

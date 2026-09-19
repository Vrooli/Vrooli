// [REQ:REQ-PS-001] SSE consumer seam — mockEventSource enables behavioral tests of subscribeSSE
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { mockEventSource } from "./mockEventSource";
import type { MockEventSourceHandle } from "./mockEventSource";

describe("mockEventSource helper", () => {
    let handle: MockEventSourceHandle;

    beforeEach(() => {
        handle = mockEventSource();
    });

    afterEach(() => {
        handle.restore();
    });

    it("records each constructed EventSource with its URL", () => {
        new EventSource("http://a/api/v1/events/subscribe");
        new EventSource("http://a/api/v1/policies/subscribe");

        expect(handle.instances).toHaveLength(2);
        expect(handle.instances[0]!.url).toBe("http://a/api/v1/events/subscribe");
        expect(handle.instances[1]!.url).toBe("http://a/api/v1/policies/subscribe");
    });

    it("drives onmessage via emitMessage", () => {
        const es = new EventSource("http://a/stream");
        const handler = vi.fn();
        (es as unknown as { onmessage: (e: MessageEvent) => void }).onmessage = handler;

        handle.instances[0]!.emitMessage({ eventId: "evt-1" });

        expect(handler).toHaveBeenCalledTimes(1);
        const received = handler.mock.calls[0]![0] as MessageEvent;
        expect(JSON.parse(received.data)).toEqual({ eventId: "evt-1" });
    });

    it("drives addEventListener('message') subscribers via emitMessage", () => {
        const es = new EventSource("http://a/stream");
        const handler = vi.fn();
        es.addEventListener("message", handler);

        handle.instances[0]!.emitMessage({ eventId: "evt-2" });

        expect(handler).toHaveBeenCalledTimes(1);
    });

    it("delivers named events only to matching listeners", () => {
        const es = new EventSource("http://a/stream");
        const messageHandler = vi.fn();
        const policyHandler = vi.fn();
        es.addEventListener("message", messageHandler);
        es.addEventListener("policy_update", policyHandler);

        handle.instances[0]!.emitNamed("policy_update", { version: 7 });

        expect(policyHandler).toHaveBeenCalledTimes(1);
        expect(messageHandler).not.toHaveBeenCalled();
    });

    it("drives onerror via emitError", () => {
        const es = new EventSource("http://a/stream");
        const onError = vi.fn();
        (es as unknown as { onerror: (e: Event) => void }).onerror = onError;

        const inst = handle.instances[0];
        if (!inst) throw new Error("expected one instance");
        inst.emitError();

        expect(onError).toHaveBeenCalledTimes(1);
    });

    it("removeEventListener prevents subsequent deliveries", () => {
        const es = new EventSource("http://a/stream");
        const handler = vi.fn();
        es.addEventListener("message", handler);
        es.removeEventListener("message", handler);

        const inst = handle.instances[0];
        if (!inst) throw new Error("expected one instance");
        inst.emitMessage({ x: 1 });

        expect(handler).not.toHaveBeenCalled();
    });

    it("tracks close() calls and marks instance closed", () => {
        const es = new EventSource("http://a/stream");
        const inst = handle.instances[0];
        if (!inst) throw new Error("expected one instance");
        expect(inst.closed).toBe(false);

        es.close();

        expect(inst.close).toHaveBeenCalledTimes(1);
        expect(inst.closed).toBe(true);
    });

    it("exposes the static enum constants on the constructor", () => {
        expect((EventSource as unknown as { CONNECTING: number }).CONNECTING).toBe(0);
        expect((EventSource as unknown as { OPEN: number }).OPEN).toBe(1);
        expect((EventSource as unknown as { CLOSED: number }).CLOSED).toBe(2);
    });

    it("restore() returns globalThis.EventSource to the previous implementation", () => {
        const mocked = globalThis.EventSource;
        handle.restore();
        expect(globalThis.EventSource).not.toBe(mocked);
    });
});

// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
// DOC: docs/internal/SEAMS.md — globalThis.EventSource is the SSE consumer seam
import { vi, type Mock } from "vitest";

/**
 * One observed EventSource instance, exposing methods to drive its listener
 * callbacks from a test (simulate the server pushing a message, named event,
 * or error).
 */
export interface MockEventSourceInstance {
    /** The URL the consumer opened. */
    readonly url: string;
    /** All listeners registered via `addEventListener`, grouped by event type. */
    readonly listeners: Map<string, Array<(e: MessageEvent | Event) => void>>;
    /** Mock for `addEventListener` (use for `.toHaveBeenCalledWith` assertions). */
    readonly addEventListener: Mock;
    /** Mock for `removeEventListener`. */
    readonly removeEventListener: Mock;
    /** Mock for `close`. Test helpers can assert it was called on cleanup. */
    readonly close: Mock;
    /** Whether the consumer called `.close()`. */
    readonly closed: boolean;

    /**
     * Simulate the server emitting a default (unnamed) message event.
     * Triggers both `onmessage` (if assigned) and any `addEventListener("message", ...)` listeners.
     */
    emitMessage(data: unknown): void;
    /**
     * Simulate the server emitting a named SSE event (e.g. `event: policy_update`).
     * Triggers any `addEventListener(name, ...)` listeners.
     */
    emitNamed(name: string, data: unknown): void;
    /** Simulate a connection error. Triggers `onerror` if assigned. */
    emitError(): void;
}

/**
 * Controllable handle returned by `mockEventSource`. Tests use this to:
 *   - inspect what URLs were opened (`.instances`)
 *   - drive event delivery on each instance
 *   - restore the original `globalThis.EventSource`
 */
export interface MockEventSourceHandle {
    /** All EventSource instances opened since this handle was created, in order. */
    readonly instances: MockEventSourceInstance[];
    /** Restore the original `globalThis.EventSource`. */
    restore(): void;
}

interface MutableMockInstance extends MockEventSourceInstance {
    onmessage: ((e: MessageEvent) => void) | null;
    onerror: ((e: Event) => void) | null;
    closed: boolean;
}

/**
 * Replace `globalThis.EventSource` with a programmable mock that records each
 * constructed instance and exposes methods to simulate server-side events.
 *
 * The default `setup.ts` mock prevents jsdom errors but cannot drive listeners.
 * This helper is what tests use to actually exercise SSE-consuming code paths
 * (subscribeSSE, components that watch for events, reconnection logic, etc.).
 *
 * Example:
 * ```ts
 * const sse = mockEventSource();
 * subscribeSSE({ onEvent: handler });
 * sse.instances[0].emitMessage({ eventId: "evt-1", ... });
 * expect(handler).toHaveBeenCalledWith(expect.objectContaining({ eventId: "evt-1" }));
 * sse.restore();
 * ```
 */
export function mockEventSource(): MockEventSourceHandle {
    const instances: MutableMockInstance[] = [];
    const originalDescriptor = Object.getOwnPropertyDescriptor(globalThis, "EventSource");

    // Use a regular (non-arrow) function so `new EventSource(...)` works — arrow
    // functions have no [[Construct]] internal method and fail with `new`.
    function Ctor(this: unknown, url: string | URL) {
        const inst = createInstance(typeof url === "string" ? url : url.toString());
        instances.push(inst);
        return inst;
    }
    // Preserve the static enum constants the EventSource interface declares.
    Object.assign(Ctor, { CONNECTING: 0, OPEN: 1, CLOSED: 2 });

    Object.defineProperty(globalThis, "EventSource", { value: Ctor, writable: true, configurable: true });

    return {
        instances,
        restore() {
            if (originalDescriptor) {
                Object.defineProperty(globalThis, "EventSource", originalDescriptor);
            } else {
                delete (globalThis as Record<string, unknown>).EventSource;
            }
        },
    };
}

function createInstance(url: string): MutableMockInstance {
    const listeners = new Map<string, Array<(e: MessageEvent | Event) => void>>();
    const addEventListener = vi.fn((name: string, cb: (e: MessageEvent | Event) => void) => {
        const list = listeners.get(name) ?? [];
        list.push(cb);
        listeners.set(name, list);
    });
    const removeEventListener = vi.fn((name: string, cb: (e: MessageEvent | Event) => void) => {
        const list = listeners.get(name);
        if (!list) return;
        const idx = list.indexOf(cb);
        if (idx >= 0) list.splice(idx, 1);
    });

    const inst = {
        url,
        listeners,
        addEventListener,
        removeEventListener,
        close: vi.fn(),
        closed: false,
        onmessage: null,
        onerror: null,

        emitMessage(data: unknown) {
            const event = makeMessageEvent(data);
            inst.onmessage?.(event);
            for (const cb of listeners.get("message") ?? []) cb(event);
        },
        emitNamed(name: string, data: unknown) {
            const event = makeMessageEvent(data);
            for (const cb of listeners.get(name) ?? []) cb(event);
        },
        emitError() {
            const event = new Event("error");
            inst.onerror?.(event);
            for (const cb of listeners.get("error") ?? []) cb(event);
        },
    } as MutableMockInstance;

    inst.close.mockImplementation(() => {
        inst.closed = true;
    });

    return inst;
}

function makeMessageEvent(data: unknown): MessageEvent {
    const payload = typeof data === "string" ? data : JSON.stringify(data);
    return new MessageEvent("message", { data: payload });
}

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useGlobalEventStream } from "../hooks/useGlobalEventStream";
import { useConversationStore } from "../stores/useConversationStore";
import { useLiveStreamStore } from "../stores/useLiveStreamStore";

const { mockRefresh } = vi.hoisted(() => ({ mockRefresh: vi.fn().mockResolvedValue({ ok: true, addedEvents: 0 }) }));
vi.mock("../hooks/useConversationSession", () => ({
  refreshConversationSession: mockRefresh,
}));

/**
 * EventSource retries on its own only while it is CONNECTING. Once it reaches
 * CLOSED it has given up for good, and nothing used to reopen it — live updates
 * stayed dead until the page was reloaded, which is exactly the "I have to
 * leave the view and come back" symptom that started this work.
 */
const CONNECTING = 0;
const CLOSED = 2;

class FakeEventSource {
  static instances: FakeEventSource[] = [];
  listeners: Record<string, ((e: MessageEvent) => void)[]> = {};
  closed = false;
  readyState = CONNECTING;
  onerror: ((event: Event) => void) | null = null;
  onopen: ((event: Event) => void) | null = null;

  constructor(public url: string) {
    FakeEventSource.instances.push(this);
  }
  addEventListener(type: string, cb: EventListener) {
    (this.listeners[type] ||= []).push(cb as (e: MessageEvent) => void);
  }
  removeEventListener(type: string, cb: EventListener) {
    this.listeners[type] = (this.listeners[type] || []).filter((f) => f !== cb);
  }
  close() { this.closed = true; }

  open() {
    this.readyState = 1;
    this.onopen?.(new Event("open"));
  }
  fail(state: number) {
    this.readyState = state;
    this.onerror?.(new Event("error"));
  }
}

beforeEach(() => {
  FakeEventSource.instances = [];
  mockRefresh.mockClear();
  useConversationStore.setState({ sessions: {}, viewModes: {} });
  useLiveStreamStore.setState({ status: "connecting", connectedGeneration: 0 });
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

const createEventSource = (url: string) => new FakeEventSource(url) as unknown as EventSource;

describe("live stream reconnection", () => {
  it("reports open and resyncs tracked sessions once connected", () => {
    useConversationStore.setState({ sessions: { s1: { events: [], cursor: { lastSeenSequence: 0, lastListenedSequence: 0 }, hydrated: true, capture: { state: "unknown", reasonCode: "", summary: "", detail: "", remediation: "", transcriptPath: "", lastCapturedAt: "" }, status: "loaded" } }, viewModes: {} });
    renderHook(() => { useGlobalEventStream({ createEventSource }); });

    act(() => { FakeEventSource.instances[0]?.open(); });

    expect(useLiveStreamStore.getState().status).toBe("open");
    // A reconnect may have missed events entirely, so every tracked session is
    // refetched; the fetch is gap-aware and cheap when nothing was missed.
    expect(mockRefresh).toHaveBeenCalledWith("s1");
  });

  it("does not stack a second connection while the browser is still retrying", () => {
    renderHook(() => { useGlobalEventStream({ createEventSource }); });

    act(() => { FakeEventSource.instances[0]?.fail(CONNECTING); });
    act(() => { vi.advanceTimersByTime(60_000); });

    expect(useLiveStreamStore.getState().status).toBe("reconnecting");
    expect(FakeEventSource.instances).toHaveLength(1);
  });

  it("reopens the stream itself once the browser gives up", () => {
    renderHook(() => { useGlobalEventStream({ createEventSource }); });

    act(() => { FakeEventSource.instances[0]?.fail(CLOSED); });
    expect(useLiveStreamStore.getState().status).toBe("reconnecting");
    expect(FakeEventSource.instances).toHaveLength(1);

    act(() => { vi.advanceTimersByTime(1000); });
    expect(FakeEventSource.instances).toHaveLength(2);

    act(() => { FakeEventSource.instances[1]?.open(); });
    expect(useLiveStreamStore.getState().status).toBe("open");
  });

  it("backs off between attempts instead of hammering the server", () => {
    renderHook(() => { useGlobalEventStream({ createEventSource }); });

    act(() => { FakeEventSource.instances[0]?.fail(CLOSED); });
    act(() => { vi.advanceTimersByTime(1000); });
    expect(FakeEventSource.instances).toHaveLength(2);

    act(() => { FakeEventSource.instances[1]?.fail(CLOSED); });
    // The second wait is longer than the first: at 1s nothing new opens yet.
    act(() => { vi.advanceTimersByTime(1000); });
    expect(FakeEventSource.instances).toHaveLength(2);
    act(() => { vi.advanceTimersByTime(1000); });
    expect(FakeEventSource.instances).toHaveLength(3);
  });

  it("retries immediately when the tab comes back to the foreground", () => {
    renderHook(() => { useGlobalEventStream({ createEventSource }); });
    act(() => { FakeEventSource.instances[0]?.fail(CLOSED); });

    // Mobile browsers suspend background connections and do not always resume
    // them; foregrounding must not wait out the backoff.
    act(() => {
      Object.defineProperty(document, "hidden", { configurable: true, value: false });
      document.dispatchEvent(new Event("visibilitychange"));
    });

    expect(FakeEventSource.instances).toHaveLength(2);
  });

  it("closes the stream and stops retrying on unmount", () => {
    const { unmount } = renderHook(() => { useGlobalEventStream({ createEventSource }); });
    act(() => { FakeEventSource.instances[0]?.fail(CLOSED); });

    unmount();
    act(() => { vi.advanceTimersByTime(60_000); });

    expect(FakeEventSource.instances).toHaveLength(1);
    expect(FakeEventSource.instances[0]?.closed).toBe(true);
    expect(useLiveStreamStore.getState().status).toBe("closed");
  });
});

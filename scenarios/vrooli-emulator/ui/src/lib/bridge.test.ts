import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import type { SessionEvent } from "./bridge";

const samplePayload = {
  sessionId: "session-1",
  status: "connected" as const,
  createdAt: "2026-01-01T00:00:00Z",
  backend: "linux",
  resolution: { width: 1280, height: 720 },
};

describe("postSessionEvent", () => {
  const originalParent = window.parent;
  let postMessage: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    postMessage = vi.fn();
    const fakeParent = { postMessage } as unknown as Window;
    Object.defineProperty(window, "parent", { value: fakeParent, configurable: true });
  });

  afterEach(() => {
    Object.defineProperty(window, "parent", { value: originalParent, configurable: true });
    vi.resetModules();
  });

  it("returns false and does not post when parent === window", async () => {
    Object.defineProperty(window, "parent", { value: window, configurable: true });
    const { postSessionEvent } = await import("./bridge");
    const event: SessionEvent = { type: "session.created", payload: samplePayload };
    expect(postSessionEvent(event)).toBe(false);
    expect(postMessage).not.toHaveBeenCalled();
  });

  it("wraps the event in the versioned envelope and posts to parent", async () => {
    const { postSessionEvent } = await import("./bridge");
    const event: SessionEvent = { type: "session.created", payload: samplePayload };
    expect(postSessionEvent(event)).toBe(true);
    expect(postMessage).toHaveBeenCalledTimes(1);
    const [message] = postMessage.mock.calls[0]!;
    expect(message).toEqual({ v: 1, t: "SESSION", event });
  });

  it("posts session.state_changed events with the documented payload shape", async () => {
    const { postSessionEvent } = await import("./bridge");
    const payload = { ...samplePayload, status: "connecting" as const };
    const event: SessionEvent = { type: "session.state_changed", payload };
    postSessionEvent(event);
    const [message] = postMessage.mock.calls[0]!;
    expect((message as { event: SessionEvent }).event.type).toBe("session.state_changed");
    expect((message as { event: SessionEvent }).event.payload).toEqual(payload);
    const keys = Object.keys((message as { event: SessionEvent }).event.payload).sort();
    expect(keys).toEqual(["backend", "createdAt", "resolution", "sessionId", "status"]);
  });

  it("posts session.error events carrying an error object", async () => {
    const { postSessionEvent } = await import("./bridge");
    const payload = {
      ...samplePayload,
      status: "failed" as const,
      error: { code: "E_BOOM", message: "boom" },
    };
    const event: SessionEvent = { type: "session.error", payload };
    postSessionEvent(event);
    const [message] = postMessage.mock.calls[0]!;
    expect((message as { event: SessionEvent }).event.payload.error).toEqual({
      code: "E_BOOM",
      message: "boom",
    });
  });

  it("posts session.destroyed events without an error field", async () => {
    const { postSessionEvent } = await import("./bridge");
    const payload = { ...samplePayload, status: "disconnected" as const };
    const event: SessionEvent = { type: "session.destroyed", payload };
    postSessionEvent(event);
    const [message] = postMessage.mock.calls[0]!;
    expect((message as { event: SessionEvent }).event.payload.error).toBeUndefined();
  });
});

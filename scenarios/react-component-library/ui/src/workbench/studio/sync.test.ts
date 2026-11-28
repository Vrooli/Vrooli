import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  DEFAULT_SYNC_CHANNEL_NAME,
  createSyncChannel,
  currentTabId,
  isStudioSyncEvent,
  reconcileDesign,
} from "./sync";
import type { StudioSyncEvent } from "./sync";

type ChannelMessageListener = (event: MessageEvent) => void;

/**
 * Stand-in for the browser channel so the BroadcastChannel path is exercised
 * even though jsdom does not implement it. `emit` is the other tab.
 */
class FakeBroadcastChannel {
  static instances: FakeBroadcastChannel[] = [];

  readonly name: string;
  readonly posted: unknown[] = [];
  readonly listeners = new Set<ChannelMessageListener>();
  closed = false;

  constructor(name: string) {
    this.name = name;
    FakeBroadcastChannel.instances.push(this);
  }

  postMessage(data: unknown): void {
    this.posted.push(data);
  }

  addEventListener(type: string, listener: ChannelMessageListener): void {
    if (type === "message") this.listeners.add(listener);
  }

  removeEventListener(type: string, listener: ChannelMessageListener): void {
    if (type === "message") this.listeners.delete(listener);
  }

  close(): void {
    this.closed = true;
    this.listeners.clear();
  }

  emit(data: unknown): void {
    const event = new MessageEvent("message", { data });
    for (const listener of this.listeners) listener(event);
  }
}

function postForeign(event: StudioSyncEvent, name = DEFAULT_SYNC_CHANNEL_NAME): void {
  window.dispatchEvent(new CustomEvent(name, { detail: event }));
}

const decisionBase = {
  activeId: "design-a",
  activeRevision: 4,
  remoteId: "design-a",
  remoteRevision: 5,
  dirty: false,
  saving: false,
};

beforeEach(() => {
  FakeBroadcastChannel.instances = [];
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("currentTabId", () => {
  it("is stable for this module instance and never empty", () => {
    expect(currentTabId()).toBe(currentTabId());
    expect(currentTabId().length).toBeGreaterThan(0);
  });
});

describe("isStudioSyncEvent", () => {
  it("accepts every well-formed event", () => {
    expect(isStudioSyncEvent({ type: "designs-changed", source: "tab-b" })).toBe(true);
    expect(
      isStudioSyncEvent({ type: "designs-changed", designId: "design-a", revision: 2, source: "tab-b" }),
    ).toBe(true);
    expect(isStudioSyncEvent({ type: "assets-changed", designId: "design-a", source: "tab-b" })).toBe(true);
    expect(isStudioSyncEvent({ type: "template-changed", source: "tab-b" })).toBe(true);
  });

  it("rejects malformed payloads", () => {
    const invalid: unknown[] = [
      null,
      undefined,
      42,
      "designs-changed",
      {},
      { type: "designs-changed" },
      { type: "designs-changed", source: "" },
      { type: "designs-changed", designId: "design-a", source: "tab-b" },
      { type: "designs-changed", designId: "design-a", revision: "2", source: "tab-b" },
      { type: "designs-changed", designId: "", revision: 2, source: "tab-b" },
      { type: "assets-changed", source: "tab-b" },
      { type: "assets-changed", designId: "", source: "tab-b" },
      { type: "assets-changed", designId: 7, source: "tab-b" },
      { type: "unknown", source: "tab-b" },
    ];
    for (const payload of invalid) {
      expect(isStudioSyncEvent(payload)).toBe(false);
    }
  });

  it("rejects a non-finite revision", () => {
    expect(
      isStudioSyncEvent({
        type: "designs-changed",
        designId: "design-a",
        revision: Number.POSITIVE_INFINITY,
        source: "tab-b",
      }),
    ).toBe(false);
  });
});

describe("reconcileDesign", () => {
  it("ignores another design even when it is newer and we have edits", () => {
    expect(
      reconcileDesign({ ...decisionBase, remoteId: "design-b", dirty: true }),
    ).toBe("ignore");
  });

  it("ignores while a save is in flight; the save settles and broadcasts", () => {
    expect(reconcileDesign({ ...decisionBase, saving: true })).toBe("ignore");
  });

  it("lets saving win over dirty: an in-flight save is not a conflict yet", () => {
    expect(reconcileDesign({ ...decisionBase, saving: true, dirty: true })).toBe("ignore");
  });

  it("reports a conflict rather than discarding unsaved local edits", () => {
    expect(reconcileDesign({ ...decisionBase, dirty: true })).toBe("conflict");
  });

  it("reports a conflict for unsaved edits even when revisions match", () => {
    expect(reconcileDesign({ ...decisionBase, dirty: true, remoteRevision: 4 })).toBe("conflict");
  });

  it("reloads a clean design when the remote revision is newer", () => {
    expect(reconcileDesign(decisionBase)).toBe("reload");
  });

  it("ignores an equal revision", () => {
    expect(reconcileDesign({ ...decisionBase, remoteRevision: 4 })).toBe("ignore");
  });

  it("ignores an older revision (replayed or reordered message)", () => {
    expect(reconcileDesign({ ...decisionBase, remoteRevision: 3 })).toBe("ignore");
  });
});

describe("sync channel on the window fallback", () => {
  beforeEach(() => {
    // jsdom has no BroadcastChannel, but pin it so a future jsdom release
    // cannot silently switch this suite onto the async path.
    vi.stubGlobal("BroadcastChannel", undefined);
  });

  it("delivers another tab's event to subscribers", () => {
    const channel = createSyncChannel();
    const received: StudioSyncEvent[] = [];
    channel.subscribe((event) => received.push(event));
    postForeign({ type: "designs-changed", designId: "design-a", revision: 3, source: "tab-b" });
    expect(received).toHaveLength(1);
    expect(received[0]).toMatchObject({ type: "designs-changed", designId: "design-a", revision: 3 });
    channel.close();
  });

  it("never delivers the tab's own event", () => {
    const channel = createSyncChannel();
    const received = vi.fn();
    channel.subscribe(received);
    channel.post({ type: "designs-changed", source: currentTabId() });
    channel.post({ type: "assets-changed", designId: "design-a", source: currentTabId() });
    expect(received).not.toHaveBeenCalled();
    channel.close();
  });

  it("stamps the source instead of trusting the caller", () => {
    const channel = createSyncChannel();
    const raw = vi.fn();
    window.addEventListener(DEFAULT_SYNC_CHANNEL_NAME, raw);
    channel.post({ type: "template-changed", source: "spoofed" });
    expect(raw).toHaveBeenCalledTimes(1);
    const event = raw.mock.calls[0]?.[0] as CustomEvent<StudioSyncEvent> | undefined;
    expect(event?.detail.source).toBe(currentTabId());
    window.removeEventListener(DEFAULT_SYNC_CHANNEL_NAME, raw);
    channel.close();
  });

  it("stops delivering after close", () => {
    const channel = createSyncChannel();
    const received = vi.fn();
    channel.subscribe(received);
    channel.close();
    postForeign({ type: "designs-changed", source: "tab-b" });
    expect(received).not.toHaveBeenCalled();
  });

  it("lets one subscriber detach without affecting the other", () => {
    const channel = createSyncChannel();
    const first = vi.fn();
    const second = vi.fn();
    const unsubscribe = channel.subscribe(first);
    channel.subscribe(second);
    unsubscribe();
    postForeign({ type: "designs-changed", source: "tab-b" });
    expect(first).not.toHaveBeenCalled();
    expect(second).toHaveBeenCalledTimes(1);
    channel.close();
  });

  it("drops malformed events without throwing", () => {
    const channel = createSyncChannel();
    const received = vi.fn();
    channel.subscribe(received);
    expect(() => {
      window.dispatchEvent(new CustomEvent(DEFAULT_SYNC_CHANNEL_NAME, { detail: null }));
      window.dispatchEvent(new CustomEvent(DEFAULT_SYNC_CHANNEL_NAME, { detail: 42 }));
      window.dispatchEvent(new CustomEvent(DEFAULT_SYNC_CHANNEL_NAME, { detail: { type: "unknown", source: "tab-b" } }));
      window.dispatchEvent(
        new CustomEvent(DEFAULT_SYNC_CHANNEL_NAME, { detail: { type: "assets-changed", source: "tab-b" } }),
      );
    }).not.toThrow();
    expect(received).not.toHaveBeenCalled();
    channel.close();
  });

  it("honours a custom channel name", () => {
    const channel = createSyncChannel("rcl.test.isolated");
    const received = vi.fn();
    channel.subscribe(received);
    postForeign({ type: "designs-changed", source: "tab-b" });
    expect(received).not.toHaveBeenCalled();
    postForeign({ type: "designs-changed", source: "tab-b" }, "rcl.test.isolated");
    expect(received).toHaveBeenCalledTimes(1);
    channel.close();
  });
});

describe("sync channel on BroadcastChannel", () => {
  beforeEach(() => {
    vi.stubGlobal("BroadcastChannel", FakeBroadcastChannel);
  });

  it("constructs the named channel and stamps outgoing messages", () => {
    const channel = createSyncChannel("rcl.test.broadcast");
    const instance = FakeBroadcastChannel.instances[0];
    expect(instance?.name).toBe("rcl.test.broadcast");
    channel.post({ type: "designs-changed", designId: "design-a", revision: 9, source: "spoofed" });
    expect(instance?.posted[0]).toMatchObject({
      type: "designs-changed",
      designId: "design-a",
      revision: 9,
      source: currentTabId(),
    });
    channel.close();
    expect(instance?.closed).toBe(true);
  });

  it("delivers a foreign message and filters our own", () => {
    const channel = createSyncChannel();
    const received = vi.fn();
    channel.subscribe(received);
    const instance = FakeBroadcastChannel.instances[0];
    instance?.emit({ type: "assets-changed", designId: "design-a", source: "tab-b" });
    instance?.emit({ type: "assets-changed", designId: "design-a", source: currentTabId() });
    instance?.emit({ type: "assets-changed", source: "tab-b" });
    expect(received).toHaveBeenCalledTimes(1);
    expect(received).toHaveBeenCalledWith({
      type: "assets-changed",
      designId: "design-a",
      source: "tab-b",
    });
    channel.close();
  });

  it("detaches the listener and stops posting after close", () => {
    const channel = createSyncChannel();
    const received = vi.fn();
    channel.subscribe(received);
    const instance = FakeBroadcastChannel.instances[0];
    channel.close();
    instance?.emit({ type: "designs-changed", source: "tab-b" });
    expect(received).not.toHaveBeenCalled();
    expect(instance?.listeners.size).toBe(0);
    expect(instance?.posted).toHaveLength(0);
  });

  it("falls back to the window channel when the constructor is refused", () => {
    // A plain function is constructable at runtime and throws like a private
    // mode BroadcastChannel; the test never writes `new` against its type.
    const refusedBroadcastChannel = function refused(): never {
      throw new Error("SecurityError");
    };
    vi.stubGlobal("BroadcastChannel", refusedBroadcastChannel);
    const channel = createSyncChannel();
    const received = vi.fn();
    channel.subscribe(received);
    postForeign({ type: "designs-changed", source: "tab-b" });
    expect(received).toHaveBeenCalledTimes(1);
    channel.close();
  });
});

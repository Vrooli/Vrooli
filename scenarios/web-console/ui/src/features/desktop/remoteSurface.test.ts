import { describe, expect, it, vi } from "vitest";
import { boundedClipboardText, boundedRemoteTextInput, mapSurfacePoint, OrderedLeaseInput, REMOTE_SURFACE_CLIPBOARD_LIMIT, REMOTE_SURFACE_TEXT_INPUT_LIMIT, SingleFlightSurfaceOperation } from "./remoteSurface";
import { createWorkspaceSurfaceContract, transitionWorkspaceSurface, workspaceSurfacePresentationAttributes, type WorkspaceSurfaceOwner } from "./workspaceSurfaceContract";

describe("shared remote-surface input contract", () => {
  it("coalesces duplicate starts while preserving queued cleanup", async () => {
    const gate = new SingleFlightSurfaceOperation();
    let finish!: (value: string) => void;
    const open = vi.fn(() => new Promise<string>((resolve) => { finish = resolve; }));
    const first = gate.run("open", open);
    const duplicate = gate.run("open", open);
    expect(gate.busy).toBe(true);
    expect(duplicate).toBe(first);
    expect(open).toHaveBeenCalledTimes(1);
    // Repeated pane open requests must resolve to the one owner's session.
    const close = vi.fn(async () => "closed");
    const closing = gate.run("close", close);
    expect(close).not.toHaveBeenCalled();
    finish("owner-session");
    await expect(first).resolves.toBe("owner-session");
    await expect(closing).resolves.toBe("closed");
    expect(close).toHaveBeenCalledTimes(1);
    await expect(gate.run("open", async () => "reconnected-session")).resolves.toBe("reconnected-session");
  });

  it("coalesces duplicate handback and never creates a second session owner", async () => {
    const gate = new SingleFlightSurfaceOperation();
    let complete!: () => void;
    const ownerOpen = vi.fn(async () => "owner-session-1");
    const session = await gate.run("open", ownerOpen);
    const returnToObserver = vi.fn(() => new Promise<void>((resolve) => { complete = resolve; }));
    const first = gate.run("handback", returnToObserver);
    const duplicate = gate.run("handback", returnToObserver);
    expect(duplicate).toBe(first);
    expect(returnToObserver).toHaveBeenCalledTimes(1);
    expect(ownerOpen).toHaveBeenCalledTimes(1);
    complete();
    await expect(first).resolves.toBeUndefined();
    expect(session).toBe("owner-session-1");
  });

  it("still runs cleanup when an owner transition rejects", async () => {
    const gate = new SingleFlightSurfaceOperation();
    let fail!: (error: Error) => void;
    const opening = gate.run("open", () => new Promise<void>((_resolve, reject) => { fail = reject; }));
    const close = vi.fn(async () => "closed");
    const closing = gate.run("close", close);
    fail(new Error("owner unavailable"));
    await expect(opening).rejects.toThrow("owner unavailable");
    await expect(closing).resolves.toBe("closed");
    expect(close).toHaveBeenCalledTimes(1);
  });

  it("serializes controller commands in arrival order", async () => {
    const queue = new OrderedLeaseInput();
    const lease = { id: "lease-a", epoch: 2n, controller: true };
    queue.updateLease(lease);
    const observed: number[] = [];
    let releaseFirst!: () => void;
    const first = queue.enqueue(lease, async (order) => {
      observed.push(order);
      await new Promise<void>((resolve) => { releaseFirst = resolve; });
    });
    const second = queue.enqueue(lease, async (order) => { observed.push(order); });
    await Promise.resolve();
    expect(observed).toEqual([1]);
    releaseFirst();
    await expect(Promise.all([first, second])).resolves.toEqual([true, true]);
    expect(observed).toEqual([1, 2]);
  });

  it("drops queued input if the owner rotates or revokes the lease", async () => {
    const queue = new OrderedLeaseInput();
    const lease = { id: "lease-a", epoch: 2n, controller: true };
    queue.updateLease(lease);
    let releaseFirst!: () => void;
    const first = queue.enqueue(lease, () => new Promise<void>((resolve) => { releaseFirst = resolve; }));
    const send = vi.fn(async () => undefined);
    const queued = queue.enqueue(lease, send);
    await Promise.resolve();
    queue.updateLease({ id: "lease-b", epoch: 1n, controller: true });
    releaseFirst();
    await Promise.all([first, queued]);
    expect(send).not.toHaveBeenCalled();
  });

  it("rejects stale input even when the newer lease is non-controlling", async () => {
    const queue = new OrderedLeaseInput();
    const lease = { id: "lease-a", epoch: 2n, controller: true };
    queue.updateLease(lease);
    let release!: () => void;
    const first = queue.enqueue(lease, () => new Promise<void>((resolve) => { release = resolve; }));
    const stale = vi.fn(async () => undefined);
    const queued = queue.enqueue(lease, stale);
    await Promise.resolve();
    queue.updateLease({ id: "", epoch: 0n, controller: false });
    release();
    await Promise.all([first, queued]);
    expect(stale).not.toHaveBeenCalled();
  });

  it("bounds queued commands when the owner transport stalls", async () => {
    const queue = new OrderedLeaseInput(2);
    const lease = { id: "lease-a", epoch: 2n, controller: true };
    queue.updateLease(lease);
    let release!: () => void;
    const first = queue.enqueue(lease, () => new Promise<void>((resolve) => { release = resolve; }));
    const second = queue.enqueue(lease, async () => undefined);
    await Promise.resolve();
    await expect(queue.enqueue(lease, async () => undefined)).resolves.toBe(false);
    const priorityRelease = queue.enqueue(lease, async () => undefined, true);
    release();
    await expect(Promise.all([first, second, priorityRelease])).resolves.toEqual([true, true, true]);
    await expect(queue.enqueue(lease, async () => undefined)).resolves.toBe(true);
  });

  it("bounds clipboard text by encoded bytes", () => {
    expect(boundedClipboardText("ok")).toBe("ok");
    expect(() => boundedClipboardText("é".repeat(REMOTE_SURFACE_CLIPBOARD_LIMIT / 2 + 1))).toThrow("64 KiB");
  });

  it("keeps browser page text input within BAS's smaller UTF-8 byte cap", () => {
    expect(boundedRemoteTextInput("é".repeat(REMOTE_SURFACE_TEXT_INPUT_LIMIT / 2))).toBe("é".repeat(REMOTE_SURFACE_TEXT_INPUT_LIMIT / 2));
    expect(() => boundedRemoteTextInput("é".repeat(REMOTE_SURFACE_TEXT_INPUT_LIMIT / 2 + 1))).toThrow("4 KiB");
  });

  it("maps contain-fitted high-DPI frame coordinates and rejects letterbox clicks", () => {
    const bounds = { left: 0, top: 0, width: 200, height: 200 };
    expect(mapSurfacePoint(100, 50, bounds, { width: 400, height: 200 }, { width: 800, height: 400 })).toEqual({ x: 400, y: 0 });
    expect(mapSurfacePoint(100, 49, bounds, { width: 400, height: 200 }, { width: 800, height: 400 })).toBeUndefined();
    expect(mapSurfacePoint(-10, 100, bounds, { width: 400, height: 200 }, { width: 800, height: 400 }, true)).toEqual({ x: 0, y: 200 });
  });
});

describe("shared workspace surface handoff contract", () => {
  it("projects focus, access, lease and lifecycle through one shared presentation contract", () => {
    expect(workspaceSurfacePresentationAttributes("bas", "controlling", true, true)).toMatchObject({
      "data-surface-owner": "bas", "data-surface-lifecycle": "ready", "data-surface-access": "control",
      "data-surface-lease-held": "true", "data-surface-can-control": "true", "data-surface-focused": "true",
    });
    expect(workspaceSurfacePresentationAttributes("device-control", "reconnecting", false, true)).toMatchObject({
      "data-surface-owner": "device-control", "data-surface-lifecycle": "reconnecting", "data-surface-access": "read-only",
      "data-surface-lease-held": "false", "data-surface-can-control": "true",
    });
    expect(workspaceSurfacePresentationAttributes("file-preview", "failed", true, true)).toMatchObject({
      "data-surface-owner": "file-preview", "data-surface-lifecycle": "failed", "data-surface-access": "read-only",
      "data-surface-lease-held": "false", "data-surface-can-control": "false",
    });
  });

  it.each<WorkspaceSurfaceOwner>(["bas", "device-control", "file-preview"])("keeps %s as the fixed owner through focus, close, and failure", (owner) => {
    let state = createWorkspaceSurfaceContract(owner, `${owner}-surface`, owner !== "file-preview");
    state = transitionWorkspaceSurface(state, { type: "focus", focused: true });
    state = transitionWorkspaceSurface(state, { type: "request", operation: "open" });
    expect(state.lifecycle).toBe("loading");
    state = transitionWorkspaceSurface(state, { type: "owner-opened", owner, canControl: owner !== "file-preview" });
    expect(state.owner).toBe(owner);
    expect(state.access).toBe(owner === "file-preview" ? "read-only" : "observe");
    state = transitionWorkspaceSurface(state, { type: "request", operation: "close" });
    expect(state.lifecycle).toBe("ready"); // close is only pending until the owner confirms it
    expect(state.pending).toBe("close");
    state = transitionWorkspaceSurface(state, { type: "owner-closed", owner });
    expect(state.lifecycle).toBe("closed");
    expect(state.focused).toBe(false);
    state = transitionWorkspaceSurface(state, { type: "request", operation: "open" });
    state = transitionWorkspaceSurface(state, { type: "owner-failed", owner, message: "owner unavailable" });
    expect(state.lifecycle).toBe("failed");
    expect(state.pending).toBeUndefined();
    expect(state.error).toBe("owner unavailable");
    expect(state.owner).toBe(owner);
    expect(state.focused).toBe(false);
  });

  it.each<WorkspaceSurfaceOwner>(["bas", "device-control"])("shares lease grant and handback transitions for %s", (owner) => {
    let state = createWorkspaceSurfaceContract(owner, `${owner}-surface`, true);
    state = transitionWorkspaceSurface(state, { type: "request", operation: "open" });
    state = transitionWorkspaceSurface(state, { type: "owner-opened", owner, canControl: true });
    state = transitionWorkspaceSurface(state, { type: "request", operation: "takeover" });
    state = transitionWorkspaceSurface(state, { type: "owner-control-granted", owner, leaseHeld: true });
    expect(state.access).toBe("control");
    expect(state.leaseHeld).toBe(true);
    state = transitionWorkspaceSurface(state, { type: "request", operation: "handback" });
    state = transitionWorkspaceSurface(state, { type: "owner-handback-complete", owner, leaseHeld: false });
    expect(state.access).toBe("observe");
    expect(state.leaseHeld).toBe(false);
  });

  it("ignores another owner's lease response, stale lease grant, and file control request", () => {
    let state = createWorkspaceSurfaceContract("file-preview", "file-1", false);
    state = transitionWorkspaceSurface(state, { type: "request", operation: "open" });
    state = transitionWorkspaceSurface(state, { type: "owner-opened", owner: "file-preview", canControl: true });
    expect(state.canControl).toBe(false);
    expect(transitionWorkspaceSurface(state, { type: "request", operation: "takeover" })).toEqual(state);
    expect(transitionWorkspaceSurface(state, { type: "owner-control-granted", owner: "bas", leaseHeld: true })).toEqual(state);
    expect(transitionWorkspaceSurface(state, { type: "owner-control-granted", owner: "file-preview", leaseHeld: true })).toEqual(state);
  });

  it("keeps control revoked until owner close and requires owner acknowledgement for reconnect", () => {
    let state = createWorkspaceSurfaceContract("device-control", "device-1", true);
    state = transitionWorkspaceSurface(state, { type: "request", operation: "open" });
    state = transitionWorkspaceSurface(state, { type: "owner-opened", owner: "device-control", canControl: true, leaseHeld: true });
    state = transitionWorkspaceSurface(state, { type: "owner-revoked", owner: "device-control" });
    expect(state.lifecycle).toBe("revoked");
    expect(state.leaseHeld).toBe(false);
    state = transitionWorkspaceSurface(state, { type: "request", operation: "reconnect" });
    expect(state.lifecycle).toBe("revoked");
    state = transitionWorkspaceSurface(state, { type: "request", operation: "close" });
    state = transitionWorkspaceSurface(state, { type: "owner-closed", owner: "device-control" });
    state = transitionWorkspaceSurface(state, { type: "request", operation: "open" });
    state = transitionWorkspaceSurface(state, { type: "owner-opened", owner: "device-control", canControl: true });
    state = transitionWorkspaceSurface(state, { type: "request", operation: "reconnect" });
    expect(state.lifecycle).toBe("reconnecting");
    expect(transitionWorkspaceSurface(state, { type: "owner-reconnected", owner: "bas", access: "observe", leaseHeld: false })).toEqual(state);
    state = transitionWorkspaceSurface(state, { type: "owner-reconnected", owner: "device-control", access: "observe", leaseHeld: false });
    expect(state.lifecycle).toBe("ready");
    expect(state.access).toBe("observe");
  });

  it("preserves cleanup when close races an open or owner transition fails", () => {
    let state = createWorkspaceSurfaceContract("bas", "browser-1", true);
    state = transitionWorkspaceSurface(state, { type: "request", operation: "open" });
    state = transitionWorkspaceSurface(state, { type: "request", operation: "close" });
    expect(state.pending).toBe("close");
    state = transitionWorkspaceSurface(state, { type: "owner-opened", owner: "bas", canControl: true });
    expect(state.pending).toBe("close");
    state = transitionWorkspaceSurface(state, { type: "owner-failed", owner: "bas", message: "partial owner open" });
    expect(state.lifecycle).toBe("failed");
    expect(state.pending).toBe("close");
    state = transitionWorkspaceSurface(state, { type: "owner-closed", owner: "bas" });
    expect(state.lifecycle).toBe("closed");
    expect(state.pending).toBeUndefined();
  });

  it("reports failed reconnect as owner failure and waits for close acknowledgement", () => {
    let state = createWorkspaceSurfaceContract("device-control", "device-2", true);
    state = transitionWorkspaceSurface(state, { type: "request", operation: "open" });
    state = transitionWorkspaceSurface(state, { type: "owner-opened", owner: "device-control", canControl: true, leaseHeld: true });
    state = transitionWorkspaceSurface(state, { type: "request", operation: "reconnect" });
    state = transitionWorkspaceSurface(state, { type: "owner-failed", owner: "device-control", message: "reconnect open failed" });
    expect(state.lifecycle).toBe("failed");
    expect(state.leaseHeld).toBe(false);
    expect(state.focused).toBe(false);
    state = transitionWorkspaceSurface(state, { type: "request", operation: "close" });
    expect(state.pending).toBe("close");
    state = transitionWorkspaceSurface(state, { type: "owner-closed", owner: "device-control" });
    expect(state.lifecycle).toBe("closed");
    expect(state.pending).toBeUndefined();
  });
});

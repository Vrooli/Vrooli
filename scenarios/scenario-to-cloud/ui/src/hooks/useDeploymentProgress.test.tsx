import "@testing-library/jest-dom";
import { act, renderHook, waitFor } from "@testing-library/react";
import { vi } from "vitest";

import { useDeploymentProgress } from "./useDeploymentProgress";

type Listener = (event: MessageEvent<string>) => void;

class FakeEventSource {
  static instances: FakeEventSource[] = [];
  readonly url: string;
  readonly listeners = new Map<string, Listener[]>();
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  closed = false;

  constructor(url: string) {
    this.url = url;
    FakeEventSource.instances.push(this);
  }

  addEventListener(name: string, listener: Listener) {
    this.listeners.set(name, [...(this.listeners.get(name) ?? []), listener]);
  }

  close() {
    this.closed = true;
  }

  emit(name: string, data: unknown) {
    const event = { data: typeof data === "string" ? data : JSON.stringify(data) } as MessageEvent<string>;
    for (const listener of this.listeners.get(name) ?? []) listener(event);
  }
}

describe("useDeploymentProgress", () => {
  beforeEach(() => {
    FakeEventSource.instances = [];
    vi.stubGlobal("EventSource", FakeEventSource);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("connects, tracks every event type, and reports successful completion", async () => {
    const onComplete = vi.fn();
    const onError = vi.fn();
    const { result } = renderHook(() => useDeploymentProgress("dep/1", {
      runId: "run 1",
      onComplete,
      onError,
    }));

    await waitFor(() => expect(result.current.progress).not.toBeNull());
    const source = FakeEventSource.instances[0];
    if (!source) throw new Error("expected EventSource instance");
    expect(source.url).toContain("dep%2F1");
    expect(source.url).toContain("run_id=run%201");

    act(() => source.onopen?.());
    expect(result.current.isConnected).toBe(true);
    act(() => source.emit("step_started", { step: "bundle_build", step_title: "Building", progress: 10 }));
    expect(result.current.progress?.steps.find((step) => step.id === "bundle_build")?.status).toBe("running");
    act(() => source.emit("step_completed", { step: "bundle_build", step_title: "Built", progress: 25 }));
    expect(result.current.progress?.steps.find((step) => step.id === "bundle_build")?.status).toBe("completed");
    act(() => source.emit("progress_update", { step: "upload", step_title: "Uploading", progress: 50 }));
    expect(result.current.progress?.currentStep).toBe("upload");
    act(() => source.emit("preflight_result", { preflight_result: { ok: true, checks: [] } }));
    expect(result.current.progress?.preflightResult).toEqual({ ok: true, checks: [] });
    act(() => source.emit("completed", { message: "Finished" }));
    expect(result.current.progress?.isComplete).toBe(true);
    expect(result.current.progress?.progress).toBe(100);
    expect(source.closed).toBe(true);
    expect(onComplete).toHaveBeenCalledWith(true);
    expect(onError).not.toHaveBeenCalled();
  });

  it("reports deployment errors, connection loss, malformed events, and reset", async () => {
    const onComplete = vi.fn();
    const onError = vi.fn();
    const { result, rerender, unmount } = renderHook(
      ({ id }: { id: string | null }) => useDeploymentProgress(id, { onComplete, onError }),
      { initialProps: { id: "deployment" as string | null } },
    );
    await waitFor(() => expect(result.current.progress).not.toBeNull());
    const source = FakeEventSource.instances[0];
    if (!source) throw new Error("expected EventSource instance");

    act(() => source.onerror?.());
    expect(result.current.connectionError).toBe("Connection lost, reconnecting...");
    act(() => {
      source.emit("step_started", "not json");
      source.emit("step_completed", "not json");
      source.emit("progress_update", "not json");
      source.emit("preflight_result", "not json");
      source.emit("completed", "not json");
      source.emit("deployment_error", "not json");
    });
    act(() => source.emit("deployment_error", {
      step: "upload",
      step_title: "Upload",
      progress: 60,
      error: "Upload failed",
    }));
    expect(result.current.progress?.error).toBe("Upload failed");
    expect(result.current.progress?.isComplete).toBe(true);
    expect(onError).toHaveBeenCalledWith("Upload failed");
    expect(onComplete).toHaveBeenCalledWith(false, "Upload failed");

    act(() => result.current.reset());
    expect(result.current.progress).toBeNull();
    expect(result.current.connectionError).toBeNull();
    rerender({ id: null });
    expect(source.closed).toBe(true);
    unmount();
  });

  it("handles sparse progress payloads and unknown steps", async () => {
    const { result } = renderHook(() => useDeploymentProgress("deployment"));
    await waitFor(() => expect(result.current.progress).not.toBeNull());
    const source = FakeEventSource.instances[0];
    if (!source) throw new Error("expected EventSource instance");
    act(() => source.emit("step_started", { step: "unknown", progress: 1 }));
    expect(result.current.progress?.currentStepTitle).toBe("unknown");
    act(() => source.emit("step_completed", { step: "deploy", progress: 2 }));
    act(() => source.emit("progress_update", { step: "not-a-real-step", progress: 3 }));
    act(() => source.emit("preflight_result", {}));
    act(() => source.emit("completed", {}));
    expect(result.current.progress?.currentStepTitle).toBe("Deployment complete");
  });
});

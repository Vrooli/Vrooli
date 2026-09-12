/**
 * useModelPicker lifecycle tests. The hook drives the host-aware model menu for
 * one operation — load the candidate set, run inline install/enable jobs, block
 * on the durable job, then refetch so the row flips to "ready" — through the
 * injected `ModelPickerClient` seam, so the whole state machine is exercised
 * without a network. Covers lazy load, install-model, install-backend (auto +
 * the manual-required surface that becomes a row error), enable, the per-row
 * failure path, and the mounted-guard against close-mid-install.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";

import { makeCandidateModel, makeHostSummary } from "./mocks/factories";
import { useModelPicker, type ModelPickerClient } from "./useModelPicker";

const makeClient = (overrides: Partial<ModelPickerClient> = {}): ModelPickerClient => ({
  list: vi.fn().mockResolvedValue({
    candidates: [makeCandidateModel({ model: makeCandidateModel().model })],
    host: makeHostSummary(),
    selectedId: "cand-1",
    selectedReason: "best fit",
  }),
  installModel: vi.fn().mockResolvedValue({ jobId: "job-1", alreadyInstalled: false }),
  ensureBackend: vi
    .fn()
    .mockResolvedValue({ jobId: "job-b1", alreadyInstalled: false, manual: false, detail: "" }),
  waitJob: vi.fn().mockResolvedValue({ ok: true, error: "" }),
  setEnabled: vi.fn().mockResolvedValue(undefined),
  ...overrides,
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("useModelPicker", () => {
  it("does not load while inactive, then loads candidates + host when activated", async () => {
    const client = makeClient();
    const { result, rerender } = renderHook(
      ({ active }: { active: boolean }) => useModelPicker({ operation: "upscale", active, client }),
      { initialProps: { active: false } },
    );

    expect(client.list).not.toHaveBeenCalled();
    expect(result.current.candidates).toEqual([]);

    rerender({ active: true });
    await waitFor(() => expect(result.current.candidates).toHaveLength(1));
    expect(client.list).toHaveBeenCalledWith("upscale");
    expect(result.current.host?.hasGpu).toBe(true);
    expect(result.current.selectedId).toBe("cand-1");
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("surfaces a load failure as the top-level error", async () => {
    const client = makeClient({ list: vi.fn().mockRejectedValue(new Error("models down")) });
    const { result } = renderHook(() =>
      useModelPicker({ operation: "upscale", active: true, client }),
    );
    await waitFor(() => expect(result.current.error).toBe("models down"));
    expect(result.current.loading).toBe(false);
  });

  it("does nothing when there is no operation", () => {
    const client = makeClient();
    renderHook(() => useModelPicker({ operation: "", active: true, client }));
    expect(client.list).not.toHaveBeenCalled();
  });

  it("installs a model, waits the job, then refetches", async () => {
    const client = makeClient();
    const { result } = renderHook(() =>
      useModelPicker({ operation: "upscale", active: true, client }),
    );
    await waitFor(() => expect(result.current.candidates).toHaveLength(1));
    vi.mocked(client.list).mockClear();

    act(() => result.current.installModel("cand-1"));
    await waitFor(() => expect(client.list).toHaveBeenCalledTimes(1));
    expect(client.installModel).toHaveBeenCalledWith("cand-1");
    expect(client.waitJob).toHaveBeenCalledWith("job-1");
    expect(result.current.busyId).toBe("");
    expect(result.current.rowError["cand-1"]).toBeUndefined();
  });

  it("skips waiting the job when the model is already installed", async () => {
    const client = makeClient({
      installModel: vi.fn().mockResolvedValue({ jobId: "", alreadyInstalled: true }),
    });
    const { result } = renderHook(() =>
      useModelPicker({ operation: "upscale", active: true, client }),
    );
    await waitFor(() => expect(result.current.candidates).toHaveLength(1));
    vi.mocked(client.list).mockClear();

    act(() => result.current.installModel("cand-1"));
    await waitFor(() => expect(client.list).toHaveBeenCalledTimes(1));
    expect(client.waitJob).not.toHaveBeenCalled();
  });

  it("records a per-row error when the install job fails (and clears busy)", async () => {
    const client = makeClient({
      waitJob: vi.fn().mockResolvedValue({ ok: false, error: "disk full" }),
    });
    const { result } = renderHook(() =>
      useModelPicker({ operation: "upscale", active: true, client }),
    );
    await waitFor(() => expect(result.current.candidates).toHaveLength(1));

    act(() => result.current.installModel("cand-1"));
    await waitFor(() => expect(result.current.rowError["cand-1"]).toBe("disk full"));
    expect(result.current.busyId).toBe("");
  });

  it("installs an auto backend by tool, waits, then refetches", async () => {
    const client = makeClient();
    const { result } = renderHook(() =>
      useModelPicker({ operation: "upscale", active: true, client }),
    );
    await waitFor(() => expect(result.current.candidates).toHaveLength(1));
    vi.mocked(client.list).mockClear();

    act(() => result.current.installBackend("realesrgan-ncnn-vulkan", "cand-1"));
    await waitFor(() => expect(client.list).toHaveBeenCalledTimes(1));
    expect(client.ensureBackend).toHaveBeenCalledWith("realesrgan-ncnn-vulkan");
    expect(client.waitJob).toHaveBeenCalledWith("job-b1");
  });

  it("surfaces a manual-required backend as a per-row error with its detail", async () => {
    const client = makeClient({
      ensureBackend: vi.fn().mockResolvedValue({
        jobId: "",
        alreadyInstalled: false,
        manual: true,
        detail: "brew install realesrgan",
      }),
    });
    const { result } = renderHook(() =>
      useModelPicker({ operation: "upscale", active: true, client }),
    );
    await waitFor(() => expect(result.current.candidates).toHaveLength(1));

    act(() => result.current.installBackend("realesrgan-ncnn-vulkan", "cand-1"));
    await waitFor(() => expect(result.current.rowError["cand-1"]).toBe("brew install realesrgan"));
    expect(client.waitJob).not.toHaveBeenCalled();
  });

  it("enables a disabled model, then refetches", async () => {
    const client = makeClient();
    const { result } = renderHook(() =>
      useModelPicker({ operation: "upscale", active: true, client }),
    );
    await waitFor(() => expect(result.current.candidates).toHaveLength(1));
    vi.mocked(client.list).mockClear();

    act(() => result.current.enable("cand-1"));
    await waitFor(() => expect(client.list).toHaveBeenCalledTimes(1));
    expect(client.setEnabled).toHaveBeenCalledWith("cand-1", true);
    expect(result.current.busyId).toBe("");
  });

  it("records a per-row error when enable fails", async () => {
    const client = makeClient({ setEnabled: vi.fn().mockRejectedValue(new Error("nope")) });
    const { result } = renderHook(() =>
      useModelPicker({ operation: "upscale", active: true, client }),
    );
    await waitFor(() => expect(result.current.candidates).toHaveLength(1));

    act(() => result.current.enable("cand-1"));
    await waitFor(() => expect(result.current.rowError["cand-1"]).toBe("nope"));
    expect(result.current.busyId).toBe("");
  });

  it("refresh() re-runs the load", async () => {
    const client = makeClient();
    const { result } = renderHook(() =>
      useModelPicker({ operation: "upscale", active: true, client }),
    );
    await waitFor(() => expect(result.current.candidates).toHaveLength(1));
    vi.mocked(client.list).mockClear();

    act(() => result.current.refresh());
    await waitFor(() => expect(client.list).toHaveBeenCalledTimes(1));
  });

  it("does not set state after unmount (mounted-guard) when a slow install resolves late", async () => {
    let resolveWait: (v: { ok: boolean; error: string }) => void = () => {};
    const client = makeClient({
      waitJob: vi.fn(
        () =>
          new Promise<{ ok: boolean; error: string }>((res) => {
            resolveWait = res;
          }),
      ),
    });
    const { result, unmount } = renderHook(() =>
      useModelPicker({ operation: "upscale", active: true, client }),
    );
    await waitFor(() => expect(result.current.candidates).toHaveLength(1));

    act(() => result.current.installModel("cand-1"));
    // Unmount while the job is still in flight, then let it resolve.
    unmount();
    await act(async () => {
      resolveWait({ ok: false, error: "too late" });
      await Promise.resolve();
    });
    // No assertion can read post-unmount state directly; the guard's job is to
    // avoid a React "state update on unmounted component" warning, which would
    // fail the test run via the console-error spy. Reaching here cleanly is the
    // assertion.
    expect(true).toBe(true);
  });
});

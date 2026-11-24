import { act, renderHook, waitFor, cleanup } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { QueryClient } from "@tanstack/react-query";

const { getExperiment, streamExperimentEvents } = vi.hoisted(() => ({ getExperiment: vi.fn(), streamExperimentEvents: vi.fn() }));
vi.mock("../../services/experiment", () => ({ getExperiment, streamExperimentEvents }));

import { useExperimentStream } from "./useExperimentStream";

const messages = { complete: "Complete", polling: "Checking…", streamClosed: "Stream closed" };
const running = { id: "exp-1", name: "run", status: "running", createdAt: "created", startedAt: "started", finishedAt: "", error: "", resultRef: "", machineJson: "", recipe: {} };
const terminal = { ...running, status: "succeeded", finishedAt: "finished" };
const setup = (overrides: Record<string, unknown> = {}) => {
  const invalidateQueries = vi.fn();
  const loadReport = vi.fn();
  const setActiveExperiment = vi.fn();
  const props = {
    selectedExperimentId: "exp-1", selectedExperimentStatus: "running" as const, selectedId: "exp-1",
    queryClient: { invalidateQueries } as unknown as QueryClient, loadReport, messages, setActiveExperiment,
    ...overrides,
  };
  return { props, invalidateQueries, loadReport, setActiveExperiment };
};

beforeEach(() => vi.clearAllMocks());
afterEach(() => { cleanup(); vi.useRealTimers(); });

describe("useExperimentStream", () => {
  it("does not subscribe for missing or terminal selections", () => {
    const missing = setup({ selectedExperimentId: "" });
    const terminalSelection = setup({ selectedExperimentStatus: "succeeded" });
    renderHook(({ props }) => useExperimentStream(props), { initialProps: { props: missing.props } }).unmount();
    renderHook(({ props }) => useExperimentStream(props), { initialProps: { props: terminalSelection.props } });
    expect(streamExperimentEvents).not.toHaveBeenCalled();
  });

  it("publishes stream events, aborts on terminal status, and refreshes list/report", async () => {
    let onEvent: ((event: { experimentId: string; status: string; progress: number; message: string; at: string }) => void) | undefined;
    let signal: AbortSignal | undefined;
    streamExperimentEvents.mockImplementation((_id, callback, abortSignal) => {
      onEvent = callback;
      signal = abortSignal;
      return new Promise(() => {});
    });
    const { props, invalidateQueries, loadReport } = setup();
    const { result, unmount } = renderHook(() => useExperimentStream(props));
    await waitFor(() => expect(streamExperimentEvents).toHaveBeenCalled());
    act(() => onEvent?.({ experimentId: "exp-1", status: "running", progress: 42, message: "Working", at: "now" }));
    expect(result.current.liveEvent).toMatchObject({ status: "running", progress: 42 });
    act(() => onEvent?.({ experimentId: "exp-1", status: "succeeded", progress: 100, message: "Done", at: "later" }));
    expect(result.current.liveEvent?.status).toBe("succeeded");
    expect(signal?.aborted).toBe(true);
    expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["experiments", "list"] });
    expect(loadReport).toHaveBeenCalledWith("exp-1");
    unmount();
  });

  it("falls back to polling after stream closure and stops polling when the experiment finishes", async () => {
    vi.useFakeTimers();
    streamExperimentEvents.mockResolvedValue(undefined);
    const { props, setActiveExperiment, loadReport } = setup();
    const { result, unmount } = renderHook(() => useExperimentStream(props));
    await act(async () => { await Promise.resolve(); await Promise.resolve(); });
    expect(result.current.streamError).toBe(messages.streamClosed);
    getExperiment.mockResolvedValue({ experiment: running });
    await act(async () => { await vi.advanceTimersByTimeAsync(2500); });
    expect(setActiveExperiment).toHaveBeenCalledWith(running);
    expect(result.current.liveEvent).toMatchObject({ status: "running", progress: 0, message: messages.polling, at: "started" });
    getExperiment.mockResolvedValue({ experiment: terminal });
    await act(async () => { await vi.advanceTimersByTimeAsync(2500); });
    expect(result.current.liveEvent).toMatchObject({ status: "succeeded", progress: 100, message: messages.complete, at: "finished" });
    expect(loadReport).toHaveBeenCalledWith("exp-1");
    const callsAfterCompletion = getExperiment.mock.calls.length;
    await act(async () => { await vi.advanceTimersByTimeAsync(5000); });
    expect(getExperiment).toHaveBeenCalledTimes(callsAfterCompletion);
    unmount();
  });

  it("reports stream failures and polls even when the stream rejects with a non-Error", async () => {
    vi.useFakeTimers();
    streamExperimentEvents.mockRejectedValue("network unavailable");
    const { props } = setup();
    const { result, unmount } = renderHook(() => useExperimentStream(props));
    await act(async () => { await Promise.resolve(); await Promise.resolve(); });

    expect(result.current.streamError).toBe("network unavailable");
    getExperiment.mockResolvedValue({ experiment: undefined });
    await act(async () => { await vi.advanceTimersByTimeAsync(2500); });
    expect(result.current.liveEvent).toBeNull();
    unmount();
  });

  it("surfaces polling errors and retains the latest progress for the same experiment", async () => {
    vi.useFakeTimers();
    streamExperimentEvents.mockResolvedValue(undefined);
    const { props } = setup();
    const { result, unmount } = renderHook(() => useExperimentStream(props));
    await act(async () => { await Promise.resolve(); await Promise.resolve(); });

    act(() => result.current.setLiveEvent({ experimentId: "exp-1", status: "running", progress: 37, message: "Working", at: "now" }));
    getExperiment.mockResolvedValue({ experiment: { ...running, startedAt: "", createdAt: "created" } });
    await act(async () => { await vi.advanceTimersByTimeAsync(2500); });
    expect(result.current.liveEvent).toMatchObject({ progress: 37, message: messages.polling, at: "created" });

    getExperiment.mockRejectedValue(new Error("poll failed"));
    await act(async () => { await vi.advanceTimersByTimeAsync(2500); });
    expect(result.current.streamError).toBe("poll failed");
    unmount();
  });

  it("ignores stream callbacks and failures after unmount", async () => {
    let rejectStream: ((error: unknown) => void) | undefined;
    let onEvent: ((event: { experimentId: string; status: string; progress: number; message: string; at: string }) => void) | undefined;
    streamExperimentEvents.mockImplementation((_id, callback) => {
      onEvent = callback;
      return new Promise((_resolve, reject) => { rejectStream = reject; });
    });
    const { props } = setup();
    const { result, unmount } = renderHook(() => useExperimentStream(props));
    await waitFor(() => expect(streamExperimentEvents).toHaveBeenCalled());
    unmount();
    act(() => onEvent?.({ experimentId: "exp-1", status: "running", progress: 50, message: "Late", at: "later" }));
    await act(async () => { rejectStream?.(new Error("late failure")); await Promise.resolve(); });
    expect(result.current.liveEvent).toBeNull();
    expect(result.current.streamError).toBe("");
  });
});

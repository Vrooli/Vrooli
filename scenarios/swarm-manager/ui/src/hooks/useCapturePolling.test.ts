import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useCapturePolling } from "./useCapturePolling";
import { useCaptureStore, captureStoreInitialState } from "../stores/capture-store";
import type { Capture } from "../types";

function makeCapture(overrides: Partial<Capture> = {}): Capture {
  return {
    id: "cap-1",
    text: "test capture",
    attachments: [],
    created: new Date().toISOString(),
    status: "classifying",
    classification: null,
    ...overrides,
  };
}

function resetStore(captures: Capture[] = []) {
  const fetchCaptures = vi.fn().mockResolvedValue(undefined);
  useCaptureStore.setState({
    ...captureStoreInitialState,
    captures,
    fetchCaptures,
  });
  return fetchCaptures;
}

describe("useCapturePolling", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    resetStore();
  });

  it("does not poll when no captures are classifying", () => {
    const fetchCaptures = resetStore([
      makeCapture({ status: "classified" }),
    ]);

    renderHook(() => useCapturePolling());

    vi.advanceTimersByTime(10_000);
    expect(fetchCaptures).not.toHaveBeenCalled();
  });

  it("polls every 3s when captures are classifying", () => {
    const fetchCaptures = resetStore([
      makeCapture({ status: "classifying", created: new Date().toISOString() }),
    ]);

    renderHook(() => useCapturePolling());

    expect(fetchCaptures).not.toHaveBeenCalled();

    vi.advanceTimersByTime(3_000);
    expect(fetchCaptures).toHaveBeenCalledTimes(1);
    expect(fetchCaptures).toHaveBeenCalledWith({ force: true });

    vi.advanceTimersByTime(3_000);
    expect(fetchCaptures).toHaveBeenCalledTimes(2);
  });

  it("stops polling when all classifying captures are older than 60s", () => {
    const staleDate = new Date(Date.now() - 70_000).toISOString();
    const fetchCaptures = resetStore([
      makeCapture({ status: "classifying", created: staleDate }),
    ]);

    renderHook(() => useCapturePolling());

    vi.advanceTimersByTime(10_000);
    expect(fetchCaptures).not.toHaveBeenCalled();
  });

  it("does not poll when captures array is empty", () => {
    const fetchCaptures = resetStore([]);

    renderHook(() => useCapturePolling());

    vi.advanceTimersByTime(10_000);
    expect(fetchCaptures).not.toHaveBeenCalled();
  });
});

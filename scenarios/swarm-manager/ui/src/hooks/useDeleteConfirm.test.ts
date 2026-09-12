import { renderHook, act, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useDeleteConfirm, strongConfirmToken } from "./useDeleteConfirm";
import type { DeleteConfirmLevel } from "../types/settings";

// Drive the configured level from the test rather than the settings API.
let mockLevel: DeleteConfirmLevel = "simple";
vi.mock("./useRuntimeConfig", () => ({
  useRuntimeConfig: () => ({
    searchDebounceMs: 300,
    toastDurationMs: 5000,
    getDeleteConfirmLevel: () => mockLevel,
  }),
}));

afterEach(() => {
  mockLevel = "simple";
  vi.clearAllMocks();
});

describe("strongConfirmToken", () => {
  it("uses the entity name for single deletes", () => {
    expect(strongConfirmToken("my-scenario")).toBe("my-scenario");
    expect(strongConfirmToken("my-scenario", 1)).toBe("my-scenario");
  });

  it("uses a DELETE <count> token for bulk deletes", () => {
    expect(strongConfirmToken("ignored", 3)).toBe("DELETE 3");
  });
});

describe("useDeleteConfirm", () => {
  it("level=none invokes onConfirm immediately without opening a dialog", () => {
    mockLevel = "none";
    const onConfirm = vi.fn();
    const { result } = renderHook(() => useDeleteConfirm("capture"));

    act(() => result.current.requestDelete({ entityName: "note", onConfirm }));

    expect(onConfirm).toHaveBeenCalledTimes(1);
    expect(result.current.dialogProps.isOpen).toBe(false);
  });

  it("level=simple opens a dialog with no confirmation text", () => {
    mockLevel = "simple";
    const onConfirm = vi.fn();
    const { result } = renderHook(() => useDeleteConfirm("session"));

    act(() => result.current.requestDelete({ entityName: "sess-1", onConfirm }));

    expect(result.current.dialogProps.isOpen).toBe(true);
    expect(result.current.dialogProps.confirmationText).toBeUndefined();
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it("level=strong requires the entity name as confirmation text", () => {
    mockLevel = "strong";
    const onConfirm = vi.fn();
    const { result } = renderHook(() => useDeleteConfirm("scenario"));

    act(() => result.current.requestDelete({ entityName: "lpbs", onConfirm }));

    expect(result.current.dialogProps.isOpen).toBe(true);
    expect(result.current.dialogProps.confirmationText).toBe("lpbs");
  });

  it("level=strong bulk uses the DELETE <count> token", () => {
    mockLevel = "strong";
    const onConfirm = vi.fn();
    const { result } = renderHook(() => useDeleteConfirm("session"));

    act(() => result.current.requestDelete({ entityName: "ignored", count: 4, onConfirm }));

    expect(result.current.dialogProps.confirmationText).toBe("DELETE 4");
  });

  it("confirming from the dialog runs onConfirm and closes on success", async () => {
    mockLevel = "simple";
    const onConfirm = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() => useDeleteConfirm("backlog"));

    act(() => result.current.requestDelete({ entityName: "item", onConfirm }));
    act(() => result.current.dialogProps.onConfirm());

    await waitFor(() => expect(result.current.dialogProps.isOpen).toBe(false));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("keeps the dialog open if onConfirm rejects", async () => {
    mockLevel = "simple";
    const onConfirm = vi.fn().mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useDeleteConfirm("backlog"));

    act(() => result.current.requestDelete({ entityName: "item", onConfirm }));
    act(() => result.current.dialogProps.onConfirm());

    await waitFor(() => expect(result.current.dialogProps.isLoading).toBe(false));
    expect(result.current.dialogProps.isOpen).toBe(true);
  });

  it("derives a sensible default title and description", () => {
    mockLevel = "simple";
    const { result } = renderHook(() => useDeleteConfirm("session"));

    act(() => result.current.requestDelete({ entityName: "sess-1", onConfirm: vi.fn() }));

    expect(result.current.dialogProps.title).toBe("Delete Session");
    expect(result.current.dialogProps.description).toContain("sess-1");
  });
});

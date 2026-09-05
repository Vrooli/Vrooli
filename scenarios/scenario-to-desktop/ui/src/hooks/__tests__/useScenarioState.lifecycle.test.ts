/**
 * Tests for useScenarioState hook - cleanup, error handling, timestamps,
 * isSaving state, hash management, and lastSavedAt.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import {
  mockFetchScenarioState,
  mockSaveScenarioState,
  createWrapper,
  createMockScenarioState,
  createLoadStateResponse,
  createSaveStateResponse,
  defaultOptions,
} from "./useScenarioState.testUtils";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useScenarioState } from "../useScenarioState";
import type { FormState, SaveStateOptions } from "../../lib/api";

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(() => {
  vi.clearAllTimers();
});

describe("useScenarioState", () => {
  describe("cleanup on unmount", () => {
    it("clears pending debounce timeout on unmount", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(mockState),
      );
      mockSaveScenarioState.mockResolvedValue(createSaveStateResponse());

      const { result, unmount } = renderHook(
        () => useScenarioState(defaultOptions),
        { wrapper: createWrapper() },
      );

      // Wait for initial load
      await act(async () => {
        await vi.runAllTimersAsync();
      });

      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Start a debounced save
      act(() => {
        result.current.updateFormState({ app_display_name: "New Name" });
      });

      // Unmount before debounce completes
      unmount();

      // Advance timer past debounce period
      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });

      // Save should not have been called after unmount
      expect(mockSaveScenarioState).not.toHaveBeenCalled();

      vi.useRealTimers();
    });
  });

  describe("error handling", () => {
    it("calls onSaveError when save fails", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(mockState),
      );
      mockSaveScenarioState.mockRejectedValue(new Error("Save failed"));

      const onSaveError = vi.fn();

      const { result } = renderHook(
        () => useScenarioState({ ...defaultOptions, onSaveError }),
        { wrapper: createWrapper() },
      );

      // Wait for initial load
      await act(async () => {
        await vi.runAllTimersAsync();
      });

      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Trigger save
      act(() => {
        result.current.updateFormState({ app_display_name: "New Name" });
      });

      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });

      await waitFor(() => {
        expect(onSaveError).toHaveBeenCalled();
      });

      expect(
        (onSaveError.mock.calls?.[0]?.[0] as Error | undefined)?.message,
      ).toBe("Save failed");

      vi.useRealTimers();
    });

    it("sets saveError when save fails", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(mockState),
      );
      mockSaveScenarioState.mockRejectedValue(new Error("Save failed"));

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Wait for initial load
      await act(async () => {
        await vi.runAllTimersAsync();
      });

      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Trigger save
      act(() => {
        result.current.updateFormState({ app_display_name: "New Name" });
      });

      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });

      await waitFor(() => {
        expect(result.current.saveError?.message).toBe("Save failed");
      });

      vi.useRealTimers();
    });
  });

  describe("timestamps and artifacts", () => {
    it("returns timestamps from server state", async () => {
      const mockState = createMockScenarioState({
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-01-15T12:30:00Z",
      });
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(mockState),
      );

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      expect(result.current.timestamps).toEqual({
        createdAt: "2024-01-01T00:00:00Z",
        updatedAt: "2024-01-15T12:30:00Z",
      });
    });

    it("returns null timestamps when no state", async () => {
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(null));

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      expect(result.current.timestamps).toBeNull();
    });

    it("returns build artifacts from server state", async () => {
      const mockArtifacts = [
        {
          platform: "win",
          status: "ready" as const,
          file_path: "/output/app.exe",
        },
        { platform: "mac", status: "building" as const },
      ];
      const mockState = createMockScenarioState({
        build_artifacts: mockArtifacts,
      });
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(mockState),
      );

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      expect(result.current.buildArtifacts).toEqual(mockArtifacts);
    });

    it("returns empty array when no build artifacts", async () => {
      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(mockState),
      );

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      expect(result.current.buildArtifacts).toEqual([]);
    });

    it("returns stages from server state", async () => {
      const mockStages = {
        bundle: {
          stage: "bundle",
          status: "valid" as const,
          validated_at: "2024-01-15T12:00:00Z",
        },
        build: {
          stage: "build",
          status: "stale" as const,
          staleness_reason: "Config changed",
        },
      };
      const mockState = createMockScenarioState({
        stages: mockStages,
      });
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(mockState),
      );

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      expect(result.current.stages).toEqual(mockStages);
    });
  });

  describe("isSaving state", () => {
    it("is true while save is in progress", async () => {
      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(mockState),
      );

      // Make save hang
      let resolveSave: () => void = () => {};
      mockSaveScenarioState.mockImplementation(
        () =>
          new Promise((resolve) => {
            resolveSave = () => {
              resolve(createSaveStateResponse());
            };
          }),
      );

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Wait for initial load
      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      expect(result.current.isSaving).toBe(false);

      // Force save immediately
      let savePromise: Promise<void>;
      act(() => {
        savePromise = result.current.saveNow();
      });

      await waitFor(() => {
        expect(result.current.isSaving).toBe(true);
      });

      // Complete save
      await act(async () => {
        resolveSave();
        await savePromise;
      });

      await waitFor(() => {
        expect(result.current.isSaving).toBe(false);
      });
    });
  });

  describe("hash management", () => {
    it("updates localHash after successful save", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState({ hash: "original" });
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(mockState),
      );
      mockSaveScenarioState.mockResolvedValue(
        createSaveStateResponse({ hash: "updated" }),
      );

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Wait for initial load
      await act(async () => {
        await vi.runAllTimersAsync();
      });

      await waitFor(() => {
        expect(result.current.localHash).toBe("original");
      });

      // Trigger save
      act(() => {
        result.current.updateFormState({ app_display_name: "New Name" });
      });

      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });

      await waitFor(() => {
        expect(result.current.localHash).toBe("updated");
      });

      vi.useRealTimers();
    });

    it("sends expectedHash with save request", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState({ hash: "currenthash" });
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(mockState),
      );
      mockSaveScenarioState.mockResolvedValue(createSaveStateResponse());

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Wait for initial load
      await act(async () => {
        await vi.runAllTimersAsync();
      });

      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Trigger save
      act(() => {
        result.current.updateFormState({ app_display_name: "New Name" });
      });

      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });

      await waitFor(() => {
        expect(mockSaveScenarioState).toHaveBeenCalled();
      });

      const call = mockSaveScenarioState.mock.calls?.[0] as
        | [string, FormState, SaveStateOptions?]
        | undefined;
      const options = call?.[2];
      expect(options?.expectedHash).toBe("currenthash");

      vi.useRealTimers();
    });
  });

  describe("lastSavedAt", () => {
    it("updates lastSavedAt after successful save", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(mockState),
      );
      mockSaveScenarioState.mockResolvedValue(
        createSaveStateResponse({ updated_at: "2024-01-20T15:00:00Z" }),
      );

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Wait for initial load
      await act(async () => {
        await vi.runAllTimersAsync();
      });

      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Initially set from server state
      expect(result.current.lastSavedAt).toBe("2024-01-02T00:00:00Z");

      // Trigger save
      act(() => {
        result.current.updateFormState({ app_display_name: "New Name" });
      });

      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });

      await waitFor(() => {
        expect(result.current.lastSavedAt).toBe("2024-01-20T15:00:00Z");
      });

      vi.useRealTimers();
    });
  });
});

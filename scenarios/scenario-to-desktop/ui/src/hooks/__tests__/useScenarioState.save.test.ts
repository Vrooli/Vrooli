/**
 * Tests for useScenarioState hook - debounced save, save guards, stage result saving, and saveNow.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useScenarioState } from "../useScenarioState";
import type { FormState, SaveStateOptions } from "../../lib/api";
import {
  mockFetchScenarioState,
  mockSaveScenarioState,
  createWrapper,
  createMockScenarioState,
  createLoadStateResponse,
  createSaveStateResponse,
  defaultOptions,
} from "./useScenarioState.testUtils";

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(() => {
  vi.clearAllTimers();
});

describe("useScenarioState", () => {
  describe("debounced save behavior", () => {
    it("does not save immediately when updateFormState is called", async () => {
      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockSaveScenarioState.mockResolvedValue(createSaveStateResponse());

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Wait for initial load
      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Update form state
      act(() => {
        result.current.updateFormState({ app_display_name: "New Name" });
      });

      // Save should not be called immediately
      expect(mockSaveScenarioState).not.toHaveBeenCalled();
    });

    it("saves after debounce timeout", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockSaveScenarioState.mockResolvedValue(createSaveStateResponse());

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Wait for initial load using runAllTimers to let query resolve
      await act(async () => {
        await vi.runAllTimersAsync();
      });

      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Update form state
      act(() => {
        result.current.updateFormState({ app_display_name: "New Name" });
      });

      // Advance timer past debounce period (600ms)
      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });

      await waitFor(() => {
        expect(mockSaveScenarioState).toHaveBeenCalled();
      });

      vi.useRealTimers();
    });

    it("merges pending updates before saving", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
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

      // Multiple updates
      act(() => {
        result.current.updateFormState({ app_display_name: "Name" });
        result.current.updateFormState({ app_description: "Description" });
        result.current.updateFormState({ framework: "tauri" });
      });

      // Advance timer
      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });

      await waitFor(() => {
        expect(mockSaveScenarioState).toHaveBeenCalledTimes(1);
      });

      // Check that all updates were merged
      const savedData = mockSaveScenarioState.mock.calls?.[0]?.[1] as FormState | undefined;
      expect(savedData?.app_display_name).toBe("Name");
      expect(savedData?.app_description).toBe("Description");
      expect(savedData?.framework).toBe("tauri");

      vi.useRealTimers();
    });
  });

  describe("save guards (preventing saves before initial load)", () => {
    it("does not save when hasInitiallyLoaded is false", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      // Keep query pending
      mockFetchScenarioState.mockImplementation(
        () => new Promise(() => {})
      );
      mockSaveScenarioState.mockResolvedValue(createSaveStateResponse());

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Try to update before load completes
      act(() => {
        result.current.updateFormState({ app_display_name: "Should Not Save" });
      });

      // Advance timer
      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });

      // Save should not have been called
      expect(mockSaveScenarioState).not.toHaveBeenCalled();

      vi.useRealTimers();
    });

    it("prevents saveStageResult before initial load", async () => {
      mockFetchScenarioState.mockImplementation(
        () => new Promise(() => {})
      );
      mockSaveScenarioState.mockResolvedValue(createSaveStateResponse());

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Try to save stage result before load completes
      await act(async () => {
        await result.current.saveStageResult("bundle", { success: true });
      });

      // Save should not have been called
      expect(mockSaveScenarioState).not.toHaveBeenCalled();
    });
  });

  describe("stage result saving", () => {
    it("saves stage result immediately (no debounce)", async () => {
      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockSaveScenarioState.mockResolvedValue(createSaveStateResponse());

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Wait for initial load
      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Save stage result
      await act(async () => {
        await result.current.saveStageResult("bundle", { success: true, path: "/output" });
      });

      // Should have saved immediately
      expect(mockSaveScenarioState).toHaveBeenCalled();
      const call = mockSaveScenarioState.mock.calls?.[0] as [string, FormState, SaveStateOptions?] | undefined;
      const options = call?.[2] as SaveStateOptions | undefined;
      expect(options?.stageResults).toEqual({ bundle: { success: true, path: "/output" } });
    });

    it("includes form state updates in stage result save", async () => {
      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockSaveScenarioState.mockResolvedValue(createSaveStateResponse());

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Wait for initial load
      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Save stage result with form updates
      await act(async () => {
        await result.current.saveStageResult(
          "bundle",
          { success: true },
          { bundle_manifest_path: "/path/to/manifest.json" }
        );
      });

      const savedFormState = mockSaveScenarioState.mock.calls?.[0]?.[1] as FormState | undefined;
      expect(savedFormState?.bundle_manifest_path).toBe("/path/to/manifest.json");
    });

    it("passes additional options to stage result save", async () => {
      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockSaveScenarioState.mockResolvedValue(createSaveStateResponse());

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Wait for initial load
      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Save stage result with extra options
      await act(async () => {
        await result.current.saveStageResult(
          "build",
          { success: true },
          undefined,
          { buildArtifacts: [{ platform: "win", status: "ready" }] }
        );
      });

      const call = mockSaveScenarioState.mock.calls?.[0] as [string, FormState, SaveStateOptions?] | undefined;
      const options = call?.[2] as SaveStateOptions | undefined;
      expect(options?.buildArtifacts).toEqual([{ platform: "win", status: "ready" }]);
    });
  });

  describe("saveNow (force save)", () => {
    it("saves immediately without debounce", async () => {
      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockSaveScenarioState.mockResolvedValue(createSaveStateResponse());

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Wait for initial load
      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Update form state
      act(() => {
        result.current.updateFormState({ app_display_name: "New Name" });
      });

      // Force save
      await act(async () => {
        await result.current.saveNow();
      });

      // Should have saved immediately without waiting for debounce
      expect(mockSaveScenarioState).toHaveBeenCalled();
    });

    it("cancels pending debounced save", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
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

      // Update form state
      act(() => {
        result.current.updateFormState({ app_display_name: "New Name" });
      });

      // Force save
      await act(async () => {
        await result.current.saveNow();
      });

      // Clear mock
      mockSaveScenarioState.mockClear();

      // Advance timer past debounce period
      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });

      // Should not have saved again (debounced save was cancelled)
      expect(mockSaveScenarioState).not.toHaveBeenCalled();

      vi.useRealTimers();
    });
  });
});

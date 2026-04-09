/**
 * Tests for useScenarioState hook - conflict detection, staleness checking, and clear state.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useScenarioState } from "../useScenarioState";
import type { CheckStalenessResponse } from "../../lib/api";
import {
  mockFetchScenarioState,
  mockSaveScenarioState,
  mockDeleteScenarioState,
  mockCheckStateStaleness,
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
  describe("conflict detection and resolution", () => {
    it("calls onConflict when server reports conflict", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState();
      const conflictingState = createMockScenarioState({
        form_state: { app_display_name: "Conflicting Name" },
      });

      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockSaveScenarioState.mockResolvedValue(
        createSaveStateResponse({
          conflict: true,
          server_state: conflictingState,
        })
      );

      const onConflict = vi.fn();

      const { result } = renderHook(
        () => useScenarioState({ ...defaultOptions, onConflict }),
        { wrapper: createWrapper() }
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
        result.current.updateFormState({ app_display_name: "My Name" });
      });

      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });

      await waitFor(() => {
        expect(onConflict).toHaveBeenCalledWith(conflictingState);
      });

      vi.useRealTimers();
    });

    it("resolves conflict with server state", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState();
      const conflictingState = createMockScenarioState({
        hash: "serverhash",
        form_state: { app_display_name: "Server Name" },
      });

      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockSaveScenarioState.mockResolvedValue(
        createSaveStateResponse({
          conflict: true,
          server_state: conflictingState,
        })
      );

      const onStateLoaded = vi.fn();

      const { result } = renderHook(
        () => useScenarioState({ ...defaultOptions, onStateLoaded }),
        { wrapper: createWrapper() }
      );

      // Wait for initial load
      await act(async () => {
        await vi.runAllTimersAsync();
      });

      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Clear initial call
      onStateLoaded.mockClear();

      // Trigger save that causes conflict
      act(() => {
        result.current.updateFormState({ app_display_name: "My Name" });
      });

      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });

      // Wait for save mutation to complete
      await act(async () => {
        await vi.runAllTimersAsync();
      });

      // Resolve with server state
      act(() => {
        result.current.resolveConflict("server");
      });

      expect(result.current.formState?.app_display_name).toBe("Server Name");
      expect(result.current.localHash).toBe("serverhash");
      expect(onStateLoaded).toHaveBeenCalledWith(conflictingState);

      vi.useRealTimers();
    });

    it("resolves conflict with local state", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState();
      const conflictingState = createMockScenarioState({
        form_state: { app_display_name: "Server Name" },
      });

      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockSaveScenarioState
        .mockResolvedValueOnce(
          createSaveStateResponse({
            conflict: true,
            server_state: conflictingState,
          })
        )
        .mockResolvedValueOnce(createSaveStateResponse());

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

      // Trigger save that causes conflict
      act(() => {
        result.current.updateFormState({ app_display_name: "My Name" });
      });

      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });

      await waitFor(() => {
        expect(mockSaveScenarioState).toHaveBeenCalledTimes(1);
      });

      // Resolve with local state
      act(() => {
        result.current.resolveConflict("local");
      });

      // Should trigger a new save with local state
      await waitFor(() => {
        expect(mockSaveScenarioState).toHaveBeenCalledTimes(2);
      });

      vi.useRealTimers();
    });
  });

  describe("staleness checking", () => {
    it("checks staleness when checkStaleness is called", async () => {
      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockCheckStateStaleness.mockResolvedValue({
        valid: true,
        changed: false,
        pending_changes: [],
      } satisfies CheckStalenessResponse);

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Wait for initial load
      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Call checkStaleness
      await act(async () => {
        await result.current.checkStaleness({ manifest_path: "/path/to/manifest.json" });
      });

      expect(mockCheckStateStaleness).toHaveBeenCalledWith(
        "test-scenario",
        { manifest_path: "/path/to/manifest.json" }
      );
    });

    it("updates isStale and pendingChanges from staleness response", async () => {
      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockCheckStateStaleness.mockResolvedValue({
        valid: false,
        changed: true,
        pending_changes: [
          { change_type: "modified", affected_stage: "bundle", reason: "Manifest updated" },
        ],
      } satisfies CheckStalenessResponse);

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Wait for initial load
      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Call checkStaleness
      await act(async () => {
        await result.current.checkStaleness({ manifest_path: "/path/to/manifest.json" });
      });

      expect(result.current.isStale).toBe(true);
      expect(result.current.pendingChanges).toEqual([
        { change_type: "modified", affected_stage: "bundle", reason: "Manifest updated" },
      ]);
    });

    it("updates validationStatus from staleness response", async () => {
      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));

      const validationStatus = {
        scenario_name: "test-scenario",
        overall_status: "stale" as const,
        stages: {
          bundle: {
            stage: "bundle",
            status: "stale" as const,
            can_reuse: false,
            staleness_reason: "Config changed",
          },
        },
      };

      mockCheckStateStaleness.mockResolvedValue({
        valid: false,
        changed: true,
        status: validationStatus,
      } satisfies CheckStalenessResponse);

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      // Wait for initial load
      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Call checkStaleness
      await act(async () => {
        await result.current.checkStaleness({ manifest_path: "/path/to/manifest.json" });
      });

      expect(result.current.validationStatus).toEqual(validationStatus);
    });
  });

  describe("clear state", () => {
    it("calls deleteScenarioState and clears local state", async () => {
      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockDeleteScenarioState.mockResolvedValue(undefined);

      const onStateCleared = vi.fn();

      const { result } = renderHook(
        () => useScenarioState({ ...defaultOptions, onStateCleared }),
        { wrapper: createWrapper() }
      );

      // Wait for initial load
      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Clear initial callback
      onStateCleared.mockClear();

      // Clear state
      await act(async () => {
        await result.current.clearState();
      });

      expect(mockDeleteScenarioState).toHaveBeenCalledWith("test-scenario");
      expect(onStateCleared).toHaveBeenCalled();
      expect(result.current.formState).toBeNull();
      expect(result.current.localHash).toBeNull();
      expect(result.current.isStale).toBe(false);
    });

    it("cancels pending debounced save when clearing state", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockDeleteScenarioState.mockResolvedValue(undefined);
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

      // Start a debounced save
      act(() => {
        result.current.updateFormState({ app_display_name: "New Name" });
      });

      // Clear state before debounce completes
      await act(async () => {
        await result.current.clearState();
      });

      // Advance timer past debounce period
      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });

      // Save should not have been called
      expect(mockSaveScenarioState).not.toHaveBeenCalled();

      vi.useRealTimers();
    });
  });
});

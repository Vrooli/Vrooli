/**
 * Tests for useScenarioState hook - query behavior, state loading, and scenario switching.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import {
  mockFetchScenarioState,
  createWrapper,
  createMockScenarioState,
  createLoadStateResponse,
  defaultOptions,
} from "./useScenarioState.testUtils";
import { renderHook, waitFor } from "@testing-library/react";
import { useScenarioState, type UseScenarioStateOptions } from "../useScenarioState";

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(() => {
  vi.clearAllTimers();
});

describe("useScenarioState", () => {
  describe("initial state and query behavior", () => {
    it("returns loading state while fetching", () => {
      // Don't resolve the promise yet
      mockFetchScenarioState.mockImplementation(
        () => new Promise(() => {})
      );

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      expect(result.current.isLoading).toBe(true);
      expect(result.current.state).toBeNull();
      expect(result.current.formState).toBeNull();
      expect(result.current.hasInitiallyLoaded).toBe(false);
    });

    it("returns null state when scenario has no saved state", async () => {
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(null, { found: false })
      );

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false);
      });

      expect(result.current.state).toBeNull();
      expect(result.current.formState).toBeNull();
      expect(result.current.hasInitiallyLoaded).toBe(true);
    });

    it("returns state when scenario has saved state", async () => {
      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(mockState)
      );

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false);
      });

      expect(result.current.state).toEqual(mockState);
      expect(result.current.formState).toEqual(mockState.form_state);
      expect(result.current.serverHash).toBe("abc123");
      expect(result.current.localHash).toBe("abc123");
      expect(result.current.hasInitiallyLoaded).toBe(true);
    });

    it("handles query error", async () => {
      mockFetchScenarioState.mockRejectedValue(new Error("Network error"));

      const { result } = renderHook(() => useScenarioState(defaultOptions), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error?.message).toBe("Network error");
      expect(result.current.state).toBeNull();
    });

    it("does not fetch when enabled is false", () => {
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(null));

      renderHook(
        () => useScenarioState({ ...defaultOptions, enabled: false }),
        { wrapper: createWrapper() }
      );

      expect(mockFetchScenarioState).not.toHaveBeenCalled();
    });

    it("does not fetch when scenarioName is empty", () => {
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(null));

      renderHook(
        () => useScenarioState({ ...defaultOptions, scenarioName: "" }),
        { wrapper: createWrapper() }
      );

      expect(mockFetchScenarioState).not.toHaveBeenCalled();
    });
  });

  describe("state loading/hydration via onStateLoaded callback", () => {
    it("calls onStateLoaded when state is loaded from server", async () => {
      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(mockState)
      );

      const onStateLoaded = vi.fn();

      renderHook(
        () => useScenarioState({ ...defaultOptions, onStateLoaded }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(onStateLoaded).toHaveBeenCalledWith(mockState);
      });
    });

    it("calls onStateCleared when server returns no state", async () => {
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(null, { found: false })
      );

      const onStateCleared = vi.fn();

      renderHook(
        () => useScenarioState({ ...defaultOptions, onStateCleared }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(onStateCleared).toHaveBeenCalled();
      });
    });

    it("calls onManifestChanged when manifest has changed", async () => {
      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(
        createLoadStateResponse(mockState, {
          manifest_changed: true,
          current_hash: "currenthash",
          stored_hash: "storedhash",
        })
      );

      const onManifestChanged = vi.fn();

      renderHook(
        () => useScenarioState({ ...defaultOptions, onManifestChanged }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(onManifestChanged).toHaveBeenCalledWith("currenthash", "storedhash");
      });
    });
  });

  describe("scenario switching (state clearing)", () => {
    it("clears local state when scenario changes", async () => {
      const mockState1 = createMockScenarioState({
        scenario_name: "scenario-1",
        form_state: { app_display_name: "App 1" },
      });
      const mockState2 = createMockScenarioState({
        scenario_name: "scenario-2",
        form_state: { app_display_name: "App 2" },
      });

      mockFetchScenarioState
        .mockResolvedValueOnce(createLoadStateResponse(mockState1))
        .mockResolvedValueOnce(createLoadStateResponse(mockState2));

      const { result, rerender } = renderHook(
        (props: UseScenarioStateOptions) => useScenarioState(props),
        {
          wrapper: createWrapper(),
          initialProps: { ...defaultOptions, scenarioName: "scenario-1" },
        }
      );

      // Wait for first state to load
      await waitFor(() => {
        expect(result.current.formState?.app_display_name).toBe("App 1");
      });

      // Switch scenario
      rerender({ ...defaultOptions, scenarioName: "scenario-2" });

      // State should be cleared and hasInitiallyLoaded reset
      expect(result.current.hasInitiallyLoaded).toBe(false);

      // Wait for second state to load
      await waitFor(() => {
        expect(result.current.formState?.app_display_name).toBe("App 2");
      });
    });

    it("resets all local state flags when scenario changes", async () => {
      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));

      const { result, rerender } = renderHook(
        (props: UseScenarioStateOptions) => useScenarioState(props),
        {
          wrapper: createWrapper(),
          initialProps: { ...defaultOptions, scenarioName: "scenario-1" },
        }
      );

      // Wait for first state to load
      await waitFor(() => {
        expect(result.current.hasInitiallyLoaded).toBe(true);
      });

      // Switch scenario
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(null));
      rerender({ ...defaultOptions, scenarioName: "scenario-2" });

      // All state should be reset
      expect(result.current.hasInitiallyLoaded).toBe(false);
      expect(result.current.localHash).toBeNull();
      expect(result.current.isStale).toBe(false);
      expect(result.current.pendingChanges).toEqual([]);
    });
  });
});

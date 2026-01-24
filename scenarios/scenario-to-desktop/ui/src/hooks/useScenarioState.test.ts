/**
 * Tests for useScenarioState hook.
 * Tests server-side scenario state persistence, debounced saves,
 * race condition prevention, conflict detection, and staleness checking.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { useScenarioState, type UseScenarioStateOptions } from "./useScenarioState";
import type {
  LoadStateResponse,
  SaveStateResponse,
  CheckStalenessResponse,
  ScenarioState,
} from "../lib/api";

// Mock the API module
vi.mock("../lib/api", () => ({
  fetchScenarioState: vi.fn(),
  saveScenarioState: vi.fn(),
  deleteScenarioState: vi.fn(),
  checkStateStaleness: vi.fn(),
}));

// Import mocks after setting up vi.mock
import {
  fetchScenarioState,
  saveScenarioState,
  deleteScenarioState,
  checkStateStaleness,
} from "../lib/api";

const mockFetchScenarioState = fetchScenarioState as ReturnType<typeof vi.fn>;
const mockSaveScenarioState = saveScenarioState as ReturnType<typeof vi.fn>;
const mockDeleteScenarioState = deleteScenarioState as ReturnType<typeof vi.fn>;
const mockCheckStateStaleness = checkStateStaleness as ReturnType<typeof vi.fn>;

// Create a wrapper with QueryClientProvider
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

// Helper to create a ScenarioState for testing
function createMockScenarioState(overrides: Partial<ScenarioState> = {}): ScenarioState {
  return {
    scenario_name: "test-scenario",
    schema_version: 1,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-02T00:00:00Z",
    hash: "abc123",
    form_state: {
      app_display_name: "Test App",
      app_description: "Test Description",
      deployment_mode: "bundled",
      framework: "electron",
    },
    ...overrides,
  };
}

// Helper to create LoadStateResponse
function createLoadStateResponse(
  state: ScenarioState | null,
  overrides: Partial<LoadStateResponse> = {}
): LoadStateResponse {
  return {
    state,
    found: state !== null,
    ...overrides,
  };
}

// Helper to create SaveStateResponse
function createSaveStateResponse(
  overrides: Partial<SaveStateResponse> = {}
): SaveStateResponse {
  return {
    success: true,
    updated_at: new Date().toISOString(),
    hash: "newhash123",
    ...overrides,
  };
}

const defaultOptions: UseScenarioStateOptions = {
  scenarioName: "test-scenario",
  enabled: true,
  checkStaleness: false, // Disable by default to simplify tests
};

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
      const savedData = mockSaveScenarioState.mock.calls?.[0]?.[1];
      expect(savedData?.app_display_name).toBe("Name");
      expect(savedData.app_description).toBe("Description");
      expect(savedData.framework).toBe("tauri");

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
      const call = mockSaveScenarioState.mock.calls?.[0];
      const options = call?.[2];
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

      const savedFormState = mockSaveScenarioState.mock.calls?.[0]?.[1];
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

      const call = mockSaveScenarioState.mock.calls?.[0];
      const options = call?.[2];
      expect(options?.buildArtifacts).toEqual([{ platform: "win", status: "ready" }]);
    });
  });

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

  describe("cleanup on unmount", () => {
    it("clears pending debounce timeout on unmount", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockSaveScenarioState.mockResolvedValue(createSaveStateResponse());

      const { result, unmount } = renderHook(
        () => useScenarioState(defaultOptions),
        { wrapper: createWrapper() }
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
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockSaveScenarioState.mockRejectedValue(new Error("Save failed"));

      const onSaveError = vi.fn();

      const { result } = renderHook(
        () => useScenarioState({ ...defaultOptions, onSaveError }),
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
        result.current.updateFormState({ app_display_name: "New Name" });
      });

      await act(async () => {
        await vi.advanceTimersByTimeAsync(700);
      });

      await waitFor(() => {
        expect(onSaveError).toHaveBeenCalled();
      });

      expect(onSaveError.mock.calls?.[0]?.[0]?.message).toBe("Save failed");

      vi.useRealTimers();
    });

    it("sets saveError when save fails", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
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
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));

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
        { platform: "win", status: "ready" as const, file_path: "/output/app.exe" },
        { platform: "mac", status: "building" as const },
      ];
      const mockState = createMockScenarioState({
        build_artifacts: mockArtifacts,
      });
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));

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
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));

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
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));

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
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));

      // Make save hang
      let resolveSave: () => void = () => {};
      mockSaveScenarioState.mockImplementation(
        () =>
          new Promise((resolve) => {
            resolveSave = () => resolve(createSaveStateResponse());
          })
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
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockSaveScenarioState.mockResolvedValue(
        createSaveStateResponse({ hash: "updated" })
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

      const call = mockSaveScenarioState.mock.calls?.[0];
      const options = call?.[2];
      expect(options?.expectedHash).toBe("currenthash");

      vi.useRealTimers();
    });
  });

  describe("lastSavedAt", () => {
    it("updates lastSavedAt after successful save", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });

      const mockState = createMockScenarioState();
      mockFetchScenarioState.mockResolvedValue(createLoadStateResponse(mockState));
      mockSaveScenarioState.mockResolvedValue(
        createSaveStateResponse({ updated_at: "2024-01-20T15:00:00Z" })
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

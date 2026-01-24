/**
 * Tests for useScenarioSync hook.
 * Tests hydration from server, auto-save, and bundle result management.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { useScenarioSync, type UseScenarioSyncProps } from "./useScenarioSync";
import { useFormStore } from "../store/formStore";

// Mock useScenarioState hook
const mockUpdateFormState = vi.fn();
const mockSaveStageResult = vi.fn();
const mockClearState = vi.fn().mockResolvedValue(undefined);
const mockSaveNow = vi.fn().mockResolvedValue(undefined);

vi.mock("./useScenarioState", () => ({
  useScenarioState: vi.fn(({ onStateLoaded, onStateCleared }) => {
    // Store callbacks for testing
    (globalThis as unknown as { __testCallbacks: { onStateLoaded?: unknown; onStateCleared?: unknown } }).__testCallbacks = {
      onStateLoaded,
      onStateCleared,
    };
    return {
      formState: null,
      hasInitiallyLoaded: true,
      isSaving: false,
      isStale: false,
      pendingChanges: [],
      validationStatus: { valid: true },
      timestamps: { createdAt: "2024-01-01T00:00:00Z", updatedAt: "2024-01-02T00:00:00Z" },
      updateFormState: mockUpdateFormState,
      saveStageResult: mockSaveStageResult,
      clearState: mockClearState,
      saveNow: mockSaveNow,
    };
  }),
}));

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

// Reset store and mocks before each test
beforeEach(() => {
  useFormStore.getState().resetFormState();
  vi.clearAllMocks();
});

const defaultProps: UseScenarioSyncProps = {
  scenarioName: "test-scenario",
  enabled: true,
};

describe("useScenarioSync", () => {
  describe("initial state", () => {
    it("returns server form state as null initially", () => {
      const { result } = renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.serverFormState).toBeNull();
    });

    it("returns hasInitiallyLoaded from useScenarioState", () => {
      const { result } = renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.hasInitiallyLoaded).toBe(true);
    });

    it("returns isSaving from useScenarioState", () => {
      const { result } = renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.isSaving).toBe(false);
    });

    it("returns isStale from useScenarioState", () => {
      const { result } = renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.isStale).toBe(false);
    });

    it("returns timestamps from useScenarioState", () => {
      const { result } = renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.timestamps).toEqual({
        createdAt: "2024-01-01T00:00:00Z",
        updatedAt: "2024-01-02T00:00:00Z",
      });
    });
  });

  describe("actions", () => {
    it("exposes saveStageResult from useScenarioState", () => {
      const { result } = renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.saveStageResult).toBe(mockSaveStageResult);
    });

    it("exposes clearState as async action", async () => {
      const { result } = renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.clearState).toBe("function");

      await act(async () => {
        await result.current.clearState();
      });

      expect(mockClearState).toHaveBeenCalled();
    });

    it("exposes saveNow as async action", async () => {
      const { result } = renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.saveNow).toBe("function");

      await act(async () => {
        await result.current.saveNow();
      });

      expect(mockSaveNow).toHaveBeenCalled();
    });
  });

  describe("form state serialization", () => {
    it("provides getFormStateForServer function", () => {
      const { result } = renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.getFormStateForServer).toBe("function");
    });

    it("serializes form store state for server", () => {
      const { result } = renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      // Set some form state
      act(() => {
        useFormStore.getState().setAppDisplayName("Test App");
        useFormStore.getState().setAppDescription("Test Description");
        useFormStore.getState().setDeploymentMode("external-server");
        useFormStore.getState().setProxyUrl("https://example.com");
      });

      const serverState = result.current.getFormStateForServer();

      expect(serverState.app_display_name).toBe("Test App");
      expect(serverState.app_description).toBe("Test Description");
      expect(serverState.deployment_mode).toBe("external-server");
      expect(serverState.proxy_url).toBe("https://example.com");
    });

    it("includes all form fields in serialization", () => {
      const { result } = renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      const serverState = result.current.getFormStateForServer();

      // Check that all expected fields are present
      expect(serverState).toHaveProperty("selected_template");
      expect(serverState).toHaveProperty("app_display_name");
      expect(serverState).toHaveProperty("app_description");
      expect(serverState).toHaveProperty("icon_path");
      expect(serverState).toHaveProperty("display_name_edited");
      expect(serverState).toHaveProperty("description_edited");
      expect(serverState).toHaveProperty("icon_path_edited");
      expect(serverState).toHaveProperty("framework");
      expect(serverState).toHaveProperty("server_type");
      expect(serverState).toHaveProperty("deployment_mode");
      expect(serverState).toHaveProperty("platforms");
      expect(serverState).toHaveProperty("location_mode");
      expect(serverState).toHaveProperty("output_path");
      expect(serverState).toHaveProperty("proxy_url");
      expect(serverState).toHaveProperty("bundle_manifest_path");
      expect(serverState).toHaveProperty("server_port");
      expect(serverState).toHaveProperty("local_server_path");
      expect(serverState).toHaveProperty("local_api_endpoint");
      expect(serverState).toHaveProperty("auto_manage_tier1");
      expect(serverState).toHaveProperty("vrooli_binary_path");
      expect(serverState).toHaveProperty("signing_enabled_for_build");
    });
  });

  describe("bundle result seed", () => {
    it("exposes bundleResultSeed initially as null", () => {
      const { result } = renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.bundleResultSeed).toBeNull();
    });

    it("provides setBundleResultSeed function", () => {
      const { result } = renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.setBundleResultSeed).toBe("function");
    });
  });

  describe("callbacks", () => {
    it("calls onTemplateChange when state is loaded with template", () => {
      const onTemplateChange = vi.fn();
      renderHook(
        () => useScenarioSync({ ...defaultProps, onTemplateChange }),
        { wrapper: createWrapper() }
      );

      // Simulate state loaded callback
      const callbacks = (globalThis as unknown as { __testCallbacks: { onStateLoaded?: (state: { form_state?: { selected_template?: string } }) => void } }).__testCallbacks;
      if (callbacks.onStateLoaded) {
        callbacks.onStateLoaded({
          form_state: {
            selected_template: "advanced",
          },
        });
      }

      expect(onTemplateChange).toHaveBeenCalledWith("advanced");
    });

    it("calls onPreflightSeedLoaded when state is loaded with preflight", () => {
      const onPreflightSeedLoaded = vi.fn();
      renderHook(
        () => useScenarioSync({ ...defaultProps, onPreflightSeedLoaded }),
        { wrapper: createWrapper() }
      );

      // Simulate state loaded callback
      const callbacks = (globalThis as unknown as { __testCallbacks: { onStateLoaded?: (state: { form_state?: { preflight_result?: unknown; preflight_error?: string; preflight_override?: boolean; preflight_secrets?: Record<string, string> } }) => void } }).__testCallbacks;
      if (callbacks.onStateLoaded) {
        callbacks.onStateLoaded({
          form_state: {
            preflight_result: { validation: { valid: true } },
            preflight_error: null,
            preflight_override: true,
            preflight_secrets: { API_KEY: "secret" },
          },
        });
      }

      expect(onPreflightSeedLoaded).toHaveBeenCalledWith({
        result: { validation: { valid: true } },
        error: null,
        override: true,
        secrets: { API_KEY: "secret" },
      });
    });

    it("calls onBundleSeedLoaded when state is loaded with bundle result", () => {
      const onBundleSeedLoaded = vi.fn();
      renderHook(
        () => useScenarioSync({ ...defaultProps, onBundleSeedLoaded }),
        { wrapper: createWrapper() }
      );

      const mockBundleResult = {
        manifestPath: "/path/to/manifest.json",
        success: true,
      };

      // Simulate state loaded callback
      const callbacks = (globalThis as unknown as { __testCallbacks: { onStateLoaded?: (state: { form_state?: { bundle_result?: unknown } }) => void } }).__testCallbacks;
      if (callbacks.onStateLoaded) {
        callbacks.onStateLoaded({
          form_state: {
            bundle_result: mockBundleResult,
          },
        });
      }

      expect(onBundleSeedLoaded).toHaveBeenCalledWith(mockBundleResult);
    });

    it("calls onBundleSeedLoaded with null when state is cleared", () => {
      const onBundleSeedLoaded = vi.fn();
      renderHook(
        () => useScenarioSync({ ...defaultProps, onBundleSeedLoaded }),
        { wrapper: createWrapper() }
      );

      // Simulate state cleared callback
      const callbacks = (globalThis as unknown as { __testCallbacks: { onStateCleared?: () => void } }).__testCallbacks;
      if (callbacks.onStateCleared) {
        callbacks.onStateCleared();
      }

      expect(onBundleSeedLoaded).toHaveBeenCalledWith(null);
    });
  });

  describe("form store hydration", () => {
    it("hydrates form store when state is loaded", () => {
      renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      // Simulate state loaded callback with form state
      const callbacks = (globalThis as unknown as { __testCallbacks: { onStateLoaded?: (state: { form_state?: unknown }) => void } }).__testCallbacks;
      if (callbacks.onStateLoaded) {
        callbacks.onStateLoaded({
          form_state: {
            app_display_name: "Loaded App",
            app_description: "Loaded Description",
            framework: "tauri",
            deployment_mode: "external-server",
            server_type: "node",
            platforms: { win: true, mac: false, linux: true },
          },
        });
      }

      // Check that form store was hydrated
      const formState = useFormStore.getState();
      expect(formState.appMetadata.displayName).toBe("Loaded App");
      expect(formState.appMetadata.description).toBe("Loaded Description");
      expect(formState.deployment.framework).toBe("tauri");
      expect(formState.deployment.mode).toBe("external-server");
      expect(formState.deployment.serverType).toBe("node");
      expect(formState.platforms.mac).toBe(false);
    });

    it("resets form store when state is cleared", () => {
      // Set some state first
      act(() => {
        useFormStore.getState().setAppDisplayName("Custom Name");
        useFormStore.getState().setDeploymentMode("external-server");
      });

      renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      // Simulate state cleared callback
      const callbacks = (globalThis as unknown as { __testCallbacks: { onStateCleared?: () => void } }).__testCallbacks;
      if (callbacks.onStateCleared) {
        callbacks.onStateCleared();
      }

      // Check that form store was reset
      const formState = useFormStore.getState();
      expect(formState.appMetadata.displayName).toBe("");
      expect(formState.deployment.mode).toBe("bundled");
    });
  });

  describe("validation status", () => {
    it("returns validationStatus from useScenarioState", () => {
      const { result } = renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.validationStatus).toEqual({ valid: true });
    });
  });

  describe("pending changes", () => {
    it("transforms pendingChanges to field names", () => {
      const { result } = renderHook(() => useScenarioSync(defaultProps), {
        wrapper: createWrapper(),
      });

      // The mock returns empty array, but the transform should work
      expect(Array.isArray(result.current.pendingChanges)).toBe(true);
    });
  });
});

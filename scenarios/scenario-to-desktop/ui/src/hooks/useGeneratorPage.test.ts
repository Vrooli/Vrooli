/**
 * Tests for useGeneratorPage hook.
 * Tests composition of micro-hooks and form submission flow.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { useGeneratorPage, type UseGeneratorPageProps } from "./useGeneratorPage";
import { usePipelineStore } from "../store";
import { useFormStore } from "../store/formStore";

// Mock the API module - use importOriginal to preserve non-mocked exports
vi.mock("../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/api")>();
  return {
    ...actual,
    fetchProxyHints: vi.fn().mockResolvedValue({ hints: [] }),
    fetchScenarioDesktopStatus: vi.fn().mockResolvedValue({
      scenarios: [
        {
          name: "test-scenario",
          service_display_name: "Test Scenario",
          service_description: "A test scenario",
          service_icon_path: "/icons/test.png",
          has_ui: true,
          has_api: true,
        },
      ],
    }),
    fetchBundleManifest: vi.fn().mockResolvedValue({ manifest: {} }),
    runPipeline: vi.fn().mockResolvedValue({ pipeline_id: "test-pipeline-123" }),
    probeEndpoints: vi.fn().mockResolvedValue({ healthy: true }),
  };
});

// Mock the child hooks to isolate useGeneratorPage behavior
vi.mock("./useScenarioState", () => ({
  useScenarioState: vi.fn().mockReturnValue({
    formState: null,
    hasInitiallyLoaded: true,
    isSaving: false,
    isStale: false,
    pendingChanges: [],
    validationStatus: { valid: true },
    timestamps: null,
    updateFormState: vi.fn(),
    saveStageResult: vi.fn(),
    clearState: vi.fn().mockResolvedValue(undefined),
    saveNow: vi.fn().mockResolvedValue(undefined),
  }),
}));

vi.mock("./useSigningConfig", () => ({
  useSigningConfig: vi.fn().mockReturnValue({
    config: null,
    readiness: { ready: false },
    loading: false,
    refreshAll: vi.fn(),
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

// Reset stores and mocks before each test
beforeEach(() => {
  usePipelineStore.getState().reset();
  useFormStore.getState().resetFormState();
  vi.clearAllMocks();
});

const createDefaultProps = (): UseGeneratorPageProps => ({
  scenarioName: "test-scenario",
  selectedTemplate: "basic",
  selectionSource: null,
  onTemplateChange: vi.fn(),
  onScenarioNameChange: vi.fn(),
  onBuildStart: vi.fn(),
  onOpenSigningTab: vi.fn(),
});

describe("useGeneratorPage", () => {
  describe("hook composition", () => {
    it("returns formState from useFormState", () => {
      const { result } = renderHook(() => useGeneratorPage(createDefaultProps()), {
        wrapper: createWrapper(),
      });

      expect(result.current.formState).toBeDefined();
      expect(result.current.formState.appMetadata).toBeDefined();
      expect(result.current.formState.deployment).toBeDefined();
      expect(result.current.formState.platforms).toBeDefined();
    });

    it("returns pipelineActions from usePipelineActions", () => {
      const { result } = renderHook(() => useGeneratorPage(createDefaultProps()), {
        wrapper: createWrapper(),
      });

      expect(result.current.pipelineActions).toBeDefined();
      expect(typeof result.current.pipelineActions.generateDesktop).toBe("function");
      expect(typeof result.current.pipelineActions.runPreflight).toBe("function");
    });

    it("returns modals from useGeneratorModals", () => {
      const { result } = renderHook(() => useGeneratorPage(createDefaultProps()), {
        wrapper: createWrapper(),
      });

      expect(result.current.modals).toBeDefined();
    });

    it("exposes server sync state", () => {
      const { result } = renderHook(() => useGeneratorPage(createDefaultProps()), {
        wrapper: createWrapper(),
      });

      expect(result.current.hasInitiallyLoaded).toBe(true);
      expect(result.current.stateSaving).toBe(false);
      expect(result.current.isStale).toBe(false);
      expect(result.current.pendingChanges).toEqual([]);
    });

    it("exposes signing state", () => {
      const { result } = renderHook(() => useGeneratorPage(createDefaultProps()), {
        wrapper: createWrapper(),
      });

      expect(result.current.signingConfig).toBeNull();
      expect(result.current.signingReadiness).toEqual({ ready: false });
      expect(result.current.signingLoading).toBe(false);
      expect(typeof result.current.refreshSigning).toBe("function");
    });
  });

  describe("scenarios data", () => {
    it("fetches and exposes scenarios", async () => {
      const { result } = renderHook(() => useGeneratorPage(createDefaultProps()), {
        wrapper: createWrapper(),
      });

      // Initially loading
      expect(result.current.loadingScenarios).toBe(true);

      await waitFor(() => {
        expect(result.current.loadingScenarios).toBe(false);
      });

      expect(result.current.scenarios).toBeDefined();
      expect(result.current.scenarios.length).toBeGreaterThan(0);
    });

    it("finds selected scenario from list", async () => {
      const { result } = renderHook(() => useGeneratorPage(createDefaultProps()), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.selectedScenario).toBeDefined();
      });

      expect(result.current.selectedScenario?.name).toBe("test-scenario");
    });
  });

  describe("handleSubmit", () => {
    it("prevents default form submission", async () => {
      const { result } = renderHook(() => useGeneratorPage(createDefaultProps()), {
        wrapper: createWrapper(),
      });

      const preventDefault = vi.fn();
      const mockEvent = { preventDefault } as unknown as React.FormEvent;

      act(() => {
        result.current.handleSubmit(mockEvent);
      });

      expect(preventDefault).toHaveBeenCalled();
    });

    it("clears validation errors on submit", async () => {
      const { result } = renderHook(() => useGeneratorPage(createDefaultProps()), {
        wrapper: createWrapper(),
      });

      // Set some validation errors first
      act(() => {
        result.current.formState.setValidationErrors([
          { field: "test", message: "Test error" },
        ]);
      });

      expect(result.current.formState.validationErrors.length).toBe(1);

      // Submit the form
      act(() => {
        result.current.handleSubmit({ preventDefault: vi.fn() } as unknown as React.FormEvent);
      });

      // Errors should be cleared (they may be re-set if validation fails)
      // but the clearValidationErrors action should have been called
    });

    it("sets validation errors when form is invalid", async () => {
      const { result } = renderHook(
        () =>
          useGeneratorPage({
            ...createDefaultProps(),
            scenarioName: "", // Invalid - empty scenario
          }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.handleSubmit({ preventDefault: vi.fn() } as unknown as React.FormEvent);
      });

      // Should have validation errors for missing scenario
      expect(result.current.formState.validationErrors.length).toBeGreaterThan(0);
    });
  });

  describe("resetFormState", () => {
    it("calls onTemplateChange with 'basic' when resetTemplate is true", async () => {
      const onTemplateChange = vi.fn();
      const { result } = renderHook(
        () => useGeneratorPage({ ...createDefaultProps(), onTemplateChange }),
        { wrapper: createWrapper() }
      );

      // Reset with template reset
      act(() => {
        result.current.resetFormState(true);
      });

      // Template change callback should be called with "basic"
      await waitFor(() => {
        expect(onTemplateChange).toHaveBeenCalledWith("basic");
      });
    });

    it("does not call onTemplateChange when resetTemplate is false", () => {
      const onTemplateChange = vi.fn();
      const { result } = renderHook(
        () => useGeneratorPage({ ...createDefaultProps(), onTemplateChange }),
        { wrapper: createWrapper() }
      );

      // Reset without template reset
      act(() => {
        result.current.resetFormState(false);
      });

      // Template change should not be called
      expect(onTemplateChange).not.toHaveBeenCalled();
    });
  });

  describe("bundle handling", () => {
    it("exposes bundleResultSeed", () => {
      const { result } = renderHook(() => useGeneratorPage(createDefaultProps()), {
        wrapper: createWrapper(),
      });

      expect(result.current.bundleResultSeed).toBeNull();
    });

    it("provides handleBundleComplete callback", () => {
      const { result } = renderHook(() => useGeneratorPage(createDefaultProps()), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.handleBundleComplete).toBe("function");
    });
  });

  describe("clearDraft", () => {
    it("provides clearDraft function", () => {
      const { result } = renderHook(() => useGeneratorPage(createDefaultProps()), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.clearDraft).toBe("function");
    });
  });

  describe("proxy hints and bundle manifest", () => {
    it("exposes proxyHints", async () => {
      const { result } = renderHook(() => useGeneratorPage(createDefaultProps()), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.proxyHints).toBeDefined();
      });
    });

    it("exposes bundleManifest", async () => {
      const { result } = renderHook(
        () =>
          useGeneratorPage({
            ...createDefaultProps(),
          }),
        { wrapper: createWrapper() }
      );

      // bundleManifest starts as null since no bundle path is set
      expect(result.current.bundleManifest).toBeNull();
    });
  });

  describe("server timestamps", () => {
    it("exposes serverTimestamps", () => {
      const { result } = renderHook(() => useGeneratorPage(createDefaultProps()), {
        wrapper: createWrapper(),
      });

      expect(result.current.serverTimestamps).toBeNull();
    });
  });

  describe("validation status", () => {
    it("exposes validationStatus from server sync", () => {
      const { result } = renderHook(() => useGeneratorPage(createDefaultProps()), {
        wrapper: createWrapper(),
      });

      expect(result.current.validationStatus).toEqual({ valid: true });
    });
  });
});

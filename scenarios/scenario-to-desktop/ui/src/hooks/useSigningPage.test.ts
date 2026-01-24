/**
 * Tests for useSigningPage hook.
 * Tests signing page state management and mutations.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { useSigningPage } from "./useSigningPage";
import type {
  SigningConfig,
  SigningConfigResponse,
  SigningReadinessResponse,
  ToolDetectionResult,
  DiscoveredCertificate,
} from "../lib/api";
import type { ScenariosResponse } from "../components/scenario-inventory/types";

// Mock the API module
vi.mock("../lib/api", () => ({
  fetchSigningConfig: vi.fn(),
  saveSigningConfig: vi.fn(),
  validateSigningConfig: vi.fn(),
  checkSigningReadiness: vi.fn(),
  fetchSigningPrerequisites: vi.fn(),
  deleteSigningConfig: vi.fn(),
  discoverCertificates: vi.fn(),
  generateLinuxSigningKey: vi.fn(),
  fetchScenarioDesktopStatus: vi.fn(),
}));

// Import mocks after setting up vi.mock
import {
  fetchSigningConfig,
  saveSigningConfig,
  validateSigningConfig,
  checkSigningReadiness,
  fetchSigningPrerequisites,
  deleteSigningConfig,
  discoverCertificates,
  generateLinuxSigningKey,
  fetchScenarioDesktopStatus,
} from "../lib/api";

const mockFetchSigningConfig = fetchSigningConfig as ReturnType<typeof vi.fn>;
const mockSaveSigningConfig = saveSigningConfig as ReturnType<typeof vi.fn>;
const mockValidateSigningConfig = validateSigningConfig as ReturnType<typeof vi.fn>;
const mockCheckSigningReadiness = checkSigningReadiness as ReturnType<typeof vi.fn>;
const mockFetchSigningPrerequisites = fetchSigningPrerequisites as ReturnType<typeof vi.fn>;
const mockDeleteSigningConfig = deleteSigningConfig as ReturnType<typeof vi.fn>;
const mockDiscoverCertificates = discoverCertificates as ReturnType<typeof vi.fn>;
const mockGenerateLinuxSigningKey = generateLinuxSigningKey as ReturnType<typeof vi.fn>;
const mockFetchScenarioDesktopStatus = fetchScenarioDesktopStatus as ReturnType<typeof vi.fn>;

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

// Helper to create mock signing config
function createMockSigningConfig(overrides: Partial<SigningConfig> = {}): SigningConfig {
  return {
    enabled: true,
    windows: {
      enabled: true,
      certificate_path: "/path/to/cert.pfx",
    },
    macos: { enabled: false },
    linux: { enabled: false },
    ...overrides,
  };
}

// Helper to create mock config response
function createMockConfigResponse(config: SigningConfig | null = null): SigningConfigResponse {
  return {
    config,
    exists: config !== null,
  };
}

// Helper to create mock readiness response
function createMockReadinessResponse(ready: boolean, issues: string[] = []): SigningReadinessResponse {
  return {
    ready,
    issues,
    platforms: {
      windows: { ready, issues: [] },
      macos: { ready: false, issues: ["Not configured"] },
      linux: { ready: false, issues: ["Not configured"] },
    },
  };
}

// Helper to create mock scenarios response
function createMockScenariosResponse(): ScenariosResponse {
  return {
    scenarios: [
      { name: "test-scenario", path: "/path/to/scenario", status: "ready" },
      { name: "other-scenario", path: "/path/to/other", status: "ready" },
    ],
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  // Set up default mock responses
  mockFetchScenarioDesktopStatus.mockResolvedValue(createMockScenariosResponse());
  mockFetchSigningPrerequisites.mockResolvedValue({ tools: [] });
  mockFetchSigningConfig.mockResolvedValue(createMockConfigResponse(null));
  mockCheckSigningReadiness.mockResolvedValue(createMockReadinessResponse(false));
});

afterEach(() => {
  vi.clearAllTimers();
});

describe("useSigningPage", () => {
  describe("initial state", () => {
    it("returns empty selected scenario initially", () => {
      const { result } = renderHook(() => useSigningPage({}), {
        wrapper: createWrapper(),
      });

      expect(result.current.selectedScenario).toBe("");
      expect(result.current.localConfig).toEqual({ enabled: false });
      expect(result.current.hasUnsavedChanges).toBe(false);
    });

    it("uses initialScenario when provided", async () => {
      const { result } = renderHook(
        () => useSigningPage({ initialScenario: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.selectedScenario).toBe("test-scenario");
      });
    });

    it("loads scenarios on mount", async () => {
      const { result } = renderHook(() => useSigningPage({}), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.scenarios).toHaveLength(2);
        expect(result.current.scenarios[0].name).toBe("test-scenario");
      });
    });
  });

  describe("scenario selection", () => {
    it("calls onScenarioChange when scenario changes", async () => {
      const onScenarioChange = vi.fn();
      const { result } = renderHook(
        () => useSigningPage({ onScenarioChange }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.setSelectedScenario("new-scenario");
      });

      expect(onScenarioChange).toHaveBeenCalledWith("new-scenario");
      expect(result.current.selectedScenario).toBe("new-scenario");
    });

    it("fetches config when scenario is selected", async () => {
      const config = createMockSigningConfig();
      mockFetchSigningConfig.mockResolvedValue(createMockConfigResponse(config));

      const { result } = renderHook(
        () => useSigningPage({ initialScenario: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(mockFetchSigningConfig).toHaveBeenCalledWith("test-scenario");
        expect(result.current.localConfig.enabled).toBe(true);
      });
    });
  });

  describe("config changes", () => {
    it("updates localConfig when handleConfigChange is called", async () => {
      // Don't set initialScenario to avoid config loading race conditions
      const { result } = renderHook(
        () => useSigningPage({}),
        { wrapper: createWrapper() }
      );

      // Initial state before any changes
      expect(result.current.localConfig.enabled).toBe(false);
      expect(result.current.hasUnsavedChanges).toBe(false);

      act(() => {
        result.current.handleConfigChange({ enabled: true });
      });

      // State should update synchronously
      expect(result.current.localConfig.enabled).toBe(true);
      expect(result.current.hasUnsavedChanges).toBe(true);
    });

    it("marks hasUnsavedChanges true after config change", async () => {
      const { result } = renderHook(
        () => useSigningPage({ initialScenario: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      expect(result.current.hasUnsavedChanges).toBe(false);

      act(() => {
        result.current.handleConfigChange({ enabled: true });
      });

      expect(result.current.hasUnsavedChanges).toBe(true);
    });
  });

  describe("save mutation", () => {
    it("calls saveSigningConfig when handleSave is called", async () => {
      mockSaveSigningConfig.mockResolvedValue({});

      const { result } = renderHook(
        () => useSigningPage({ initialScenario: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.selectedScenario).toBe("test-scenario");
      });

      act(() => {
        result.current.handleConfigChange({ enabled: true });
      });

      await act(async () => {
        result.current.handleSave();
      });

      await waitFor(() => {
        expect(mockSaveSigningConfig).toHaveBeenCalled();
      });
    });

    it("resets hasUnsavedChanges after successful save", async () => {
      mockSaveSigningConfig.mockResolvedValue({});

      // Select a scenario first so save has a target
      const { result } = renderHook(
        () => useSigningPage({}),
        { wrapper: createWrapper() }
      );

      // Set scenario without triggering config fetch
      act(() => {
        result.current.setSelectedScenario("test-scenario");
      });

      // Make a change
      act(() => {
        result.current.handleConfigChange({ enabled: true });
      });

      expect(result.current.hasUnsavedChanges).toBe(true);

      // Save the config
      await act(async () => {
        result.current.handleSave();
      });

      // Wait for save to complete
      await waitFor(() => {
        expect(mockSaveSigningConfig).toHaveBeenCalled();
      });

      await waitFor(() => {
        expect(result.current.hasUnsavedChanges).toBe(false);
      });
    });
  });

  describe("validation", () => {
    it("calls validateSigningConfig when handleValidate is called", async () => {
      mockValidateSigningConfig.mockResolvedValue({
        valid: true,
        platforms: {},
      });

      const { result } = renderHook(
        () => useSigningPage({ initialScenario: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.selectedScenario).toBe("test-scenario");
      });

      await act(async () => {
        result.current.handleValidate();
      });

      await waitFor(() => {
        expect(mockValidateSigningConfig).toHaveBeenCalledWith("test-scenario");
      });
    });
  });

  describe("certificate discovery", () => {
    it("defaults to windows platform for discovery", () => {
      const { result } = renderHook(() => useSigningPage({}), {
        wrapper: createWrapper(),
      });

      expect(result.current.discoverPlatform).toBe("windows");
    });

    it("updates discover platform", () => {
      const { result } = renderHook(() => useSigningPage({}), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.setDiscoverPlatform("macos");
      });

      expect(result.current.discoverPlatform).toBe("macos");
    });

    it("calls discoverCertificates when onDiscover is called", async () => {
      const mockCerts: DiscoveredCertificate[] = [
        {
          fingerprint: "ABC123",
          subject: "Test Cert",
          issuer: "Test CA",
          is_expired: false,
        },
      ];
      mockDiscoverCertificates.mockResolvedValue({ certificates: mockCerts });

      const { result } = renderHook(() => useSigningPage({}), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.onDiscover();
      });

      await waitFor(() => {
        expect(mockDiscoverCertificates).toHaveBeenCalledWith("windows");
        expect(result.current.discovered).toHaveLength(1);
      });
    });
  });

  describe("prerequisites", () => {
    it("loads prerequisites on mount", async () => {
      const mockTools: ToolDetectionResult[] = [
        { tool: "signtool", platform: "windows", installed: true },
      ];
      mockFetchSigningPrerequisites.mockResolvedValue({ tools: mockTools });

      const { result } = renderHook(() => useSigningPage({}), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.prerequisitesData).toHaveLength(1);
        expect(result.current.prerequisitesData[0].tool).toBe("signtool");
      });
    });

    it("refetches prerequisites when refetchPrerequisites is called", async () => {
      mockFetchSigningPrerequisites.mockResolvedValue({ tools: [] });

      const { result } = renderHook(() => useSigningPage({}), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(mockFetchSigningPrerequisites).toHaveBeenCalled();
      });

      // Update mock for second call
      mockFetchSigningPrerequisites.mockResolvedValue({
        tools: [{ tool: "gpg", platform: "linux", installed: true }],
      });

      await act(async () => {
        result.current.refetchPrerequisites();
      });

      await waitFor(() => {
        expect(mockFetchSigningPrerequisites).toHaveBeenCalledTimes(2);
      });
    });
  });

  describe("delete mutation", () => {
    it("calls deleteSigningConfig when handleDelete is called and confirmed", async () => {
      mockDeleteSigningConfig.mockResolvedValue({});
      // Mock window.confirm to return true
      vi.spyOn(window, "confirm").mockReturnValue(true);

      const { result } = renderHook(
        () => useSigningPage({ initialScenario: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.selectedScenario).toBe("test-scenario");
      });

      await act(async () => {
        result.current.handleDelete();
      });

      await waitFor(() => {
        expect(mockDeleteSigningConfig).toHaveBeenCalled();
      });
    });

    it("does not call deleteSigningConfig when handleDelete is cancelled", async () => {
      // Mock window.confirm to return false
      vi.spyOn(window, "confirm").mockReturnValue(false);

      const { result } = renderHook(
        () => useSigningPage({ initialScenario: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.selectedScenario).toBe("test-scenario");
      });

      await act(async () => {
        result.current.handleDelete();
      });

      expect(mockDeleteSigningConfig).not.toHaveBeenCalled();
    });
  });

  describe("applyCertificate", () => {
    it("updates local config and marks as unsaved", () => {
      const { result } = renderHook(() => useSigningPage({}), {
        wrapper: createWrapper(),
      });

      const cert: DiscoveredCertificate = {
        fingerprint: "ABC123",
        subject: "CN=Test Cert",
        issuer: "CN=Test CA",
        is_expired: false,
      };

      act(() => {
        result.current.setDiscoverPlatform("windows");
      });

      act(() => {
        result.current.applyCertificate(cert);
      });

      expect(result.current.hasUnsavedChanges).toBe(true);
    });
  });
});

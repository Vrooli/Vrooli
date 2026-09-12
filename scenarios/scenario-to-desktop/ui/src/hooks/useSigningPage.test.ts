/**
 * Tests for useSigningPage hook.
 * Tests signing page state management and mutations.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { useSigningPage } from "./useSigningPage";
import type { SigningConfig, DiscoveredCertificate } from "../domain/signing";
import type { ScenariosResponse } from "../components/scenario-inventory/types";

// Scenario inventory remains a separate generated-client surface.
vi.mock("../lib/api", () => ({
  fetchScenarioDesktopStatus: vi.fn(),
}));
vi.mock("../lib/api/connect", () => ({
  signingConnectClient: {
    getSigningConfig: vi.fn(),
    putSigningConfig: vi.fn(),
    validateSigningConfig: vi.fn(),
    getSigningReadiness: vi.fn(),
    listSigningPrerequisites: vi.fn(),
    deleteSigningConfig: vi.fn(),
    discoverSigningCertificates: vi.fn(),
    generateLinuxSigningKey: vi.fn(),
  },
}));

import { fetchScenarioDesktopStatus } from "../lib/api";
import { signingConnectClient } from "../lib/api/connect";

const mockFetchSigningConfig =
  signingConnectClient.getSigningConfig as ReturnType<typeof vi.fn>;
const mockSaveSigningConfig =
  signingConnectClient.putSigningConfig as ReturnType<typeof vi.fn>;
const mockValidateSigningConfig =
  signingConnectClient.validateSigningConfig as ReturnType<typeof vi.fn>;
const mockCheckSigningReadiness =
  signingConnectClient.getSigningReadiness as ReturnType<typeof vi.fn>;
const mockFetchSigningPrerequisites =
  signingConnectClient.listSigningPrerequisites as ReturnType<typeof vi.fn>;
const mockDeleteSigningConfig =
  signingConnectClient.deleteSigningConfig as ReturnType<typeof vi.fn>;
const mockDiscoverCertificates =
  signingConnectClient.discoverSigningCertificates as ReturnType<typeof vi.fn>;
const _mockGenerateLinuxSigningKey =
  signingConnectClient.generateLinuxSigningKey as ReturnType<typeof vi.fn>;
const mockFetchScenarioDesktopStatus = fetchScenarioDesktopStatus as ReturnType<
  typeof vi.fn
>;

// Create a wrapper with QueryClientProvider
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(
      QueryClientProvider,
      { client: queryClient },
      children,
    );
  };
}

// Helper to create mock signing config
function createMockSigningConfig(
  overrides: Partial<SigningConfig> = {},
): SigningConfig {
  return {
    enabled: true,
    windows: {
      certificate_source: "file",
      certificate_file: "/path/to/cert.pfx",
    },
    macos: {
      identity: "",
      team_id: "",
      hardened_runtime: false,
      notarize: false,
    },
    linux: {
      gpg_key_id: "",
    },
    ...overrides,
  };
}

// Helper to create mock scenarios response
function createMockScenariosResponse(): ScenariosResponse {
  return {
    scenarios: [
      {
        name: "test-scenario",
        has_desktop: true,
        desktop_path: "/path/to/scenario",
      },
      {
        name: "other-scenario",
        has_desktop: true,
        desktop_path: "/path/to/other",
      },
    ],
    stats: { total: 2, with_desktop: 2, built: 0, web_only: 0 },
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  // Set up default mock responses
  mockFetchScenarioDesktopStatus.mockResolvedValue(
    createMockScenariosResponse(),
  );
  mockFetchSigningPrerequisites.mockResolvedValue({ tools: [] });
  mockFetchSigningConfig.mockResolvedValue({ config: undefined });
  mockCheckSigningReadiness.mockResolvedValue({
    ready: false,
    message: "Not configured",
    platforms: [],
  });
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
        { wrapper: createWrapper() },
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
        expect(result.current.scenarios?.[0]?.name).toBe("test-scenario");
      });
    });
  });

  describe("scenario selection", () => {
    it("calls onScenarioChange when scenario changes", async () => {
      const onScenarioChange = vi.fn();
      const { result } = renderHook(
        () => useSigningPage({ onScenarioChange }),
        { wrapper: createWrapper() },
      );

      act(() => {
        result.current.setSelectedScenario("new-scenario");
      });

      expect(onScenarioChange).toHaveBeenCalledWith("new-scenario");
      expect(result.current.selectedScenario).toBe("new-scenario");
    });

    it("fetches config when scenario is selected", async () => {
      const config = createMockSigningConfig();
      mockFetchSigningConfig.mockResolvedValue({
        config: {
          enabled: config.enabled,
          windows: {
            enabled: true,
            certificateSource: 1,
            certificatePath: config.windows?.certificate_file,
          },
        },
      });

      const { result } = renderHook(
        () => useSigningPage({ initialScenario: "test-scenario" }),
        { wrapper: createWrapper() },
      );

      await waitFor(() => {
        expect(mockFetchSigningConfig).toHaveBeenCalledWith({
          scenarioName: "test-scenario",
        });
        expect(result.current.localConfig.enabled).toBe(true);
      });
    });
  });

  describe("config changes", () => {
    it("updates localConfig when handleConfigChange is called", async () => {
      // Don't set initialScenario to avoid config loading race conditions
      const { result } = renderHook(() => useSigningPage({}), {
        wrapper: createWrapper(),
      });

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
        { wrapper: createWrapper() },
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
        { wrapper: createWrapper() },
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
      const { result } = renderHook(() => useSigningPage({}), {
        wrapper: createWrapper(),
      });

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
        errors: [],
        warnings: [],
      });

      const { result } = renderHook(
        () => useSigningPage({ initialScenario: "test-scenario" }),
        { wrapper: createWrapper() },
      );

      await waitFor(() => {
        expect(result.current.selectedScenario).toBe("test-scenario");
      });

      await act(async () => {
        result.current.handleValidate();
      });

      await waitFor(() => {
        expect(mockValidateSigningConfig).toHaveBeenCalledWith(
          expect.objectContaining({ scenarioName: "test-scenario" }),
        );
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
          id: "ABC123",
          subject: "Test Cert",
          issuer: "Test CA",
          is_expired: false,
        },
      ];
      mockDiscoverCertificates.mockResolvedValue({
        certificates: mockCerts.map((cert) => ({
          id: cert.id,
          subject: cert.subject,
          issuer: cert.issuer,
          expired: cert.is_expired,
          platform: 1,
        })),
      });

      const { result } = renderHook(() => useSigningPage({}), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        result.current.onDiscover();
      });

      await waitFor(() => {
        expect(mockDiscoverCertificates).toHaveBeenCalledWith({ platform: 1 });
        expect(result.current.discovered).toHaveLength(1);
      });
    });
  });

  describe("prerequisites", () => {
    it("loads prerequisites on mount", async () => {
      mockFetchSigningPrerequisites.mockResolvedValue({
        tools: [{ tool: "signtool", platform: 1, installed: true }],
      });

      const { result } = renderHook(() => useSigningPage({}), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.prerequisitesData).toHaveLength(1);
        expect(result.current.prerequisitesData?.[0]?.tool).toBe("signtool");
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
        tools: [{ tool: "gpg", platform: 3, installed: true }],
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
        { wrapper: createWrapper() },
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
        { wrapper: createWrapper() },
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
        id: "ABC123",
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

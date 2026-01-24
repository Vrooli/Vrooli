/**
 * Tests for useSigningConfig hook.
 * Tests signing configuration and readiness queries.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { useSigningConfig } from "./useSigningConfig";
import type {
  SigningConfig,
  SigningConfigResponse,
  SigningReadinessResponse,
} from "../lib/api";

// Mock the API module
vi.mock("../lib/api", () => ({
  fetchSigningConfig: vi.fn(),
  checkSigningReadiness: vi.fn(),
}));

// Import mocks after setting up vi.mock
import { fetchSigningConfig, checkSigningReadiness } from "../lib/api";

const mockFetchSigningConfig = fetchSigningConfig as ReturnType<typeof vi.fn>;
const mockCheckSigningReadiness = checkSigningReadiness as ReturnType<typeof vi.fn>;

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
    macos: {
      enabled: false,
    },
    linux: {
      enabled: false,
    },
    ...overrides,
  };
}

// Helper to create mock config response
function createMockConfigResponse(
  config: SigningConfig | null = null
): SigningConfigResponse {
  return {
    config,
    exists: config !== null,
  };
}

// Helper to create mock readiness response
function createMockReadinessResponse(
  ready: boolean,
  issues: string[] = []
): SigningReadinessResponse {
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

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(() => {
  vi.clearAllTimers();
});

describe("useSigningConfig", () => {
  describe("initial state", () => {
    it("returns loading states initially", () => {
      mockFetchSigningConfig.mockImplementation(() => new Promise(() => {}));
      mockCheckSigningReadiness.mockImplementation(() => new Promise(() => {}));

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      expect(result.current.loading).toBe(true);
      expect(result.current.configLoading).toBe(true);
      expect(result.current.readinessLoading).toBe(true);
    });

    it("config starts as null", () => {
      mockFetchSigningConfig.mockImplementation(() => new Promise(() => {}));
      mockCheckSigningReadiness.mockImplementation(() => new Promise(() => {}));

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      expect(result.current.config).toBeNull();
    });

    it("enabledForBuild starts as false", () => {
      mockFetchSigningConfig.mockImplementation(() => new Promise(() => {}));
      mockCheckSigningReadiness.mockImplementation(() => new Promise(() => {}));

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      expect(result.current.enabledForBuild).toBe(false);
    });
  });

  describe("config query", () => {
    it("fetches signing config for scenario", async () => {
      const mockConfig = createMockSigningConfig();
      mockFetchSigningConfig.mockResolvedValue(createMockConfigResponse(mockConfig));
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(true)
      );

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.configLoading).toBe(false);
      });

      expect(mockFetchSigningConfig).toHaveBeenCalledWith("test-scenario");
      expect(result.current.config).toEqual(mockConfig);
    });

    it("returns null config when none exists", async () => {
      mockFetchSigningConfig.mockResolvedValue(createMockConfigResponse(null));
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(false)
      );

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.configLoading).toBe(false);
      });

      expect(result.current.config).toBeNull();
    });

    it("does not fetch when scenarioName is empty", async () => {
      mockFetchSigningConfig.mockResolvedValue(createMockConfigResponse(null));
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(false)
      );

      renderHook(
        () => useSigningConfig({ scenarioName: "" }),
        { wrapper: createWrapper() }
      );

      // Wait a tick to ensure no fetch is made
      await act(async () => {
        await new Promise((r) => setTimeout(r, 50));
      });

      expect(mockFetchSigningConfig).not.toHaveBeenCalled();
    });
  });

  describe("readiness query", () => {
    it("fetches signing readiness for scenario", async () => {
      mockFetchSigningConfig.mockResolvedValue(
        createMockConfigResponse(createMockSigningConfig())
      );
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(true)
      );

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.readinessLoading).toBe(false);
      });

      expect(mockCheckSigningReadiness).toHaveBeenCalledWith("test-scenario");
    });

    it("returns readiness data", async () => {
      mockFetchSigningConfig.mockResolvedValue(
        createMockConfigResponse(createMockSigningConfig())
      );
      const mockReadiness = createMockReadinessResponse(true);
      mockCheckSigningReadiness.mockResolvedValue(mockReadiness);

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.readiness).toEqual(mockReadiness);
      });
    });
  });

  describe("enabledForBuild", () => {
    it("syncs enabledForBuild with config.enabled", async () => {
      const mockConfig = createMockSigningConfig({ enabled: true });
      mockFetchSigningConfig.mockResolvedValue(createMockConfigResponse(mockConfig));
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(true)
      );

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.enabledForBuild).toBe(true);
      });
    });

    it("sets enabledForBuild to false when config is null", async () => {
      mockFetchSigningConfig.mockResolvedValue(createMockConfigResponse(null));
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(false)
      );

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.configLoading).toBe(false);
      });

      expect(result.current.enabledForBuild).toBe(false);
    });

    it("setEnabledForBuild updates local state", async () => {
      mockFetchSigningConfig.mockResolvedValue(
        createMockConfigResponse(createMockSigningConfig({ enabled: false }))
      );
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(true)
      );

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.configLoading).toBe(false);
      });

      act(() => {
        result.current.setEnabledForBuild(true);
      });

      expect(result.current.enabledForBuild).toBe(true);
    });
  });

  describe("isReady", () => {
    it("returns true when readiness.ready is true", async () => {
      mockFetchSigningConfig.mockResolvedValue(
        createMockConfigResponse(createMockSigningConfig())
      );
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(true)
      );

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isReady).toBe(true);
      });
    });

    it("returns false when readiness.ready is false", async () => {
      mockFetchSigningConfig.mockResolvedValue(
        createMockConfigResponse(createMockSigningConfig())
      );
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(false, ["Missing certificate"])
      );

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.readinessLoading).toBe(false);
      });

      expect(result.current.isReady).toBe(false);
    });

    it("returns false when readiness is undefined", () => {
      mockFetchSigningConfig.mockImplementation(() => new Promise(() => {}));
      mockCheckSigningReadiness.mockImplementation(() => new Promise(() => {}));

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      expect(result.current.isReady).toBe(false);
    });
  });

  describe("firstIssue", () => {
    it("returns first issue when present", async () => {
      mockFetchSigningConfig.mockResolvedValue(
        createMockConfigResponse(createMockSigningConfig())
      );
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(false, ["First issue", "Second issue"])
      );

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.readinessLoading).toBe(false);
      });

      expect(result.current.firstIssue).toBe("First issue");
    });

    it("returns undefined when no issues", async () => {
      mockFetchSigningConfig.mockResolvedValue(
        createMockConfigResponse(createMockSigningConfig())
      );
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(true, [])
      );

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.readinessLoading).toBe(false);
      });

      expect(result.current.firstIssue).toBeUndefined();
    });
  });

  describe("loading state", () => {
    it("loading is true while either query is loading", async () => {
      mockFetchSigningConfig.mockResolvedValue(
        createMockConfigResponse(createMockSigningConfig())
      );
      // Keep readiness loading
      mockCheckSigningReadiness.mockImplementation(() => new Promise(() => {}));

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.configLoading).toBe(false);
      });

      // Readiness is still loading
      expect(result.current.loading).toBe(true);
    });

    it("loading is false when both queries complete", async () => {
      mockFetchSigningConfig.mockResolvedValue(
        createMockConfigResponse(createMockSigningConfig())
      );
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(true)
      );

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });
    });
  });

  describe("refreshAll", () => {
    it("refetches both config and readiness", async () => {
      mockFetchSigningConfig.mockResolvedValue(
        createMockConfigResponse(createMockSigningConfig())
      );
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(true)
      );

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      // Clear mock call counts
      mockFetchSigningConfig.mockClear();
      mockCheckSigningReadiness.mockClear();

      act(() => {
        result.current.refreshAll();
      });

      await waitFor(() => {
        expect(mockFetchSigningConfig).toHaveBeenCalled();
        expect(mockCheckSigningReadiness).toHaveBeenCalled();
      });
    });

    it("does not refetch when scenarioName is empty", async () => {
      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "" }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.refreshAll();
      });

      expect(mockFetchSigningConfig).not.toHaveBeenCalled();
      expect(mockCheckSigningReadiness).not.toHaveBeenCalled();
    });
  });

  describe("refetch functions", () => {
    it("refetchConfig refetches only config", async () => {
      mockFetchSigningConfig.mockResolvedValue(
        createMockConfigResponse(createMockSigningConfig())
      );
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(true)
      );

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      mockFetchSigningConfig.mockClear();

      act(() => {
        result.current.refetchConfig();
      });

      await waitFor(() => {
        expect(mockFetchSigningConfig).toHaveBeenCalled();
      });
    });

    it("refetchReadiness refetches only readiness", async () => {
      mockFetchSigningConfig.mockResolvedValue(
        createMockConfigResponse(createMockSigningConfig())
      );
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(true)
      );

      const { result } = renderHook(
        () => useSigningConfig({ scenarioName: "test-scenario" }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      mockCheckSigningReadiness.mockClear();

      act(() => {
        result.current.refetchReadiness();
      });

      await waitFor(() => {
        expect(mockCheckSigningReadiness).toHaveBeenCalled();
      });
    });
  });

  describe("scenario change", () => {
    it("resets enabledForBuild when scenario changes", async () => {
      const mockConfig = createMockSigningConfig({ enabled: true });
      mockFetchSigningConfig.mockResolvedValue(createMockConfigResponse(mockConfig));
      mockCheckSigningReadiness.mockResolvedValue(
        createMockReadinessResponse(true)
      );

      const { result, rerender } = renderHook(
        (props: { scenarioName: string }) => useSigningConfig(props),
        {
          wrapper: createWrapper(),
          initialProps: { scenarioName: "scenario-1" },
        }
      );

      await waitFor(() => {
        expect(result.current.enabledForBuild).toBe(true);
      });

      // Change to a scenario with no config
      mockFetchSigningConfig.mockResolvedValue(createMockConfigResponse(null));
      rerender({ scenarioName: "scenario-2" });

      await waitFor(() => {
        expect(result.current.enabledForBuild).toBe(false);
      });
    });
  });
});

import "@testing-library/jest-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { vi } from "vitest";

const getExpectedSecrets = vi.hoisted(() => vi.fn());
vi.mock("../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../lib/api")>("../lib/api");
  return { ...actual, getExpectedSecrets };
});

import { useExpectedSecrets } from "./useExpectedSecrets";

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("useExpectedSecrets", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getExpectedSecrets.mockResolvedValue({
      expected_secrets: [{ key: "API_TOKEN", required: true }],
      scenario_id: "demo",
      tier: "desktop",
      summary: { total: 1, configured: 0 },
    });
  });

  it("returns safe empty state when no deployment is selected", () => {
    const { result } = renderHook(() => useExpectedSecrets(null), { wrapper });
    expect(result.current.expectedSecrets).toEqual([]);
    expect(result.current.scenarioId).toBeNull();
    expect(result.current.tier).toBeNull();
    expect(result.current.summary).toBeNull();
    expect(getExpectedSecrets).not.toHaveBeenCalled();
  });

  it("loads expected secret requirements for the requested tier", async () => {
    const { result } = renderHook(() => useExpectedSecrets("deployment-1", "desktop"), { wrapper });
    await waitFor(() => expect(result.current.expectedSecrets).toHaveLength(1));
    expect(getExpectedSecrets).toHaveBeenCalledWith("deployment-1", "desktop");
    expect(result.current.scenarioId).toBe("demo");
    expect(result.current.tier).toBe("desktop");
    expect(result.current.summary).toEqual({ total: 1, configured: 0 });
  });
});

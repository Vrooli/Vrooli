import "@testing-library/jest-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { vi } from "vitest";

const api = vi.hoisted(() => ({
  listVPSSecrets: vi.fn(),
  getVPSSecret: vi.fn(),
  createVPSSecret: vi.fn(),
  updateVPSSecret: vi.fn(),
  deleteVPSSecret: vi.fn(),
}));

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../lib/api")>("../lib/api");
  return { ...actual, ...api };
});

import { useVPSSecrets } from "./useVPSSecrets";

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("useVPSSecrets", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.listVPSSecrets.mockResolvedValue({
      secrets: [{ key: "API_TOKEN", masked: true }],
      metadata: { count: 1 },
    });
    api.getVPSSecret.mockResolvedValue({ secret: { key: "API_TOKEN", value: "secret-value" } });
    api.createVPSSecret.mockResolvedValue({ ok: true });
    api.updateVPSSecret.mockResolvedValue({ ok: true });
    api.deleteVPSSecret.mockResolvedValue({ ok: true });
  });

  it("returns safe masked defaults for a null deployment", () => {
    const { result } = renderHook(() => useVPSSecrets(null), { wrapper });
    expect(result.current.secrets).toEqual([]);
    expect(result.current.metadata).toBeNull();
    expect(result.current.getSecretValue("API_TOKEN")).toEqual({ value: "********", masked: true });
    expect(result.current.isRevealed("API_TOKEN")).toBe(false);
  });

  it("loads, reveals, hides, and mutates secrets with restart options", async () => {
    const { result } = renderHook(() => useVPSSecrets("deployment-1"), { wrapper });
    await waitFor(() => expect(result.current.secrets).toHaveLength(1));
    expect(result.current.metadata).toEqual({ count: 1 });

    await act(async () => {
      await expect(result.current.revealSecret("API_TOKEN")).resolves.toBe("secret-value");
    });
    expect(api.getVPSSecret).toHaveBeenCalledWith("deployment-1", "API_TOKEN", true);
    expect(result.current.getSecretValue("API_TOKEN")).toEqual({ value: "secret-value", masked: false });
    expect(result.current.isRevealed("API_TOKEN")).toBe(true);
    act(() => result.current.hideSecret("API_TOKEN"));
    expect(result.current.isRevealed("API_TOKEN")).toBe(false);

    await act(async () => {
      await result.current.create({ key: "NEW", value: "value", restartScenario: true });
      await result.current.update({ key: "API_TOKEN", value: "new-value", restartScenario: true });
      await result.current.delete({ key: "API_TOKEN", restartScenario: true });
    });
    expect(api.createVPSSecret).toHaveBeenCalledWith("deployment-1", "NEW", "value", true);
    expect(api.updateVPSSecret).toHaveBeenCalledWith("deployment-1", "API_TOKEN", "new-value", true);
    expect(api.deleteVPSSecret).toHaveBeenCalledWith("deployment-1", "API_TOKEN", true);
  });

  it("propagates reveal failures without exposing a value", async () => {
    api.getVPSSecret.mockRejectedValueOnce(new Error("denied"));
    const { result } = renderHook(() => useVPSSecrets("deployment-1"), { wrapper });
    await waitFor(() => expect(result.current.secrets).toHaveLength(1));
    await expect(result.current.revealSecret("API_TOKEN")).rejects.toThrow("denied");
    expect(result.current.getSecretValue("API_TOKEN")).toEqual({ value: "********", masked: true });
  });
});

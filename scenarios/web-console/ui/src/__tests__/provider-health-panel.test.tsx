import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import ProviderHealthPanel, { type ProviderHealthPanelApi } from "../components/ProviderHealthPanel";
import { renderWithProviders } from "../test-utils";

function createApiMock(): ProviderHealthPanelApi {
  return {
    getConfig: vi.fn(),
    updateConfig: vi.fn(),
  };
}

const configResponse = {
  providers: [
    { name: "ollama", enabled: true, priority: 1, timeout_sec: 30, max_retries: 2 },
  ],
  health: [
    { name: "ollama", available: true, error_count: 0, success_count: 5, error_rate: 0, last_latency: "110ms" },
  ],
};

describe("ProviderHealthPanel seam behavior", () => {
  it("loads providers when opened", async () => {
    const api = createApiMock();
    vi.mocked(api.getConfig).mockResolvedValueOnce(configResponse);

    renderWithProviders(<ProviderHealthPanel open={true} api={api} />);

    await waitFor(() => {
      expect(api.getConfig).toHaveBeenCalledTimes(1);
      expect(screen.getByTestId("provider-card-ollama")).toBeTruthy();
    });
  });

  it("does not load providers while closed", () => {
    const api = createApiMock();
    renderWithProviders(<ProviderHealthPanel open={false} api={api} />);
    expect(api.getConfig).not.toHaveBeenCalled();
  });

  it("refresh button re-fetches provider config", async () => {
    const api = createApiMock();
    vi.mocked(api.getConfig)
      .mockResolvedValueOnce(configResponse)
      .mockResolvedValueOnce(configResponse);

    renderWithProviders(<ProviderHealthPanel open={true} api={api} />);

    await waitFor(() => {
      expect(api.getConfig).toHaveBeenCalledTimes(1);
    });

    fireEvent.click(screen.getByTestId("provider-refresh"));

    await waitFor(() => {
      expect(api.getConfig).toHaveBeenCalledTimes(2);
    });
  });

  it("toggles provider through injected API seam", async () => {
    const api = createApiMock();
    vi.mocked(api.getConfig).mockResolvedValueOnce(configResponse);
    vi.mocked(api.updateConfig).mockResolvedValueOnce({
      providers: [
        {
          name: "ollama",
          enabled: false,
          priority: 1,
          timeout_sec: 30,
          max_retries: 2,
        },
      ],
      health: configResponse.health,
    });

    renderWithProviders(<ProviderHealthPanel open={true} api={api} />);

    await waitFor(() => {
      expect(screen.getByTestId("provider-toggle-ollama")).toBeTruthy();
    });

    fireEvent.click(screen.getByTestId("provider-toggle-ollama"));

    await waitFor(() => {
      expect(api.updateConfig).toHaveBeenCalledWith({
        name: "ollama",
        enabled: false,
      });
    });
  });

  it("renders error banner on config load failure", async () => {
    const api = createApiMock();
    vi.mocked(api.getConfig).mockRejectedValueOnce(new Error("boom"));

    renderWithProviders(<ProviderHealthPanel open={true} api={api} />);

    await waitFor(() => {
      expect(screen.getByTestId("provider-error")).toBeTruthy();
      expect(screen.getByText("boom")).toBeTruthy();
    });
  });
});

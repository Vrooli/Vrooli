import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { LPBSReleaseConfigCard } from "./LPBSReleaseConfigCard";
import * as api from "../../lib/api";

vi.mock("../../lib/api");

const wrap = (ui: React.ReactNode) => {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>;
};

describe("LPBSReleaseConfigCard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the loading state initially", () => {
    vi.mocked(api.getProfileLPBSConfig).mockReturnValue(new Promise(() => {}));
    renderWithProviders(wrap(<LPBSReleaseConfigCard profileId="p1" />));
    expect(screen.getByTestId("lpbs-config-loading")).toBeInTheDocument();
  });

  it("renders the saved config and disables fields by default", async () => {
    vi.mocked(api.getProfileLPBSConfig).mockResolvedValue({
      profile_id: "p1",
      lpbs_domain: "example.com",
      lpbs_remote_profile: "prod",
      lpbs_app_key: "myapp",
      default_channel: "stable",
      update_url: "https://example.com/api/v1/updates",
    });
    renderWithProviders(wrap(<LPBSReleaseConfigCard profileId="p1" />));

    await waitFor(() => {
      const domain = screen.getByLabelText("LPBS Domain") as HTMLInputElement;
      expect(domain.value).toBe("example.com");
    });
    const domain = screen.getByLabelText("LPBS Domain") as HTMLInputElement;
    expect(domain.disabled).toBe(true);
  });

  it("enables editing when Edit clicked and saves the new values", async () => {
    vi.mocked(api.getProfileLPBSConfig).mockResolvedValue({
      profile_id: "p1",
      lpbs_domain: "old.example.com",
      lpbs_remote_profile: "",
      lpbs_app_key: "",
      default_channel: "stable",
      update_url: "",
    });
    vi.mocked(api.saveProfileLPBSConfig).mockImplementation(async (_id, cfg) => ({
      profile_id: "p1",
      lpbs_domain: cfg.lpbs_domain ?? "",
      lpbs_remote_profile: cfg.lpbs_remote_profile ?? "",
      lpbs_app_key: cfg.lpbs_app_key ?? "",
      default_channel: cfg.default_channel ?? "stable",
      update_url: cfg.update_url ?? "",
    }));

    renderWithProviders(wrap(<LPBSReleaseConfigCard profileId="p1" />));
    await screen.findByLabelText("LPBS Domain");
    fireEvent.click(screen.getByText("Edit"));

    const domain = screen.getByLabelText("LPBS Domain") as HTMLInputElement;
    fireEvent.change(domain, { target: { value: "new.example.com" } });
    fireEvent.change(screen.getByLabelText("Remote Profile Tag"), { target: { value: "desktop" } });
    fireEvent.change(screen.getByLabelText("App Key"), { target: { value: "app" } });
    fireEvent.change(screen.getByLabelText("Default Channel"), { target: { value: "beta" } });
    fireEvent.change(screen.getByLabelText("Update URL (optional)"), { target: { value: "https://updates.example.com" } });
    fireEvent.click(screen.getByText("Save"));

    await waitFor(() => {
      expect(vi.mocked(api.saveProfileLPBSConfig)).toHaveBeenCalledWith(
        "p1",
        expect.objectContaining({ lpbs_domain: "new.example.com" })
      );
    });

    fireEvent.click(screen.getByText("Edit"));
    fireEvent.click(screen.getByText("Cancel"));
    expect((screen.getByLabelText("LPBS Domain") as HTMLInputElement).disabled).toBe(true);
  });

  it("shows an error if loading fails", async () => {
    vi.mocked(api.getProfileLPBSConfig).mockRejectedValue(new Error("boom"));
    renderWithProviders(wrap(<LPBSReleaseConfigCard profileId="p1" />));
    expect(await screen.findByTestId("lpbs-config-error")).toHaveTextContent("boom");
  });
});

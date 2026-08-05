import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReleasesPanel } from "./ReleasesPanel";
import * as api from "../../lib/api";

vi.mock("../../lib/api");

const wrap = (ui: React.ReactNode) => {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{ui}</QueryClientProvider>;
};

describe("ReleasesPanel", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders the empty state when no releases", async () => {
    vi.mocked(api.listProfileReleases).mockResolvedValue({ releases: [] });
    renderWithProviders(wrap(<ReleasesPanel profileId="p1" />));
    expect(await screen.findByTestId("releases-empty")).toBeInTheDocument();
  });

  it("renders releases with per-platform statuses", async () => {
    vi.mocked(api.listProfileReleases).mockResolvedValue({
      releases: [
        {
          id: "abcdef1234567890",
          profile_id: "p1",
          git_commit_hash: "deadbeef",
          release_version: "1.2.3",
          channel: "stable",
          status: "published",
          created_at: "2026-04-01T00:00:00Z",
          updated_at: "2026-04-01T00:01:00Z",
          platforms: [
            { release_id: "abcdef1234567890", platform: "linux-x64", status: "published" },
            { release_id: "abcdef1234567890", platform: "darwin-arm64", status: "verify_failed" },
          ],
        },
      ],
    });
    renderWithProviders(wrap(<ReleasesPanel profileId="p1" />));

    await screen.findByText("1.2.3 on stable");
    expect(screen.getByText("linux-x64")).toBeInTheDocument();
    expect(screen.getByText("darwin-arm64")).toBeInTheDocument();
  });

  it("triggers re-verify when button clicked", async () => {
    vi.mocked(api.listProfileReleases).mockResolvedValue({
      releases: [
        {
          id: "abcdef1234567890",
          profile_id: "p1",
          git_commit_hash: "x",
          release_version: "1.0.0",
          channel: "stable",
          status: "verify_failed",
          created_at: "2026-04-01",
          updated_at: "2026-04-01",
          platforms: [{ release_id: "abcdef1234567890", platform: "linux-x64", status: "verify_failed" }],
        },
      ],
    });
    vi.mocked(api.reverifyRelease).mockResolvedValue({
      id: "abcdef1234567890",
      profile_id: "p1",
      git_commit_hash: "x",
      release_version: "1.0.0",
      channel: "stable",
      status: "published",
      created_at: "2026-04-01",
      updated_at: "2026-04-01",
    });
    renderWithProviders(wrap(<ReleasesPanel profileId="p1" />));

    const btn = await screen.findByTestId("release-reverify-abcdef1234567890");
    fireEvent.click(btn);

    await waitFor(() => {
      expect(vi.mocked(api.reverifyRelease)).toHaveBeenCalledWith("abcdef1234567890");
    });
  });

  it("renders an error when the list call fails", async () => {
    vi.mocked(api.listProfileReleases).mockRejectedValue(new Error("server unavailable"));
    renderWithProviders(wrap(<ReleasesPanel profileId="p1" />));
    expect(await screen.findByTestId("releases-error")).toHaveTextContent("server unavailable");
  });
});

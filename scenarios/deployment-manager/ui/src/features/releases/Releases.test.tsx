import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { Releases } from "./Releases";
import * as api from "../../lib/api";
import { renderWithProviders } from "../../test-utils/renderWithProviders";

vi.mock("../../lib/api");

describe("Releases", () => {
  beforeEach(() => vi.clearAllMocks());

  it("reads real release records and platform counts", async () => {
    vi.mocked(api.listProfiles).mockResolvedValue([{ id: "p1", name: "Production", scenario: "demo", tiers: [2], version: 1 }]);
    vi.mocked(api.listProfileReleases).mockResolvedValue({ releases: [{
      id: "release-1",
      profile_id: "p1",
      git_commit_hash: "abc123",
      release_version: "1.4.0",
      channel: "stable",
      status: "published",
      created_at: "2026-08-04T12:00:00Z",
      updated_at: "2026-08-04T12:01:00Z",
      platforms: [
        { release_id: "release-1", platform: "linux", status: "published" },
        { release_id: "release-1", platform: "windows", status: "published" },
      ],
    }] });

    renderWithProviders(<Releases />);
    await screen.findByRole("option", { name: "Production" });
    fireEvent.change(screen.getByRole("combobox"), { target: { value: "p1" } });
    expect(await screen.findByText("1.4.0 · stable")).toBeInTheDocument();
    expect(screen.getByText("Platforms: 2")).toBeInTheDocument();
    expect(screen.getByText("abc123")).toBeInTheDocument();
    expect(api.listProfileReleases).toHaveBeenCalledWith("p1", 50);
    fireEvent.click(screen.getByRole("button", { name: /refresh/i }));
  });

  it("shows an explicit empty state for a selected profile", async () => {
    vi.mocked(api.listProfiles).mockResolvedValue([{ id: "p1", name: "Production", scenario: "demo", tiers: [2], version: 1 }]);
    vi.mocked(api.listProfileReleases).mockResolvedValue({ releases: [] });
    renderWithProviders(<Releases />);
    await screen.findByRole("option", { name: "Production" });
    fireEvent.change(screen.getByRole("combobox"), { target: { value: "p1" } });
    expect(await screen.findByText("No release records for this profile.")).toBeInTheDocument();
  });
});

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";

const { planCleanup, cleanupVersions } = vi.hoisted(() => ({
  planCleanup: vi.fn(),
  cleanupVersions: vi.fn(),
}));

vi.mock("../../api/versionLedger", () => ({
  versionLifecycleClient: { planCleanup, cleanupVersions },
}));

import { VersionCleanupPanel } from "./VersionCleanupPanel";

describe("VersionCleanupPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
    planCleanup.mockResolvedValue({
      items: [
        {
          version: { libraryId: "rcl:Button", version: "1.0.0", status: "deprecated" },
          eligible: true,
          reason: "safe to retire",
          ageDays: 120,
        },
        {
          version: { libraryId: "rcl:Button", version: "2.0.0", status: "released" },
          eligible: false,
          reason: "latest version",
          ageDays: 30,
        },
      ],
      planHash: "plan-123456789",
    });
    cleanupVersions.mockResolvedValue({ retiredCount: 1, applied: true });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("previews an asset cleanup and requires acknowledgement before applying it", async () => {
    const user = userEvent.setup();
    renderWithProviders(<VersionCleanupPanel componentId="component-1" />);

    await user.click(screen.getByRole("button", { name: "Review cleanup" }));
    await user.click(screen.getByRole("button", { name: "Preview cleanup" }));
    await screen.findByText("1.0.0");

    const retire = screen.getByRole("button", { name: "Retire 1 version" });
    expect(retire).toBeDisabled();
    await user.click(screen.getByRole("checkbox"));
    expect(retire).not.toBeDisabled();
    await user.click(retire);

    await waitFor(() =>
      expect(cleanupVersions).toHaveBeenCalledWith(
        expect.objectContaining({
          planHash: "plan-123456789",
          confirm: true,
          scope: { componentId: "component-1", olderThanDays: 30 },
        }),
      ),
    );
  });

  it("offers the same workflow without an asset scope for library cleanup", async () => {
    const user = userEvent.setup();
    renderWithProviders(<VersionCleanupPanel />);
    expect(screen.getByTestId("library-version-cleanup")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Review cleanup" }));
    await user.click(screen.getByRole("button", { name: "Preview cleanup" }));
    await waitFor(() =>
      expect(planCleanup).toHaveBeenCalledWith({
        scope: { olderThanDays: 30 },
      }),
    );
  });
});

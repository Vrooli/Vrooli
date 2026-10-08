import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TrackedBinariesSection, formatBytes } from "./SettingsTabTrackedBinaries";
import { renderWithQueryClient } from "../test-utils";

const connectMocks = vi.hoisted(() => ({
  repoClient: {
    getTrackedBinaries: vi.fn(),
    untrackBinary: vi.fn(),
  },
  humanControlClient: {
    getAuthorityStatus: vi.fn(),
    prepareMutation: vi.fn(),
    confirmMutation: vi.fn(),
  },
}));

vi.mock("../lib/connect", () => connectMocks);

const oneBinary = {
  binaries: [
    {
      path: "scenarios/tidiness-manager/cli/cli",
      bytes: 8_460_000,
      format: "elf",
      owner_dir: "scenarios/tidiness-manager",
      ignore_pattern: "/cli/cli",
      already_ignored: false,
    },
  ],
  total_bytes: 8_460_000,
  history_warning:
    "Untracking removes these from the working tree and future commits. The bytes remain in git history, so repository size is unchanged until history is rewritten.",
};

describe("TrackedBinariesSection", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    connectMocks.repoClient.getTrackedBinaries.mockResolvedValue({
      binaries: [{
        path: "scenarios/tidiness-manager/cli/cli",
        bytes: 8_460_000,
        format: "elf",
        ownerDir: "scenarios/tidiness-manager",
        ignorePattern: "/cli/cli",
        alreadyIgnored: false,
      }],
      totalBytes: 8_460_000,
      historyWarning: oneBinary.history_warning,
    });
    connectMocks.repoClient.untrackBinary.mockResolvedValue({ success: true, removedFromIndex: true, ignoreAddedTo: "scenarios/tidiness-manager/.gitignore" });
    connectMocks.humanControlClient.getAuthorityStatus.mockResolvedValue({ canMutate: true });
    connectMocks.humanControlClient.prepareMutation.mockResolvedValue({ repositoryId: "repo-1", operation: "repo.tracked-binaries.untrack", expectedRevision: "head", subjectDigest: "digest" });
    connectMocks.humanControlClient.confirmMutation.mockResolvedValue({ intentId: "intent-1" });
  });

  it("lists tracked binaries with size and ignore target", async () => {
    renderWithQueryClient(<TrackedBinariesSection isMobile={false} repoId="repo-1" />);

    expect(await screen.findByText("scenarios/tidiness-manager/cli/cli")).toBeInTheDocument();
    // Size shows twice on purpose: once as the row's own cost, once in the
    // header total, so the panel answers "how big is this problem" at a glance.
    expect(screen.getAllByText(/8\.1 MB/).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/scenarios\/tidiness-manager\/\.gitignore/)).toBeInTheDocument();
  });

  // Untracking never shrinks the repo. If the panel omitted this, users would
  // reasonably conclude the space was reclaimed.
  it("states that history is unchanged", async () => {
    renderWithQueryClient(<TrackedBinariesSection isMobile={false} repoId="repo-1" />);

    expect(await screen.findByText(/remain in git history/i)).toBeInTheDocument();
  });

  it("posts the untrack request with the owning gitignore target", async () => {
    renderWithQueryClient(<TrackedBinariesSection isMobile={false} repoId="repo-1" />);

    fireEvent.click(await screen.findByRole("button", { name: /untrack & ignore/i }));

    await waitFor(() => {
      expect(connectMocks.repoClient.untrackBinary).toHaveBeenCalledWith(expect.objectContaining({
        repositoryId: "repo-1",
        intentId: "intent-1",
        path: "scenarios/tidiness-manager/cli/cli",
        ownerDir: "scenarios/tidiness-manager",
        ignorePattern: "/cli/cli",
      }));
    });
  });

  it("reports a clean repository instead of an empty list", async () => {
    connectMocks.repoClient.getTrackedBinaries.mockResolvedValue({ binaries: [], totalBytes: 0, historyWarning: "" });

    renderWithQueryClient(<TrackedBinariesSection isMobile={false} repoId="repo-1" />);

    expect(await screen.findByText(/no compiled binaries are tracked/i)).toBeInTheDocument();
  });
});

describe("formatBytes", () => {
  it("scales units and keeps one decimal below ten", () => {
    expect(formatBytes(512)).toBe("512 B");
    expect(formatBytes(8_460_000)).toBe("8.1 MB");
    expect(formatBytes(164 * 1024 * 1024)).toBe("164 MB");
  });
});

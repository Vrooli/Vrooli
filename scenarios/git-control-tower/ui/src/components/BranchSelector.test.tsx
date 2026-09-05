import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { BranchSelector, type BranchActions, type RepoActions } from "./BranchSelector";
import type { RepoStatus, SyncStatusResponse } from "../lib/api";

const timestamp = "2026-05-01T00:00:00Z";

function status(overrides: Partial<RepoStatus> = {}): RepoStatus {
  return {
    repo_dir: "/work/repo",
    branch: {
      head: "main",
      oid: "abcdef1234567890",
      upstream: "origin/main",
      ahead: 0,
      behind: 0,
    },
    files: {
      staged: [],
      unstaged: [],
      untracked: [],
      conflicts: [],
    },
    summary: {
      staged: 0,
      unstaged: 0,
      untracked: 0,
      conflicts: 0,
    },
    author: {},
    timestamp,
    ...overrides,
  };
}

function syncStatus(overrides: Partial<SyncStatusResponse> = {}): SyncStatusResponse {
  return {
    branch: "main",
    upstream: "origin/main",
    ahead: 0,
    behind: 0,
    has_upstream: true,
    can_push: false,
    can_pull: false,
    needs_push: false,
    needs_pull: false,
    has_uncommitted_changes: false,
    fetched: true,
    timestamp,
    ...overrides,
  };
}

function branchActions(overrides: Partial<BranchActions> = {}): BranchActions {
  return {
    branches: {
      current: "main",
      locals: [
        { name: "main", is_current: true, upstream: "origin/main" },
        { name: "feature/search", is_current: false },
      ],
      remotes: [
        { name: "origin/main" },
        { name: "origin/review-ui" },
      ],
      timestamp,
    },
    isLoading: false,
    createBranch: vi.fn().mockResolvedValue({
      success: true,
      branch: { name: "feature/new", is_current: true },
      timestamp,
    }),
    switchBranch: vi.fn().mockResolvedValue({
      success: true,
      branch: { name: "feature/search", is_current: true },
      timestamp,
    }),
    publishBranch: vi.fn().mockResolvedValue({
      success: true,
      remote: "origin",
      branch: "main",
      timestamp,
    }),
    isCreating: false,
    isSwitching: false,
    isPublishing: false,
    ...overrides,
  };
}

function repoActions(overrides: Partial<RepoActions> = {}): RepoActions {
  return {
    repos: {
      active_id: 1,
      repos: [
        {
          id: 1,
          name: "tower",
          path: "/work/tower",
          remote_url: "git@example.com:org/tower.git",
          added_at: timestamp,
        },
        {
          id: 2,
          name: "sandbox",
          path: "/work/sandbox",
          remote_url: "git@example.com:org/sandbox.git",
          added_at: timestamp,
        },
      ],
      timestamp,
    },
    isLoading: false,
    openRepo: vi.fn().mockResolvedValue({
      repo: { id: 3, name: "opened", path: "/work/opened", added_at: timestamp },
      timestamp,
    }),
    cloneRepo: vi.fn().mockResolvedValue({
      repo: { id: 4, name: "cloned", path: "/work/cloned", added_at: timestamp },
      timestamp,
    }),
    setActiveRepo: vi.fn().mockResolvedValue({
      repo: { id: 2, name: "sandbox", path: "/work/sandbox", added_at: timestamp },
      timestamp,
    }),
    removeRepo: vi.fn().mockResolvedValue({ removed: true, timestamp }),
    isOpening: false,
    isCloning: false,
    isSettingActive: false,
    isRemoving: false,
    ...overrides,
  };
}

describe("BranchSelector", () => {
  it("keeps the mobile trigger touch-target sized", () => {
    render(
      <BranchSelector
        status={status()}
        syncStatus={syncStatus()}
        actions={branchActions()}
        variant="mobile"
      />,
    );

    expect(screen.getByTestId("branch-selector-trigger")).toHaveClass("touch-target");
  });

  it("filters desktop branches and routes branch switching through the branch seam", async () => {
    const actions = branchActions();

    render(
      <BranchSelector
        status={status()}
        syncStatus={syncStatus()}
        actions={actions}
      />,
    );

    fireEvent.click(screen.getByTestId("branch-selector-trigger"));
    const panel = screen.getByTestId("branch-selector-panel");

    fireEvent.change(within(panel).getByTestId("branch-search"), {
      target: { value: "review" },
    });

    expect(within(panel).queryByTestId("branch-local-feature/search")).not.toBeInTheDocument();
    fireEvent.click(within(panel).getByTestId("branch-remote-origin/review-ui"));

    await waitFor(() => {
      expect(actions.switchBranch).toHaveBeenCalledWith({ name: "origin/review-ui" });
    });
  });

  it("confirms dirty branch-switch warnings with the required override flags", async () => {
    const switchBranch = vi
      .fn()
      .mockResolvedValueOnce({
        success: false,
        warning: {
          message: "Working tree has changes",
          requires_confirmation: true,
          requires_tracking: true,
          dirty_summary: {
            staged: 1,
            unstaged: 2,
            untracked: 3,
            conflicts: 0,
          },
        },
        timestamp,
      })
      .mockResolvedValueOnce({
        success: true,
        branch: { name: "feature/search", is_current: true },
        timestamp,
      });
    const actions = branchActions({ switchBranch });

    render(
      <BranchSelector
        status={status({
          files: {
            staged: ["a.ts"],
            unstaged: ["b.ts", "c.ts"],
            untracked: ["d.ts", "e.ts", "f.ts"],
            conflicts: [],
          },
          summary: {
            staged: 1,
            unstaged: 2,
            untracked: 3,
            conflicts: 0,
          },
        })}
        syncStatus={syncStatus()}
        actions={actions}
      />,
    );

    fireEvent.click(screen.getByTestId("branch-selector-trigger"));
    fireEvent.click(screen.getByTestId("branch-local-feature/search"));

    expect(await screen.findByTestId("branch-warning")).toHaveTextContent("Dirty: 1 staged, 2 modified, 3 untracked, 0 conflicts");

    fireEvent.click(screen.getByTestId("branch-warning-confirm"));

    await waitFor(() => {
      expect(switchBranch).toHaveBeenLastCalledWith({
        name: "feature/search",
        allow_dirty: true,
        track_remote: true,
      });
    });
  });

  it("validates and submits branch creation options", async () => {
    const actions = branchActions();

    render(
      <BranchSelector
        status={status()}
        syncStatus={syncStatus()}
        actions={actions}
      />,
    );

    fireEvent.click(screen.getByTestId("branch-selector-trigger"));
    fireEvent.click(screen.getByTestId("branch-create-toggle"));
    fireEvent.click(screen.getByTestId("branch-create-submit"));

    expect(await screen.findByTestId("branch-warning")).toHaveTextContent("Branch name is required");

    fireEvent.change(screen.getByTestId("branch-create-name"), {
      target: { value: " feature/new " },
    });
    fireEvent.change(screen.getByTestId("branch-create-from"), {
      target: { value: " main " },
    });
    fireEvent.click(screen.getByTestId("branch-create-checkout"));
    fireEvent.click(screen.getByTestId("branch-create-submit"));

    await waitFor(() => {
      expect(actions.createBranch).toHaveBeenCalledWith({
        name: "feature/new",
        from: "main",
        checkout: false,
      });
    });
  });

  it("opens repository selection when no active repo exists and routes repo actions", async () => {
    const repos = repoActions({
      repos: {
        active_id: undefined,
        repos: [
          {
            id: 2,
            name: "sandbox",
            path: "/work/sandbox",
            remote_url: "git@example.com:org/sandbox.git",
            added_at: timestamp,
          },
        ],
        timestamp,
      },
    });
    const onRepoChange = vi.fn();

    render(
      <BranchSelector
        status={status()}
        syncStatus={syncStatus()}
        actions={branchActions()}
        repoActions={repos}
        onRepoChange={onRepoChange}
      />,
    );

    fireEvent.click(screen.getByTestId("branch-selector-trigger"));
    expect(screen.getByTestId("repo-selector-panel")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("repo-item-2"));

    await waitFor(() => {
      expect(repos.setActiveRepo).toHaveBeenCalledWith({ id: 2 });
      expect(onRepoChange).toHaveBeenCalledWith("2");
    });

    fireEvent.click(screen.getByTestId("repo-change-button"));
    fireEvent.click(screen.getByTestId("repo-open-submit"));

    expect(await screen.findByText("Repository path is required")).toBeInTheDocument();

    fireEvent.change(screen.getByTestId("repo-open-path"), {
      target: { value: " /work/opened " },
    });
    fireEvent.click(screen.getByTestId("repo-open-submit"));

    await waitFor(() => {
      expect(repos.openRepo).toHaveBeenCalledWith({ path: "/work/opened" });
      expect(onRepoChange).toHaveBeenCalledWith("3");
    });
  });

  it("publishes branches and confirms fetch-required publish warnings", async () => {
    const publishBranch = vi
      .fn()
      .mockResolvedValueOnce({
        success: false,
        remote: "origin",
        branch: "main",
        warning: {
          message: "Fetch before publishing",
          requires_fetch: true,
        },
        timestamp,
      })
      .mockResolvedValueOnce({
        success: true,
        remote: "origin",
        branch: "main",
        timestamp,
      });
    const actions = branchActions({ publishBranch });

    render(
      <BranchSelector
        status={status({ branch: { head: "feature/new", oid: "abc1234" } })}
        syncStatus={syncStatus({
          branch: "feature/new",
          upstream: undefined,
          has_upstream: false,
          ahead: 2,
          needs_push: true,
          can_push: true,
        })}
        actions={actions}
      />,
    );

    fireEvent.click(screen.getByTestId("branch-selector-trigger"));
    fireEvent.click(screen.getByTestId("branch-publish-button"));

    expect(await screen.findByTestId("branch-warning")).toHaveTextContent("Fetch before publishing");

    fireEvent.click(screen.getByTestId("branch-warning-confirm"));

    await waitFor(() => {
      expect(publishBranch).toHaveBeenLastCalledWith({ fetch: true });
    });
  });
});

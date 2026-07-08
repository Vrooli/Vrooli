import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MobileHeader } from "./MobileHeader";
import { MobileNav } from "./MobileNav";
import type { BranchActions } from "./BranchSelector";
import type { HealthResponse, RepoStatus, SyncStatusResponse } from "../lib/api";

vi.mock("./BranchSelector", () => ({
  BranchSelector: ({ status }: { status?: RepoStatus }) => (
    <button type="button" data-testid="branch-selector">
      {status?.branch.head ?? "No branch"}
    </button>
  ),
}));

function repoStatus(overrides: Partial<RepoStatus> = {}): RepoStatus {
  return {
    repo_dir: "/work/repo",
    branch: {
      head: "main",
      oid: "abcdef1234567890",
      upstream: "origin/main",
    },
    files: {
      staged: ["src/staged.ts"],
      unstaged: ["src/changed.ts", "src/other.ts"],
      untracked: ["notes.md"],
      conflicts: [],
    },
    summary: {
      staged: 1,
      unstaged: 2,
      untracked: 1,
      conflicts: 0,
    },
    author: {},
    timestamp: "2026-05-01T00:00:00Z",
    ...overrides,
  };
}

function health(readiness = true): HealthResponse {
  return {
    status: readiness ? "ok" : "degraded",
    service: "git-control-tower",
    timestamp: "2026-05-01T00:00:00Z",
    readiness,
    checks: {
      database: { status: readiness ? "ok" : "error" },
      git: { status: "ok" },
      repo: { status: "ok" },
    },
  };
}

function syncStatus(overrides: Partial<SyncStatusResponse> = {}): SyncStatusResponse {
  return {
    branch: "main",
    upstream: "origin/main",
    ahead: 2,
    behind: 1,
    has_upstream: true,
    can_push: true,
    can_pull: true,
    needs_push: true,
    needs_pull: true,
    has_uncommitted_changes: true,
    fetched: true,
    timestamp: "2026-05-01T00:00:00Z",
    ...overrides,
  };
}

function branchActions(): BranchActions {
  return {
    branches: {
      current: "main",
      locals: [{ name: "main", is_current: true }],
      remotes: [{ name: "origin/main" }],
      timestamp: "2026-05-01T00:00:00Z",
    },
    isLoading: false,
    createBranch: vi.fn(),
    switchBranch: vi.fn(),
    publishBranch: vi.fn(),
    isCreating: false,
    isSwitching: false,
    isPublishing: false,
  };
}

describe("MobileNav", () => {
  it("routes panel changes and shows capped change badges", () => {
    const onPanelChange = vi.fn();

    render(
      <MobileNav
        activePanel="diff"
        onPanelChange={onPanelChange}
        stagedCount={120}
        unstagedCount={4}
      />,
    );

    expect(screen.getByTestId("mobile-nav")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-nav")).toHaveClass("pb-safe");
    expect(screen.getByTestId("mobile-nav-diff")).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId("mobile-nav-changes")).toHaveTextContent("99+");
    expect(screen.getByTestId("mobile-nav-commit")).toHaveTextContent("99+");

    fireEvent.click(screen.getByTestId("mobile-nav-review"));

    expect(onPanelChange).toHaveBeenCalledWith("review");
  });
});

describe("MobileHeader", () => {
  it("routes top-level actions and exposes repository details in the menu sheet", () => {
    const onRefresh = vi.fn();
    const onOpenSettings = vi.fn();
    const onOpenReview = vi.fn();
    const onOpenFileSearch = vi.fn();

    render(
      <MobileHeader
        status={repoStatus()}
        health={health(true)}
        syncStatus={syncStatus()}
        branchActions={branchActions()}
        isLoading={false}
        onRefresh={onRefresh}
        onOpenSettings={onOpenSettings}
        onOpenReview={onOpenReview}
        onOpenFileSearch={onOpenFileSearch}
        onPush={vi.fn()}
        onPull={vi.fn()}
      />,
    );

    expect(screen.getByTestId("mobile-header")).toBeInTheDocument();
    expect(screen.getByTestId("branch-selector")).toHaveTextContent("main");

    fireEvent.click(screen.getByTestId("mobile-review-button"));
    fireEvent.click(screen.getByTestId("mobile-search-button"));

    expect(onOpenReview).toHaveBeenCalledTimes(1);
    expect(onOpenFileSearch).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId("mobile-menu-button"));

    const dialog = screen.getByRole("dialog", { name: "Settings & Info" });
    expect(within(dialog).getByText("abcdef1")).toBeInTheDocument();
    expect(within(dialog).getByText("origin/main")).toBeInTheDocument();
    expect(within(dialog).getByText("1 staged")).toBeInTheDocument();
    expect(within(dialog).getByText("2 modified")).toBeInTheDocument();
    expect(within(dialog).getByText("1 untracked")).toBeInTheDocument();
    expect(within(dialog).getByText("All systems healthy")).toBeInTheDocument();

    fireEvent.click(within(dialog).getByRole("button", { name: /Refresh status/i }));
    expect(onRefresh).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("dialog", { name: "Settings & Info" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("mobile-menu-button"));
    fireEvent.click(screen.getByRole("button", { name: /Settings/i }));

    expect(onOpenSettings).toHaveBeenCalledTimes(1);
  });

  it("renders compact history and blame headers instead of the default mobile header", () => {
    const onExitHistoryMode = vi.fn();
    const onExitBlameMode = vi.fn();
    const baseProps = {
      branchActions: branchActions(),
      isLoading: false,
      onRefresh: vi.fn(),
      onOpenSettings: vi.fn(),
    };

    const { rerender } = render(
      <MobileHeader
        {...baseProps}
        viewingCommit={{
          hash: "abc123",
          subject: "feat: add tests",
          files: ["src/app.ts"],
        }}
        onExitHistoryMode={onExitHistoryMode}
      />,
    );

    expect(screen.getByText("History")).toBeInTheDocument();
    expect(screen.getByText("abc123")).toBeInTheDocument();
    expect(screen.queryByTestId("mobile-menu-button")).not.toBeInTheDocument();

    rerender(
      <MobileHeader
        {...baseProps}
        viewingFileBlame={{
          path: "src/app.ts",
          filename: "app.ts",
        }}
        onExitBlameMode={onExitBlameMode}
      />,
    );

    expect(screen.getByText("File History")).toBeInTheDocument();
    expect(screen.getByText("app.ts")).toBeInTheDocument();
    expect(screen.queryByTestId("mobile-menu-button")).not.toBeInTheDocument();
  });
});

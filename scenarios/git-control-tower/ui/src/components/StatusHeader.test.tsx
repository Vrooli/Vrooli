import "@testing-library/jest-dom";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { RepoStatus, SyncStatusResponse } from "../lib/api";
import { StatusHeader } from "./StatusHeader";

describe("StatusHeader", () => {
  it("clarifies clean working tree with ahead/behind info", () => {
    const status: RepoStatus = {
      repo_dir: "/repo",
      branch: { head: "main", upstream: "origin/main", oid: "abc1234" },
      files: { staged: [], unstaged: [], untracked: [], conflicts: [] },
      scopes: {},
      summary: { staged: 0, unstaged: 0, untracked: 0, conflicts: 0 },
      author: {},
      timestamp: "2025-01-01T00:00:00Z"
    };
    const syncStatus: SyncStatusResponse = {
      branch: "main",
      upstream: "origin/main",
      remote_url: "git@example.com:repo.git",
      ahead: 3,
      behind: 0,
      has_upstream: true,
      can_push: true,
      can_pull: false,
      needs_push: true,
      needs_pull: false,
      has_uncommitted_changes: false,
      fetched: false,
      timestamp: "2025-01-01T00:00:00Z"
    };

    render(
      <StatusHeader
        status={status}
        syncStatus={syncStatus}
        isLoading={false}
        onRefresh={() => {}}
        onOpenLayoutSettings={() => {}}
      />
    );

    expect(screen.getByText("Working tree clean (3 ahead)")).toBeInTheDocument();
  });
});

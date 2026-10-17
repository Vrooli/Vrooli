import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";

import { ClosedSandboxDetail } from "./ClosedSandboxDetail";
import type { DiffArchive, DiffResult } from "../lib/api";

function makeArchive(overrides: Partial<DiffArchive> = {}): DiffArchive {
  return {
    sandboxId: "00000000-0000-0000-0000-000000000abc",
    snapshotAt: "2026-01-01T12:00:00Z",
    archiveState: "complete",
    sandboxStatus: "approved",
    files: [],
    stats: {
      filesChanged: 0,
      filesAdded: 0,
      filesModified: 0,
      filesDeleted: 0,
      linesAdded: 0,
      linesRemoved: 0,
      totalBytes: 0,
    },
    totalBlobBytes: 0,
    projectRoot: "/home/user/Vrooli",
    owner: "agent-xyz",
    agentManagerRunId: "run-42",
    unifiedDiffSha256: "abcdef0123456789".repeat(4),
    ...overrides,
  };
}

function makeDiff(overrides: Partial<DiffResult> = {}): DiffResult {
  return {
    sandboxId: "00000000-0000-0000-0000-000000000abc",
    files: [],
    unifiedDiff: "",
    generated: "2026-01-01T12:00:00Z",
    stats: {
      filesChanged: 0,
      filesAdded: 0,
      filesModified: 0,
      filesDeleted: 0,
      linesAdded: 0,
      linesRemoved: 0,
      totalBytes: 0,
    },
    archiveState: "complete",
    ...overrides,
  };
}

describe("ClosedSandboxDetail", () => {
  it("renders archive metadata with terminal status badge", () => {
    render(
      <ClosedSandboxDetail
        archive={makeArchive({ sandboxStatus: "approved" })}
        diff={makeDiff()}
        isDiffLoading={false}
      />,
    );
    expect(screen.getByText("Archive")).toBeInTheDocument();
    expect(screen.getByText("Approved")).toBeInTheDocument();
    expect(screen.getByText("/home/user/Vrooli")).toBeInTheDocument();
    expect(screen.getByText("run-42")).toBeInTheDocument();
  });

  it("surfaces the not-captured banner for skipped archives", () => {
    render(
      <ClosedSandboxDetail
        archive={makeArchive({ archiveState: "not_captured", sandboxStatus: "deleted" })}
        diff={makeDiff({ archiveState: "not_captured" })}
        isDiffLoading={false}
      />,
    );
    expect(screen.getByTestId("archive-not-captured")).toBeInTheDocument();
    expect(screen.getByTestId("archive-not-captured")).toHaveTextContent("No diff captured");
  });

  it("does not render the not-captured banner for complete archives", () => {
    render(
      <ClosedSandboxDetail
        archive={makeArchive({ archiveState: "complete" })}
        diff={makeDiff()}
        isDiffLoading={false}
      />,
    );
    expect(screen.queryByTestId("archive-not-captured")).not.toBeInTheDocument();
  });
});

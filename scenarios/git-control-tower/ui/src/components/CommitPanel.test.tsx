import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CommitPanel } from "./CommitPanel";

describe("CommitPanel", () => {
  it("shows push action when ahead even without a new commit", () => {
    render(
      <CommitPanel
        stagedCount={0}
        commitMessage=""
        onCommitMessageChange={() => {}}
        onCommit={vi.fn()}
        isCommitting={false}
        onPush={vi.fn()}
        canPush
        aheadCount={3}
      />
    );

    expect(screen.getByTestId("push-button")).toBeInTheDocument();
    expect(screen.getByText(/push \(3\)/i)).toBeInTheDocument();
  });

  it("disables amend when upstream is not available", () => {
    render(
      <CommitPanel
        stagedCount={1}
        commitMessage="fix: adjust"
        onCommitMessageChange={() => {}}
        onCommit={vi.fn()}
        isCommitting={false}
        canAmend={false}
        amendDisabledReason="Set upstream before amending"
      />
    );

    fireEvent.click(screen.getByRole("button", { name: /advanced/i }));
    const amendCheckbox = screen.getByTestId("amend-commit-checkbox");
    expect(amendCheckbox).toBeDisabled();
    expect(screen.getByText(/set upstream before amending/i)).toBeInTheDocument();
  });

  it("shows commit checks in history mode", () => {
    render(
      <CommitPanel
        stagedCount={0}
        commitMessage=""
        onCommitMessageChange={() => {}}
        onCommit={vi.fn()}
        isCommitting={false}
        isHistoryMode
        historyCommit={{
          hash: "abc1234",
          subject: "fix: adjust",
          checks: [{
            kind: "precommit",
            status: "failed",
            command: "custom check",
            exit_code: 7,
            summary: "checks failed",
            stderr: "nope",
            duration_ms: 25,
            timestamp: "2026-05-09T12:00:00Z"
          }]
        }}
      />
    );

    expect(screen.getByText("Commit Checks")).toBeInTheDocument();
    expect(screen.getByText("custom check")).toBeInTheDocument();
    expect(screen.getByText("failed")).toBeInTheDocument();
    expect(screen.getByText("nope")).toBeInTheDocument();
  });
});

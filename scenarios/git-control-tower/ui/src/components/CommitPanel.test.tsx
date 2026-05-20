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

  it("shows live precommit progress panel with elapsed and tail", () => {
    render(
      <CommitPanel
        stagedCount={1}
        commitMessage="fix: x"
        onCommitMessageChange={() => {}}
        onCommit={vi.fn()}
        isCommitting={true}
        precommitProgress={{
          running: true,
          command: "vrooli hygiene --fail-on error",
          elapsedMs: 12500,
          tail: ["check 1 ok", "check 2 ok", "running drift..."],
          onCancel: vi.fn(),
        }}
      />
    );

    const progress = screen.getByTestId("commit-precommit-progress");
    expect(progress).toBeInTheDocument();
    expect(progress).toHaveTextContent(/Running pre-commit/i);
    expect(progress).toHaveTextContent(/12\.5s/);
    expect(progress).toHaveTextContent(/running drift/);
    expect(screen.getByRole("button", { name: /cancel/i })).toBeInTheDocument();
  });

  it("hides precommit progress panel when not running and no failure", () => {
    render(
      <CommitPanel
        stagedCount={1}
        commitMessage="fix: x"
        onCommitMessageChange={() => {}}
        onCommit={vi.fn()}
        isCommitting={false}
        precommitProgress={{ running: false, elapsedMs: 0, tail: [] }}
      />
    );
    expect(screen.queryByTestId("commit-precommit-progress")).not.toBeInTheDocument();
    expect(screen.queryByTestId("commit-precommit-failure")).not.toBeInTheDocument();
  });

  it("renders persistent failure panel with output and actions when precommit failed", () => {
    const onDismiss = vi.fn();
    const onCommitAnyway = vi.fn();
    const onRunAgain = vi.fn();
    const onDisable = vi.fn();
    render(
      <CommitPanel
        stagedCount={1}
        commitMessage="fix: x"
        onCommitMessageChange={() => {}}
        onCommit={vi.fn()}
        isCommitting={false}
        precommitProgress={{
          running: false,
          elapsedMs: 12500,
          tail: [],
          failedResult: {
            status: "failed",
            exit_code: 1,
            summary: "Precommit checks failed",
            stdout: "running hygiene...",
            stderr: "drift detected",
            duration_ms: 12500,
            override_allowed: true,
            timestamp: "2026-05-20T22:14:59Z",
          },
          onDismissFailure: onDismiss,
          onCommitAnyway,
          onRunAgain,
          onDisable,
        }}
      />
    );

    const failure = screen.getByTestId("commit-precommit-failure");
    expect(failure).toBeInTheDocument();
    expect(failure).toHaveTextContent(/Pre-commit checks did not pass/i);
    expect(failure).toHaveTextContent(/Precommit checks failed/);
    expect(failure).toHaveTextContent(/drift detected/);
    expect(failure).toHaveTextContent(/exit 1/);

    fireEvent.click(screen.getByRole("button", { name: /Run Again/i }));
    expect(onRunAgain).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole("button", { name: /Commit Anyway/i }));
    expect(onCommitAnyway).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole("button", { name: /Disable Checks/i }));
    expect(onDisable).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByLabelText(/Dismiss pre-commit failure/i));
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("hides Commit Anyway when override is not allowed", () => {
    render(
      <CommitPanel
        stagedCount={1}
        commitMessage="fix: x"
        onCommitMessageChange={() => {}}
        onCommit={vi.fn()}
        isCommitting={false}
        precommitProgress={{
          running: false,
          elapsedMs: 0,
          tail: [],
          failedResult: {
            status: "failed",
            exit_code: 1,
            summary: "Precommit checks failed",
            stdout: "",
            stderr: "",
            duration_ms: 0,
            override_allowed: false,
            timestamp: "2026-05-20T22:14:59Z",
          },
          onCommitAnyway: vi.fn(),
        }}
      />
    );
    expect(screen.queryByRole("button", { name: /Commit Anyway/i })).not.toBeInTheDocument();
  });

  it("does not show the failure panel while the stream is still running", () => {
    render(
      <CommitPanel
        stagedCount={1}
        commitMessage="fix: x"
        onCommitMessageChange={() => {}}
        onCommit={vi.fn()}
        isCommitting={true}
        precommitProgress={{
          running: true,
          elapsedMs: 1000,
          tail: ["line"],
          failedResult: {
            status: "failed",
            exit_code: 1,
            summary: "old failure",
            stdout: "",
            stderr: "",
            duration_ms: 0,
            override_allowed: true,
            timestamp: "2026-05-20T22:14:59Z",
          },
        }}
      />
    );
    expect(screen.getByTestId("commit-precommit-progress")).toBeInTheDocument();
    expect(screen.queryByTestId("commit-precommit-failure")).not.toBeInTheDocument();
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

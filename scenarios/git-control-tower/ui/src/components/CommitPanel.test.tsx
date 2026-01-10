import "@testing-library/jest-dom";
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

    expect(screen.getByText(/ready to push 3 commits/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /push \(3 commits\)/i })).toBeInTheDocument();
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
});

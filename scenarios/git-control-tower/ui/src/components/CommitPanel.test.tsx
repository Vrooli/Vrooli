import "@testing-library/jest-dom";
import { render, screen } from "@testing-library/react";
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
});

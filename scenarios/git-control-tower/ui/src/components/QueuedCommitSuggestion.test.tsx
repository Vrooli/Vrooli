import { create } from "@bufbuild/protobuf";
import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { StagedMessageJobSchema, StagedMessageSuggestionSchema } from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";
import * as messageApi from "../features/commit-message/api";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import { CommitPanel } from "./CommitPanel";

const suggestion = create(StagedMessageSuggestionSchema, { id: "suggestion", title: "fix: queued update", body: "Preserve the queued content." });
const job = create(StagedMessageJobSchema, {
  id: "job-1", repositoryId: "repo-1", status: "succeeded", applicable: true,
  result: suggestion,
});
const selection = {
  id: "queue-1", requestKey: "queue-1", jobId: "job-1", paths: ["src/queued.ts"],
  queuedAt: "2026-11-12T12:00:00.000Z", state: "staged" as const,
  suggestionMode: "deep" as const, suggestionState: "succeeded" as const,
};

beforeEach(() => {
  vi.spyOn(messageApi, "getMessageJob").mockReset().mockResolvedValue(job);
  vi.spyOn(messageApi, "waitMessageJob").mockReset();
});

function renderPanel(commitMessage: string, onCommitMessageChange: (message: string) => void) {
  return renderWithProviders(<CommitPanel
    repoId="repo-1"
    stagedCount={1}
    commitMessage={commitMessage}
    onCommitMessageChange={onCommitMessageChange}
    onCommit={vi.fn()}
    isCommitting={false}
    queuedStageSelections={[selection]}
  />);
}

describe("queued commit suggestion adoption", () => {
  it("fills an empty editor only after the operator chooses Use", async () => {
    const onChange = vi.fn();
    renderPanel("", onChange);

    const use = await screen.findByRole("button", { name: "Use suggestion" });
    expect(onChange).not.toHaveBeenCalled();
    fireEvent.click(use);
    expect(onChange).toHaveBeenCalledWith("fix: queued update\n\nPreserve the queued content.");
  });

  it("shows a comparison instead of replacing existing human text", async () => {
    const onChange = vi.fn();
    renderPanel("keep my wording", onChange);

    fireEvent.click(await screen.findByRole("button", { name: "Use suggestion" }));
    expect(onChange).not.toHaveBeenCalled();
    expect(screen.getByTestId("queued-suggestion-comparison")).toHaveTextContent("keep my wording");
    expect(screen.getByTestId("queued-suggestion-comparison")).toHaveTextContent("fix: queued update");
    fireEvent.click(screen.getByRole("button", { name: "Replace with suggestion" }));
    expect(onChange).toHaveBeenCalledWith("fix: queued update\n\nPreserve the queued content.");
  });
});

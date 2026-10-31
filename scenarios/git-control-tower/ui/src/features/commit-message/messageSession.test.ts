import { describe, expect, it } from "vitest";
import { QueryClient } from "@tanstack/react-query";
import { create } from "@bufbuild/protobuf";
import { StagedMessageCollectionSchema, StagedMessageJobSchema } from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";
import { messageSessionKey, stagedSuggestionHold, type DraftSession } from "./messageSession";

const snapshot = { digest: "selection", repositoryId: "repo", files: [{ path: "a.go" }, { path: "b.go" }] };
const described = ["a.go", "b.go"];
function clientWith(session: Partial<DraftSession>, status?: string, applicable = true) {
  const client = new QueryClient();
  client.setQueryData<DraftSession>(messageSessionKey("repo"), { generation: 1, phase: "idle", initialMessage: "", selectedContextIDs: [], note: "", ...session });
  if (status) client.setQueryData(["commit-message", "job", "repo", "job-1"], create(StagedMessageJobSchema, {
    id: "job-1", repositoryId: "repo", snapshot, snapshotDigest: snapshot.digest, status, applicable,
    result: status === "succeeded" ? { id: "draft", title: "Draft" } : undefined,
  }));
  return client;
}

describe("stagedSuggestionHold", () => {
  it("holds staging while an agent drafts for the staged selection", () => {
    expect(stagedSuggestionHold(clientWith({ jobID: "job-1" }, "running"), "repo", described)).toEqual({ state: "drafting", paths: described });
  });

  it("holds staging while a completed suggestion or reusable message is ready", () => {
    expect(stagedSuggestionHold(clientWith({ jobID: "job-1" }, "succeeded"), "repo", described)?.state).toBe("ready");
    const collection = create(StagedMessageCollectionSchema, { id: "c", snapshot, suggestions: [{ id: "s", title: "Reuse" }] });
    expect(stagedSuggestionHold(clientWith({ phase: "collected", collection }), "repo", described)?.state).toBe("ready");
  });

  it("does not hold without a suggestion, after failure, or once the selection already diverged", () => {
    expect(stagedSuggestionHold(new QueryClient(), "repo", described)).toBeUndefined();
    expect(stagedSuggestionHold(clientWith({ jobID: "job-1" }, "failed"), "repo", described)).toBeUndefined();
    expect(stagedSuggestionHold(clientWith({ jobID: "job-1" }, "succeeded"), "repo", [...described, "c.go"])).toBeUndefined();
    expect(stagedSuggestionHold(clientWith({ jobID: "job-1" }, "succeeded", false), "repo", described)).toBeUndefined();
    expect(stagedSuggestionHold(clientWith({ jobID: "job-1", staleReason: "changed" }, "succeeded"), "repo", described)).toBeUndefined();
  });
});

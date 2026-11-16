import { describe, expect, it } from "vitest";
import { QueryClient } from "@tanstack/react-query";
import { create } from "@bufbuild/protobuf";
import { StagedMessageCollectionSchema, StagedMessageJobSchema, StagedMessageMode } from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";
import { initialSession, messageSessionKey, stagedSuggestionHold, storageKey, type DraftSession } from "./messageSession";

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
    expect(stagedSuggestionHold(clientWith({ jobID: "job-1" }, "running"), "repo")).toEqual({ state: "drafting", paths: described });
  });

  it("holds staging while a completed suggestion or reusable message is ready", () => {
    expect(stagedSuggestionHold(clientWith({ jobID: "job-1" }, "succeeded"), "repo")?.state).toBe("ready");
    const collection = create(StagedMessageCollectionSchema, { id: "c", snapshot, suggestions: [{ id: "s", title: "Reuse" }] });
    expect(stagedSuggestionHold(clientWith({ phase: "collected", collection }), "repo")?.state).toBe("ready");
  });

  it("holds staging for a saved suggestion even after the index diverges", () => {
    expect(stagedSuggestionHold(new QueryClient(), "repo")).toBeUndefined();
    expect(stagedSuggestionHold(clientWith({ jobID: "job-1" }, "failed"), "repo")).toBeUndefined();
    expect(stagedSuggestionHold(clientWith({ jobID: "job-1" }, "succeeded", false), "repo")?.state).toBe("ready");
    expect(stagedSuggestionHold(clientWith({ jobID: "job-1", staleReason: "changed" }, "succeeded"), "repo")?.state).toBe("ready");
  });
});

describe("draft admission recovery", () => {
  it("restores direct deep mode with the frozen request after a lost launch reply", () => {
    const admissionRequest = { repositoryId: "repo", collectionId: "collection", snapshotDigest: "subject", selectedContextIds: [], requestKey: "deep-key", operatorNote: "", mode: StagedMessageMode.DEEP };
    localStorage.setItem(storageKey("repo"), JSON.stringify({ requestKey: admissionRequest.requestKey, admissionRequest }));
    expect(initialSession("repo").admissionRequest).toEqual(admissionRequest);
    localStorage.removeItem(storageKey("repo"));
  });

  it("treats a restored pre-mode request as the compatible quick default", () => {
    const admissionRequest = { repositoryId: "repo", collectionId: "collection", snapshotDigest: "subject", selectedContextIds: [], requestKey: "old-key", operatorNote: "" };
    localStorage.setItem(storageKey("repo"), JSON.stringify({ requestKey: admissionRequest.requestKey, admissionRequest }));
    expect(initialSession("repo").admissionRequest?.mode).toBe(StagedMessageMode.QUICK);
    localStorage.removeItem(storageKey("repo"));
  });
});

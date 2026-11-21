import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { MutationPreviewSchema } from "@vrooli/proto-types/git-control-tower/v1/human_control/human_control_pb";
import { commitPreviewMatchesCurrent } from "./commit-authorization";

const preview = (change: "repository" | "operation" | "revision" | "subject" | "none" = "none") => create(MutationPreviewSchema, {
  repositoryId: change === "repository" ? "repo-b" : "repo-a",
  operation: change === "operation" ? "repo.push" : "repo.commit",
  expectedRevision: change === "revision" ? "revision-2" : "revision-1",
  subjectDigest: change === "subject" ? "digest-2" : "digest-1",
});

describe("commit preview freshness", () => {
  it("accepts an unchanged reviewed mutation", () => {
    expect(commitPreviewMatchesCurrent(preview(), preview())).toBe(true);
  });

  it.each(["repository", "operation", "revision", "subject"] as const)("rejects a changed %s", change => {
    expect(commitPreviewMatchesCurrent(preview(), preview(change))).toBe(false);
  });
});

import { create } from "@bufbuild/protobuf";
import { PushSafetyReportSchema } from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";
import { beforeEach, expect, test, vi } from "vitest";
const mocks = vi.hoisted(() => ({ inspect: vi.fn(), prepare: vi.fn(), status: vi.fn(), intent: vi.fn() }));
vi.mock("./connect", () => ({ repoClient: { inspectPushSafety: mocks.inspect, preparePushRecovery: mocks.prepare, getPushRecovery: mocks.status } }));
vi.mock("./api-core", () => ({ issueMutationIntentForOperation: mocks.intent }));
import { getPushRecovery, inspectPushSafety, preparePushRecovery } from "./api-push-safety";
beforeEach(() => vi.resetAllMocks());

test("inspection and reattachment use typed read requests without acquiring authority", async () => {
  await inspectPushSafety("repo-one"); await getPushRecovery("fingerprint", "repo-one");
  expect(mocks.inspect).toHaveBeenCalledWith(expect.objectContaining({ $typeName: "vrooli.git_control_tower.v1.repo.InspectPushSafetyRequest", repositoryId: "repo-one" }));
  expect(mocks.status).toHaveBeenCalledWith(expect.objectContaining({ fingerprint: "fingerprint", repositoryId: "repo-one" }));
  expect(mocks.intent).not.toHaveBeenCalled();
});
test("preparation binds human intent and the RPC to the exact inspected fingerprint", async () => {
  mocks.intent.mockResolvedValue({ repositoryId: "repo-one", intentId: "intent-one" });
  const report = create(PushSafetyReportSchema, { fingerprint: "preview", remote: "origin", branch: "agi" });
  await preparePushRecovery(report, "repo-one");
  expect(mocks.intent).toHaveBeenCalledWith("repo.recovery.prepare", "repo-one", "preview");
  expect(mocks.prepare).toHaveBeenCalledWith(expect.objectContaining({ fingerprint: "preview", intentId: "intent-one", repositoryId: "repo-one", branch: "agi", remote: "origin" }), { timeoutMs: 600000 });
});
test("authority refusal stops preparation at the client boundary", async () => {
  mocks.intent.mockRejectedValue(new Error("human authentication required"));
  await expect(preparePushRecovery(create(PushSafetyReportSchema, { fingerprint: "preview" }))).rejects.toThrow("human authentication required");
  expect(mocks.prepare).not.toHaveBeenCalled();
});

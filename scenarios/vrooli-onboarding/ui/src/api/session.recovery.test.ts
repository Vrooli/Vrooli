import { beforeEach, describe, expect, it, vi } from "vitest";

const rpc = vi.hoisted(() => ({ getProfileSession: vi.fn(), saveProfileSession: vi.fn() }));

vi.mock("@connectrpc/connect", async () => ({
  ...(await vi.importActual<typeof import("@connectrpc/connect")>("@connectrpc/connect")),
  createClient: () => rpc,
}));
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://onboarding.test/api/v1",
  createScenarioConnectTransport: () => ({}),
}));

import { Code, ConnectError } from "@connectrpc/connect";
import { saveProfileSession, type WizardProfileSessionSaveRequest } from "./session";

const request: WizardProfileSessionSaveRequest = {
  target: "node-1",
  expectedRevision: "r8",
  mode: "guided",
  profileId: "local-first",
  profileVersion: "3",
  catalogRevision: "catalog-12",
  consequenceDigest: "digest-5",
  baseRevision: "base-4",
  answers: { hosting: "local", region: "north" },
  manualDecisions: { "scenario-a": true },
  targetContext: { host: "node-1" },
};

function wireSession(revision: string, overrides: Record<string, unknown> = {}) {
  return {
    target: request.target,
    actor: "operator-1",
    mode: request.mode,
    profileId: request.profileId,
    profileVersion: request.profileVersion,
    catalogRevision: request.catalogRevision,
    consequenceDigest: request.consequenceDigest,
    baseRevision: request.baseRevision,
    answers: request.answers,
    manualDecisions: request.manualDecisions,
    targetContext: request.targetContext,
    reconciliationReasons: [],
    reconciliationChanges: [],
    revision,
    ...overrides,
  };
}

describe("profile session save recovery", () => {
  beforeEach(() => {
    rpc.getProfileSession.mockReset();
    rpc.saveProfileSession.mockReset();
  });

  it("reconciles an ambiguous response from the target owner without replaying the write", async () => {
    rpc.saveProfileSession.mockRejectedValue(new ConnectError("node response was lost", Code.Unavailable));
    rpc.getProfileSession.mockResolvedValue({ session: wireSession("r9", { answers: { region: "north", hosting: "local" } }) });

    await expect(saveProfileSession(request)).resolves.toMatchObject({
      target: "node-1", revision: "r9", answers: request.answers,
      manualDecisions: request.manualDecisions, targetContext: request.targetContext,
    });

    expect(rpc.saveProfileSession).toHaveBeenCalledTimes(1);
    expect(rpc.getProfileSession).toHaveBeenCalledWith({ target: "node-1" });
  });

  it("keeps the original failure when the owner has different work", async () => {
    const failure = new ConnectError("node response was lost", Code.DeadlineExceeded);
    rpc.saveProfileSession.mockRejectedValue(failure);
    rpc.getProfileSession.mockResolvedValue({ session: wireSession("r9", { answers: { hosting: "remote" } }) });

    await expect(saveProfileSession(request)).rejects.toBe(failure);
    expect(rpc.saveProfileSession).toHaveBeenCalledTimes(1);
    expect(rpc.getProfileSession).toHaveBeenCalledTimes(1);
  });

  it("does not turn a definite application error into a recovery read", async () => {
    const failure = new ConnectError("revision conflict", Code.Aborted);
    rpc.saveProfileSession.mockRejectedValue(failure);

    await expect(saveProfileSession(request)).rejects.toBe(failure);
    expect(rpc.getProfileSession).not.toHaveBeenCalled();
  });
});

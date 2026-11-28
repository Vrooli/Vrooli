import { beforeEach, describe, expect, it, vi } from "vitest";

const rpc = vi.hoisted(() => ({ getDraft: vi.fn(), saveDraft: vi.fn() }));

vi.mock("@connectrpc/connect", async () => ({
  ...(await vi.importActual<typeof import("@connectrpc/connect")>("@connectrpc/connect")),
  createClient: () => rpc,
}));
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://onboarding.test/api/v1",
  createScenarioConnectTransport: () => ({}),
}));

import { Code, ConnectError } from "@connectrpc/connect";
import { saveDraft, type WizardDraft, type WizardDraftSaveRequest } from "./session";

const request: WizardDraftSaveRequest = {
  target: "node-1",
  actor: "operator-1",
  expectedRevision: "r8",
  baseRevision: "setup-5",
  stepId: "core-set",
  choices: { "scenario.alpha": "enabled", region: "north" },
};

function draft(overrides: Partial<WizardDraft> = {}): WizardDraft {
  return {
    target: request.target,
    actor: request.actor,
    baseRevision: request.baseRevision,
    revision: "r9",
    stepId: request.stepId!,
    choices: { region: "north", "scenario.alpha": "enabled" },
    ...overrides,
  };
}

describe("draft save recovery", () => {
  beforeEach(() => {
    rpc.getDraft.mockReset();
    rpc.saveDraft.mockReset();
  });

  it("recognizes a committed draft after an ambiguous remote response without replay", async () => {
    rpc.saveDraft.mockRejectedValue(new ConnectError("node response was lost", Code.Unavailable));
    rpc.getDraft.mockResolvedValue(draft());

    await expect(saveDraft(request)).resolves.toEqual(draft());
    expect(rpc.saveDraft).toHaveBeenCalledTimes(1);
    expect(rpc.getDraft).toHaveBeenCalledWith({ target: "node-1", actor: "operator-1" });
  });

  it("preserves the original error when the owner has different saved work", async () => {
    const failure = new ConnectError("node response was lost", Code.DeadlineExceeded);
    rpc.saveDraft.mockRejectedValue(failure);
    rpc.getDraft.mockResolvedValue(draft({ choices: { region: "south" } }));

    await expect(saveDraft(request)).rejects.toBe(failure);
    expect(rpc.saveDraft).toHaveBeenCalledTimes(1);
    expect(rpc.getDraft).toHaveBeenCalledTimes(1);
  });

  it("does not read or retry after a definite revision conflict", async () => {
    const failure = new ConnectError("revision conflict", Code.Aborted);
    rpc.saveDraft.mockRejectedValue(failure);

    await expect(saveDraft(request)).rejects.toBe(failure);
    expect(rpc.getDraft).not.toHaveBeenCalled();
    expect(rpc.saveDraft).toHaveBeenCalledTimes(1);
  });
});

import { Code, ConnectError } from "@connectrpc/connect";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("../../api/workspaces", () => ({
  workspacesClient: { assistWorkspace: vi.fn() },
}));

import { workspacesClient } from "../../api/workspaces";
import { DesignProposalError } from "./proposal";
import { proposeWorkspacePatch, type WorkspaceProposeInput } from "./workspaceAssist";

function input(overrides: Partial<WorkspaceProposeInput> = {}): WorkspaceProposeInput {
  return {
    pageId: "onboarding",
    activeStates: [{ displayId: "desktop", stateId: "welcome" }],
    prompt: "Refine the first screen",
    scope: { kind: "layer", frameId: "desktop", nodeId: "headline" },
    revision: 8,
    idempotencyKey: "request-123",
    ...overrides,
  };
}

function response(
  proposalJson: string,
  overrides: { revision?: number; operation?: Record<string, string> } = {},
) {
  return {
    proposalJson,
    revision: overrides.revision ?? 9,
    operation: overrides.operation,
  } as unknown as Awaited<ReturnType<typeof workspacesClient.assistWorkspace>>;
}

function assistCall(mock = workspacesClient.assistWorkspace) {
  const call = vi.mocked(mock).mock.calls[0];
  if (!call) throw new Error("Expected the workspace assist transport to be called.");
  return call;
}

describe("proposeWorkspacePatch", () => {
  afterEach(() => vi.clearAllMocks());

  it("maps workspace context into the Connect request and returns the parsed operation", async () => {
    const proposalJson = JSON.stringify({
      summary: "Clearer first step",
      operations: [{ kind: "setText", nodeId: "headline", text: "Start here" }],
    });
    vi.mocked(workspacesClient.assistWorkspace).mockResolvedValue(
      response(proposalJson, {
        operation: { id: "operation-1", state: "pending", provider: "local", model: "fixture" },
      }),
    );
    const controller = new AbortController();

    const result = await proposeWorkspacePatch("workspace-a", input(), controller.signal);

    expect(workspacesClient.assistWorkspace).toHaveBeenCalledWith(
      {
        id: "workspace-a",
        baseRevision: 8,
        pageId: "onboarding",
        activeStates: [{ displayId: "desktop", stateId: "welcome" }],
        prompt: "Refine the first screen",
        scope: { kind: "layer", displayId: "desktop", nodeId: "headline" },
        idempotencyKey: "request-123",
      },
      { signal: controller.signal },
    );
    expect(result).toEqual({
      summary: "Clearer first step",
      operations: [{ kind: "setText", nodeId: "headline", text: "Start here" }],
      revision: 9,
      operationId: "operation-1",
      state: "pending",
      provider: "local",
      model: "fixture",
    });
  });

  it.each([
    [
      { kind: "frame", frameId: "mobile" },
      { kind: "frame", displayId: "mobile", nodeId: "" },
    ],
    [{ kind: "design" }, { kind: "design", displayId: "", nodeId: "" }],
  ] as const)("normalizes absent target ids for %s scope", async (scope, expectedScope) => {
    vi.mocked(workspacesClient.assistWorkspace).mockResolvedValue(
      response(JSON.stringify({ operations: [{ kind: "setTheme", key: "bg", value: "#fff" }] })),
    );
    await proposeWorkspacePatch("workspace-a", input({ scope }));

    expect(assistCall()[0].scope).toEqual(expectedScope);
  });

  it("supplies safe defaults when the provider omits optional response metadata", async () => {
    vi.mocked(workspacesClient.assistWorkspace).mockResolvedValue(
      response(JSON.stringify({ operations: [{ kind: "setTheme" }] })),
    );

    await expect(
      proposeWorkspacePatch("workspace-a", input({ scope: { kind: "design" } })),
    ).resolves.toMatchObject({
      summary: "",
      operationId: "",
      state: "completed",
      revision: 9,
    });
  });

  it.each([
    [Code.Aborted, "conflict"],
    [Code.AlreadyExists, "conflict"],
    [Code.Unavailable, "ai_unavailable"],
    [Code.FailedPrecondition, "invalid_proposal"],
    [Code.InvalidArgument, "invalid_argument"],
    [Code.Internal, "ai_failed"],
  ] as const)("maps Connect status %s to the studio error contract", async (code, expectedCode) => {
    vi.mocked(workspacesClient.assistWorkspace).mockRejectedValue(
      new ConnectError("provider refused", code),
    );

    await expect(proposeWorkspacePatch("workspace-a", input())).rejects.toMatchObject({
      name: "DesignProposalError",
      code: expectedCode,
      status: 0,
      message: "provider refused",
    });
  });

  it("reports cancellation consistently when an aborted request rejects", async () => {
    const controller = new AbortController();
    vi.mocked(workspacesClient.assistWorkspace).mockImplementation(() => {
      controller.abort();
      return Promise.reject(new Error("aborted"));
    });

    await expect(
      proposeWorkspacePatch("workspace-a", input(), controller.signal),
    ).rejects.toMatchObject({
      name: "DesignProposalError",
      code: "ai_unavailable",
      message: "The request was canceled.",
    });
  });

  it.each([
    ["not-json", "The gateway returned unreadable JSON."],
    [JSON.stringify({ summary: "No operations" }), "The gateway returned no patch."],
    [JSON.stringify({ operations: [] }), "The gateway returned no patch."],
  ])("rejects malformed or empty provider patches", async (proposalJson, message) => {
    vi.mocked(workspacesClient.assistWorkspace).mockResolvedValue(response(proposalJson));

    await expect(proposeWorkspacePatch("workspace-a", input())).rejects.toEqual(
      new DesignProposalError("invalid_proposal", 0, message),
    );
  });
});

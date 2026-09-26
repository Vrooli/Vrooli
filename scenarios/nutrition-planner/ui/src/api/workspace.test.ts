import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { WorkspaceSchema } from "@vrooli/proto-types/nutrition-planner/v1/workspace/workspace_pb";

const listWorkspaces = vi.hoisted(() => vi.fn());
const createWorkspace = vi.hoisted(() => vi.fn());
vi.mock("@connectrpc/connect", () => ({ createClient: () => ({ listWorkspaces, createWorkspace }) }));
vi.mock("./client", () => ({ transport: {} }));

import { ensureWorkspace } from "./workspace";

describe("workspace API", () => {
  beforeEach(() => vi.clearAllMocks());

  it("reuses the first workspace", async () => {
    const workspace = create(WorkspaceSchema, { id: "w1", name: "Daily" });
    listWorkspaces.mockResolvedValue({ workspaces: [workspace] });
    await expect(ensureWorkspace()).resolves.toBe(workspace);
    expect(createWorkspace).not.toHaveBeenCalled();
  });

  it("creates a workspace when none exists", async () => {
    const workspace = create(WorkspaceSchema, { id: "w1", name: "Daily" });
    listWorkspaces.mockResolvedValue({ workspaces: [] });
    createWorkspace.mockResolvedValue({ workspace });
    await expect(ensureWorkspace()).resolves.toBe(workspace);
    expect(createWorkspace).toHaveBeenCalledWith(expect.objectContaining({ name: "My Daily workspace", idempotencyKey: expect.any(String) }));
  });

  it("rejects a hollow create response", async () => {
    listWorkspaces.mockResolvedValue({ workspaces: [] });
    createWorkspace.mockResolvedValue({});
    await expect(ensureWorkspace()).rejects.toThrow("no workspace");
  });
});

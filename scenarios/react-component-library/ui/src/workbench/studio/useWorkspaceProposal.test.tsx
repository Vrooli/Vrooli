import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { DisplayDevice, StateStatus, type Workspace } from "../../api/workspaces";
import { docSignature } from "./model";
import { pageDocument } from "../workspaces/spec";
import { node } from "../templates/spec";
import type { StudioProposal } from "./proposal";
import { useWorkspaceProposal } from "./useWorkspaceProposal";

type Apply = Parameters<typeof useWorkspaceProposal>[0]["apply"];

function fixture() {
  const spec = node("page", "page", {}, [node("headline", "text", { text: "Hello" })]);
  const state = {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceState" as const,
    id: "state-1",
    name: "Default",
    argsJson: JSON.stringify({ spec }),
    scenarioDataJson: "{}",
    status: StateStatus.DRAFT,
    captureId: "",
    sortOrder: 0,
  };
  const page = {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspacePage" as const,
    id: "page-1",
    name: "Home",
    route: "/",
    pageTypeId: "",
    sortOrder: 0,
    specJson: JSON.stringify(spec),
    displays: [
      {
        $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceDisplay" as const,
        id: "phone",
        device: DisplayDevice.PHONE,
        width: 390,
        height: 844,
        sortOrder: 0,
        states: [state],
      },
    ],
  };
  const workspace = {
    $typeName: "vrooli.react_component_library.v1.workspace.Workspace" as const,
    id: "workspace-1",
    name: "Workspace",
    summary: "",
    provenance: 0,
    tags: [],
    bindingScenario: "",
    bindingPage: "",
    themeJson: "{}",
    optionsJson: "{}",
    revision: 3,
    createdAt: "",
    updatedAt: "",
    originWorkspaceId: "",
    pages: [page],
  } as Workspace;
  const states = { phone: state.id };
  const signature = docSignature(pageDocument(workspace, page, states));
  return { workspace, page, state, signature };
}

function proposal(signature: string, overrides: Partial<StudioProposal> = {}): StudioProposal {
  return {
    summary: "Update heading",
    operations: [{ kind: "setText", nodeId: "headline", text: "Welcome" }],
    revision: 3,
    operationId: "operation-1",
    state: "state-1",
    designId: "workspace-1",
    signature,
    ...overrides,
  };
}

function setup(current: Workspace | null = fixture().workspace, pageId = "page-1") {
  const workspaceRef = { current };
  const pageIdRef = { current: pageId };
  const statesRef = { current: { phone: "state-1" } };
  const apply = vi.fn<Apply>();
  const { result } = renderHook(() =>
    useWorkspaceProposal({ workspaceRef, pageIdRef, statesRef, apply }),
  );
  return { result, workspaceRef, pageIdRef, statesRef, apply };
}

describe("useWorkspaceProposal", () => {
  it("refuses application when the workspace or active page is unavailable", () => {
    const empty = setup(null);
    expect(empty.result.current(proposal("anything"))).toMatchObject({
      ok: false,
      reason: "invalid",
      detail: "There is no active page.",
    });

    const missingPage = setup({ ...fixture().workspace, pages: [] }, "missing");
    expect(missingPage.result.current(proposal("anything"))).toMatchObject({
      ok: false,
      reason: "invalid",
      detail: "There is no active page.",
    });
    expect(missingPage.apply).not.toHaveBeenCalled();
  });

  it("rejects proposals for a different design or composition as stale", () => {
    const { result, apply } = setup();
    const current = fixture();

    expect(
      result.current(proposal(current.signature, { designId: "another-workspace" })),
    ).toMatchObject({ ok: false, reason: "stale" });
    expect(result.current(proposal("old-composition-signature"))).toMatchObject({
      ok: false,
      reason: "stale",
    });
    expect(apply).not.toHaveBeenCalled();
  });

  it("applies valid text changes to the active display state without rewriting theme", () => {
    const current = fixture();
    const { result, apply } = setup(current.workspace);

    expect(result.current(proposal(current.signature))).toEqual({ ok: true });
    expect(apply).toHaveBeenCalledOnce();
    expect(apply).toHaveBeenCalledWith("AI · Update heading", undefined, expect.any(Function));
    const change = apply.mock.calls[0]?.[2];
    if (!change) throw new Error("proposal change was not provided");
    const draft = structuredClone(current.workspace);
    change(draft);
    const stateJson = JSON.parse(draft.pages[0]!.displays[0]!.states[0]!.argsJson);
    expect(stateJson.spec.children[0].props.text).toBe("Welcome");
    expect(draft.themeJson).toBe(current.workspace.themeJson);
  });

  it("persists a theme only when the accepted patch contains a theme operation", () => {
    const current = fixture();
    const { result, apply } = setup(current.workspace);
    const themed = proposal(current.signature, {
      operations: [{ kind: "setTheme", key: "accent", value: "#123abc" }],
    });

    expect(result.current(themed)).toEqual({ ok: true });
    const change = apply.mock.calls[0]?.[2];
    if (!change) throw new Error("theme change was not provided");
    const draft = structuredClone(current.workspace);
    change(draft);
    expect(JSON.parse(draft.themeJson).accent).toBe("#123abc");
  });

  it("returns actionable invalid results for rejected patches and malformed operations", () => {
    const current = fixture();
    const { result, apply } = setup(current.workspace);

    expect(
      result.current(
        proposal(current.signature, {
          operations: [{ kind: "setProp", nodeId: "headline", key: "onClick", value: "run" }],
        }),
      ),
    ).toMatchObject({
      ok: false,
      reason: "invalid",
      detail: expect.stringContaining("cannot be set"),
    });
    expect(result.current(proposal(current.signature, { operations: null as never }))).toEqual({
      ok: false,
      reason: "invalid",
      detail: "The patch could not be applied.",
    });
    expect(apply).not.toHaveBeenCalled();
  });
});

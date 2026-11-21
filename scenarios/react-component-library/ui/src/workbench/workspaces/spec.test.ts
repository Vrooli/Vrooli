import { describe, expect, it } from "vitest";

import { DisplayDevice, StateStatus, WorkspaceProvenance } from "../../api/workspaces";
import type { Workspace, WorkspaceDisplay, WorkspacePage, WorkspaceState } from "../../api/workspaces";
import {
  deviceOf,
  displaySpec,
  pageDocument,
  parseOptions,
  parsePalette,
  protoDevice,
  readStateSpec,
  writeStateSpec,
} from "./spec";

function node(id: string) {
  return { id, type: "page" as const, props: {} };
}

function state(overrides: Partial<WorkspaceState> = {}): WorkspaceState {
  return {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceState",
    id: "st_default",
    name: "Default",
    argsJson: "{}",
    scenarioDataJson: "{}",
    status: StateStatus.DRAFT,
    captureId: "",
    sortOrder: 0,
    ...overrides,
  };
}

const display: WorkspaceDisplay = {
  $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceDisplay",
  id: "dp_phone",
  device: DisplayDevice.PHONE,
  width: 390,
  height: 844,
  sortOrder: 0,
  states: [state()],
};

const page: WorkspacePage = {
  $typeName: "vrooli.react_component_library.v1.workspace.WorkspacePage",
  id: "pg_welcome",
  name: "Welcome",
  route: "/onboarding",
  pageTypeId: "templates.onboarding",
  sortOrder: 0,
  specJson: "{}",
  displays: [display],
};

const workspace: Workspace = {
  $typeName: "vrooli.react_component_library.v1.workspace.Workspace",
  id: "ws_harbor",
  name: "Harbor",
  summary: "",
  provenance: WorkspaceProvenance.AUTHORED,
  tags: [],
  bindingScenario: "harbor",
  bindingPage: "welcome",
  themeJson: '{"accent":"#f00"}',
  optionsJson: '{"radius":8,"display":"serif"}',
  revision: 4,
  createdAt: "2026-11-19T10:00:00Z",
  updatedAt: "2026-11-19T11:00:00Z",
  originWorkspaceId: "",
  pages: [page],
};

describe("workspace spec envelope", () => {
  it("reads a state spec from the envelope and from a bare node", () => {
    expect(readStateSpec(state({ argsJson: JSON.stringify({ spec: node("a") }) }))).toEqual(node("a"));
    expect(readStateSpec(state({ argsJson: JSON.stringify(node("b")) }))).toEqual(node("b"));
    expect(readStateSpec(state({ argsJson: '{"variant":"filled"}' }))).toBeNull();
    expect(readStateSpec(undefined)).toBeNull();
  });

  it("writes the spec without discarding prop overrides", () => {
    const target = state({ argsJson: '{"variant":"filled"}' });
    writeStateSpec(target, node("c"));
    expect(JSON.parse(target.argsJson)).toEqual({ variant: "filled", spec: node("c") });
  });

  it("prefers a display composition over the shared page composition", () => {
    const own = { ...display, states: [state({ argsJson: JSON.stringify({ spec: node("own") }) })] };
    expect(displaySpec(page, own).id).toBe("own");
    const shared = { ...display, states: [state()] };
    expect(displaySpec({ ...page, specJson: JSON.stringify(node("page")) }, shared).id).toBe("page");
  });

  it("maps device vocabularies both ways", () => {
    expect(deviceOf(DisplayDevice.PHONE)).toBe("phone");
    expect(deviceOf(DisplayDevice.TABLET)).toBe("tablet");
    expect(deviceOf(DisplayDevice.DESKTOP)).toBe("desktop");
    expect(protoDevice("phone")).toBe(DisplayDevice.PHONE);
  });

  it("projects a page into the document the studio renders", () => {
    const doc = pageDocument(workspace, page);
    expect(doc).toMatchObject({
      id: "ws_harbor",
      name: "Welcome",
      revision: 4,
      scenario: "harbor",
      page: "welcome",
      options: { radius: 8, display: "serif" },
    });
    expect(doc.frames).toHaveLength(1);
    expect(doc.frames[0]).toMatchObject({ id: "dp_phone", device: "phone", name: "Phone" });
    expect(doc.theme.accent).toBe("#f00");
  });

  it("falls back to the default palette and drops unusable options", () => {
    expect(parsePalette("{}").accent).toBeTruthy();
    expect(parseOptions('{"radius":"big","display":"comic"}')).toEqual({});
    expect(parseOptions("not json")).toEqual({});
  });
});

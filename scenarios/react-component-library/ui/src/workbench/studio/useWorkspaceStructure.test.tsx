import type { Dispatch, SetStateAction } from "react";
import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { DisplayDevice, StateStatus, type Workspace } from "../../api/workspaces";
import { node } from "../templates/spec";
import { useWorkspaceStructure } from "./useWorkspaceStructure";
import type { WorkspaceDisplay, WorkspacePage, WorkspaceState } from "../../api/workspaces";

type Options = Parameters<typeof useWorkspaceStructure>[0];

function state(
  id: string,
  name: string,
  spec = node(`${id}-page`, "page", {}, [node(`${id}-copy`, "text", { text: name })]),
): WorkspaceState {
  return {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceState",
    id,
    name,
    argsJson: JSON.stringify({ spec }),
    scenarioDataJson: "{}",
    status: StateStatus.DRAFT,
    captureId: "",
    sortOrder: 0,
  };
}

function display(id: string, device: DisplayDevice, states: WorkspaceState[]): WorkspaceDisplay {
  return {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceDisplay",
    id,
    device,
    width: device === DisplayDevice.PHONE ? 390 : 1180,
    height: device === DisplayDevice.PHONE ? 844 : 780,
    sortOrder: 0,
    states,
  };
}

function page(id: string, name: string, displays: WorkspaceDisplay[]): WorkspacePage {
  return {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspacePage",
    id,
    name,
    route: `/${id}`,
    pageTypeId: "",
    sortOrder: 0,
    specJson: JSON.stringify(node(`${id}-root`, "page", {}, [])),
    displays,
  };
}

function workspaceFixture(): Workspace {
  const desktop = display("desktop", DisplayDevice.DESKTOP, [state("desktop-default", "Desktop")]);
  const phone = display("phone", DisplayDevice.PHONE, [
    state("phone-default", "Phone"),
    state("phone-error", "Error"),
  ]);
  const first = page("first", "First page", [desktop, phone]);
  const second = page("second", "Second page", [
    display("second-desktop", DisplayDevice.DESKTOP, [state("second-default", "Second")]),
  ]);
  return {
    $typeName: "vrooli.react_component_library.v1.workspace.Workspace",
    id: "workspace-1",
    name: "Workspace",
    summary: "",
    provenance: 0,
    tags: [],
    bindingScenario: "",
    bindingPage: "",
    themeJson: "{}",
    optionsJson: "{}",
    revision: 1,
    createdAt: "",
    updatedAt: "",
    originWorkspaceId: "",
    pages: [first, second],
  };
}

function setup(overrides: Partial<Options> = {}) {
  const workspace = workspaceFixture();
  const pageModel = workspace.pages[0]!;
  const displayModel = pageModel.displays[1]!;
  const selectedState = displayModel.states[1]!;
  const workspaceRef = { current: workspace };
  const setPageId = vi.fn<Dispatch<SetStateAction<string>>>();
  const setDisplayId = vi.fn<Dispatch<SetStateAction<string>>>();
  const setStateByDisplay = vi.fn<Dispatch<SetStateAction<Record<string, string>>>>();
  const setSelectedId = vi.fn<Dispatch<SetStateAction<string | null>>>();
  const applied: Array<{ label: string; draft: Workspace }> = [];
  const apply: Options["apply"] = (label, _mergeKey, change) => {
    const draft = structuredClone(workspace);
    change(draft);
    applied.push({ label, draft });
  };
  const options: Options = {
    workspace,
    workspaceRef,
    page: pageModel,
    display: displayModel,
    state: selectedState,
    displayId: displayModel.id,
    stateByDisplay: { [displayModel.id]: selectedState.id },
    apply,
    setPageId,
    setDisplayId,
    setStateByDisplay,
    setSelectedId,
    ...overrides,
  };
  const hook = renderHook(() => useWorkspaceStructure(options));
  return {
    ...hook,
    workspace,
    pageModel,
    displayModel,
    selectedState,
    setPageId,
    setDisplayId,
    setStateByDisplay,
    setSelectedId,
    applied,
  };
}

function nextValue<T>(setter: ReturnType<typeof vi.fn>, previous: T): T {
  const updater = setter.mock.calls[0]?.[0] as SetStateAction<T> | undefined;
  if (typeof updater === "function") return (updater as (value: T) => T)(previous);
  if (updater === undefined) throw new Error("state updater was not called");
  return updater;
}

describe("useWorkspaceStructure navigation", () => {
  it("selects only existing pages and clears layer selection on navigation", () => {
    const { result, setPageId, setDisplayId, setSelectedId } = setup();

    act(() => result.current.selectPage("second"));
    expect(setPageId).toHaveBeenCalledWith("second");
    expect(setDisplayId).toHaveBeenCalledWith("second-desktop");
    expect(setSelectedId).toHaveBeenCalledWith(null);

    setPageId.mockClear();
    act(() => result.current.selectPage("missing"));
    expect(setPageId).not.toHaveBeenCalled();
  });

  it("applies address, display, frame, and state selection with valid state mapping", () => {
    const { result, setPageId, setDisplayId, setStateByDisplay, setSelectedId } = setup();

    act(() => result.current.selectAddress("second", "second-desktop", "second-default"));
    expect(setPageId).toHaveBeenCalledWith("second");
    expect(setDisplayId).toHaveBeenCalledWith("second-desktop");
    expect(nextValue(setStateByDisplay, {})).toEqual({ "second-desktop": "second-default" });
    expect(setSelectedId).toHaveBeenCalledWith(null);

    setStateByDisplay.mockClear();
    act(() => result.current.selectAddress("missing", "", ""));
    expect(setPageId).toHaveBeenCalledTimes(1);
    expect(setDisplayId).toHaveBeenLastCalledWith("");
    expect(setStateByDisplay).not.toHaveBeenCalled();

    act(() => result.current.selectDisplay("desktop"));
    expect(setDisplayId).toHaveBeenLastCalledWith("desktop");
    act(() => result.current.selectFrame("phone"));
    expect(setDisplayId).toHaveBeenLastCalledWith("phone");
    act(() => result.current.selectFrame("unknown"));
    expect(setDisplayId).toHaveBeenLastCalledWith("phone");
    act(() => result.current.selectState("phone-default"));
    expect(nextValue(setStateByDisplay, {})).toEqual({ phone: "phone-default" });
  });
});

describe("useWorkspaceStructure page and display mutations", () => {
  it("adds and duplicates pages with independent display/state ids", () => {
    const { result, applied, setPageId, setDisplayId } = setup();

    act(() => result.current.addPage("  New flow  "));
    expect(applied[0]?.label).toBe("Add page");
    const added = applied[0]?.draft.pages.at(-1);
    expect(added?.name).toBe("New flow");
    expect(added?.displays).toHaveLength(1);
    expect(setPageId).toHaveBeenCalledWith(added?.id);
    expect(setDisplayId).toHaveBeenCalledWith(added?.displays[0]?.id);

    act(() => result.current.duplicatePage());
    expect(applied[1]?.label).toBe("Duplicate page");
    const duplicate = applied[1]?.draft.pages.at(-1);
    expect(duplicate?.name).toBe("First page copy");
    expect(duplicate?.id).not.toBe("first");
    expect(duplicate?.displays[1]?.id).not.toBe("phone");
    expect(duplicate?.displays[1]?.states[0]?.id).not.toBe("phone-default");
  });

  it("removes pages and frames while preserving a usable selection", () => {
    const { result, applied, setPageId, setDisplayId, setStateByDisplay, setSelectedId } = setup();

    act(() => result.current.removePage());
    expect(applied[0]?.draft.pages.map((entry) => entry.id)).toEqual(["second"]);
    expect(applied[0]?.draft.pages[0]?.sortOrder).toBe(0);
    expect(setPageId).toHaveBeenCalledWith("second");
    expect(setDisplayId).toHaveBeenCalledWith("second-desktop");
    expect(setSelectedId).toHaveBeenCalledWith(null);

    act(() => result.current.removeFrame("phone"));
    expect(applied[1]?.draft.pages[0]?.displays.map((entry) => entry.id)).toEqual(["desktop"]);
    expect(
      nextValue(setStateByDisplay, { phone: "phone-error", desktop: "desktop-default" }),
    ).toEqual({ desktop: "desktop-default" });
    expect(setDisplayId).toHaveBeenLastCalledWith("desktop");
  });

  it("adds a display by copying every source state and its selected state", () => {
    const { result, applied, setDisplayId, setStateByDisplay, setSelectedId } = setup();

    act(() => result.current.addFrame("tablet"));
    const added = applied[0]?.draft.pages[0]?.displays.at(-1);
    expect(applied[0]?.label).toBe("Add Tablet display");
    expect(added?.width).toBe(834);
    expect(added?.states.map((entry) => entry.name)).toEqual(["Phone", "Error"]);
    expect(added?.states[0]?.id).not.toBe("phone-default");
    expect(setDisplayId).toHaveBeenCalledWith(added?.id);
    expect(nextValue(setStateByDisplay, {})).toEqual({ [added?.id ?? ""]: added?.states[1]?.id });
    expect(setSelectedId).toHaveBeenCalledWith(null);
  });
});

describe("useWorkspaceStructure state mutations", () => {
  it("adds, renames and removes states while maintaining order and selection", () => {
    const { result, applied, setStateByDisplay, setSelectedId } = setup();

    act(() => result.current.addState());
    const added = applied[0]?.draft.pages[0]?.displays[1]?.states.at(-1);
    expect(applied[0]?.label).toBe("Add state");
    expect(added?.name).toBe("State 3");
    expect(setSelectedId).toHaveBeenCalledWith(null);
    expect(nextValue(setStateByDisplay, {})).toEqual({ phone: added?.id });

    act(() => result.current.renameState("Validation errors"));
    expect(applied[1]?.label).toBe("Rename state");
    expect(applied[1]?.draft.pages[0]?.displays[1]?.states[1]?.name).toBe("Validation errors");

    setStateByDisplay.mockClear();
    act(() => result.current.removeState());
    expect(applied[2]?.label).toBe("Remove state");
    expect(applied[2]?.draft.pages[0]?.displays[1]?.states.map((entry) => entry.id)).toEqual([
      "phone-default",
    ]);
    expect(applied[2]?.draft.pages[0]?.displays[1]?.states[0]?.sortOrder).toBe(0);
    expect(nextValue(setStateByDisplay, { phone: "phone-error" })).toEqual({});
  });

  it("keeps required page, display and state structures when removal is unavailable", () => {
    const workspace = workspaceFixture();
    const onlyPage = workspace.pages[0]!;
    workspace.pages = [onlyPage];
    const onlyDisplay = onlyPage.displays[0]!;
    onlyPage.displays = [onlyDisplay];
    onlyDisplay.states = [onlyDisplay.states[0]!];
    const { result, applied } = setup({
      workspace,
      workspaceRef: { current: workspace },
      page: onlyPage,
      display: onlyDisplay,
      state: onlyDisplay.states[0]!,
    });

    act(() => result.current.removePage());
    act(() => result.current.removeFrame(onlyDisplay.id));
    act(() => result.current.removeState());
    expect(applied).toHaveLength(0);
  });
});

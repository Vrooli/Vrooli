import { useCallback, type Dispatch, type MutableRefObject, type SetStateAction } from "react";

import {
  StateStatus,
  type Workspace,
  type WorkspaceDisplay,
  type WorkspacePage,
  type WorkspaceState,
} from "../../api/workspaces";
import type { DesignDevice } from "../design-types";
import { DEVICE_NAME, rekeySpec } from "../studio/model";
import { activeStateOf, displaySizes, protoDevice, readStateSpec, seedSpec, writeStateSpec } from "../workspaces/spec";
import type { TemplateNode } from "../templates/spec";

type ApplyWorkspaceChange = (
  label: string,
  mergeKey: string | undefined,
  change: (draft: Workspace) => void,
) => void;

interface WorkspaceStructureOptions {
  workspace: Workspace | null;
  workspaceRef: MutableRefObject<Workspace | null>;
  page: WorkspacePage | null;
  display: WorkspaceDisplay | null;
  state: WorkspaceState | null;
  displayId: string;
  stateByDisplay: Record<string, string>;
  apply: ApplyWorkspaceChange;
  setPageId: Dispatch<SetStateAction<string>>;
  setDisplayId: Dispatch<SetStateAction<string>>;
  setStateByDisplay: Dispatch<SetStateAction<Record<string, string>>>;
  setSelectedId: Dispatch<SetStateAction<string | null>>;
}

function withoutKey(record: Record<string, string>, key: string): Record<string, string> {
  return Object.fromEntries(Object.entries(record).filter(([entry]) => entry !== key));
}

function localId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 10)}${Date.now().toString(36).slice(-6)}`;
}

function makeDisplay(device: DesignDevice, spec: TemplateNode): WorkspaceDisplay {
  const size = displaySizes(device);
  return {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceDisplay",
    id: localId("dp"),
    device: protoDevice(device),
    width: size.width,
    height: size.height,
    sortOrder: 0,
    states: [makeState("Default", spec, 0)],
  };
}

function makeState(name: string, spec: TemplateNode, order: number): WorkspaceState {
  const entry: WorkspaceState = {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceState",
    id: localId("st"),
    name,
    argsJson: "{}",
    scenarioDataJson: "{}",
    status: StateStatus.DRAFT,
    captureId: "",
    sortOrder: order,
  };
  writeStateSpec(entry, spec);
  return entry;
}

function makePage(name: string, spec: TemplateNode, order: number): WorkspacePage {
  return {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspacePage",
    id: localId("pg"),
    name,
    route: "",
    pageTypeId: "",
    sortOrder: order,
    specJson: JSON.stringify(spec),
    displays: [makeDisplay("desktop", spec)],
  };
}

/** Owns page/display/state structure mutations and navigation within a workspace. */
export function useWorkspaceStructure({
  workspace,
  workspaceRef,
  page,
  display,
  state,
  displayId,
  stateByDisplay,
  apply,
  setPageId,
  setDisplayId,
  setStateByDisplay,
  setSelectedId,
}: WorkspaceStructureOptions) {
  const selectPage = useCallback(
    (id: string) => {
      const target = workspace?.pages.find((entry) => entry.id === id);
      if (!target) return;
      setPageId(id);
      setDisplayId(target.displays[0]?.id ?? "");
      setSelectedId(null);
    },
    [setDisplayId, setPageId, setSelectedId, workspace],
  );

  const selectAddress = useCallback(
    (nextPage: string, nextDisplay: string, nextState: string) => {
      if (workspaceRef.current) {
        const target = workspaceRef.current.pages.find((entry) => entry.id === nextPage);
        if (target) setPageId(target.id);
      }
      setDisplayId(nextDisplay);
      if (nextDisplay && nextState) {
        setStateByDisplay((current) => ({ ...current, [nextDisplay]: nextState }));
      }
      setSelectedId(null);
    },
    [setDisplayId, setPageId, setSelectedId, setStateByDisplay, workspaceRef],
  );

  const selectDisplay = useCallback(
    (id: string) => {
      setDisplayId(id);
      setSelectedId(null);
    },
    [setDisplayId, setSelectedId],
  );

  const selectFrame = useCallback(
    (id: string) => {
      if (!page?.displays.some((entry) => entry.id === id)) return;
      setDisplayId(id);
      setSelectedId(null);
    },
    [page, setDisplayId, setSelectedId],
  );

  const selectState = useCallback(
    (id: string) => {
      if (!display) return;
      setStateByDisplay((current) => ({ ...current, [display.id]: id }));
      setSelectedId(null);
    },
    [display, setSelectedId, setStateByDisplay],
  );

  const addPage = useCallback(
    (name?: string, spec?: TemplateNode) => {
      const added = makePage(name?.trim() || "New page", spec ?? seedSpec("page"), 0);
      apply("Add page", undefined, (draft) => {
        added.sortOrder = draft.pages.length;
        draft.pages.push(added);
      });
      setPageId(added.id);
      setDisplayId(added.displays[0]?.id ?? "");
      setSelectedId(null);
    },
    [apply, setDisplayId, setPageId, setSelectedId],
  );

  const duplicatePage = useCallback(() => {
    if (!page) return;
    const copy = structuredClone(page);
    copy.id = localId("pg");
    copy.name = `${page.name} copy`;
    copy.sortOrder = page.sortOrder + 1;
    copy.displays = copy.displays.map((entry) => {
      const nextDisplayId = localId("dp");
      const states = entry.states.map((stateEntry) => ({ ...stateEntry, id: localId("st") }));
      const first = states[0];
      if (first) {
        const spec = readStateSpec(first);
        if (spec) writeStateSpec(first, rekeySpec(spec, nextDisplayId));
      }
      return { ...entry, id: nextDisplayId, states };
    });
    apply("Duplicate page", undefined, (draft) => draft.pages.push(copy));
    setPageId(copy.id);
    setDisplayId(copy.displays[0]?.id ?? "");
  }, [apply, page, setDisplayId, setPageId]);

  const removePage = useCallback(() => {
    if (!page || !workspace || workspace.pages.length <= 1) return;
    const removed = page.id;
    apply("Remove page", undefined, (draft) => {
      draft.pages = draft.pages.filter((entry) => entry.id !== removed);
      draft.pages.forEach((entry, index) => {
        entry.sortOrder = index;
      });
    });
    const next = workspace.pages.find((entry) => entry.id !== removed);
    setPageId(next?.id ?? "");
    setDisplayId(next?.displays[0]?.id ?? "");
    setSelectedId(null);
  }, [apply, page, setDisplayId, setPageId, setSelectedId, workspace]);

  const addFrame = useCallback(
    (device: DesignDevice) => {
      if (!page) return;
      const source = display && page.displays.some((entry) => entry.id === display.id)
        ? display
        : page.displays[0];
      const sourceStates = source?.states.length ? source.states : [];
      const selectedSourceState = activeStateOf(source, stateByDisplay);
      const added = makeDisplay(
        device,
        readStateSpec(selectedSourceState)
          ?? readStateSpec(sourceStates[0])
          ?? seedSpec(`${page.id}:${localId("seed")}`),
      );
      added.states = sourceStates.length
        ? sourceStates.map((sourceState, index) => {
            const spec = readStateSpec(sourceState) ?? seedSpec(`${added.id}:${index}`);
            const cloned = makeState(sourceState.name || `State ${index + 1}`, rekeySpec(spec, added.id), index);
            cloned.argsJson = sourceState.argsJson;
            writeStateSpec(cloned, rekeySpec(spec, added.id));
            cloned.scenarioDataJson = sourceState.scenarioDataJson;
            return cloned;
          })
        : added.states;
      added.sortOrder = page.displays.length;
      apply(`Add ${DEVICE_NAME[device]} display`, undefined, (draft) => {
        const target = draft.pages.find((entry) => entry.id === page.id);
        if (target) target.displays.push(added);
      });
      setDisplayId(added.id);
      const selectedIndex = sourceStates.findIndex((entry) => entry.id === selectedSourceState?.id);
      const selectedClone = added.states[Math.max(0, selectedIndex)];
      if (selectedClone) setStateByDisplay((current) => ({ ...current, [added.id]: selectedClone.id }));
      setSelectedId(null);
    },
    [apply, display, page, setDisplayId, setSelectedId, setStateByDisplay, stateByDisplay],
  );

  const removeFrame = useCallback(
    (id: string) => {
      if (!page || page.displays.length <= 1) return;
      apply("Remove display", undefined, (draft) => {
        const target = draft.pages.find((entry) => entry.id === page.id);
        if (!target) return;
        target.displays = target.displays.filter((entry) => entry.id !== id);
        target.displays.forEach((entry, index) => {
          entry.sortOrder = index;
        });
      });
      setStateByDisplay((current) => withoutKey(current, id));
      if (displayId === id) {
        const next = page.displays.find((entry) => entry.id !== id);
        setDisplayId(next?.id ?? "");
        setSelectedId(null);
      }
    },
    [apply, displayId, page, setDisplayId, setSelectedId, setStateByDisplay],
  );

  const addState = useCallback(() => {
    if (!page || !display) return;
    const current = activeStateOf(display, stateByDisplay);
    const spec = readStateSpec(current) ?? readStateSpec(display.states[0]) ?? seedSpec(display.id);
    const added = makeState(`State ${display.states.length + 1}`, spec, display.states.length);
    apply("Add state", undefined, (draft) => {
      const target = draft.pages
        .find((entry) => entry.id === page.id)
        ?.displays.find((entry) => entry.id === display.id);
      target?.states.push(added);
    });
    setStateByDisplay((values) => ({ ...values, [display.id]: added.id }));
    setSelectedId(null);
  }, [apply, display, page, setSelectedId, setStateByDisplay, stateByDisplay]);

  const renameState = useCallback(
    (name: string) => {
      if (!page || !display || !state) return;
      apply("Rename state", `state-name:${state.id}`, (draft) => {
        const target = draft.pages
          .find((entry) => entry.id === page.id)
          ?.displays.find((entry) => entry.id === display.id)
          ?.states.find((entry) => entry.id === state.id);
        if (target) target.name = name;
      });
    },
    [apply, display, page, state],
  );

  const removeState = useCallback(() => {
    if (!page || !display || !state || display.states.length <= 1) return;
    const removed = state.id;
    apply("Remove state", undefined, (draft) => {
      const target = draft.pages
        .find((entry) => entry.id === page.id)
        ?.displays.find((entry) => entry.id === display.id);
      if (!target) return;
      target.states = target.states.filter((entry) => entry.id !== removed);
      target.states.forEach((entry, index) => {
        entry.sortOrder = index;
      });
    });
    setStateByDisplay((current) => withoutKey(current, display.id));
  }, [apply, display, page, setStateByDisplay, state]);

  return {
    selectPage,
    selectAddress,
    selectDisplay,
    selectFrame,
    selectState,
    addPage,
    duplicatePage,
    removePage,
    addFrame,
    removeFrame,
    addState,
    renameState,
    removeState,
  };
}

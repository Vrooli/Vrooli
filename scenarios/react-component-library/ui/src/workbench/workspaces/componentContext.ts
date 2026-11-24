import {
  createWorkspace,
  DisplayDevice,
  forkWorkspace,
  saveWorkspace,
  StateStatus,
  WorkspaceProvenance,
  WorkspaceDisplaySchema,
  WorkspacePageSchema,
  WorkspaceStateSchema,
  type Workspace,
} from "../../api/workspaces";
import { create } from "@bufbuild/protobuf";
import {
  pageComposition,
  displaySpec,
  parseOptions,
  parsePalette,
  protoDevice,
  editableSpec,
  writeStateSpec,
} from "./spec";
import {
  node,
  slotNode,
  type Template,
  type TemplateNode,
  type TemplateCategory,
} from "../templates/spec";
import { isTemplateCategory } from "../templates/spec";
import type { DesignDevice } from "../design-types";
import { findSlot } from "../templates/spec";

export const DEFAULT_COMPONENT_WORKSPACE_ID = "ws_curated_aperture";

/** Old template IDs were workspace seed slugs with the `ws_curated_` prefix. */
export function curatedWorkspaceIdForTemplate(templateRef: string | null | undefined): string {
  if (!templateRef) return DEFAULT_COMPONENT_WORKSPACE_ID;
  return templateRef.startsWith("ws_curated_") ? templateRef : `ws_curated_${templateRef}`;
}

/** Add one explicit component slot to a workspace composition when it has none. */
export function withComponentSlot(spec: TemplateNode): TemplateNode {
  if (findSlot(spec)) return spec;
  const slot = slotNode("workspace-component.slot", {
    hint: "Live component",
    minH: 112,
    role: "hero",
    accepts: ["action", "button", "input", "search", "metric", "stat", "number", "card", "text"],
  });
  if (spec.type === "page") return { ...spec, children: [...(spec.children ?? []), slot] };
  return node("workspace-component.page", "page", { pad: 24 }, [spec, slot]);
}

/** Attach a selected component/story identity to the first slot, adding a slot
 * when the curated workspace did not declare one. */
export function bindComponentToSlot(
  spec: TemplateNode,
  componentId: string,
  storyId: string,
  componentVersion?: string,
): TemplateNode {
  const withSlot = withComponentSlot(spec);
  const target = findSlot(withSlot);
  if (!target) return withSlot;
  const visit = (current: TemplateNode): TemplateNode => {
    if (current.id === target.id) {
      const existingProps = { ...current.props };
      delete existingProps.componentVersion;
      return {
        ...current,
        props: {
          ...existingProps,
          componentId,
          storyId,
          ...(componentVersion ? { componentVersion } : {}),
        },
      };
    }
    if (!current.children) return current;
    const children = current.children.map(visit);
    if (children.every((child, index) => child === current.children?.[index])) return current;
    return { ...current, children };
  };
  return visit(withSlot);
}

export function workspaceTemplate(workspace: Workspace, device: DesignDevice): Template {
  const page = workspace.pages[0];
  const targetDevice = protoDevice(device);
  const display =
    page?.displays.find((entry) => entry.device === targetDevice) ?? page?.displays[0];
  const state = display?.states[0];
  const categoryTag = workspace.tags.find(isTemplateCategory);
  const category: TemplateCategory = categoryTag ?? "operations";
  const spec =
    page && display ? displaySpec(page, display, state?.id) : page ? pageComposition(page) : null;
  return {
    id: workspace.id,
    name: workspace.name,
    tagline: workspace.summary,
    category,
    idea: workspace.summary,
    slotHint: "Live component in this workspace page",
    display: parseOptions(workspace.optionsJson).display ?? "sans",
    radius: parseOptions(workspace.optionsJson).radius ?? 16,
    palette: parsePalette(workspace.themeJson),
    spec: withComponentSlot(spec ?? node("workspace-component.page", "page", { pad: 24 }, [])),
  };
}

/** Forks a curated workspace and persists component/story identity in each
 * state composition, so reopening any display reads the same live slot. */
export async function createComponentWorkspace(input: {
  templateWorkspace: Workspace;
  componentId: string;
  storyId: string;
  componentVersion?: string;
}): Promise<Workspace> {
  if (input.templateWorkspace.provenance !== WorkspaceProvenance.CURATED) {
    throw new Error("Component context can only start from a curated workspace template.");
  }
  const workspace = await forkWorkspace(input.templateWorkspace.id);
  const updated: Workspace = {
    ...workspace,
    pages: workspace.pages.map((page) => ({
      ...page,
      displays: page.displays.map((display) => ({
        ...display,
        states: display.states.map((state) => {
          const nextState = { ...state };
          const spec = editableSpec(page, display, state.id);
          writeStateSpec(
            nextState,
            bindComponentToSlot(spec, input.componentId, input.storyId, input.componentVersion),
          );
          return nextState;
        }),
      })),
    })),
  };
  return saveWorkspace(updated, workspace.revision);
}

/** One-time import of an unavailable legacy custom template into the workspace
 * aggregate. No template row or browser shelf is created or kept in sync. */
export async function createComponentWorkspaceFromTemplate(input: {
  template: Template;
  componentId: string;
  storyId: string;
  componentVersion?: string;
}): Promise<Workspace> {
  const suffix = `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 10)}`;
  const pageId = `pg_${suffix}`;
  const displayId = `dp_${suffix}`;
  const stateId = `st_${suffix}`;
  const state = create(WorkspaceStateSchema, {
    id: stateId,
    name: "Default",
    argsJson: "{}",
    scenarioDataJson: "{}",
    status: StateStatus.DRAFT,
    captureId: "",
    sortOrder: 0,
  });
  writeStateSpec(
    state,
    bindComponentToSlot(
      input.template.spec,
      input.componentId,
      input.storyId,
      input.componentVersion,
    ),
  );
  return createWorkspace({
    name: `${input.template.name} · ${input.componentId}`,
    summary: `Imported once from the saved legacy template “${input.template.name}”.`,
    provenance: WorkspaceProvenance.IMPORTED,
    tags: ["component-context", input.template.category],
    pages: [
      create(WorkspacePageSchema, {
        id: pageId,
        name: input.template.name,
        route: "/",
        pageTypeId: "",
        sortOrder: 0,
        specJson: "{}",
        displays: [
          create(WorkspaceDisplaySchema, {
            id: displayId,
            device: DisplayDevice.DESKTOP,
            width: 1280,
            height: 800,
            sortOrder: 0,
            states: [state],
          }),
        ],
      }),
    ],
    themeJson: JSON.stringify(input.template.palette),
    optionsJson: JSON.stringify({ radius: input.template.radius, display: input.template.display }),
  });
}

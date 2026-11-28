import type {
  Workspace,
  WorkspaceDisplay,
  WorkspacePage,
  WorkspaceState,
} from "../../api/workspaces";
import { DisplayDevice } from "../../api/workspaces";
import type { DesignDevice, DesignDocument, DesignFrame, DesignOptions } from "../design-types";
import { DEVICE_NAME, FRAME_SIZE, rekeySpec } from "../studio/model";
import { DEFAULT_TEMPLATE } from "../templates/library";
import type { TemplateNode, TemplatePalette } from "../templates/spec";

/**
 * A state carries props in `args_json`. When a state also owns a distinct
 * composition, the composition lives under the `spec` key. Legacy designs were
 * imported exactly that way (`{"spec": <frame spec>}`), so one reader serves
 * both the migration and new authored states. Other keys are preserved so a
 * future prop-override editor can add them without a second store.
 */
const SPEC_KEY = "spec";

export function deviceOf(device: DisplayDevice): DesignDevice {
  switch (device) {
    case DisplayDevice.TABLET:
      return "tablet";
    case DisplayDevice.PHONE:
      return "phone";
    default:
      return "desktop";
  }
}

export function protoDevice(device: DesignDevice): DisplayDevice {
  switch (device) {
    case "tablet":
      return DisplayDevice.TABLET;
    case "phone":
      return DisplayDevice.PHONE;
    default:
      return DisplayDevice.DESKTOP;
  }
}

function isTemplateNode(value: unknown): value is TemplateNode {
  return Boolean(
    value &&
      typeof value === "object" &&
      "id" in value &&
      "type" in value &&
      typeof (value as { id?: unknown }).id === "string",
  );
}

function parseJSON(source: string | undefined): unknown {
  if (!source) return null;
  try {
    return JSON.parse(source);
  } catch {
    return null;
  }
}

export function readStateSpec(state: WorkspaceState | undefined): TemplateNode | null {
  const parsed = parseJSON(state?.argsJson);
  if (!parsed || typeof parsed !== "object") return null;
  const candidate = (parsed as Record<string, unknown>)[SPEC_KEY] ?? parsed;
  return isTemplateNode(candidate) ? candidate : null;
}

export function writeStateSpec(state: WorkspaceState, spec: TemplateNode): void {
  const parsed = parseJSON(state.argsJson);
  const base =
    parsed && typeof parsed === "object" && !Array.isArray(parsed)
      ? (parsed as Record<string, unknown>)
      : {};
  state.argsJson = JSON.stringify({ ...base, [SPEC_KEY]: spec });
}

/**
 * The composition shown for one display. A display's own state composition
 * wins; otherwise the page composition is the shared starting point. Only the
 * page fallback is shared, never mutated, so editing a display cannot silently
 * change a sibling.
 */
/** The composition a page renders when no display/state carries its own. */
export function pageComposition(page: WorkspacePage): TemplateNode {
  const parsed = parseJSON(page.specJson);
  if (isTemplateNode(parsed)) return parsed;
  if (parsed && typeof parsed === "object") {
    const nested = (parsed as { spec?: unknown }).spec;
    if (isTemplateNode(nested)) return nested;
  }
  return DEFAULT_TEMPLATE.spec;
}

export function displaySpec(
  page: WorkspacePage,
  display: WorkspaceDisplay,
  stateId?: string,
): TemplateNode {
  const state = display.states.find((entry) => entry.id === stateId) ?? display.states[0];
  return readStateSpec(state) ?? pageComposition(page);
}

/**
 * The composition an edit starts from. It mirrors `displaySpec` exactly, so the
 * first edit on a page whose stored spec is still empty writes the composition
 * the person was looking at instead of silently doing nothing.
 */
export function editableSpec(
  page: WorkspacePage,
  display: WorkspaceDisplay,
  stateId: string | undefined,
): TemplateNode {
  const state = display.states.find((entry) => entry.id === stateId) ?? display.states[0];
  return readStateSpec(state) ?? readStateSpec(display.states[0]) ?? pageComposition(page);
}

export function frameOf(
  page: WorkspacePage,
  display: WorkspaceDisplay,
  stateId?: string,
): DesignFrame {
  return {
    id: display.id,
    name: DEVICE_NAME[deviceOf(display.device)],
    device: deviceOf(display.device),
    spec: displaySpec(page, display, stateId),
  };
}

export function parsePalette(themeJson: string | undefined): TemplatePalette {
  const base = { ...DEFAULT_TEMPLATE.palette };
  const parsed = parseJSON(themeJson);
  if (!parsed || typeof parsed !== "object") return base;
  // A theme may be partial. Merging only string-valued known tokens keeps a
  // malformed key from reaching the renderer while preserving the rest.
  for (const key of Object.keys(base) as Array<keyof TemplatePalette>) {
    const value = (parsed as Record<string, unknown>)[key];
    if (typeof value === "string" && value.trim() !== "") base[key] = value;
  }
  return base;
}

export function parseOptions(optionsJson: string | undefined): DesignOptions {
  const parsed = parseJSON(optionsJson);
  if (parsed && typeof parsed === "object") {
    const options: DesignOptions = {};
    const radius = (parsed as { radius?: unknown }).radius;
    const display = (parsed as { display?: unknown }).display;
    if (typeof radius === "number") options.radius = radius;
    if (display === "serif" || display === "sans") options.display = display;
    return options;
  }
  return {};
}

/** One page of a workspace projected into the shape the studio renders. */
export function pageDocument(
  workspace: Workspace,
  page: WorkspacePage,
  stateByDisplay: Record<string, string> = {},
): DesignDocument {
  return {
    id: workspace.id,
    name: page.name || workspace.name,
    updatedAt: workspace.updatedAt,
    revision: workspace.revision,
    frames: page.displays.map((display) => frameOf(page, display, stateByDisplay[display.id])),
    theme: parsePalette(workspace.themeJson),
    options: parseOptions(workspace.optionsJson),
    scenario: workspace.bindingScenario || null,
    page: workspace.bindingPage || null,
  };
}

export function displaySizes(device: DesignDevice): { width: number; height: number } {
  return FRAME_SIZE[device];
}

export function seedSpec(prefix: string): TemplateNode {
  return rekeySpec(DEFAULT_TEMPLATE.spec, prefix);
}

export function activeStateOf(
  display: WorkspaceDisplay | undefined,
  stateByDisplay: Record<string, string>,
): WorkspaceState | undefined {
  if (!display) return undefined;
  return (
    display.states.find((state) => state.id === stateByDisplay[display.id]) ?? display.states[0]
  );
}

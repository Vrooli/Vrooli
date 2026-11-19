import type { CreateDesignInput, DesignDevice, DesignDocument, DesignFrame } from "../designs";
import { DEFAULT_TEMPLATE } from "../templates/library";
import type { Template, TemplateNode, TemplateNodeType, TemplatePalette } from "../templates/spec";

/** Layer types that can own children; the tree and asset drop both use it. */
export const CONTAINER_TYPES = new Set<TemplateNodeType>([
  "page",
  "stack",
  "row",
  "grid",
  "card",
  "surface",
]);

/** One shared definition of the frames the studio renders. */
export const FRAME_DEVICES: DesignDevice[] = ["desktop", "tablet", "phone"];

export const FRAME_SIZE: Record<DesignDevice, { width: number; height: number }> = {
  desktop: { width: 1180, height: 780 },
  tablet: { width: 834, height: 900 },
  phone: { width: 390, height: 844 },
};

/** Frame names are stored data, not interface copy. */
export const DEVICE_NAME: Record<DesignDevice, string> = {
  desktop: "Desktop",
  tablet: "Tablet",
  phone: "Phone",
};

export const PALETTE_KEYS: Array<keyof TemplatePalette> = [
  "bg",
  "surface",
  "surfaceAlt",
  "ink",
  "muted",
  "line",
  "accent",
  "accentInk",
  "accentSoft",
  "secondary",
];

export interface PaletteToken {
  key: keyof TemplatePalette;
  value: string;
}

export function paletteTokens(palette: TemplatePalette): PaletteToken[] {
  return PALETTE_KEYS.map((key) => ({ key, value: palette[key] }));
}

/**
 * Re-key a whole tree. Every frame owns its spec, so ids must stay unique
 * between frames. The frame prefix plus the original id keeps the semantic
 * tail that `layerLabel` falls back to ("hero", "nav", "chartCard") instead of
 * flattening every container to "Row · 3".
 */
export function rekeySpec(spec: TemplateNode, prefix: string): TemplateNode {
  const visit = (current: TemplateNode): TemplateNode => {
    const next: TemplateNode = {
      ...current,
      id: `${prefix}.${current.id}`,
      props: { ...current.props },
    };
    if (current.children) next.children = current.children.map(visit);
    return next;
  };
  return visit(spec);
}

export function seedFrames(template: Template): DesignFrame[] {
  return FRAME_DEVICES.map((device) => ({
    id: `${template.id}-${device}`,
    name: DEVICE_NAME[device],
    device,
    spec: rekeySpec(template.spec, `${template.id}.${device}`),
  }));
}

export interface DesignBinding {
  scenario?: string | null;
  page?: string | null;
}

export function seedInput(
  template: Template,
  name?: string,
  binding: DesignBinding = {},
): CreateDesignInput {
  return {
    name: name ?? template.name,
    frames: seedFrames(template),
    theme: { ...template.palette },
    // The template's chrome becomes the design's starting point, so editing
    // radius/display later is a design change rather than an inherited default.
    options: { radius: template.radius, display: template.display },
    scenario: binding.scenario ?? null,
    page: binding.page ?? null,
  };
}

/** The renderer still speaks Template; a design frame adapts into one. */
export function frameTemplate(doc: DesignDocument, frame: DesignFrame): Template {
  return {
    id: `${doc.id}:${frame.id}`,
    name: frame.name,
    tagline: doc.name,
    category: DEFAULT_TEMPLATE.category,
    slotHint: DEFAULT_TEMPLATE.slotHint,
    idea: DEFAULT_TEMPLATE.idea,
    palette: doc.theme,
    display: doc.options?.display ?? DEFAULT_TEMPLATE.display,
    radius: doc.options?.radius ?? DEFAULT_TEMPLATE.radius,
    spec: frame.spec,
  };
}

export function duplicateInput(doc: DesignDocument, name: string): CreateDesignInput {
  const stamp = Date.now().toString(36);
  const frames = doc.frames.map((frame) => {
    const id = `${frame.id}-copy-${stamp}`;
    return { ...frame, id, spec: rekeySpec(frame.spec, id) };
  });
  return {
    name,
    frames,
    theme: { ...doc.theme },
    options: { ...(doc.options ?? {}) },
    scenario: doc.scenario ?? null,
    page: doc.page ?? null,
  };
}

export function anonymousFrame(active: DesignFrame, device: DesignDevice, name: string): DesignFrame {
  const id = `frame-${device}-${Date.now().toString(36)}`;
  return {
    id,
    name,
    device,
    spec: rekeySpec(active.spec, id),
  };
}

/** Content identity for autosave: revision metadata is deliberately excluded. */
export function docSignature(doc: DesignDocument): string {
  return JSON.stringify({
    name: doc.name,
    scenario: doc.scenario ?? null,
    page: doc.page ?? null,
    theme: doc.theme,
    // Normalised so an absent options object and {} fingerprint the same; a
    // radius-only or display-only edit is still a real change.
    options: {
      radius: doc.options?.radius ?? null,
      display: doc.options?.display ?? null,
    },
    frames: doc.frames.map((frame) => ({
      id: frame.id,
      name: frame.name,
      device: frame.device,
      spec: frame.spec,
    })),
  });
}

export interface LayerRow {
  node: TemplateNode;
  depth: number;
  hasChildren: boolean;
  collapsed: boolean;
}

/** Layer rows with collapsed groups folded away. */
export function visibleLayerRows(
  root: TemplateNode,
  collapsed: ReadonlySet<string>,
): LayerRow[] {
  const rows: LayerRow[] = [];
  const visit = (current: TemplateNode, depth: number, hidden: boolean) => {
    if (hidden) return;
    const hasChildren = Boolean(current.children && current.children.length > 0);
    const isCollapsed = hasChildren && collapsed.has(current.id);
    rows.push({ node: current, depth, hasChildren, collapsed: isCollapsed });
    for (const child of current.children ?? []) visit(child, depth + 1, isCollapsed);
  };
  visit(root, 0, false);
  return rows;
}

export function firstNodeOfType(spec: TemplateNode, type: TemplateNodeType): string | null {
  if (spec.type === type) return spec.id;
  for (const child of spec.children ?? []) {
    const found = firstNodeOfType(child, type);
    if (found) return found;
  }
  return null;
}

export function frameForDevice(frames: DesignFrame[], device: DesignDevice): DesignFrame | undefined {
  return frames.find((frame) => frame.device === device);
}

export function themePatch(palette: TemplatePalette, key: keyof TemplatePalette, value: string): TemplatePalette {
  return { ...palette, [key]: value };
}

/** Catalog key per frame device, so no callsite builds a key from a variable. */
export const DEVICE_KEY: Record<DesignDevice, string> = {
  desktop: "workbench.studio.deviceName.desktop",
  tablet: "workbench.studio.deviceName.tablet",
  phone: "workbench.studio.deviceName.phone",
};

export interface AddableNode {
  /** Stable option value; never derived from a translated label. */
  id: string;
  type: TemplateNodeType;
  /** Catalog key for the layer name. `label` is the English default. */
  labelKey: string;
  label: string;
  props: Record<string, string | number | boolean>;
}

/** The layer palette offered in the tree footer. */
export const ADDABLE_NODES: AddableNode[] = [
  {
    id: "text",
    type: "text",
    labelKey: "workbench.studio.add.text",
    label: "Text",
    props: { text: "New line of copy", variant: "body", tone: "muted" },
  },
  {
    id: "heading",
    type: "text",
    labelKey: "workbench.studio.add.heading",
    label: "Heading",
    props: { text: "A stronger headline", variant: "display" },
  },
  { id: "card", type: "card", labelKey: "workbench.studio.add.card", label: "Card", props: { pad: 18, elevated: true } },
  { id: "stack", type: "stack", labelKey: "workbench.studio.add.stack", label: "Stack", props: { gap: 14 } },
  { id: "row", type: "row", labelKey: "workbench.studio.add.row", label: "Row", props: { gap: 10, wrap: true } },
  { id: "grid", type: "grid", labelKey: "workbench.studio.add.grid", label: "Grid", props: { gap: 14, cols: 2 } },
  {
    id: "action",
    type: "button",
    labelKey: "workbench.studio.add.action",
    label: "Action",
    props: { text: "Action", variant: "ghost", size: "sm" },
  },
  { id: "input", type: "input", labelKey: "workbench.studio.add.input", label: "Input", props: { placeholder: "Field" } },
  { id: "chip", type: "chip", labelKey: "workbench.studio.add.chip", label: "Chip", props: { text: "New" } },
  {
    id: "stat",
    type: "stat",
    labelKey: "workbench.studio.add.stat",
    label: "Stat",
    props: { label: "Measure", value: "0", detail: "sample" },
  },
  {
    id: "meter",
    type: "meter",
    labelKey: "workbench.studio.add.meter",
    label: "Meter",
    props: { label: "Progress", value: 42 },
  },
  {
    id: "divider",
    type: "divider",
    labelKey: "workbench.studio.add.divider",
    label: "Divider",
    props: {},
  },
  {
    id: "spacer",
    type: "spacer",
    labelKey: "workbench.studio.add.spacer",
    label: "Spacer",
    props: { h: 24 },
  },
  {
    id: "block",
    type: "swatch",
    labelKey: "workbench.studio.add.block",
    label: "Block",
    props: { h: 120, shape: "arch", tone: "accent" },
  },
];

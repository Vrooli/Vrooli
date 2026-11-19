import type { CSSProperties, ReactNode } from "react";

/**
 * A template is a small, declarative page composition built from real RCL
 * assets. It is intentionally data (not JSX) so the Design studio can render
 * the same composition as an artboard, edit it as a layer tree, and persist a
 * local draft without a second implementation.
 */
export type TemplateNodeType =
  | "page"
  | "stack"
  | "row"
  | "grid"
  | "surface"
  | "text"
  | "button"
  | "input"
  | "card"
  | "stat"
  | "avatar"
  | "chip"
  | "meter"
  | "divider"
  | "spacer"
  | "slot"
  | "swatch";

export interface TemplateNode {
  id: string;
  type: TemplateNodeType;
  props: Record<string, string | number | boolean>;
  children?: TemplateNode[];
}

export type TemplateCategory =
  | "analytics"
  | "commerce"
  | "ai"
  | "climate"
  | "portfolio"
  | "health"
  | "onboarding"
  | "operations";

export interface TemplatePalette {
  bg: string;
  surface: string;
  surfaceAlt: string;
  ink: string;
  muted: string;
  line: string;
  accent: string;
  accentInk: string;
  accentSoft: string;
  secondary: string;
}

export interface Template {
  id: string;
  name: string;
  tagline: string;
  category: TemplateCategory;
  /** Where the inspected component is placed when the template is used in context. */
  slotHint: string;
  /** A short line describing the page's design idea. */
  idea: string;
  palette: TemplatePalette;
  display: "serif" | "sans";
  radius: number;
  spec: TemplateNode;
  /** True when the template lives only in this browser (duplicated or created in the studio). */
  local?: boolean;
}

/**
 * A component slot placed as a layer on the page. `x`/`y` are layout pixels
 * relative to the template root, `w`/`h` size the live harness. The page stays
 * in its authored layout; the slot floats above it so moving the component
 * never reflows the composition.
 */
export interface SlotPlacement {
  x: number;
  y: number;
  w: number;
  h: number;
}

export const DEFAULT_SLOT_HEIGHT = 56;

export function node(
  id: string,
  type: TemplateNodeType,
  props: Record<string, string | number | boolean> = {},
  children?: TemplateNode[],
): TemplateNode {
  return children ? { id, type, props, children } : { id, type, props };
}

export function walk(nodeToWalk: TemplateNode, visit: (current: TemplateNode) => void): void {
  visit(nodeToWalk);
  for (const child of nodeToWalk.children ?? []) walk(child, visit);
}

export function findNode(root: TemplateNode, id: string): TemplateNode | undefined {
  if (root.id === id) return root;
  for (const child of root.children ?? []) {
    const found = findNode(child, id);
    if (found) return found;
  }
  return undefined;
}

export function updateNode(
  root: TemplateNode,
  id: string,
  update: (current: TemplateNode) => TemplateNode,
): TemplateNode {
  if (root.id === id) return update(root);
  if (!root.children) return root;
  return { ...root, children: root.children.map((child) => updateNode(child, id, update)) };
}

export function setProp(root: TemplateNode, id: string, key: string, value: string | number | boolean): TemplateNode {
  return updateNode(root, id, (current) => ({ ...current, props: { ...current.props, [key]: value } }));
}

export function removeNode(root: TemplateNode, id: string): TemplateNode {
  if (!root.children) return root;
  return {
    ...root,
    children: root.children
      .filter((child) => child.id !== id)
      .map((child) => removeNode(child, id)),
  };
}

export function findSlot(root: TemplateNode): TemplateNode | undefined {
  if (root.type === "slot") return root;
  for (const child of root.children ?? []) {
    const found = findSlot(child);
    if (found) return found;
  }
  return undefined;
}

export function slotPlacement(node: TemplateNode | undefined): SlotPlacement | undefined {
  if (!node || node.type !== "slot") return undefined;
  const { x, y, w, h } = node.props;
  if (typeof x !== "number" || typeof y !== "number" || typeof w !== "number") return undefined;
  return {
    x: Math.max(0, x),
    y: Math.max(0, y),
    w: Math.max(80, w),
    h: Math.max(24, typeof h === "number" ? h : DEFAULT_SLOT_HEIGHT),
  };
}

export function setSlotPlacement(
  root: TemplateNode,
  id: string,
  placement: SlotPlacement,
): TemplateNode {
  return updateNode(root, id, (current) => ({
    ...current,
    props: {
      ...current.props,
      x: Math.max(0, Math.round(placement.x)),
      y: Math.max(0, Math.round(placement.y)),
      w: Math.max(80, Math.round(placement.w)),
      h: Math.max(24, Math.round(placement.h)),
    },
  }));
}

export function clearSlotPlacement(root: TemplateNode, id: string): TemplateNode {
  return updateNode(root, id, (current) => {
    const props = { ...current.props };
    delete props.x;
    delete props.y;
    delete props.w;
    delete props.h;
    return { ...current, props };
  });
}

export function moveNode(root: TemplateNode, id: string, direction: -1 | 1): TemplateNode {
  const siblings = (nodes: TemplateNode[]): TemplateNode[] => {
    const index = nodes.findIndex((child) => child.id === id);
    if (index === -1) return nodes.map((child) => ({ ...child, children: child.children ? siblings(child.children) : undefined }));
    const next = [...nodes];
    const target = index + direction;
    if (target < 0 || target >= next.length) return next;
    const moved = next[index];
    const displaced = next[target];
    if (!moved || !displaced) return next;
    next[index] = displaced;
    next[target] = moved;
    return next;
  };
  return { ...root, children: root.children ? siblings(root.children) : undefined };
}

export function duplicateNode(root: TemplateNode, id: string): TemplateNode {
  const stamp = Date.now().toString(36);
  const clone = (current: TemplateNode, seed: string): TemplateNode => ({
    ...current,
    id: `${seed}-copy-${stamp}`,
    children: current.children?.map((child, index) => clone(child, `${seed}-${index}`)),
  });
  const parent = (current: TemplateNode): TemplateNode => {
    if (!current.children) return current;
    const index = current.children.findIndex((child) => child.id === id);
    if (index !== -1) {
      const original = current.children[index];
      if (!original) return current;
      const children = [...current.children];
      children.splice(index + 1, 0, clone(original, `${id}-copy`));
      return { ...current, children };
    }
    return { ...current, children: current.children.map(parent) };
  };
  return parent(root);
}

export function appendChild(root: TemplateNode, parentId: string, child: TemplateNode): TemplateNode {
  return updateNode(root, parentId, (current) => ({
    ...current,
    children: [...(current.children ?? []), child],
  }));
}

export function layerRows(root: TemplateNode): Array<{ node: TemplateNode; depth: number }> {
  const rows: Array<{ node: TemplateNode; depth: number }> = [];
  const visit = (current: TemplateNode, depth: number) => {
    rows.push({ node: current, depth });
    for (const child of current.children ?? []) visit(child, depth + 1);
  };
  visit(root, 0);
  return rows;
}

const LAYER_NAMES: Record<TemplateNodeType, string> = {
  page: "Page",
  stack: "Stack",
  row: "Row",
  grid: "Grid",
  surface: "Surface",
  text: "Text",
  button: "Button",
  input: "Input",
  card: "Card",
  stat: "Stat",
  avatar: "Avatar",
  chip: "Chip",
  meter: "Meter",
  divider: "Divider",
  spacer: "Spacer",
  slot: "Component slot",
  swatch: "Block",
};

export function layerLabel(current: TemplateNode): string {
  if (current.type === "page") return LAYER_NAMES.page;
  const explicit = current.props.label;
  if (typeof explicit === "string" && explicit.trim()) return explicit;
  if (typeof current.props.text === "string" && current.props.text.trim()) {
    return current.props.text.slice(0, 42);
  }
  if (typeof current.props.placeholder === "string" && current.props.placeholder.trim()) {
    return current.props.placeholder.slice(0, 42);
  }
  if (typeof current.props.name === "string" && current.props.name.trim()) return current.props.name;
  const suffix = current.id.split(".").pop()?.replace(/[-_]/g, " ");
  return suffix && suffix !== current.type
    ? `${LAYER_NAMES[current.type]} · ${suffix}`
    : LAYER_NAMES[current.type];
}

export function templateStyle(template: Template): CSSProperties {
  const p = template.palette;
  return {
    "--t-bg": p.bg,
    "--t-surface": p.surface,
    "--t-surface-alt": p.surfaceAlt,
    "--t-ink": p.ink,
    "--t-muted": p.muted,
    "--t-line": p.line,
    "--t-accent": p.accent,
    "--t-accent-ink": p.accentInk,
    "--t-accent-soft": p.accentSoft,
    "--t-secondary": p.secondary,
    "--t-radius": `${template.radius}px`,
    "--t-display": template.display === "serif" ? "var(--wb-serif)" : "var(--wb-sans)",
    "--color-background": p.bg,
    "--color-surface": p.surface,
    "--color-surface-muted": p.surfaceAlt,
    "--color-surface-raised": p.surface,
    "--color-foreground": p.ink,
    "--color-muted-foreground": p.muted,
    "--color-border": p.line,
    "--color-border-strong": p.muted,
    "--color-primary": p.accent,
    "--color-primary-foreground": p.accentInk,
    "--color-primary-hover": p.accent,
    "--color-focus": p.accent,
    "--color-field": p.surfaceAlt,
    "--color-danger": "#e05a5a",
    "--color-success": "#3f9c6b",
    "--color-warning": "#c98a2e",
    color: p.ink,
    background: p.bg,
    fontFamily: "var(--wb-sans)",
  } as CSSProperties;
}

export type SlotRender = (hint: string) => ReactNode;

import type {
  CSSProperties,
  KeyboardEvent as ReactKeyboardEvent,
  PointerEvent as ReactPointerEvent,
  ReactNode,
} from "react";
import { cx } from "../../cx";
import { isRclLayer } from "../../studio/rcl";
import type { LayerDropPosition } from "../../studio/layerMovement";
import type { Template, TemplateNode } from "../spec";

export interface NodeSlotContext {
  placed: boolean;
  editing: boolean;
  onDragStart: (event: ReactPointerEvent<HTMLElement>) => void;
  onKeyMove: (event: ReactKeyboardEvent<HTMLElement>) => void;
}

export const toneClass = (tone: unknown): string => (typeof tone === "string" ? `is-${tone}` : "");

export function paddingStyle(pad: unknown): CSSProperties | undefined {
  if (typeof pad === "number") return { padding: pad };
  if (typeof pad === "string") return { padding: pad };
  return undefined;
}

export function gapStyle(gap: unknown, fallback = 12): CSSProperties {
  return { gap: typeof gap === "number" ? gap : fallback };
}

export function importedStyle(serialized: unknown): CSSProperties | undefined {
  if (typeof serialized !== "string" || serialized.length > 12_000) return undefined;
  try {
    const values: unknown = JSON.parse(serialized);
    if (!values || typeof values !== "object" || Array.isArray(values)) return undefined;
    const style: Record<string, string | number> = {};
    for (const [rawName, rawValue] of Object.entries(values)) {
      if (
        !/^(--[\w-]{1,64}|[a-z][a-zA-Z0-9-]{0,63})$/.test(rawName) ||
        !(typeof rawValue === "string" || typeof rawValue === "number") ||
        (typeof rawValue === "string" &&
          (rawValue.length > 160 || /url\s*\(|expression\s*\(|javascript:|[<>;{}]/i.test(rawValue)))
      )
        continue;
      const name = rawName.startsWith("--")
        ? rawName
        : rawName.replace(/-([a-z])/g, (_, letter: string) => letter.toUpperCase());
      style[name] = rawValue;
    }
    return Object.keys(style).length ? style : undefined;
  } catch {
    return undefined;
  }
}

export function importedControlStyle(style: CSSProperties | undefined): CSSProperties | undefined {
  if (!style) return undefined;
  const visual = [
    "background",
    "backgroundColor",
    "backgroundImage",
    "border",
    "borderColor",
    "borderRadius",
    "borderStyle",
    "borderWidth",
    "boxShadow",
    "color",
    "fontFamily",
    "fontFeatureSettings",
    "fontSize",
    "fontStyle",
    "fontVariantNumeric",
    "fontWeight",
    "letterSpacing",
    "lineHeight",
    "opacity",
    "outline",
    "textAlign",
    "textDecoration",
    "textTransform",
    "whiteSpace",
    "alignItems",
    "display",
    "flexDirection",
    "flex",
    "flexBasis",
    "flexGrow",
    "flexShrink",
    "gap",
    "height",
    "justifyContent",
    "maxHeight",
    "maxWidth",
    "minHeight",
    "minWidth",
    "padding",
    "width",
  ];
  const picked: Record<string, string | number> = {};
  for (const property of visual) {
    const value = style[property as keyof CSSProperties];
    if (typeof value === "string" || typeof value === "number") picked[property] = value;
  }
  return Object.keys(picked).length ? picked : undefined;
}

export function controlContentStyle(style: CSSProperties | undefined): CSSProperties | undefined {
  if (!style) return undefined;
  const layout = ["alignItems", "flexDirection", "gap", "justifyContent", "textAlign"] as const;
  const picked: Record<string, string | number> = { minWidth: 0, maxWidth: "100%" };
  for (const property of layout) {
    const value = style[property];
    if (typeof value === "string" || typeof value === "number") picked[property] = value;
  }
  if (style.display === "flex" || style.display === "inline-flex") picked.display = "flex";
  return picked;
}

export function importedButtonUsesDenseTapTarget(style: CSSProperties | undefined): boolean {
  if (!style) return false;
  const height = style.height;
  const pixelValue =
    typeof height === "string" ? /^(\d+(?:\.\d+)?|\.\d+)px$/.exec(height.trim()) : null;
  if (typeof height !== "number" && !pixelValue) return false;
  const pixels = typeof height === "number" ? height : Number(pixelValue?.[1]);
  return Number.isFinite(pixels) && pixels < 44;
}

export function importedButtonStyle(style: CSSProperties | undefined): CSSProperties | undefined {
  const controlStyle = importedControlStyle(style);
  if (!importedButtonUsesDenseTapTarget(style)) return controlStyle;
  return { ...controlStyle, minHeight: style?.height };
}

export interface NodeContext {
  template: Template;
  slot?: ReactNode;
  renderComponentSlot?: (node: TemplateNode) => ReactNode;
  selectedId?: string;
  canvasDrag?: {
    sourceId: string;
    targetId: string | null;
    position: LayerDropPosition;
  } | null;
  coverageMode: boolean;
  interactive: boolean;
  mode: "canvas" | "preview" | "thumb";
  sourceView: boolean;
  slotContext?: NodeSlotContext;
  /** The one slot that receives the live component; others show a placeholder. */
  activeSlotId?: string;
  labels: {
    move: string;
    moveAria: string;
    placed: string;
    slot: string;
    unknownType: string;
    imageEmpty: string;
  };
}

export function nodeAttrs(node: TemplateNode, ctx: NodeContext) {
  const props = node.props;
  return {
    "data-node-id": node.id,
    "data-node-type": node.type,
    role: typeof props.role === "string" ? props.role : undefined,
    "aria-label": typeof props.ariaLabel === "string" ? props.ariaLabel : undefined,
    tabIndex: typeof props.tabIndex === "number" ? props.tabIndex : undefined,
    "data-rcl-coverage":
      ctx.coverageMode && node.type !== "page" ? (isRclLayer(node) ? "rcl" : "custom") : undefined,
    "data-selected": ctx.selectedId === node.id ? "true" : undefined,
    "data-canvas-drag-source": ctx.canvasDrag?.sourceId === node.id ? "true" : undefined,
    "data-canvas-drop": ctx.canvasDrag?.targetId === node.id ? ctx.canvasDrag.position : undefined,
    className: cx(ctx.selectedId === node.id && "is-selected"),
  };
}

export interface NodeRenderState {
  node: TemplateNode;
  context: NodeContext;
  children: ReactNode[] | undefined;
  attrs: ReturnType<typeof nodeAttrs>;
  sourceStyle: CSSProperties | undefined;
  textAlign: CSSProperties["textAlign"] | undefined;
  anchorRef?: { current: HTMLDivElement | null };
}

import type { CSSProperties, ReactNode } from "react";
import type { TemplateNode } from "./spec";
import type { NodeContext } from "./node-renderers/shared";
import { importedStyle, nodeAttrs } from "./node-renderers/shared";
import { renderLayoutNode } from "./node-renderers/layout";
import { renderContentNode } from "./node-renderers/content";
import { renderMediaNode } from "./node-renderers/media";

export type { NodeSlotContext } from "./node-renderers/shared";

function renderNode(
  node: TemplateNode,
  context: NodeContext,
  anchorRef?: { current: HTMLDivElement | null },
): ReactNode {
  const children = node.children?.map((child) => renderNode(child, context, anchorRef));
  const attrs = nodeAttrs(node, context);
  const sourceStyle = importedStyle(node.props.css);
  const textAlign =
    typeof node.props.align === "string"
      ? (node.props.align as CSSProperties["textAlign"])
      : undefined;
  const state = { node, context, children, attrs, sourceStyle, textAlign, anchorRef };
  switch (node.type) {
    case "page":
    case "stack":
    case "row":
    case "grid":
    case "surface":
      return renderLayoutNode(state);
    case "text":
    case "button":
    case "input":
    case "card":
    case "stat":
    case "metric":
    case "icon":
    case "avatar":
    case "chip":
    case "meter":
      return renderContentNode(state);
    default:
      return renderMediaNode(state);
  }
}

export function TemplateNodeRenderer({
  node,
  context,
  anchorRef,
}: {
  node: TemplateNode;
  context: NodeContext;
  anchorRef?: { current: HTMLDivElement | null };
}) {
  return <>{renderNode(node, context, anchorRef)}</>;
}

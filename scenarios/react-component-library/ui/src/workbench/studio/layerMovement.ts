import type { TemplateNode } from "../templates/spec";
import { CONTAINER_TYPES } from "./model";

export type LayerDropPosition = "inside" | "before" | "after";

/** Move a layer in the tree. Invalid moves return the original tree unchanged. */
export function reparentLayer(
  root: TemplateNode,
  sourceId: string,
  targetId: string,
  position: LayerDropPosition,
): TemplateNode {
  if (!sourceId || !targetId || sourceId === root.id || sourceId === targetId) return root;
  let source: TemplateNode | undefined;
  let target: TemplateNode | undefined;
  const find = (node: TemplateNode) => {
    if (node.id === sourceId) source = node;
    if (node.id === targetId) target = node;
    node.children?.forEach(find);
  };
  find(root);
  if (!source || !target) return root;
  if (position === "inside" && !CONTAINER_TYPES.has(target.type)) return root;
  const sourceContainsTarget = (node: TemplateNode): boolean =>
    node.id === targetId || Boolean(node.children?.some(sourceContainsTarget));
  if (sourceContainsTarget(source)) return root;

  let moved: TemplateNode | undefined;
  const remove = (node: TemplateNode): TemplateNode => {
    if (!node.children) return node;
    const children = node.children
      .filter((child) => {
        if (child.id !== sourceId) return true;
        moved = child;
        return false;
      })
      .map(remove);
    return children.length === node.children.length &&
      children.every((child, i) => child === node.children?.[i])
      ? node
      : { ...node, children };
  };
  const without = remove(root);
  if (!moved) return root;
  const moving = moved;
  const insert = (node: TemplateNode): TemplateNode => {
    if (position === "inside" && node.id === targetId) {
      return { ...node, children: [...(node.children ?? []), moving] };
    }
    if (position !== "inside" && node.children) {
      const index = node.children.findIndex((child) => child.id === targetId);
      if (index >= 0) {
        const children = [...node.children];
        children.splice(index + (position === "after" ? 1 : 0), 0, moving);
        return { ...node, children };
      }
    }
    if (!node.children) return node;
    const children = node.children.map(insert);
    return children.every((child, i) => child === node.children?.[i])
      ? node
      : { ...node, children };
  };
  return insert(without);
}

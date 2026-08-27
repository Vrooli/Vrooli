/**
 * @libraryId react-component-library:TreeView
 * @displayName TreeView
 * @description
 * @version 1.0.6
 * @tags ["data-display","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/**
 * @vrooliComponentSource react-component-library:TreeView
 * @deps {"react":"^18","lucide-react":"^0.424.0"}
 */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import {
  Children,
  isValidElement,
  type KeyboardEvent,
  type ReactNode,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { ChevronDown, ChevronRight, File, FolderOpen } from "lucide-react";

export interface TreeNode {
  id: string;
  label: ReactNode;
  /** Optional accessible name when the visible label contains badges or icons. */
  ariaLabel?: string;
  children?: TreeNode[];
  disabled?: boolean;
  icon?: ReactNode;
  defaultExpanded?: boolean;
}

export interface TreeViewProps {
  /** Structured nodes are preferred; `nodes` remains for simple callers. */
  items?: TreeNode[];
  nodes?: string[];
  label?: string;
  selectedId?: string;
  defaultSelectedId?: string;
  defaultExpandedIds?: string[];
  onSelect?: (node: TreeNode) => void;
}

interface VisibleNode {
  node: TreeNode;
  level: number;
  parentId?: string;
}

const EMPTY_DEFAULT_EXPANDED_IDS: string[] = [];

const treeStyles = `
[data-rcl-tree] { min-inline-size: 0; color: var(--color-foreground); }
[data-rcl-tree] .rcl-tree-item { display: flex; min-block-size: var(--tap-target-min); min-inline-size: 0; align-items: center; gap: var(--space-2xs); border: var(--border-hairline) solid transparent; border-radius: var(--radius-control); color: var(--color-foreground); padding: var(--space-3xs) var(--space-xs); cursor: pointer; outline: none; transition: background-color var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard); }
[data-rcl-tree] .rcl-tree-item:hover:not([aria-disabled="true"]) { background: var(--color-surface-muted); }
[data-rcl-tree] .rcl-tree-item[aria-selected="true"] { border-color: color-mix(in srgb, var(--color-primary) 42%, var(--color-border)); background: color-mix(in srgb, var(--color-primary) 10%, var(--color-surface)); color: var(--color-foreground); }
[data-rcl-tree] .rcl-tree-item[aria-disabled="true"] { cursor: not-allowed; opacity: var(--opacity-disabled); }
[data-rcl-tree] .rcl-tree-children { margin-inline-start: var(--space-sm); border-inline-start: var(--border-hairline) solid var(--color-border-subtle); padding-inline-start: var(--space-2xs); }
[data-rcl-tree] .rcl-tree-disclosure { display: inline-grid; min-block-size: var(--touch-target); min-inline-size: var(--touch-target); flex: 0 0 auto; place-items: center; border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); cursor: pointer; }
[data-rcl-tree] .rcl-tree-disclosure:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-tree] .rcl-tree-icon { display: inline-grid; min-inline-size: var(--space-sm); flex: 0 0 auto; place-items: center; color: var(--color-primary); }
[data-rcl-tree] .rcl-tree-label { display: flex; min-inline-size: 0; overflow: hidden; flex: 1; align-items: center; gap: var(--space-2xs); text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-tree] .rcl-tree-label-main { min-inline-size: 0; overflow: hidden; flex: 1; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-tree] .rcl-tree-label-meta { color: var(--color-muted-foreground); font: var(--text-caption); white-space: nowrap; }
[data-rcl-tree] .rcl-tree-empty { border: var(--border-hairline) dashed var(--color-border); border-radius: var(--radius-control); color: var(--color-muted-foreground); padding: var(--space-sm); font: var(--text-body-sm); }


`;

function defaultId(label: string, index: number) {
  return "tree-node-" + String(index) + "-" + label.toLowerCase().replace(/[^a-z0-9]+/g, "-");
}

function labelText(label: ReactNode): string {
  if (typeof label === "string" || typeof label === "number") return String(label);
  const text = Children.toArray(label)
    .map((child) => {
      if (typeof child === "string" || typeof child === "number") return String(child);
      if (isValidElement<{ children?: ReactNode }>(child)) {
        return labelText(child.props.children);
      }
      return "";
    })
    .filter(Boolean)
    .join(" ");
  return text || "item";
}

function collectVisible(
  nodes: TreeNode[],
  expanded: Set<string>,
  level = 1,
  parentId?: string,
): VisibleNode[] {
  return nodes.flatMap((node) => [
    { node, level, parentId },
    ...(node.children && expanded.has(node.id)
      ? collectVisible(node.children, expanded, level + 1, node.id)
      : []),
  ]);
}

function collectAllIds(nodes: TreeNode[], result = new Set<string>()) {
  for (const node of nodes) {
    if (node.defaultExpanded) result.add(node.id);
    if (node.children) collectAllIds(node.children, result);
  }
  return result;
}

export const TreeView = withClassName(function TreeView({
  items,
  nodes = [],
  label,
  selectedId: controlledSelectedId,
  defaultSelectedId,
  defaultExpandedIds: defaultExpandedIdsProp,
  onSelect,
}: TreeViewProps) {
  const libraryStrings = useStrings();
  label = label ?? libraryStrings("data-display.tree-view.tree", "Tree");
  const strings = useStrings();
  const defaultExpandedIds = defaultExpandedIdsProp ?? EMPTY_DEFAULT_EXPANDED_IDS;
  const resolvedNodes = useMemo<TreeNode[]>(
    () =>
      items ??
      nodes.map((node, index) => ({
        id: defaultId(node, index),
        label: node,
      })),
    [items, nodes],
  );
  const [expanded, setExpanded] = useState(
    () => new Set([...collectAllIds(resolvedNodes), ...defaultExpandedIds]),
  );
  const defaultExpansionKey = defaultExpandedIds.join("\u0000");
  useEffect(() => {
    const defaults = new Set([...collectAllIds(resolvedNodes), ...defaultExpandedIds]);
    if (!defaults.size) return;
    setExpanded((current) => {
      const next = new Set(current);
      defaults.forEach((id) => next.add(id));
      return next.size === current.size ? current : next;
    });
  }, [defaultExpandedIds, defaultExpansionKey, resolvedNodes]);
  const [internalSelectedId, setInternalSelectedId] = useState(defaultSelectedId);
  const selectedId = controlledSelectedId ?? internalSelectedId;
  const visibleNodes = useMemo(
    () => collectVisible(resolvedNodes, expanded),
    [expanded, resolvedNodes],
  );
  const refs = useRef(new Map<string, HTMLDivElement>());

  const focusNode = (id: string | undefined) => {
    if (!id) return;
    refs.current.get(id)?.focus();
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>, visible: VisibleNode) => {
    const index = visibleNodes.findIndex(({ node }) => node.id === visible.node.id);
    const next = visibleNodes[index + 1];
    const previous = visibleNodes[index - 1];
    const children = visible.node.children ?? [];
    const isExpanded = expanded.has(visible.node.id);
    switch (event.key) {
      case "ArrowDown":
        event.preventDefault();
        focusNode(next?.node.id);
        break;
      case "ArrowUp":
        event.preventDefault();
        focusNode(previous?.node.id);
        break;
      case "ArrowRight":
        event.preventDefault();
        if (children.length && !isExpanded) {
          setExpanded((current) => new Set(current).add(visible.node.id));
        } else {
          focusNode(children[0]?.id);
        }
        break;
      case "ArrowLeft":
        event.preventDefault();
        if (children.length && isExpanded) {
          setExpanded((current) => {
            const nextExpanded = new Set(current);
            nextExpanded.delete(visible.node.id);
            return nextExpanded;
          });
        } else {
          focusNode(visible.parentId);
        }
        break;
      case "Home":
        event.preventDefault();
        focusNode(visibleNodes[0]?.node.id);
        break;
      case "End":
        event.preventDefault();
        focusNode(visibleNodes.at(-1)?.node.id);
        break;
      case "Enter":
      case " ":
        event.preventDefault();
        if (!visible.node.disabled) {
          setInternalSelectedId(visible.node.id);
          onSelect?.(visible.node);
        }
        break;
      default:
        break;
    }
  };

  const renderNode = (visible: VisibleNode): ReactNode => {
    const { node, level } = visible;
    const hasChildren = Boolean(node.children?.length);
    const isExpanded = expanded.has(node.id);
    const accessibleLabel = node.ariaLabel || labelText(node.label);
    return (
      <div key={node.id}>
        <div
          ref={(element) => {
            if (element) refs.current.set(node.id, element);
            else refs.current.delete(node.id);
          }}
          role="treeitem"
          aria-level={level}
          aria-selected={selectedId === node.id}
          aria-expanded={hasChildren ? isExpanded : undefined}
          aria-disabled={node.disabled || undefined}
          tabIndex={
            selectedId === node.id || (!selectedId && visibleNodes[0]?.node.id === node.id) ? 0 : -1
          }
          className="rcl-tree-item"
          onClick={() => {
            if (!node.disabled) {
              setInternalSelectedId(node.id);
              onSelect?.(node);
            }
          }}
          onKeyDown={(event) => handleKeyDown(event, visible)}
        >
          {hasChildren ? (
            <button
              data-testid="data-display.tree-view"
              type="button"
              tabIndex={-1}
              aria-label={isExpanded ? "Collapse " + accessibleLabel : "Expand " + accessibleLabel}
              className="rcl-tree-disclosure"
              onClick={(event) => {
                event.stopPropagation();
                setExpanded((current) => {
                  const nextExpanded = new Set(current);
                  if (nextExpanded.has(node.id)) nextExpanded.delete(node.id);
                  else nextExpanded.add(node.id);
                  return nextExpanded;
                });
              }}
            >
              {isExpanded ? (
                <ChevronDown aria-hidden size={16} />
              ) : (
                <ChevronRight aria-hidden size={16} />
              )}
            </button>
          ) : (
            <span aria-hidden className="rcl-tree-disclosure" />
          )}
          <span aria-hidden className="rcl-tree-icon">
            {node.icon ?? (hasChildren ? <FolderOpen size={17} /> : <File size={17} />)}
          </span>
          <span className="rcl-tree-label">{node.label}</span>
        </div>
        {hasChildren && isExpanded ? (
          <div role="group" className="rcl-tree-children">
            {node.children?.map((child) =>
              renderNode({
                node: child,
                level: level + 1,
                parentId: node.id,
              }),
            )}
          </div>
        ) : null}
      </div>
    );
  };

  return (
    <div data-rcl-tree role="tree" aria-label={label}>
      <StyleSheet name="treeview-1-0-6-1" css={treeStyles} />
      {resolvedNodes.length ? (
        resolvedNodes.map((node) => renderNode({ node, level: 1 }))
      ) : (
        <div className="rcl-tree-empty" role="status">
          {strings("data-display.tree-view.nothing-to-display", "Nothing to display.")}
        </div>
      )}
    </div>
  );
});

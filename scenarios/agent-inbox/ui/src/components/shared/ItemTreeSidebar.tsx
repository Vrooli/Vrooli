/**
 * ItemTreeSidebar - Reusable tree sidebar for navigating templates/skills.
 *
 * Features:
 * - Builds tree from modes[] arrays dynamically
 * - Expandable/collapsible category nodes
 * - Dirty indicator (amber dot) on modified items
 * - Click to select items
 */

import { useMemo, useCallback, type ReactNode } from "react";
import { ChevronRight, ChevronDown, FolderOpen, PanelLeftClose, PanelLeftOpen } from "lucide-react";

// Tree node structure
export interface TreeNode {
  id: string;           // Unique node ID (path-based for categories, "item-{id}" for leaves)
  label: string;        // Display name
  isCategory: boolean;  // true for folders, false for items
  children: TreeNode[];
  itemId?: string;      // Only for leaf nodes - the actual item ID
  depth: number;
}

interface BaseItem {
  id: string;
  name: string;
  modes?: string[];
}

interface ItemTreeSidebarProps<T extends BaseItem> {
  items: T[];
  selectedItemId: string | null;
  onSelectItem: (id: string) => void;
  dirtyItemIds: Set<string>;
  expandedNodes: Set<string>;
  onToggleNode: (nodeId: string) => void;
  renderItemIcon?: (item: T) => ReactNode;
  title: string;
  className?: string;
  // Collapse/expand functionality
  isCollapsed?: boolean;
  onToggleCollapse?: () => void;
}

/**
 * Build a tree structure from items based on their modes[] arrays.
 */
function buildTree<T extends BaseItem>(items: T[]): TreeNode[] {
  const root: TreeNode[] = [];
  const nodeMap = new Map<string, TreeNode>();

  for (const item of items) {
    const modes = item.modes ?? [];

    if (modes.length === 0) {
      // Items without modes go to "Other" category
      let other = root.find((n) => n.id === "__other__");
      if (!other) {
        other = {
          id: "__other__",
          label: "Other",
          isCategory: true,
          children: [],
          depth: 0,
        };
        root.push(other);
      }
      other.children.push({
        id: `item-${item.id}`,
        label: item.name,
        isCategory: false,
        children: [],
        itemId: item.id,
        depth: 1,
      });
      continue;
    }

    // Build category path
    let currentPath = "";
    let currentChildren = root;

    for (let i = 0; i < modes.length; i++) {
      const mode = modes[i];
      if (!mode) continue;
      currentPath = currentPath ? `${currentPath}/${mode}` : mode;

      let node = nodeMap.get(currentPath);
      if (!node) {
        node = {
          id: currentPath,
          label: mode,
          isCategory: true,
          children: [],
          depth: i,
        };
        nodeMap.set(currentPath, node);
        currentChildren.push(node);
      }
      currentChildren = node.children;
    }

    // Add item as leaf
    currentChildren.push({
      id: `item-${item.id}`,
      label: item.name,
      isCategory: false,
      children: [],
      itemId: item.id,
      depth: modes.length,
    });
  }

  // Sort: categories first, then alphabetically
  const sortNodes = (nodes: TreeNode[]): TreeNode[] => {
    return nodes
      .map((n) => ({ ...n, children: sortNodes(n.children) }))
      .sort((a, b) => {
        if (a.isCategory !== b.isCategory) return a.isCategory ? -1 : 1;
        return a.label.localeCompare(b.label);
      });
  };

  return sortNodes(root);
}

interface TreeNodeComponentProps<T extends BaseItem> {
  node: TreeNode;
  items: T[];
  selectedItemId: string | null;
  onSelectItem: (id: string) => void;
  dirtyItemIds: Set<string>;
  expandedNodes: Set<string>;
  onToggleNode: (nodeId: string) => void;
  renderItemIcon?: (item: T) => ReactNode;
}

function TreeNodeComponent<T extends BaseItem>({
  node,
  items,
  selectedItemId,
  onSelectItem,
  dirtyItemIds,
  expandedNodes,
  onToggleNode,
  renderItemIcon,
}: TreeNodeComponentProps<T>) {
  const isExpanded = expandedNodes.has(node.id);
  const paddingLeft = `${node.depth * 12 + 8}px`;

  if (node.isCategory) {
    // Count dirty children for this category
    const countDirtyInSubtree = (n: TreeNode): number => {
      if (!n.isCategory && n.itemId) {
        return dirtyItemIds.has(n.itemId) ? 1 : 0;
      }
      return n.children.reduce((acc, child) => acc + countDirtyInSubtree(child), 0);
    };
    const dirtyCount = countDirtyInSubtree(node);

    return (
      <div>
        <button
          type="button"
          onClick={() => onToggleNode(node.id)}
          className="w-full flex items-center gap-2 py-1.5 px-2 text-slate-400 hover:text-slate-200 hover:bg-white/5 transition-colors text-xs"
          style={{ paddingLeft }}
        >
          {isExpanded ? (
            <ChevronDown className="h-3.5 w-3.5 flex-shrink-0" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5 flex-shrink-0" />
          )}
          <FolderOpen className="h-3.5 w-3.5 flex-shrink-0 text-slate-500" />
          <span className="truncate flex-1 text-left">{node.label}</span>
          {dirtyCount > 0 && (
            <span className="w-2 h-2 bg-amber-500 rounded-full flex-shrink-0" />
          )}
        </button>
        {isExpanded && (
          <div>
            {node.children.map((child) => (
              <TreeNodeComponent
                key={child.id}
                node={child}
                items={items}
                selectedItemId={selectedItemId}
                onSelectItem={onSelectItem}
                dirtyItemIds={dirtyItemIds}
                expandedNodes={expandedNodes}
                onToggleNode={onToggleNode}
                renderItemIcon={renderItemIcon}
              />
            ))}
          </div>
        )}
      </div>
    );
  }

  // Leaf node (item)
  const item = items.find((i) => i.id === node.itemId);
  const isSelected = selectedItemId === node.itemId;
  const isDirty = node.itemId ? dirtyItemIds.has(node.itemId) : false;

  return (
    <button
      type="button"
      onClick={() => node.itemId && onSelectItem(node.itemId)}
      className={`w-full flex items-center gap-2 py-1.5 px-2 text-left transition-colors text-xs relative ${
        isSelected
          ? "bg-indigo-600/30 text-white"
          : "text-slate-300 hover:bg-white/5 hover:text-white"
      }`}
      style={{ paddingLeft }}
    >
      {renderItemIcon && item ? (
        renderItemIcon(item)
      ) : (
        <div className="w-3.5 h-3.5 flex-shrink-0" /> // Spacer when no icon
      )}
      <span className="truncate flex-1">{node.label}</span>
      {isDirty && (
        <span className="w-2 h-2 bg-amber-500 rounded-full flex-shrink-0" title="Unsaved changes" />
      )}
    </button>
  );
}

export function ItemTreeSidebar<T extends BaseItem>({
  items,
  selectedItemId,
  onSelectItem,
  dirtyItemIds,
  expandedNodes,
  onToggleNode,
  renderItemIcon,
  title,
  className = "",
  isCollapsed = false,
  onToggleCollapse,
}: ItemTreeSidebarProps<T>) {
  const treeData = useMemo(() => buildTree(items), [items]);

  // Auto-expand nodes that contain the selected item
  const _expandToItem = useCallback(
    (itemId: string) => {
      const item = items.find((i) => i.id === itemId);
      if (!item?.modes) return;

      // Build paths that should be expanded
      let path = "";
      for (const mode of item.modes) {
        path = path ? `${path}/${mode}` : mode;
        if (!expandedNodes.has(path)) {
          onToggleNode(path);
        }
      }
    },
    [items, expandedNodes, onToggleNode]
  );

  // Count total dirty items
  const dirtyCount = dirtyItemIds.size;

  // Collapsed state - show narrow strip with expand button
  if (isCollapsed) {
    return (
      <div className={`flex flex-col h-full border-r border-white/10 w-10 flex-shrink-0 ${className}`}>
        <div className="flex flex-col items-center py-2 gap-2">
          {onToggleCollapse && (
            <button
              type="button"
              onClick={onToggleCollapse}
              className="p-1.5 rounded hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
              title={`Expand ${title}`}
            >
              <PanelLeftOpen className="h-4 w-4" />
            </button>
          )}
          {dirtyCount > 0 && (
            <span
              className="w-5 h-5 flex items-center justify-center text-[10px] font-medium bg-amber-500/20 text-amber-400 rounded-full"
              title={`${dirtyCount} unsaved`}
            >
              {dirtyCount}
            </span>
          )}
        </div>
      </div>
    );
  }

  // Expanded state - full sidebar
  return (
    <div className={`flex flex-col h-full border-r border-white/10 ${className}`}>
      {/* Header */}
      <div className="flex-shrink-0 px-3 py-2 border-b border-white/10">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium text-slate-300">{title}</h3>
          <div className="flex items-center gap-2">
            {dirtyCount > 0 && (
              <span className="text-xs text-amber-400">
                {dirtyCount} unsaved
              </span>
            )}
            {onToggleCollapse && (
              <button
                type="button"
                onClick={onToggleCollapse}
                className="p-1 rounded hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
                title="Collapse sidebar"
              >
                <PanelLeftClose className="h-4 w-4" />
              </button>
            )}
          </div>
        </div>
        <p className="text-xs text-slate-500 mt-0.5">
          {items.length} {items.length === 1 ? "item" : "items"}
        </p>
      </div>

      {/* Tree */}
      <div className="flex-1 overflow-y-auto py-1">
        {treeData.length === 0 ? (
          <p className="text-xs text-slate-500 text-center py-4">No items</p>
        ) : (
          treeData.map((node) => (
            <TreeNodeComponent
              key={node.id}
              node={node}
              items={items}
              selectedItemId={selectedItemId}
              onSelectItem={onSelectItem}
              dirtyItemIds={dirtyItemIds}
              expandedNodes={expandedNodes}
              onToggleNode={onToggleNode}
              renderItemIcon={renderItemIcon}
            />
          ))
        )}
      </div>
    </div>
  );
}

// Re-export for use by consumers
export { buildTree };

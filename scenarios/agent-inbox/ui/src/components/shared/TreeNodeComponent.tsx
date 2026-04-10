/**
 * TreeNodeComponent - Renders a single node in the item tree sidebar.
 * Handles both category (folder) and leaf (item) nodes.
 */

import type { ReactNode } from "react";
import { ChevronRight, ChevronDown, FolderOpen } from "lucide-react";
import type { TreeNode } from "./itemTreeTypes";

interface BaseItem {
  id: string;
  name: string;
  modes?: string[];
}

export interface TreeNodeComponentProps<T extends BaseItem> {
  node: TreeNode;
  items: T[];
  selectedItemId: string | null;
  onSelectItem: (id: string) => void;
  dirtyItemIds: Set<string>;
  expandedNodes: Set<string>;
  onToggleNode: (nodeId: string) => void;
  renderItemIcon?: (item: T) => ReactNode;
}

export function TreeNodeComponent<T extends BaseItem>({
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

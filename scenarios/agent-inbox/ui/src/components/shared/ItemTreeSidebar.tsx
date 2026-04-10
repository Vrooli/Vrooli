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
import { PanelLeftClose, PanelLeftOpen } from "lucide-react";
import { buildTree } from "./itemTreeTypes";
import { TreeNodeComponent } from "./TreeNodeComponent";

// Re-export types for consumers
export type { TreeNode } from "./itemTreeTypes";
export { buildTree } from "./itemTreeTypes";

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

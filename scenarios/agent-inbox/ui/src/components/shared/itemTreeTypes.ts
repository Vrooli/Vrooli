/**
 * Types and tree-building logic for ItemTreeSidebar.
 */

// Tree node structure
export interface TreeNode {
  id: string;           // Unique node ID (path-based for categories, "item-{id}" for leaves)
  label: string;        // Display name
  isCategory: boolean;  // true for folders, false for items
  children: TreeNode[];
  itemId?: string;      // Only for leaf nodes - the actual item ID
  depth: number;
}

export interface BaseItem {
  id: string;
  name: string;
  modes?: string[];
}

/**
 * Build a tree structure from items based on their modes[] arrays.
 */
export function buildTree(items: BaseItem[]): TreeNode[] {
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

import type { Edge, Node } from 'reactflow';

/**
 * Normalizes raw node data from API/storage into valid ReactFlow Node objects.
 * Ensures all required fields exist with proper defaults.
 */
export const normalizeNodes = (nodes: unknown[] | undefined | null): Node[] => {
  if (!Array.isArray(nodes)) return [];
  return nodes.map((node, index) => {
    const nodeData = node as Record<string, unknown>;
    const id = nodeData?.id ? String(nodeData.id) : `node-${index + 1}`;
    const type = nodeData?.type ? String(nodeData.type) : 'navigate';
    const positionData = nodeData?.position as Record<string, unknown> | undefined;
    const position = {
      x: Number(positionData?.x ?? 100 + index * 200) || 0,
      y: Number(positionData?.y ?? 100 + index * 120) || 0,
    };
    const data = nodeData?.data && typeof nodeData.data === 'object' ? nodeData.data : {};
    // Preserve V2 action field if present (from API responses)
    const action = nodeData?.action && typeof nodeData.action === 'object' ? nodeData.action : undefined;
    const result: Record<string, unknown> = {
      ...nodeData,
      id,
      type,
      position,
      data,
    };
    // Only include action if it exists
    if (action) {
      result.action = action;
    }
    return result as Node;
  });
};

/**
 * Normalizes raw edge data from API/storage into valid ReactFlow Edge objects.
 * Filters out invalid edges that are missing source or target.
 */
export const normalizeEdges = (edges: unknown[] | undefined | null): Edge[] => {
  if (!Array.isArray(edges)) return [];
  return edges
    .map((edge, index) => {
      const edgeData = edge as Record<string, unknown>;
      const id = edgeData?.id ? String(edgeData.id) : `edge-${index + 1}`;
      const source = edgeData?.source ? String(edgeData.source) : '';
      const target = edgeData?.target ? String(edgeData.target) : '';
      if (!source || !target) return null;
      const normalized: Edge = {
        ...edgeData,
        id,
        source,
        target,
      } as Edge;
      const data = (edgeData?.data && typeof edgeData.data === 'object') ? edgeData.data as Record<string, unknown> : undefined;
      if (data) {
        normalized.data = data;
      }
      const condition = typeof data?.condition === 'string' ? data.condition : undefined;
      if (condition === 'if_true' || condition === 'if_false') {
        const stroke = condition === 'if_true' ? '#4ade80' : '#f87171';
        normalized.label = condition === 'if_true' ? 'IF TRUE' : 'IF FALSE';
        normalized.style = { ...(normalized.style ?? {}), stroke };
      }
      if (condition === 'loop_body') {
        normalized.label = 'LOOP BODY';
        normalized.style = { ...(normalized.style ?? {}), stroke: '#38bdf8' };
      }
      if (condition === 'loop_next') {
        normalized.label = 'AFTER LOOP';
        normalized.style = { ...(normalized.style ?? {}), stroke: '#7c3aed' };
      }
      if (condition === 'loop_continue') {
        normalized.label = 'CONTINUE';
        normalized.style = { ...(normalized.style ?? {}), stroke: '#22c55e' };
      }
      if (condition === 'loop_break') {
        normalized.label = 'BREAK';
        normalized.style = { ...(normalized.style ?? {}), stroke: '#f43f5e' };
      }
      return normalized;
    })
    .filter(Boolean) as Edge[];
};

/**
 * Automatically layouts nodes if they don't have valid positions.
 * Uses a simple BFS-based layering algorithm.
 */
export const autoLayoutNodes = (nodes: Node[], edges: Edge[]): Node[] => {
  if (nodes.length === 0) return [];

  // Check if we need to layout: if all nodes are at (0,0) or very close, we assume they need layout.
  // Or if the user specifically requested "optional positions", we can check if positions are missing in the raw data.
  // Since normalizeNodes supplies default (0,0) or index-based positions, we might need a better heuristic.
  // For now, let's assume if the first few nodes are all at x=0, we should layout.
  // Actually, normalizeNodes gives `100 + index * 200` for X.
  // Let's check if the "raw" positions were missing. But we don't have raw data here.
  // We can rely on a heuristic: if all nodes have y=0 (which normalizeNodes does NOT do by default, it does `100 + index * 120`),
  // checking for the default pattern from normalizeNodes might be tricky if we want to support "some positions present".
  //
  // However, the requirement is: "remove them from all workflows... and add logic which properly spaces them automatically when position data isn't provided."
  // If we strip positions, normalizeNodes will assign the diagonal layout:
  // x: 100 + index * 200
  // y: 100 + index * 120
  // We can detect this specific pattern or just always run auto-layout if we detect "default-like" positions.
  //
  // Better approach: Let's just run the layout algorithm. It's deterministic.
  // If the nodes already have good positions, we might not want to overwrite them.
  // But how do we know?
  //
  // Let's look at `normalizeNodes` again.
  // It assigns defaults if `positionData` is missing.
  //
  // To be safe and explicit, maybe we should export a function that takes the RAW data?
  // But `WorkflowBuilder` calls `normalizeNodes` then `normalizeEdges`.
  //
  // Let's implement a layout that respects existing non-default positions?
  // Or, simpler: If we detect that the nodes are in the "default diagonal" (which is what normalizeNodes does when pos is missing), we re-layout.
  //
  // Default diagonal: x = 100 + i*200, y = 100 + i*120.
  // Let's check if the nodes follow this pattern.

  const isDefaultLayout = nodes.every((node, i) => {
    return node.position.x === 100 + i * 200 && node.position.y === 100 + i * 120;
  });

  // Also check if all are at 0,0 (legacy default)
  const isZeroLayout = nodes.every(n => n.position.x === 0 && n.position.y === 0);

  if (!isDefaultLayout && !isZeroLayout) {
    return nodes;
  }

  // Build adjacency list
  const adj = new Map<string, string[]>();
  const inDegree = new Map<string, number>();

  nodes.forEach(node => {
    adj.set(node.id, []);
    inDegree.set(node.id, 0);
  });

  edges.forEach(edge => {
    if (adj.has(edge.source) && adj.has(edge.target)) {
      adj.get(edge.source)?.push(edge.target);
      inDegree.set(edge.target, (inDegree.get(edge.target) || 0) + 1);
    }
  });

  // BFS for levels
  const levels = new Map<string, number>();
  const queue: string[] = [];

  // Find roots (in-degree 0)
  nodes.forEach(node => {
    if ((inDegree.get(node.id) || 0) === 0) {
      levels.set(node.id, 0);
      queue.push(node.id);
    }
  });

  // If no roots (cycle?), pick the first one
  if (queue.length === 0 && nodes.length > 0) {
    const first = nodes[0].id;
    levels.set(first, 0);
    queue.push(first);
  }

  const visited = new Set<string>(queue);

  while (queue.length > 0) {
    const currId = queue.shift()!;
    const currLevel = levels.get(currId)!;

    const neighbors = adj.get(currId) || [];
    for (const nextId of neighbors) {
      if (!visited.has(nextId)) {
        visited.add(nextId);
        levels.set(nextId, currLevel + 1);
        queue.push(nextId);
      } else {
        // If already visited, we might want to push it deeper if this path is longer?
        // For simple tree-like, max level is better.
        if (levels.get(nextId)! < currLevel + 1) {
          levels.set(nextId, currLevel + 1);
          // If we update level, we might need to re-process children? 
          // For a simple DAG, topological sort is better, but this BFS is "okay" for simple flows.
          // Let's stick to simple BFS for now to avoid infinite loops in cycles.
        }
      }
    }
  }

  // Group by level
  const levelGroups = new Map<number, Node[]>();
  nodes.forEach(node => {
    const lvl = levels.get(node.id) ?? 0;
    if (!levelGroups.has(lvl)) {
      levelGroups.set(lvl, []);
    }
    levelGroups.get(lvl)?.push(node);
  });

  // Assign positions
  const HORIZONTAL_SPACING = 300;
  const VERTICAL_SPACING = 150;

  return nodes.map(node => {
    const lvl = levels.get(node.id) ?? 0;
    const nodesInLevel = levelGroups.get(lvl)!;
    const indexInLevel = nodesInLevel.indexOf(node);

    return {
      ...node,
      position: {
        x: lvl * HORIZONTAL_SPACING,
        y: indexInLevel * VERTICAL_SPACING + 100, // +100 padding
      },
    };
  });
};

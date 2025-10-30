import { Node, Edge } from 'reactflow';

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
    return {
      ...nodeData,
      id,
      type,
      position,
      data,
    } as Node;
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
      return {
        ...edgeData,
        id,
        source,
        target,
      } as Edge;
    })
    .filter(Boolean) as Edge[];
};

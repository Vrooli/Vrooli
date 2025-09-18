import { Node, Edge } from 'reactflow';

/**
 * Finds the most recent Navigate node upstream from a given node
 * by tracing back through the workflow connections
 */
export function findUpstreamNavigateNode(
  nodeId: string,
  nodes: Node[],
  edges: Edge[]
): Node | null {
  const visited = new Set<string>();
  const queue: string[] = [nodeId];

  while (queue.length > 0) {
    const currentNodeId = queue.shift()!;
    
    if (visited.has(currentNodeId)) {
      continue;
    }
    visited.add(currentNodeId);

    // Find the current node
    const currentNode = nodes.find(n => n.id === currentNodeId);
    if (!currentNode) {
      continue;
    }

    // If this is a Navigate node and it's not the starting node, return it
    if (currentNode.type === 'navigate' && currentNodeId !== nodeId) {
      return currentNode;
    }

    // Find all edges that target this node (incoming edges)
    const incomingEdges = edges.filter(edge => edge.target === currentNodeId);
    
    // Add all source nodes to the queue for traversal
    for (const edge of incomingEdges) {
      if (!visited.has(edge.source)) {
        queue.push(edge.source);
      }
    }
  }

  return null;
}

/**
 * Gets the URL from a Navigate node's data
 */
export function getNavigateNodeUrl(node: Node): string | null {
  if (node.type !== 'navigate' || !node.data?.url) {
    return null;
  }
  return node.data.url;
}

/**
 * Gets the upstream URL for a node by finding its Navigate node
 */
export function getUpstreamUrl(
  nodeId: string,
  nodes: Node[],
  edges: Edge[]
): string | null {
  const navigateNode = findUpstreamNavigateNode(nodeId, nodes, edges);
  if (!navigateNode) {
    return null;
  }
  return getNavigateNodeUrl(navigateNode);
}
import type { Node, Edge } from 'reactflow';
import { getConfig } from '../config';

// Cache for resolved scenario URLs to avoid redundant API calls
const scenarioUrlCache = new Map<string, { url: string; timestamp: number }>();
const CACHE_DURATION_MS = 30000; // 30 seconds

export interface NodeScreenshot {
  dataUrl: string;
  nodeId: string;
  nodeType: string;
  capturedAt?: string;
  sourceUrl?: string;
}

const pickString = (obj: Record<string, unknown> | null | undefined, key: string): string | null => {
  if (!obj || typeof obj !== 'object') {
    return null;
  }
  const value = obj[key];
  return typeof value === 'string' && value.trim().length > 0 ? value : null;
};

const extractScreenshotFromNode = (node: Node): NodeScreenshot | null => {
  const data = (node?.data ?? {}) as Record<string, unknown>;
  if (!data) {
    return null;
  }

  const candidateKeys = [
    'previewScreenshot',
    'screenshot',
    'screenshotDataUrl',
    'screenshotUrl',
    'previewImage',
  ];

  let screenshot: string | null = null;
  for (const key of candidateKeys) {
    screenshot = pickString(data, key);
    if (screenshot) break;
  }

  if (!screenshot) {
    const preview = data.preview as Record<string, unknown> | undefined;
    screenshot = pickString(preview, 'screenshot') ?? pickString(preview, 'image') ?? pickString(preview, 'dataUrl');
  }

  if (!screenshot) {
    const elementInfo = data.elementInfo as Record<string, unknown> | undefined;
    const elementScreenshot = elementInfo?.screenshot as Record<string, unknown> | string | undefined;
    if (typeof elementScreenshot === 'string') {
      screenshot = elementScreenshot;
    } else if (elementScreenshot && typeof elementScreenshot === 'object') {
      screenshot = pickString(elementScreenshot, 'dataUrl') ?? pickString(elementScreenshot, 'url');
    }
  }

  if (!screenshot || screenshot.trim().length === 0) {
    return null;
  }

  const capturedAt = pickString(data, 'previewScreenshotCapturedAt') ?? pickString(data, 'screenshotCapturedAt');
  const sourceUrl = pickString(data, 'previewScreenshotSourceUrl') ?? pickString(data, 'screenshotSourceUrl');

  return {
    dataUrl: screenshot,
    nodeId: node.id,
    nodeType: typeof node.type === 'string' ? node.type : 'unknown',
    capturedAt: capturedAt ?? undefined,
    sourceUrl: sourceUrl ?? undefined,
  };
};

/**
 * Resolves a scenario name (and optional path) to its actual URL
 */
async function resolveScenarioUrl(scenarioName: string, scenarioPath?: string): Promise<string | null> {
  const cacheKey = `${scenarioName}:${scenarioPath || ''}`;
  const cached = scenarioUrlCache.get(cacheKey);

  if (cached && Date.now() - cached.timestamp < CACHE_DURATION_MS) {
    return cached.url;
  }

  try {
    const config = await getConfig();
    const response = await fetch(`${config.API_URL}/scenarios/${encodeURIComponent(scenarioName)}/port`);

    if (!response.ok) {
      return null;
    }

    const info = await response.json();
    const baseUrl: string | undefined = typeof info?.url === 'string' && info.url.trim() !== ''
      ? info.url
      : info?.port
        ? `http://localhost:${info.port}`
        : undefined;

    if (!baseUrl) {
      return null;
    }

    const trimmedPath = scenarioPath?.trim() || '';
    const resolvedUrl = trimmedPath
      ? `${baseUrl.replace(/\/+$/, '')}/${trimmedPath.replace(/^\/+/, '')}`
      : baseUrl;

    // Cache the result
    scenarioUrlCache.set(cacheKey, { url: resolvedUrl, timestamp: Date.now() });

    return resolvedUrl;
  } catch (error) {
    return null;
  }
}

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

export function getUpstreamScreenshot(
  nodeId: string,
  nodes: Node[],
  edges: Edge[]
): NodeScreenshot | null {
  if (!nodeId) {
    return null;
  }

  const nodeMap = new Map(nodes.map((node) => [node.id, node]));
  const visited = new Set<string>();
  const queue: string[] = [nodeId];

  while (queue.length > 0) {
    const currentId = queue.shift()!;
    if (visited.has(currentId)) {
      continue;
    }
    visited.add(currentId);

    const currentNode = nodeMap.get(currentId);
    if (!currentNode) {
      continue;
    }

    if (currentId !== nodeId) {
      const screenshot = extractScreenshotFromNode(currentNode);
      if (screenshot) {
        return screenshot;
      }
    }

    const incomingEdges = edges.filter((edge) => edge.target === currentId);
    for (const edge of incomingEdges) {
      if (!visited.has(edge.source)) {
        queue.push(edge.source);
      }
    }
  }

  return null;
}

/**
 * Gets the URL from a Navigate node's data (synchronous version)
 * For scenario navigation, returns a placeholder URL
 * Use getNavigateNodeUrlAsync for actual resolution
 */
export function getNavigateNodeUrl(node: Node): string | null {
  if (node.type !== 'navigate') {
    return null;
  }

  const destinationType = node.data?.destinationType || '';
  const scenarioName = node.data?.scenario || node.data?.scenarioName || '';

  // If it's a scenario navigation, return a placeholder
  if (destinationType === 'scenario' || (scenarioName && !node.data?.url)) {
    if (!scenarioName) {
      return null;
    }

    const scenarioPath = node.data?.scenarioPath || '';
    return scenarioPath
      ? `scenario://${scenarioName}${scenarioPath}`
      : `scenario://${scenarioName}`;
  }

  // Regular URL navigation
  if (!node.data?.url) {
    return null;
  }
  return node.data.url;
}

/**
 * Gets the URL from a Navigate node's data (async version)
 * Resolves scenario URLs to actual HTTP URLs
 */
export async function getNavigateNodeUrlAsync(node: Node): Promise<string | null> {
  if (node.type !== 'navigate') {
    return null;
  }

  const destinationType = node.data?.destinationType || '';
  const scenarioName = node.data?.scenario || node.data?.scenarioName || '';

  // If it's a scenario navigation, resolve the URL
  if (destinationType === 'scenario' || (scenarioName && !node.data?.url)) {
    if (!scenarioName) {
      return null;
    }

    const scenarioPath = node.data?.scenarioPath || '';
    return await resolveScenarioUrl(scenarioName, scenarioPath);
  }

  // Regular URL navigation
  if (!node.data?.url) {
    return null;
  }
  return node.data.url;
}

/**
 * Gets the upstream URL for a node by finding its Navigate node (synchronous)
 * Returns placeholder URLs for scenario navigation
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

/**
 * Gets the upstream URL for a node by finding its Navigate node (async)
 * Resolves scenario URLs to actual HTTP URLs
 */
export async function getUpstreamUrlAsync(
  nodeId: string,
  nodes: Node[],
  edges: Edge[]
): Promise<string | null> {
  const navigateNode = findUpstreamNavigateNode(nodeId, nodes, edges);
  if (!navigateNode) {
    return null;
  }
  return await getNavigateNodeUrlAsync(navigateNode);
}

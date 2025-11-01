import { useEffect, useState } from 'react';
import { useWorkflowStore } from '../stores/workflowStore';
import { getUpstreamScreenshot, type NodeScreenshot } from '../utils/nodeUtils';

export function useUpstreamScreenshot(nodeId: string): NodeScreenshot | null {
  const nodes = useWorkflowStore((state) => state.nodes);
  const edges = useWorkflowStore((state) => state.edges);
  const [screenshot, setScreenshot] = useState<NodeScreenshot | null>(null);

  useEffect(() => {
    if (!nodeId || !nodes || !edges) {
      setScreenshot(null);
      return;
    }

    const resolved = getUpstreamScreenshot(nodeId, nodes, edges);
    setScreenshot(resolved);
  }, [edges, nodeId, nodes]);

  return screenshot;
}

export default useUpstreamScreenshot;

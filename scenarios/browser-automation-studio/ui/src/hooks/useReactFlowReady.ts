import { useEffect, useState } from "react";
import type { ReactFlowInstance } from "reactflow";
import { logger } from "@utils/logger";

interface UseReactFlowReadyOptions {
  /** The React Flow instance from useReactFlow() */
  reactFlowInstance: ReactFlowInstance | null;
  /** Current view mode - only tracks readiness in "visual" mode */
  viewMode: "visual" | "code";
  /** Number of nodes in the workflow */
  nodeCount: number;
  /** Optional workflow ID for logging */
  workflowId?: string;
}

/**
 * Hook to detect when React Flow is fully initialized and interactive.
 *
 * React Flow initializes asynchronously, and drag-drop operations require
 * the instance to be fully ready with viewport properly set.
 *
 * For empty workflows, readiness is detected quickly since fitView is a no-op.
 * For workflows with nodes, we wait for fitView animation to complete.
 *
 * @example
 * ```tsx
 * const reactFlowInstance = useReactFlow();
 * const isReady = useReactFlowReady({
 *   reactFlowInstance,
 *   viewMode: "visual",
 *   nodeCount: nodes.length,
 * });
 *
 * return (
 *   <div data-testid="builder" data-ready={isReady}>
 *     ...
 *   </div>
 * );
 * ```
 */
export function useReactFlowReady({
  reactFlowInstance,
  viewMode,
  nodeCount,
  workflowId,
}: UseReactFlowReadyOptions): boolean {
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    // Only check readiness in visual mode
    if (viewMode !== "visual") {
      setIsReady(false);
      return;
    }

    const hasNodes = nodeCount > 0;

    // Empty workflows can be marked ready immediately
    if (!hasNodes) {
      logger.debug("Empty workflow detected, marking ready immediately", {
        component: "useReactFlowReady",
        nodeCount,
        workflowId,
      });
      setIsReady(true);
      return;
    }

    logger.debug("Workflow has nodes, using progressive readiness detection", {
      component: "useReactFlowReady",
      nodeCount,
      workflowId,
    });

    const checkReadiness = (): boolean => {
      try {
        // 1. Instance must exist
        if (!reactFlowInstance) return false;

        // 2. Core methods must be available (required for drag-drop)
        if (typeof reactFlowInstance.project !== "function") return false;
        if (typeof reactFlowInstance.getViewport !== "function") return false;

        // 3. Try to get viewport
        const viewport = reactFlowInstance.getViewport();
        if (!viewport) return false;

        // 4. For non-empty workflows, ensure fitView has completed
        // fitView changes viewport from initial state (0, 0, 1)
        const isDefaultViewport =
          viewport.x === 0 && viewport.y === 0 && viewport.zoom === 1;

        if (isDefaultViewport) return false;

        return true;
      } catch {
        // React Flow may throw if not fully initialized
        return false;
      }
    };

    // Don't re-run if already ready
    if (isReady) {
      return;
    }

    // Check immediately
    if (checkReadiness()) {
      setIsReady(true);
      return;
    }

    // React Flow initializes asynchronously - check at progressive intervals
    // Non-empty workflows need longer for fitView animation to complete
    const timers = [
      setTimeout(() => {
        if (checkReadiness()) setIsReady(true);
      }, 100),
      setTimeout(() => {
        if (checkReadiness()) setIsReady(true);
      }, 300),
      setTimeout(() => {
        if (checkReadiness()) setIsReady(true);
      }, 600),
      setTimeout(() => {
        // Final check - after fitView animation should complete
        if (checkReadiness()) setIsReady(true);
      }, 1200),
      // Fallback: Force ready after 2s if still not ready
      setTimeout(() => {
        if (reactFlowInstance) {
          logger.warn("Forcing ready state after timeout", {
            component: "useReactFlowReady",
            hasInstance: !!reactFlowInstance,
            hasViewport: !!reactFlowInstance?.getViewport(),
            workflowId,
          });
          setIsReady(true);
        }
      }, 2000),
      // Emergency fallback: Force ready after 4s regardless (for tests)
      setTimeout(() => {
        logger.info("Emergency fallback: forcing ready state", {
          component: "useReactFlowReady",
          hasNodes,
          hasInstance: !!reactFlowInstance,
          workflowId,
        });
        setIsReady(true);
      }, 4000),
    ];

    return () => timers.forEach(clearTimeout);
  }, [reactFlowInstance, viewMode, nodeCount, workflowId, isReady]);

  // Reset ready state when workflow changes
  useEffect(() => {
    setIsReady(false);
  }, [workflowId]);

  return isReady;
}

export default useReactFlowReady;

/**
 * useAgentUpdates Hook
 *
 * Listens to WebSocket messages and invalidates React Query caches
 * when agent updates are received. This enables real-time UI updates
 * without polling.
 */

import { useEffect, useCallback, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useWebSocket, type AgentUpdateData, type AgentOutputData } from '../contexts/WebSocketContext.shared';

interface AgentOutputBuffer {
  [agentId: string]: {
    output: string;
    lastSequence: number;
  };
}

interface UseAgentUpdatesOptions {
  /** Callback when any agent update is received */
  onAgentUpdate?: (agent: AgentUpdateData) => void;
  /** Callback for real-time output streaming */
  onAgentOutput?: (data: AgentOutputData) => void;
  /** Callback when an agent is stopped */
  onAgentStopped?: (agentId: string) => void;
  /** Callback when all agents are stopped */
  onAllAgentsStopped?: (stoppedCount: number) => void;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isAgentUpdateData(value: unknown): value is AgentUpdateData {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.status === "string"
  );
}

function isAgentOutputData(value: unknown): value is AgentOutputData {
  return (
    isRecord(value) &&
    typeof value.agentId === "string" &&
    typeof value.output === "string"
  );
}

export function useAgentUpdates(options: UseAgentUpdatesOptions = {}) {
  const { lastMessage, isConnected } = useWebSocket();
  const queryClient = useQueryClient();
  const outputBufferRef = useRef<AgentOutputBuffer>({});

  const { onAgentUpdate, onAgentOutput, onAgentStopped, onAllAgentsStopped } = options;

  // Process WebSocket messages
  useEffect(() => {
    if (!lastMessage) return;

    const { type, data } = lastMessage;

    switch (type) {
      case 'connected':
        // Connection established - could refresh state here
        console.log('[AgentUpdates] WebSocket connected');
        break;

      case 'agent_updated': {
        // Agent status changed - invalidate queries to refresh UI
        queryClient.invalidateQueries({ queryKey: ['active-agents'] });

        // Call optional callback with agent data
        if (onAgentUpdate && isAgentUpdateData(data)) {
          onAgentUpdate(data);
        }
        break;
      }

      case 'agent_output': {
        // Real-time output streaming
        if (onAgentOutput && isAgentOutputData(data)) {
          // Track sequence numbers to avoid duplicates
          const buffer = outputBufferRef.current[data.agentId] || { output: '', lastSequence: -1 };
          const sequence = data.sequence ?? Date.now(); // Use timestamp as fallback sequence

          if (sequence > buffer.lastSequence) {
            buffer.output += data.output;
            buffer.lastSequence = sequence;
            outputBufferRef.current[data.agentId] = buffer;

            onAgentOutput(data);
          }
        }
        break;
      }

      case 'agent_stopped': {
        // Single agent stopped
        queryClient.invalidateQueries({ queryKey: ['active-agents'] });

        if (onAgentStopped && isRecord(data) && typeof data.agentId === "string") {
          onAgentStopped(data.agentId);

          // Clean up output buffer for this agent
          delete outputBufferRef.current[data.agentId];
        }
        break;
      }

      case 'agents_stopped_all': {
        // All agents stopped
        queryClient.invalidateQueries({ queryKey: ['active-agents'] });

        if (onAllAgentsStopped && isRecord(data) && typeof data.stoppedCount === "number") {
          onAllAgentsStopped(data.stoppedCount);

          // Clear all output buffers
          outputBufferRef.current = {};
        }
        break;
      }

      default:
        // Unknown message type - log for debugging
        console.log('[AgentUpdates] Unknown message type:', type);
    }
  }, [lastMessage, queryClient, onAgentUpdate, onAgentOutput, onAgentStopped, onAllAgentsStopped]);

  // Get accumulated output for an agent
  const getAgentOutput = useCallback((agentId: string): string => {
    return outputBufferRef.current[agentId]?.output || '';
  }, []);

  // Clear output buffer for an agent
  const clearAgentOutput = useCallback((agentId: string) => {
    delete outputBufferRef.current[agentId];
  }, []);

  return {
    isConnected,
    getAgentOutput,
    clearAgentOutput,
  };
}

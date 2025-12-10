/**
 * useAgentUpdates Hook
 *
 * Listens to WebSocket messages and invalidates React Query caches
 * when agent updates are received. This enables real-time UI updates
 * without polling.
 */

import { useEffect, useCallback, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useWebSocket, type WebSocketMessage, type AgentUpdateData, type AgentOutputData } from '../contexts/WebSocketContext';

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
        if (onAgentUpdate && data) {
          onAgentUpdate(data as unknown as AgentUpdateData);
        }
        break;
      }

      case 'agent_output': {
        // Real-time output streaming
        if (data && onAgentOutput) {
          const outputData = data as unknown as AgentOutputData;

          // Track sequence numbers to avoid duplicates
          const buffer = outputBufferRef.current[outputData.agentId] || { output: '', lastSequence: -1 };

          if (outputData.sequence > buffer.lastSequence) {
            buffer.output += outputData.output;
            buffer.lastSequence = outputData.sequence;
            outputBufferRef.current[outputData.agentId] = buffer;

            onAgentOutput(outputData);
          }
        }
        break;
      }

      case 'agent_stopped': {
        // Single agent stopped
        queryClient.invalidateQueries({ queryKey: ['active-agents'] });

        if (onAgentStopped && data) {
          const agentId = (data as { agentId: string }).agentId;
          onAgentStopped(agentId);

          // Clean up output buffer for this agent
          delete outputBufferRef.current[agentId];
        }
        break;
      }

      case 'agents_stopped_all': {
        // All agents stopped
        queryClient.invalidateQueries({ queryKey: ['active-agents'] });

        if (onAllAgentsStopped && data) {
          const stoppedCount = (data as { stoppedCount: number }).stoppedCount;
          onAllAgentsStopped(stoppedCount);

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

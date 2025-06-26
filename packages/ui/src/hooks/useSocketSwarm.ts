import { type ChatConfigObject, type SwarmSocketEventPayloads } from "@vrooli/shared";
import { useCallback, useEffect, useRef } from "react";
import { SocketService } from "../api/socket.js";

interface UseSocketSwarmProps {
    chatId?: string | null;
    onConfigUpdate?: (config: Partial<ChatConfigObject>) => void;
    onStateUpdate?: (state: SwarmSocketEventPayloads["swarmStateUpdate"]) => void;
    onResourceUpdate?: (resources: SwarmSocketEventPayloads["swarmResourceUpdate"]) => void;
    onTeamUpdate?: (team: SwarmSocketEventPayloads["swarmTeamUpdate"]) => void;
}

/**
 * Hook to handle swarm-related socket events
 */
export function useSocketSwarm({
    chatId,
    onConfigUpdate,
    onStateUpdate,
    onResourceUpdate,
    onTeamUpdate,
}: UseSocketSwarmProps) {
    const socketRef = useRef<ReturnType<typeof SocketService.get> | null>(null);
    const handlersRef = useRef<{
        swarmConfigUpdate?: (data: SwarmSocketEventPayloads["swarmConfigUpdate"]) => void;
        swarmStateUpdate?: (data: SwarmSocketEventPayloads["swarmStateUpdate"]) => void;
        swarmResourceUpdate?: (data: SwarmSocketEventPayloads["swarmResourceUpdate"]) => void;
        swarmTeamUpdate?: (data: SwarmSocketEventPayloads["swarmTeamUpdate"]) => void;
    }>({});

    // Handle swarm config updates
    const handleSwarmConfigUpdate = useCallback((data: SwarmSocketEventPayloads["swarmConfigUpdate"]) => {
        if (!chatId || data.chatId !== chatId) return;
        
        console.debug("Received swarm config update", { chatId, config: data.config });
        onConfigUpdate?.(data.config);
    }, [chatId, onConfigUpdate]);

    // Handle swarm state updates
    const handleSwarmStateUpdate = useCallback((data: SwarmSocketEventPayloads["swarmStateUpdate"]) => {
        if (!chatId || data.chatId !== chatId) return;
        
        console.debug("Received swarm state update", { chatId, state: data.state, message: data.message });
        onStateUpdate?.(data);
    }, [chatId, onStateUpdate]);

    // Handle swarm resource updates
    const handleSwarmResourceUpdate = useCallback((data: SwarmSocketEventPayloads["swarmResourceUpdate"]) => {
        if (!chatId || data.chatId !== chatId) return;
        
        console.debug("Received swarm resource update", { chatId, resources: data.resources });
        onResourceUpdate?.(data);
    }, [chatId, onResourceUpdate]);

    // Handle swarm team updates
    const handleSwarmTeamUpdate = useCallback((data: SwarmSocketEventPayloads["swarmTeamUpdate"]) => {
        if (!chatId || data.chatId !== chatId) return;
        
        console.debug("Received swarm team update", { chatId, team: data });
        onTeamUpdate?.(data);
    }, [chatId, onTeamUpdate]);

    // Update handlers ref
    useEffect(() => {
        handlersRef.current = {
            swarmConfigUpdate: handleSwarmConfigUpdate,
            swarmStateUpdate: handleSwarmStateUpdate,
            swarmResourceUpdate: handleSwarmResourceUpdate,
            swarmTeamUpdate: handleSwarmTeamUpdate,
        };
    }, [handleSwarmConfigUpdate, handleSwarmStateUpdate, handleSwarmResourceUpdate, handleSwarmTeamUpdate]);

    // Set up socket listeners
    useEffect(() => {
        if (!chatId) return;

        // Skip socket setup in test/storybook environments
        if (typeof window !== "undefined" && window.location?.pathname?.includes("iframe.html")) {
            return;
        }

        socketRef.current = SocketService.get();
        const socket = socketRef.current;

        // Register event handlers
        const unsubscribers: (() => void)[] = [];

        if (handlersRef.current.swarmConfigUpdate) {
            unsubscribers.push(
                socket.onEvent("swarmConfigUpdate", handlersRef.current.swarmConfigUpdate),
            );
        }

        if (handlersRef.current.swarmStateUpdate) {
            unsubscribers.push(
                socket.onEvent("swarmStateUpdate", handlersRef.current.swarmStateUpdate),
            );
        }

        if (handlersRef.current.swarmResourceUpdate) {
            unsubscribers.push(
                socket.onEvent("swarmResourceUpdate", handlersRef.current.swarmResourceUpdate),
            );
        }

        if (handlersRef.current.swarmTeamUpdate) {
            unsubscribers.push(
                socket.onEvent("swarmTeamUpdate", handlersRef.current.swarmTeamUpdate),
            );
        }

        console.debug("Swarm socket listeners registered", { chatId });

        // Cleanup function
        return () => {
            unsubscribers.forEach(unsubscribe => unsubscribe());
            console.debug("Swarm socket listeners unregistered", { chatId });
        };
    }, [chatId]);

    return {
        // Expose methods if needed in the future
    };
}

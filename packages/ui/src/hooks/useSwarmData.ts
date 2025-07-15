import { useCallback, useEffect, useMemo, useState } from "react";
import { type ChatConfigObject, type User, type Team, type NoteVersion, type RoutineVersion, endpointsTeam } from "@vrooli/shared";
import { useFindMany } from "./useFindMany";
import { useLazyFetch } from "./useFetch";

interface SwarmDataResult {
    // Loaded entities
    bots: Record<string, User>;
    resources: Record<string, NoteVersion | RoutineVersion>;
    team: Team | null;
    
    // Loading states
    loading: {
        bots: boolean;
        resources: boolean;
        team: boolean;
    };
    
    // Errors
    errors: {
        bots?: Error;
        resources?: Error;
        team?: Error;
    };
    
    // Helper functions
    getBotById: (id: string) => User | undefined;
    getResourceById: (id: string) => NoteVersion | RoutineVersion | undefined;
}

/**
 * Hook to load all referenced data from a ChatConfigObject
 * Handles batch loading of bots, resources, teams, etc.
 */
export function useSwarmData(config: ChatConfigObject | null): SwarmDataResult {
    console.log("useSwarmData - called with config:", config);
    console.log("useSwarmData - teamId:", config?.teamId);
    
    // Extract all unique bot IDs from the config
    const botIds = useMemo(() => {
        if (!config) return [];
        
        const ids = new Set<string>();
        
        // Add swarm leader
        if (config.swarmLeader) ids.add(config.swarmLeader);
        
        // Add subtask leaders
        if (config.subtaskLeaders) {
            Object.values(config.subtaskLeaders).forEach(id => ids.add(id));
        }
        
        // Add assignees from subtasks
        config.subtasks?.forEach(task => {
            if (task.assignee_bot_id) ids.add(task.assignee_bot_id);
        });
        
        // Add resource creators
        config.resources?.forEach(resource => {
            if (resource.creator_bot_id) ids.add(resource.creator_bot_id);
        });
        
        // Add tool call invokers
        config.records?.forEach(record => {
            if (record.caller_bot_id) ids.add(record.caller_bot_id);
        });
        
        // Add pending tool call bots
        config.pendingToolCalls?.forEach(pending => {
            if (pending.callerBotId) ids.add(pending.callerBotId);
        });
        
        return Array.from(ids);
    }, [config]);
    
    // Extract all unique resource IDs
    const resourceIds = useMemo(() => {
        if (!config) return [];
        
        const ids = new Set<string>();
        
        // Add resources
        config.resources?.forEach(resource => {
            ids.add(resource.id);
        });
        
        // Add output resources from tool calls
        config.records?.forEach(record => {
            record.output_resource_ids?.forEach(id => ids.add(id));
        });
        
        return Array.from(ids);
    }, [config]);
    
    // Load bots using useFindMany
    const { 
        data: botsData, 
        loading: botsLoading, 
        error: botsError, 
    } = useFindMany<User>({
        searchType: "User",
        where: botIds.length > 0 ? {
            id: { in: botIds },
            isBot: true,
        } : undefined,
        skip: botIds.length === 0,
    });
    
    // Load resources (for now just Notes, can be expanded)
    const { 
        data: resourcesData, 
        loading: resourcesLoading, 
        error: resourcesError, 
    } = useFindMany<NoteVersion>({
        searchType: "NoteVersion",
        where: resourceIds.length > 0 ? {
            id: { in: resourceIds },
        } : undefined,
        skip: resourceIds.length === 0,
    });
    
    // Load team data using single team endpoint
    const [fetchTeam, { data: teamResponse, loading: teamLoading, errors: teamErrors }] = useLazyFetch<{ publicId: string }, Team>({
        endpoint: endpointsTeam.findOne.endpoint,
    });
    
    useEffect(() => {
        if (config?.teamId) {
            console.log("useSwarmData - fetching team with ID:", config.teamId);
            console.log("useSwarmData - endpoint:", endpointsTeam.findOne.endpoint);
            console.log("useSwarmData - calling fetchTeam with:", { publicId: config.teamId });
            fetchTeam({ publicId: config.teamId });
        }
    }, [config?.teamId, fetchTeam]);
    
    console.log("useSwarmData - team response:", teamResponse);
    console.log("useSwarmData - team loading:", teamLoading);
    console.log("useSwarmData - team errors:", teamErrors);
    
    // Convert arrays to lookup maps
    const bots = useMemo(() => {
        const map: Record<string, User> = {};
        botsData?.edges?.forEach(edge => {
            if (edge.node) {
                map[edge.node.id] = edge.node;
            }
        });
        return map;
    }, [botsData]);
    
    const resources = useMemo(() => {
        const map: Record<string, NoteVersion | RoutineVersion> = {};
        resourcesData?.edges?.forEach(edge => {
            if (edge.node) {
                map[edge.node.id] = edge.node;
            }
        });
        return map;
    }, [resourcesData]);
    
    const team = teamResponse || null;
    console.log("useSwarmData - extracted team:", team);
    
    // Helper functions
    const getBotById = useCallback((id: string) => bots[id], [bots]);
    const getResourceById = useCallback((id: string) => resources[id], [resources]);
    
    const result = {
        bots,
        resources,
        team,
        loading: {
            bots: botsLoading,
            resources: resourcesLoading,
            team: teamLoading,
        },
        errors: {
            bots: botsError,
            resources: resourcesError,
            team: teamErrors?.[0], // useLazyFetch returns errors array
        },
        getBotById,
        getResourceById,
    };
    
    console.log("useSwarmData - returning:", result);
    return result;
}

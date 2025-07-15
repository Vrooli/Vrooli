import { useCallback, useEffect, useMemo, useState } from "react";
import { type ChatMessageRunConfig } from "@vrooli/shared";

interface RunStatesStorage {
    [messageId: string]: {
        [runId: string]: boolean; // true = collapsed, false = expanded
    };
}

const STORAGE_KEY = "chat-run-states";

/**
 * Hook to manage collapsed/expanded states for multiple run executors within a message
 */
export function useRunStates(messageId: string, runs: ChatMessageRunConfig[]) {
    const [collapsedStates, setCollapsedStates] = useState<Record<string, boolean>>({});

    // Load states from localStorage on mount
    useEffect(() => {
        try {
            const storedStates = localStorage.getItem(STORAGE_KEY);
            if (storedStates) {
                const parsed: RunStatesStorage = JSON.parse(storedStates);
                const messageStates = parsed[messageId] || {};
                setCollapsedStates(messageStates);
            }
        } catch (error) {
            console.warn("Failed to load run states from localStorage:", error);
        }
    }, [messageId]);

    // Save states to localStorage whenever they change
    const saveStates = useCallback((newStates: Record<string, boolean>) => {
        try {
            const storedStates = localStorage.getItem(STORAGE_KEY);
            const allStates: RunStatesStorage = storedStates ? JSON.parse(storedStates) : {};
            allStates[messageId] = newStates;
            localStorage.setItem(STORAGE_KEY, JSON.stringify(allStates));
        } catch (error) {
            console.warn("Failed to save run states to localStorage:", error);
        }
    }, [messageId]);

    // Toggle collapse state for a specific run
    const toggleCollapsed = useCallback((runId: string) => {
        setCollapsedStates(prev => {
            const newStates = {
                ...prev,
                [runId]: !prev[runId], // Default to false (expanded) if not set
            };
            saveStates(newStates);
            return newStates;
        });
    }, [saveStates]);

    // Remove a run from the states (when user closes/removes it)
    const removeRun = useCallback((runId: string) => {
        setCollapsedStates(prev => {
            const newStates = { ...prev };
            delete newStates[runId];
            saveStates(newStates);
            return newStates;
        });
    }, [saveStates]);

    // Check if all runs are collapsed
    const allCollapsed = useMemo(() => {
        if (runs.length === 0) return false;
        return runs.every(run => collapsedStates[run.runId] === true);
    }, [runs, collapsedStates]);

    // Check if all runs are expanded
    const allExpanded = useMemo(() => {
        if (runs.length === 0) return false;
        return runs.every(run => collapsedStates[run.runId] !== true);
    }, [runs, collapsedStates]);

    // Toggle all runs collapsed/expanded
    const toggleAllCollapsed = useCallback(() => {
        const newCollapsedState = !allCollapsed;
        const newStates: Record<string, boolean> = {};
        
        runs.forEach(run => {
            newStates[run.runId] = newCollapsedState;
        });

        setCollapsedStates(newStates);
        saveStates(newStates);
    }, [runs, allCollapsed, saveStates]);

    // Collapse all runs
    const collapseAll = useCallback(() => {
        const newStates: Record<string, boolean> = {};
        runs.forEach(run => {
            newStates[run.runId] = true;
        });
        setCollapsedStates(newStates);
        saveStates(newStates);
    }, [runs, saveStates]);

    // Expand all runs
    const expandAll = useCallback(() => {
        const newStates: Record<string, boolean> = {};
        runs.forEach(run => {
            newStates[run.runId] = false;
        });
        setCollapsedStates(newStates);
        saveStates(newStates);
    }, [runs, saveStates]);

    // Clean up old message states to prevent localStorage bloat
    const cleanupOldStates = useCallback(() => {
        try {
            const storedStates = localStorage.getItem(STORAGE_KEY);
            if (!storedStates) return;

            const allStates: RunStatesStorage = JSON.parse(storedStates);
            const messageIds = Object.keys(allStates);
            
            // Keep only the last 50 messages to prevent unlimited growth
            if (messageIds.length > 50) {
                const sortedIds = messageIds.sort();
                const toKeep = sortedIds.slice(-50);
                const cleanedStates: RunStatesStorage = {};
                
                toKeep.forEach(id => {
                    cleanedStates[id] = allStates[id];
                });
                
                localStorage.setItem(STORAGE_KEY, JSON.stringify(cleanedStates));
            }
        } catch (error) {
            console.warn("Failed to cleanup old run states:", error);
        }
    }, []);

    // Run cleanup periodically
    useEffect(() => {
        const cleanup = setTimeout(cleanupOldStates, 1000);
        return () => clearTimeout(cleanup);
    }, [cleanupOldStates]);

    return {
        collapsedStates,
        toggleCollapsed,
        removeRun,
        allCollapsed,
        allExpanded,
        toggleAllCollapsed,
        collapseAll,
        expandAll,
    };
}

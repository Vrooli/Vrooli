import { endpointsResource, ResourceType, ResourceVersion, ResourceVersionSearchInput, ResourceVersionSearchResult, ResourceVersionSortBy } from "@local/shared";
import { useContext, useEffect } from "react";
import { create } from "zustand";
import { fetchData } from "../api/fetchData.js";
import { ServerResponseParser } from "../api/responseParser.js";
import { SessionContext } from "../contexts/session.js";
import { checkIfLoggedIn } from "../utils/authentication/session.js";

interface ProjectsState {
    projects: ResourceVersion[];
    isLoading: boolean;
    error: string | null;
    /** Selected project ID */
    selectedProjectId: string | null;
    /** Fetch projects from the server */
    fetchProjects: (signal?: AbortSignal) => Promise<ResourceVersion[]>;
    /** Set projects list */
    setProjects: (projects: ResourceVersion[] | ((prev: ResourceVersion[]) => ResourceVersion[])) => void;
    /** Add a new project to the list */
    addProject: (project: ResourceVersion) => void;
    /** Remove a project from the list by ID */
    removeProject: (projectId: string) => void;
    /** Update an existing project in the list */
    updateProject: (updatedProject: ResourceVersion) => void;
    /** Select a project by ID */
    selectProject: (projectId: string) => void;
    /** Clear all projects and reset state */
    clearProjects: () => void;
}

export const useProjectsStore = create<ProjectsState>()((set, get) => ({
    projects: [],
    isLoading: false,
    error: null,
    selectedProjectId: null,
    fetchProjects: async (signal) => {
        // Avoid refetching if already fetched or in progress.
        if (get().projects.length > 0 || get().isLoading) {
            return get().projects;
        }
        set({ isLoading: true, error: null });
        try {
            const response = await fetchData<ResourceVersionSearchInput, ResourceVersionSearchResult>({
                ...endpointsResource.findMany,
                inputs: { resourceType: ResourceType.Project, sortBy: ResourceVersionSortBy.DateUpdatedDesc },
                signal,
            });
            if (response.errors) {
                ServerResponseParser.displayErrors(response.errors);
                set({ error: "Failed to fetch projects", isLoading: false });
                throw new Error("Failed to fetch projects");
            }
            const projects = response.data?.edges.map(edge => edge.node) ?? [];

            // If there are projects and none selected, select the first one
            if (projects.length > 0 && get().selectedProjectId === null) {
                set({ selectedProjectId: projects[0].id });
            }

            set({ projects, isLoading: false });
            return projects;
        } catch (error) {
            if ((error as { name?: string }).name === "AbortError") return [];
            console.error("Error fetching projects:", error);
            set({ error: "Error fetching projects", isLoading: false });
            return [];
        }
    },
    setProjects: (projects) => {
        set((state) => ({
            projects:
                typeof projects === "function"
                    ? projects(state.projects)
                    : projects,
        }));
    },
    addProject: (project) => {
        set((state) => ({
            projects: [project, ...state.projects],
        }));
    },
    removeProject: (projectId) => {
        set((state) => ({
            projects: state.projects.filter((project) => project.id !== projectId),
            // Reset selected project if the selected one was removed
            selectedProjectId: state.selectedProjectId === projectId ? null : state.selectedProjectId,
        }));
    },
    updateProject: (updatedProject) => {
        set((state) => ({
            projects: state.projects.map((project) =>
                project.id === updatedProject.id ? updatedProject : project,
            ),
        }));
    },
    selectProject: (projectId) => {
        set({ selectedProjectId: projectId });
    },
    clearProjects: () => {
        set({ projects: [], isLoading: false, error: null, selectedProjectId: null });
    },
}));

/**
 * Custom hook to manage project fetching and potentially real-time updates.
 * This hook handles:
 * 1. Fetching initial projects when the user logs in and the store is empty.
 * 2. Clearing projects from the store when the user logs out.
 *
 * Components needing the project list should consume it directly from the store:
 * `const projects = useProjectsStore(state => state.projects);`
 */
export function useProjects(): void {
    const fetchProjects = useProjectsStore(state => state.fetchProjects);
    const clearProjectsStore = useProjectsStore(state => state.clearProjects);
    const getStoreState = useProjectsStore.getState;

    const session = useContext(SessionContext);
    const isLoggedIn = checkIfLoggedIn(session);

    useEffect(function fetchExistingProjectsEffect() {
        const abortController = new AbortController();

        if (!isLoggedIn) {
            if (getStoreState().projects.length > 0) {
                clearProjectsStore();
            }
        } else {
            const currentState = getStoreState();
            if (currentState.projects.length === 0 && !currentState.isLoading && !currentState.error) {
                fetchProjects(abortController.signal).catch(error => {
                    if ((error as { name?: string }).name !== "AbortError") {
                        console.error("Initial project fetch failed:", error);
                    }
                });
            }
        }

        return () => {
            abortController.abort();
        };
    }, [isLoggedIn, fetchProjects, clearProjectsStore, session]);

    // Add WebSocket listener for real-time updates if needed later
    // useEffect(function listenForProjectsEffect() { ... });
} 

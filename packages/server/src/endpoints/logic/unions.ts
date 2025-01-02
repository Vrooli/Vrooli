import { PageInfo, ProjectOrRoutine, ProjectOrRoutineSearchInput, ProjectOrRoutineSearchResult, ProjectOrTeam, ProjectOrTeamSearchInput, ProjectOrTeamSearchResult, ProjectSortBy, RoutineSortBy, RunProjectOrRunRoutine, RunProjectOrRunRoutineSearchInput, RunProjectOrRunRoutineSearchResult, RunProjectOrRunRoutineSortBy, TeamSortBy, VisibilityType } from "@local/shared";
import { readManyAsFeedHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { SessionService } from "../../auth/session";
import { addSupplementalFieldsMultiTypes } from "../../builders/addSupplementalFieldsMultiTypes";
import { toPartialGqlInfo } from "../../builders/toPartialGqlInfo";
import { ApiEndpointInfo } from "../../builders/types";
import { ApiEndpoint, FindManyResult } from "../../types";
import { SearchMap } from "../../utils";

const DEFAULT_TAKE = 10;

export type EndpointsUnions = {
    projectOrRoutines: ApiEndpoint<ProjectOrRoutineSearchInput, FindManyResult<ProjectOrRoutine>>;
    runProjectOrRunRoutines: ApiEndpoint<RunProjectOrRunRoutineSearchInput, FindManyResult<RunProjectOrRunRoutine>>;
    projectOrTeams: ApiEndpoint<ProjectOrTeamSearchInput, FindManyResult<ProjectOrTeam>>;
}

export const unions: EndpointsUnions = {
    projectOrRoutines: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 2000, req });
        const partial = toPartialGqlInfo(info, {
            __typename: "ProjectOrRoutineSearchResult",
            Project: "Project",
            Routine: "Routine",
        }, true);
        const take = Math.ceil((input.take ?? DEFAULT_TAKE) / 2);
        const commonReadParams = { req };
        // If any "after" cursor is provided, we can assume that missing cursors mean that we've reached the end for that object type
        const anyAfters = Object.entries(input).some(([key, value]) => key.endsWith("After") && typeof value === "string" && value.trim() !== "");
        // Checks if object type should be included in results
        function shouldInclude(objectType: `${ProjectOrRoutineSearchInput["objectType"]}`) {
            if (anyAfters && (input[`${objectType.toLowerCase()}After`]?.trim() ?? "") === "") return false;
            return input.objectType ? input.objectType === objectType : true;
        }
        // Collect search data
        const userData = SessionService.getUser(req.session);
        const visibility = input.visibility ?? VisibilityType.Public;
        const searchData = { userData, visibility };
        const searchLanguageFunc = SearchMap.translationLanguagesLatestVersion;
        // Query projects
        const { nodes: projects, pageInfo: projectsInfo } = shouldInclude("Project") ? await readManyAsFeedHelper({
            ...commonReadParams,
            info: partial.Project as ApiEndpointInfo,
            input: {
                after: input.projectAfter,
                createdTimeFrame: input.createdTimeFrame,
                excludeIds: input.excludeIds,
                hasCompleteVersion: input.hasCompleteVersion,
                hasCompleteExceptions: input.hasCompleteVersionExceptions,
                ids: input.ids,
                languages: input.translationLanguagesLatestVersion && searchLanguageFunc ? searchLanguageFunc(input.translationLanguagesLatestVersion, { ...searchData, objectType: "Project", searchInput: {} }) : undefined,
                maxBookmarks: input.maxBookmarks,
                maxScore: input.maxScore,
                minBookmarks: input.minBookmarks,
                minScore: input.minScore,
                minViews: input.minViews,
                parentId: input.parentId,
                reportId: input.reportId,
                resourceTypes: input.resourceTypes,
                searchString: input.searchString ? input.searchString : undefined,
                sortBy: input.sortBy as ProjectSortBy | undefined,
                tags: input.tags,
                take,
                teamId: input.teamId,
                updatedTimeFrame: input.updatedTimeFrame,
                userId: userData?.id,
            },
            objectType: "Project",
            visibility,
        }) : { nodes: [], pageInfo: {} } as { nodes: object[], pageInfo: Partial<PageInfo> };
        // Query routines
        const { nodes: routines, pageInfo: routinesInfo } = shouldInclude("Routine") ? await readManyAsFeedHelper({
            ...commonReadParams,
            info: partial.Routine as ApiEndpointInfo,
            input: {
                after: input.routineAfter,
                createdTimeFrame: input.createdTimeFrame,
                excludeIds: input.excludeIds,
                ids: input.ids,
                isInternal: false,
                hasCompleteVersion: input.hasCompleteVersion,
                hasCompleteExceptions: input.hasCompleteVersionExceptions,
                languages: input.translationLanguagesLatestVersion && searchLanguageFunc ? searchLanguageFunc(input.translationLanguagesLatestVersion, { ...searchData, objectType: "Routine", searchInput: {} }) : undefined,
                minComplexity: input.routineMinComplexity,
                maxComplexity: input.routineMaxComplexity,
                maxBookmarks: input.maxBookmarks,
                maxScore: input.maxScore,
                minBookmarks: input.minBookmarks,
                minScore: input.minScore,
                minSimplicity: input.routineMinSimplicity,
                minTimesCompleted: input.routineMinTimesCompleted,
                maxTimesCompleted: input.routineMaxTimesCompleted,
                minViews: input.minViews,
                parentId: input.parentId,
                projectId: input.routineProjectId,
                reportId: input.reportId,
                resourceTypes: input.resourceTypes,
                searchString: input.searchString ? input.searchString : undefined,
                sortBy: input.sortBy as unknown as RoutineSortBy,
                tags: input.tags,
                take,
                teamId: input.teamId,
                updatedTimeFrame: input.updatedTimeFrame,
                userId: userData?.id,
            },
            objectType: "Routine",
            visibility,
        }) : { nodes: [], pageInfo: {} } as { nodes: object[], pageInfo: Partial<PageInfo> };
        // Add supplemental fields to every result
        const withSupplemental = await addSupplementalFieldsMultiTypes({ projects, routines }, {
            projects: { type: "Project", ...(partial.Project as ApiEndpointInfo) },
            routines: { type: "Routine", ...(partial.Routine as ApiEndpointInfo) },
        }, userData);
        // Combine nodes, alternating between each type
        const properties = Object.values(withSupplemental);
        const maxLen = Math.max(...properties.map(arr => arr.length));
        const nodes: ProjectOrRoutine[] = Array.from({ length: maxLen })
            .flatMap((_, i) => properties.map(prop => prop[i]))
            .filter(Boolean);
        // Combine pageInfo
        const combined: ProjectOrRoutineSearchResult = {
            __typename: "ProjectOrRoutineSearchResult" as const,
            pageInfo: {
                __typename: "ProjectOrRoutinePageInfo" as const,
                hasNextPage: projectsInfo.hasNextPage || routinesInfo.hasNextPage || false,
                endCursorProject: projectsInfo.endCursor ?? "",
                endCursorRoutine: routinesInfo.endCursor ?? "",
            },
            edges: nodes.map((node) => ({ __typename: "ProjectOrRoutineEdge" as const, cursor: node.id, node })),
        };
        return combined;
    },
    runProjectOrRunRoutines: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 2000, req });
        const partial = toPartialGqlInfo(info, {
            __typename: "RunProjectOrRunRoutineSearchResult",
            RunProject: "RunProject",
            RunRoutine: "RunRoutine",
        }, true);
        const take = Math.ceil((input.take ?? DEFAULT_TAKE) / 2);
        const commonReadParams = { req };
        // If any "after" cursor is provided, we can assume that missing cursors mean that we've reached the end for that object type
        const anyAfters = Object.entries(input).some(([key, value]) => key.endsWith("After") && typeof value === "string" && value.trim() !== "");
        // Checks if object type should be included in results
        function shouldInclude(objectType: `${RunProjectOrRunRoutineSearchInput["objectType"]}`) {
            if (anyAfters && (input[`${objectType.toLowerCase()}After`]?.trim() ?? "") === "") return false;
            return input.objectType ? input.objectType === objectType : true;
        }
        const userData = SessionService.getUser(req.session);
        const visibility = input.visibility ?? VisibilityType.Public;
        // Query run projects
        const { nodes: runProjects, pageInfo: runProjectsInfo } = shouldInclude("RunProject") ? await readManyAsFeedHelper({
            ...commonReadParams,
            info: partial.RunProject as ApiEndpointInfo,
            input: {
                after: input.runProjectAfter,
                createdTimeFrame: input.createdTimeFrame,
                startedTimeFrame: input.startedTimeFrame,
                completedTimeFrame: input.completedTimeFrame,
                excludeIds: input.excludeIds,
                ids: input.ids,
                status: input.status,
                statuses: input.statuses,
                projectVersionId: input.projectVersionId,
                searchString: input.searchString ? input.searchString : undefined,
                sortBy: input.sortBy as unknown as RunProjectOrRunRoutineSortBy,
                take,
                updatedTimeFrame: input.updatedTimeFrame,
                userId: userData?.id,
            },
            objectType: "RunProject",
            visibility,
        }) : { nodes: [], pageInfo: {} } as { nodes: object[], pageInfo: Partial<PageInfo> };
        // Query routines
        const { nodes: runRoutines, pageInfo: runRoutinesInfo } = shouldInclude("RunRoutine") ? await readManyAsFeedHelper({
            ...commonReadParams,
            info: partial.RunRoutine as ApiEndpointInfo,
            input: {
                after: input.runProjectAfter,
                createdTimeFrame: input.createdTimeFrame,
                startedTimeFrame: input.startedTimeFrame,
                completedTimeFrame: input.completedTimeFrame,
                excludeIds: input.excludeIds,
                ids: input.ids,
                status: input.status,
                statuses: input.statuses,
                projectVersionId: input.projectVersionId,
                searchString: input.searchString ? input.searchString : undefined,
                sortBy: input.sortBy as unknown as RunProjectOrRunRoutineSortBy,
                take,
                updatedTimeFrame: input.updatedTimeFrame,
                userId: userData?.id,
            },
            objectType: "RunRoutine",
            visibility,
        }) : { nodes: [], pageInfo: {} } as { nodes: object[], pageInfo: Partial<PageInfo> };
        // Add supplemental fields to every result
        const withSupplemental = await addSupplementalFieldsMultiTypes({ runProjects, runRoutines }, {
            runProjects: { type: "RunProject", ...(partial.RunProject as ApiEndpointInfo) },
            runRoutines: { type: "RunRoutine", ...(partial.RunRoutine as ApiEndpointInfo) },
        }, userData);
        // Combine nodes, alternating between each type
        const properties = Object.values(withSupplemental);
        const maxLen = Math.max(...properties.map(arr => arr.length));
        const nodes: RunProjectOrRunRoutine[] = Array.from({ length: maxLen })
            .flatMap((_, i) => properties.map(prop => prop[i]))
            .filter(Boolean);
        // Combine pageInfo
        const combined: RunProjectOrRunRoutineSearchResult = {
            __typename: "RunProjectOrRunRoutineSearchResult" as const,
            pageInfo: {
                __typename: "RunProjectOrRunRoutinePageInfo" as const,
                hasNextPage: runProjectsInfo.hasNextPage || runRoutinesInfo.hasNextPage || false,
                endCursorRunProject: runProjectsInfo.endCursor ?? "",
                endCursorRunRoutine: runRoutinesInfo.endCursor ?? "",
            },
            edges: nodes.map((node) => ({ __typename: "RunProjectOrRunRoutineEdge" as const, cursor: node.id, node })),
        };
        return combined;
    },
    projectOrTeams: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 2000, req });
        const partial = toPartialGqlInfo(info, {
            __typename: "ProjectOrTeamSearchResult",
            Project: "Project",
            Team: "Team",
        }, true);
        const take = Math.ceil((input.take ?? DEFAULT_TAKE) / 2);
        const commonReadParams = { req };
        // If any "after" cursor is provided, we can assume that missing cursors mean that we've reached the end for that object type
        const anyAfters = Object.entries(input).some(([key, value]) => key.endsWith("After") && typeof value === "string" && value.trim() !== "");
        // Checks if object type should be included in results
        function shouldInclude(objectType: `${ProjectOrTeamSearchInput["objectType"]}`) {
            if (anyAfters && (input[`${objectType.toLowerCase()}After`]?.trim() ?? "") === "") return false;
            return input.objectType ? input.objectType === objectType : true;
        }
        // Collect search data
        const userData = SessionService.getUser(req.session);
        const visibility = input.visibility ?? VisibilityType.Public;
        const searchData = { userData, visibility };
        const searchLanguageFunc = SearchMap.translationLanguagesLatestVersion;
        // Query projects
        const { nodes: projects, pageInfo: projectsInfo } = shouldInclude("Project") ? await readManyAsFeedHelper({
            ...commonReadParams,
            info: partial.Project as ApiEndpointInfo,
            input: {
                after: input.projectAfter,
                createdTimeFrame: input.createdTimeFrame,
                excludeIds: input.excludeIds,
                ids: input.ids,
                isComplete: input.projectIsComplete,
                isCompleteExceptions: input.projectIsCompleteExceptions,
                languages: input.translationLanguagesLatestVersion && searchLanguageFunc ? searchLanguageFunc(input.translationLanguagesLatestVersion, { ...searchData, objectType: "Project", searchInput: {} }) : undefined,
                maxBookmarks: input.maxBookmarks,
                maxScore: input.projectMaxScore,
                maxViews: input.maxViews,
                minBookmarks: input.minBookmarks,
                minScore: input.projectMinScore,
                minViews: input.minViews,
                parentId: input.projectParentId,
                reportId: input.reportId,
                resourceTypes: input.resourceTypes,
                searchString: input.searchString ? input.searchString : undefined,
                sortBy: input.sortBy as unknown as ProjectSortBy,
                tags: input.tags,
                take,
                teamId: input.projectTeamId,
                updatedTimeFrame: input.updatedTimeFrame,
                userId: userData?.id,
            },
            objectType: "Project",
            visibility,
        }) : { nodes: [], pageInfo: {} } as { nodes: object[], pageInfo: Partial<PageInfo> };
        // Query teams
        const { nodes: teams, pageInfo: teamsInfo } = shouldInclude("Team") ? await readManyAsFeedHelper({
            ...commonReadParams,
            info: partial.Team as ApiEndpointInfo,
            input: {
                after: input.teamAfter,
                createdTimeFrame: input.createdTimeFrame,
                excludeIds: input.excludeIds,
                ids: input.ids,
                languages: input.translationLanguagesLatestVersion,
                maxBookmarks: input.maxBookmarks,
                minBookmarks: input.minBookmarks,
                maxViews: input.maxViews,
                minViews: input.minViews,
                isOpenToNewMembers: input.teamIsOpenToNewMembers,
                projectId: input.teamProjectId,
                routineId: input.teamRoutineId,
                reportId: input.reportId,
                resourceTypes: input.resourceTypes,
                searchString: input.searchString ? input.searchString : undefined,
                sortBy: input.sortBy as unknown as TeamSortBy,
                tags: input.tags,
                take,
                updatedTimeFrame: input.updatedTimeFrame,
                userId: userData?.id,
            },
            objectType: "Team",
            visibility,
        }) : { nodes: [], pageInfo: {} } as { nodes: object[], pageInfo: Partial<PageInfo> };
        // Add supplemental fields to every result
        const withSupplemental = await addSupplementalFieldsMultiTypes({ projects, teams }, {
            projects: { type: "Project", ...(partial.Project as ApiEndpointInfo) },
            teams: { type: "Team", ...(partial.Team as ApiEndpointInfo) },
        }, userData);
        // Combine nodes, alternating between each type
        const properties = Object.values(withSupplemental);
        const maxLen = Math.max(...properties.map(arr => arr.length));
        const nodes: ProjectOrTeam[] = Array.from({ length: maxLen })
            .flatMap((_, i) => properties.map(prop => prop[i]))
            .filter(Boolean);
        // Combine pageInfo
        const combined: ProjectOrTeamSearchResult = {
            __typename: "ProjectOrTeamSearchResult" as const,
            pageInfo: {
                __typename: "ProjectOrTeamPageInfo" as const,
                hasNextPage: projectsInfo.hasNextPage || teamsInfo.hasNextPage || false,
                endCursorProject: projectsInfo.endCursor ?? "",
                endCursorTeam: teamsInfo.endCursor ?? "",
            },
            edges: nodes.map((node) => ({ __typename: "ProjectOrTeamEdge" as const, cursor: node.id, node })),
        };
        return combined;
    },
};

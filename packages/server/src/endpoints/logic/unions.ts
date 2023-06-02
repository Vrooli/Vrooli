import { OrganizationSortBy, ProjectOrOrganization, ProjectOrOrganizationSearchInput, ProjectOrOrganizationSearchResult, ProjectOrRoutine, ProjectOrRoutineSearchInput, ProjectOrRoutineSearchResult, ProjectSortBy, RoutineSortBy, RunProjectOrRunRoutine, RunProjectOrRunRoutineSearchInput, RunProjectOrRunRoutineSearchResult, RunProjectOrRunRoutineSortBy } from "@local/shared";
import { readManyAsFeedHelper } from "../../actions";
import { getUser } from "../../auth";
import { addSupplementalFieldsMultiTypes, toPartialGqlInfo } from "../../builders";
import { rateLimit } from "../../middleware";
import { FindManyResult, GQLEndpoint } from "../../types";
import { SearchMap } from "../../utils";

export type EndpointsUnions = {
    Query: {
        projectOrRoutines: GQLEndpoint<ProjectOrRoutineSearchInput, FindManyResult<ProjectOrRoutine>>;
        projectOrOrganizations: GQLEndpoint<ProjectOrOrganizationSearchInput, FindManyResult<ProjectOrOrganization>>;
        runProjectOrRunRoutines: GQLEndpoint<RunProjectOrRunRoutineSearchInput, FindManyResult<RunProjectOrRunRoutine>>;
    },
}

export const UnionsEndpoints: EndpointsUnions = {
    Query: {
        projectOrRoutines: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 2000, req });
            const partial = toPartialGqlInfo(info, {
                __typename: "ProjectOrRoutineSearchResult",
                Project: "Project",
                Routine: "Routine",
            }, req.languages, true);
            const take = Math.ceil((input.take ?? 10) / 2);
            const commonReadParams = { prisma, req };
            // Query projects
            let projects;
            if (input.objectType === undefined || input.objectType === "Project") {
                projects = await readManyAsFeedHelper({
                    ...commonReadParams,
                    info: (partial as any).Project,
                    input: {
                        after: input.projectAfter,
                        createdTimeFrame: input.createdTimeFrame,
                        excludeIds: input.excludeIds,
                        hasCompleteVersion: input.hasCompleteVersion,
                        hasCompleteExceptions: input.hasCompleteVersionExceptions,
                        ids: input.ids,
                        languages: input.translationLanguagesLatestVersion ? SearchMap.translationLanguagesLatestVersion(input.translationLanguagesLatestVersion) : undefined,
                        maxBookmarks: input.maxBookmarks,
                        maxScore: input.maxScore,
                        minBookmarks: input.minBookmarks,
                        minScore: input.minScore,
                        minViews: input.minViews,
                        organizationId: input.organizationId,
                        parentId: input.parentId,
                        reportId: input.reportId,
                        resourceTypes: input.resourceTypes,
                        searchString: input.searchString,
                        sortBy: input.sortBy as unknown as ProjectSortBy,
                        tags: input.tags,
                        take,
                        updatedTimeFrame: input.updatedTimeFrame,
                        userId: getUser(req)?.id,
                        visibility: input.visibility,
                    },
                    objectType: "Project",
                });
            }
            // Query routines
            let routines;
            if (input.objectType === undefined || input.objectType === "Routine") {
                routines = await readManyAsFeedHelper({
                    ...commonReadParams,
                    info: (partial as any).Routine,
                    input: {
                        after: input.routineAfter,
                        createdTimeFrame: input.createdTimeFrame,
                        excludeIds: input.excludeIds,
                        ids: input.ids,
                        isInternal: false,
                        hasCompleteVersion: input.hasCompleteVersion,
                        hasCompleteExceptions: input.hasCompleteVersionExceptions,
                        languages: input.translationLanguagesLatestVersion ? SearchMap.translationLanguagesLatestVersion(input.translationLanguagesLatestVersion) : undefined,
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
                        organizationId: input.organizationId,
                        parentId: input.parentId,
                        projectId: input.routineProjectId,
                        reportId: input.reportId,
                        resourceTypes: input.resourceTypes,
                        searchString: input.searchString,
                        sortBy: input.sortBy as unknown as RoutineSortBy,
                        tags: input.tags,
                        take,
                        updatedTimeFrame: input.updatedTimeFrame,
                        userId: getUser(req)?.id,
                        visibility: input.visibility,
                    },
                    objectType: "Routine",
                });
            }
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes({
                projects: projects?.nodes ?? [],
                routines: routines?.nodes ?? [],
            }, {
                projects: { type: "Project", ...(partial as any).Project },
                routines: { type: "Routine", ...(partial as any).Routine },
            }, prisma, getUser(req));
            // Combine nodes, alternating between projects and routines
            const nodes: ProjectOrRoutine[] = [];
            for (let i = 0; i < Math.max(withSupplemental.projects.length, withSupplemental.routines.length); i++) {
                if (i < withSupplemental.projects.length) {
                    nodes.push(withSupplemental.projects[i]);
                }
                if (i < withSupplemental.routines.length) {
                    nodes.push(withSupplemental.routines[i]);
                }
            }
            // Combine pageInfo
            const combined: ProjectOrRoutineSearchResult = {
                __typename: "ProjectOrRoutineSearchResult" as const,
                pageInfo: {
                    __typename: "ProjectOrRoutinePageInfo" as const,
                    hasNextPage: projects?.pageInfo?.hasNextPage ?? routines?.pageInfo?.hasNextPage ?? false,
                    endCursorProject: projects?.pageInfo?.endCursor ?? "",
                    endCursorRoutine: routines?.pageInfo?.endCursor ?? "",
                },
                edges: nodes.map((node) => ({ __typename: "ProjectOrRoutineEdge" as const, cursor: node.id, node })),
            };
            return combined;
        },
        projectOrOrganizations: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 2000, req });
            const partial = toPartialGqlInfo(info, {
                __typename: "ProjectOrOrganizationSearchResult",
                Project: "Project",
                Organization: "Organization",
            }, req.languages, true);
            const take = Math.ceil((input.take ?? 10) / 2);
            const commonReadParams = { prisma, req };
            // Query projects
            let projects;
            if (input.objectType === undefined || input.objectType === "Project") {
                projects = await readManyAsFeedHelper({
                    ...commonReadParams,
                    info: (partial as any).Project,
                    input: {
                        after: input.projectAfter,
                        createdTimeFrame: input.createdTimeFrame,
                        excludeIds: input.excludeIds,
                        ids: input.ids,
                        isComplete: input.projectIsComplete,
                        isCompleteExceptions: input.projectIsCompleteExceptions,
                        languages: input.translationLanguagesLatestVersion ? SearchMap.translationLanguagesLatestVersion(input.translationLanguagesLatestVersion) : undefined,
                        maxBookmarks: input.maxBookmarks,
                        maxScore: input.projectMaxScore,
                        maxViews: input.maxViews,
                        minBookmarks: input.minBookmarks,
                        minScore: input.projectMinScore,
                        minViews: input.minViews,
                        organizationId: input.projectOrganizationId,
                        parentId: input.projectParentId,
                        reportId: input.reportId,
                        resourceTypes: input.resourceTypes,
                        searchString: input.searchString,
                        sortBy: input.sortBy as unknown as ProjectSortBy,
                        tags: input.tags,
                        take,
                        updatedTimeFrame: input.updatedTimeFrame,
                        userId: getUser(req)?.id,
                        visibility: input.visibility,
                    },
                    objectType: "Project",
                });
            }
            // Query organizations
            let organizations;
            if (input.objectType === undefined || input.objectType === "Organization") {
                organizations = await readManyAsFeedHelper({
                    ...commonReadParams,
                    info: (partial as any).Organization,
                    input: {
                        after: input.organizationAfter,
                        createdTimeFrame: input.createdTimeFrame,
                        excludeIds: input.excludeIds,
                        ids: input.ids,
                        languages: input.translationLanguagesLatestVersion,
                        maxBookmarks: input.maxBookmarks,
                        minBookmarks: input.minBookmarks,
                        maxViews: input.maxViews,
                        minViews: input.minViews,
                        isOpenToNewMembers: input.organizationIsOpenToNewMembers,
                        projectId: input.organizationProjectId,
                        routineId: input.organizationRoutineId,
                        reportId: input.reportId,
                        resourceTypes: input.resourceTypes,
                        searchString: input.searchString,
                        sortBy: input.sortBy as unknown as OrganizationSortBy,
                        tags: input.tags,
                        take,
                        updatedTimeFrame: input.updatedTimeFrame,
                        userId: getUser(req)?.id,
                        visibility: input.visibility,
                    },
                    objectType: "Organization",
                });
            }
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes({
                projects: projects?.nodes ?? [],
                organizations: organizations?.nodes ?? [],
            }, {
                projects: { type: "Project", ...(partial as any).Project },
                organizations: { type: "Organization", ...(partial as any).Organization },
            }, prisma, getUser(req));
            // Combine nodes, alternating between projects and organizations
            const nodes: ProjectOrOrganization[] = [];
            for (let i = 0; i < Math.max(withSupplemental.projects.length, withSupplemental.organizations.length); i++) {
                if (i < withSupplemental.projects.length) {
                    nodes.push(withSupplemental.projects[i]);
                }
                if (i < withSupplemental.organizations.length) {
                    nodes.push(withSupplemental.organizations[i]);
                }
            }
            // Combine pageInfo
            const combined: ProjectOrOrganizationSearchResult = {
                __typename: "ProjectOrOrganizationSearchResult" as const,
                pageInfo: {
                    __typename: "ProjectOrOrganizationPageInfo" as const,
                    hasNextPage: projects?.pageInfo?.hasNextPage ?? organizations?.pageInfo?.hasNextPage ?? false,
                    endCursorProject: projects?.pageInfo?.endCursor ?? "",
                    endCursorOrganization: organizations?.pageInfo?.endCursor ?? "",
                },
                edges: nodes.map((node) => ({ __typename: "ProjectOrOrganizationEdge" as const, cursor: node.id, node })),
            };
            return combined;
        },
        runProjectOrRunRoutines: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 2000, req });
            const partial = toPartialGqlInfo(info, {
                __typename: "RunProjectOrRunRoutineSearchResult",
                RunProject: "RunProject",
                RunRoutine: "RunRoutine",
            }, req.languages, true);
            const take = Math.ceil((input.take ?? 10) / 2);
            const commonReadParams = { prisma, req };
            // Query run projects
            let runProjects;
            if (input.objectType === undefined || input.objectType === "RunProject") {
                runProjects = await readManyAsFeedHelper({
                    ...commonReadParams,
                    info: (partial as any).RunProject,
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
                        searchString: input.searchString,
                        sortBy: input.sortBy as unknown as RunProjectOrRunRoutineSortBy,
                        take,
                        updatedTimeFrame: input.updatedTimeFrame,
                        userId: getUser(req)?.id,
                        visibility: input.visibility,
                    },
                    objectType: "RunProject",
                });
            }
            // Query routines
            let runRoutines;
            if (input.objectType === undefined || input.objectType === "RunRoutine") {
                runRoutines = await readManyAsFeedHelper({
                    ...commonReadParams,
                    info: (partial as any).RunRoutine,
                    input: {
                        after: input.runRoutineAfter,
                        createdTimeFrame: input.createdTimeFrame,
                        startedTimeFrame: input.startedTimeFrame,
                        completedTimeFrame: input.completedTimeFrame,
                        excludeIds: input.excludeIds,
                        ids: input.ids,
                        status: input.status,
                        statuses: input.statuses,
                        routineVersionId: input.routineVersionId,
                        searchString: input.searchString,
                        sortBy: input.sortBy as unknown as RunProjectOrRunRoutineSortBy,
                        take,
                        updatedTimeFrame: input.updatedTimeFrame,
                        userId: getUser(req)?.id,
                        visibility: input.visibility,
                    },
                    objectType: "RunRoutine",
                });
            }
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes({
                runProjects: runProjects?.nodes ?? [],
                runRoutines: runRoutines?.nodes ?? [],
            }, {
                runProjects: { type: "RunProject", ...(partial as any).RunProject },
                runRoutines: { type: "RunRoutine", ...(partial as any).RunRoutine },
            }, prisma, getUser(req));
            // Combine nodes, alternating between runProjects and runRoutines
            const nodes: RunProjectOrRunRoutine[] = [];
            for (let i = 0; i < Math.max(withSupplemental.runProjects.length, withSupplemental.runRoutines.length); i++) {
                if (i < withSupplemental.runProjects.length) {
                    nodes.push(withSupplemental.runProjects[i]);
                }
                if (i < withSupplemental.runRoutines.length) {
                    nodes.push(withSupplemental.runRoutines[i]);
                }
            }
            // Combine pageInfo
            const combined: RunProjectOrRunRoutineSearchResult = {
                __typename: "RunProjectOrRunRoutineSearchResult" as const,
                pageInfo: {
                    __typename: "RunProjectOrRunRoutinePageInfo" as const,
                    hasNextPage: runProjects?.pageInfo?.hasNextPage ?? runRoutines?.pageInfo?.hasNextPage ?? false,
                    endCursorRunProject: runProjects?.pageInfo?.endCursor ?? "",
                    endCursorRunRoutine: runRoutines?.pageInfo?.endCursor ?? "",
                },
                edges: nodes.map((node) => ({ __typename: "RunProjectOrRunRoutineEdge" as const, cursor: node.id, node })),
            };
            return combined;
        },
    },
};

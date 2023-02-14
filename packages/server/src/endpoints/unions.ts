/**
 * Endpoints optimized for specific pages
 */
import { gql } from 'apollo-server-express';
import { OrganizationSortBy, ProjectSortBy, RoutineSortBy, ProjectOrRoutineSearchInput, ProjectOrRoutineSearchResult, ProjectOrOrganizationSearchInput, ProjectOrOrganizationSearchResult, ProjectOrRoutine, ProjectOrOrganization, ProjectOrRoutineSortBy, ProjectOrOrganizationSortBy, RunProjectOrRunRoutineSortBy, RunProjectOrRunRoutineSearchInput, RunProjectOrRunRoutine, RunProjectSortBy, RunProjectOrRunRoutineSearchResult } from '@shared/consts';
import { FindManyResult, GQLEndpoint, UnionResolver } from '../types';
import { rateLimit } from '../middleware';
import { resolveUnion } from './resolvers';
import { addSupplementalFieldsMultiTypes, toPartialGraphQLInfo } from '../builders';
import { PartialGraphQLInfo } from '../builders/types';
import { readManyAsFeedHelper } from '../actions';
import { getUser } from '../auth';

export const typeDef = gql`
    enum ProjectOrRoutineSortBy {
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        IssuesAsc
        IssuesDesc
        PullRequestsAsc
        PullRequestsDesc
        QuestionsAsc
        QuestionsDesc
        ScoreAsc
        ScoreDesc
        BookmarksAsc
        BookmarksDesc
        VersionsAsc
        VersionsDesc
        ViewsAsc
        ViewsDesc
    }

    enum ProjectOrOrganizationSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        BookmarksAsc
        BookmarksDesc
    }

    enum RunProjectOrRunRoutineSortBy {
        ContextSwitchesAsc
        ContextSwitchesDesc
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateStartedAsc
        DateStartedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StepsAsc
        StepsDesc
    }

    union ProjectOrRoutine = Project | Routine
    union ProjectOrOrganization = Project | Organization
    union RunProjectOrRunRoutine = RunProject | RunRoutine

    input ProjectOrRoutineSearchInput {
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        isComplete: Boolean
        isCompleteExceptions: [SearchException!]
        languages: [String!]
        maxBookmarks: Int
        maxScore: Int
        minBookmarks: Int
        minScore: Int
        minViews: Int
        objectType: String
        organizationId: ID
        parentId: ID
        projectAfter: String
        reportId: ID
        resourceTypes: [ResourceUsedFor!]
        routineAfter: String
        routineIsInternal: Boolean
        routineMinComplexity: Int
        routineMaxComplexity: Int
        routineMinSimplicity: Int
        routineMaxSimplicity: Int
        routineMaxTimesCompleted: Int
        routineMinTimesCompleted: Int
        routineProjectId: ID
        searchString: String
        sortBy: ProjectOrRoutineSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }
    type ProjectOrRoutineSearchResult {
        pageInfo: ProjectOrRoutinePageInfo!
        edges: [ProjectOrRoutineEdge!]!
    }
    type ProjectOrRoutinePageInfo {
        hasNextPage: Boolean!
        endCursorProject: String
        endCursorRoutine: String
    }
    type ProjectOrRoutineEdge {
        cursor: String!
        node: ProjectOrRoutine!
    }

    input ProjectOrOrganizationSearchInput {
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        languages: [String!]
        maxBookmarks: Int
        maxViews: Int
        minBookmarks: Int
        minViews: Int
        objectType: String
        organizationAfter: String
        organizationIsOpenToNewMembers: Boolean
        organizationProjectId: ID
        organizationRoutineId: ID
        projectAfter: String
        projectIsComplete: Boolean
        projectIsCompleteExceptions: [SearchException!]
        projectMaxScore: Int
        projectMinScore: Int
        projectOrganizationId: ID
        projectParentId: ID
        reportId: ID
        resourceTypes: [ResourceUsedFor!]
        searchString: String
        sortBy: ProjectOrOrganizationSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }
    type ProjectOrOrganizationSearchResult {
        pageInfo: ProjectOrOrganizationPageInfo!
        edges: [ProjectOrOrganizationEdge!]!
    }
    type ProjectOrOrganizationPageInfo {
        hasNextPage: Boolean!
        endCursorProject: String
        endCursorOrganization: String
    }
    type ProjectOrOrganizationEdge {
        cursor: String!
        node: ProjectOrOrganization!
    }

    input RunProjectOrRunRoutineSearchInput {
        createdTimeFrame: TimeFrame
        startedTimeFrame: TimeFrame
        completedTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        status: RunStatus
        objectType: String
        projectVersionId: ID
        routineVersionId: ID
        runProjectAfter: String
        runRoutineAfter: String
        searchString: String
        sortBy: RunProjectOrRunRoutineSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }
    type RunProjectOrRunRoutineSearchResult {
        pageInfo: RunProjectOrRunRoutinePageInfo!
        edges: [RunProjectOrRunRoutineEdge!]!
    }
    type RunProjectOrRunRoutinePageInfo {
        hasNextPage: Boolean!
        endCursorProject: String
        endCursorRoutine: String
    }
    type RunProjectOrRunRoutineEdge {
        cursor: String!
        node: RunProjectOrRunRoutine!
    }

    type Query {
        projectOrRoutines(input: ProjectOrRoutineSearchInput!): ProjectOrRoutineSearchResult!
        projectOrOrganizations(input: ProjectOrOrganizationSearchInput!): ProjectOrOrganizationSearchResult!
        runProjectOrRunRoutines(input: RunProjectOrRunRoutineSearchInput!): RunProjectOrRunRoutineSearchResult!
    }
`

export const resolvers: {
    ProjectOrRoutineSortBy: typeof ProjectOrRoutineSortBy,
    ProjectOrOrganizationSortBy: typeof ProjectOrOrganizationSortBy,
    RunProjectOrRunRoutineSortBy: typeof RunProjectOrRunRoutineSortBy,
    ProjectOrRoutine: UnionResolver;
    ProjectOrOrganization: UnionResolver;
    Query: {
        projectOrRoutines: GQLEndpoint<ProjectOrRoutineSearchInput, FindManyResult<ProjectOrRoutine>>;
        projectOrOrganizations: GQLEndpoint<ProjectOrOrganizationSearchInput, FindManyResult<ProjectOrOrganization>>;
        runProjectOrRunRoutines: GQLEndpoint<RunProjectOrRunRoutineSearchInput, FindManyResult<RunProjectOrRunRoutine>>;
    },
} = {
    ProjectOrRoutineSortBy,
    ProjectOrOrganizationSortBy,
    RunProjectOrRunRoutineSortBy,
    ProjectOrRoutine: { __resolveType(obj: any) { return resolveUnion(obj) } },
    ProjectOrOrganization: { __resolveType(obj: any) { return resolveUnion(obj) } },
    Query: {
        projectOrRoutines: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 2000, req });
            const partial = toPartialGraphQLInfo(info, {
                __typename: 'ProjectOrRoutineSearchResult',
                Project: 'Project',
                Routine: 'Routine',
            }, req.languages, true);
            const take = Math.ceil((input.take ?? 10) / 2);
            const commonReadParams = { prisma, req }
            // Query projects
            let projects;
            if (input.objectType === undefined || input.objectType === 'Project') {
                projects = await readManyAsFeedHelper({
                    ...commonReadParams,
                    info: (partial as any).Project,
                    input: {
                        after: input.projectAfter,
                        createdTimeFrame: input.createdTimeFrame,
                        excludeIds: input.excludeIds,
                        ids: input.ids,
                        isComplete: input.isComplete,
                        isCompleteExceptions: input.isCompleteExceptions,
                        languages: input.languages,
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
                    objectType: 'Project',
                });
            }
            // Query routines
            let routines;
            if (input.objectType === undefined || input.objectType === 'Routine') {
                routines = await readManyAsFeedHelper({
                    ...commonReadParams,
                    info: (partial as any).Routine,
                    input: {
                        after: input.routineAfter,
                        createdTimeFrame: input.createdTimeFrame,
                        excludeIds: input.excludeIds,
                        ids: input.ids,
                        isInternal: false,
                        isComplete: input.isComplete,
                        isCompleteExceptions: input.isCompleteExceptions,
                        languages: input.languages,
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
                    objectType: 'Routine',
                });
            }
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [projects?.nodes ?? [], routines?.nodes ?? []],
                [{ type: 'Project', ...(partial as any).Project }, { type: 'Routine', ...(partial as any).Routine }] as PartialGraphQLInfo[],
                ['p', 'r'],
                getUser(req),
                prisma,
            )
            // Combine nodes, alternating between projects and routines
            const nodes: ProjectOrRoutine[] = [];
            for (let i = 0; i < Math.max(withSupplemental['p'].length, withSupplemental['r'].length); i++) {
                if (i < withSupplemental['p'].length) {
                    nodes.push(withSupplemental['p'][i]);
                }
                if (i < withSupplemental['r'].length) {
                    nodes.push(withSupplemental['r'][i]);
                }
            }
            // Combine pageInfo
            const combined: ProjectOrRoutineSearchResult = {
                __typename: 'ProjectOrRoutineSearchResult' as const,
                pageInfo: {
                    __typename: 'ProjectOrRoutinePageInfo' as const,
                    hasNextPage: projects?.pageInfo?.hasNextPage ?? routines?.pageInfo?.hasNextPage ?? false,
                    endCursorProject: projects?.pageInfo?.endCursor ?? '',
                    endCursorRoutine: routines?.pageInfo?.endCursor ?? '',
                },
                edges: nodes.map((node) => ({ __typename: 'ProjectOrRoutineEdge' as const, cursor: node.id, node })),
            }
            return combined;
        },
        projectOrOrganizations: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 2000, req });
            const partial = toPartialGraphQLInfo(info, {
                __typename: 'ProjectOrOrganizationSearchResult',
                Project: 'Project',
                Organization: 'Organization',
            }, req.languages, true);
            const take = Math.ceil((input.take ?? 10) / 2);
            const commonReadParams = { prisma, req }
            // Query projects
            let projects;
            if (input.objectType === undefined || input.objectType === 'Project') {
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
                        languages: input.languages,
                        maxBookmarks: input.maxBookmarks,
                        maxScore: input.maxScore,
                        minBookmarks: input.minBookmarks,
                        maxScore: input.projectMaxScore,
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
                    objectType: 'Project',
                });
            }
            // Query organizations
            let organizations;
            if (input.objectType === undefined || input.objectType === 'Organization') {
                organizations = await readManyAsFeedHelper({
                    ...commonReadParams,
                    info: (partial as any).Organization,
                    input: {
                        after: input.organizationAfter,
                        createdTimeFrame: input.createdTimeFrame,
                        excludeIds: input.excludeIds,
                        ids: input.ids,
                        languages: input.languages,
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
                    objectType: 'Organization',
                });
            }
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [projects?.nodes ?? [], organizations?.nodes ?? []],
                [{ __typename: 'Project', ...(partial as any).Project }, { __typename: 'Organization', ...(partial as any).Organization }] as PartialGraphQLInfo[],
                ['p', 'o'],
                getUser(req),
                prisma,
            )
            // Combine nodes, alternating between projects and organizations
            const nodes: ProjectOrOrganization[] = [];
            for (let i = 0; i < Math.max(withSupplemental['p'].length, withSupplemental['o'].length); i++) {
                if (i < withSupplemental['p'].length) {
                    nodes.push(withSupplemental['p'][i]);
                }
                if (i < withSupplemental['o'].length) {
                    nodes.push(withSupplemental['o'][i]);
                }
            }
            // Combine pageInfo
            const combined: ProjectOrOrganizationSearchResult = {
                __typename: 'ProjectOrOrganizationSearchResult' as const,
                pageInfo: {
                    __typename: 'ProjectOrOrganizationPageInfo' as const,
                    hasNextPage: projects?.pageInfo?.hasNextPage ?? organizations?.pageInfo?.hasNextPage ?? false,
                    endCursorProject: projects?.pageInfo?.endCursor ?? '',
                    endCursorOrganization: organizations?.pageInfo?.endCursor ?? '',
                },
                edges: nodes.map((node) => ({ __typename: 'ProjectOrOrganizationEdge' as const, cursor: node.id, node })),
            }
            return combined;
        },
        runProjectOrRunRoutines: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 2000, req });
            const partial = toPartialGraphQLInfo(info, {
                __typename: 'RunProjectOrRunRoutineSearchResult',
                RunProject: 'RunProject',
                RunRoutine: 'RunRoutine',
            }, req.languages, true);
            const take = Math.ceil((input.take ?? 10) / 2);
            const commonReadParams = { prisma, req }
            // Query run projects
            let runProjects;
            if (input.objectType === undefined || input.objectType === 'RunProject') {
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
                        projectVersionId: input.projectVersionId,
                        searchString: input.searchString,
                        sortBy: input.sortBy as unknown as RunProjectOrRunRoutineSortBy,
                        take: take,
                        updatedTimeFrame: input.updatedTimeFrame,
                        userId: getUser(req)?.id,
                        visibility: input.visibility,
                    },
                    objectType: 'RunProject',
                });
            }
            // Query routines
            let runRoutines;
            if (input.objectType === undefined || input.objectType === 'RunRoutine') {
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
                        routineVersionId: input.routineVersionId,
                        searchString: input.searchString,
                        sortBy: input.sortBy as unknown as RunProjectOrRunRoutineSortBy,
                        take: take,
                        updatedTimeFrame: input.updatedTimeFrame,
                        userId: getUser(req)?.id,
                        visibility: input.visibility,
                    },
                    objectType: 'RunRoutine',
                });
            }
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [runProjects?.nodes ?? [], runRoutines?.nodes ?? []],
                [{ type: 'RunProject', ...(partial as any).RunProject }, { type: 'RunRoutine', ...(partial as any).RunRoutine }] as PartialGraphQLInfo[],
                ['p', 'r'],
                getUser(req),
                prisma,
            )
            // Combine nodes, alternating between runProjects and runRoutines
            const nodes: RunProjectOrRunRoutine[] = [];
            for (let i = 0; i < Math.max(withSupplemental['p'].length, withSupplemental['r'].length); i++) {
                if (i < withSupplemental['p'].length) {
                    nodes.push(withSupplemental['p'][i]);
                }
                if (i < withSupplemental['r'].length) {
                    nodes.push(withSupplemental['r'][i]);
                }
            }
            // Combine pageInfo
            const combined: RunProjectOrRunRoutineSearchResult = {
                __typename: 'RunProjectOrRunRoutineSearchResult' as const,
                pageInfo: {
                    __typename: 'RunProjectOrRunRoutinePageInfo' as const,
                    hasNextPage: runProjects?.pageInfo?.hasNextPage ?? runRoutines?.pageInfo?.hasNextPage ?? false,
                    endCursorProject: runProjects?.pageInfo?.endCursor ?? '',
                    endCursorRoutine: runRoutines?.pageInfo?.endCursor ?? '',
                },
                edges: nodes.map((node) => ({ __typename: 'RunProjectOrRunRoutineEdge' as const, cursor: node.id, node })),
            }
            return combined;
        },
    },
}
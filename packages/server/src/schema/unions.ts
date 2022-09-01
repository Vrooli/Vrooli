/**
 * Endpoints optimized for specific pages
 */
import { gql } from 'apollo-server-express';
import { GraphQLResolveInfo } from "graphql";
import { HomePageInput, HomePageResult, DevelopPageResult, LearnPageResult, OrganizationSortBy, ProjectSortBy, ResearchPageResult, ResourceUsedFor, RoutineSortBy, StandardSortBy, UserSortBy, HistoryPageInput, HistoryPageResult, StatisticsPageInput, StatisticsPageResult, Project, Routine, RunStatus, RunSortBy, ViewSortBy, ProjectOrRoutineSearchInput, ProjectOrRoutineSearchResult, ProjectOrOrganizationSearchInput, ProjectOrOrganizationSearchResult } from './types';
import { CODE } from '@shared/consts';
import { IWrap } from '../types';
import { Context } from '../context';
import { addSupplementalFieldsMultiTypes, modelToGraphQL, OrganizationModel, PartialGraphQLInfo, ProjectModel, readManyAsFeed, readManyHelper, RoutineModel, RunModel, StandardModel, StarModel, toPartialGraphQLInfo, UserModel, ViewModel } from '../models';
import { CustomError } from '../error';
import { rateLimit } from '../rateLimit';
import { resolveProjectOrOrganization, resolveProjectOrOrganizationOrRoutineOrStandardOrUser, resolveProjectOrRoutine } from './resolvers';
import { genErrorCode } from '../logger';

export const typeDef = gql`
    enum ProjectOrRoutineSortBy {
        CommentsAsc
        CommentsDesc
        ForksAsc
        ForksDesc
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
        VotesAsc
        VotesDesc
    }

    enum ProjectOrOrganizationSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
    }

    union ProjectOrRoutine = Project | Routine
    union ProjectOrOrganization = Project | Organization
    union ProjectOrOrganizationOrRoutineOrStandardOrUser = Project | Organization | Routine | Standard | User

    input ProjectOrRoutineSearchInput {
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        includePrivate: Boolean
        isComplete: Boolean
        isCompleteExceptions: [BooleanSearchException!]
        languages: [String!]
        minScore: Int
        minStars: Int
        minViews: Int
        objectType: String
        organizationId: ID
        parentId: ID
        projectAfter: String
        reportId: ID
        resourceLists: [String!]
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
        includePrivate: Boolean
        languages: [String!]
        minStars: Int
        minViews: Int
        objectType: String
        organizationAfter: String
        organizationIsOpenToNewMembers: Boolean
        organizationProjectId: ID
        organizationRoutineId: ID
        projectAfter: String
        projectIsComplete: Boolean
        projectIsCompleteExceptions: [BooleanSearchException!]
        projectMinScore: Int
        projectOrganizationId: ID
        projectParentId: ID
        reportId: ID
        resourceLists: [String!]
        resourceTypes: [ResourceUsedFor!]
        searchString: String
        sortBy: ProjectOrOrganizationSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
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

    type Query {
        projectOrRoutines(input: ProjectOrRoutineSearchInput!): ProjectOrRoutineSearchResult!
        projectOrOrganizations(input: ProjectOrOrganizationSearchInput!): ProjectOrOrganizationSearchResult!
    }
`

export const resolvers = {
    // ProjectOrRoutineSortBy: ProjectOrRoutineSortBy,
    // ProjectOrOrganizationSortBy: ProjectOrOrganizationSortBy,
    ProjectOrRoutine: {
        __resolveType(obj: any) { return resolveProjectOrRoutine(obj) }
    },
    ProjectOrOrganization: {
        __resolveType(obj: any) { return resolveProjectOrOrganization(obj) }
    },
    ProjectOrOrganizationOrRoutineOrStandardOrUser: {
        __resolveType(obj: any) { return resolveProjectOrOrganizationOrRoutineOrStandardOrUser(obj) }
    },
    Query: {
        projectOrRoutines: async (_parent: undefined, { input }: IWrap<ProjectOrRoutineSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<ProjectOrRoutineSearchResult> => {
            await rateLimit({ info, max: 2000, req });
            const partial = toPartialGraphQLInfo(info, {
                '__typename': 'ProjectOrRoutineSearchResult',
                'Project': 'Project',
                'Routine': 'Routine',
            }) as PartialGraphQLInfo;
            const userId = req.userId;
            if (!userId) throw new CustomError(CODE.NotAuthorized, { code: genErrorCode('0254') });
            const take = Math.ceil((input.take ?? 10) / 2);
            const commonReadParams = { prisma, userId }
            // Query projects
            let projects;
            if (input.objectType === undefined || input.objectType === 'Project') {
                projects = await readManyAsFeed({
                    ...commonReadParams,
                    additionalQueries: input.includePrivate ?
                        ProjectModel.permissions(prisma).ownershipQuery(userId) :
                        { isPrivate: false },
                    info: (partial as any).Project,
                    input: {
                        after: input.projectAfter,
                        createdTimeFrame: input.createdTimeFrame,
                        excludeIds: input.excludeIds,
                        ids: input.ids,
                        includePrivate: input.includePrivate,
                        isComplete: input.isComplete,
                        isCompleteExceptions: input.isCompleteExceptions,
                        languages: input.languages,
                        minScore: input.minScore,
                        minStars: input.minStars,
                        minViews: input.minViews,
                        organizationId: input.organizationId,
                        parentId: input.parentId,
                        reportId: input.reportId,
                        resourceLists: input.resourceLists,
                        resourceTypes: input.resourceTypes,
                        searchString: input.searchString,
                        sortBy: input.sortBy as unknown as ProjectSortBy,
                        tags: input.tags,
                        take,
                        updatedTimeFrame: input.updatedTimeFrame,
                        userId,
                    },
                    model: ProjectModel,
                });
            }
            // Query routines
            let routines;
            if (input.objectType === undefined || input.objectType === 'Routine') {
                routines = await readManyAsFeed({
                    ...commonReadParams,
                    additionalQueries: input.includePrivate ?
                        RoutineModel.permissions(prisma).ownershipQuery(userId) :
                        { isPrivate: false },
                    info: (partial as any).Routine,
                    input: {
                        after: input.routineAfter,
                        createdTimeFrame: input.createdTimeFrame,
                        excludeIds: input.excludeIds,
                        ids: input.ids,
                        includePrivate: input.includePrivate,
                        isInternal: false,
                        isComplete: input.isComplete,
                        isCompleteExceptions: input.isCompleteExceptions,
                        languages: input.languages,
                        minComplexity: input.routineMinComplexity,
                        maxComplexity: input.routineMaxComplexity,
                        minScore: input.minScore,
                        minSimplicity: input.routineMinSimplicity,
                        maxSimplicity: input.routineMaxSimplicity,
                        minStars: input.minStars,
                        minTimesCompleted: input.routineMinTimesCompleted,
                        maxTimesCompleted: input.routineMaxTimesCompleted,
                        minViews: input.minViews,
                        organizationId: input.organizationId,
                        parentId: input.parentId,
                        projectId: input.routineProjectId,
                        reportId: input.reportId,
                        resourceLists: input.resourceLists,
                        resourceTypes: input.resourceTypes,
                        searchString: input.searchString,
                        sortBy: input.sortBy as unknown as RoutineSortBy,
                        tags: input.tags,
                        take,
                        updatedTimeFrame: input.updatedTimeFrame,
                        userId,
                    },
                    model: RoutineModel,
                });
            }
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [projects?.nodes ?? [], routines?.nodes ?? []],
                [{ __typename: 'Project', ...(partial as any).Project }, { __typename: 'Routine', ...(partial as any).Routine }] as PartialGraphQLInfo[],
                ['p', 'r'],
                userId,
                prisma,
            )
            // Combine nodes, alternating between projects and routines
            const nodes = [];
            for (let i = 0; i < Math.max(withSupplemental['p'].length, withSupplemental['r'].length); i++) {
                if (i < withSupplemental['p'].length) {
                    nodes.push(withSupplemental['p'][i]);
                }
                if (i < withSupplemental['r'].length) {
                    nodes.push(withSupplemental['r'][i]);
                }
            }
            // Combine pageInfo
            const combined = {
                pageInfo: {
                    hasNextPage: projects?.pageInfo?.hasNextPage ?? routines?.pageInfo?.hasNextPage ?? false,
                    endCursorProject: projects?.pageInfo?.endCursor ?? '',
                    endCursorRoutine: routines?.pageInfo?.endCursor ?? '',
                },
                edges: nodes.map((node) => ({ cursor: node.id, node })),
            }
            return combined;
        },
        projectOrOrganizations: async (_parent: undefined, { input }: IWrap<ProjectOrOrganizationSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<ProjectOrOrganizationSearchResult> => {
            await rateLimit({ info, max: 2000, req });
            const partial = toPartialGraphQLInfo(info, {
                '__typename': 'ProjectOrOrganizationSearchResult',
                'Project': 'Project',
                'Organization': 'Organization',
            }) as PartialGraphQLInfo;
            const userId = req.userId;
            if (!userId) throw new CustomError(CODE.NotAuthorized, { code: genErrorCode('0255') });
            const take = Math.ceil((input.take ?? 10) / 2);
            const commonReadParams = { prisma, userId }
            // Query projects
            let projects;
            if (input.objectType === undefined || input.objectType === 'Project') {
                projects = await readManyAsFeed({
                    ...commonReadParams,
                    additionalQueries: input.includePrivate ?
                        ProjectModel.permissions(prisma).ownershipQuery(userId) :
                        { isPrivate: false },
                    info: (partial as any).Project,
                    input: {
                        after: input.projectAfter,
                        createdTimeFrame: input.createdTimeFrame,
                        excludeIds: input.excludeIds,
                        ids: input.ids,
                        includePrivate: input.includePrivate,
                        isComplete: input.projectIsComplete,
                        isCompleteExceptions: input.projectIsCompleteExceptions,
                        languages: input.languages,
                        minScore: input.projectMinScore,
                        minStars: input.minStars,
                        minViews: input.minViews,
                        organizationId: input.projectOrganizationId,
                        parentId: input.projectParentId,
                        reportId: input.reportId,
                        resourceLists: input.resourceLists,
                        resourceTypes: input.resourceTypes,
                        searchString: input.searchString,
                        sortBy: input.sortBy as unknown as ProjectSortBy,
                        tags: input.tags,
                        take,
                        updatedTimeFrame: input.updatedTimeFrame,
                        userId,
                    },
                    model: ProjectModel,
                });
            }
            // Query organizations
            let organizations;
            if (input.objectType === undefined || input.objectType === 'Organization') {
                organizations = await readManyAsFeed({
                    ...commonReadParams,
                    additionalQueries: input.includePrivate ?
                        OrganizationModel.permissions(prisma).ownershipQuery(userId) :
                        { isPrivate: false },
                    info: (partial as any).Organization,
                    input: {
                        after: input.organizationAfter,
                        createdTimeFrame: input.createdTimeFrame,
                        excludeIds: input.excludeIds,
                        ids: input.ids,
                        includePrivate: input.includePrivate,
                        languages: input.languages,
                        minStars: input.minStars,
                        minViews: input.minViews,
                        isOpenToNewMembers: input.organizationIsOpenToNewMembers,
                        projectId: input.organizationProjectId,
                        routineId: input.organizationRoutineId,
                        reportId: input.reportId,
                        resourceLists: input.resourceLists,
                        resourceTypes: input.resourceTypes,
                        searchString: input.searchString,
                        sortBy: input.sortBy as unknown as OrganizationSortBy,
                        tags: input.tags,
                        take,
                        updatedTimeFrame: input.updatedTimeFrame,
                        userId,
                    },
                    model: OrganizationModel,
                });
            }
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [projects?.nodes ?? [], organizations?.nodes ?? []],
                [{ __typename: 'Project', ...(partial as any).Project }, { __typename: 'Organization', ...(partial as any).Organization }] as PartialGraphQLInfo[],
                ['p', 'o'],
                userId,
                prisma,
            )
            // Combine nodes, alternating between projects and organizations
            const nodes = [];
            for (let i = 0; i < Math.max(withSupplemental['p'].length, withSupplemental['o'].length); i++) {
                if (i < withSupplemental['p'].length) {
                    nodes.push(withSupplemental['p'][i]);
                }
                if (i < withSupplemental['o'].length) {
                    nodes.push(withSupplemental['o'][i]);
                }
            }
            // Combine pageInfo
            const combined = {
                pageInfo: {
                    hasNextPage: projects?.pageInfo?.hasNextPage ?? organizations?.pageInfo?.hasNextPage ?? false,
                    endCursorProject: projects?.pageInfo?.endCursor ?? '',
                    endCursorOrganization: organizations?.pageInfo?.endCursor ?? '',
                },
                edges: nodes.map((node) => ({ cursor: node.id, node })),
            }
            return combined;
        },
    },
}
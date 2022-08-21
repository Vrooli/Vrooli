/**
 * Endpoints optimized for specific pages
 */
import { gql } from 'apollo-server-express';
import { GraphQLResolveInfo } from "graphql";
import { HomePageInput, HomePageResult, DevelopPageResult, LearnPageResult, OrganizationSortBy, ProjectSortBy, ResearchPageResult, ResourceUsedFor, RoutineSortBy, StandardSortBy, UserSortBy, HistoryPageInput, HistoryPageResult, StatisticsPageInput, StatisticsPageResult, Project, Routine, RunStatus, RunSortBy, ViewSortBy } from './types';
import { CODE } from '@shared/consts';
import { IWrap } from '../types';
import { Context } from '../context';
import { addSupplementalFieldsMultiTypes, OrganizationModel, PartialGraphQLInfo, ProjectModel, readManyAsFeed, RoutineModel, RunModel, StandardModel, StarModel, toPartialGraphQLInfo, UserModel, ViewModel } from '../models';
import { CustomError } from '../error';
import { rateLimit } from '../rateLimit';
import { resolveProjectOrOrganization, resolveProjectOrOrganizationOrRoutineOrStandardOrUser, resolveProjectOrRoutine } from './resolvers';

export const typeDef = gql`
 
    union ProjectOrRoutine = Project | Routine
    union ProjectOrOrganization = Project | Organization
    union ProjectOrOrganizationOrRoutineOrStandardOrUser = Project | Organization | Routine | Standard | User

    input ProjectOrRoutineSearchInput {
        after: String
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
        organizationId: ID
        parentId: ID
        reportId: ID
        resourceLists: [String!]
        resourceTypes: [ResourceUsedFor!]
        routineIsInternal: Boolean
        routineIsInternalExceptions: [BooleanSearchException!]
        routineMinComplexity: Int
        routineMaxComplexity: Int
        routineMinSimplicity: Int
        routineMaxSimplicity: Int
        routineMaxTimesCompleted: Int
        routineMinTimesCompleted: Int
        routineProjectId: ID
        searchString: String
        sortBy: RoutineSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
    }

    type ProjectOrRoutineSearchResult {
        pageInfo: PageInfo!
        edges: [ProjectOrRoutineEdge!]!
    }

    type ProjectOrRoutineEdge {
        cursor: String!
        node: ProjectOrRoutine!
    }

    input ProjectOrOrganizationSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        includePrivate: Boolean
        languages: [String!]
        minStars: Int
        minViews: Int
        organizationIsOpenToNewMembers: Boolean
        organizationProjectId: ID
        organizationRoutineId: ID
        projectIsComplete: Boolean
        projectIsCompleteExceptions: [BooleanSearchException!]
        projectMinScore: Int
        projectOrganizationId: ID
        projectParentId: ID
        reportId: ID
        resourceLists: [String!]
        resourceTypes: [ResourceUsedFor!]
        searchString: String
        sortBy: RoutineSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
    }

    type ProjectOrOrganizationSearchResult {
        pageInfo: PageInfo!
        edges: [ProjectOrOrganizationEdge!]!
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
        projectOrRoutines: async (_parent: undefined, { input }: IWrap<HomePageInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<HomePageResult> => {
            await rateLimit({ info, max: 2000, req });
            const partial = toPartialGraphQLInfo(info, {
                '__typename': 'HomePageResult',
                'organizations': 'Organization',
                'projects': 'Project',
                'routines': 'Routine',
                'standards': 'Standard',
                'users': 'User',
            }) as PartialGraphQLInfo;
            const userId = req.userId;
            const MinimumStars = 0; // Minimum stars required to show up in results. Will increase in the future.
            const starsQuery = { stars: { gte: MinimumStars } };
            const take = 5;
            const commonReadParams = {
                additionalQueries: { ...starsQuery },
                prisma,
                userId,
            }
            // Query organizations
            const organizations = await readManyAsFeed({
                ...commonReadParams,
                info: partial.organizations as PartialGraphQLInfo,
                input: { ...input, take, sortBy: OrganizationSortBy.StarsDesc },
                model: OrganizationModel,
            });
            // Query projects
            const projects = await readManyAsFeed({
                ...commonReadParams,
                info: partial.projects as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ProjectSortBy.StarsDesc, isComplete: true },
                model: ProjectModel,
            });
            // Query routines
            const routines = await readManyAsFeed({
                ...commonReadParams,
                info: partial.routines as PartialGraphQLInfo,
                input: { ...input, take, sortBy: RoutineSortBy.StarsDesc, isComplete: true, isInternal: false },
                model: RoutineModel,
            });
            // Query standards
            const standards = await readManyAsFeed({
                ...commonReadParams,
                info: partial.standards as PartialGraphQLInfo,
                input: { ...input, take, sortBy: StandardSortBy.StarsDesc, type: 'JSON' },
                model: StandardModel,
            });
            // Query users
            const users = await readManyAsFeed({
                ...commonReadParams,
                info: partial.users as PartialGraphQLInfo,
                input: { ...input, take, sortBy: UserSortBy.StarsDesc },
                model: UserModel,
            });
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [organizations, projects, routines, standards, users],
                [partial.organizations, partial.projects, partial.routines, partial.standards, partial.users] as PartialGraphQLInfo[],
                ['o', 'p', 'r', 's', 'u'],
                userId,
                prisma,
            )
            // Return results
            return {
                organizations: withSupplemental['o'],
                projects: withSupplemental['p'],
                routines: withSupplemental['r'],
                standards: withSupplemental['s'],
                users: withSupplemental['u'],
            }
        },
        projectOrOrganizations: async (_parent: undefined, _args: undefined, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<LearnPageResult> => {
            await rateLimit({ info, max: 2000, req });
            const partial = toPartialGraphQLInfo(info, {
                '__typename': 'LearnPageResult',
                'courses': 'Project',
                'tutorials': 'Routine',
            }) as PartialGraphQLInfo;
            const userId = req.userId;
            const MinimumStars = 0; // Minimum stars required to show up in autocomplete results. Will increase in the future.
            const starsQuery = { stars: { gte: MinimumStars } };
            const take = 5;
            const commonReadParams = {
                additionalQueries: { ...starsQuery },
                prisma,
                userId,
            }
            // Query courses
            const courses = await readManyAsFeed({
                ...commonReadParams,
                info: partial.courses as PartialGraphQLInfo,
                input: { take, sortBy: ProjectSortBy.VotesDesc, tags: ['Learn'], isComplete: true, },
                model: ProjectModel,
            });
            // Query tutorials
            const tutorials = await readManyAsFeed({
                ...commonReadParams,
                info: partial.tutorials as PartialGraphQLInfo,
                input: { take, sortBy: ProjectSortBy.VotesDesc, tags: ['Learn'], isComplete: true, },
                model: RoutineModel,
            });
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [courses, tutorials],
                [partial.courses, partial.tutorials] as PartialGraphQLInfo[],
                ['c', 't'],
                userId,
                prisma,
            )
            // Return data
            return {
                courses: withSupplemental['c'],
                tutorials: withSupplemental['t'],
            }
        },
    },
}
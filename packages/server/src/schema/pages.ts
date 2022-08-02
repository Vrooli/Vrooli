/**
 * Endpoints optimized for specific pages
 */
import { gql } from 'apollo-server-express';
import { GraphQLResolveInfo } from "graphql";
import { HomePageInput, HomePageResult, DevelopPageResult, LearnPageResult, OrganizationSortBy, ProjectSortBy, ResearchPageResult, ResourceUsedFor, RoutineSortBy, StandardSortBy, UserSortBy, HistoryPageInput, HistoryPageResult, StatisticsPageInput, StatisticsPageResult, Project, Routine, RunStatus, RunSortBy, ViewSortBy } from './types';
import { CODE } from '@local/shared';
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

    input HomePageInput {
        searchString: String!
        take: Int
    }
 
    type HomePageResult {
        organizations: [Organization!]!
        projects: [Project!]!
        routines: [Routine!]!
        standards: [Standard!]!
        users: [User!]!
    }

    input HistoryPageInput {
        searchString: String!
        take: Int
    }

    type HistoryPageResult {
        activeRuns: [Run!]!
        completedRuns: [Run!]!
        recentlyViewed: [View!]!
        recentlyStarred: [Star!]!
    }
 
    type LearnPageResult {
        courses: [Project!]!
        tutorials: [Routine!]!
    }
 
    type ResearchPageResult {
        processes: [Routine!]!
        newlyCompleted: [ProjectOrRoutine!]!
        needVotes: [Project!]!
        needInvestments: [Project!]!
        needMembers: [Organization!]!
    }
 
    type DevelopPageResult {
        completed: [ProjectOrRoutine!]!
        inProgress: [ProjectOrRoutine!]!
        recent: [ProjectOrRoutine!]!
    }

    input StatisticsPageInput {
        searchString: String!
        take: Int
    }

    type StatisticsTimeFrame {
        organizations: [Int!]!
        projects: [Int!]!
        routines: [Int!]!
        standards: [Int!]!
        users: [Int!]!
    }

    type StatisticsPageResult {
        daily: StatisticsTimeFrame!
        weekly: StatisticsTimeFrame!
        monthly: StatisticsTimeFrame!
        yearly: StatisticsTimeFrame!
        allTime: StatisticsTimeFrame!
    }
 
    type Query {
        homePage(input: HomePageInput!): HomePageResult!
        historyPage(input: HistoryPageInput!): HistoryPageResult!
        learnPage: LearnPageResult!
        researchPage: ResearchPageResult!
        developPage: DevelopPageResult!
        statisticsPage(input: StatisticsPageInput!): StatisticsPageResult!
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
        homePage: async (_parent: undefined, { input }: IWrap<HomePageInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<HomePageResult> => {
            await rateLimit({ info, max: 5000, req });
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
        /**
         * Queries data shown on Learn page
         */
        learnPage: async (_parent: undefined, _args: undefined, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<LearnPageResult> => {
            await rateLimit({ info, max: 5000, req });
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
        /**
         * Queries data shown on Research page
         */
        researchPage: async (_parent: undefined, _args: undefined, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<ResearchPageResult> => {
            await rateLimit({ info, max: 5000, req });
            const partial = toPartialGraphQLInfo(info, {
                '__typename': 'ResearchPageResult',
                'processes': 'Routine',
                'newlyCompleted': {
                    'Project': 'Project',
                    'Routine': 'Routine',
                },
                'needVotes': 'Project',
                'needInvestments': 'Project',
                'needMembers': 'Organization',
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
            // Query processes
            const processes = await readManyAsFeed({
                ...commonReadParams,
                info: partial.processes as PartialGraphQLInfo,
                input: { take, sortBy: RoutineSortBy.VotesDesc, tags: ['Research'], isComplete: true, isInternal: false },
                model: RoutineModel,
            });
            // Query newlyCompleted
            const newlyCompletedProjects = await readManyAsFeed({
                ...commonReadParams,
                info: (partial.newlyCompleted as PartialGraphQLInfo)?.Project as PartialGraphQLInfo,
                input: { take, sortBy: ProjectSortBy.DateCompletedAsc, isComplete: true },
                model: ProjectModel,
            });
            const newlyCompletedRoutines = await readManyAsFeed({
                ...commonReadParams,
                info: (partial.newlyCompleted as PartialGraphQLInfo)?.Routine as PartialGraphQLInfo,
                input: { take, isComplete: true, isInternal: false, sortBy: RoutineSortBy.DateCompletedAsc },
                model: RoutineModel,
            });
            // Query needVotes
            const needVotes = await readManyAsFeed({
                ...commonReadParams,
                info: partial.needVotes as PartialGraphQLInfo,
                input: { take, isComplete: false, resourceTypes: [ResourceUsedFor.Proposal] },
                model: ProjectModel,
            });
            // Query needInvestments
            const needInvestments = await readManyAsFeed({
                ...commonReadParams,
                info: partial.needInvestments as PartialGraphQLInfo,
                input: { take, isComplete: false, resourceTypes: [ResourceUsedFor.Donation] },
                model: ProjectModel,
            });
            // Query needMembers
            const needMembers = await readManyAsFeed({
                ...commonReadParams,
                info: partial.needMembers as PartialGraphQLInfo,
                input: { take, isOpenToNewMembers: true, sortBy: OrganizationSortBy.StarsDesc },
                model: OrganizationModel,
            });
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [processes, newlyCompletedProjects, newlyCompletedRoutines, needVotes, needInvestments, needMembers],
                [partial.processes, (partial.newlyCompleted as PartialGraphQLInfo)?.Project, (partial.newlyCompleted as PartialGraphQLInfo)?.Routine, partial.needVotes, partial.needInvestments, partial.needMembers] as PartialGraphQLInfo[],
                ['p', 'ncp', 'ncr', 'nv', 'ni', 'nm'],
                userId,
                prisma,
            )
            // Return data
            return {
                processes: withSupplemental['p'],
                // newlyCompleted combines projects and routines, and sorts by date completed
                newlyCompleted: [...withSupplemental['ncp'], ...withSupplemental['ncr']].sort((a, b) => {
                    if (a.completedAt < b.completedAt) return -1;
                    if (a.completedAt > b.completedAt) return 1;
                    return 0;
                }),
                needVotes: withSupplemental['nv'],
                // needInvestments combines projects and organizations, and sorts by stars
                needInvestments: withSupplemental['ni'].sort((a, b) => {
                    if (a.stars < b.stars) return 1;
                    if (a.stars > b.stars) return -1;
                    return 0;
                }),
                needMembers: withSupplemental['nm'],
            }
        },
        /**
         * Queries data shown on Develop page
         */
        developPage: async (_parent: undefined, _args: undefined, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<DevelopPageResult> => {
            await rateLimit({ info, max: 5000, req });
            const partial = toPartialGraphQLInfo(info, {
                '__typename': 'DevelopPageResult',
                'completed': {
                    'Project': 'Project',
                    'Routine': 'Routine',
                },
                'inProgress': {
                    'Project': 'Project',
                    'Routine': 'Routine',
                },
                'recent': {
                    'Project': 'Project',
                    'Routine': 'Routine',
                },
            }) as PartialGraphQLInfo;
            const userId = req.userId;
            // If not signed in, return empty data
            if (!userId) return {
                completed: [],
                inProgress: [],
                recent: [],
            }
            const take = 5;
            const commonReadParams = {
                additionalQueries: { },
                prisma,
                userId,
            }
            // Query for routines you've completed
            const completedRoutines = await readManyAsFeed({
                ...commonReadParams,
                info: (partial.completed as PartialGraphQLInfo)?.Routine as PartialGraphQLInfo,
                input: { take, isComplete: true, isInternal: false, userId, sortBy: RoutineSortBy.DateCompletedAsc },
                model: RoutineModel,
            });
            // Query for projects you've completed
            const completedProjects = await readManyAsFeed({
                ...commonReadParams,
                info: (partial.completed as PartialGraphQLInfo)?.Project as PartialGraphQLInfo,
                input: { take, isComplete: true, userId, sortBy: ProjectSortBy.DateCompletedAsc },
                model: ProjectModel,
            });
            // Query for routines you're currently working on
            const inProgressRoutines = await readManyAsFeed({
                ...commonReadParams,
                info: (partial.inProgress as PartialGraphQLInfo)?.Routine as PartialGraphQLInfo,
                input: { take, isComplete: false, isInternal: false, userId, sortBy: RoutineSortBy.DateCreatedAsc },
                model: RoutineModel,
            });
            // Query for projects you're currently working on
            const inProgressProjects = await readManyAsFeed({
                ...commonReadParams,
                info: (partial.inProgress as PartialGraphQLInfo)?.Project as PartialGraphQLInfo,
                input: { take, isComplete: false, userId, sortBy: ProjectSortBy.DateCreatedAsc },
                model: ProjectModel,
            });
            // Query recently created/updated routines
            const recentRoutines = await readManyAsFeed({
                ...commonReadParams,
                info: (partial.recent as PartialGraphQLInfo)?.Routine as PartialGraphQLInfo,
                input: { take, userId, sortBy: RoutineSortBy.DateUpdatedAsc, isInternal: false },
                model: RoutineModel,
            });
            // Query recently created/updated projects
            const recentProjects = await readManyAsFeed({
                ...commonReadParams,
                info: (partial.recent as PartialGraphQLInfo)?.Project as PartialGraphQLInfo,
                input: { take, userId, sortBy: ProjectSortBy.DateUpdatedAsc },
                model: ProjectModel,
            });
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [completedRoutines, completedProjects, inProgressRoutines, inProgressProjects, recentRoutines, recentProjects],
                [(partial.completed as PartialGraphQLInfo)?.Routine, (partial.completed as PartialGraphQLInfo)?.Project, (partial.inProgress as PartialGraphQLInfo)?.Routine, (partial.inProgress as PartialGraphQLInfo)?.Project, (partial.recent as PartialGraphQLInfo)?.Routine, (partial.recent as PartialGraphQLInfo)?.Project] as PartialGraphQLInfo[],
                ['cr', 'cp', 'ipr', 'ipp', 'rr', 'rp'],
                userId,
                prisma,
            )
            // Combine arrays
            const completed: Array<Project | Routine> = [...withSupplemental['cr'], ...withSupplemental['cp']];
            const inProgress: Array<Project | Routine> = [...withSupplemental['ipr'], ...withSupplemental['ipp']];
            const recent: Array<Project | Routine> = [...withSupplemental['rr'], ...withSupplemental['rp']];
            // Sort arrays
            completed.sort((a, b) => {
                if (a.completedAt < b.completedAt) return -1;
                if (a.completedAt > b.completedAt) return 1;
                return 0;
            });
            inProgress.sort((a, b) => {
                if (a.updated_at < b.updated_at) return -1;
                if (a.updated_at > b.updated_at) return 1;
                return 0;
            });
            recent.sort((a, b) => {
                if (a.updated_at < b.updated_at) return -1;
                if (a.updated_at > b.updated_at) return 1;
                return 0;
            });
            // Return data
            return {
                completed,
                inProgress,
                recent,
            }
        },
        /**
         * Queries data shown on History page 
         */
        historyPage: async (_parent: undefined, { input }: IWrap<HistoryPageInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<HistoryPageResult> => {
            // Only accessible if logged in and not using an API key
            if (!req.userId || req.apiToken) throw new CustomError(CODE.Unauthorized, 'Must be signed in to see this page');
            await rateLimit({ info, max: 5000, req });
            const partial = toPartialGraphQLInfo(info, {
                '__typename': 'HistoryPageResult',
                'activeRuns': 'Run',
                'completedRuns': 'Run',
                'recentlyViewed': 'View',
                'recentlyStarred': 'Star',
            }) as PartialGraphQLInfo;
            const userId = req.userId;
            const take = 5;
            const commonReadParams = {
                prisma,
                userId,
            }
            // Query for incomplete runs
            const activeRuns = await readManyAsFeed({
                ...commonReadParams,
                info: partial.activeRuns as PartialGraphQLInfo,
                input: { take, ...input, status: RunStatus.InProgress, sortBy: RunSortBy.DateUpdatedDesc },
                model: RunModel,
            });
            // Query for complete runs
            const completedRuns = await readManyAsFeed({
                ...commonReadParams,
                info: partial.completedRuns as PartialGraphQLInfo,
                input: { take, ...input, status: RunStatus.Completed, sortBy: RunSortBy.DateUpdatedDesc },
                model: RunModel,
            });
            // Query recently viewed objects (of any type)
            const recentlyViewed = await readManyAsFeed({
                ...commonReadParams,
                info: partial.recentlyViewed as PartialGraphQLInfo,
                input: { take, ...input, sortBy: ViewSortBy.LastViewedDesc },
                model: ViewModel,
            });
            // Query recently starred objects (of any type). Make sure to ignore tags
            const recentlyStarred = await readManyAsFeed({
                ...commonReadParams,
                info: partial.recentlyStarred as PartialGraphQLInfo,
                input: { take, ...input, sortBy: UserSortBy.DateCreatedDesc },
                model: StarModel,
            });
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [activeRuns, completedRuns, recentlyViewed, recentlyStarred],
                [partial.activeRuns, partial.completedRuns, partial.recentlyViewed, partial.recentlyStarred] as PartialGraphQLInfo[],
                ['ar', 'cr', 'rv', 'rs'],
                userId,
                prisma,
            )
            // Return results
            return {
                activeRuns: withSupplemental['ar'],
                completedRuns: withSupplemental['cr'],
                recentlyViewed: withSupplemental['rv'],
                recentlyStarred: withSupplemental['rs'],
            } as any
        },
        /**
         * Returns site-wide statistics
         */
        statisticsPage: async (_parent: undefined, { input }: IWrap<StatisticsPageInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<StatisticsPageResult> => {
            await rateLimit({ info, max: 500, req });
            // Query current stats
            // Read historical stats from file
            throw new CustomError(CODE.NotImplemented);
        },
    },
}
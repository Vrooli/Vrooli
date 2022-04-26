/**
 * Endpoints optimized for specific pages
 */
import { gql } from 'apollo-server-express';
import { GraphQLResolveInfo } from "graphql";
import { HomePageInput, HomePageResult, DevelopPageResult, LearnPageResult, LogSearchResult, LogSortBy, OrganizationSortBy, ProjectSortBy, ResearchPageResult, ResourceUsedFor, RoutineSortBy, StandardSortBy, UserSortBy, ForYouPageInput, ForYouPageResult, StatisticsPageInput, StatisticsPageResult, Project, Routine, Log, Organization, Standard, User } from './types';
import { CODE } from '@local/shared';
import { IWrap } from '../types';
import { Context } from '../context';
import { addSupplementalFieldsMultiTypes, GraphQLModelType, logSearcher, LogType, modelToGraphQL, OrganizationModel, paginatedMongoSearch, PartialInfo, ProjectModel, readManyHelper, RoutineModel, StandardModel, toPartialSelect, UserModel } from '../models';
import { CustomError } from '../error';
import { rateLimit } from '../rateLimit';

// Query fields shared across multiple endpoints
const tagSelect = {
    __typename: 'Tag',
    id: true,
    created_at: true,
    tag: true,
    stars: true,
    isStarred: true,
    translations: {
        id: true,
        language: true,
        description: true,
    }
}
const organizationSelect = {
    __typename: 'Organization',
    id: true,
    handle: true,
    stars: true,
    isStarred: true,
    translations: {
        id: true,
        language: true,
        name: true,
    },
    tags: tagSelect,
}
const projectSelect = {
    __typename: 'Project',
    id: true,
    handle: true,
    stars: true,
    score: true,
    isStarred: true,
    isUpvoted: true,
    translations: {
        id: true,
        language: true,
        name: true,
    },
    tags: tagSelect,
}
const routineSelect = {
    __typename: 'Routine',
    id: true,
    created_at: true,
    complexity: true,
    simplicity: true,
    stars: true,
    score: true,
    isComplete: true,
    isStarred: true,
    isUpvoted: true,
    translations: {
        id: true,
        language: true,
        title: true,
        instructions: true,
    },
    tags: tagSelect,
}
const standardSelect = {
    __typename: 'Standard',
    id: true,
    name: true,
    stars: true,
    score: true,
    isStarred: true,
    isUpvoted: true,
    translations: {
        id: true,
        language: true,
    },
    tags: tagSelect,
}
const userSelect = {
    __typename: 'User',
    id: true,
    name: true,
    handle: true,
    stars: true,
    isStarred: true,
}

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

    input ForYouPageInput {
        searchString: String!
        take: Int
    }

    type ForYouPageResult {
        activeRoutines: [Routine!]!
        completedRoutines: [Routine!]!
        recent: [ProjectOrOrganizationOrRoutineOrStandardOrUser!]!
        starred: [ProjectOrOrganizationOrRoutineOrStandardOrUser!]!
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
        forYouPage(input: ForYouPageInput!): ForYouPageResult!
        learnPage: LearnPageResult!
        researchPage: ResearchPageResult!
        developPage: DevelopPageResult!
        statisticsPage(input: StatisticsPageInput!): StatisticsPageResult!
    }
 `

export const resolvers = {
    ProjectOrRoutine: {
        __resolveType(obj: any) {
            // Only a project has a handle field
            if (obj.hasOwnProperty('handle')) return GraphQLModelType.Project;
            return GraphQLModelType.Routine;
        }
    },
    ProjectOrOrganization: {
        __resolveType(obj: any) {
            // Only a project has a score field
            if (obj.hasOwnProperty('score')) return GraphQLModelType.Project;
            return GraphQLModelType.Organization;
        }
    },
    ProjectOrOrganizationOrRoutineOrStandardOrUser: {
        __resolveType(obj: any) {
            // Only a routine has a complexity field
            if (obj.hasOwnProperty('complexity')) return GraphQLModelType.Routine;
            // Only a user has an untranslated name field
            if (obj.hasOwnProperty('name')) return GraphQLModelType.User;
            // Out of the remaining types, only an organization does not have isUpvoted field
            if (!obj.hasOwnProperty('isUpvoted')) return GraphQLModelType.Organization;
            // Out of the remaining types, only a project has a handle field
            if (obj.hasOwnProperty('handle')) return GraphQLModelType.Project;
            // There is only one remaining type, the standard
            return GraphQLModelType.Standard;
        }
    },
    Query: {
        homePage: async (_parent: undefined, { input }: IWrap<HomePageInput>, context: Context, info: GraphQLResolveInfo): Promise<HomePageResult> => {
            await rateLimit({ context, info, max: 5000 });
            const MinimumStars = 0; // Minimum stars required to show up in results. Will increase in the future.
            const starsQuery = { stars: { gte: MinimumStars } };
            const take = 5;
            // Initialize models
            const oModel = OrganizationModel(context.prisma);
            const pModel = ProjectModel(context.prisma);
            const rModel = RoutineModel(context.prisma);
            const sModel = StandardModel(context.prisma);
            const uModel = UserModel(context.prisma);
            // Query organizations
            let organizations = (await readManyHelper(
                context.req.userId,
                { ...input, take, sortBy: OrganizationSortBy.StarsDesc },
                organizationSelect,
                oModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(organizationSelect, oModel.relationshipMap) as PartialInfo)) as any[]
            // Query projects
            let projects = (await readManyHelper(
                context.req.userId,
                { ...input, take, sortBy: ProjectSortBy.StarsDesc },
                projectSelect,
                pModel,
                { ...starsQuery },
                false
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(projectSelect, pModel.relationshipMap) as PartialInfo)) as any[]
            // Query routines
            let routines = (await readManyHelper(
                context.req.userId,
                { ...input, take, sortBy: RoutineSortBy.StarsDesc },
                routineSelect,
                rModel,
                { ...starsQuery },
                false
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(routineSelect, rModel.relationshipMap) as PartialInfo)) as any[]
            // Query standards
            let standards = (await readManyHelper(
                context.req.userId,
                { ...input, take, sortBy: StandardSortBy.StarsDesc },
                standardSelect,
                sModel,
                { ...starsQuery },
                false
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(standardSelect, sModel.relationshipMap) as PartialInfo)) as any[]
            // Query users
            let users = (await readManyHelper(
                context.req.userId,
                { ...input, take, sortBy: UserSortBy.StarsDesc },
                userSelect,
                uModel,
                { ...starsQuery },
                false
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(userSelect, uModel.relationshipMap) as PartialInfo)) as any[]
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [organizations, projects, routines, standards, users],
                [organizationSelect, projectSelect, routineSelect, standardSelect, userSelect] as any,
                ['o', 'p', 'r', 's', 'u'],
                context.req.userId,
                context.prisma,
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
        learnPage: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<LearnPageResult> => {
            const MinimumStars = 0; // Minimum stars required to show up in autocomplete results. Will increase in the future.
            const starsQuery = { stars: { gte: MinimumStars } };
            const take = 5;
            // Initialize models
            const pModel = ProjectModel(context.prisma);
            const rModel = RoutineModel(context.prisma);
            // Query courses
            const courses = (await readManyHelper(
                context.req.userId,
                { take, sortBy: ProjectSortBy.VotesDesc, tags: ['learn'] },
                projectSelect,
                pModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(projectSelect, pModel.relationshipMap) as PartialInfo)) as any[]
            // Query tutorials
            const tutorials = (await readManyHelper(
                context.req.userId,
                { take, sortBy: RoutineSortBy.VotesDesc, tags: ['learn'] },
                routineSelect,
                rModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(routineSelect, rModel.relationshipMap) as PartialInfo)) as any[]
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [courses, tutorials],
                [projectSelect, routineSelect] as any,
                ['c', 't'],
                context.req.userId,
                context.prisma,
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
        researchPage: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<ResearchPageResult> => {
            const MinimumStars = 0; // Minimum stars required to show up in autocomplete results. Will increase in the future.
            const starsQuery = { stars: { gte: MinimumStars } };
            const take = 5;
            // Initialize models
            const oModel = OrganizationModel(context.prisma);
            const pModel = ProjectModel(context.prisma);
            const rModel = RoutineModel(context.prisma);
            // Query processes
            const processes = (await readManyHelper(
                context.req.userId,
                { take, sortBy: RoutineSortBy.VotesDesc, tags: ['research'] },
                routineSelect,
                rModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(routineSelect, rModel.relationshipMap) as PartialInfo)) as any[]
            // Query newlyCompleted
            const newlyCompletedProjects = (await readManyHelper(
                context.req.userId,
                { take, isComplete: true, sortBy: ProjectSortBy.DateCompletedAsc },
                projectSelect,
                pModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(projectSelect, pModel.relationshipMap) as PartialInfo)) as any[]
            const newlyCompletedRoutines = (await readManyHelper(
                context.req.userId,
                { take, isComplete: true, sortBy: RoutineSortBy.DateCompletedAsc },
                routineSelect,
                rModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(routineSelect, rModel.relationshipMap) as PartialInfo)) as any[]
            // Query needVotes
            const needVotes = (await readManyHelper(
                context.req.userId,
                { take, isComplete: false, resourceTypes: [ResourceUsedFor.Proposal] },
                projectSelect,
                pModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(projectSelect, pModel.relationshipMap) as PartialInfo)) as any[]
            // Query needInvestments
            const needInvestmentsProjects = (await readManyHelper(
                context.req.userId,
                { take, isComplete: false, resourceTypes: [ResourceUsedFor.Donation] },
                projectSelect,
                pModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(projectSelect, pModel.relationshipMap) as PartialInfo)) as any[]
            const needInvestmentsOrganizations = (await readManyHelper(
                context.req.userId,
                { take, resourceTypes: [ResourceUsedFor.Donation] },
                organizationSelect,
                oModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(organizationSelect, oModel.relationshipMap) as PartialInfo)) as any[]
            // Query needMembers
            const needMembers = (await readManyHelper(
                context.req.userId,
                { take, isOpenToNewMembers: true, sortBy: OrganizationSortBy.StarsDesc },
                organizationSelect,
                oModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(organizationSelect, oModel.relationshipMap) as PartialInfo)) as any[]
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [processes, newlyCompletedProjects, newlyCompletedRoutines, needVotes, needInvestmentsProjects, needInvestmentsOrganizations, needMembers],
                [routineSelect, projectSelect, routineSelect, projectSelect, projectSelect, organizationSelect] as any,
                ['p', 'ncp', 'ncr', 'nv', 'nip', 'nio', 'nm'],
                context.req.userId,
                context.prisma,
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
                needInvestments: [...withSupplemental['nip'], ...withSupplemental['nio']].sort((a, b) => {
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
        developPage: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<DevelopPageResult> => {
            // If not signed in, return empty data
            if (!context.req.userId) return {
                completed: [],
                inProgress: [],
                recent: [],
            }
            const MinimumStars = 0; // Minimum stars required to show up in autocomplete results. Will increase in the future.
            const starsQuery = { stars: { gte: MinimumStars } };
            const take = 5;
            // Initialize models
            const oModel = OrganizationModel(context.prisma);
            const pModel = ProjectModel(context.prisma);
            const rModel = RoutineModel(context.prisma);
            // Find completed logs for routines
            const completedLogs = (await paginatedMongoSearch<LogSearchResult>({
                findQuery: logSearcher().getFindQuery(context.req.userId, { actions: [LogType.RoutineComplete] }),
                sortQuery: logSearcher().getSortQuery(LogSortBy.DateCreatedAsc),
                take,
                project: logSearcher().defaultProjection,
            })).edges.map(({ node }: any) => node) as Log[]
            console.log('develop query completed logs', JSON.stringify(completedLogs));
            // Use logs to find full routine data from Prisma
            const completedIds = completedLogs.map((node: any) => node.object1Id);
            const completedRoutines = (await readManyHelper(
                context.req.userId,
                { ids: completedIds, sortBy: RoutineSortBy.DateCompletedAsc },
                routineSelect,
                rModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(routineSelect, rModel.relationshipMap) as PartialInfo)) as any[]
            // Projects can be found without looking up logs
            const completedProjects = (await readManyHelper(
                context.req.userId,
                { take, isComplete: true, userId: context.req.userId, sortBy: ProjectSortBy.DateCompletedAsc },
                projectSelect,
                pModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(projectSelect, pModel.relationshipMap) as PartialInfo)) as any[]
            // Find in progress logs
            const inProgressLogs = (await paginatedMongoSearch<LogSearchResult>({
                findQuery: logSearcher().getFindQuery(context.req.userId ?? '', { actions: [LogType.RoutineStartIncomplete] }),
                sortQuery: logSearcher().getSortQuery(LogSortBy.DateCreatedAsc),
                take,
                project: logSearcher().defaultProjection,
            })).edges.map(({ node }: any) => node) as Log[]
            console.log('develop query in progress logs', JSON.stringify(inProgressLogs));
            // Use logs to find full routine data from Prisma
            const inProgressIds = inProgressLogs.map((node: any) => node.object1Id);
            const inProgressRoutines = (await readManyHelper(
                context.req.userId,
                { ids: inProgressIds, sortBy: RoutineSortBy.DateCreatedAsc },
                routineSelect,
                rModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(routineSelect, rModel.relationshipMap) as PartialInfo)) as any[]
            // Projects can be found without looking up logs
            const inProgressProjects = (await readManyHelper(
                context.req.userId,
                { take, isComplete: false, userId: context.req.userId, sortBy: ProjectSortBy.DateCreatedAsc },
                projectSelect,
                pModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(projectSelect, pModel.relationshipMap) as PartialInfo)) as any[]
            // Query recently created/updated routines
            const recentRoutines = (await readManyHelper(
                context.req.userId,
                { take, userId: context.req.userId, sortBy: RoutineSortBy.DateUpdatedAsc },
                routineSelect,
                rModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(routineSelect, rModel.relationshipMap) as PartialInfo)) as any[]
            // Query recently created/updated projects
            const recentProjects = (await readManyHelper(
                context.req.userId,
                { take, userId: context.req.userId, sortBy: ProjectSortBy.DateUpdatedAsc },
                projectSelect,
                pModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(projectSelect, pModel.relationshipMap) as PartialInfo)) as any[]
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [completedRoutines, completedProjects, inProgressRoutines, inProgressProjects, recentRoutines, recentProjects],
                [routineSelect, projectSelect, routineSelect, projectSelect, routineSelect, projectSelect] as any,
                ['cr', 'cp', 'ipr', 'ipp', 'rr', 'rp'],
                context.req.userId,
                context.prisma,
            )
            // Combine arrays
            const completed: Array<Project | Routine> = [...withSupplemental['cr'], ...withSupplemental['cp']];
            const inProgress: Array<Project | Routine> = [...withSupplemental['ipr'], ...withSupplemental['ipp']];
            const recent: Array<Project | Routine> = [...withSupplemental['rr'], ...withSupplemental['rp']];
            // Sort arrays by date completed/updated. Completed and inProgress have to be sorted differently than other 
            // sorts, because the timestamp for routines is in the log, but the timestamp for projects is in the project data.
            completed.sort((a, b) => {
                // Check for log, which means it's a routine
                const aLog = completedLogs.find((log: Log) => log.id === a.id);
                const bLog = completedLogs.find((log: Log) => log.id === b.id);
                // Determine completedAt
                const aCompletedAt = aLog ? aLog.timestamp : (a as Project).completedAt;
                const bCompletedAt = bLog ? bLog.timestamp : (b as Project).completedAt;
                // Compare
                if (aCompletedAt < bCompletedAt) return -1;
                if (aCompletedAt > bCompletedAt) return 1;
                return 0;
            });
            inProgress.sort((a, b) => {
                // Check for log, which means it's a routine
                const aLog = inProgressLogs.find((log: Log) => log.id === a.id);
                const bLog = inProgressLogs.find((log: Log) => log.id === b.id);
                // Determine updatedAt
                const aUpdatedAt = aLog ? aLog.timestamp : (a as Project).updated_at;
                const bUpdatedAt = bLog ? bLog.timestamp : (b as Project).updated_at;
                // Compare
                if (aUpdatedAt < bUpdatedAt) return -1;
                if (aUpdatedAt > bUpdatedAt) return 1;
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
         * Queries data shown on For You page 
         */
        forYouPage: async (_parent: undefined, { input }: IWrap<ForYouPageInput>, context: Context, info: GraphQLResolveInfo): Promise<ForYouPageResult> => {
            // If not signed in, shouldn't be able to see this page
            if (!context.req.userId) throw new CustomError(CODE.Unauthorized, 'Must be signed in to see this page');
            await rateLimit({ context, info, max: 5000 });
            const take = 5;
            // Initialize models
            const oModel = OrganizationModel(context.prisma);
            const pModel = ProjectModel(context.prisma);
            const rModel = RoutineModel(context.prisma);
            const sModel = StandardModel(context.prisma);
            const uModel = UserModel(context.prisma);
            // Query RoutineIncomplete logs
            const activeLogs = (await paginatedMongoSearch<LogSearchResult>({
                findQuery: logSearcher().getFindQuery(context.req.userId, { actions: [LogType.RoutineStartIncomplete] }),
                sortQuery: logSearcher().getSortQuery(LogSortBy.DateCreatedAsc),
                take,
                project: logSearcher().defaultProjection,
            })).edges.map(({ node }: any) => node) as Log[]
            console.log('active logs!', JSON.stringify(activeLogs));
            // Use logs to find full routine data from Prisma
            const activeRoutineIds = activeLogs.map((node: any) => node.object1Id);
            const activeRoutines = (await readManyHelper(
                context.req.userId,
                { ids: activeRoutineIds },
                routineSelect,
                rModel,
                {},
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(routineSelect, rModel.relationshipMap) as PartialInfo)) as any[];
            // Sort by timestamp of corresponding log
            activeRoutines.sort((a, b) => {
                const aLog = activeLogs.find((log: Log) => log.object1Id === a.id);
                const bLog = activeLogs.find((log: Log) => log.object1Id === b.id);
                if (aLog && bLog) {
                    if (aLog.timestamp < bLog.timestamp) return -1;
                    if (aLog.timestamp > bLog.timestamp) return 1;
                    return 0;
                }
                return 0;
            });
            console.log('active routines!', JSON.stringify(activeRoutines));
            // Query recently completed routines
            const completedLogs = (await paginatedMongoSearch<LogSearchResult>({
                findQuery: logSearcher().getFindQuery(context.req.userId, { actions: [LogType.RoutineComplete] }),
                sortQuery: logSearcher().getSortQuery(LogSortBy.DateCreatedAsc),
                take,
                project: logSearcher().defaultProjection,
            })).edges.map(({ node }: any) => node) as Log[]
            // Use logs to find full routine data from Prisma
            const completedRoutineIds = completedLogs.map((node: any) => node.object1Id);
            const completedRoutines = (await readManyHelper(
                context.req.userId,
                { ids: completedRoutineIds },
                routineSelect,
                rModel,
                {},
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(routineSelect, rModel.relationshipMap) as PartialInfo)) as any[];
            // Sort by timestamp of corresponding log
            completedRoutines.sort((a, b) => {
                const aLog = activeLogs.find((log: Log) => log.object1Id === a.id);
                const bLog = activeLogs.find((log: Log) => log.object1Id === b.id);
                if (aLog && bLog) {
                    if (aLog.timestamp < bLog.timestamp) return -1;
                    if (aLog.timestamp > bLog.timestamp) return 1;
                    return 0;
                }
                return 0;
            });
            // Query recently viewed objects (of any type)
            const viewedLogs = (await paginatedMongoSearch<LogSearchResult>({
                findQuery: logSearcher().getFindQuery(context.req.userId, { actions: [LogType.View] }),
                sortQuery: logSearcher().getSortQuery(LogSortBy.DateCreatedAsc),
                take,
                project: logSearcher().defaultProjection,
            })).edges.map(({ node }: any) => node) as Log[]
            console.log('viewed logs!', JSON.stringify(viewedLogs));
            // Query recently starred objects (of any type)
            const starredLogs = (await paginatedMongoSearch<LogSearchResult>({
                findQuery: logSearcher().getFindQuery(context.req.userId, { actions: [LogType.Star] }),
                sortQuery: logSearcher().getSortQuery(LogSortBy.DateCreatedAsc),
                take,
                project: logSearcher().defaultProjection,
            })).edges.map(({ node }: any) => node) as Log[]
            console.log('starred logs!', JSON.stringify(starredLogs));
            // Group log ids from viewedLogs and starredLogs by object type, to reduce database calls
            const combinedLogs = [...viewedLogs, ...starredLogs];
            const organizationLogIds = combinedLogs.filter((log: Log) => Boolean(log.object1Id) && log.object1Type === GraphQLModelType.Organization).map((log: Log) => log.object1Id) as string[];
            const projectLogIds = combinedLogs.filter((log: Log) => Boolean(log.object1Id) && log.object1Type === GraphQLModelType.Project).map((log: Log) => log.object1Id) as string[];
            const routineLogIds = combinedLogs.filter((log: Log) => Boolean(log.object1Id) && log.object1Type === GraphQLModelType.Routine).map((log: Log) => log.object1Id) as string[];
            const standardLogIds = combinedLogs.filter((log: Log) => Boolean(log.object1Id) && log.object1Type === GraphQLModelType.Standard).map((log: Log) => log.object1Id) as string[];
            const userLogIds = combinedLogs.filter((log: Log) => Boolean(log.object1Id) && log.object1Type === GraphQLModelType.User).map((log: Log) => log.object1Id) as string[];
            console.log('combined logs', JSON.stringify(combinedLogs));
            console.log('organization log ids', JSON.stringify(organizationLogIds));
            console.log('project log ids', JSON.stringify(projectLogIds));
            console.log('routine log ids', JSON.stringify(routineLogIds));
            console.log('standard log ids', JSON.stringify(standardLogIds));
            console.log('user log ids', JSON.stringify(userLogIds));
            // Query each object type separately
            const organizations = (await readManyHelper(
                context.req.userId,
                { ids: organizationLogIds },
                organizationSelect,
                oModel,
                {},
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(organizationSelect, oModel.relationshipMap) as PartialInfo)) as any[];
            const projects = (await readManyHelper(
                context.req.userId,
                { ids: projectLogIds },
                projectSelect,
                pModel,
                {},
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(projectSelect, pModel.relationshipMap) as PartialInfo)) as any[];
            const routines = (await readManyHelper(
                context.req.userId,
                { ids: routineLogIds },
                routineSelect,
                rModel,
                {},
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(routineSelect, rModel.relationshipMap) as PartialInfo)) as any[];
            const standards = (await readManyHelper(
                context.req.userId,
                { ids: standardLogIds },
                standardSelect,
                sModel,
                {},
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(standardSelect, sModel.relationshipMap) as PartialInfo)) as any[];
            const users = (await readManyHelper(
                context.req.userId,
                { ids: userLogIds },
                userSelect,
                uModel,
                {},
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(userSelect, uModel.relationshipMap) as PartialInfo)) as any[];
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [activeRoutines, completedRoutines, organizations, projects, routines, standards, users],
                [routineSelect, routineSelect, organizationSelect, projectSelect, routineSelect, standardSelect, userSelect] as any,
                ['ar', 'cr', 'o', 'p', 'r', 's', 'u'],
                context.req.userId,
                context.prisma,
            )
            // Recombine organizations, projects, routines, standards, and users into two arrays - one for viewed and one for starred
            // Also sort by timestamp of corresponding log
            const viewedAndStarred = [...withSupplemental['o'], ...withSupplemental['p'], ...withSupplemental['r'], ...withSupplemental['s'], ...withSupplemental['u']];
            const viewed = viewedAndStarred.filter((obj: Organization | Project | Routine | Standard | User) => {
                const log = viewedLogs.find((log: Log) => log.object1Id === obj.id);
                return Boolean(log);
            }).sort((a, b) => {
                const aLog = viewedLogs.find((log: Log) => log.object1Id === a.id);
                const bLog = viewedLogs.find((log: Log) => log.object1Id === b.id);
                if (aLog && bLog) {
                    if (aLog.timestamp < bLog.timestamp) return -1;
                    if (aLog.timestamp > bLog.timestamp) return 1;
                    return 0;
                }
                return 0;
            });
            const starred = viewedAndStarred.filter((obj: Organization | Project | Routine | Standard | User) => {
                const log = starredLogs.find((log: Log) => log.object1Id === obj.id);
                return Boolean(log);
            }).sort((a, b) => {
                const aLog = starredLogs.find((log: Log) => log.object1Id === a.id);
                const bLog = starredLogs.find((log: Log) => log.object1Id === b.id);
                if (aLog && bLog) {
                    if (aLog.timestamp < bLog.timestamp) return -1;
                    if (aLog.timestamp > bLog.timestamp) return 1;
                    return 0;
                }
                return 0;
            });
            // Return results
            return {
                activeRoutines: withSupplemental['ar'],
                completedRoutines: withSupplemental['cr'],
                recent: viewed,
                starred,
            } as any
        },
        /**
         * Returns site-wide statistics
         */
        statisticsPage: async (_parent: undefined, { input }: IWrap<StatisticsPageInput>, context: Context, info: GraphQLResolveInfo): Promise<StatisticsPageResult> => {
            await rateLimit({ context, info, max: 500 });
            // Query current stats
            // Read historical stats from file
            throw new CustomError(CODE.NotImplemented);
        },
    },
}
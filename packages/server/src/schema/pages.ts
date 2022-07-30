/**
 * Endpoints optimized for specific pages
 */
import { gql } from 'apollo-server-express';
import { GraphQLResolveInfo } from "graphql";
import { HomePageInput, HomePageResult, DevelopPageResult, LearnPageResult, OrganizationSortBy, ProjectSortBy, ResearchPageResult, ResourceUsedFor, RoutineSortBy, StandardSortBy, UserSortBy, HistoryPageInput, HistoryPageResult, StatisticsPageInput, StatisticsPageResult, Project, Routine, RunStatus, RunSortBy, ViewSortBy } from './types';
import { CODE } from '@local/shared';
import { IWrap } from '../types';
import { Context } from '../context';
import { addSupplementalFieldsMultiTypes, GraphQLModelType, modelToGraphQL, OrganizationModel, PartialGraphQLInfo, ProjectModel, readManyHelper, RoutineModel, RunModel, StandardModel, StarModel, toPartialGraphQLInfo, UserModel, ViewModel } from '../models';
import { CustomError } from '../error';
import { rateLimit } from '../rateLimit';
import { resolveProjectOrOrganization, resolveProjectOrOrganizationOrRoutineOrStandardOrUser, resolveProjectOrRoutine } from './resolvers';

// Query fields shared across multiple endpoints
const tagSelect = {
    __typename: GraphQLModelType.Tag,
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
    __typename: GraphQLModelType.Organization,
    id: true,
    commentsCount: true,
    handle: true,
    stars: true,
    isOpenToNewMembers: true,
    isPrivate: true,
    isStarred: true,
    permissionsOrganization: {
        canAddMembers: true,
        canDelete: true,
        canEdit: true,
        canStar: true,
        canReport: true,
        isMember: true,
    },
    reportsCount: true,
    translations: {
        id: true,
        language: true,
        name: true,
    },
    tags: tagSelect,
}
const projectSelect = {
    __typename: GraphQLModelType.Project,
    id: true,
    commentsCount: true,
    completedAt: true,
    handle: true,
    stars: true,
    score: true,
    isComplete: true,
    isPrivate: true,
    isStarred: true,
    isUpvoted: true,
    permissionsProject: {
        canDelete: true,
        canEdit: true,
        canStar: true,
        canReport: true,
        canVote: true,
    },
    reportsCount: true,
    translations: {
        id: true,
        language: true,
        name: true,
    },
    tags: tagSelect,
}
const routineSelect = {
    __typename: GraphQLModelType.Routine,
    id: true,
    commentsCount: true,
    created_at: true,
    completedAt: true,
    complexity: true,
    simplicity: true,
    stars: true,
    score: true,
    isComplete: true,
    isDeleted: true,
    isPrivate: true,
    isStarred: true,
    isUpvoted: true,
    permissionsRoutine: {
        canDelete: true,
        canEdit: true,
        canStar: true,
        canReport: true,
        canVote: true,
    },
    reportsCount: true,
    translations: {
        id: true,
        language: true,
        title: true,
        instructions: true,
    },
    tags: tagSelect,
    version: true,
    versionGroupId: true,
}
const runSelect = {
    __typename: GraphQLModelType.Run,
    id: true,
    completedComplexity: true,
    contextSwitches: true,
    isPrivate: true,
    timeStarted: true,
    timeElapsed: true,
    timeCompleted: true,
    title: true,
    status: true,
    version: true,
    routine: routineSelect,
}
const standardSelect = {
    __typename: GraphQLModelType.Standard,
    id: true,
    commentsCount: true,
    name: true,
    props: true,
    stars: true,
    score: true,
    isDeleted: true,
    isInternal: true,
    isPrivate: true,
    isStarred: true,
    isUpvoted: true,
    permissionsStandard: {
        canDelete: true,
        canEdit: true,
        canStar: true,
        canReport: true,
        canVote: true,
    },
    reportsCount: true,
    translations: {
        id: true,
        description: true,
        language: true,
    },
    tags: tagSelect,
    type: true,
    version: true,
    versionGroupId: true,
}
const userSelect = {
    __typename: GraphQLModelType.User,
    id: true,
    name: true,
    handle: true,
    reportsCount: true,
    stars: true,
    isStarred: true,
    translations: {
        id: true,
        language: true,
        bio: true,
    },
}
const viewSelect = {
    __typename: GraphQLModelType.View,
    id: true,
    lastViewed: true,
    title: true,
    to: {
        Organization: organizationSelect,
        Project: projectSelect,
        Routine: routineSelect,
        Standard: standardSelect,
        User: userSelect,
    }
}
const commentSelect = {
    __typename: GraphQLModelType.Comment,
    id: true,
    created_at: true,
    updated_at: true,
    score: true,
    isUpvoted: true,
    permissionsComment: {
        canDelete: true,
        canEdit: true,
        canStar: true,
        canReport: true,
        canVote: true,
    },
    isStarred: true,
    commentedOn: {
        Project: projectSelect,
        Routine: routineSelect,
        Standard: standardSelect,
    },
    creator: {
        Organization: organizationSelect,
        User: userSelect,
    },
    translations: {
        id: true,
        language: true,
        text: true,
    }
}
const starSelect = {
    __typename: GraphQLModelType.Star,
    id: true,
    to: {
        Comment: commentSelect,
        Organization: organizationSelect,
        Project: projectSelect,
        Routine: routineSelect,
        Standard: standardSelect,
        User: userSelect,
    }
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
                { take, ...input, sortBy: OrganizationSortBy.StarsDesc },
                organizationSelect,
                oModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(organizationSelect, oModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query projects
            let projects = (await readManyHelper(
                context.req.userId,
                { take, ...input, sortBy: ProjectSortBy.StarsDesc, isComplete: true, },
                projectSelect,
                pModel,
                { ...starsQuery },
                false
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(projectSelect, pModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query routines
            let routines = (await readManyHelper(
                context.req.userId,
                { take, ...input, sortBy: RoutineSortBy.StarsDesc, isComplete: true, isInternal: false },
                routineSelect,
                rModel,
                { ...starsQuery },
                false
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(routineSelect, rModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query standards
            let standards = (await readManyHelper(
                context.req.userId,
                { take, ...input, sortBy: StandardSortBy.StarsDesc, type: 'JSON' },
                standardSelect,
                sModel,
                { ...starsQuery },
                false
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(standardSelect, sModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query users
            let users = (await readManyHelper(
                context.req.userId,
                { take, ...input, sortBy: UserSortBy.StarsDesc },
                userSelect,
                uModel,
                { ...starsQuery },
                false
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(userSelect, uModel.relationshipMap) as PartialGraphQLInfo)) as any[]
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
                { take, sortBy: ProjectSortBy.VotesDesc, tags: ['Learn'], isComplete: true, },
                projectSelect,
                pModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(projectSelect, pModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query tutorials
            const tutorials = (await readManyHelper(
                context.req.userId,
                { take, sortBy: RoutineSortBy.VotesDesc, tags: ['Learn'], isComplete: true, isInternal: false },
                routineSelect,
                rModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(routineSelect, rModel.relationshipMap) as PartialGraphQLInfo)) as any[]
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
                { take, sortBy: RoutineSortBy.VotesDesc, tags: ['Research'], isComplete: true, isInternal: false },
                routineSelect,
                rModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(routineSelect, rModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query newlyCompleted
            const newlyCompletedProjects = (await readManyHelper(
                context.req.userId,
                { take, sortBy: ProjectSortBy.DateCompletedAsc, isComplete: true },
                projectSelect,
                pModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(projectSelect, pModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            const newlyCompletedRoutines = (await readManyHelper(
                context.req.userId,
                { take, isComplete: true, isInternal: false, sortBy: RoutineSortBy.DateCompletedAsc },
                routineSelect,
                rModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(routineSelect, rModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query needVotes
            const needVotes = (await readManyHelper(
                context.req.userId,
                { take, isComplete: false, resourceTypes: [ResourceUsedFor.Proposal] },
                projectSelect,
                pModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(projectSelect, pModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query needInvestments
            const needInvestmentsProjects = (await readManyHelper(
                context.req.userId,
                { take, isComplete: false, resourceTypes: [ResourceUsedFor.Donation] },
                projectSelect,
                pModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(projectSelect, pModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            const needInvestmentsOrganizations = (await readManyHelper(
                context.req.userId,
                { take, resourceTypes: [ResourceUsedFor.Donation] },
                organizationSelect,
                oModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(organizationSelect, oModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query needMembers
            const needMembers = (await readManyHelper(
                context.req.userId,
                { take, isOpenToNewMembers: true, sortBy: OrganizationSortBy.StarsDesc },
                organizationSelect,
                oModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(organizationSelect, oModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [processes, newlyCompletedProjects, newlyCompletedRoutines, needVotes, needInvestmentsProjects, needInvestmentsOrganizations, needMembers],
                [routineSelect, projectSelect, routineSelect, projectSelect, projectSelect, organizationSelect, organizationSelect] as any,
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
            const take = 5;
            // Initialize models
            const pModel = ProjectModel(context.prisma);
            const rModel = RoutineModel(context.prisma);
            // Query for routines you've completed
            const completedRoutines = (await readManyHelper(
                context.req.userId,
                { take, isComplete: true, isInternal: false, userId: context.req.userId, sortBy: RoutineSortBy.DateCompletedAsc },
                routineSelect,
                rModel,
                {},
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(routineSelect, rModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query for projects you've completed
            const completedProjects = (await readManyHelper(
                context.req.userId,
                { take, isComplete: true, userId: context.req.userId, sortBy: ProjectSortBy.DateCompletedAsc },
                projectSelect,
                pModel,
                {},
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(projectSelect, pModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query for routines you're currently working on
            const inProgressRoutines = (await readManyHelper(
                context.req.userId,
                { take, isComplete: false, isInternal: false, userId: context.req.userId, sortBy: RoutineSortBy.DateCreatedAsc },
                routineSelect,
                rModel,
                {},
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(routineSelect, rModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query for projects you're currently working on
            const inProgressProjects = (await readManyHelper(
                context.req.userId,
                { take, isComplete: false, userId: context.req.userId, sortBy: ProjectSortBy.DateCreatedAsc },
                projectSelect,
                pModel,
                {},
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(projectSelect, pModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query recently created/updated routines
            const recentRoutines = (await readManyHelper(
                context.req.userId,
                { take, userId: context.req.userId, sortBy: RoutineSortBy.DateUpdatedAsc, isInternal: false },
                routineSelect,
                rModel,
                {},
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(routineSelect, rModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query recently created/updated projects
            const recentProjects = (await readManyHelper(
                context.req.userId,
                { take, userId: context.req.userId, sortBy: ProjectSortBy.DateUpdatedAsc },
                projectSelect,
                pModel,
                {},
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(projectSelect, pModel.relationshipMap) as PartialGraphQLInfo)) as any[]
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
        historyPage: async (_parent: undefined, { input }: IWrap<HistoryPageInput>, context: Context, info: GraphQLResolveInfo): Promise<HistoryPageResult> => {
            // If not signed in, shouldn't be able to see this page
            if (!context.req.userId) throw new CustomError(CODE.Unauthorized, 'Must be signed in to see this page');
            await rateLimit({ context, info, max: 5000 });
            const take = 5;
            // Initialize models
            const runModel = RunModel(context.prisma);
            const starModel = StarModel(context.prisma);
            const viewModel = ViewModel(context.prisma);
            // Query for incomplete runs
            const activeRuns = (await readManyHelper(
                context.req.userId,
                { take, ...input, status: RunStatus.InProgress, sortBy: RunSortBy.DateUpdatedDesc },
                runSelect,
                runModel,
                { userId: context.req.userId },
                false
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(runSelect, runModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query for complete runs
            const completedRuns = (await readManyHelper(
                context.req.userId,
                { take, ...input, status: RunStatus.Completed, sortBy: RunSortBy.DateUpdatedDesc },
                runSelect,
                runModel,
                { userId: context.req.userId },
                false
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(runSelect, runModel.relationshipMap) as PartialGraphQLInfo)) as any[]
            // Query recently viewed objects (of any type)
            const recentlyViewed = (await readManyHelper(
                context.req.userId,
                { take, ...input, sortBy: ViewSortBy.LastViewedDesc },
                viewSelect,
                viewModel,
                { byId: context.req.userId },
                false
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(viewSelect, viewModel.relationshipMap) as PartialGraphQLInfo)) as any[];
            // Query recently starred objects (of any type). Make sure to ignore tags
            const recentlyStarred = (await readManyHelper(
                context.req.userId,
                { take, ...input, sortBy: UserSortBy.DateCreatedDesc },
                starSelect,
                starModel,
                { byId: context.req.userId, tagId: null },
                false
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialGraphQLInfo(starSelect, starModel.relationshipMap) as PartialGraphQLInfo)) as any[];
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [activeRuns, completedRuns, recentlyViewed, recentlyStarred],
                [runSelect, runSelect, viewSelect, starSelect] as any,
                ['ar', 'cr', 'rv', 'rs'],
                context.req.userId,
                context.prisma,
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
        statisticsPage: async (_parent: undefined, { input }: IWrap<StatisticsPageInput>, context: Context, info: GraphQLResolveInfo): Promise<StatisticsPageResult> => {
            await rateLimit({ context, info, max: 500 });
            // Query current stats
            // Read historical stats from file
            throw new CustomError(CODE.NotImplemented);
        },
    },
}
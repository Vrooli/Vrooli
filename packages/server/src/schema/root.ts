import { gql } from 'apollo-server-express';
import { GraphQLResolveInfo, GraphQLScalarType } from "graphql";
import { GraphQLUpload } from 'graphql-upload';
import { readFiles, saveFiles } from '../utils';
// import ogs from 'open-graph-scraper';
import { AutocompleteInput, AutocompleteResult, DevelopPageResult, LearnPageResult, LogSearchResult, LogSortBy, OrganizationSortBy, ProjectSortBy, ResearchPageResult, ResourceUsedFor, RoutineSortBy, StandardSortBy, UserSortBy } from './types';
import { CODE } from '@local/shared';
import { IWrap } from '../types';
import { Context } from '../context';
import { addSupplementalFieldsMultiTypes, GraphQLModelType, logSearcher, LogType, modelToGraphQL, OrganizationModel, paginatedMongoSearch, PartialInfo, ProjectModel, readManyHelper, RoutineModel, StandardModel, toPartialSelect, UserModel } from '../models';
import { CustomError } from '../error';
import { rateLimit } from '../rateLimit';
import { genErrorCode, logger, LogLevel } from '../logger';

// Defines common inputs, outputs, and types for all GraphQL queries and mutations.
export const typeDef = gql`
    scalar Date
    scalar Upload

    # Used for Projects, Standards, and Routines, since they can be created 
    # by either a User or an Organization.
    union Contributor = User | Organization

    # Used for LRD pages
    union ProjectOrRoutine = Project | Routine
    union ProjectOrOrganization = Project | Organization

    # Used for filtering by date created/updated, as well as fetching metrics (e.g. monthly active users)
    input TimeFrame {
        after: Date
        before: Date
    }

    # Return type for a cursor-based pagination's pageInfo response
    type PageInfo {
        hasNextPage: Boolean!
        endCursor: String
    }
    # Return type for delete mutations,
    # which return the number of affected rows
    type Count {
        count: Int
    }
    # Return type for mutations with a success boolean
    # Could return just the boolean, but this makes it clear what the result means
    type Success {
        success: Boolean
    }
    # Return type for error messages
    type Response {
        code: Int
        message: String!
    }

    input ReadAssetsInput {
        files: [String!]!
    }

    input WriteAssetsInput {
        files: [Upload!]!
    }

    # Input for finding object by id
    input FindByIdInput {
        id: ID!
    }

    # Input for deleting one object
    input DeleteOneInput {
        id: ID!
    }

    # Input for deleting multiple objects
    input DeleteManyInput {
        ids: [ID!]!
    }

    # Input for site-wide autocomplete search
    input AutocompleteInput {
        searchString: String!
        take: Int
    }

    type AutocompleteResult {
        organizations: [Organization!]!
        projects: [Project!]!
        routines: [Routine!]!
        standards: [Standard!]!
        users: [User!]!
    }

    type StatisticsResult {
        daily: StatisticsTimeFrame!
        weekly: StatisticsTimeFrame!
        monthly: StatisticsTimeFrame!
        yearly: StatisticsTimeFrame!
        allTime: StatisticsTimeFrame!
    }

    type StatisticsTimeFrame {
        organizations: [Int!]!
        projects: [Int!]!
        routines: [Int!]!
        standards: [Int!]!
        users: [Int!]!
    }

    # Result returned from learn page query
    type LearnPageResult {
        courses: [Project!]!
        tutorials: [Routine!]!
    }

    # Result returned from research page query
    type ResearchPageResult {
        processes: [Routine!]!
        newlyCompleted: [ProjectOrRoutine!]!
        needVotes: [Project!]!
        needInvestments: [Project!]!
        needMembers: [Organization!]!
    }

    # Result returned from develop page query
    type DevelopPageResult {
        completed: [ProjectOrRoutine!]!
        inProgress: [ProjectOrRoutine!]!
        recent: [ProjectOrRoutine!]!
    }

    # Base query. Must contain something,
    # which can be as simple as '_empty: String'
    type Query {
        # _empty: String
        readAssets(input: ReadAssetsInput!): [String]!
        autocomplete(input: AutocompleteInput!): AutocompleteResult!
        learnPage: LearnPageResult!
        researchPage: ResearchPageResult!
        developPage: DevelopPageResult!
        statistics: StatisticsResult!
    }
    # Base mutation. Must contain something,
    # which can be as simple as '_empty: String'
    type Mutation {
        # _empty: String
        writeAssets(input: WriteAssetsInput!): Boolean
    }
`

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
    stars: true,
    score: true,
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

export const resolvers = {
    Upload: GraphQLUpload,
    Date: new GraphQLScalarType({
        name: "Date",
        description: "Custom description for the date scalar",
        // Assumes data is either Unix timestamp or Date object
        parseValue(value) {
            return new Date(value).toISOString(); // value from the client
        },
        serialize(value) {
            return new Date(value).getTime(); // value sent to the client
        },
        parseLiteral(ast: any) {
            return new Date(ast).toDateString(); // ast value is always in string format
        }
    }),
    Contributor: {
        __resolveType(obj: any) {
            // Only a user has a name field
            if (obj.hasOwnProperty('name')) return GraphQLModelType.User;
            return GraphQLModelType.Organization;
        },
    },
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
    Query: {
        readAssets: async (_parent: undefined, { input }: any, context: Context, info: GraphQLResolveInfo): Promise<Array<String | null>> => {
            await rateLimit({ context, info, max: 1000 });
            return await readFiles(input.files);
        },
        /**
         * Autocomplete endpoint for main page. Combines search queries for all main objects
         */
        autocomplete: async (_parent: undefined, { input }: IWrap<AutocompleteInput>, context: Context, info: GraphQLResolveInfo): Promise<AutocompleteResult> => {
            await rateLimit({ context, info, max: 5000 });
            const MinimumStars = 0; // Minimum stars required to show up in autocomplete results. Will increase in the future.
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
            )).edges.map(({ node }: any) =>  modelToGraphQL(node, toPartialSelect(standardSelect, sModel.relationshipMap) as PartialInfo)) as any[]
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
            const MinimumStars = 0; // Minimum stars required to show up in autocomplete results. Will increase in the future.
            const starsQuery = { stars: { gte: MinimumStars } };
            const take = 5;
            // Initialize models
            const oModel = OrganizationModel(context.prisma);
            const pModel = ProjectModel(context.prisma);
            const rModel = RoutineModel(context.prisma);
            // Find completed logs
            const completedLogs = await paginatedMongoSearch<LogSearchResult>({
                findQuery: logSearcher().getFindQuery(context.req.userId ?? '', { action: LogType.RoutineComplete }),
                sortQuery: logSearcher().getSortQuery(LogSortBy.DateCreatedAsc),
                take,
                project: logSearcher().defaultProjection,
            });
            console.log('develop query completed logs', JSON.stringify(completedLogs));
            // Use logs to find full routine data from Prisma
            const completedIds = completedLogs.edges.map(({ node }: any) => node.object1Id);
            const completedRoutines = (await readManyHelper(
                context.req.userId,
                { ids: completedIds, sortBy: RoutineSortBy.DateCompletedAsc },
                routineSelect,
                rModel,
                { ...starsQuery },
                false,
            )).edges.map(({ node }: any) => modelToGraphQL(node, toPartialSelect(routineSelect, rModel.relationshipMap) as PartialInfo)) as any[]
            throw new CustomError(CODE.NotImplemented);
            // Query completed
            //TODO
            // Query inProgress
            //TODO
            // Query recent
            //TODO
            // Return data
            // return {
            //     completed,
            //     inProgress,
            //     recent,
            // }
        },
        /**
         * Returns site-wide statistics
         */
        statistics: async (_parent: undefined, { input }: IWrap<AutocompleteInput>, context: Context, info: GraphQLResolveInfo): Promise<any> => {
            await rateLimit({ context, info, max: 500 });
            // Query current stats
            // Read historical stats from file
            throw new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        writeAssets: async (_parent: undefined, { input }: any, context: Context, info: GraphQLResolveInfo): Promise<boolean> => {
            await rateLimit({ context, info, max: 500 });
            throw new CustomError(CODE.NotImplemented); // TODO add safety checks before allowing uploads
            const data = await saveFiles(input.files);
            // Any failed writes will return null
            return !data.some(d => d === null)
        },
    }
}
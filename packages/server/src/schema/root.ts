import { gql } from 'apollo-server-express';
import { GraphQLResolveInfo, GraphQLScalarType } from "graphql";
import { GraphQLUpload } from 'graphql-upload';
import { readFiles, saveFiles } from '../utils';
// import ogs from 'open-graph-scraper';
import { AutocompleteInput, AutocompleteResult, OrganizationSortBy, ProjectSortBy, RoutineSortBy, StandardSortBy, UserSortBy } from './types';
import { CODE } from '@local/shared';
import { IWrap } from '../types';
import { Context } from '../context';
import { GraphQLModelType, OrganizationModel, ProjectModel, readManyHelper, RoutineModel, StandardModel, UserModel } from '../models';
import { CustomError } from '../error';

// Defines common inputs, outputs, and types for all GraphQL queries and mutations.
export const typeDef = gql`
    scalar Date
    scalar Upload

    # Used for Projects, Standards, and Routines, since they can be created 
    # by either a User or an Organization.
    union Contributor = User | Organization

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

    # Base query. Must contain something,
    # which can be as simple as '_empty: String'
    type Query {
        # _empty: String
        readAssets(input: ReadAssetsInput!): [String]!
        autocomplete(input: AutocompleteInput!): AutocompleteResult!
        statistics: StatisticsResult!
    }
    # Base mutation. Must contain something,
    # which can be as simple as '_empty: String'
    type Mutation {
        # _empty: String
        writeAssets(input: WriteAssetsInput!): Boolean
    }
`

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
    Query: {
        readAssets: async (_parent: undefined, { input }: any): Promise<Array<String | null>> => {
            return await readFiles(input.files);
        },
        /**
         * Autocomplete endpoint for main page. Combines search queries for all main objects
         */
        autocomplete: async (_parent: undefined, { input }: IWrap<AutocompleteInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<AutocompleteResult> => {
            const MinimumStars = 0; // Minimum stars required to show up in autocomplete results. Will increase in the future.
            const starsQuery = { stars: { gte: MinimumStars } }
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
            // Query organizations
            const organizations = (await readManyHelper(
                req.userId, 
                { ...input, take: 5, sortBy: OrganizationSortBy.StarsDesc }, 
                {
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
                },
                OrganizationModel(prisma),
                { ...starsQuery }
            )).edges.map(({ node }: any) => node);
            // Query projects
            const projects = (await readManyHelper(
                req.userId, 
                { ...input, take: 5, sortBy: ProjectSortBy.StarsDesc }, 
                {
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
                },
                ProjectModel(prisma),
                { ...starsQuery }
            )).edges.map(({ node }: any) => node);
            // Query routines
            const routines = (await readManyHelper(
                req.userId, 
                { ...input, take: 5, sortBy: RoutineSortBy.StarsDesc }, 
                {
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
                },
                RoutineModel(prisma),
                { ...starsQuery }
            )).edges.map(({ node }: any) => node);
            // Query standards
            const standards = (await readManyHelper(
                req.userId, 
                { ...input, take: 5, sortBy: StandardSortBy.StarsDesc }, 
                {
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
                },
                StandardModel(prisma),
                { ...starsQuery }
            )).edges.map(({ node }: any) => node);
            // Query users
            const users = (await readManyHelper(
                req.userId, 
                { ...input, take: 5, sortBy: UserSortBy.StarsDesc }, 
                {
                    __typename: 'User',
                    id: true,
                    name: true,
                    handle: true,
                    stars: true,
                    isStarred: true,
                },
                UserModel(prisma),
                { ...starsQuery }
            )).edges.map(({ node }: any) => node);
            return {
                organizations,
                projects,
                routines,
                standards,
                users
            }
        },
        /**
         * Returns site-wide statistics
         */
        statistics: async (_parent: undefined, { input }: IWrap<AutocompleteInput>): Promise<any> => {
            // Query current stats
            // Read historical stats from file
            return new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        writeAssets: async (_parent: undefined, { input }: any): Promise<boolean> => {
            const data = await saveFiles(input.files);
            // Any failed writes will return null
            return !data.some(d => d === null)
        },
    }
}
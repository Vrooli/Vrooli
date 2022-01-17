import { gql } from 'apollo-server-express';
import { CODE, RoutineSortBy } from '@local/shared';
import { CustomError } from '../error';
import { IWrap, RecursivePartial } from '../types';
import { DeleteOneInput, FindByIdInput, Routine, RoutineCountInput, RoutineInput, RoutineSearchInput, Success } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { RoutineModel } from '../models';

export const typeDef = gql`
    enum RoutineSortBy {
        AlphabeticalAsc
        AlphabeticalDesc
        CommentsAsc
        CommentsDesc
        ForksAsc
        ForksDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
        VotesAsc
        VotesDesc
    }

    input RoutineInput {
        id: ID
        version: String
        title: String
        description: String
        instructions: String
        isAutomatable: Boolean
        inputs: [RoutineInputItemInput!]
        outputs: [RoutineOutputItemInput!]
    }

    type Routine {
        id: ID!
        created_at: Date!
        updated_at: Date!
        version: String
        title: String
        description: String
        instructions: String
        isAutomatable: Boolean
        stars: Int!
        score: Int!
        isUpvoted: Boolean
        inputs: [RoutineInputItem!]!
        outputs: [RoutineOutputItem!]!
        nodes: [Node!]!
        contextualResources: [Resource!]!
        externalResources: [Resource!]!
        project: Project
        tags: [Tag!]!
        owner: Contributor
        creator: Contributor
        starredBy: [User!]!
        parent: Routine
        forks: [Routine!]!
        nodeLists: [NodeRoutineList!]!
        reports: [Report!]!
        comments: [Comment!]!
    }

    input RoutineInputItemInput {
        id: ID
        routineId: ID!
        standardId: ID
    }

    type RoutineInputItem {
        id: ID!
        routine: Routine!
        standard: Standard!
    }

    input RoutineOutputItemInput {
        id: ID
        routineId: ID!
        standardId: ID
    }

    type RoutineOutputItem {
        id: ID!
        routine: Routine!
        standard: Standard!
    }

    input RoutineSearchInput {
        userId: ID
        organizationId: ID
        parentId: ID
        reportId: ID
        ids: [ID!]
        sortBy: RoutineSortBy
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        searchString: String
        after: String
        take: Int
    }

    # Return type for search result
    type RoutineSearchResult {
        pageInfo: PageInfo!
        edges: [RoutineEdge!]!
    }

    # Return type for search result edge
    type RoutineEdge {
        cursor: String!
        node: Routine!
    }

    # Input for count
    input RoutineCountInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
    }

    extend type Query {
        routine(input: FindByIdInput!): Routine
        routines(input: RoutineSearchInput!): RoutineSearchResult!
        routinesCount(input: RoutineCountInput!): Int!
    }

    extend type Mutation {
        routineAdd(input: RoutineInput!): Routine!
        routineUpdate(input: RoutineInput!): Routine!
        routineDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    RoutineSortBy: RoutineSortBy,
    Query: {
        routine: async (_parent: undefined, { input }: IWrap<FindByIdInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            throw new CustomError(CODE.NotImplemented);
        },
        routines: async (_parent: undefined, { input }: IWrap<RoutineSearchInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<any> => {
            // Create query for specified user
            const userQuery = input.userId ? { user: { id: input.userId } } : undefined;
            // return search query
            return await RoutineModel(prisma).search({...userQuery,}, input, info);
        },
        routinesCount: async (_parent: undefined, { input }: IWrap<RoutineCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            // Return count query
            return await RoutineModel(prisma).count({}, input);
        },
    },
    Mutation: {
        routineAdd: async (_parent: undefined, { input }: IWrap<RoutineInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            // Create object
            const dbModel = await RoutineModel(prisma).create(input as any, info);
            // Format object to GraphQL type
            if (dbModel) return RoutineModel().toGraphQL(dbModel);
            throw new CustomError(CODE.ErrorUnknown);
        },
        routineUpdate: async (_parent: undefined, { input }: IWrap<RoutineInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            // Update object
            //const dbModel = await RoutineModel(prisma).update(input, info);
            // Format to GraphQL type
            //return RoutineModel().toGraphQL(dbModel);
            throw new CustomError(CODE.NotImplemented);
        },
        routineDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            const success = await RoutineModel(prisma).delete(input);
            return { success };
        },
    }
}
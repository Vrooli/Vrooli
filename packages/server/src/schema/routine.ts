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
        organizationId: ID
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
        isStarred: Boolean
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
        routine: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            const data = await RoutineModel(prisma).findRoutine(req.userId, input, info);
            if (!data) throw new CustomError(CODE.ErrorUnknown);
            return data;
        },
        routines: async (_parent: undefined, { input }: IWrap<RoutineSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<any> => {
            const data = await RoutineModel(prisma).searchRoutines({}, req.userId, input, info);
            if (!data) throw new CustomError(CODE.ErrorUnknown);
            return data;
        },
        routinesCount: async (_parent: undefined, { input }: IWrap<RoutineCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            // Return count query
            return await RoutineModel(prisma).count({}, input);
        },
    },
    Mutation: {
        routineAdd: async (_parent: undefined, { input }: IWrap<RoutineInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Create object
            const created = await RoutineModel(prisma).addRoutine(req.userId, input, info);
            if (!created) throw new CustomError(CODE.ErrorUnknown);
            return created;
        },
        routineUpdate: async (_parent: undefined, { input }: IWrap<RoutineInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Update object
            const updated = await RoutineModel(prisma).updateRoutine(req.userId, input, info);
            if (!updated) throw new CustomError(CODE.ErrorUnknown);
            return updated;
        },
        routineDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            return await RoutineModel(prisma).deleteRoutine(req.userId, input);
        },
    }
}
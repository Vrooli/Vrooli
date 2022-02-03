import { gql } from 'apollo-server-express';
import { CODE, RoutineSortBy } from '@local/shared';
import { CustomError } from '../error';
import { IWrap, RecursivePartial } from '../types';
import { DeleteOneInput, FindByIdInput, Routine, RoutineCountInput, RoutineAddInput, RoutineUpdateInput, RoutineSearchInput, Success } from './types';
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

    input RoutineAddInput {
        description: String
        instructions: String
        isAutomatable: Boolean
        title: String!
        version: String
        parentId: ID
        createdByUserId: ID
        createdByOrganizationId: ID
        nodesConnect: [ID!]
        nodesAdd: [NodeAddInput!]
        inputsAdd: [InputItemAddInput!]
        outputsAdd: [OutputItemAddInput!]
        resourcesContextualAdd: [ResourceAddInput!]
        resourcesExternalAdd: [ResourceAddInput!]
        tagsConnect: [ID!]
        tagsAdd: [TagAddInput!]
    }
    input RoutineUpdateInput {
        id: ID!
        description: String
        instructions: String
        isAutomatable: Boolean
        title: String
        version: String
        parentId: ID
        userId: ID
        organizationId: ID
        nodesConnect: [ID!]
        nodesDisconnect: [ID!]
        nodesDelete: [ID!]
        nodesAdd: [NodeAddInput!]
        nodesUpdate: [NodeUpdateInput!]
        inputsAdd: [InputItemAddInput!]
        inputsUpdate: [InputItemUpdateInput!]
        inputsDelete: [ID!]
        outputsAdd: [OutputItemAddInput!]
        outputsUpdate: [OutputItemUpdateInput!]
        outputsDelete: [ID!]
        resourcesContextualDelete: [ID!]
        resourcesContextualAdd: [ResourceAddInput!]
        resourcesContextualUpdate: [ResourceUpdateInput!]
        resourcesExternalDelete: [ID!]
        resourcesExternalAdd: [ResourceAddInput!]
        resourcesExternalUpdate: [ResourceUpdateInput!]
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
        tagsDelete: [ID!]
        tagsAdd: [TagAddInput!]
        tagsUpdate: [TagUpdateInput!]
    }
    type Routine {
        id: ID!
        created_at: Date!
        updated_at: Date!
        description: String
        instructions: String
        isAutomatable: Boolean
        isStarred: Boolean
        isUpvoted: Boolean
        score: Int!
        stars: Int!
        title: String
        version: String
        comments: [Comment!]!
        contextualResources: [Resource!]!
        creator: Contributor
        externalResources: [Resource!]!
        forks: [Routine!]!
        inputs: [InputItem!]!
        nodeLists: [NodeRoutineList!]!
        nodes: [Node!]!
        outputs: [OutputItem!]!
        owner: Contributor
        parent: Routine
        project: Project
        reports: [Report!]!
        starredBy: [User!]!
        tags: [Tag!]!
    }

    input InputItemAddInput {
        description: String
        isRequired: Boolean
        name: String
        standardConnect: ID
        standardAdd: StandardAddInput
    }
    input InputItemUpdateInput {
        id: ID!
        description: String
        isRequired: Boolean
        name: String
        standardConnect: ID
        standardAdd: StandardAddInput
    }
    type InputItem {
        id: ID!
        description: String
        isRequired: Boolean
        name: String
        routine: Routine!
        standard: Standard
    }

    input OutputItemAddInput {
        description: String
        name: String
        standardConnect: ID
        standardAdd: StandardAddInput
    }
    input OutputItemUpdateInput {
        id: ID!
        description: String
        name: String
        standardConnect: ID
        standardAdd: StandardAddInput
    }
    type OutputItem {
        id: ID!
        description: String
        name: String
        routine: Routine!
        standard: Standard
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
        routineAdd(input: RoutineAddInput!): Routine!
        routineUpdate(input: RoutineUpdateInput!): Routine!
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
        routineAdd: async (_parent: undefined, { input }: IWrap<RoutineAddInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Create object
            const created = await RoutineModel(prisma).addRoutine(req.userId, input, info);
            if (!created) throw new CustomError(CODE.ErrorUnknown);
            return created;
        },
        routineUpdate: async (_parent: undefined, { input }: IWrap<RoutineUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
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
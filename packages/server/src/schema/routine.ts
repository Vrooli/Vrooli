import { gql } from 'apollo-server-express';
import { CODE, RoutineSortBy } from '@local/shared';
import { CustomError } from '../error';
import { IWrap, RecursivePartial } from '../types';
import { DeleteOneInput, FindByIdInput, Routine, RoutineCountInput, RoutineCreateInput, RoutineUpdateInput, RoutineSearchInput, Success, RoutineSearchResult } from './types';
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

    input RoutineCreateInput {
        description: String
        instructions: String
        isAutomatable: Boolean
        title: String!
        version: String
        parentId: ID
        createdByUserId: ID
        createdByOrganizationId: ID
        nodesConnect: [ID!]
        nodesCreate: [NodeCreateInput!]
        inputsCreate: [InputItemCreateInput!]
        outputsCreate: [OutputItemCreateInput!]
        resourcesContextualCreate: [ResourceCreateInput!]
        resourcesExternalCreate: [ResourceCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
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
        nodesCreate: [NodeCreateInput!]
        nodesUpdate: [NodeUpdateInput!]
        inputsCreate: [InputItemCreateInput!]
        inputsUpdate: [InputItemUpdateInput!]
        inputsDelete: [ID!]
        outputsCreate: [OutputItemCreateInput!]
        outputsUpdate: [OutputItemUpdateInput!]
        outputsDelete: [ID!]
        resourcesContextualDelete: [ID!]
        resourcesContextualCreate: [ResourceCreateInput!]
        resourcesContextualUpdate: [ResourceUpdateInput!]
        resourcesExternalDelete: [ID!]
        resourcesExternalCreate: [ResourceCreateInput!]
        resourcesExternalUpdate: [ResourceUpdateInput!]
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
        tagsCreate: [TagCreateInput!]
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

    input InputItemCreateInput {
        description: String
        isRequired: Boolean
        name: String
        standardConnect: ID
        standardCreate: StandardCreateInput
    }
    input InputItemUpdateInput {
        id: ID!
        description: String
        isRequired: Boolean
        name: String
        standardConnect: ID
        standardCreate: StandardCreateInput
    }
    type InputItem {
        id: ID!
        description: String
        isRequired: Boolean
        name: String
        routine: Routine!
        standard: Standard
    }

    input OutputItemCreateInput {
        description: String
        name: String
        standardConnect: ID
        standardCreate: StandardCreateInput
    }
    input OutputItemUpdateInput {
        id: ID!
        description: String
        name: String
        standardConnect: ID
        standardCreate: StandardCreateInput
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
        routineCreate(input: RoutineCreateInput!): Routine!
        routineUpdate(input: RoutineUpdateInput!): Routine!
        routineDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    RoutineSortBy: RoutineSortBy,
    Query: {
        routine: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            const data = await RoutineModel(prisma).find(req.userId, input, info);
            if (!data) throw new CustomError(CODE.ErrorUnknown);
            return data;
        },
        routines: async (_parent: undefined, { input }: IWrap<RoutineSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RoutineSearchResult> => {
            const data = await RoutineModel(prisma).search({}, req.userId, input, info);
            if (!data) throw new CustomError(CODE.ErrorUnknown);
            return data;
        },
        routinesCount: async (_parent: undefined, { input }: IWrap<RoutineCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            // Return count query
            return await RoutineModel(prisma).count({}, input);
        },
    },
    Mutation: {
        routineCreate: async (_parent: undefined, { input }: IWrap<RoutineCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            // Create object
            const created = await RoutineModel(prisma).create(req.userId, input, info);
            if (!created) throw new CustomError(CODE.ErrorUnknown);
            return created;
        },
        routineUpdate: async (_parent: undefined, { input }: IWrap<RoutineUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            // Update object
            const updated = await RoutineModel(prisma).update(req.userId, input, info);
            if (!updated) throw new CustomError(CODE.ErrorUnknown);
            return updated;
        },
        routineDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            return await RoutineModel(prisma).delete(req.userId, input);
        },
    }
}
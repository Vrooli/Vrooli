import { gql } from 'apollo-server-express';
import { RoutineSortBy } from '@local/shared';
import { IWrap, RecursivePartial } from '../types';
import { DeleteOneInput, FindByIdInput, Routine, RoutineCountInput, RoutineCreateInput, RoutineUpdateInput, RoutineSearchInput, Success, RoutineSearchResult } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { countHelper, createHelper, deleteOneHelper, readManyHelper, readOneHelper, RoutineModel, updateHelper } from '../models';

export const typeDef = gql`
    enum RoutineSortBy {
        AlphabeticalAsc
        AlphabeticalDesc
        CommentsAsc
        CommentsDesc
        ForksAsc
        ForksDesc
        DateCompletedAsc
        DateCompletedDesc
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
        isComplete: Boolean
        isInternal: Boolean
        title: String!
        version: String
        parentId: ID
        projectId: ID
        createdByUserId: ID
        createdByOrganizationId: ID
        nodesCreate: [NodeCreateInput!]
        nodeLinksCreate: [NodeLinkCreateInput!]
        inputsCreate: [InputItemCreateInput!]
        outputsCreate: [OutputItemCreateInput!]
        resourcesCreate: [ResourceCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
    }
    input RoutineUpdateInput {
        id: ID!
        description: String
        instructions: String
        isAutomatable: Boolean
        isComplete: Boolean
        isInternal: Boolean
        title: String
        version: String
        parentId: ID
        userId: ID
        organizationId: ID
        nodesDelete: [ID!]
        nodesCreate: [NodeCreateInput!]
        nodesUpdate: [NodeUpdateInput!]
        nodeLinksDelete: [ID!]
        nodeLinksCreate: [NodeLinkCreateInput!]
        nodeLinksUpdate: [NodeLinkUpdateInput!]
        inputsCreate: [InputItemCreateInput!]
        inputsUpdate: [InputItemUpdateInput!]
        inputsDelete: [ID!]
        outputsCreate: [OutputItemCreateInput!]
        outputsUpdate: [OutputItemUpdateInput!]
        outputsDelete: [ID!]
        resourcesDelete: [ID!]
        resourcesCreate: [ResourceCreateInput!]
        resourcesUpdate: [ResourceUpdateInput!]
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
        tagsCreate: [TagCreateInput!]
    }
    type Routine {
        id: ID!
        completedAt: Date
        created_at: Date!
        updated_at: Date!
        description: String
        instructions: String
        isAutomatable: Boolean
        isComplete: Boolean!
        isInternal: Boolean
        isStarred: Boolean!
        role: MemberRole
        isUpvoted: Boolean
        score: Int!
        stars: Int!
        title: String
        version: String
        comments: [Comment!]!
        creator: Contributor
        forks: [Routine!]!
        inputs: [InputItem!]!
        nodeLists: [NodeRoutineList!]!
        nodes: [Node!]!
        nodeLinks: [NodeLink!]!
        outputs: [OutputItem!]!
        owner: Contributor
        parent: Routine
        project: Project
        reports: [Report!]!
        resources: [Resource!]!
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
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        isComplete: Boolean
        minScore: Int
        minStars: Int
        organizationId: ID
        projectId: ID
        parentId: ID
        reportId: ID
        searchString: String
        sortBy: RoutineSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
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
            return readOneHelper(req.userId, input, info, RoutineModel(prisma));
        },
        routines: async (_parent: undefined, { input }: IWrap<RoutineSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RoutineSearchResult> => {
            return readManyHelper(req.userId, input, info, RoutineModel(prisma));
        },
        routinesCount: async (_parent: undefined, { input }: IWrap<RoutineCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            return countHelper(input, RoutineModel(prisma));
        },
    },
    Mutation: {
        routineCreate: async (_parent: undefined, { input }: IWrap<RoutineCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            return createHelper(req.userId, input, info, RoutineModel(prisma));
        },
        routineUpdate: async (_parent: undefined, { input }: IWrap<RoutineUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            return updateHelper(req.userId, input, info, RoutineModel(prisma));
        },
        routineDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            return deleteOneHelper(req.userId, input, RoutineModel(prisma));
        },
    }
}
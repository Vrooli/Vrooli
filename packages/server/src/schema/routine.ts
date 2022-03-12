import { gql } from 'apollo-server-express';
import { IWrap, RecursivePartial } from '../types';
import { DeleteOneInput, FindByIdInput, Routine, RoutineCountInput, RoutineCreateInput, RoutineUpdateInput, RoutineSearchInput, Success, RoutineSearchResult, RoutineSortBy } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { countHelper, createHelper, deleteOneHelper, readManyHelper, readOneHelper, RoutineModel, updateHelper } from '../models';

export const typeDef = gql`
    enum RoutineSortBy {
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
        isAutomatable: Boolean
        isComplete: Boolean
        isInternal: Boolean
        version: String
        parentId: ID
        projectId: ID
        createdByUserId: ID
        createdByOrganizationId: ID
        nodesCreate: [NodeCreateInput!]
        nodeLinksCreate: [NodeLinkCreateInput!]
        inputsCreate: [InputItemCreateInput!]
        outputsCreate: [OutputItemCreateInput!]
        resourceListsCreate: [ResourceListCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [RoutineTranslationCreateInput!]
    }
    input RoutineUpdateInput {
        id: ID!
        isAutomatable: Boolean
        isComplete: Boolean
        isInternal: Boolean
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
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
        tagsCreate: [TagCreateInput!]
        translationsDelete: [ID!]
        translationsCreate: [RoutineTranslationCreateInput!]
        translationsUpdate: [RoutineTranslationUpdateInput!]
    }
    type Routine {
        id: ID!
        completedAt: Date
        complexity: Int!
        created_at: Date!
        updated_at: Date!
        isAutomatable: Boolean
        isComplete: Boolean!
        isInternal: Boolean
        isStarred: Boolean!
        role: MemberRole
        isUpvoted: Boolean
        score: Int!
        simplicity: Int!
        stars: Int!
        version: String
        comments: [Comment!]!
        creator: Contributor
        forks: [Routine!]!
        inputs: [InputItem!]!
        nodeLists: [NodeRoutineList!]!
        nodes: [Node!]!
        nodesCount: Int
        nodeLinks: [NodeLink!]!
        outputs: [OutputItem!]!
        owner: Contributor
        parent: Routine
        project: Project
        reports: [Report!]!
        resourceLists: [ResourceList!]!
        starredBy: [User!]!
        tags: [Tag!]!
        translations: [RoutineTranslation!]!
    }

    input RoutineTranslationCreateInput {
        language: String!
        description: String
        instructions: String!
        title: String!
    }
    input RoutineTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        instructions: String
        title: String
    }
    type RoutineTranslation {
        id: ID!
        language: String!
        description: String
        instructions: String!
        title: String!
    }

    input InputItemCreateInput {
        isRequired: Boolean
        name: String
        standardConnect: ID
        standardCreate: StandardCreateInput
        translationsDelete: [ID!]
        translationsCreate: [InputItemTranslationCreateInput!]
        translationsUpdate: [InputItemTranslationUpdateInput!]
    }
    input InputItemUpdateInput {
        id: ID!
        isRequired: Boolean
        name: String
        standardConnect: ID
        standardCreate: StandardCreateInput
        translationsDelete: [ID!]
        translationsCreate: [InputItemTranslationCreateInput!]
        translationsUpdate: [InputItemTranslationUpdateInput!]
    }
    type InputItem {
        id: ID!
        isRequired: Boolean
        name: String
        routine: Routine!
        standard: Standard
        translations: [InputItemTranslation!]!
    }

    input InputItemTranslationCreateInput {
        language: String!
        description: String
    }
    input InputItemTranslationUpdateInput {
        id: ID!
        language: String
        description: String
    }
    type InputItemTranslation {
        id: ID!
        language: String!
        description: String
    }

    input OutputItemCreateInput {
        name: String
        standardConnect: ID
        standardCreate: StandardCreateInput
        translationsCreate: [OutputItemTranslationCreateInput!]
    }
    input OutputItemUpdateInput {
        id: ID!
        name: String
        standardConnect: ID
        standardCreate: StandardCreateInput
        translationsDelete: [ID!]
        translationsCreate: [OutputItemTranslationCreateInput!]
        translationsUpdate: [OutputItemTranslationUpdateInput!]
    }
    type OutputItem {
        id: ID!
        name: String
        routine: Routine!
        standard: Standard
        translations: [OutputItemTranslation!]!
    }

    input OutputItemTranslationCreateInput {
        language: String!
        description: String
    }
    input OutputItemTranslationUpdateInput {
        id: ID!
        language: String
        description: String
    }
    type OutputItemTranslation {
        id: ID!
        language: String!
        description: String
    }

    input RoutineSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        isComplete: Boolean
        languages: [String!]
        minComplexity: Int
        maxComplexity: Int
        minSimplicity: Int
        maxSimplicity: Int
        minScore: Int
        minStars: Int
        organizationId: ID
        projectId: ID
        parentId: ID
        reportId: ID
        resourceLists: [String!]
        resourceTypes: [ResourceUsedFor!]
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
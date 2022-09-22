import { gql } from 'apollo-server-express';
import { IWrap, RecursivePartial } from '../types';
import { DeleteOneInput, FindByIdInput, Routine, RoutineCountInput, RoutineCreateInput, RoutineUpdateInput, RoutineSearchInput, Success, RoutineSearchResult, RoutineSortBy } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { countHelper, createHelper, deleteOneHelper, readManyHelper, readOneHelper, RoutineModel, updateHelper, visibilityBuilder } from '../models';
import { rateLimit } from '../rateLimit';

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
        id: ID!
        isAutomatable: Boolean
        isComplete: Boolean
        isInternal: Boolean
        isPrivate: Boolean
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
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [RoutineTranslationCreateInput!]
    }
    input RoutineUpdateInput {
        id: ID!
        isAutomatable: Boolean
        isComplete: Boolean
        isInternal: Boolean
        isPrivate: Boolean
        version: String
        userId: ID
        organizationId: ID
        projectId: ID
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
        tagsConnect: [String!]
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
        isDeleted: Boolean!
        isInternal: Boolean
        isPrivate: Boolean!
        isStarred: Boolean!
        isUpvoted: Boolean
        isViewed: Boolean!
        score: Int!
        simplicity: Int!
        stars: Int!
        views: Int!
        version: String!
        versionGroupId: ID!
        comments: [Comment!]!
        commentsCount: Int!
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
        permissionsRoutine: RoutinePermission!
        project: Project
        reports: [Report!]!
        reportsCount: Int!
        resourceLists: [ResourceList!]!
        runs: [Run!]!
        starredBy: [User!]!
        tags: [Tag!]!
        translations: [RoutineTranslation!]!
    }

    type RoutinePermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canFork: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canRun: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input RoutineTranslationCreateInput {
        id: ID!
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
        id: ID!
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
        id: ID!
        language: String!
        description: String
        helpText: String
    }
    input InputItemTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        helpText: String
    }
    type InputItemTranslation {
        id: ID!
        language: String!
        description: String
        helpText: String
    }

    input OutputItemCreateInput {
        id: ID!
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
        id: ID!
        language: String!
        description: String
        helpText: String
    }
    input OutputItemTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        helpText: String
    }
    type OutputItemTranslation {
        id: ID!
        language: String!
        description: String
        helpText: String
    }

    input RoutineSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        isComplete: Boolean
        isCompleteExceptions: [SearchException!]
        isInternal: Boolean
        isInternalExceptions: [SearchException!]
        languages: [String!]
        minComplexity: Int
        maxComplexity: Int
        minSimplicity: Int
        maxSimplicity: Int
        maxTimesCompleted: Int
        minScore: Int
        minStars: Int
        minTimesCompleted: Int
        minViews: Int
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
        visibility: VisibilityType
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
            await rateLimit({ info, max: 1000, req });
            return readOneHelper({ info, input, model: RoutineModel, prisma, userId: req.userId });
        },
        routines: async (_parent: undefined, { input }: IWrap<RoutineSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RoutineSearchResult> => {
            await rateLimit({ info, max: 1000, req });
            return readManyHelper({ info, input, model: RoutineModel, prisma, userId: req.userId });
        },
        routinesCount: async (_parent: undefined, { input }: IWrap<RoutineCountInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ info, max: 1000, req });
            return countHelper({ input, model: RoutineModel, prisma, userId: req.userId });
        },
    },
    Mutation: {
        routineCreate: async (_parent: undefined, { input }: IWrap<RoutineCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            await rateLimit({ info, max: 500, byAccountOrKey: true, req });
            return createHelper({ info, input, model: RoutineModel, prisma, userId: req.userId });
        },
        routineUpdate: async (_parent: undefined, { input }: IWrap<RoutineUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            await rateLimit({ info, max: 1000, byAccountOrKey: true, req });
            return updateHelper({ info, input, model: RoutineModel, prisma, userId: req.userId });
        },
        routineDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Success> => {
            await rateLimit({ info, max: 250, byAccountOrKey: true, req });
            return deleteOneHelper({ input, model: RoutineModel, prisma, userId: req.userId });
        },
    }
}
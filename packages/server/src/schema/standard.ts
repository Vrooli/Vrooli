import { gql } from 'apollo-server-express';
import { countHelper, createHelper, readManyHelper, readOneHelper, StandardModel, updateHelper } from '../models';
import { IWrap, RecursivePartial } from '../types';
import { Standard, StandardCountInput, StandardCreateInput, StandardUpdateInput, StandardSearchInput, StandardSearchResult, StandardSortBy, FindByVersionInput } from './types';
import { Context, rateLimit } from '../middleware';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    enum StandardSortBy {
        CommentsAsc
        CommentsDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
        VotesAsc
        VotesDesc
    }

    input StandardCreateInput {
        id: ID!
        default: String
        isInternal: Boolean
        isPrivate: Boolean
        name: String
        type: String!
        props: String!
        yup: String
        versionLabel: String
        createdByUserId: ID
        createdByOrganizationId: ID
        resourceListsCreate: [ResourceListCreateInput!]
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [StandardTranslationCreateInput!]
    }
    input StandardUpdateInput {
        id: ID!
        makeAnonymous: Boolean
        isPrivate: Boolean
        userId: ID
        organizationId: ID
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsDelete: [ID!]
        translationsCreate: [StandardTranslationCreateInput!]
        translationsUpdate: [StandardTranslationUpdateInput!]
        versionLabel: String! # Cannot update existing version, so we must pass in a new version label
    }
    type Standard {
        id: ID!
        created_at: Date!
        updated_at: Date!
        default: String
        name: String!
        isDeleted: Boolean!
        isInternal: Boolean!
        isPrivate: Boolean!
        isStarred: Boolean!
        isUpvoted: Boolean
        isViewed: Boolean!
        type: String!
        props: String!
        yup: String
        versionLabel: String!
        rootId: ID!
        versions: [Version!]!
        score: Int!
        stars: Int!
        views: Int!
        comments: [Comment!]!
        commentsCount: Int!
        creator: Contributor
        permissionsStandard: StandardPermission!
        reports: [Report!]!
        reportsCount: Int!
        resourceLists: [ResourceList!]!
        routineInputs: [Routine!]!
        routineOutputs: [Routine!]!
        starredBy: [User!]!
        tags: [Tag!]!
        translations: [StandardTranslation!]!
    }

    type StandardPermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input StandardTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        jsonVariable: String
    }
    input StandardTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        jsonVariable: String
    }
    type StandardTranslation {
        id: ID!
        language: String!
        description: String
        jsonVariable: String
    }

    input StandardSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        languages: [String!]
        minScore: Int
        minStars: Int
        minViews: Int
        organizationId: ID
        projectId: ID
        reportId: ID
        routineId: ID
        searchString: String
        sortBy: StandardSortBy
        tags: [String!]
        take: Int
        type: String
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }

    type StandardSearchResult {
        pageInfo: PageInfo!
        edges: [StandardEdge!]!
    }

    type StandardEdge {
        cursor: String!
        node: Standard!
    }

    # Input for count
    input StandardCountInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
    }

    extend type Query {
        standard(input: FindByVersionInput!): Standard
        standards(input: StandardSearchInput!): StandardSearchResult!
        standardsCount(input: StandardCountInput!): Int!
    }

    extend type Mutation {
        standardCreate(input: StandardCreateInput!): Standard!
        standardUpdate(input: StandardUpdateInput!): Standard!
    }
`

export const resolvers = {
    StandardSortBy: StandardSortBy,
    Query: {
        standard: async (_parent: undefined, { input }: IWrap<FindByVersionInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard> | null> => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, model: StandardModel, prisma, req });
        },
        standards: async (_parent: undefined, { input }: IWrap<StandardSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<StandardSearchResult> => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, model: StandardModel, prisma, req });
        },
        standardsCount: async (_parent: undefined, { input }: IWrap<StandardCountInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ info, maxUser: 1000, req });
            return countHelper({ input, model: StandardModel, prisma, req });
        },
    },
    Mutation: {
        standardCreate: async (_parent: undefined, { input }: IWrap<StandardCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard>> => {
            await rateLimit({ info, maxUser: 250, req });
            return createHelper({ info, input, objectType: 'Standard', prisma, req });
        },
        standardUpdate: async (_parent: undefined, { input }: IWrap<StandardUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard>> => {
            await rateLimit({ info, maxUser: 500, req });
            return updateHelper({ info, input, objectType: 'Standard', prisma, req });
        },
    }
}
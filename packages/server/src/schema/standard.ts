import { gql } from 'apollo-server-express';
import { countHelper, createHelper, readManyHelper, readOneHelper, StandardModel, updateHelper, visibilityBuilder } from '../models';
import { IWrap, RecursivePartial } from '../types';
import { FindByIdInput, Standard, StandardCountInput, StandardCreateInput, StandardUpdateInput, StandardSearchInput, StandardSearchResult, StandardSortBy, FindByVersionInput } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';

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
        version: String
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
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsDelete: [ID!]
        translationsCreate: [StandardTranslationCreateInput!]
        translationsUpdate: [StandardTranslationUpdateInput!]
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
        version: String!
        versionGroupId: ID!
        versions: [String!]!
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
            await rateLimit({ info, max: 1000, req });
            return readOneHelper({ info, input, model: StandardModel, prisma, userId: req.userId });
        },
        standards: async (_parent: undefined, { input }: IWrap<StandardSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<StandardSearchResult> => {
            await rateLimit({ info, max: 1000, req });
            return readManyHelper({ info, input, model: StandardModel, prisma, userId: req.userId });
        },
        standardsCount: async (_parent: undefined, { input }: IWrap<StandardCountInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ info, max: 1000, req });
            return countHelper({ input, model: StandardModel, prisma, userId: req.userId });
        },
    },
    Mutation: {
        /**
         * Create a new standard
         * @returns Standard object if successful
         */
        standardCreate: async (_parent: undefined, { input }: IWrap<StandardCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard>> => {
            await rateLimit({ info, max: 250, byAccountOrKey: true, req });
            return createHelper({ info, input, model: StandardModel, prisma, userId: req.userId });
        },
        /**
         * Update a standard you created.
         * NOTE: You can only update the description and tags. If you need to update 
         * the other fields, you must either create a new standard (could be the same but with an updated
         * version number) or delete the old one and create a new one.
         * @returns Standard object if successful
         */
        standardUpdate: async (_parent: undefined, { input }: IWrap<StandardUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard>> => {
            await rateLimit({ info, max: 500, byAccountOrKey: true, req });
            return updateHelper({ info, input, model: StandardModel, prisma, userId: req.userId });
        },
    }
}
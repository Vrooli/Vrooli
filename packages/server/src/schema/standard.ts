import { gql } from 'apollo-server-express';
import { countHelper, createHelper, readManyHelper, readOneHelper, StandardModel, updateHelper } from '../models';
import { IWrap, RecursivePartial } from '../types';
import { FindByIdInput, Standard, StandardCountInput, StandardCreateInput, StandardUpdateInput, StandardSearchInput, StandardSearchResult, StandardSortBy } from './types';
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
        id: ID
        default: String
        name: String
        type: String!
        props: String!
        yup: String
        version: String
        createdByUserId: ID
        createdByOrganizationId: ID
        resourceListsCreate: [ResourceListCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [StandardTranslationCreateInput!]
    }
    input StandardUpdateInput {
        id: ID!
        makeAnonymous: Boolean
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
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
        isStarred: Boolean!
        role: MemberRole
        isUpvoted: Boolean
        isViewed: Boolean!
        type: String!
        props: String!
        yup: String
        version: String!
        score: Int!
        stars: Int!
        views: Int!
        comments: [Comment!]!
        commentsCount: Int!
        creator: Contributor
        reports: [Report!]!
        reportsCount: Int!
        resourceLists: [ResourceList!]!
        routineInputs: [Routine!]!
        routineOutputs: [Routine!]!
        starredBy: [User!]!
        tags: [Tag!]!
        translations: [StandardTranslation!]!
    }

    input StandardTranslationCreateInput {
        id: ID
        language: String!
        description: String
        jsonVariables: String
    }
    input StandardTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        jsonVariables: String
    }
    type StandardTranslation {
        id: ID!
        language: String!
        description: String
        jsonVariables: String
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
    }

    # Return type for search result
    type StandardSearchResult {
        pageInfo: PageInfo!
        edges: [StandardEdge!]!
    }

    # Return type for search result edge
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
        standard(input: FindByIdInput!): Standard
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
        standard: async (_parent: undefined, { input }: IWrap<FindByIdInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard> | null> => {
            await rateLimit({ context, info, max: 1000 });
            return readOneHelper(context.req.userId, input, info, StandardModel(context.prisma));
        },
        standards: async (_parent: undefined, { input }: IWrap<StandardSearchInput>, context: Context, info: GraphQLResolveInfo): Promise<StandardSearchResult> => {
            await rateLimit({ context, info, max: 1000 });
            return readManyHelper(context.req.userId, input, info, StandardModel(context.prisma));
        },
        standardsCount: async (_parent: undefined, { input }: IWrap<StandardCountInput>, context: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ context, info, max: 1000 });
            return countHelper(input, StandardModel(context.prisma));
        },
    },
    Mutation: {
        /**
         * Create a new standard
         * @returns Standard object if successful
         */
        standardCreate: async (_parent: undefined, { input }: IWrap<StandardCreateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard>> => {
            await rateLimit({ context, info, max: 250, byAccount: true });
            return createHelper(context.req.userId, input, info, StandardModel(context.prisma));
        },
        /**
         * Update a standard you created.
         * NOTE: You can only update the description and tags. If you need to update 
         * the other fields, you must either create a new standard (could be the same but with an updated
         * version number) or delete the old one and create a new one.
         * @returns Standard object if successful
         */
        standardUpdate: async (_parent: undefined, { input }: IWrap<StandardUpdateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Standard>> => {
            await rateLimit({ context, info, max: 500, byAccount: true });
            return updateHelper(context.req.userId, input, info, StandardModel(context.prisma));
        },
    }
}
import { gql } from 'apollo-server-express';
import { Comment, CommentCountInput, CommentCreateInput, CommentFor, CommentSearchInput, CommentSearchResult, CommentSortBy, CommentUpdateInput, FindByIdInput } from './types';
import { IWrap, RecursivePartial } from 'types';
import { CommentModel, countHelper, createHelper, readManyHelper, readOneHelper, updateHelper } from '../models';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';
import { resolveCommentedOn } from './resolvers';

export const typeDef = gql`
    enum CommentFor {
        Project
        Routine
        Standard
    }   

    enum CommentSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
        VotesAsc
        VotesDesc
    }

    input CommentCreateInput {
        id: ID
        createdFor: CommentFor!
        forId: ID!
        translationsCreate: [CommentTranslationCreateInput!]
    }
    input CommentUpdateInput {
        id: ID!
        translationsDelete: [ID!]
        translationsCreate: [CommentTranslationCreateInput!]
        translationsUpdate: [CommentTranslationUpdateInput!]
    }

    union CommentedOn = Project | Routine | Standard

    type Comment {
        id: ID!
        created_at: Date!
        updated_at: Date!
        commentedOn: CommentedOn!
        creator: Contributor
        isStarred: Boolean!
        isUpvoted: Boolean
        reports: [Report!]!
        reportsCount: Int!
        role: MemberRole
        score: Int!
        stars: Int!
        starredBy: [User!]
        translations: [CommentTranslation!]!
    }

    input CommentTranslationCreateInput {
        id: ID
        language: String!
        text: String!
    }
    input CommentTranslationUpdateInput {
        id: ID!
        language: String
        text: String
    }
    type CommentTranslation {
        id: ID!
        language: String!
        text: String!
    }

    input CommentSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        languages: [String!]
        minScore: Int
        minStars: Int
        organizationId: ID
        projectId: ID
        routineId: ID
        searchString: String
        sortBy: CommentSortBy
        standardId: ID
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
    }

    type CommentThread {
        childThreads: [CommentThread!]!
        comment: Comment!
        endCursor: String
        totalInThread: Int
    }

    type CommentSearchResult {
        endCursor: String
        threads: [CommentThread!]
        totalThreads: Int
    }

    # Input for count
    input CommentCountInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
    }

    extend type Query {
        comment(input: FindByIdInput!): Comment
        comments(input: CommentSearchInput!): CommentSearchResult!
        commentsCount(input: CommentCountInput!): Int!
    }

    extend type Mutation {
        commentCreate(input: CommentCreateInput!): Comment!
        commentUpdate(input: CommentUpdateInput!): Comment!
    }
`

export const resolvers = {
    CommentFor: CommentFor,
    CommentSortBy: CommentSortBy,
    CommentedOn: {
        __resolveType(obj: any) { return resolveCommentedOn(obj) },
    },
    Query: {
        comment: async (_parent: undefined, { input }: IWrap<FindByIdInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment> | null> => {
            await rateLimit({ context, info, max: 1000 });
            return readOneHelper(context.req.userId, input, info, CommentModel(context.prisma));
        },
        comments: async (_parent: undefined, { input }: IWrap<CommentSearchInput>, context: Context, info: GraphQLResolveInfo): Promise<CommentSearchResult> => {
            await rateLimit({ context, info, max: 1000 });
            return CommentModel(context.prisma).searchNested(context.req.userId, input, info);
        },
        commentsCount: async (_parent: undefined, { input }: IWrap<CommentCountInput>, context: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ context, info, max: 1000 });
            return countHelper(input, CommentModel(context.prisma));
        },
    },
    Mutation: {
        commentCreate: async (_parent: undefined, { input }: IWrap<CommentCreateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            await rateLimit({ context, info, max: 250, byAccount: true });
            return createHelper(context.req.userId, input, info, CommentModel(context.prisma));
        },
        commentUpdate: async (_parent: undefined, { input }: IWrap<CommentUpdateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            await rateLimit({ context, info, max: 1000, byAccount: true });
            return updateHelper(context.req.userId, input, info, CommentModel(context.prisma));
        },
    }
}
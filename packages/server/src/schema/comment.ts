import { gql } from 'apollo-server-express';
import { Comment, CommentCountInput, CommentCreateInput, CommentFor, CommentSearchInput, CommentSearchResult, CommentSortBy, CommentUpdateInput, FindByIdInput } from './types';
import { IWrap, RecursivePartial } from '../types';
import { CommentModel, countHelper, createHelper, readOneHelper, updateHelper } from '../models';
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
        id: ID!
        createdFor: CommentFor!
        forId: ID!
        parentId: ID
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
        permissionsComment: CommentPermission
        reports: [Report!]!
        reportsCount: Int!
        score: Int!
        stars: Int!
        starredBy: [User!]
        translations: [CommentTranslation!]!
    }

    type CommentPermission {
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReply: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input CommentTranslationCreateInput {
        id: ID!
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
        comment: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment> | null> => {
            await rateLimit({ info, max: 1000, req });
            return readOneHelper({ info, input, model: CommentModel, prisma, userId: req.userId })
        },
        comments: async (_parent: undefined, { input }: IWrap<CommentSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<CommentSearchResult> => {
            await rateLimit({ info, max: 1000, req });
            return CommentModel.query(prisma).searchNested(req.userId, input, info);
        },
        commentsCount: async (_parent: undefined, { input }: IWrap<CommentCountInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ info, max: 1000, req });
            return countHelper({ input, model: CommentModel, prisma, userId: req.userId })
        },
    },
    Mutation: {
        commentCreate: async (_parent: undefined, { input }: IWrap<CommentCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            await rateLimit({ info, max: 250, byAccountOrKey: true, req });
            return createHelper({ info, input, model: CommentModel, prisma, userId: req.userId })
        },
        commentUpdate: async (_parent: undefined, { input }: IWrap<CommentUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            await rateLimit({ info, max: 1000, byAccountOrKey: true, req });
            return updateHelper({ info, input, model: CommentModel, prisma, userId: req.userId })
        },
    }
}
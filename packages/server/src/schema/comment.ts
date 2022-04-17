import { gql } from 'apollo-server-express';
import { Comment, CommentCountInput, CommentCreateInput, CommentFor, CommentSearchInput, CommentSearchResult, CommentSortBy, CommentUpdateInput, DeleteOneInput, FindByIdInput, Success } from './types';
import { IWrap, RecursivePartial } from 'types';
import { CommentModel, countHelper, createHelper, deleteOneHelper, GraphQLModelType, readManyHelper, readOneHelper, updateHelper } from '../models';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';

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
        role: MemberRole
        score: Int
        stars: Int
        starredBy: [User!]
        translations: [CommentTranslation!]!
    }

    input CommentTranslationCreateInput {
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
        ids: [ID!]
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

    # Return type for search result
    type CommentSearchResult {
        pageInfo: PageInfo!
        edges: [CommentEdge!]!
    }

    # Return type for search result edge
    type CommentEdge {
        cursor: String!
        node: Comment!
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
        commentDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    CommentFor: CommentFor,
    CommentSortBy: CommentSortBy,
    CommentedOn: {
        __resolveType(obj: any) {
            // Only a Standard has an isFile field
            if (obj.hasOwnProperty('isFile')) return GraphQLModelType.Standard;
            // Only a Project has a name field
            if (obj.hasOwnProperty('isComplete')) return GraphQLModelType.Project;
            return GraphQLModelType.Routine;
        },
    },
    Query: {
        comment: async (_parent: undefined, { input }: IWrap<FindByIdInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment> | null> => {
            await rateLimit({ context, info, max: 1000 });
            return readOneHelper(context.req.userId, input, info, CommentModel(context.prisma));
        },
        comments: async (_parent: undefined, { input }: IWrap<CommentSearchInput>, context: Context, info: GraphQLResolveInfo): Promise<CommentSearchResult> => {
            await rateLimit({ context, info, max: 1000 });
            return readManyHelper(context.req.userId, input, info, CommentModel(context.prisma));
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
        commentDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, context: Context, info: GraphQLResolveInfo): Promise<Success> => {
            await rateLimit({ context, info, max: 1000, byAccount: true });
            return deleteOneHelper(context.req.userId, input, CommentModel(context.prisma));
        },
    }
}
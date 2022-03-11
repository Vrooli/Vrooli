import { gql } from 'apollo-server-express';
import { Comment, CommentCountInput, CommentCreateInput, CommentFor, CommentSearchInput, CommentSearchResult, CommentSortBy, CommentUpdateInput, DeleteOneInput, FindByIdInput, Success } from './types';
import { IWrap, RecursivePartial } from 'types';
import { CommentModel, countHelper, createHelper, deleteOneHelper, GraphQLModelType, readManyHelper, readOneHelper, updateHelper } from '../models';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';

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
        comment: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment> | null> => {
            return readOneHelper(req.userId, input, info, CommentModel(prisma));
        },
        comments: async (_parent: undefined, { input }: IWrap<CommentSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<CommentSearchResult> => {
            return readManyHelper(req.userId, input, info, CommentModel(prisma));
        },
        commentsCount: async (_parent: undefined, { input }: IWrap<CommentCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            return countHelper(input, CommentModel(prisma));
        },
    },
    Mutation: {
        commentCreate: async (_parent: undefined, { input }: IWrap<CommentCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            return createHelper(req.userId, input, info, CommentModel(prisma));
        },
        commentUpdate: async (_parent: undefined, { input }: IWrap<CommentUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            return updateHelper(req.userId, input, info, CommentModel(prisma));
        },
        commentDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            return deleteOneHelper(req.userId, input, CommentModel(prisma));
        },
    }
}
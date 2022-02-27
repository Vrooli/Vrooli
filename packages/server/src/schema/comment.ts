import { gql } from 'apollo-server-express';
import { Comment, CommentCountInput, CommentCreateInput, CommentFor, CommentSearchInput, CommentSearchResult, CommentUpdateInput, DeleteOneInput, FindByIdInput, Success } from './types';
import { IWrap, RecursivePartial } from 'types';
import { CommentModel, countHelper, createHelper, deleteOneHelper, readManyHelper, readOneHelper, updateHelper } from '../models';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { CommentSortBy } from '@local/shared';

export const typeDef = gql`
    enum CommentFor {
        Project
        Routine
        Standard
    }   

    enum CommentSortBy {
        AlphabeticalAsc
        AlphabeticalDesc
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
        text: String!
        createdFor: CommentFor!
        forId: ID!
    }
    input CommentUpdateInput {
        id: ID!
        text: String
    }

    union CommentedOn = Project | Routine | Standard

    type Comment {
        id: ID!
        text: String
        created_at: Date!
        updated_at: Date!
        creator: Contributor
        commentedOn: CommentedOn!
        reports: [Report!]!
        stars: Int
        isStarred: Boolean!
        starredBy: [User!]
        score: Int
        isUpvoted: Boolean
        role: MemberRole
    }

    input CommentSearchInput {
        userId: ID
        organizationId: ID
        projectId: ID
        routineId: ID
        standardId: ID
        ids: [ID!]
        sortBy: CommentSortBy
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        searchString: String
        after: String
        take: Int
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
            console.log('IN COMMENT __resolveType', obj);
            // Only a Project has a name field
            if (obj.hasOwnProperty('name')) return 'Project';
            // Only a Routine has a title field
            if (obj.hasOwnProperty('title')) return 'Routine';
            // Only a Standard has an isFile field
            if (obj.hasOwnProperty('isFile')) return 'Standard';
            return null; // GraphQLError is thrown
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
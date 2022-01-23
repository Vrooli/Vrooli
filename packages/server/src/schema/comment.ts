import { gql } from 'apollo-server-express';
import { CODE, CommentFor } from '@local/shared';
import { CustomError } from '../error';
import { Comment, CommentInput, DeleteOneInput, Success } from './types';
import { IWrap, RecursivePartial } from 'types';
import { CommentModel } from '../models';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    enum CommentFor {
        Organization
        Project
        Routine
        Standard
        User
    }   

    input CommentInput {
        id: ID
        text: String
        createdFor: CommentFor!
        forId: ID!
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
        isStarred: Boolean
        starredBy: [User!]
        score: Int
        isUpvoted: Boolean
    }

    input DeleteCommentInput {
        id: ID!
        createdFor: CommentFor!
        forId: ID!
    }

    extend type Mutation {
        commentAdd(input: CommentInput!): Comment!
        commentUpdate(input: CommentInput!): Comment!
        commentDeleteOne(input: DeleteCommentInput!): Success!
    }
`

export const resolvers = {
    CommentFor: CommentFor,
    Mutation: {
        commentAdd: async (_parent: undefined, { input }: IWrap<CommentInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Create object
            const created = await CommentModel(prisma).addComment(req.userId, input, info);
            if (!created) throw new CustomError(CODE.ErrorUnknown);
            return created;
        },
        commentUpdate: async (_parent: undefined, { input }: IWrap<CommentInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Update object
            const updated = await CommentModel(prisma).updateComment(req.userId, input, info);
            if (!updated) throw new CustomError(CODE.ErrorUnknown);
            return updated;
        },
        commentDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Delete object
            return await CommentModel(prisma).deleteComment(req.userId, input);
        },
    }
}
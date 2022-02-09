import { gql } from 'apollo-server-express';
import { CODE, CommentFor } from '@local/shared';
import { CustomError } from '../error';
import { Comment, CommentCreateInput, CommentUpdateInput, DeleteOneInput, Success } from './types';
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
        isStarred: Boolean
        starredBy: [User!]
        score: Int
        isUpvoted: Boolean
    }

    extend type Mutation {
        commentCreate(input: CommentCreateInput!): Comment!
        commentUpdate(input: CommentUpdateInput!): Comment!
        commentDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    CommentFor: CommentFor,
    Mutation: {
        commentCreate: async (_parent: undefined, { input }: IWrap<CommentCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            // Create object
            const created = await CommentModel(prisma).create(req.userId, input, info);
            if (!created) throw new CustomError(CODE.ErrorUnknown);
            return created;
        },
        commentUpdate: async (_parent: undefined, { input }: IWrap<CommentUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            // Update object
            const updated = await CommentModel(prisma).update(req.userId, input, info);
            if (!updated) throw new CustomError(CODE.ErrorUnknown);
            return updated;
        },
        commentDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            // Delete object
            return await CommentModel(prisma).delete(req.userId, input);
        },
    }
}
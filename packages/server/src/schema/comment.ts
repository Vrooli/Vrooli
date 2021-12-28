import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { Comment, CommentInput, DeleteOneInput, ReportInput, Success, VoteInput } from './types';
import { IWrap, RecursivePartial } from 'types';
import { CommentModel } from '../models';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    input CommentInput {
        id: ID
        text: String
        objectType: String
        objectId: ID
    }

    type Comment {
        id: ID!
        text: String
        created_at: Date!
        updated_at: Date!
        userId: ID
        user: User
        organizationId: ID
        organization: Organization
        projectId: ID
        project: Project
        resourceId: ID
        resource: Resource
        routineId: ID
        routine: Routine
        standardId: ID
        standard: Standard
        reports: [Report!]!
        stars: Int
        vote: Int
    }

    input VoteInput {
        id: ID!
        isUpvote: Boolean!
    }

    extend type Mutation {
        commentAdd(input: CommentInput!): Comment!
        commentUpdate(input: CommentInput!): Comment!
        commentDeleteOne(input: DeleteOneInput!): Success!
        commentReport(input: ReportInput!): Success!
        commentVote(input: VoteInput!): Success!
    }
`

export const resolvers = {
    Mutation: {
        commentAdd: async (_parent: undefined, { input }: IWrap<CommentInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await CommentModel(prisma).create(input, info);
        },
        commentUpdate: async (_parent: undefined, { input }: IWrap<CommentInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            // Must be logged in
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Validate input
            if (!input.id) throw new CustomError(CODE.InvalidArgs);
            const authenticated = await CommentModel(prisma).isAuthenticatedToModify(input.id, req.userId);
            if (!authenticated) throw new CustomError(CODE.Unauthorized);
            // Update and return info
            return await CommentModel(prisma).update(input, info);
        },
        commentDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Validate input
            if (!input.id) throw new CustomError(CODE.InvalidArgs);
            const authenticated = await CommentModel(prisma).isAuthenticatedToModify(input.id, req.userId);
            if (!authenticated) throw new CustomError(CODE.Unauthorized);
            // Delete and return result
            const success = await CommentModel(prisma).delete(input);
            return { success };
        },
        /**
         * Reports a comment. After enough reports, the comment will be deleted.
         * @returns True if report was successfully recorded
         */
        commentReport: async (_parent: undefined, { input }: IWrap<ReportInput>, context: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!context.req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            const success = await CommentModel(context.prisma).report(input);
            return { success };
        },
        commentVote: async (_parent: undefined, { input }: IWrap<VoteInput>, { prisma, req}: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Validate input
            if (!input.id) throw new CustomError(CODE.InvalidArgs);
            const authenticated = await CommentModel(prisma).isAuthenticatedToVote(input.id, req.userId);
            if (!authenticated) throw new CustomError(CODE.Unauthorized);
            // Vote and return result
            const success = await CommentModel(prisma).vote(input, req.userId);
            return { success };
        }
    }
}
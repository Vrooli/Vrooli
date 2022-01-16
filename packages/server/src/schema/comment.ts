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

    extend type Mutation {
        commentAdd(input: CommentInput!): Comment!
        commentUpdate(input: CommentInput!): Comment!
        commentDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    CommentFor: CommentFor,
    Mutation: {
        commentAdd: async (_parent: undefined, { input }: IWrap<CommentInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // Create object
            const dbModel = await CommentModel(prisma).create(input, info);
            // Format object to GraphQL type
            return CommentModel().toGraphQL(dbModel);
        },
        commentUpdate: async (_parent: undefined, { input }: IWrap<CommentInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            // Must be logged in
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Validate input
            if (!input.id) throw new CustomError(CODE.InvalidArgs);
            const authenticated = await CommentModel(prisma).isAuthenticatedToModify(input.id, req.userId);
            if (!authenticated) throw new CustomError(CODE.Unauthorized);
            // Update object
            const dbModel = await CommentModel(prisma).update(input, info);
            // Format to GraphQL type
            return CommentModel().toGraphQL(dbModel);
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
    }
}
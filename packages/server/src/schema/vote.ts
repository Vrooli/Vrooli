import { gql } from 'apollo-server-express';
import { CODE, VoteFor } from '@local/shared';
import { CustomError } from '../error';
import { DeleteOneInput, Vote, VoteInput, Success } from './types';
import { IWrap, RecursivePartial } from 'types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { VoteModel } from '../models';

export const typeDef = gql`
    enum VoteFor {
        Comment
        Project
        Routine
        Standard
        Tag
    }   

    input VoteInput {
        id: ID
        isUpvote: Boolean!
        createdFor: VoteFor!
        forId: ID!
    }

    type Vote {
        id: ID
        isUpvote: Boolean!
    }

    extend type Mutation {
        voteAdd(input: VoteInput!): Vote!
        voteUpdate(input: VoteInput!): Vote!
        voteDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    VoteFor: VoteFor,
    Mutation: {
        voteAdd: async (_parent: undefined, { input }: IWrap<VoteInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Vote>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // Create object
            const dbModel = await VoteModel(prisma).create(input, info);
            // Format object to GraphQL type
            return VoteModel().toGraphQL(dbModel);
        },
        voteUpdate: async (_parent: undefined, { input }: IWrap<VoteInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Vote>> => {
            // Must be logged in
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Validate input
            //TODO
            // Update object
            const dbModel = await VoteModel(prisma).update(input, info);
            // Format to GraphQL type
            return VoteModel().toGraphQL(dbModel);
        },
        voteDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Validate input
            //TODO
            // Delete and return result
            const success = await VoteModel(prisma).delete(input);
            return { success };
        },
    }
}
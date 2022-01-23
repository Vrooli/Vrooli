import { gql } from 'apollo-server-express';
import { CODE, StarFor } from '@local/shared';
import { CustomError } from '../error';
import { StarInput, Success } from './types';
import { IWrap } from 'types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { StarModel } from '../models';

export const typeDef = gql`
    enum StarFor {
        Comment
        Organization
        Project
        Routine
        Standard
        Tag
        User
    }   

    input StarInput {
        isStar: Boolean!
        starFor: StarFor!
        forId: ID!
    }

    extend type Mutation {
        star(input: StarInput!): Success!
    }
`

export const resolvers = {
    StarFor: StarFor,
    Mutation: {
        /**
         * Adds or removes a star to an object. A user can only star an object once.
         * @returns 
         */
        star: async (_parent: undefined, { input }: IWrap<StarInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            const success = await StarModel(prisma).star(req.userId, input);
            return { success };
        },
    }
}
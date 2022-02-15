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

    union Stars = Comment | Organization | Project | Routine | Standard | Tag

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
    Stars: {
        __resolveType(obj: any) {
            console.log('IN STAR __resolveType', obj);
            console.log('IN Contributor __resolveType', obj);
            // Only an Organization has a name and bio field
            if (obj.hasOwnProperty('name') && obj.hasOwnProperty('bio')) return 'Organization';
            // Only a Project has a name and description field
            if (obj.hasOwnProperty('name') && obj.hasOwnProperty('description')) return 'Project';
            // Only a Routine has a title and description field
            if (obj.hasOwnProperty('title') && obj.hasOwnProperty('description')) return 'Routine';
            // Only a Standard has an isFile field
            if (obj.hasOwnProperty('isFile')) return 'Standard';
            // Only a user has a username field
            if (obj.hasOwnProperty('username')) return 'User';
            return null; // GraphQLError is thrown
        },
    },
    Mutation: {
        /**
         * Adds or removes a star to an object. A user can only star an object once.
         * @returns 
         */
        star: async (_parent: undefined, { input }: IWrap<StarInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            const success = await StarModel(prisma).star(req.userId, input);
            return { success };
        },
    }
}
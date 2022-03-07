import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { StarFor, StarInput, Success } from './types';
import { IWrap } from 'types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { GraphQLModelType, StarModel } from '../models';

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

    union StarTo = Comment | Organization | Project | Routine | Standard | Tag

    input StarInput {
        isStar: Boolean!
        starFor: StarFor!
        forId: ID!
    }
    type Star {
        from: User!
        to: StarTo!
    }

    extend type Mutation {
        star(input: StarInput!): Success!
    }
`

export const resolvers = {
    StarFor: StarFor,
    Star: {
        __resolveType(obj: any) {
            if (obj.hasOwnProperty('isFile')) return GraphQLModelType.Standard;
            if (obj.hasOwnProperty('username')) return GraphQLModelType.User;
            if (obj.hasOwnProperty('isComplete')) return GraphQLModelType.Project;
            if (obj.hasOwnProperty('isOpenToNewMembers')) return GraphQLModelType.Organization;
            return GraphQLModelType.Routine;
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
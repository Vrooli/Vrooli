import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { StarFor, StarInput, Success } from './types';
import { IWrap } from 'types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { GraphQLModelType, StarModel } from '../models';
import { rateLimit } from '../rateLimit';
import { genErrorCode } from '../logger';

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
            if (obj.hasOwnProperty('isComplete')) return GraphQLModelType.Project;
            if (obj.hasOwnProperty('isOpenToNewMembers')) return GraphQLModelType.Organization;
            if (obj.hasOwnProperty('name')) return GraphQLModelType.User;
            return GraphQLModelType.Routine;
        },
    },
    Mutation: {
        /**
         * Adds or removes a star to an object. A user can only star an object once.
         * @returns 
         */
        star: async (_parent: undefined, { input }: IWrap<StarInput>, context: Context, info: GraphQLResolveInfo): Promise<Success> => {
            if (!context.req.userId) 
                throw new CustomError(CODE.Unauthorized, 'Must be logged in to star', { code: genErrorCode('0157') });
            await rateLimit({ context, info, max: 1000, byAccount: true });
            const success = await StarModel(context.prisma).star(context.req.userId, input);
            return { success };
        },
    }
}
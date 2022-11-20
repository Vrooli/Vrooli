import { gql } from 'apollo-server-express';
import { DeleteOneInput, Success } from './types';
import { IWrap } from '../types';
import { deleteOneHelper, ObjectMap } from '../models';
import { Context, rateLimit } from '../middleware';
import { GraphQLResolveInfo } from 'graphql';
import { CustomError } from '../events/error';
import { DeleteOneType } from '@shared/consts';

export const typeDef = gql`
    enum DeleteOneType {
        Comment
        Email
        Node
        Organization
        Project
        Report
        Routine
        Run
        Standard
        Wallet
    }   

    # Input for deleting one object
    input DeleteOneInput {
        id: ID!
        objectType: DeleteOneType!
    }

    extend type Mutation {
        deleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    DeleteOneType: DeleteOneType,
    Mutation: {
        deleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Success> => {
            await rateLimit({ info, maxUser: 1000, req });
            const model = ObjectMap[input.objectType];
            if (!model) throw new CustomError('InvalidArgs', 'Invalid delete object type.', { trace: '0216' });
            return deleteOneHelper({ input, model, prisma, req });
        },
    }
}
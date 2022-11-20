import { gql } from 'apollo-server-express';
import { ForkInput, ForkResult } from './types';
import { IWrap } from '../types';
import { forkHelper, lowercaseFirstLetter, ObjectMap } from '../models';
import { Context, rateLimit } from '../middleware';
import { GraphQLResolveInfo } from 'graphql';
import { CustomError } from '../events/error';
import { ForkType } from '@shared/consts';

export const typeDef = gql`

    enum ForkType {
        Organization
        Project
        Routine
        Standard
    }  
 
    input ForkInput {
        id: ID!
        intendToPullRequest: Boolean!
        objectType: ForkType!
    }

    type ForkResult {
        organization: Organization
        project: Project
        routine: Routine
        standard: Standard
    }
 
    extend type Mutation {
        fork(input: ForkInput!): ForkResult!
    }
 `

export const resolvers = {
    ForkType: ForkType,
    Mutation: {
        fork: async (_parent: undefined, { input }: IWrap<ForkInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<ForkResult> => {
            await rateLimit({ info, maxUser: 500, req });
            const model = ObjectMap[input.objectType];
            if (!model) throw new CustomError('InvalidArgs', 'Invalid fork object type.', { trace: '0228' });
            const result = await forkHelper({ info, input, model: model, prisma, req })
            return { [lowercaseFirstLetter(input.objectType)]: result };
        }
    }
}
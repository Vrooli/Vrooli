import { gql } from 'apollo-server-express';
import { ForkInput, ForkResult } from './types';
import { IWrap } from '../types';
import { Context, rateLimit } from '../middleware';
import { GraphQLResolveInfo } from 'graphql';
import { ForkType } from '@shared/consts';
import { forkHelper } from '../actions';
import { lowercaseFirstLetter } from '../builders';

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
            const result = await forkHelper({ info, input, objectType: input.objectType, prisma, req })
            return { [lowercaseFirstLetter(input.objectType)]: result };
        }
    }
}
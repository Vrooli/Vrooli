import { gql } from 'apollo-server-express';
import { ForkInput, ForkResult } from './types';
import { IWrap } from '../types';
import { forkHelper, lowercaseFirstLetter, ObjectMap } from '../models';
import { Context, rateLimit } from '../middleware';
import { GraphQLResolveInfo } from 'graphql';
import { CustomError } from '../events/error';
import { CODE, ForkType } from '@shared/consts';
import { genErrorCode } from '../events/logger';
import { GraphQLModelType, ModelLogic } from '../models/types';

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
        fork: async (_parent: undefined, { input }: IWrap<ForkInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<ForkResult> => {
            await rateLimit({ info, maxUser: 500, req });
            const validTypes: Array<keyof typeof ForkType> = [
                ForkType.Organization,
                ForkType.Project,
                ForkType.Routine,
                ForkType.Standard,
            ];
            if (!validTypes.includes(input.objectType as keyof typeof ForkType)) {
                throw new CustomError(CODE.InvalidArgs, 'Invalid fork object type.', { code: genErrorCode('0228') });
            }
            const model: ModelLogic<any, any, any> = ObjectMap[input.objectType as GraphQLModelType] as ModelLogic<any, any, any>;
            const result = await forkHelper({ info, input, model: model, prisma, req })
            return { [lowercaseFirstLetter(input.objectType)]: result };
        }
    }
}
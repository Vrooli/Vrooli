/**
 * Handles copying and forking of objects. 
 * Copying objects is useful when you want your own version, and do not care about changes to the original.
 * Forking is useful when you want to track changes and suggest changes to the original.
 */
import { gql } from 'apollo-server-express';
import { CopyInput, CopyResult, ForkInput, ForkResult } from './types';
import { IWrap } from '../types';
import { copyHelper, forkHelper, GraphQLModelType, lowercaseFirstLetter, ModelLogic, ObjectMap } from '../models';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';
import { CustomError } from '../error';
import { CODE, CopyType, ForkType } from '@local/shared';
import { genErrorCode } from '../logger';

export const typeDef = gql`
    enum CopyType {
        Node
        Organization
        Project
        Routine
        Standard
    }  

    enum ForkType {
        Organization
        Project
        Routine
        Standard
    }  
 
    input CopyInput {
        id: ID!
        objectType: CopyType!
    }

    input ForkInput {
        id: ID!
        objectType: ForkType!
    }

    type CopyResult {
        node: Node
        organization: Organization
        project: Project
        routine: Routine
        standard: Standard
    }

    type ForkResult {
        organization: Organization
        project: Project
        routine: Routine
        standard: Standard
    }
 
    extend type Mutation {
        copy(input: CopyInput!): CopyResult!
        fork(input: ForkInput!): ForkResult!
    }
 `

export const resolvers = {
    CopyType: CopyType,
    ForkType: ForkType,
    Mutation: {
        copy: async (_parent: undefined, { input }: IWrap<CopyInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<CopyResult> => {
            await rateLimit({ info, max: 500, byAccountOrKey: true, req });
            const validTypes: Array<keyof typeof CopyType> = [
                CopyType.Node,
                CopyType.Organization,
                CopyType.Project,
                CopyType.Routine,
                CopyType.Standard,
            ];
            if (!validTypes.includes(input.objectType as keyof typeof CopyType)) {
                throw new CustomError(CODE.InvalidArgs, 'Invalid copy object type.', { code: genErrorCode('0227') });
            }
            const model: ModelLogic<any, any, any> = ObjectMap[input.objectType as keyof typeof GraphQLModelType] as ModelLogic<any, any, any>;
            const result = await copyHelper({ info, input, model: model, prisma, userId: req.userId })
            return { [lowercaseFirstLetter(input.objectType)]: result };
        },
        fork: async (_parent: undefined, { input }: IWrap<ForkInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<ForkResult> => {
            await rateLimit({ info, max: 500, byAccountOrKey: true, req });
            const validTypes: Array<keyof typeof ForkType> = [
                ForkType.Organization,
                ForkType.Project,
                ForkType.Routine,
                ForkType.Standard,
            ];
            if (!validTypes.includes(input.objectType as keyof typeof ForkType)) {
                throw new CustomError(CODE.InvalidArgs, 'Invalid fork object type.', { code: genErrorCode('0228') });
            }
            const model: ModelLogic<any, any, any> = ObjectMap[input.objectType as keyof typeof GraphQLModelType] as ModelLogic<any, any, any>;
            const result = await forkHelper({ info, input, model: model, prisma, userId: req.userId })
            return { [lowercaseFirstLetter(input.objectType)]: result };
        }
    }
}
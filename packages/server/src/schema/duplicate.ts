/**
 * Handles copying and forking of objects. 
 * Copying objects is useful when you want your own version, and do not care about changes to the original.
 * Forking is useful when you want to track changes and suggest changes to the original.
 */
import { gql } from 'apollo-server-express';
import { CopyInput, CopyResult, ForkInput, ForkResult } from './types';
import { IWrap } from 'types';
import { copyHelper, forkHelper, NodeModel, OrganizationModel, ProjectModel, RoutineModel, StandardModel } from '../models';
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
        copy: async (_parent: undefined, { input }: IWrap<CopyInput>, context: Context, info: GraphQLResolveInfo): Promise<CopyResult> => {
            await rateLimit({ context, info, max: 500, byAccount: true });
            switch (input.objectType) {
                case CopyType.Node:
                    const node = await copyHelper(context.req.userId, input, info, NodeModel(context.prisma));
                    return { node };
                case CopyType.Organization:
                    const organization = await copyHelper(context.req.userId, input, info, OrganizationModel(context.prisma));
                    return { organization };
                case CopyType.Project:
                    const project = await copyHelper(context.req.userId, input, info, ProjectModel(context.prisma));
                    return { project };
                case CopyType.Routine:
                    const routine = await copyHelper(context.req.userId, input, info, RoutineModel(context.prisma));
                    return { routine };
                case CopyType.Standard:
                    const standard = await copyHelper(context.req.userId, input, info, StandardModel(context.prisma));
                    return { standard };
            }
            throw new CustomError(CODE.InvalidArgs, 'Invalid copy object type.', { code: genErrorCode('0227') });
        },
        fork: async (_parent: undefined, { input }: IWrap<ForkInput>, context: Context, info: GraphQLResolveInfo): Promise<ForkResult> => {
            await rateLimit({ context, info, max: 500, byAccount: true });
            switch (input.objectType) {
                case ForkType.Organization:
                    const organization = await forkHelper(context.req.userId, input, info, OrganizationModel(context.prisma));
                    return { organization };
                case ForkType.Project:
                    const project = await forkHelper(context.req.userId, input, info, ProjectModel(context.prisma));
                    return { project };
                case ForkType.Routine:
                    const routine = await forkHelper(context.req.userId, input, info, RoutineModel(context.prisma));
                    return { routine };
                case ForkType.Standard:
                    const standard = await forkHelper(context.req.userId, input, info, StandardModel(context.prisma));
                    return { standard };
            }
            throw new CustomError(CODE.InvalidArgs, 'Invalid fork object type.', { code: genErrorCode('0228') });
        }
    }
}
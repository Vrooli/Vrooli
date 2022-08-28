import { gql } from 'apollo-server-express';
import { DeleteOneInput, Success } from './types';
import { IWrap } from '../types';
import { deleteOneHelper, GraphQLModelType, ModelLogic, ObjectMap } from '../models';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';
import { CustomError } from '../error';
import { CODE, DeleteOneType } from '@shared/consts';
import { genErrorCode } from '../logger';

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
            await rateLimit({ info, max: 1000, byAccountOrKey: true, req });
            const validTypes: Array<keyof typeof DeleteOneType> = [
                DeleteOneType.Comment, 
                DeleteOneType.Email,
                DeleteOneType.Node,
                DeleteOneType.Organization,
                DeleteOneType.Project,
                DeleteOneType.Report,
                DeleteOneType.Routine,
                DeleteOneType.Run,
                DeleteOneType.Standard,
                DeleteOneType.Wallet,
            ];
            if (!validTypes.includes(input.objectType as keyof typeof DeleteOneType)) {
                throw new CustomError(CODE.InvalidArgs, 'Invalid delete object type.', { code: genErrorCode('0216') });
            }
            const model: ModelLogic<any, any, any> = ObjectMap[input.objectType as keyof typeof GraphQLModelType] as ModelLogic<any, any, any>;
            return deleteOneHelper({ input, model, prisma, userId: req.userId });
        },
    }
}
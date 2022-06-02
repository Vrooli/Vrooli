import { gql } from 'apollo-server-express';
import { DeleteOneInput, Success } from './types';
import { IWrap } from 'types';
import { CommentModel, deleteOneHelper, EmailModel, ModelBusinessLayer, NodeModel, OrganizationModel, ProjectModel, ReportModel, RoutineModel, StandardModel, WalletModel } from '../models';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';
import { CustomError } from '../error';
import { CODE, DeleteOneType } from '@local/shared';
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
        deleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, context: Context, info: GraphQLResolveInfo): Promise<Success> => {
            await rateLimit({ context, info, max: 1000, byAccount: true });
            let model: ModelBusinessLayer<any, any> | undefined;
            switch (input.objectType) {
                case DeleteOneType.Comment:
                    model = CommentModel(context.prisma);
                    break;
                case DeleteOneType.Email:
                    model = EmailModel(context.prisma);
                    break;
                case DeleteOneType.Node:
                    model = NodeModel(context.prisma);
                    break;
                case DeleteOneType.Organization:
                    model = OrganizationModel(context.prisma);
                    break;
                case DeleteOneType.Project:
                    model = ProjectModel(context.prisma);
                    break;
                case DeleteOneType.Report:
                    model = ReportModel(context.prisma);
                    break;
                case DeleteOneType.Routine:
                    model = RoutineModel(context.prisma);
                    break;
                case DeleteOneType.Standard:
                    model = StandardModel(context.prisma);
                    break;
                case DeleteOneType.Wallet:
                    model = WalletModel(context.prisma);
                    break;
            }
            if (model) return deleteOneHelper(context.req.userId, input, model);
            throw new CustomError(CODE.InvalidArgs, 'Invalid delete object type.', { code: genErrorCode('0216') });
        },
    }
}
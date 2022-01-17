import { gql } from 'apollo-server-express';
import { CODE, ReportFor } from '@local/shared';
import { CustomError } from '../error';
import { DeleteOneInput, Report, ReportInput, Success } from './types';
import { IWrap, RecursivePartial } from 'types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { ReportModel } from '../models';

export const typeDef = gql`
    enum ReportFor {
        Comment
        Organization
        Project
        Routine
        Standard
        Tag
        User
    }   

    input ReportInput {
        id: ID
        reason: String!
        details: String
        createdFor: ReportFor!
        forId: ID!
    }

    type Report {
        id: ID
        reason: String!
        details: String
    }

    extend type Mutation {
        reportAdd(input: ReportInput!): Report!
        reportUpdate(input: ReportInput!): Report!
        reportDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    ReportFor: ReportFor,
    Mutation: {
        reportAdd: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Report>> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            return await ReportModel(prisma).addReport(req.userId, input);
        },
        reportUpdate: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Report>> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            return await ReportModel(prisma).updateReport(req.userId, input);
        },
        reportDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            const success = await ReportModel(prisma).deleteReport(req.userId, input);
            return { success };
        },
    }
}
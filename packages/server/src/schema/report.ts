import { gql } from 'apollo-server-express';
import { CODE, ReportFor } from '@local/shared';
import { CustomError } from '../error';
import { DeleteOneInput, Report, ReportAddInput, ReportUpdateInput, Success } from './types';
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

    input ReportAddInput {
        createdFor: ReportFor!
        createdForId: ID!
        details: String
        reason: String!
    }
    input ReportUpdateInput {
        id: ID!
        details: String
        reason: String
    }

    type Report {
        id: ID
        reason: String!
        details: String
    }

    extend type Mutation {
        reportAdd(input: ReportAddInput!): Report!
        reportUpdate(input: ReportUpdateInput!): Report!
        reportDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    ReportFor: ReportFor,
    Mutation: {
        reportAdd: async (_parent: undefined, { input }: IWrap<ReportAddInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Report>> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            return await ReportModel(prisma).addReport(req.userId, input);
        },
        reportUpdate: async (_parent: undefined, { input }: IWrap<ReportUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Report>> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            return await ReportModel(prisma).updateReport(req.userId, input);
        },
        reportDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            return await ReportModel(prisma).deleteReport(req.userId, input);
        },
    }
}
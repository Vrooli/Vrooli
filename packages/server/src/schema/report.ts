import { gql } from 'apollo-server-express';
import { CODE, ReportFor } from '@local/shared';
import { CustomError } from '../error';
import { DeleteOneInput, Report, ReportCreateInput, ReportUpdateInput, Success } from './types';
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

    input ReportCreateInput {
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
        reportCreate(input: ReportCreateInput!): Report!
        reportUpdate(input: ReportUpdateInput!): Report!
        reportDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    ReportFor: ReportFor,
    Mutation: {
        reportCreate: async (_parent: undefined, { input }: IWrap<ReportCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Report>> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            return await ReportModel(prisma).create(req.userId, input);
        },
        reportUpdate: async (_parent: undefined, { input }: IWrap<ReportUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Report>> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            return await ReportModel(prisma).update(req.userId, input);
        },
        reportDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            return await ReportModel(prisma).delete(req.userId, input);
        },
    }
}
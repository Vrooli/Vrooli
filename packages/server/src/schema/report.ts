import { gql } from 'apollo-server-express';
import { ReportFor } from '@local/shared';
import { DeleteOneInput, Report, ReportCreateInput, ReportUpdateInput, Success } from './types';
import { IWrap, RecursivePartial } from 'types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { createHelper, deleteOneHelper, ReportModel, updateHelper } from '../models';

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
        isOwn: Boolean!
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
            return createHelper(req.userId, input, info, ReportModel(prisma).cud);
        },
        reportUpdate: async (_parent: undefined, { input }: IWrap<ReportUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Report>> => {
            return updateHelper(req.userId, input, info, ReportModel(prisma).cud);
        },
        reportDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            return deleteOneHelper(req.userId, input, ReportModel(prisma).cud);
        },
    }
}
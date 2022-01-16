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
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // Create object
            const dbModel = await ReportModel(prisma).create(input, info);
            // Format object to GraphQL type
            return ReportModel().toGraphQL(dbModel);
        },
        reportUpdate: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Report>> => {
            // Must be logged in
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Validate input
            //TODO
            // Update object
            const dbModel = await ReportModel(prisma).update(input, info);
            // Format to GraphQL type
            return ReportModel().toGraphQL(dbModel);
        },
        reportDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Validate input
            //TODO
            // Delete and return result
            const success = await ReportModel(prisma).delete(input);
            return { success };
        },
    }
}